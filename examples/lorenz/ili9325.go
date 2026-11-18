//go:build ili9325

package main

import (
	"image/color"
	"machine"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ili9325"
)

// Pins of the ITead ITDB02-2.8 Arduino shield.
var (
	data = [8]machine.Pin{machine.D0, machine.D1, machine.D2, machine.D3, machine.D4, machine.D5, machine.D6, machine.D7}
	rs   = machine.ADC5
	wr   = machine.ADC4
	cs   = machine.ADC3
	rst  = machine.ADC2
)

func newDisplay() drivers.Displayer {
	out := machine.PinConfig{Mode: machine.PinOutput}
	for _, p := range append(data[:], rs, wr, cs, rst) {
		p.Configure(out)
	}
	display := ili9325.New(newBus(), rs, cs, rst)
	display.Configure(ili9325.Config{Rotation: drivers.Rotation90})
	display.FillScreen(color.RGBA{0, 0, 0, 255})
	return display
}
