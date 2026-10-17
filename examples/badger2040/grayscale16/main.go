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
		println("Badger 4bpp:", err.Error())
	}
}

func run() error {
	for _, pin := range []machine.Pin{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B} {
		pin.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}
	pins := controls.Pins{machine.BUTTON_UP, machine.BUTTON_DOWN, machine.BUTTON_A, machine.BUTTON_B}
	machine.LED.Configure(machine.PinConfig{Mode: machine.PinOutput})
	display, err := board.NewDisplay16(nil)
	if err != nil {
		return err
	}
	println("Badger 4bpp: sixteen shades, five masked passes")
	println("Badger button mask (HIGH=pressed):", pins.Read())
	cal := defaultCalibration()
	if err := updateDisplay(display, cal); err != nil {
		return err
	}
	var debounce controls.Debouncer
	for {
		time.Sleep(10 * time.Millisecond)
		if cal.adjust(debounce.Update(pins.Read(), time.Now())) {
			if err := updateDisplay(display, cal); err != nil {
				return err
			}
		}
	}
}

func updateDisplay(display testerController, cal calibration) error {
	machine.LED.High()
	defer machine.LED.Low()
	minimum, maximum := cal.limits()
	println("Badger 4bpp: selected shade", cal.shade, "frames", cal.frames[14-cal.shade], "range", minimum, maximum)
	print("pulse frames, light to dark:")
	for _, duration := range cal.frames {
		print(" ", duration)
	}
	println()
	if err := refreshTester(display, cal); err != nil {
		return err
	}
	println("Badger 4bpp: ready")
	return nil
}
