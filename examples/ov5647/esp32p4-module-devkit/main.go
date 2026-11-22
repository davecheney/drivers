// This example captures frames from a Raspberry Pi Camera v1.3 (OV5647) on the
// CSI port of a Waveshare ESP32-P4-Module-DEV-KIT and shows them on a 128x128
// ST7735S display. It needs a TinyGo with ESP32-P4 PSRAM support.
package main

import (
	"machine"
	"time"
	"unsafe"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/ov5647"
	"tinygo.org/x/drivers/pixel"
	"tinygo.org/x/drivers/st7735"
)

// The full sensor frame, captured into PSRAM.
const (
	width  = 800
	height = 640

	// Show colour bars instead of the camera image.
	testPattern = false
)

func main() {
	time.Sleep(time.Second)

	machine.SPI2.Configure(machine.SPIConfig{Frequency: 40 * machine.MHz, SCK: machine.GPIO0, SDO: machine.GPIO3, SDI: machine.NoPin})
	display := st7735.New(machine.SPI2, machine.GPIO20, machine.GPIO1, machine.GPIO36, machine.GPIO4)
	display.Configure(st7735.Config{Width: 128, Height: 128, Rotation: drivers.Rotation90, RowOffset: 3, ColumnOffset: 2})

	machine.I2C0.Configure(machine.I2CConfig{SDA: machine.SDA_PIN, SCL: machine.SCL_PIN, Frequency: 400 * machine.KHz})
	camera := ov5647.New(machine.I2C0)
	if err := camera.Configure(ov5647.Config{Width: width, Height: height}); err != nil {
		println("camera:", err.Error())
		return
	}
	camera.SetTestPattern(testPattern)
	camera.Start()

	if err := startCSI(ov5647.LaneRate, width, height); err != nil {
		println("csi:", err.Error())
		return
	}

	mem := machine.PSRAM()
	if len(mem) < width*height+64 {
		println("not enough PSRAM")
		return
	}
	pad := int(-uintptr(unsafe.Pointer(&mem[0])) & 63)
	frame := mem[pad : pad+width*height]

	var preview [128 * 128 * 2]byte
	image := pixel.NewImageFromBytes[pixel.RGB565BE](128, 128, preview[:])
	for n := 0; ; n++ {
		start := time.Now()
		if !capture(frame, 200*time.Millisecond) {
			println("frame", n, "timeout")
			return
		}
		took := time.Since(start)
		if n < 3 || n%50 == 0 {
			printStats(n, took, frame)
		}
		scale(frame, preview[:])
		display.DrawBitmap(0, 0, image)
	}
}

func printStats(n int, took time.Duration, frame []byte) {
	lo, hi, sum := 255, 0, 0
	for _, b := range frame {
		v := int(b)
		lo = min(lo, v)
		hi = max(hi, v)
		sum += v
	}
	println("frame", n, "ms", took.Milliseconds(), "min", lo, "max", hi, "mean", sum/len(frame))
}

// scale takes the centre square of a BGGR frame and makes one RGB565 pixel
// from one 2x2 Bayer block in each step x step area.
func scale(frame, out []byte) {
	const (
		step = height / 128
		x0   = (width - height) / 2
	)
	for y := 0; y < 128; y++ {
		sy := y * step &^ 1
		row0 := frame[sy*width:]
		row1 := frame[(sy+1)*width:]
		for x := 0; x < 128; x++ {
			sx := x0 + x*step&^1
			b := int(row0[sx])
			g := (int(row0[sx+1]) + int(row1[sx])) / 2
			r := int(row1[sx+1])
			c := uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
			i := (y*128 + x) * 2
			out[i], out[i+1] = byte(c>>8), byte(c)
		}
	}
}
