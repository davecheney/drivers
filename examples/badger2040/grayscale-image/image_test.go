package main

import (
	"bytes"
	"errors"
	"image/color"
	"testing"
)

type imageDisplay struct {
	t         *testing.T
	pixels    [296 * 128]color.RGBA
	writes    int
	refreshes int
	err       error
}

func (d *imageDisplay) Size() (int16, int16) { return 296, 128 }

func (d *imageDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.t.Helper()
	if x < 0 || x >= 296 || y < 0 || y >= 128 {
		d.t.Fatalf("pixel outside Badger landscape display: %d,%d", x, y)
	}
	d.pixels[int(y)*296+int(x)] = c
	d.writes++
}

func (d *imageDisplay) Display() error {
	d.refreshes++
	return d.err
}

func TestTainigoImage(t *testing.T) {
	if len(testImage) != 9476 || !bytes.Equal(testImage[:4], []byte{0x28, 0x01, 0x80, 0x00}) {
		t.Fatal("expected 296x128 packed four-shade image with a four-byte header")
	}
	d := &imageDisplay{t: t}
	if err := showImage(d); err != nil {
		t.Fatal(err)
	}
	if d.writes != 37888 || d.refreshes != 1 {
		t.Fatalf("writes = %d, refreshes = %d", d.writes, d.refreshes)
	}
	var shades [4]int
	for y := 0; y < 128; y++ {
		for x := 0; x < 296; x++ {
			c := d.pixels[y*296+x]
			if c.R != c.G || c.R != c.B || c.A != 255 || c.R%85 != 0 {
				t.Fatalf("invalid shade at %d,%d: %v", x, y, c)
			}
			shades[c.R/85]++
			if (x < 31 || x >= 264) && c.R != 255 {
				t.Fatalf("aspect-ratio padding is not white at %d,%d", x, y)
			}
		}
	}
	for shade, count := range shades {
		if count == 0 {
			t.Fatalf("shade %d is missing", shade)
		}
	}
}

func TestTainigoDisplayError(t *testing.T) {
	d := &imageDisplay{t: t, err: errors.New("SPI failure")}
	if err := showImage(d); !errors.Is(err, d.err) {
		t.Fatalf("display error = %v, want %v", err, d.err)
	}
}
