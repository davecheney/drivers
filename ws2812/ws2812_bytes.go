//go:build !avr

package ws2812

import "image/color"

func (d Device) writeBytes(buf []byte) (err error) {
	for _, c := range buf {
		err = d.WriteByte(c)
	}
	return
}

func (d Device) writePixels(buf []color.RGBA, brightness uint8, alpha bool) (err error) {
	for _, color := range buf {
		r, g, b := applyBrightness(color, brightness)
		d.WriteByte(g)
		d.WriteByte(r)
		err = d.WriteByte(b)
		if alpha {
			err = d.WriteByte(color.A)
		}
	}
	return
}
