//go:build gopher_badge && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7789"
)

var display st7789.Device

func newDisplay() drivers.Displayer {
	machine.SPI0.Configure(machine.SPIConfig{Frequency: 8 * machine.MHz})
	display = st7789.New(machine.SPI0, machine.TFT_RST, machine.TFT_WRX, machine.TFT_CS, machine.TFT_BACKLIGHT)
	display.Configure(st7789.Config{Rotation: st7789.ROTATION_270, Height: 320})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return &display
}
