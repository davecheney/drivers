package main

import (
	"image/color"
	"testing"
)

type swatchDisplay struct {
	t      *testing.T
	pixels [296 * 128]color.RGBA
}

func (d *swatchDisplay) Size() (int16, int16) { return 296, 128 }
func (d *swatchDisplay) Display() error       { return nil }
func (d *swatchDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.t.Helper()
	if x < 0 || x >= 296 || y < 0 || y >= 128 {
		d.t.Fatalf("pixel outside Badger landscape display: %d,%d", x, y)
	}
	d.pixels[int(y)*296+int(x)] = c
}

func TestSwatches(t *testing.T) {
	d := &swatchDisplay{t: t}
	drawSwatches(d)
	for i := 0; i < 4; i++ {
		left, right := i*296/4, (i+1)*296/4
		gray := uint8(i * 85)
		want := color.RGBA{gray, gray, gray, 255}
		for y := 29; y < 101; y++ {
			for x := left + 3; x < right-3; x++ {
				if got := d.pixels[y*296+x]; got != want {
					t.Fatalf("swatch %d at %d,%d = %v, want %v", i, x, y, got, want)
				}
			}
		}
		if got := d.pixels[28*296+left+2]; got != (color.RGBA{A: 255}) {
			t.Fatalf("swatch %d is missing its black border", i)
		}
	}
}
