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
	configs     []uc8151.GrayscaleConfig
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

func (d *testerDisplay) Configure(cfg uc8151.GrayscaleConfig) error {
	d.configs = append(d.configs, cfg)
	return d.configErr
}

func imagePixel(i int) color.RGBA {
	shade := (testImage[4+i/4] >> uint(6-i%4*2)) & 3
	gray := shade * 85
	return color.RGBA{gray, gray, gray, 255}
}

func TestPhotoWithoutOverlays(t *testing.T) {
	if len(testImage) != 9476 || !bytes.Equal(testImage[:4], []byte{0x28, 0x01, 0x80, 0x00}) {
		t.Fatal("expected a 296x128 packed 2bpp image")
	}
	d := &testerDisplay{t: t}
	if err := showTester(d); err != nil {
		t.Fatal(err)
	}
	if len(d.frames) != 1 || d.writes != 296*128 {
		t.Fatal("expected one complete image with no overlay writes")
	}
	var shades [4]int
	for i, c := range d.pixels {
		if c != imagePixel(i) {
			t.Fatalf("image pixel %d was altered or covered: %v", i, c)
		}
		shades[c.R/85]++
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
	for _, tc := range []struct {
		button controls.Buttons
		want   calibration
	}{
		{0, calibration{9, 5}},
		{controls.Up, calibration{10, 5}},
		{controls.Down, calibration{8, 5}},
		{controls.A, calibration{9, 6}},
		{controls.B, calibration{9, 4}},
		{controls.Up | controls.Down, calibration{9, 5}},
		{controls.A | controls.B, calibration{9, 5}},
		{controls.Up | controls.A, calibration{10, 6}},
	} {
		cal := defaultCalibration()
		changed := cal.adjust(tc.button)
		if cal != tc.want || changed != (tc.want != defaultCalibration()) {
			t.Fatalf("button %d: got %v, changed %v; want %v", tc.button, cal, changed, tc.want)
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
		if cfg.Rotation != drivers.Rotation270 || cfg.DarkGrayFrames != cal.dark || cfg.LightGrayFrames != cal.light {
			t.Fatal("calibration was not applied to the controller")
		}
		for i, c := range d.frames[attempt*2] {
			if c != (color.RGBA{A: 255}) {
				t.Fatalf("cleaning frame pixel %d is not solid black", i)
			}
		}
		for i, c := range d.frames[attempt*2+1] {
			if c != imagePixel(i) {
				t.Fatalf("photo pixel %d was not restored after cleaning frame", i)
			}
		}
	}
	if d.frames[1] != d.frames[3] {
		t.Fatal("blind calibration should not add text or markers to the photo")
	}
	if d.configs[0].DarkGrayFrames == d.configs[1].DarkGrayFrames {
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
