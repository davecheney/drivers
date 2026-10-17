//go:build tinygo && badger2040

package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/examples/badger2040/internal/board"
)

func main() {
	if err := run(); err != nil {
		println("Badger grayscale gradient:", err.Error())
	}
}

func run() error {
	for _, pin := range []machine.Pin{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B} {
		pin.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	}
	machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display, err := board.NewDisplay(nil)
	if err != nil {
		return err
	}
	println("Badger grayscale gradient: white to black, 8x8 Bayer, four shades")
	cal := defaultCalibration()
	if err := updateDisplay(display, cal); err != nil {
		return err
	}
	var debounce buttonDebouncer
	for {
		time.Sleep(10 * time.Millisecond)
		if cal.adjust(debounce.update(readButtons(), time.Now())) {
			if err := updateDisplay(display, cal); err != nil {
				return err
			}
		}
	}
}

func readButtons() buttons {
	var pressed buttons
	for i, pin := range [...]machine.Pin{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B} {
		if !pin.Get() {
			pressed |= 1 << uint(i)
		}
	}
	return pressed
}

func updateDisplay(display gradientController, cal calibration) error {
	machine.LED.High()
	defer machine.LED.Low()
	println("Badger gray frames: dark", cal.dark, "light", cal.light)
	if err := refreshGradient(display, cal); err != nil {
		return err
	}
	println("Badger grayscale gradient: ready")
	return nil
}
