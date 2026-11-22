// Package ov5647 provides a driver for the OmniVision OV5647 camera sensor,
// as used on the Raspberry Pi Camera v1.3.
//
// The driver sets up the sensor over SCCB (I2C). Capturing pixel data from the
// MIPI CSI-2 bus is specific to each microcontroller and is not part of this
// package.
package ov5647 // import "tinygo.org/x/drivers/ov5647"

import (
	"errors"
	"time"

	"tinygo.org/x/drivers"
)

// Address is the SCCB address of the sensor.
const Address = 0x36

// ChipID is the value of the chip ID registers.
const ChipID = 0x5647

var (
	ErrNotDetected = errors.New("ov5647: sensor not detected")
	ErrFrameSize   = errors.New("ov5647: unsupported frame size")
)

// Config is the sensor configuration.
//
// The sensor runs 800x640 RAW8 at 50 fps over 2 MIPI lanes, with Bayer order
// BGGR. A smaller size is a centred crop of that frame with the same order. The sensor PLL
// settings expect a 25 MHz XCLK, which the Raspberry Pi Camera v1.3 has on the
// camera board.
type Config struct {
	// Width and Height of the output frame. The default is 800x640. Width
	// must be a multiple of 8 and Height a multiple of 4.
	Width  uint16
	Height uint16
}

// LaneRate is the MIPI data rate of each lane in Mbps for the 800x640 mode.
const LaneRate = 400

// Device is an OV5647 connected over SCCB.
type Device struct {
	bus     drivers.I2C
	Address uint16
}

// New returns a new OV5647 driver. The I2C bus must already be configured.
func New(bus drivers.I2C) *Device {
	return &Device{
		bus:     bus,
		Address: Address,
	}
}

// Connected returns whether the sensor responds with the OV5647 chip ID.
func (d *Device) Connected() bool {
	hi, err := d.readReg(regChipIDHigh)
	if err != nil {
		return false
	}
	lo, err := d.readReg(regChipIDLow)
	if err != nil {
		return false
	}
	return uint16(hi)<<8|uint16(lo) == ChipID
}

// Configure resets the sensor and sets the frame size. The stream stays
// off until Start.
func (d *Device) Configure(cfg Config) error {
	if cfg.Width == 0 && cfg.Height == 0 {
		cfg.Width, cfg.Height = 800, 640
	}
	if cfg.Width == 0 || cfg.Width > 800 || cfg.Width%8 != 0 ||
		cfg.Height == 0 || cfg.Height > 640 || cfg.Height%4 != 0 {
		return ErrFrameSize
	}
	if !d.Connected() {
		return ErrNotDetected
	}
	if err := d.writeReg(regModeSelect, 0x00); err != nil {
		return err
	}
	if err := d.reset(); err != nil {
		return err
	}
	// Put the clock lane in LP-11 while the receiver is set up.
	if err := d.writeReg(regMIPICtrl00, 0x01); err != nil {
		return err
	}
	if err := d.writeRegs(mode800x640RAW8); err != nil {
		return err
	}
	if err := d.writeRegs(aeTarget); err != nil {
		return err
	}
	if err := d.setCrop(cfg.Width, cfg.Height); err != nil {
		return err
	}
	return d.Stop()
}

// setCrop sets the output size and centres it in the 800x640 frame.
func (d *Device) setCrop(w, h uint16) error {
	x := 8 + (800-w)/2
	y := (640 - h) / 2
	for _, r := range []struct{ reg, val uint16 }{
		{regXOutputSize, w},
		{regYOutputSize, h},
		{regXOffset, x},
		{regYOffset, y},
	} {
		if err := d.writeReg16(r.reg, r.val); err != nil {
			return err
		}
	}
	return nil
}

// SetTestPattern turns the colour bar test pattern on or off.
func (d *Device) SetTestPattern(on bool) error {
	v := uint8(0)
	if on {
		v = 0x80
	}
	return d.writeReg(regTestPattern, v)
}

// Start starts the MIPI output stream.
func (d *Device) Start() error {
	// Bus idle and clock lane gating, from ov5647.c.
	if err := d.writeReg(regMIPICtrl00, 0x24); err != nil {
		return err
	}
	return d.writeReg(regModeSelect, 0x01)
}

// Stop stops the MIPI output stream.
func (d *Device) Stop() error {
	if err := d.writeReg(regMIPICtrl00, 0x25); err != nil {
		return err
	}
	return d.writeReg(regModeSelect, 0x00)
}

func (d *Device) reset() error {
	if err := d.writeReg(regSoftwareReset, 0x01); err != nil {
		return err
	}
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (d *Device) readReg(r uint16) (uint8, error) {
	var buf [1]byte
	err := d.bus.Tx(d.Address, []byte{byte(r >> 8), byte(r)}, buf[:])
	return buf[0], err
}

func (d *Device) writeReg(r uint16, v uint8) error {
	return d.bus.Tx(d.Address, []byte{byte(r >> 8), byte(r), v}, nil)
}

func (d *Device) writeReg16(r uint16, v uint16) error {
	if err := d.writeReg(r, uint8(v>>8)); err != nil {
		return err
	}
	return d.writeReg(r+1, uint8(v))
}

func (d *Device) writeRegs(regs []reg) error {
	for _, r := range regs {
		if r.addr == regDelay {
			time.Sleep(time.Duration(r.val) * time.Millisecond)
			continue
		}
		if err := d.writeReg(r.addr, r.val); err != nil {
			return err
		}
	}
	return nil
}
