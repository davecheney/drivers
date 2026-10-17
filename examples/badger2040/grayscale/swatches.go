package main

import (
	"image/color"

	"tinygo.org/x/drivers"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

func drawSwatches(display drivers.Displayer) {
	black := color.RGBA{A: 255}
	w, h := display.Size()
	tinyfont.WriteLine(display, &freemono.Regular9pt7b, 4, 18, "TOP LEFT", black)
	for i, label := range []string{"0", "1", "2", "3"} {
		left := int16(i) * w / 4
		right := int16(i+1) * w / 4
		c := uint8(i * 85)
		shade := color.RGBA{c, c, c, 255}
		for y := int16(28); y < h-26; y++ {
			for x := left + 2; x < right-2; x++ {
				value := shade
				if x == left+2 || x == right-3 || y == 28 || y == h-27 {
					value = black
				}
				display.SetPixel(x, y, value)
			}
		}
		tinyfont.WriteLine(display, &freemono.Regular9pt7b, left+24, h-8, label, black)
	}
}
