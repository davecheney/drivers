// Touch pins GPIO1 to GPIO14 on an ESP32-S3 devkit, and the RGB LED shows a
// different colour for each pin.
package main

import (
	"image/color"
	"machine"
	"machine/touch"
	"time"

	"tinygo.org/x/drivers/ws2812"
)

const numPads = 14

func main() {
	var pads [numPads]touch.Pad
	for i := range pads {
		pads[i] = touch.Pad{Pin: machine.Pin(i + 1)}
		if err := pads[i].Configure(); err != nil {
			println("configure", i+1, err.Error())
		}
	}

	machine.WS2812.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.WS2812.Low()
	time.Sleep(time.Millisecond)
	led := ws2812.NewWS2812(machine.WS2812)
	led.WriteColors([]color.RGBA{{}})

	// Measure the untouched values first, so do not touch the pins at startup.
	time.Sleep(500 * time.Millisecond)
	var base [numPads]uint32
	for n := 0; n < 8; n++ {
		for i := range pads {
			base[i] += pads[i].Get() / 8
		}
		time.Sleep(50 * time.Millisecond)
	}
	led.WriteColors([]color.RGBA{{}})

	last := -1
	for {
		best, bestRise := -1, uint32(0)
		for i := range pads {
			v := pads[i].Get()
			if base[i] == 0 || v <= base[i] {
				continue
			}
			// Touched when the count rises more than 5 percent.
			rise := (v - base[i]) * 1000 / base[i]
			if rise > 50 && rise > bestRise {
				best, bestRise = i, rise
			}
		}
		if best != last {
			c := color.RGBA{}
			if best >= 0 {
				c = hue(best * 360 / numPads)
				println("touch GPIO", best+1)
			} else {
				println("release")
			}
			led.WriteColors([]color.RGBA{c})
			last = best
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// hue returns a dim colour for h in degrees.
func hue(h int) color.RGBA {
	const max = 40
	x := uint8((h % 60) * max / 60)
	switch h / 60 {
	case 0:
		return color.RGBA{max, x, 0, 255}
	case 1:
		return color.RGBA{max - x, max, 0, 255}
	case 2:
		return color.RGBA{0, max, x, 255}
	case 3:
		return color.RGBA{0, max - x, max, 255}
	case 4:
		return color.RGBA{x, 0, max, 255}
	default:
		return color.RGBA{max, 0, max - x, 255}
	}
}
