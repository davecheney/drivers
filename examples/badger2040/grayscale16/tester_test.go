package main

import (
	"bytes"
	"errors"
	"image/color"
	"testing"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
	"tinygo.org/x/drivers/uc8151"
)

type testerDisplay struct {
	t           *testing.T
	pixels      [296 * 128]color.RGBA
	writes      int
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
	d.writes++
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

func TestPhotoWithoutOverlays(t *testing.T) {
	if len(testImage) != 18948 || !bytes.Equal(testImage[:4], []byte{0x28, 0x01, 0x80, 0x00}) {
		t.Fatal("expected a 296x128 packed 4bpp image")
	}
	d := &testerDisplay{t: t}
	if err := showTester(d); err != nil {
		t.Fatal(err)
	}
	if len(d.frames) != 1 || d.writes != 296*128 {
		t.Fatal("expected one complete image with no overlay writes")
	}
	var shades [16]int
	for i, c := range d.pixels {
		value := (testImage[4+i/2] >> uint(4-i%2*4)) & 15
		gray := value * 17
		if c != (color.RGBA{gray, gray, gray, 255}) {
			t.Fatalf("image pixel %d was altered or covered: %v", i, c)
		}
		shades[value]++
	}
	for shade, count := range shades {
		if count == 0 {
			t.Fatalf("photo shade %d is missing", shade)
		}
	}
	for _, x := range []int{0, 295} {
		content := false
		for y := 0; y < 128; y++ {
			content = content || d.pixels[y*296+x].R != 255
		}
		if !content {
			t.Fatalf("photo edge %d has a blank border", x)
		}
	}
}

func TestCalibrationButtons(t *testing.T) {
	initial := defaultCalibration()
	if initial.shade != 4 || initial.frames[10] != 11 {
		t.Fatal("expected initial S4/F11 calibration")
	}
	for _, tc := range []struct {
		button  controls.Buttons
		shade   byte
		frame   byte
		changed bool
	}{
		{controls.A, 5, 10, true},
		{controls.B, 3, 12, true},
		{controls.Up, 4, 12, true},
		{controls.Down, 4, 11, false},
		{controls.A | controls.B, 4, 11, false},
		{controls.Up | controls.Down, 4, 11, false},
	} {
		cal := defaultCalibration()
		changed := cal.adjust(tc.button)
		if cal.shade != tc.shade || cal.frames[14-cal.shade] != tc.frame {
			t.Fatalf("button %d: shade %d frames %d", tc.button, cal.shade, cal.frames[14-cal.shade])
		}
		if changed != tc.changed {
			t.Fatalf("button %d: changed = %v", tc.button, changed)
		}
	}
}

func TestCalibrationPulseBounds(t *testing.T) {
	for shade := byte(1); shade < 15; shade++ {
		index := 14 - int(shade)
		minimum, maximum := 15-shade, 32-shade
		for duration := minimum; duration <= maximum; duration++ {
			for _, button := range []controls.Buttons{controls.Up, controls.Down} {
				cal := calibration{shade: shade}
				for i := 0; i < index; i++ {
					cal.frames[i] = byte(i + 1)
				}
				for i := index; i < 14; i++ {
					cal.frames[i] = duration + byte(i-index)
				}
				cal.frames[14] = 32
				if low, high := cal.limits(); low != minimum || high != maximum {
					t.Fatalf("shade %d bounds = %d-%d", shade, low, high)
				}
				changed := cal.adjust(button)
				want := duration
				if button == controls.Up && duration < maximum {
					want++
				}
				if button == controls.Down && duration > minimum {
					want--
				}
				if cal.frames[index] != want || changed != (want != duration) {
					t.Fatalf("shade %d duration %d button %d: frames %d, changed %v", shade, duration, button, cal.frames[index], changed)
				}
				for i, value := range cal.frames {
					if value == 0 || i > 0 && value <= cal.frames[i-1] {
						t.Fatal("neighbor propagation lost distinct, ordered timings")
					}
				}
				if cal.frames[14] != 32 || cal.shade != shade {
					t.Fatal("pulse adjustment changed full black or the selection")
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

func TestCalibrationNeighborPropagation(t *testing.T) {
	cal := defaultCalibration()
	if !cal.adjust(controls.Up) {
		t.Fatal("S4/F11 should be able to darken by shifting its neighbor")
	}
	want := [15]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 17, 32}
	if cal.frames != want {
		t.Fatalf("darkening propagation = %v, want %v", cal.frames, want)
	}
	for i := 0; i < 14; i++ {
		cal.frames[i] = byte(i + 3)
	}
	if !cal.adjust(controls.Down) {
		t.Fatal("expected lightening with propagation")
	}
	for i, value := range cal.frames {
		want := byte(i + 3)
		if i <= 10 {
			want--
		}
		if i == 14 {
			want = 32
		}
		if value != want {
			t.Fatalf("lightening propagation at %d = %d, want %d", i, value, want)
		}
	}
}

func TestRefreshTester(t *testing.T) {
	d := &testerDisplay{t: t}
	cal := defaultCalibration()
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			cal.adjust(controls.Up)
		}
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
		for i, pixel := range frame {
			value := (testImage[4+i/2] >> uint(4-i%2*4)) & 15
			if pixel.R != value*17 {
				t.Fatalf("photo pixel %d was not restored after cleaning frame", i)
			}
		}
	}
	if d.frames[1] != d.frames[3] {
		t.Fatal("blind calibration should not add text or markers to the photo")
	}
	if d.configs[0].Frames == d.configs[1].Frames {
		t.Fatal("physical tone adjustment was not applied to the controller")
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
