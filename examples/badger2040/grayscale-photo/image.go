package main

import (
	_ "embed"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/grayimage"
)

//go:embed assets/test-photo.bin
var testImage []byte

func showTester(display drivers.Displayer) error {
	return grayimage.Display(display, testImage)
}
