package ssd1306

import (
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/legacy"
	"tinygo.org/x/drivers/internal/pin"
)

type SPIBus struct {
	wire     drivers.SPI
	dcPin    pin.Output
	resetPin pin.Output
	csPin    pin.Output
	buffer   []byte
	cmd      [1]byte
}

// NewSPI creates a new SSD1306 connection. The SPI wire must already be configured.
func NewSPI(bus drivers.SPI, dcPin, resetPin, csPin pin.Output) *Device {
	// configure GPIO pins (on baremetal targets only, for backwards compatibility)
	legacy.ConfigurePinOut(dcPin)
	legacy.ConfigurePinOut(resetPin)
	legacy.ConfigurePinOut(csPin)
	return &Device{
		bus: &SPIBus{
			wire:     bus,
			dcPin:    dcPin,
			resetPin: resetPin,
			csPin:    csPin,
		},
	}
}

// configure pins with the SPI bus and allocate the buffer
func (b *SPIBus) configure(address uint16, buffer []byte, size int16) []byte {
	b.csPin.Set(false)
	b.dcPin.Set(false)
	b.resetPin.Set(false)

	b.resetPin.Set(true)
	time.Sleep(1 * time.Millisecond)
	b.resetPin.Set(false)
	time.Sleep(10 * time.Millisecond)
	b.resetPin.Set(true)

	b.buffer = buffer
	if len(b.buffer) != int(size) {
		b.buffer = make([]byte, size)
	}
	return b.buffer
}

// command sends a command to the display
func (b *SPIBus) command(cmd uint8) error {
	b.cmd[0] = cmd
	return b.tx(b.cmd[:], true)
}

// flush sends the image to the display
func (b *SPIBus) flush() error {
	return b.tx(b.buffer, false)
}

// tx sends data to the display
func (b *SPIBus) tx(data []byte, isCommand bool) error {
	b.csPin.Set(true)
	b.dcPin.Set(!isCommand)
	b.csPin.Set(false)
	err := b.wire.Tx(data, nil)
	b.csPin.Set(true)
	return err
}
