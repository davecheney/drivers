package main

import (
	"errors"
	"image/color"
	"testing"
	"time"

	"tinygo.org/x/drivers"
)

type buttonLevel bool

func (level buttonLevel) Get() bool { return bool(level) }

func TestActiveHighButtons(t *testing.T) {
	for mask := buttons(0); mask < 16; mask++ {
		pins := buttonPins{
			buttonLevel(mask&buttonUp != 0),
			buttonLevel(mask&buttonDown != 0),
			buttonLevel(mask&buttonA != 0),
			buttonLevel(mask&buttonB != 0),
		}
		if got := pins.read(); got != mask {
			t.Fatalf("HIGH pin mask %04b gives buttons %04b", mask, got)
		}
	}
}

func TestCalibration(t *testing.T) {
	if got := defaultCalibration(); got != (calibration{dark: 10, light: 4}) {
		t.Fatalf("initial calibration = %v", got)
	}
	for dark := byte(2); dark < 32; dark++ {
		for light := byte(1); light < dark; light++ {
			for _, button := range []buttons{buttonUp, buttonDown, buttonA, buttonB} {
				before := calibration{dark: dark, light: light}
				cal := before
				changed := cal.adjust(button)
				want := before
				switch button {
				case buttonUp:
					if dark < 31 {
						want.dark++
					}
				case buttonDown:
					if dark > light+1 {
						want.dark--
					}
				case buttonA:
					if light+1 < dark {
						want.light++
					}
				case buttonB:
					if light > 1 {
						want.light--
					}
				}
				if cal != want || changed != (want != before) {
					t.Fatalf("%v + button %d = %v, changed %v; want %v", before, button, cal, changed, want)
				}
				if cal.light < 1 || cal.light >= cal.dark || cal.dark >= 32 {
					t.Fatalf("invalid calibration after button %d: %v", button, cal)
				}
			}
		}
	}
}

func TestCalibrationSimultaneousButtons(t *testing.T) {
	for _, tc := range []struct {
		pressed buttons
		want    calibration
	}{
		{0, calibration{10, 4}},
		{buttonUp | buttonDown, calibration{10, 4}},
		{buttonA | buttonB, calibration{10, 4}},
		{buttonUp | buttonA, calibration{11, 5}},
		{buttonDown | buttonB, calibration{9, 3}},
		{buttonUp | buttonDown | buttonA | buttonB, calibration{10, 4}},
	} {
		cal := defaultCalibration()
		cal.adjust(tc.pressed)
		if cal != tc.want {
			t.Fatalf("buttons %d = %v, want %v", tc.pressed, cal, tc.want)
		}
	}
}

func TestButtonDebounce(t *testing.T) {
	var debounce buttonDebouncer
	start := time.Now()
	for _, tc := range []struct {
		ms   int
		raw  buttons
		want buttons
	}{
		{0, 0, 0},
		{10, buttonUp, 0},
		{20, 0, 0},
		{25, buttonUp, 0},
		{54, buttonUp, 0},
		{55, buttonUp, buttonUp},
		{5000, buttonUp, 0},
		{5010, buttonUp | buttonA, 0},
		{5040, buttonUp | buttonA, buttonA},
		{5050, 0, 0},
		{5080, 0, 0},
		{5090, buttonUp, 0},
		{5120, buttonUp, buttonUp},
	} {
		if got := debounce.update(tc.raw, start.Add(time.Duration(tc.ms)*time.Millisecond)); got != tc.want {
			t.Fatalf("at %dms: pressed %d, want %d", tc.ms, got, tc.want)
		}
	}
}

func TestRefreshGradient(t *testing.T) {
	d := &gradientDisplay{t: t}
	cal := defaultCalibration()
	for attempt := 0; attempt < 2; attempt++ {
		cal.adjust(buttonUp)
		if err := refreshGradient(d, cal); err != nil {
			t.Fatal(err)
		}
		if len(d.configs) != attempt+1 || d.refreshes != (attempt+1)*2 {
			t.Fatalf("attempt %d: configs %d, refreshes %d", attempt, len(d.configs), d.refreshes)
		}
		cfg := d.configs[attempt]
		if cfg.Rotation != drivers.Rotation270 || cfg.DarkGrayFrames != cal.dark || cfg.LightGrayFrames != cal.light {
			t.Fatalf("attempt %d: wrong controller config %v", attempt, cfg)
		}
		for i, c := range d.frames[attempt*2] {
			if c != (color.RGBA{A: 255}) {
				t.Fatalf("attempt %d: clearing frame pixel %d is not solid black", attempt, i)
			}
		}
		frame := d.frames[attempt*2+1]
		if frame[64*296+8] != (color.RGBA{255, 255, 255, 255}) || frame[64*296+287] != (color.RGBA{A: 255}) {
			t.Fatalf("attempt %d: missing gradient after clearing frame", attempt)
		}
	}
	if d.frames[1] == d.frames[3] {
		t.Fatal("on-screen calibration values did not change")
	}
}

func TestRefreshGradientErrors(t *testing.T) {
	failure := errors.New("SPI failure")
	for _, tc := range []struct {
		name      string
		configErr error
		failAt    int
		refreshes int
	}{
		{"configure", failure, 0, 0},
		{"black clearing frame", nil, 1, 1},
		{"gradient frame", nil, 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &gradientDisplay{t: t, configErr: tc.configErr, err: failure, failRefresh: tc.failAt}
			if err := refreshGradient(d, defaultCalibration()); !errors.Is(err, failure) {
				t.Fatalf("error = %v, want %v", err, failure)
			}
			if len(d.configs) != 1 || d.refreshes != tc.refreshes {
				t.Fatalf("configs = %d, refreshes = %d", len(d.configs), d.refreshes)
			}
		})
	}
}
