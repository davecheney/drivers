//go:build ssd1306_64x48

package main

import "tinygo.org/x/drivers/ssd1306"

// A 64x48 panel shows columns 32 to 95 of the 128 column controller RAM.
var ssd1306Config = ssd1306.Config{Address: 0x3c, Width: 64, Height: 48, ResetCol: ssd1306.ResetValue{32, 95}}
