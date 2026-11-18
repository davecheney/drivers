//go:build ssd1306_128x32

package main

import "tinygo.org/x/drivers/ssd1306"

var ssd1306Config = ssd1306.Config{Address: 0x3c, Width: 128, Height: 32}
