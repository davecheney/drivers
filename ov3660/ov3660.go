// Package ov3660 provides a driver for the OmniVision OV3660 camera sensor.
//
// The driver sets up the sensor over SCCB (I2C). Capturing pixel data from the
// parallel DVP bus is specific to each microcontroller and is not part of this
// package.
//
// The register settings come from the Espressif esp32-camera driver:
// https://github.com/espressif/esp32-camera/blob/master/sensors/ov3660.c
package ov3660 // import "tinygo.org/x/drivers/ov3660"

import (
	"errors"
	"time"

	"tinygo.org/x/drivers"
)

// Address is the SCCB address of the sensor.
const Address = 0x3C

// ChipID is the value of the chip ID registers.
const ChipID = 0x3660

// PixelFormat is the format of the pixel data on the DVP bus.
type PixelFormat uint8

const (
	// RGB565 sends each pixel as 2 bytes, big endian.
	RGB565 PixelFormat = iota
	// YUV422 sends pixels as YUYV.
	YUV422
	// Grayscale sends 1 byte of luminance for each pixel.
	Grayscale
)

var (
	ErrNotDetected = errors.New("ov3660: sensor not detected")
	ErrFrameSize   = errors.New("ov3660: unsupported frame size")
)

// Config is the sensor configuration.
//
// The sensor PLL settings expect a 16 MHz XCLK.
type Config struct {
	// Width and Height of the output frame. The aspect ratio must be one of
	// 4:3, 3:2, 16:10, 5:3, 16:9, 21:9, 5:4, 1:1 or 9:16. The default is
	// 320x240.
	Width  uint16
	Height uint16

	Format PixelFormat
}

// Device is an OV3660 connected over SCCB.
type Device struct {
	bus     drivers.I2C
	Address uint16

	binning bool
	vflip   bool
	hmirror bool
}

// New returns a new OV3660 driver. The I2C bus must already be configured and
// XCLK must already run.
func New(bus drivers.I2C) *Device {
	return &Device{
		bus:     bus,
		Address: Address,
	}
}

// Connected returns whether the sensor responds with the OV3660 chip ID.
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

// Configure resets the sensor and sets the frame size and pixel format.
func (d *Device) Configure(cfg Config) error {
	if cfg.Width == 0 || cfg.Height == 0 {
		cfg.Width, cfg.Height = 320, 240
	}
	ratio, ok := findRatio(cfg.Width, cfg.Height)
	if !ok {
		return ErrFrameSize
	}
	if !d.Connected() {
		return ErrNotDetected
	}

	if err := d.writeReg(regSystemCtrl0, 0x82); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)
	if err := d.writeRegs(defaultRegs); err != nil {
		return err
	}
	if err := d.setAELevel(0); err != nil {
		return err
	}
	time.Sleep(100 * time.Millisecond)

	if err := d.setFormat(cfg.Format); err != nil {
		return err
	}
	return d.setFrameSize(cfg.Width, cfg.Height, ratio)
}

// SetFlip sets vertical flip and horizontal mirror.
func (d *Device) SetFlip(vflip, hmirror bool) error {
	d.vflip = vflip
	d.hmirror = hmirror
	return d.setImageOptions()
}

func (d *Device) setFormat(f PixelFormat) error {
	switch f {
	case RGB565:
		return d.writeRegs([]reg{{regFormatCtrl, 0x01}, {regFormatCtrl00, 0x61}})
	case YUV422:
		return d.writeRegs([]reg{{regFormatCtrl, 0x00}, {regFormatCtrl00, 0x30}})
	case Grayscale:
		return d.writeRegs([]reg{{regFormatCtrl, 0x00}, {regFormatCtrl00, 0x10}})
	}
	return errors.New("ov3660: unsupported pixel format")
}

func findRatio(w, h uint16) (ratioSettings, bool) {
	for _, r := range ratioTable {
		if uint32(w)*uint32(r.maxHeight) == uint32(h)*uint32(r.maxWidth) && w <= r.maxWidth {
			return r, true
		}
	}
	return ratioSettings{}, false
}

func (d *Device) setFrameSize(w, h uint16, r ratioSettings) error {
	d.binning = w <= r.maxWidth/2 && h <= r.maxHeight/2
	scale := !(w == r.maxWidth && h == r.maxHeight) && !(w == r.maxWidth/2 && h == r.maxHeight/2)

	totalY := r.totalY
	offX, offY := r.offsetX, r.offsetY
	if d.binning {
		totalY = r.totalY/2 + 1
		offX, offY = 8, 2
	}
	for _, v := range [...]struct{ reg, val uint16 }{
		{regXAddrStart, r.startX}, {regYAddrStart, r.startY},
		{regXAddrEnd, r.endX}, {regYAddrEnd, r.endY},
		{regXOutputSize, w}, {regYOutputSize, h},
		{regXTotalSize, r.totalX}, {regYTotalSize, totalY},
		{regXOffset, offX}, {regYOffset, offY},
	} {
		if err := d.writeReg16(v.reg, v.val); err != nil {
			return err
		}
	}

	isp, err := d.readReg(regISPControl01)
	if err != nil {
		return err
	}
	if scale {
		isp |= 0x20
	} else {
		isp &^= 0x20
	}
	if err := d.writeReg(regISPControl01, isp); err != nil {
		return err
	}
	if err := d.setImageOptions(); err != nil {
		return err
	}

	switch pixels := uint32(w) * uint32(h); {
	case pixels > 480*320:
		// 8 MHz SYSCLK and 8 MHz PCLK
		return d.setPLL(4, 1, 0, 2, 2)
	case pixels >= 320*240:
		// 16 MHz SYSCLK and 8 MHz PCLK
		return d.setPLL(8, 1, 0, 2, 4)
	default:
		// 32 MHz SYSCLK and 8 MHz PCLK
		return d.setPLL(8, 1, 0, 0, 8)
	}
}

func (d *Device) setImageOptions() error {
	var reg20, reg21, test uint8
	if d.binning {
		reg20 |= 0x01
		reg21 |= 0x01
		test |= 4
	} else {
		reg20 |= 0x40
	}
	if d.vflip {
		reg20 |= 0x06
		test |= 1
	}
	if d.hmirror {
		reg21 |= 0x06
		test |= 2
	}
	reg4514 := [8]uint8{0x88, 0x88, 0xbb, 0xbb, 0xaa, 0xbb, 0xbb, 0xaa}[test]

	regs := []reg{
		{regTimingTC20, uint16(reg20)},
		{regTimingTC21, uint16(reg21)},
		{0x4514, uint16(reg4514)},
		{0x4520, 0xb0},
		{regXIncrement, 0x11},
		{regYIncrement, 0x11},
	}
	if d.binning {
		regs[3].val = 0x0b
		regs[4].val = 0x31
		regs[5].val = 0x31
	}
	return d.writeRegs(regs)
}

func (d *Device) setPLL(multiplier, sysDiv, preDiv, seld5, pclkDiv uint8) error {
	return d.writeRegs([]reg{
		{regPLLCtrl0, 0x00},
		{regPLLCtrl1, uint16(multiplier & 0x1f)},
		{regPLLCtrl2, uint16(0x10 | sysDiv&0x0f)},
		{regPLLCtrl3, uint16((preDiv&0x3)<<4 | seld5)},
		{regPCLKRatio, uint16(pclkDiv & 0x1f)},
		{regVFIFOCtrl0C, 0x22},
	})
}

func (d *Device) setAELevel(level int) error {
	target := (level+5)*10 + 5
	low := target * 23 / 25
	high := target * 27 / 25
	fastLow := low >> 1
	fastHigh := min(high<<1, 255)
	return d.writeRegs([]reg{
		{0x3a0f, uint16(high)},
		{0x3a10, uint16(low)},
		{0x3a1b, uint16(high)},
		{0x3a1e, uint16(low)},
		{0x3a11, uint16(fastHigh)},
		{0x3a1f, uint16(fastLow)},
	})
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
		if err := d.writeReg(r.addr, uint8(r.val)); err != nil {
			return err
		}
	}
	return nil
}
