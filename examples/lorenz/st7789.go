//go:build st7789

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7789"
)

var display st7789.Device

func newDisplay() drivers.Displayer {
	lcdBus.Configure(machine.SPIConfig{
		Frequency: 40 * machine.MHz,
		SCK:       lcdSCK,
		SDO:       lcdSDO,
		SDI:       machine.NoPin,
	})
	display = st7789.New(lcdBus, lcdRST, lcdDC, lcdCS, lcdBL)
	display.Configure(st7789.Config{Width: 240, Height: 240, Rotation: drivers.Rotation90})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return &display
}
