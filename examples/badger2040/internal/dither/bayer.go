package dither

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

// Four applies 8x8 Bayer dithering to the four-shade grayscale palette.
func Four(gray uint8, x, y int16) uint8 {
	return ordered(gray, x, y, 85)
}

// Sixteen applies 8x8 Bayer dithering to the sixteen-shade grayscale palette.
func Sixteen(gray uint8, x, y int16) uint8 {
	return ordered(gray, x, y, 17)
}

func ordered(gray uint8, x, y int16, step uint8) uint8 {
	level := gray / step
	if uint16(gray%step)*128 > (2*uint16(bayer8[y&7][x&7])+1)*uint16(step) {
		level++
	}
	return level * step
}
