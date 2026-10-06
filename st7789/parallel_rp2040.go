//go:build rp2040 || rp2350

package st7789

import (
	"errors"
	"image/color"
	"machine"
	"time"

	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
	"tinygo.org/x/drivers"
)

// ParallelDevice implements a driver for ST7789 displays connected via an 8-bit
// parallel bus driven by RP2040/RP2350 PIO state machine.
type ParallelDevice struct {
	cs machine.Pin
	dc machine.Pin
	rd machine.Pin
	bl machine.Pin

	pl *piolib.Parallel

	width    int16
	height   int16
	rotation drivers.Rotation

	fb []byte

	buf [6]byte
}

// ParallelConfig holds configuration options for the ST7789 parallel display.
type ParallelConfig struct {
	Width       int16
	Height      int16
	Rotation    drivers.Rotation
	Framebuffer []byte
}

// NewParallel creates a new parallel ST7789 connection.
func NewParallel(pl *piolib.Parallel, csPin, dcPin, rdPin, blPin machine.Pin) ParallelDevice {
	return ParallelDevice{
		pl:     pl,
		cs:     csPin,
		dc:     dcPin,
		rd:     rdPin,
		bl:     blPin,
		width:  320,
		height: 240,
	}
}

// SetFramebuffer sets or replaces the framebuffer slice used by Display and SetPixel.
func (d *ParallelDevice) SetFramebuffer(fb []byte) {
	d.fb = fb
}

// Configure initializes the ST7789 display.
func (d *ParallelDevice) Configure(cfg ParallelConfig) {
	if cfg.Width != 0 {
		d.width = cfg.Width
	} else {
		d.width = 320
	}
	if cfg.Height != 0 {
		d.height = cfg.Height
	} else {
		d.height = 240
	}
	d.rotation = cfg.Rotation
	if cfg.Framebuffer != nil {
		d.fb = cfg.Framebuffer
	}

	// Assume caller configured control pins as outputs (CS high, DC high, RD high)
	// before bringing up the PIO parallel state machine.
	time.Sleep(10 * time.Millisecond)

	d.EnableBacklight(false)

	d.command(SWRESET, nil)
	time.Sleep(150 * time.Millisecond)

	d.command(COLMOD, []byte{0x05})                          // 16 bits per pixel
	d.command(PORCTRL, []byte{0x0c, 0x0c, 0x00, 0x33, 0x33}) // porch intervals
	d.command(LCMCTRL, []byte{0x2c})                         // LCM control
	d.command(VDVVRHEN, []byte{0x01})                        // take VDV and VRH from command registers
	d.command(VRHS, []byte{0x12})                            // VRH ~4.45V
	d.command(VDVS, []byte{0x20})                            // VDV 0V
	d.command(PWCTR1, []byte{0xa4, 0xa1})                    // AVDD 6.8V, AVCL -4.8V, VDS 2.3V
	d.command(FRCTRL2, []byte{0x0f})                         // 60Hz frame rate
	d.command(RAMCTRL, []byte{0x00, 0xc0})                   // MCU interface, big-endian pixel data
	d.command(GCTRL, []byte{0x35})                           // gate voltages VGH 13.26V, VGL -10.43V
	d.command(VCOMS, []byte{0x1b})                           // VCOM 0.875V

	// Gamma correction curves tuned for parallel ST7789 panels
	d.command(GMCTRP1, []byte{0xf0, 0x00, 0x06, 0x04, 0x05, 0x05, 0x31, 0x44, 0x48, 0x36, 0x12, 0x12, 0x2b, 0x34})
	d.command(GMCTRN1, []byte{0xf0, 0x0b, 0x0f, 0x0f, 0x0d, 0x26, 0x31, 0x43, 0x47, 0x38, 0x14, 0x14, 0x2c, 0x32})

	d.command(INVON, nil)
	d.command(SLPOUT, nil)

	time.Sleep(100 * time.Millisecond)

	d.SetRotation(d.rotation)

	d.command(TEON, []byte{0x00})
	d.command(STE, []byte{0x00, 0x00})
	d.command(DISPON, nil)

	if d.bl != machine.NoPin {
		time.Sleep(50 * time.Millisecond)
		d.EnableBacklight(true)
	}
}

// EnableBacklight turns the display backlight on or off using PWM if configured.
func (d *ParallelDevice) EnableBacklight(on bool) {
	if d.bl == machine.NoPin {
		return
	}
	pwm := machine.PWM1
	pwm.Configure(machine.PWMConfig{})
	ch, err := pwm.Channel(d.bl)
	if err != nil {
		return
	}
	if on {
		pwm.Set(ch, pwm.Top())
	} else {
		pwm.Set(ch, 0)
	}
}

// Size returns the current size of the display.
func (d *ParallelDevice) Size() (w, h int16) {
	return d.width, d.height
}

// SetPixel writes a single RGB565 pixel into the framebuffer. Out-of-range
// coordinates are silently ignored to match drivers.Displayer.
func (d *ParallelDevice) SetPixel(x, y int16, c color.RGBA) {
	if x < 0 || y < 0 || x >= d.width || y >= d.height {
		return
	}
	if d.fb == nil {
		return
	}
	c565 := RGBATo565(c)
	i := (int(y)*int(d.width) + int(x)) * 2
	if i+1 < len(d.fb) {
		d.fb[i] = uint8(c565 >> 8) // panel expects high byte first
		d.fb[i+1] = uint8(c565)    // low byte
	}
}

// Display streams the framebuffer to the panel with a single RAMWR.
func (d *ParallelDevice) Display() error {
	if d.fb == nil {
		return nil
	}
	d.setWindow(0, 0, d.width, d.height)

	d.dc.Low()
	d.cs.Low()
	d.pl.Tx8([]byte{RAMWR})
	time.Sleep(10 * time.Microsecond)
	d.dc.High()

	if err := d.pl.Tx8(d.fb); err != nil {
		d.cs.High()
		return err
	}
	time.Sleep(10 * time.Microsecond)
	d.cs.High()
	return nil
}

// FillRectangle fills a rectangle at given coordinates with a color.
func (d *ParallelDevice) FillRectangle(x, y, width, height int16, c color.RGBA) error {
	if x < 0 || y < 0 || width <= 0 || height <= 0 ||
		x >= d.width || (x+width) > d.width || y >= d.height || (y+height) > d.height {
		return errors.New("rectangle coordinates outside display area")
	}
	d.setWindow(x, y, width, height)

	c565 := RGBATo565(c)
	var chunk [512]byte
	for j := 0; j < len(chunk); j += 2 {
		chunk[j] = uint8(c565 >> 8)
		chunk[j+1] = uint8(c565)
	}

	d.dc.Low()
	d.cs.Low()
	d.pl.Tx8([]byte{RAMWR})
	time.Sleep(10 * time.Microsecond)
	d.dc.High()
	remaining := int(width) * int(height) * 2
	for remaining > 0 {
		n := remaining
		if n > len(chunk) {
			n = len(chunk)
		}
		if err := d.pl.Tx8(chunk[:n]); err != nil {
			d.cs.High()
			return err
		}
		remaining -= n
	}
	time.Sleep(10 * time.Microsecond)
	d.cs.High()
	return nil
}

// FillScreen fills the screen with a given color.
func (d *ParallelDevice) FillScreen(c color.RGBA) {
	d.FillRectangle(0, 0, d.width, d.height, c)
}

// Rotation returns the current rotation of the device.
func (d *ParallelDevice) Rotation() drivers.Rotation {
	return d.rotation
}

// SetRotation changes the rotation of the device.
func (d *ParallelDevice) SetRotation(rotation drivers.Rotation) error {
	d.rotation = rotation
	portrait := rotation == drivers.Rotation90 || rotation == drivers.Rotation270
	flipped := rotation == drivers.Rotation180 || rotation == drivers.Rotation270

	if portrait {
		d.width, d.height = 240, 320
	} else {
		d.width, d.height = 320, 240
	}

	var madctl uint8
	if portrait {
		if flipped {
			madctl = MADCTL_MY | MADCTL_MX
		}
	} else {
		if flipped {
			madctl = MADCTL_MX
		} else {
			madctl = MADCTL_MY
		}
		madctl |= MADCTL_MV
	}

	caset := []uint16{0, uint16(d.width - 1)}
	raset := []uint16{0, uint16(d.height - 1)}

	d.command(CASET, []byte{byte(caset[0] >> 8), byte(caset[0]), byte(caset[1] >> 8), byte(caset[1])})
	d.command(RASET, []byte{byte(raset[0] >> 8), byte(raset[0]), byte(raset[1] >> 8), byte(raset[1])})
	d.command(MADCTL, []byte{madctl})
	return nil
}

func (d *ParallelDevice) setWindow(x, y, w, h int16) {
	copy(d.buf[:4], []uint8{uint8(x >> 8), uint8(x), uint8((x + w - 1) >> 8), uint8(x + w - 1)})
	d.command(CASET, d.buf[:4])
	copy(d.buf[:4], []uint8{uint8(y >> 8), uint8(y), uint8((y + h - 1) >> 8), uint8(y + h - 1)})
	d.command(RASET, d.buf[:4])
}

func (d *ParallelDevice) command(command byte, data []byte) {
	d.dc.Low()
	d.cs.Low()
	d.pl.Tx8([]byte{command})

	if len(data) > 0 {
		time.Sleep(10 * time.Microsecond)
		d.dc.High()
		d.pl.Tx8(data)
	}
	time.Sleep(10 * time.Microsecond)
	d.cs.High()
}

// Compile-time interface assertion
var _ drivers.Displayer = (*ParallelDevice)(nil)

// Suppress unused import warning if pio is not directly referenced in build
var _ = pio.PIO0
