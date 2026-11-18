//go:build ili9341

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ili9341"
)

func newDisplay() drivers.Displayer {
	machine.SPI0.Configure(machine.SPIConfig{Frequency: 24 * machine.MHz})
	display := ili9341.NewSPI(machine.SPI0, machine.D9, machine.D10, machine.D8)
	display.Configure(ili9341.Config{Rotation: drivers.Rotation90})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return display
}
