package main

import (
	"errors"
	"image/color"
	"testing"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
	"tinygo.org/x/drivers/examples/badger2040/internal/dither"
	"tinygo.org/x/drivers/uc8151"
)

type testerDisplay struct {
	t           *testing.T
	pixels      [296 * 128]color.RGBA
	frames      [][296 * 128]color.RGBA
	configs     []uc8151.Grayscale16Config
	configErr   error
	err         error
	failRefresh int
}

func (d *testerDisplay) Size() (int16, int16) { return 296, 128 }

func (d *testerDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.t.Helper()
	if x < 0 || x >= 296 || y < 0 || y >= 128 {
		d.t.Fatalf("pixel outside Badger display: %d,%d", x, y)
	}
	d.pixels[int(y)*296+int(x)] = c
}

func (d *testerDisplay) Display() error {
	d.frames = append(d.frames, d.pixels)
	if d.failRefresh == 0 || len(d.frames) == d.failRefresh {
		return d.err
	}
	return nil
}

func (d *testerDisplay) Configure(cfg uc8151.Grayscale16Config) error {
	d.configs = append(d.configs, cfg)
	return d.configErr
}

func TestSixteenShadeTester(t *testing.T) {
	d := &testerDisplay{t: t}
	if err := showTester(d, defaultCalibration()); err != nil {
		t.Fatal(err)
	}
	if len(d.frames) != 1 {
		t.Fatal("expected one image refresh")
	}
	for i, c := range d.pixels {
		if c.R != c.G || c.R != c.B || c.R%17 != 0 || c.A != 255 {
			t.Fatalf("invalid 4bpp color at pixel %d: %v", i, c)
		}
	}
	for index := 0; index < 16; index++ {
		left, right := 8+index*280/16, 8+(index+1)*280/16
		want := uint8(15-index) * 17
		for y := 28; y < 64; y++ {
			for x := left; x < right; x++ {
				if got := d.pixels[y*296+x].R; got != want {
					t.Fatalf("solid swatch %d at %d,%d = %d, want %d", index, x, y, got, want)
				}
			}
		}
	}
	var shades [16]int
	for y := 78; y < 102; y++ {
		for x := 8; x < 288; x++ {
			gray := uint8(((287-x)*255 + 139) / 279)
			want := dither.Sixteen(gray, int16(x-8), int16(y-78))
			got := d.pixels[y*296+x].R
			if got != want {
				t.Fatalf("Bayer gradient at %d,%d = %d, want %d", x, y, got, want)
			}
			shades[got/17]++
		}
		if d.pixels[y*296+8].R != 255 || d.pixels[y*296+287].R != 0 {
			t.Fatal("gradient endpoints are not solid white and black")
		}
	}
	for shade, count := range shades {
		if count == 0 {
			t.Fatalf("gradient shade %d is missing", shade)
		}
	}
}

func TestSelectedShadeMarker(t *testing.T) {
	for shade := byte(1); shade < 15; shade++ {
		cal := defaultCalibration()
		cal.shade = shade
		d := &testerDisplay{t: t}
		if err := showTester(d, cal); err != nil {
			t.Fatal(err)
		}
		index := 15 - int(shade)
		left, right := 8+index*280/16, 8+(index+1)*280/16
		var marked int
		for x := 0; x < 296; x++ {
			if d.pixels[68*296+x].R == 0 {
				if x < left || x >= right {
					t.Fatalf("shade %d marker is outside its swatch", shade)
				}
				marked++
			}
		}
		if marked != 7 {
			t.Fatalf("shade %d marker width = %d, want 7", shade, marked)
		}
	}
}

func TestCalibrationButtons(t *testing.T) {
	for _, tc := range []struct {
		button controls.Buttons
		shade  byte
		frame  byte
	}{
		{controls.A, 3, 12},
		{controls.B, 1, 17},
		{controls.Up, 2, 15},
		{controls.Down, 2, 13},
		{controls.A | controls.B, 2, 14},
		{controls.Up | controls.Down, 2, 14},
	} {
		cal := defaultCalibration()
		changed := cal.adjust(tc.button)
		if cal.shade != tc.shade || cal.frames[14-cal.shade] != tc.frame {
			t.Fatalf("button %d: shade %d frames %d", tc.button, cal.shade, cal.frames[14-cal.shade])
		}
		if changed != (tc.button != controls.A|controls.B && tc.button != controls.Up|controls.Down) {
			t.Fatalf("button %d: changed = %v", tc.button, changed)
		}
	}
}

func TestCalibrationPulseBounds(t *testing.T) {
	for shade := byte(1); shade < 15; shade++ {
		index := 14 - shade
		base := defaultCalibration()
		previous := byte(0)
		if index > 0 {
			previous = base.frames[index-1]
		}
		next := base.frames[index+1]
		base.shade = shade
		if minimum, maximum := base.limits(); minimum != previous+1 || maximum != next-1 {
			t.Fatalf("shade %d range = %d-%d, want %d-%d", shade, minimum, maximum, previous+1, next-1)
		}
		for duration := previous + 1; duration < next; duration++ {
			for _, button := range []controls.Buttons{controls.Up, controls.Down} {
				cal := base
				cal.shade, cal.frames[index] = shade, duration
				before := cal
				changed := cal.adjust(button)
				want := duration
				if button == controls.Up && duration+1 < next {
					want++
				}
				if button == controls.Down && duration > previous+1 {
					want--
				}
				if cal.frames[index] != want || changed != (want != duration) {
					t.Fatalf("shade %d duration %d button %d: frames %d, changed %v", shade, duration, button, cal.frames[index], changed)
				}
				before.frames[index] = want
				if cal != before {
					t.Fatal("pulse adjustment changed another shade")
				}
			}
		}
	}
	cal := defaultCalibration()
	for range 20 {
		cal.adjust(controls.A)
	}
	if cal.shade != 14 {
		t.Fatal("A should stop at the lightest non-white shade")
	}
	for range 20 {
		cal.adjust(controls.B)
	}
	if cal.shade != 1 {
		t.Fatal("B should stop at the darkest non-black shade")
	}
}

func TestCalibrationStatus(t *testing.T) {
	cal := defaultCalibration()
	if got := cal.status(); got != "S2 F14 [13-16] A</B> U/D" {
		t.Fatalf("initial status = %q", got)
	}
	cal.shade = 8
	before := cal
	if cal.adjust(controls.Up) || cal.adjust(controls.Down) || cal != before {
		t.Fatal("one-frame neighbor spacing should prevent an individual adjustment")
	}
	if got := cal.status(); got != "S8 F7 [7-7] A</B> U/D" {
		t.Fatalf("locked shade status = %q", got)
	}
}

func TestRefreshTester(t *testing.T) {
	d := &testerDisplay{t: t}
	cal := defaultCalibration()
	for attempt := 0; attempt < 2; attempt++ {
		cal.adjust(controls.A)
		if err := refreshTester(d, cal); err != nil {
			t.Fatal(err)
		}
		if len(d.configs) != attempt+1 || len(d.frames) != (attempt+1)*2 {
			t.Fatal("expected a reset and black cleaning frame before each image")
		}
		cfg := d.configs[attempt]
		if cfg.Rotation != drivers.Rotation270 || cfg.Frames != cal.frames {
			t.Fatal("calibration was not applied to the controller")
		}
		for i, c := range d.frames[attempt*2] {
			if c != (color.RGBA{A: 255}) {
				t.Fatalf("cleaning frame pixel %d is not solid black", i)
			}
		}
		frame := d.frames[attempt*2+1]
		if frame[40*296+8].R != 255 || frame[40*296+287].R != 0 {
			t.Fatal("missing swatches after cleaning frame")
		}
	}
	if d.frames[1] == d.frames[3] {
		t.Fatal("selection marker or status did not change")
	}
}

func TestTesterErrors(t *testing.T) {
	failure := errors.New("SPI failure")
	for _, tc := range []struct {
		name      string
		configErr error
		failAt    int
		refreshes int
	}{
		{"configure", failure, 0, 0},
		{"black cleaning frame", nil, 1, 1},
		{"tester frame", nil, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &testerDisplay{t: t, configErr: tc.configErr, err: failure, failRefresh: tc.failAt}
			if err := refreshTester(d, defaultCalibration()); !errors.Is(err, failure) {
				t.Fatalf("error = %v, want %v", err, failure)
			}
			if len(d.frames) != tc.refreshes {
				t.Fatalf("refreshes = %d, want %d", len(d.frames), tc.refreshes)
			}
		})
	}
}
