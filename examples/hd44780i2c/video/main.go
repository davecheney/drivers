// Command lcdvideo plays a video on a 16x4 HD44780 LCD with a PCF8574 I2C
// backpack. Generate video.bin with tools/lcdvideogen.
package main

import (
	_ "embed"
	"encoding/binary"
	"machine"
	"time"

	"tinygo.org/x/drivers/hd44780i2c"
)

//go:embed video.bin
var video string

const (
	addr = 0x26

	// PCF8574 bits on the usual backpack.
	bl = 0x08
	en = 0x04
	rs = 0x01

	cgramSet = 0x40
	ddramSet = 0x80
)

var rowOffset = [4]byte{0x00, 0x40, 0x10, 0x50}

// buf holds 6 PCF8574 bytes for each LCD byte, see put.
var buf = make([]byte, 0, 6*256)

func put(v, mode byte) {
	hi := v&0xf0 | mode | bl
	lo := v<<4 | mode | bl
	buf = append(buf, hi, hi|en, hi, lo, lo|en, lo)
}

func main() {
	led := machine.LED
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})

	i2c := machine.I2C0
	i2c.Configure(machine.I2CConfig{Frequency: 100 * machine.KHz})
	lcd := hd44780i2c.New(i2c, addr)
	lcd.Configure(hd44780i2c.Config{Width: 16, Height: 4})
	i2c.Configure(machine.I2CConfig{Frequency: 400 * machine.KHz})

	if len(video) < 20 || video[:4] != "L16D" {
		for {
			led.Set(!led.Get())
			time.Sleep(100 * time.Millisecond)
		}
	}
	num := binary.LittleEndian.Uint32([]byte(video[4:8]))
	den := binary.LittleEndian.Uint32([]byte(video[8:12]))
	period := time.Second * time.Duration(den) / time.Duration(num)

	for {
		lcd.ClearDisplay()
		led.High()
		pos := 20
		next := time.Now()
		for pos < len(video) {
			buf = buf[:0]
			n := int(video[pos])
			pos++
			for ; n > 0; n-- {
				put(cgramSet|video[pos]<<3, 0)
				for i := pos + 1; i < pos+9; i++ {
					put(video[i], rs)
				}
				pos += 9
			}
			n = int(video[pos])
			pos++
			for ; n > 0; n-- {
				cell, length := video[pos], int(video[pos+1])
				put(ddramSet|(rowOffset[cell/16]+cell%16), 0)
				for i := pos + 2; i < pos+2+length; i++ {
					put(video[i], rs)
				}
				pos += 2 + length
			}
			if len(buf) > 0 {
				i2c.Tx(addr, buf, nil)
			}

			next = next.Add(period)
			if d := time.Until(next); d > 0 {
				time.Sleep(d)
			}
		}
		led.Low()
		time.Sleep(2 * time.Second)
	}
}
