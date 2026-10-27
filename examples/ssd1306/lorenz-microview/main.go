//go:build microview

// Draws the Lorenz attractor on the 64x48 SPI OLED of a SparkFun MicroView.
// Nothing is allocated, so it builds with -gc=none and the trail can use the free RAM.
package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers/ssd1306"
)

const (
	trail = 250
	sigma = 10
	rho   = 28
	beta  = 8.0 / 3
	dt    = 0.006
	scale = 0.95
	steps = 16
)

var (
	on     = color.RGBA{255, 255, 255, 255}
	off    = color.RGBA{0, 0, 0, 255}
	px, py [trail]int8
	buf    [64 * 48 / 8]byte
	// A package level display is made static at compile time.
	display = ssd1306.NewSPI(machine.SPI0, machine.OLED_DC, machine.OLED_RST, machine.OLED_CS)
)

func main() {
	// OLED_PWR enables the 3.3V regulator for the OLED.
	machine.OLED_PWR.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.OLED_PWR.High()
	machine.SPI0.Configure(machine.SPIConfig{Frequency: 8_000_000})
	// A 64x48 panel shows columns 32 to 95 of the 128 column controller RAM.
	display.Configure(ssd1306.Config{Width: 64, Height: 48, ResetCol: ssd1306.ResetValue{32, 95}, Buffer: buf[:]})
	display.ClearDisplay()

	for i := range px {
		px[i] = -1
	}

	w, h := display.Size()
	cx, cy := float32(w/2), float32(h/2)
	x, y, z := float32(0.1), float32(0), float32(0)
	cos, sin := float32(1), float32(0)
	for i := 0; ; i++ {
		dx := sigma * (y - x)
		dy := x*(rho-z) - y
		dz := x*y - beta*z
		x += dx * dt
		y += dy * dt
		z += dz * dt

		// Rotate by a small step to avoid math.Sin and math.Cos, which do not fit.
		if i%64 == 0 {
			cos, sin = cos-sin*0.004, sin+cos*0.004
		}
		sx := int16(cx + (x*cos-y*sin)*scale)
		sy := int16(cy - (z-25)*0.95)

		j := i % trail
		if px[j] >= 0 {
			display.SetPixel(int16(px[j]), int16(py[j]), off)
		}
		display.SetPixel(sx, sy, on)
		if sx >= 0 && sx < w && sy >= 0 && sy < h {
			px[j], py[j] = int8(sx), int8(sy)
		} else {
			px[j] = -1
		}

		if i%steps == 0 {
			display.Display()
		}
	}
}
