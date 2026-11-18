//go:build wioterminal && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ili9341"
)

func newDisplay() drivers.Displayer {
	machine.SPI3.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40 * machine.MHz,
	})
	backlight := machine.LCD_BACKLIGHT
	backlight.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display := ili9341.NewSPI(machine.SPI3, machine.LCD_DC, machine.LCD_SS_PIN, machine.LCD_RESET)
	display.Configure(ili9341.Config{})
	display.SetRotation(ili9341.Rotation270)
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	backlight.High()
	return display
}
