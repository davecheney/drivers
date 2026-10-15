//go:build tinygo

// Four-shade slideshow for the 2.7in V1 HAT on a Waveshare RP2040-PiZero.
// Flash with tinygo flash -target=waveshare-rp2040-zero ./examples/waveshare-epd/epd2in7gray-slideshow
package main

import (
	"embed"
	"errors"
	"machine"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/waveshare-epd/epd2in7gray"
)

//go:embed assets/*
var assets embed.FS

const slideInterval = 10 * time.Second

func main() {
	time.Sleep(3 * time.Second)
	if err := run(); err != nil {
		println("epd2in7gray slideshow:", err.Error())
	}
}

func run() error {
	names, err := assetNames(assets)
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return errors.New("no 2bpp images embedded; add .bin files to assets")
	}

	if err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 4_000_000,
		SCK:       machine.GPIO10,
		SDO:       machine.GPIO11,
		SDI:       machine.GPIO12,
		Mode:      0,
	}); err != nil {
		return err
	}
	cs, dc, rst, busy := machine.GPIO8, machine.GPIO25, machine.GPIO17, machine.GPIO24
	for _, pin := range []machine.Pin{cs, dc, rst} {
		pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	}
	busy.Configure(machine.PinConfig{Mode: machine.PinInput})
	display := epd2in7gray.New(machine.SPI1, cs, dc, rst, busy)
	if err := display.Configure(epd2in7gray.Config{Rotation: drivers.Rotation270}); err != nil {
		return err
	}
	println("epd2in7gray slideshow: init done, clearing display")
	if err := display.ClearDisplay(); err != nil {
		return err
	}

	for i := 0; ; i = (i + 1) % len(names) {
		name := names[i]
		println("epd2in7gray slideshow: showing", name)
		data, err := assets.ReadFile(name)
		if err == nil {
			err = showAsset(display, data)
		}
		if err != nil {
			println("showing", name, "failed:", err.Error())
		}
		time.Sleep(slideInterval)
	}
}
