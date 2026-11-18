//go:build pinetime && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7789"
)

var display st7789.Device

func newDisplay() drivers.Displayer {
	machine.SPI0.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK,
		SDO:       machine.LCD_SDI,
		SDI:       machine.NoPin,
		Frequency: 8 * machine.MHz,
		Mode:      3,
	})
	// The backlight pins are active low. The driver drives LCD_BACKLIGHT_LOW
	// high, so it stays off and LCD_BACKLIGHT_HIGH sets full brightness.
	display = st7789.New(machine.SPI0, machine.LCD_RESET, machine.LCD_RS, machine.LCD_CS, machine.LCD_BACKLIGHT_LOW)
	display.Configure(st7789.Config{Width: 240, Height: 240, Rotation: st7789.NO_ROTATION})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	backlight := machine.LCD_BACKLIGHT_HIGH
	backlight.Configure(machine.PinConfig{Mode: machine.PinOutput})
	backlight.Low()
	return &display
}
