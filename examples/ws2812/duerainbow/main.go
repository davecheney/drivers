//go:build arduino_due

// Shows a moving rainbow on a strip of 10 WS2812 LEDs connected to D6 on an
// Arduino Due.
package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

var leds [10]color.RGBA

func main() {
	neo := machine.D6
	neo.Configure(machine.PinConfig{Mode: machine.PinOutput})
	ws := ws2812.NewWS2812(neo)

	for offset := 0; ; offset = (offset + 3) % 768 {
		for i := range leds {
			leds[i] = wheel((offset + i*768/len(leds)) % 768)
		}
		ws.WriteColors(leds[:])
		time.Sleep(20 * time.Millisecond)
	}
}

// wheel maps 0 to 767 around red, green and blue at a quarter brightness.
func wheel(p int) color.RGBA {
	v := uint8(p % 256 / 4)
	switch p / 256 {
	case 0:
		return color.RGBA{R: 63 - v, G: v}
	case 1:
		return color.RGBA{G: 63 - v, B: v}
	default:
		return color.RGBA{B: 63 - v, R: v}
	}
}
