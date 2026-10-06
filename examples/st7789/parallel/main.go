//go:build rp2040 || rp2350

package main

import (
	"image/color"
	"machine"
	"math"
	"time"

	pio "github.com/tinygo-org/pio/rp2-pio"
	"github.com/tinygo-org/pio/rp2-pio/piolib"
	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/st7789"
)

const (
	csPin  = machine.GPIO10 // LCD_CS
	dcPin  = machine.GPIO11 // LCD_DC
	wrPin  = machine.GPIO12 // LCD_WR
	rdPin  = machine.GPIO13 // LCD_RD
	db0Pin = machine.GPIO14 // LCD_DB0
	blPin  = machine.GPIO2  // LCD_BACKLIGHT
)

// busBaud controls the PIO parallel bus clock rate driving WR.
const busBaud = 15_000_000

const displayW, displayH = 320, 240

var framebuffer [displayW * displayH * 2]byte

func main() {
	// Configure control pins to safe idle levels BEFORE bringing up the PIO parallel bus.
	csPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	csPin.High()
	dcPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	dcPin.High()
	rdPin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	rdPin.High()

	sm, err := pio.PIO0.ClaimStateMachine()
	if err != nil {
		panic(err.Error())
	}

	// Drive the 8-bit parallel bus from PIO, clocking data out on WR.
	p8tx, err := piolib.NewParallel(sm, piolib.ParallelConfig{
		Baud:        busBaud,
		Clock:       wrPin,
		DataBase:    db0Pin,
		BusWidth:    8,
		BitsPerPull: 8,
	})
	if err != nil {
		panic(err.Error())
	}

	display := st7789.NewParallel(p8tx, csPin, dcPin, rdPin, blPin)

	if err := p8tx.EnableDMA(true); err != nil {
		panic(err.Error())
	}

	display.Configure(st7789.ParallelConfig{
		Width:       displayW,
		Height:      displayH,
		Rotation:    drivers.Rotation0,
		Framebuffer: framebuffer[:],
	})

	const demoDuration = 5 * time.Second
	rotations := []drivers.Rotation{
		drivers.Rotation0,
		drivers.Rotation90,
		drivers.Rotation180,
		drivers.Rotation270,
	}

	for {
		for _, r := range rotations {
			display.SetRotation(r)
			runBouncingRects(&display, demoDuration)
			runPlasma(&display, demoDuration)
		}
	}
}

func fillFB(c565 uint16, fb []byte) {
	hi := uint8(c565 >> 8)
	lo := uint8(c565)
	for i := 0; i < len(fb); i += 2 {
		fb[i] = hi
		fb[i+1] = lo
	}
}

func runBouncingRects(display *st7789.ParallelDevice, d time.Duration) {
	w, h := display.Size()
	type rect struct {
		x, y, dx, dy, w, h int16
		c                  color.RGBA
	}
	rects := []rect{
		{20, 20, 3, 2, 40, 30, color.RGBA{255, 64, 64, 255}},
		{90, 60, -2, 3, 50, 20, color.RGBA{64, 255, 64, 255}},
		{160, 120, 4, -3, 30, 60, color.RGBA{64, 128, 255, 255}},
		{200, 40, -3, -2, 60, 40, color.RGBA{255, 255, 64, 255}},
	}
	bg := st7789.RGBATo565(color.RGBA{0, 0, 0, 255})
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		fillFB(bg, framebuffer[:])
		for i := range rects {
			r := &rects[i]
			r.x += r.dx
			r.y += r.dy
			if r.x < 0 {
				r.x = 0
				r.dx = -r.dx
			}
			if r.y < 0 {
				r.y = 0
				r.dy = -r.dy
			}
			if r.x+r.w >= w {
				r.x = w - r.w - 1
				r.dx = -r.dx
			}
			if r.y+r.h >= h {
				r.y = h - r.h - 1
				r.dy = -r.dy
			}
			for py := r.y; py < r.y+r.h; py++ {
				for px := r.x; px < r.x+r.w; px++ {
					display.SetPixel(px, py, r.c)
				}
			}
		}
		if err := display.Display(); err != nil {
			return
		}
	}
}

var sinTable [256]int16

func init() {
	for i := 0; i < 256; i++ {
		sinTable[i] = int16(math.Round(math.Sin(2*math.Pi*float64(i)/256) * 127))
	}
}

func isin(a int) int16 {
	return sinTable[uint8(a)]
}

func runPlasma(display *st7789.ParallelDevice, d time.Duration) {
	w, h := display.Size()
	deadline := time.Now().Add(d)
	t := 0
	for time.Now().Before(deadline) {
		for py := int16(0); py < h; py++ {
			for px := int16(0); px < w; px++ {
				v := int(isin(int(px)*4+t)) +
					int(isin(int(py)*5-t)) +
					int(isin(int(px+py)*3+t)) +
					int(isin(int(px-py)*2-t))
				u := uint8(((v + 512) >> 2) & 0xff)
				r := uint8(isin(int(u))) + 128
				g := uint8(isin(int(u)+85)) + 128
				b := uint8(isin(int(u)+170)) + 128
				display.SetPixel(px, py, color.RGBA{r, g, b, 255})
			}
		}
		if err := display.Display(); err != nil {
			return
		}
		t += 3
	}
}
