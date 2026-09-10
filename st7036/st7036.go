// Package st7036 implements a driver for the Sitronix ST7036 character LCD
// controller, as used on the Pimoroni Display-o-Tron HAT and Display-o-Tron
// 3000.
//
// The controller talks over a 3-wire SPI link: a register-select pin picks
// command mode or data mode, and each byte is sent as its own chip-select
// pulse.
//
// Datasheet: https://www.crystalfontz.com/controllers/Sitronix/ST7036/
//
// This driver ports the register sequence used by the Pimoroni Python
// driver: https://github.com/pimoroni/st7036
package st7036 // import "tinygo.org/x/drivers/st7036"

import (
	"errors"
	"machine"
	"time"

	"tinygo.org/x/drivers"
)

var (
	errInvalidContrast = errors.New("st7036: contrast must be in the range 0..0x3F")
	errInvalidPosition = errors.New("st7036: row and column must be within the display size")
	errInvalidCharSlot = errors.New("st7036: character slot must be in the range 0..7")
)

// Config holds the display geometry. Both Pimoroni boards use a 16 column,
// 3 row display.
type Config struct {
	Rows    uint8 // number of rows, 1 to 3. Zero defaults to 3.
	Columns uint8 // number of columns. Zero defaults to 16.
}

// Device is a handle to a ST7036 character LCD.
type Device struct {
	bus   drivers.SPI
	cs    machine.Pin
	rs    machine.Pin
	reset machine.Pin

	rows       uint8
	columns    uint8
	rowOffsets []uint8

	instructionSetTemplate uint8
	doubleHeight           uint8

	enabled       bool
	cursorEnabled bool
	cursorBlink   bool
}

// New returns a new st7036 driver. Pass in a fully configured SPI bus.
//
// rs is the register-select pin (low selects command mode, high selects
// data mode). reset is the display reset pin, pass machine.NoPin if the
// reset line is not connected.
func New(bus drivers.SPI, cs, rs, reset machine.Pin) Device {
	cs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	cs.High()
	rs.Configure(machine.PinConfig{Mode: machine.PinOutput})
	if reset != machine.NoPin {
		reset.Configure(machine.PinConfig{Mode: machine.PinOutput})
		reset.High()
	}
	return Device{
		bus:                    bus,
		cs:                     cs,
		rs:                     rs,
		reset:                  reset,
		instructionSetTemplate: defaultInstructionSetTemplate,
	}
}

// Configure resets the display and puts it into a known state: 2 line bias,
// a contrast of 40 and a cleared screen.
func (d *Device) Configure(cfg Config) error {
	d.rows = cfg.Rows
	if d.rows == 0 {
		d.rows = 3
	}
	d.columns = cfg.Columns
	if d.columns == 0 {
		d.columns = 16
	}
	switch d.rows {
	case 1:
		d.rowOffsets = []uint8{0x00}
	case 2:
		d.rowOffsets = []uint8{0x00, 0x40}
	case 3:
		d.rowOffsets = []uint8{0x00, 0x10, 0x20}
	default:
		return errInvalidPosition
	}

	d.Reset()

	d.enabled = true
	d.updateDisplayMode()

	// Entry mode: no shift, cursor moves right after each character.
	d.writeCommand(entryMove|entryShift, 0)

	d.SetBias(1)

	if err := d.SetContrast(40); err != nil {
		return err
	}

	return d.Clear()
}

// Reset pulses the reset line, if connected. It is a no-op otherwise.
func (d *Device) Reset() {
	if d.reset == machine.NoPin {
		return
	}
	d.reset.Low()
	time.Sleep(time.Millisecond)
	d.reset.High()
	time.Sleep(time.Millisecond)
}

// Size returns the number of columns and rows on the display.
func (d *Device) Size() (columns, rows uint8) {
	return d.columns, d.rows
}

// Clear clears the display and homes the cursor.
func (d *Device) Clear() error {
	d.writeCommand(commandClear, 0)
	time.Sleep(1500 * time.Microsecond)
	return d.Home()
}

// Home moves the cursor to column 0, row 0.
func (d *Device) Home() error {
	return d.SetCursorPosition(0, 0)
}

// SetContrast sets the display contrast, in the range 0 to 0x3F.
func (d *Device) SetContrast(contrast uint8) error {
	if contrast > 0x3F {
		return errInvalidContrast
	}

	// The booster must be on for 3.3v operation. It shares a command with
	// the top two bits of contrast.
	d.writeCommand(0b01010100|((contrast>>4)&0x03), 1)
	d.writeCommand(0b01101011, 1)

	// Bottom nibble of contrast.
	d.writeCommand(0b01110000|(contrast&0x0F), 1)
	return nil
}

// SetDisplayMode enables or disables the display, cursor and cursor blink.
func (d *Device) SetDisplayMode(enable, cursor, blink bool) {
	d.enabled = enable
	d.cursorEnabled = cursor
	d.cursorBlink = blink
	d.updateDisplayMode()
}

func (d *Device) updateDisplayMode() {
	mask := uint8(commandSetDisplayMode)
	if d.enabled {
		mask |= displayOn
	}
	if d.cursorEnabled {
		mask |= cursorOn
	}
	if d.cursorBlink {
		mask |= blinkOn
	}
	d.writeCommand(mask, 0)
}

// SetBias sets the internal LCD bias, 1 selects 1/5 bias for a 3 line
// display.
func (d *Device) SetBias(bias uint8) {
	d.writeCommand(commandBias|(bias<<4)|1, 1)
}

// SetCursorOffset sets the cursor position directly, as a DDRAM address.
func (d *Device) SetCursorOffset(offset uint8) {
	d.writeCommand(setDDRAM|offset, 0)
}

// SetCursorPosition moves the cursor to the given column and row.
func (d *Device) SetCursorPosition(column, row uint8) error {
	if row >= d.rows || column >= d.columns {
		return errInvalidPosition
	}
	d.SetCursorOffset(d.rowOffsets[row] + column)
	time.Sleep(1500 * time.Microsecond) // allow at least 1.08ms to execute
	return nil
}

// CreateChar defines one of the 8 custom character slots (0 to 7). charMap
// holds 8 bytes, one per pixel row, with the lower 5 bits of each byte
// setting the pixel state.
func (d *Device) CreateChar(slot uint8, charMap [8]byte) error {
	if slot > 7 {
		return errInvalidCharSlot
	}
	base := slot * 8
	for i := uint8(0); i < 8; i++ {
		d.writeCommand(setCGRAM|(base+i), 0)
		d.writeChar(charMap[i])
	}
	return d.Home()
}

// Write sends a string to the display at the current cursor position. It
// implements io.Writer, so custom characters can be written with
// chr(0) to chr(7).
func (d *Device) Write(data []byte) (int, error) {
	d.rs.High()
	for _, b := range data {
		d.transfer(b)
		time.Sleep(50 * time.Microsecond)
	}
	return len(data), nil
}

func (d *Device) writeChar(value byte) {
	d.rs.High()
	d.transfer(value)
	time.Sleep(100 * time.Microsecond)
}

// writeCommand selects the given instruction set, then sends value as a
// command byte.
func (d *Device) writeCommand(value, instructionSet uint8) {
	d.rs.Low()
	d.writeInstructionSet(instructionSet)
	d.transfer(value)
	time.Sleep(60 * time.Microsecond)
}

func (d *Device) writeInstructionSet(instructionSet uint8) {
	d.rs.Low()
	d.transfer(d.instructionSetTemplate | instructionSet | (d.doubleHeight << 2))
	time.Sleep(60 * time.Microsecond)
}

// transfer sends a single byte, bracketed by its own chip-select pulse, to
// match the framing the ST7036 expects on this 3-wire link.
func (d *Device) transfer(b byte) {
	d.cs.Low()
	d.bus.Transfer(b)
	d.cs.High()
}
