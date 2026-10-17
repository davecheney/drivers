//go:build tinygo && badger2040

package main

import (
	"time"

	"tinygo.org/x/drivers/examples/badger2040/internal/board"
	"tinygo.org/x/drivers/uc8151"
)

const stagePause = 5 * time.Second

func traceStage(stage string) {
	println("Badger grayscale:", stage)
	time.Sleep(stagePause)
}

func main() {
	if err := run(); err != nil {
		println("Badger grayscale:", err.Error())
	}
}

func run() error {
	println("gray pulse frames: dark", uc8151.DefaultDarkGrayFrames, "light", uc8151.DefaultLightGrayFrames)
	display, err := board.NewDisplay(traceStage)
	if err != nil {
		return err
	}
	if err := display.ClearDisplay(); err != nil {
		return err
	}
	traceStage("white refresh complete")
	drawSwatches(display)
	if err := display.Display(); err != nil {
		return err
	}
	traceStage("four-shade refresh complete")
	println("left to right: black, dark gray, light gray, white")
	if err := display.DeepSleep(); err != nil {
		return err
	}
	traceStage("deep sleep complete")
	return nil
}
