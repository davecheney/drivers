package main

import (
	"image/color"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/uc8151"
)

type buttons uint8

const (
	buttonUp buttons = 1 << iota
	buttonDown
	buttonA
	buttonB
)

type calibration struct {
	dark, light byte
}

func defaultCalibration() calibration {
	return calibration{dark: uc8151.DefaultDarkGrayFrames, light: uc8151.DefaultLightGrayFrames}
}

func (c calibration) config() uc8151.GrayscaleConfig {
	return uc8151.GrayscaleConfig{
		Rotation:        drivers.Rotation270,
		DarkGrayFrames:  c.dark,
		LightGrayFrames: c.light,
	}
}

func (c *calibration) adjust(pressed buttons) bool {
	before := *c
	switch pressed & (buttonUp | buttonDown) {
	case buttonUp:
		if c.dark < 31 {
			c.dark++
		}
	case buttonDown:
		if c.dark > c.light+1 {
			c.dark--
		}
	}
	switch pressed & (buttonA | buttonB) {
	case buttonA:
		if c.light+1 < c.dark {
			c.light++
		}
	case buttonB:
		if c.light > 1 {
			c.light--
		}
	}
	return *c != before
}

const buttonDebounce = 30 * time.Millisecond

type buttonDebouncer struct {
	last, stable buttons
	changed      time.Time
}

func (b *buttonDebouncer) update(raw buttons, now time.Time) buttons {
	if raw != b.last {
		b.last = raw
		b.changed = now
	}
	if now.Sub(b.changed) < buttonDebounce {
		return 0
	}
	pressed := raw &^ b.stable
	b.stable = raw
	return pressed
}

type gradientController interface {
	drivers.Displayer
	Configure(uc8151.GrayscaleConfig) error
}

func refreshGradient(display gradientController, cal calibration) error {
	if err := display.Configure(cal.config()); err != nil {
		return err
	}
	w, h := display.Size()
	black := color.RGBA{A: 255}
	for y := int16(0); y < h; y++ {
		for x := int16(0); x < w; x++ {
			display.SetPixel(x, y, black)
		}
	}
	if err := display.Display(); err != nil {
		return err
	}
	return showGradient(display, cal)
}
