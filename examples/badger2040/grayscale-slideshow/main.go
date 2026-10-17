//go:build tinygo && badger2040

package main

import (
	"time"

	"tinygo.org/x/drivers/examples/badger2040/internal/board"
	"tinygo.org/x/drivers/internal/grayimage"
)

const slideInterval = 10 * time.Second

func main() {
	if err := run(); err != nil {
		println("Badger grayscale slideshow:", err.Error())
	}
}

func run() error {
	names, err := slideshowNames(assets)
	if err != nil {
		return err
	}
	display, err := board.NewDisplay(nil)
	if err != nil {
		return err
	}
	for i := 0; ; i = (i + 1) % len(names) {
		name := names[i]
		println("Badger grayscale slideshow: showing", name)
		data, err := assets.ReadFile(name)
		if err != nil {
			return err
		}
		if err := grayimage.Display(display, data); err != nil {
			return err
		}
		time.Sleep(slideInterval)
	}
}
