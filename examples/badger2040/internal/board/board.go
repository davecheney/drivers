//go:build tinygo && badger2040

package board

import (
	"machine"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/uc8151"
)

func configureHardware() error {
	// Keep battery power on, as in Pimoroni's Badger2040::init().
	// https://github.com/pimoroni/pimoroni-pico/blob/bbb1792/libraries/badger2040/badger2040.cpp
	machine.ENABLE_3V3.Configure(machine.PinConfig{Mode: machine.PinOutput})
	machine.ENABLE_3V3.High()
	time.Sleep(3 * time.Second)
	if err := machine.SPI0.Configure(machine.SPIConfig{
		Frequency: 12 * machine.MHz,
		SCK:       machine.EPD_SCK_PIN,
		SDO:       machine.EPD_SDO_PIN,
		Mode:      0,
	}); err != nil {
		return err
	}
	cs, dc, rst, busy := machine.EPD_CS_PIN, machine.EPD_DC_PIN, machine.EPD_RESET_PIN, machine.EPD_BUSY_PIN
	for _, p := range []machine.Pin{cs, dc, rst} {
		p.Configure(machine.PinConfig{Mode: machine.PinOutput})
	}
	busy.Configure(machine.PinConfig{Mode: machine.PinInputPullup})
	return nil
}

func NewDisplay(trace func(string)) (*uc8151.GrayscaleDevice, error) {
	if err := configureHardware(); err != nil {
		return nil, err
	}
	display := uc8151.NewGrayscale(machine.SPI0, machine.EPD_CS_PIN, machine.EPD_DC_PIN, machine.EPD_RESET_PIN, machine.EPD_BUSY_PIN)
	if err := display.Configure(uc8151.GrayscaleConfig{
		Rotation:    drivers.Rotation270,
		OnInitStage: trace,
	}); err != nil {
		return nil, err
	}
	return display, nil
}

func NewDisplay16(trace func(string)) (*uc8151.Grayscale16Device, error) {
	if err := configureHardware(); err != nil {
		return nil, err
	}
	display := uc8151.NewGrayscale16(machine.SPI0, machine.EPD_CS_PIN, machine.EPD_DC_PIN, machine.EPD_RESET_PIN, machine.EPD_BUSY_PIN)
	if err := display.Configure(uc8151.Grayscale16Config{
		Rotation: drivers.Rotation270, OnInitStage: trace,
	}); err != nil {
		return nil, err
	}
	return display, nil
}
