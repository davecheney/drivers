//go:build ssd1306 || ssd1306_128x32 || ssd1306_64x48

package main

import (
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ssd1306"
)

func newDisplay() drivers.Displayer {
	machine.I2C0.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})
	display := ssd1306.NewI2C(machine.I2C0)
	display.Configure(ssd1306Config)
	display.ClearDisplay()
	return display
}
