//go:build clue_alpha && !ssd1306 && !ssd1306_128x32 && !ssd1306_64x48 && !ili9341 && !ili9325 && !st7789 && !st7735

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7789"
)

var display st7789.Device

func newDisplay() drivers.Displayer {
	machine.SPI1.Configure(machine.SPIConfig{
		SCK:       machine.TFT_SCK,
		SDO:       machine.TFT_SDO,
		SDI:       machine.NoPin,
		Frequency: 8 * machine.MHz,
	})
	display = st7789.New(machine.SPI1, machine.TFT_RESET, machine.TFT_DC, machine.TFT_CS, machine.TFT_LITE)
	display.Configure(st7789.Config{Rotation: st7789.NO_ROTATION, RowOffset: 80})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return &display
}
