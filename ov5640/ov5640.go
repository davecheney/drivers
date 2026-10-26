// Package ov5640 provides OV5640 camera control over I2C.
// See the OV5640 datasheet linked in README.md, sections 2.6 through 2.9.
package ov5640

import (
	"errors"
	"time"

	"tinygo.org/x/drivers"
)

const (
	Address = 0x3c
	ID      = 0x5640

	regSystemControl = 0x3008
	regChipIDHigh    = 0x300a
	regChipIDLow     = 0x300b

	softwareReset   = 1 << 7
	softwareStandby = 1 << 6
)

var ErrUnexpectedID = errors.New("ov5640: unexpected chip ID")

type Device struct {
	bus     drivers.I2C
	Address uint16
	buf     [3]byte
}

func New(bus drivers.I2C) *Device {
	return &Device{bus: bus, Address: Address}
}

func (d *Device) ReadRegister(register uint16) (uint8, error) {
	d.buf[0] = byte(register >> 8)
	d.buf[1] = byte(register)
	// SCCB requires separate address and read cycles.
	// See the SCCB specification, sections 3.2.1.2 and 3.2.1.3.
	if err := d.bus.Tx(d.Address, d.buf[:2], nil); err != nil {
		return 0, err
	}
	if err := d.bus.Tx(d.Address, nil, d.buf[:1]); err != nil {
		return 0, err
	}
	return d.buf[0], nil
}

func (d *Device) WriteRegister(register uint16, value uint8) error {
	d.buf[0] = byte(register >> 8)
	d.buf[1] = byte(register)
	d.buf[2] = value
	return d.bus.Tx(d.Address, d.buf[:], nil)
}

func (d *Device) ChipID() (uint16, error) {
	high, err := d.ReadRegister(regChipIDHigh)
	if err != nil {
		return 0, err
	}
	low, err := d.ReadRegister(regChipIDLow)
	if err != nil {
		return 0, err
	}
	return uint16(high)<<8 | uint16(low), nil
}

func (d *Device) CheckID() error {
	id, err := d.ChipID()
	if err != nil {
		return err
	}
	if id != ID {
		return ErrUnexpectedID
	}
	return nil
}

func (d *Device) Reset() error {
	value, err := d.ReadRegister(regSystemControl)
	if err != nil {
		return err
	}
	if err := d.WriteRegister(regSystemControl, value|softwareReset); err != nil {
		return err
	}
	// Software reset settles in less than 1 ms.
	// See the OV5640 datasheet, table 8-4.
	time.Sleep(time.Millisecond)
	return nil
}

func (d *Device) SetStandby(enabled bool) error {
	value, err := d.ReadRegister(regSystemControl)
	if err != nil {
		return err
	}
	value &^= softwareStandby
	if enabled {
		value |= softwareStandby
	}
	return d.WriteRegister(regSystemControl, value)
}
