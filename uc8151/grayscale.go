package uc8151

import (
	"errors"
	"image/color"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/pin"
	"tinygo.org/x/drivers/pixel"
)

const (
	DefaultBusyTimeout          = 30 * time.Second
	DefaultLightGrayFrames byte = 5
	DefaultDarkGrayFrames  byte = 9
)

var (
	ErrInvalidRotation        = errors.New("uc8151: unsupported rotation")
	ErrInvalidBusyTimeout     = errors.New("uc8151: busy timeout must not be negative")
	ErrInvalidGrayscaleFrames = errors.New("uc8151: require 0 < light gray frames < dark gray frames < 32")
	ErrBusyTimeout            = errors.New("uc8151: busy timeout")
	ErrNotConfigured          = errors.New("uc8151: grayscale device is not configured")
)

type GrayscaleConfig struct {
	Rotation drivers.Rotation
	// BusyTimeout defaults to DefaultBusyTimeout when zero.
	BusyTimeout time.Duration
	// Gray pulse lengths at 100Hz. Zero selects the corresponding default.
	// Require 0 < LightGrayFrames < DarkGrayFrames < 32 (black).
	LightGrayFrames byte
	DarkGrayFrames  byte
	// OnInitStage may log or pause after each initialization stage.
	OnInitStage func(stage string)
}

// GrayscaleDevice drives the 128x296 UC8151/IL0373 panel in four shades.
// Custom waveforms reset it to white, then apply three different black pulses.
type GrayscaleDevice struct {
	bus                     drivers.SPI
	cs, dc                  pin.Output
	rst                     pin.Output
	busy                    pin.Input
	rotation                drivers.Rotation
	timeout                 time.Duration
	lightFrames, darkFrames byte
	configured              bool
	buffer                  [EPD_WIDTH * EPD_HEIGHT / 4]byte
	tx                      [64]byte
}

var _ drivers.Displayer = (*GrayscaleDevice)(nil)

// NewGrayscale creates a white 2bpp framebuffer without performing I/O.
// The SPI bus and pins must be configured before calling Configure.
func NewGrayscale(bus drivers.SPI, cs, dc, rst pin.Output, busy pin.Input) *GrayscaleDevice {
	d := &GrayscaleDevice{
		bus: bus, cs: cs, dc: dc, rst: rst, busy: busy,
		timeout: DefaultBusyTimeout, lightFrames: DefaultLightGrayFrames, darkFrames: DefaultDarkGrayFrames,
	}
	d.ClearBuffer()
	return d
}

// Configure initializes four-shade output without refreshing the panel.
// It clears the framebuffer. This mode is only for the 128x296 UC8151/IL0373.
func (d *GrayscaleDevice) Configure(cfg GrayscaleConfig) error {
	if cfg.Rotation > drivers.Rotation270 {
		return ErrInvalidRotation
	}
	if cfg.BusyTimeout < 0 {
		return ErrInvalidBusyTimeout
	}
	if cfg.BusyTimeout == 0 {
		cfg.BusyTimeout = DefaultBusyTimeout
	}
	if cfg.LightGrayFrames == 0 {
		cfg.LightGrayFrames = DefaultLightGrayFrames
	}
	if cfg.DarkGrayFrames == 0 {
		cfg.DarkGrayFrames = DefaultDarkGrayFrames
	}
	if cfg.LightGrayFrames >= cfg.DarkGrayFrames || cfg.DarkGrayFrames >= 32 {
		return ErrInvalidGrayscaleFrames
	}
	d.configured = false
	d.rotation = cfg.Rotation
	d.timeout = cfg.BusyTimeout
	d.lightFrames, d.darkFrames = cfg.LightGrayFrames, cfg.DarkGrayFrames
	d.ClearBuffer()
	d.cs.Set(true)
	d.rst.Set(false)
	time.Sleep(10 * time.Millisecond)
	d.rst.Set(true)
	time.Sleep(10 * time.Millisecond)
	if err := d.waitReady(); err != nil {
		return err
	}
	if cfg.OnInitStage != nil {
		cfg.OnInitStage("reset")
	}

	for _, setting := range grayscaleInitialization {
		if err := d.writeRegister(setting.command, setting.data); err != nil {
			return err
		}
		if err := d.waitReady(); err != nil {
			return err
		}
		if cfg.OnInitStage != nil {
			cfg.OnInitStage(setting.stage)
		}
	}
	d.configured = true
	return nil
}

// SetPixel sets one buffered pixel using RGB luminance, ignoring alpha.
func (d *GrayscaleDevice) SetPixel(x, y int16, c color.RGBA) {
	index, shift, ok := d.pixelIndex(x, y)
	if !ok {
		return
	}
	shade := byte(pixel.NewGrayscale2bit(c.R, c.G, c.B))
	d.buffer[index] = d.buffer[index]&^(3<<shift) | shade<<shift
}

func (d *GrayscaleDevice) pixelIndex(x, y int16) (int, uint, bool) {
	w, h := d.Size()
	if x < 0 || y < 0 || x >= w || y >= h {
		return 0, 0, false
	}
	switch d.rotation {
	case drivers.Rotation90:
		x, y = EPD_WIDTH-y-1, x
	case drivers.Rotation180:
		x, y = EPD_WIDTH-x-1, EPD_HEIGHT-y-1
	case drivers.Rotation270:
		x, y = y, EPD_HEIGHT-x-1
	}
	return (int(y)*EPD_WIDTH + int(x)) / 4, uint(6 - (x%4)*2), true
}

// ClearBuffer fills the framebuffer with white without refreshing the panel.
func (d *GrayscaleDevice) ClearBuffer() {
	for i := range d.buffer {
		d.buffer[i] = 0xff
	}
}

// Display resets the panel to white, renders four shades, and powers off.
// Every call performs a full refresh, including when the image is unchanged.
func (d *GrayscaleDevice) Display() error {
	if err := d.ClearDisplay(); err != nil {
		return err
	}
	return d.refresh(false)
}

// ClearDisplay refreshes the panel to white without modifying the framebuffer.
func (d *GrayscaleDevice) ClearDisplay() error {
	return d.refresh(true)
}

func (d *GrayscaleDevice) refresh(white bool) error {
	if !d.configured {
		return ErrNotConfigured
	}
	if err := d.writeLUTs(white); err != nil {
		return err
	}
	return d.refreshPlanes(func(command byte, old bool) error {
		return d.sendPlane(command, old, white)
	})
}

func (d *GrayscaleDevice) refreshPlanes(send func(command byte, old bool) error) error {
	if err := d.writeRegister(PON, nil); err != nil {
		return err
	}
	if err := d.writeRegister(PTOU, nil); err != nil {
		return err
	}
	if err := send(DTM1, true); err != nil {
		return err
	}
	if err := send(DTM2, false); err != nil {
		return err
	}
	if err := d.writeRegister(DRF, nil); err != nil {
		return err
	}
	time.Sleep(time.Millisecond)
	if err := d.writeRegister(POF, nil); err != nil {
		return err
	}
	return d.waitReady()
}

func (d *GrayscaleDevice) sendPlane(command byte, old, white bool) error {
	if err := d.writeRegister(command, nil); err != nil {
		return err
	}
	for offset := 0; offset < len(d.buffer); {
		n := 0
		for n < len(d.tx) && offset < len(d.buffer) {
			d.tx[n] = 0
			if !white {
				d.tx[n] = grayscalePlane(d.buffer[offset], d.buffer[offset+1], old)
			}
			n++
			offset += 2
		}
		if err := d.sendData(d.tx[:n]); err != nil {
			return err
		}
	}
	return d.writeRegister(DSP, nil)
}

// LUT selectors follow set_pixels_for_greyscale in antirez/uc8151_micropython.
// https://github.com/antirez/uc8151_micropython/blob/cc2a0de/uc8151.py
func grayscalePlane(first, second byte, old bool) byte {
	pixels := uint16(first)<<8 | uint16(second)
	var plane byte
	for i := 0; i < 8; i++ {
		shade := byte(pixels >> 14)
		bit := shade >> 1
		if old {
			bit = shade & 1
		}
		plane = plane<<1 | bit
		pixels <<= 2
	}
	if !old {
		plane = ^plane
	}
	return plane
}

// Size returns the fixed panel dimensions after rotation.
func (d *GrayscaleDevice) Size() (w, h int16) {
	if d.rotation == drivers.Rotation90 || d.rotation == drivers.Rotation270 {
		return EPD_HEIGHT, EPD_WIDTH
	}
	return EPD_WIDTH, EPD_HEIGHT
}

func (d *GrayscaleDevice) Rotation() drivers.Rotation {
	return d.rotation
}

// SetRotation changes drawing coordinates without modifying buffered pixels.
func (d *GrayscaleDevice) SetRotation(rotation drivers.Rotation) error {
	if rotation > drivers.Rotation270 {
		return ErrInvalidRotation
	}
	d.rotation = rotation
	return nil
}

// DeepSleep retains the image. Call Configure before using the panel again.
func (d *GrayscaleDevice) DeepSleep() error {
	if !d.configured {
		return ErrNotConfigured
	}
	if err := d.writeRegister(POF, nil); err != nil {
		return err
	}
	if err := d.writeRegister(DSLP, []byte{0xa5}); err != nil {
		return err
	}
	d.configured = false
	return nil
}

func (d *GrayscaleDevice) waitReady() error {
	deadline := time.Now().Add(d.timeout)
	for !d.busy.Get() {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return ErrBusyTimeout
		}
		time.Sleep(min(remaining, 10*time.Millisecond))
	}
	return nil
}

func (d *GrayscaleDevice) writeRegister(command byte, data []byte) error {
	if err := d.waitReady(); err != nil {
		return err
	}
	d.dc.Set(false)
	d.cs.Set(false)
	_, err := d.bus.Transfer(command)
	d.cs.Set(true)
	if err != nil || len(data) == 0 {
		return err
	}
	return d.sendData(data)
}

func (d *GrayscaleDevice) sendData(data []byte) error {
	d.dc.Set(true)
	d.cs.Set(false)
	err := d.bus.Tx(data, nil)
	d.cs.Set(true)
	return err
}
