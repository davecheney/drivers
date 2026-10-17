//go:build tinygo && badger2040

package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/examples/badger2040/internal/board"
	"tinygo.org/x/drivers/examples/badger2040/internal/controls"
)

func main() {
	if err := run(); err != nil {
		println("Badger 2bpp:", err.Error())
	}
}

func run() error {
	for _, pin := range []machine.Pin{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B} {
		pin.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}
	pins := controls.Pins{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B}
	machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display, err := board.NewDisplay(nil)
	if err != nil {
		return err
	}
	println("Badger 2bpp: photo, Floyd-Steinberg, blind calibration")
	println("Badger button mask (HIGH=pressed):", pins.Read())
	cal := defaultCalibration()
	if err := updateDisplay(display, cal); err != nil {
		return err
	}
	var debounce controls.Debouncer
	for {
		time.Sleep(10 * time.Millisecond)
		pressed := debounce.Update(pins.Read(), time.Now())
		if pressed == 0 {
			continue
		}
		if cal.adjust(pressed) {
			if err := updateDisplay(display, cal); err != nil {
				return err
			}
		} else {
			println("Badger 2bpp: unchanged, dark", cal.dark, "light", cal.light)
		}
	}
}

func updateDisplay(display testerController, cal calibration) error {
	machine.LED.High()
	defer machine.LED.Low()
	println("Badger 2bpp: dark", cal.dark, "light", cal.light)
	if err := refreshTester(display, cal); err != nil {
		return err
	}
	println("Badger 2bpp: ready")
	return nil
}
