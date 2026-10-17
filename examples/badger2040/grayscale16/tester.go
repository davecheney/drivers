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
	return calibration{shade: 8, frames: uc8151.DefaultGrayscale16Frames()}
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
	switch pressed & (controls.Up | controls.Down) {
	case controls.Up:
		if c.frames[index]+1 < c.frames[index+1] {
			c.frames[index]++
		}
	case controls.Down:
		previous := byte(0)
		if index > 0 {
			previous = c.frames[index-1]
		}
		if c.frames[index] > previous+1 {
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
	status := "S" + strconv.Itoa(int(cal.shade)) + " F" + strconv.Itoa(int(cal.frames[14-cal.shade])) + " A</B> U+/D-"
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, left, h-6, status, black)
	return display.Display()
}
