//go:build macropad_rp2040 && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/sh1106"
)

var display sh1106.Device

func newDisplay() drivers.Displayer {
	machine.SPI1.Configure(machine.SPIConfig{Frequency: 48 * machine.MHz})
	display = sh1106.NewSPI(machine.SPI1, machine.OLED_DC, machine.OLED_RST, machine.OLED_CS)
	display.Configure(sh1106.Config{Width: 128, Height: 64})
	display.ClearDisplay()
	return &display
}
