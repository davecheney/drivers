//go:build (st7789 || st7735) && (waveshare_rp2040_pizero || waveshare_rp2350_pizero)

package main

import "machine"

// Waveshare 1.3inch (st7789) and 1.44inch (st7735) LCD HATs on the PiZero
// 40 pin header. https://www.waveshare.com/wiki/1.3inch_LCD_HAT
var (
	lcdBus = machine.SPI1
	lcdSCK = machine.SPI1_SCK_PIN
	lcdSDO = machine.SPI1_SDO_PIN
	lcdRST = machine.GPIO27
	lcdDC  = machine.GPIO25
	lcdCS  = machine.GPIO8
	lcdBL  = machine.GPIO24
)
