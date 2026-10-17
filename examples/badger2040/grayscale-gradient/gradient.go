package main

import (
	"image/color"
	"strconv"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/examples/badger2040/internal/dither"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

func bayerGray(gray uint8, x, y int16) uint8 {
	return dither.Four(gray, x, y)
}

func showGradient(display drivers.Displayer, cal calibration) error {
	w, h := display.Size()
	const left int16 = 8
	right, top, bottom := w-8, h/4, h*3/4
	for y := int16(0); y < h; y++ {
		for x := int16(0); x < w; x++ {
			gray := uint8(255)
			if x >= left && x < right && y >= top && y < bottom {
				gray = uint8((int32(right-1-x)*255 + int32(right-left-1)/2) / int32(right-left-1))
				gray = bayerGray(gray, x-left, y-top)
			}
			display.SetPixel(x, y, color.RGBA{gray, gray, gray, 255})
		}
	}
	black := color.RGBA{A: 255}
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, left, top-12, "WHITE", black)
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, right-60, top-12, "BLACK", black)
	status := "D" + strconv.Itoa(int(cal.dark)) + " U+/D- L" + strconv.Itoa(int(cal.light)) + " A+/B-"
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, left, h-10, status, black)
	return display.Display()
}
