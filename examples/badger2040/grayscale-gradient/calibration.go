package main

import (
	"image/color"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
	"tinygo.org/x/drivers/uc8151"
)

type buttons = controls.Buttons
type buttonPins = controls.Pins
type buttonDebouncer = controls.Debouncer

const (
	buttonUp   = controls.Up
	buttonDown = controls.Down
	buttonA    = controls.A
	buttonB    = controls.B
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
	c.dark, c.light = controls.AdjustGrayFrames(c.dark, c.light, pressed)
	return *c != before
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
