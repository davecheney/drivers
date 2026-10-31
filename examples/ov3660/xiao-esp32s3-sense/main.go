// This example shows a live preview from the OV3660 camera on a Seeed XIAO
// ESP32S3 Sense on a 240x240 ST7789 display. The shutter button saves the
// frame as a BMP file on the microSD card of the Sense board.
//
// ST7789 wiring: SCK D8, MOSI D10, RST D0, DC D1, CS D2, BL D3.
// Shutter button from D7 to GND.
package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ov3660"
	"tinygo.org/x/drivers/st7789"
)

const (
	width  = 240
	height = 240

	displayHz = 40_000_000
)

var (
	frame  [width * height * 2]byte
	rowBuf [16 * height * 2]byte
)

// The display and the microSD card share SPI0. The card CS is GPIO21.
func setSPI(hz uint32) {
	machine.SPI0.Configure(machine.SPIConfig{
		Frequency: hz,
		SCK:       machine.D8,
		SDO:       machine.D10,
		SDI:       machine.D9,
	})
}

func main() {
	setSPI(displayHz)
	display := st7789.New(machine.SPI0, machine.D0, machine.D1, machine.D2, machine.D3)
	display.Configure(st7789.Config{
		Width:     width,
		Height:    height,
		Rotation:  drivers.Rotation270,
		RowOffset: 80,
	})

	store := mount()

	// Camera pins from the Seeed XIAO ESP32S3 Sense pinout.
	pins := camPins{
		xclk:  machine.GPIO10,
		pclk:  machine.GPIO13,
		vsync: machine.GPIO38,
		href:  machine.GPIO47,
		data: [8]machine.Pin{
			machine.GPIO15, machine.GPIO17, machine.GPIO18, machine.GPIO16,
			machine.GPIO14, machine.GPIO12, machine.GPIO11, machine.GPIO48,
		},
	}
	startXCLK(pins.xclk, 16_000_000)
	time.Sleep(10 * time.Millisecond)

	machine.I2C1.Configure(machine.I2CConfig{
		SDA:       machine.GPIO40,
		SCL:       machine.GPIO39,
		Frequency: 100_000,
	})
	sensor := ov3660.New(machine.I2C1)
	for {
		err := sensor.Configure(ov3660.Config{Width: width, Height: height, Format: ov3660.RGB565})
		if err == nil {
			break
		}
		println("ov3660:", err.Error())
		time.Sleep(time.Second)
	}

	shutter := machine.D7
	shutter.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	cam := newCamera(pins, frame[:], 3840)
	wasPressed := false
	for {
		pressed := !shutter.Get()
		if pressed && !wasPressed {
			if store == nil {
				store = mount()
			}
			if store != nil {
				save(store)
			}
		}
		wasPressed = pressed

		n, ok := cam.capture(500 * time.Millisecond)
		if !ok || n != len(frame) {
			println("capture failed, bytes:", n)
			continue
		}
		display.DrawRGBBitmap8(0, 0, frame[:], width, height)
	}
}

func mount() *storage {
	s, err := mountSD(machine.SPI0, machine.GPIO21)
	setSPI(displayHz)
	if err != nil {
		println("sd:", err.Error())
		return nil
	}
	return s
}

// save writes the frame on the display to the next free file.
func save(s *storage) {
	start := time.Now()
	setSPI(s.hz)
	defer setSPI(displayHz)

	name, err := s.create()
	if err != nil {
		println("sd create:", err.Error())
		return
	}
	err = writeBMP(&s.file, frame[:], width, height, rowBuf[:])
	if cerr := s.file.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		println("sd write:", name, err.Error())
		return
	}
	println("saved", name, "in", time.Since(start).Milliseconds(), "ms")
}
