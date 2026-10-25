//go:build xiao_esp32c3

// Draws the Lorenz attractor on a 64x48 SSD1306 I2C display on a XIAO
// ESP32C3. A 400 point trail traces the attractor while the view slowly
// rotates around the z axis.
package main

import (
	"image/color"
	"machine"
	"math"

	"tinygo.org/x/drivers/ssd1306"
)

const (
	trail = 400
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
	px, py [trail]int16
)

func main() {
	machine.I2C0.Configure(machine.I2CConfig{Frequency: 400_000, SDA: machine.SDA_PIN, SCL: machine.SCL_PIN})
	display := ssd1306.NewI2C(machine.I2C0)
	// A 64x48 panel shows columns 32 to 95 of the 128 column controller RAM.
	display.Configure(ssd1306.Config{Address: 0x3c, Width: 64, Height: 48, ResetCol: ssd1306.ResetValue{32, 95}})
	display.ClearDisplay()

	for i := range px {
		px[i] = -1
	}

	w, h := display.Size()
	cx, cy := float32(w/2), float32(h/2)
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
		sy := int16(cy - (z-25)*0.95)

		j := i % trail
		if px[j] >= 0 {
			display.SetPixel(px[j], py[j], off)
		}
		display.SetPixel(sx, sy, on)
		px[j], py[j] = sx, sy

		if i%steps == 0 {
			display.Display()
		}
	}
}
