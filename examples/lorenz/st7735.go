//go:build st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7735"
)

var display st7735.Device

func newDisplay() drivers.Displayer {
	lcdBus.Configure(machine.SPIConfig{
		Frequency: 40 * machine.MHz,
		SCK:       lcdSCK,
		SDO:       lcdSDO,
		SDI:       machine.NoPin,
	})
	display = st7735.New(lcdBus, lcdRST, lcdDC, lcdCS, lcdBL)
	// The 128x128 panel sits in the 132x162 controller memory.
	display.Configure(st7735.Config{Width: 128, Height: 128, Rotation: drivers.Rotation270, RowOffset: 1, ColumnOffset: 2})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return &display
}
