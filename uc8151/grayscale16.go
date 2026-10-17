package uc8151

import (
	"errors"
	"image/color"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/pin"
)

var ErrInvalidGrayscale16Frames = errors.New("uc8151: require 15 strictly increasing gray pulse lengths from 1 to 32, ending at 32")

// DefaultGrayscale16Frames returns pulse lengths from lightest non-white to black.
// These are experimental timings, not calibrated perceptual levels.
func DefaultGrayscale16Frames() [15]byte {
	return [15]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 14, 17, 32}
}

type Grayscale16Config struct {
	Rotation    drivers.Rotation
	BusyTimeout time.Duration
	OnInitStage func(stage string)
	// Frames runs from lightest non-white to black at 100Hz. All zero selects defaults.
	// Require strictly increasing positive durations, ending at the fixed 32-frame black.
	Frames [15]byte
}

// Grayscale16Device renders 16 shades using five disjoint masked passes.
// Two packed 2bpp planes hold an 18,944-byte, 4bpp framebuffer.
type Grayscale16Device struct {
	core   *GrayscaleDevice
	high   [EPD_WIDTH * EPD_HEIGHT / 4]byte
	frames [15]byte
}

var _ drivers.Displayer = (*Grayscale16Device)(nil)

// NewGrayscale16 creates a white 4bpp framebuffer without hardware I/O.
func NewGrayscale16(bus drivers.SPI, cs, dc, rst pin.Output, busy pin.Input) *Grayscale16Device {
	d := &Grayscale16Device{
		core:   NewGrayscale(bus, cs, dc, rst, busy),
		frames: DefaultGrayscale16Frames(),
	}
	d.ClearBuffer()
	return d
}

func (d *Grayscale16Device) Configure(cfg Grayscale16Config) error {
	frames := cfg.Frames
	if frames == ([15]byte{}) {
		frames = DefaultGrayscale16Frames()
	}
	previous := byte(0)
	for _, duration := range frames {
		if duration <= previous || duration > 32 {
			return ErrInvalidGrayscale16Frames
		}
		previous = duration
	}
	if previous != 32 {
		return ErrInvalidGrayscale16Frames
	}
	if err := d.core.Configure(GrayscaleConfig{
		Rotation: cfg.Rotation, BusyTimeout: cfg.BusyTimeout, OnInitStage: cfg.OnInitStage,
	}); err != nil {
		return err
	}
	d.frames = frames
	d.ClearBuffer()
	return nil
}

// SetPixel uses RGB luminance for shades 0 (black) through 15 (white), ignoring alpha.
func (d *Grayscale16Device) SetPixel(x, y int16, c color.RGBA) {
	index, shift, ok := d.core.pixelIndex(x, y)
	if !ok {
		return
	}
	shade := byte((77*uint32(c.R) + 150*uint32(c.G) + 29*uint32(c.B)) >> 12)
	d.core.buffer[index] = d.core.buffer[index]&^(3<<shift) | (shade&3)<<shift
	d.high[index] = d.high[index]&^(3<<shift) | (shade>>2)<<shift
}

func (d *Grayscale16Device) ClearBuffer() {
	d.core.ClearBuffer()
	for i := range d.high {
		d.high[i] = 0xff
	}
}

func (d *Grayscale16Device) Size() (int16, int16) {
	return d.core.Size()
}

func (d *Grayscale16Device) Rotation() drivers.Rotation {
	return d.core.Rotation()
}

func (d *Grayscale16Device) SetRotation(rotation drivers.Rotation) error {
	return d.core.SetRotation(rotation)
}

func (d *Grayscale16Device) DeepSleep() error {
	return d.core.DeepSleep()
}

func (d *Grayscale16Device) ClearDisplay() error {
	return d.core.ClearDisplay()
}

// Display resets to white, then drives each non-white pixel in exactly one pass.
// Empty passes are skipped; other pixels receive a no-op waveform between passes.
func (d *Grayscale16Device) Display() error {
	if err := d.ClearDisplay(); err != nil {
		return err
	}
	for first := byte(1); first < 16; first += 3 {
		if !d.hasPass(first) {
			continue
		}
		if err := d.writePassLUTs(first); err != nil {
			return err
		}
		if err := d.core.refreshPlanes(func(command byte, old bool) error {
			return d.sendPassPlane(command, old, first)
		}); err != nil {
			return err
		}
	}
	return nil
}

func (d *Grayscale16Device) hasPass(first byte) bool {
	for i, low := range d.core.buffer {
		high := d.high[i]
		for shift := uint(0); shift < 8; shift += 2 {
			darkness := 15 - ((low>>shift)&3 | ((high>>shift)&3)<<2)
			if darkness >= first && darkness < first+3 {
				return true
			}
		}
	}
	return false
}

// Masking and waveform selection follow update_greyscale in antirez's driver.
// https://github.com/antirez/uc8151_micropython/blob/cc2a0de/uc8151.py
func (d *Grayscale16Device) writePassLUTs(first byte) error {
	vcom := [44]byte{0, d.frames[first+1], 0, 0, 0, 1}
	tables := [4]LUTType{
		{0x55, d.frames[first-1], 0, 0, 0, 1},
		{},
		{0x55, d.frames[first+1], 0, 0, 0, 1},
		{0x55, d.frames[first], 0, 0, 0, 1},
	}
	return d.core.writeGrayscaleLUTs(&vcom, &tables)
}

func (d *Grayscale16Device) sendPassPlane(command byte, old bool, first byte) error {
	if err := d.core.writeRegister(command, nil); err != nil {
		return err
	}
	for offset := 0; offset < len(d.core.buffer); {
		n := 0
		for n < len(d.core.tx) && offset < len(d.core.buffer) {
			low := uint16(d.core.buffer[offset])<<8 | uint16(d.core.buffer[offset+1])
			high := uint16(d.high[offset])<<8 | uint16(d.high[offset+1])
			d.core.tx[n] = grayscale16Plane(low, high, first, old)
			n++
			offset += 2
		}
		if err := d.core.sendData(d.core.tx[:n]); err != nil {
			return err
		}
	}
	return d.core.writeRegister(DSP, nil)
}

func grayscale16Plane(low, high uint16, first byte, old bool) byte {
	var selectors uint16
	for i := 0; i < 8; i++ {
		darkness := 15 - (byte(low>>14) | byte(high>>14)<<2)
		selector := byte(3)
		if darkness >= first && darkness < first+3 {
			selector = 2 - (darkness - first)
		}
		selectors = selectors<<2 | uint16(selector)
		low <<= 2
		high <<= 2
	}
	return grayscalePlane(byte(selectors>>8), byte(selectors), old)
}
