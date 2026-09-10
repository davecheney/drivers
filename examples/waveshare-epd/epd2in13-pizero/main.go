// Demo for the Waveshare 2.13in e-Paper HAT V1 on a Waveshare RP2040-PiZero.
//
// Panel version.
//
// This demo is only for the V1 panel, which uses the IL3820 controller. It
// uses the epd2in13 driver, which sends the IL3820 init sequence and a 30
// byte LUT. Later panels use the SSD1680 controller, which needs a different
// init sequence and a 153 byte LUT, so this demo does not drive them.
//
// Identify the panel from the date code on the ribbon cable. V1 shipped from
// 2015, V2 from 2019 and V3 and V4 from 2021.
//
//	V1 (IL3820,  122x250)  works, this demo
//	V2 (SSD1680, 122x250)  not supported here, try the epd2in9v2 driver
//	V3 (SSD1680, 122x250)  not supported here, try the epd2in9v2 driver
//	V4 (SSD1680, 122x250)  not supported here, try the epd2in9v2 driver
//	2.13in B, C and D variants (three colour or flexible) not supported here,
//	                       use epd2in13x for the B and C variants
//
// Board support.
//
// The pin numbers below are only correct for the RP2040-PiZero. Any other
// board needs its own mapping.
//
//	Waveshare RP2040-PiZero          tested, works
//	Raspberry Pi Pico and Pico W     untested, needs its own pin mapping
//	                                 because it has no 40 pin header
//	Other RP2040 boards              untested, needs its own pin mapping
//	Raspberry Pi 40 pin SBC headers  not applicable, this is TinyGo firmware
//	                                 for a microcontroller, not Linux
//
// The RP2040-PiZero has a Raspberry Pi 40 pin header. Waveshare exchanges
// GPIO10 and GPIO11 on the header so that RP2040 SPI1 SCK and TX align with
// the Raspberry Pi SCLK and MOSI positions.
// Schematic: https://files.waveshare.com/wiki/RP2040-PiZero/RP2040-PiZero.pdf
//
// Header to RP2040 GPIO for the e-Paper HAT signals:
//
//	pin 11 RST   GPIO17
//	pin 18 BUSY  GPIO24
//	pin 19 MOSI  GPIO11 (SPI1 SDO)
//	pin 22 DC    GPIO25
//	pin 23 SCLK  GPIO10 (SPI1 SCK)
//	pin 24 CS    GPIO8
//
// Build and flash:
//
//	tinygo flash -target=pico ./examples/waveshare-epd/epd2in13-pizero
package main

import (
	"image/color"
	"machine"
	"time"

	"tinygo.org/x/drivers/waveshare-epd/epd2in13"
)

const (
	epdWidth  = 122
	epdHeight = 250
)

var (
	black = color.RGBA{0, 0, 0, 255}
	white = color.RGBA{255, 255, 255, 255}

	display epd2in13.Device
)

func main() {
	// Wait for the USB serial console to attach so the log is not lost.
	time.Sleep(3 * time.Second)

	err := machine.SPI1.Configure(machine.SPIConfig{
		Frequency: 4000000,
		SCK:       machine.GPIO10,
		SDO:       machine.GPIO11,
		SDI:       machine.GPIO12,
		Mode:      0,
	})
	if err != nil {
		println("SPI1 configure failed:", err.Error())
		return
	}

	display = epd2in13.New(machine.SPI1, machine.GPIO8, machine.GPIO25, machine.GPIO17, machine.GPIO24)
	display.Configure(epd2in13.Config{
		Width:  epdWidth,
		Height: epdHeight,
	})

	println("clear the display")
	display.ClearBuffer()
	display.ClearDisplay()
	display.WaitUntilIdle()

	println("draw the test pattern")
	drawTestPattern()
	display.Display()
	display.WaitUntilIdle()

	println("done, the display holds the image without power")
	display.DeepSleep()

	for {
		time.Sleep(time.Second)
	}
}

// drawTestPattern draws a border, two diagonals and a grey scale ramp. A
// correct panel shows all three. Missing or shifted parts point to a wrong
// pin, a wrong panel version or a bad ribbon connection.
func drawTestPattern() {
	fill(0, 0, epdWidth, epdHeight, white)

	// Border, 2 pixels wide.
	fill(0, 0, epdWidth, 2, black)
	fill(0, epdHeight-2, epdWidth, 2, black)
	fill(0, 0, 2, epdHeight, black)
	fill(epdWidth-2, 0, 2, epdHeight, black)

	// Diagonals across the full panel.
	for y := int16(0); y < epdHeight; y++ {
		x := int16(int32(y) * epdWidth / epdHeight)
		setPixel(x, y, black)
		setPixel(epdWidth-1-x, y, black)
	}

	// Solid black block. This shows the darkest level the panel can reach.
	fill(16, 20, 40, 40, black)

	// Dither ramp from light to dark in eight steps.
	for step := int16(0); step < 8; step++ {
		y0 := 100 + step*16
		for y := y0; y < y0+14; y++ {
			for x := int16(10); x < epdWidth-10; x++ {
				if (x+y*3)%8 < step+1 {
					setPixel(x, y, black)
				}
			}
		}
	}
}

func fill(x, y, w, h int16, c color.RGBA) {
	for i := x; i < x+w; i++ {
		for j := y; j < y+h; j++ {
			setPixel(i, j, c)
		}
	}
}

func setPixel(x, y int16, c color.RGBA) {
	if x < 0 || x >= epdWidth || y < 0 || y >= epdHeight {
		return
	}
	display.SetPixel(x, y, c)
}
