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
	// Front buttons are active HIGH in Pimoroni's Badger2040::init().
	// https://github.com/pimoroni/pimoroni-pico/blob/bbb1792/libraries/badger2040/badger2040.cpp
	for _, pin := range []machine.Pin{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B} {
		pin.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}
	pins := buttonPins{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B}
	machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display, err := board.NewDisplay(nil)
	if err != nil {
		return err
	}
	println("Badger grayscale gradient: white to black, 8x8 Bayer, four shades")
	println("Badger button mask (HIGH=pressed):", pins.Read())
	cal := defaultCalibration()
	if err := updateDisplay(display, cal); err != nil {
		return err
	}
	var debounce buttonDebouncer
	for {
		time.Sleep(10 * time.Millisecond)
		if cal.adjust(debounce.Update(pins.Read(), time.Now())) {
			if err := updateDisplay(display, cal); err != nil {
				return err
			}
		}
	}
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
