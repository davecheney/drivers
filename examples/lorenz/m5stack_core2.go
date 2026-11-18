//go:build m5stack_core2 && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	axp192 "tinygo.org/x/drivers/axp192/m5stack-core2-axp192"
	"tinygo.org/x/drivers/i2csoft"
	"tinygo.org/x/drivers/ili9341"
)

func newDisplay() drivers.Displayer {
	machine.SPI2.Configure(machine.SPIConfig{
		SCK:       machine.LCD_SCK_PIN,
		SDO:       machine.LCD_SDO_PIN,
		SDI:       machine.LCD_SDI_PIN,
		Frequency: 40 * machine.MHz,
	})
	// The AXP192 power chip drives the backlight.
	i2c := i2csoft.New(machine.SCL0_PIN, machine.SDA0_PIN)
	i2c.Configure(i2csoft.I2CConfig{Frequency: 100 * machine.KHz})
	axp192.New(i2c)
	display := ili9341.NewSPI(machine.SPI2, machine.LCD_DC_PIN, machine.LCD_SS_PIN, machine.NoPin)
	display.Configure(ili9341.Config{Width: 320, Height: 240, DisplayInversion: true})
	display.SetRotation(ili9341.Rotation0Mirror)
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return display
}
