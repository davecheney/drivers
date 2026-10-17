package main

import (
	"image/color"
	"strconv"

	"tinygo.org/x/drivers"
	"tinygo.org/x/tinyfont"
	"tinygo.org/x/tinyfont/freemono"
)

var bayer8 = [8][8]uint8{
	{0, 48, 12, 60, 3, 51, 15, 63},
	{32, 16, 44, 28, 35, 19, 47, 31},
	{8, 56, 4, 52, 11, 59, 7, 55},
	{40, 24, 36, 20, 43, 27, 39, 23},
	{2, 50, 14, 62, 1, 49, 13, 61},
	{34, 18, 46, 30, 33, 17, 45, 29},
	{10, 58, 6, 54, 9, 57, 5, 53},
	{42, 26, 38, 22, 41, 25, 37, 21},
}

func bayerGray(gray uint8, x, y int16) uint8 {
	level := gray / 85
	if uint16(gray%85)*128 > (2*uint16(bayer8[y&7][x&7])+1)*85 {
		level++
	}
	return level * 85
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
