// Demo for the Pimoroni Display-o-Tron HAT on a Waveshare RP2040-PiZero.
//
// Board wiring.
//
// The pin numbers below are only correct for the RP2040-PiZero. Any other
// board needs its own mapping. See the epd2in13-pizero demo in this
// examples tree for how that mapping was measured.
//
// The RP2040-PiZero has a Raspberry Pi 40 pin header. Waveshare exchanges
// GPIO10 and GPIO11 on the header so that RP2040 SPI1 SCK and TX align with
// the Raspberry Pi SCLK and MOSI positions. Every other header pin passes
// straight through to the RP2040 GPIO of the same number.
// Schematic: https://files.waveshare.com/wiki/RP2040-PiZero/RP2040-PiZero.pdf
//
// Header to RP2040 GPIO for the Display-o-Tron HAT signals:
//
//	pin 3  SDA          GPIO2  (I2C1 SDA)
//	pin 5  SCL          GPIO3  (I2C1 SCL)
//	pin 19 LCD MOSI     GPIO11 (SPI1 SDO)
//	pin 22 LCD RS       GPIO25
//	pin 23 LCD SCLK     GPIO10 (SPI1 SCK)
//	pin 24 LCD CS       GPIO8
//	pin 32 LCD RESET    GPIO12
//
// The LCD is write only, so SPI1 SDI is left unconfigured (NoPin) and
// GPIO12, which doubles as SPI1's native SDI pin, is safe to drive as a
// plain reset output.
//
// The SN3218 backlight driver and CAP1166 touch and bar graph controller
// both sit on I2C1, at their fixed addresses 0x54 and 0x2C.
//
// Build and flash:
//
//	tinygo flash -target=pico ./examples/dothat-pizero
package main

import (
	"machine"
	"time"

	"tinygo.org/x/drivers/cap1166"
	"tinygo.org/x/drivers/sn3218"
	"tinygo.org/x/drivers/st7036"
)

var (
	lcd   st7036.Device
	light sn3218.Device
	touch cap1166.Device
)

func main() {
	// Wait for the USB serial console to attach so the log is not lost.
	time.Sleep(3 * time.Second)

	err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 1000000,
		SCK:       machine.GPIO10,
		SDO:       machine.GPIO11,
		SDI:       machine.NoPin,
		Mode:      0,
	})
	if err != nil {
		println("SPI1 configure failed:", err.Error())
		return
	}

	lcd = st7036.New(machine.SPI1, machine.GPIO8, machine.GPIO25, machine.GPIO12)
	if err := lcd.Configure(st7036.Config{Rows: 3, Columns: 16}); err != nil {
		println("LCD configure failed:", err.Error())
		return
	}

	err = machine.I2C1.Configure(machine.I2CConfig{
		Frequency: 400000,
		SDA:       machine.GPIO2,
		SCL:       machine.GPIO3,
	})
	if err != nil {
		println("I2C1 configure failed:", err.Error())
		return
	}

	light = sn3218.New(machine.I2C1)
	if err := light.Configure(); err != nil {
		println("backlight configure failed:", err.Error())
		return
	}

	touch = cap1166.New(machine.I2C1, cap1166.DefaultAddress)
	if err := touch.Configure(); err != nil {
		println("touch configure failed:", err.Error())
		return
	}

	println("Display-o-Tron HAT bring-up")
	lcd.SetCursorPosition(0, 0)
	lcd.Write([]byte("Display-o-Tron"))
	lcd.SetCursorPosition(0, 1)
	lcd.Write([]byte("touch a button"))

	lastButton := ""
	var hue float32

	for {
		// Sweep the 6 backlight zones through the colour wheel.
		hue += 0.01
		if hue >= 1 {
			hue -= 1
		}
		setBacklightSweep(hue)

		// Poll the 6 touch buttons and report the first one pressed.
		status, err := touch.InputStatus()
		if err == nil && status != 0 {
			name := buttonName(status)
			if name != lastButton {
				lastButton = name
				println("pressed:", name)
				lcd.SetCursorPosition(0, 2)
				lcd.Write([]byte(name + "           "))
			}
			touch.ClearInterrupt()
		} else {
			lastButton = ""
		}

		// Use the bar graph to show backlight hue as a fraction of the
		// colour wheel, so it pulses in time with the backlight sweep.
		touch.SetGraph(hue)

		time.Sleep(20 * time.Millisecond)
	}
}

// buttonName returns the label of the lowest numbered touched input in
// status, matching the silk-screened button names on the HAT.
func buttonName(status uint8) string {
	switch {
	case status&(1<<cap1166.Cancel) != 0:
		return "cancel"
	case status&(1<<cap1166.Up) != 0:
		return "up"
	case status&(1<<cap1166.Down) != 0:
		return "down"
	case status&(1<<cap1166.Left) != 0:
		return "left"
	case status&(1<<cap1166.Button) != 0:
		return "select"
	case status&(1<<cap1166.Right) != 0:
		return "right"
	default:
		return ""
	}
}

// setBacklightSweep lights the 6 backlight zones with a gradient of hues
// centred on hue.
func setBacklightSweep(hue float32) {
	const sweepRange = 0.0833

	var values [sn3218.NumChannels]uint8
	for zone := 0; zone < 6; zone++ {
		h := hue + sweepRange*float32(zone)
		for h >= 1 {
			h -= 1
		}
		r, g, b := hueToRGB(h)
		// Channel order per zone is B, G, R.
		values[zone*3+0] = b
		values[zone*3+1] = g
		values[zone*3+2] = r
	}
	light.SetChannels(values)
}

// hueToRGB converts a hue in the range 0.0 to 1.0 to full brightness,
// fully saturated RGB values.
func hueToRGB(hue float32) (r, g, b uint8) {
	h6 := hue * 6
	i := int(h6)
	f := h6 - float32(i)
	q := 1 - f
	t := f

	var rf, gf, bf float32
	switch i % 6 {
	case 0:
		rf, gf, bf = 1, t, 0
	case 1:
		rf, gf, bf = q, 1, 0
	case 2:
		rf, gf, bf = 0, 1, t
	case 3:
		rf, gf, bf = 0, q, 1
	case 4:
		rf, gf, bf = t, 0, 1
	default:
		rf, gf, bf = 1, 0, q
	}
	return uint8(rf * 255), uint8(gf * 255), uint8(bf * 255)
}
