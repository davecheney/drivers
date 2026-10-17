package main

import (
	"image/color"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
	"tinygo.org/x/drivers/uc8151"
)

type calibration struct {
	dark, light byte
}

func defaultCalibration() calibration {
	return calibration{dark: uc8151.DefaultDarkGrayFrames, light: uc8151.DefaultLightGrayFrames}
}

func (c *calibration) adjust(pressed controls.Buttons) bool {
	before := *c
	c.dark, c.light = controls.AdjustGrayFrames(c.dark, c.light, pressed)
	return *c != before
}

type testerController interface {
	drivers.Displayer
	Configure(uc8151.GrayscaleConfig) error
}

func refreshTester(display testerController, cal calibration) error {
	if err := display.Configure(uc8151.GrayscaleConfig{
		Rotation:        drivers.Rotation270,
		DarkGrayFrames:  cal.dark,
		LightGrayFrames: cal.light,
	}); err != nil {
		return err
	}
	w, h := display.Size()
	for y := int16(0); y < h; y++ {
		for x := int16(0); x < w; x++ {
			display.SetPixel(x, y, color.RGBA{A: 255})
		}
	}
	if err := display.Display(); err != nil {
		return err
	}
	return showTester(display)
}
