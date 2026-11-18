// Draws the Lorenz attractor on any display. A trail of points traces the
// attractor while the view slowly rotates around the z axis.
//
// Boards with a built-in display use it:
//
//	tinygo flash -target=wioterminal ./examples/lorenz
//
// Other boards need a display tag:
//
//	tinygo flash -target=xiao-esp32c3 -tags=ssd1306_64x48 ./examples/lorenz
//	tinygo flash -target=arduino-due -tags=ili9341 ./examples/lorenz
//
// The display tags are ssd1306, ssd1306_128x32 and ssd1306_64x48 (I2C0),
// ili9341 (SPI0, DC D9, CS D10, RST D8), ili9325 (ITDB02 Arduino shield),
// and the Waveshare PiZero LCD HATs st7789 (1.3inch) and st7735 (1.44inch).
package main

import (
	"image/color"
	"math"

	"tinygo.org/x/drivers"
)

const (
	sigma = 10
	rho   = 28
	beta  = 8.0 / 3
	dt    = 0.006
)

func main() {
	run(newDisplay())
}

func run(d drivers.Displayer) {
	w, h := d.Size()
	n := min(int(w)*int(h)/8, maxTrail)
	px, py := make([]int16, n), make([]int16, n)
	for i := range px {
		px[i] = -1
	}

	var palette [64]color.RGBA
	for i := range palette {
		palette[i] = hue(float32(i) / float32(len(palette)))
	}
	black := color.RGBA{0, 0, 0, 255}

	// The attractor is about 62 units wide and 52 units high.
	scale := min(float32(w)/62, float32(h)/52)
	cx, cy := float32(w)/2, float32(h)/2
	x, y, z := float32(0.1), float32(0), float32(0)
	theta := float64(0)
	cos, sin := float32(1), float32(0)
	for i := 0; ; i++ {
		dx := sigma * (y - x)
		dy := x*(rho-z) - y
		dz := x*y - beta*z
		x += dx * dt
		y += dy * dt
		z += dz * dt

		if i%64 == 0 {
			theta += 0.004
			cos, sin = float32(math.Cos(theta)), float32(math.Sin(theta))
		}
		sx := int16(cx + (x*cos-y*sin)*scale)
		sy := int16(cy - (z-25)*scale)

		j := i % n
		if px[j] >= 0 {
			d.SetPixel(px[j], py[j], black)
		}
		zi := max(int(z*1.3), 0)
		d.SetPixel(sx, sy, palette[zi%len(palette)])
		px[j], py[j] = sx, sy

		if i%16 == 0 {
			d.Display()
		}
	}
}

// hue returns a fully saturated colour for h in [0, 1).
func hue(h float32) color.RGBA {
	h6 := h * 6
	f := h6 - float32(int(h6))
	up, down := uint8(255*f), uint8(255*(1-f))
	switch int(h6) % 6 {
	case 0:
		return color.RGBA{255, up, 0, 255}
	case 1:
		return color.RGBA{down, 255, 0, 255}
	case 2:
		return color.RGBA{0, 255, up, 255}
	case 3:
		return color.RGBA{0, down, 255, 255}
	case 4:
		return color.RGBA{up, 0, 255, 255}
	default:
		return color.RGBA{255, 0, down, 255}
	}
}
