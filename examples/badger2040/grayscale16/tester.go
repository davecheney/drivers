package main

import (
	"image/color"
	"strconv"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
	"tinygo.org/x/drivers/examples/badger2040/internal/dither"
	"tinygo.org/x/drivers/uc8151"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

type calibration struct {
	shade  byte
	frames [15]byte
}

func defaultCalibration() calibration {
	return calibration{shade: 2, frames: uc8151.DefaultGrayscale16Frames()}
}

func (c calibration) limits() (minimum, maximum byte) {
	index := 14 - c.shade
	minimum = 1
	if index > 0 {
		minimum = c.frames[index-1] + 1
	}
	return minimum, c.frames[index+1] - 1
}

func (c calibration) status() string {
	minimum, maximum := c.limits()
	return "S" + strconv.Itoa(int(c.shade)) + " F" + strconv.Itoa(int(c.frames[14-c.shade])) +
		" [" + strconv.Itoa(int(minimum)) + "-" + strconv.Itoa(int(maximum)) + "] A</B> U/D"
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
	index := 14 - c.shade
	minimum, maximum := c.limits()
	switch pressed & (controls.Up | controls.Down) {
	case controls.Up:
		if c.frames[index] < maximum {
			c.frames[index]++
		}
	case controls.Down:
		if c.frames[index] > minimum {
			c.frames[index]--
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
	return showTester(display, cal)
}

func showTester(display drivers.Displayer, cal calibration) error {
	w, h := display.Size()
	const left int16 = 8
	right := w - 8
	for y := int16(0); y < h; y++ {
		for x := int16(0); x < w; x++ {
			gray := uint8(255)
			if x >= left && x < right {
				if y >= 28 && y < 64 {
					// Match the integer swatch boundaries when 280 is not divisible by 16.
					index := ((int32(x-left)+1)*16 - 1) / int32(right-left)
					gray = uint8(15-index) * 17
				} else if y >= 78 && y < 102 {
					gray = uint8((int32(right-1-x)*255 + int32(right-left-1)/2) / int32(right-left-1))
					gray = dither.Sixteen(gray, x-left, y-78)
				}
			}
			display.SetPixel(x, y, color.RGBA{gray, gray, gray, 255})
		}
	}
	black := color.RGBA{A: 255}
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, left, 18, "16 SHADES / 4BPP", black)
	index := int16(15 - cal.shade)
	center := left + (2*index+1)*(right-left)/32
	for y := int16(67); y < 71; y++ {
		for x := center - 3; x <= center+3; x++ {
			display.SetPixel(x, y, black)
		}
	}
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, left, h-6, cal.status(), black)
	return display.Display()
}
