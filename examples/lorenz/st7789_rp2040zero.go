//go:build st7789 && waveshare_rp2040_zero

package main

import "machine"

// A 240x240 ST7789 wired to the RP2040-Zero SPI1 pins.
var (
	lcdBus = machine.SPI1
	lcdSCK = machine.GPIO10
	lcdSDO = machine.GPIO11
	lcdRST = machine.GPIO9
	lcdDC  = machine.GPIO12
	lcdCS  = machine.GPIO13
	lcdBL  = machine.GPIO14
)
