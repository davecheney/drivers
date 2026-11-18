//go:build (st7789 || st7735) && esp32p4

package main

import "machine"

// Waveshare LCD HATs on the ESP32-P4-Module-DEV-KIT 40 pin header.
// https://www.waveshare.com/wiki/ESP32-P4-Module-DEV-KIT-StartPage
var (
	lcdBus = machine.SPI2
	lcdSCK = machine.GPIO0
	lcdSDO = machine.GPIO3
	lcdRST = machine.GPIO20
	lcdDC  = machine.GPIO1
	lcdCS  = machine.GPIO36
	lcdBL  = machine.GPIO4
)
