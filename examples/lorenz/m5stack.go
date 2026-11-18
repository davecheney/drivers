//go:build m5stack && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ili9341"
)

func newDisplay() drivers.Displayer {
	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.SPI0_SCK_PIN,
		SDO:       machine.SPI0_SDO_PIN,
		SDI:       machine.SPI0_SDI_PIN,
		Frequency: 40 * machine.MHz,
	})
	backlight := machine.LCD_BL_PIN
	backlight.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display := ili9341.NewSPI(machine.SPI2, machine.LCD_DC_PIN, machine.LCD_SS_PIN, machine.LCD_RST_PIN)
	display.Configure(ili9341.Config{Width: 320, Height: 240, DisplayInversion: true})
	display.SetRotation(ili9341.Rotation0Mirror)
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	backlight.High()
	return display
}
