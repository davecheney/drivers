package main

import (
	"image/color"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
	"tinygo.org/x/drivers/uc8151"
)

type calibration struct {
	shade  byte
	frames [15]byte
}

func defaultCalibration() calibration {
	return calibration{shade: 4, frames: uc8151.DefaultGrayscale16Frames()}
}

func (c calibration) limits() (minimum, maximum byte) {
	return 15 - c.shade, 32 - c.shade
}

func (c *calibration) adjust(pressed controls.Buttons) bool {
	before := *c
	switch pressed & (controls.A | controls.B) {
	case controls.A:
		if c.shade < 14 {
			c.shade++
		}
	case controls.B:
		if c.shade > 1 {
			c.shade--
		}
	}
	index := 14 - int(c.shade)
	minimum, maximum := c.limits()
	switch pressed & (controls.Up | controls.Down) {
	case controls.Up:
		if c.frames[index] < maximum {
			c.frames[index]++
			for i := index + 1; i < 14 && c.frames[i] <= c.frames[i-1]; i++ {
				c.frames[i] = c.frames[i-1] + 1
			}
		}
	case controls.Down:
		if c.frames[index] > minimum {
			c.frames[index]--
			for i := index - 1; i >= 0 && c.frames[i] >= c.frames[i+1]; i-- {
				c.frames[i] = c.frames[i+1] - 1
			}
		}
	}
	return *c != before
}

type testerController interface {
	drivers.Displayer
	Configure(uc8151.Grayscale16Config) error
}

func refreshTester(display testerController, cal calibration) error {
	if err := display.Configure(uc8151.Grayscale16Config{
		Rotation: drivers.Rotation270, Frames: cal.frames,
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
