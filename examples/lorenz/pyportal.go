//go:build pyportal && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ili9341"
)

func newDisplay() drivers.Displayer {
	display := ili9341.NewParallel(machine.LCD_DATA0, machine.TFT_WR, machine.TFT_DC, machine.TFT_CS, machine.TFT_RESET, machine.TFT_RD)
	backlight := machine.TFT_BACKLIGHT
	backlight.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display.Configure(ili9341.Config{})
	display.SetRotation(ili9341.Rotation270)
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	backlight.High()
	return display
}
