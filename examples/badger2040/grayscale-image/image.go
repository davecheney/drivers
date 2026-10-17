package main

import (
	_ "embed"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/grayimage"
)

//go:embed assets/tainigo.bin
var testImage []byte

func showImage(display drivers.Displayer) error {
	return grayimage.Display(display, testImage)
}
