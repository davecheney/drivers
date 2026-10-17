//go:build tinygo && badger2040

package main

import "tinygo.org/x/drivers/examples/badger2040/internal/board"

func main() {
	if err := run(); err != nil {
		println("Badger grayscale image:", err.Error())
	}
}

func run() error {
	display, err := board.NewDisplay(nil)
	if err != nil {
		return err
	}
	println("Badger grayscale image: showing Tainigo")
	if err := showImage(display); err != nil {
		return err
	}
	if err := display.DeepSleep(); err != nil {
		return err
	}
	println("Badger grayscale image: complete")
	return nil
}
