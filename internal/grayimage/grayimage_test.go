package grayimage

import (
	"encoding/binary"
	"errors"
	"image/color"
	"testing"

	"tinygo.org/x/drivers"
)

type testDisplay struct {
	w, h      int16
	pixels    []color.RGBA
	refreshes int
	err       error
}

func (d *testDisplay) Size() (int16, int16) { return d.w, d.h }
func (d *testDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.pixels = append(d.pixels, c)
}
func (d *testDisplay) Display() error {
	d.refreshes++
	return d.err
}

func packedImage(w, h uint16, pixels ...byte) []byte {
	data := make([]byte, 4, 4+len(pixels))
	binary.LittleEndian.PutUint16(data[:2], w)
	binary.LittleEndian.PutUint16(data[2:4], h)
	return append(data, pixels...)
}

func TestPackedFormats(t *testing.T) {
	for _, tc := range []struct {
		name string
		show func(drivers.Displayer, []byte) error
		data []byte
		want []byte
	}{
		{"2bpp", Display, packedImage(5, 1, 0x1b, 0x40), []byte{0, 85, 170, 255, 85}},
		{"4bpp", Display16, packedImage(17, 1, 0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef, 0x50),
			[]byte{0, 17, 34, 51, 68, 85, 102, 119, 136, 153, 170, 187, 204, 221, 238, 255, 85}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &testDisplay{w: int16(len(tc.want)), h: 1}
			if err := tc.show(d, tc.data); err != nil {
				t.Fatal(err)
			}
			if len(d.pixels) != len(tc.want) || d.refreshes != 1 {
				t.Fatal("incorrect pixel or refresh count")
			}
			for i, gray := range tc.want {
				if d.pixels[i] != (color.RGBA{gray, gray, gray, 255}) {
					t.Fatalf("pixel %d = %v, want gray %d", i, d.pixels[i], gray)
				}
			}
			failure := errors.New("SPI failure")
			d.err = failure
			if err := tc.show(d, tc.data); !errors.Is(err, failure) {
				t.Fatalf("display error = %v", err)
			}
		})
	}
}

func TestInvalidImagesDoNotDraw(t *testing.T) {
	for _, show := range []func(drivers.Displayer, []byte) error{Display, Display16} {
		for _, tc := range []struct {
			data []byte
			err  error
		}{
			{nil, ErrHeader},
			{[]byte{1, 2, 3}, ErrHeader},
			{packedImage(0, 1), ErrDimensions},
			{packedImage(4, 2, 0, 0, 0, 0), ErrDimensions},
			{packedImage(5, 1, 0, 0, 0), ErrDimensions},
			{packedImage(4, 1), ErrLength},
			{packedImage(4, 1, 0, 0, 0), ErrLength},
		} {
			d := &testDisplay{w: 4, h: 1}
			if err := show(d, tc.data); !errors.Is(err, tc.err) {
				t.Fatalf("image %x: error = %v, want %v", tc.data, err, tc.err)
			}
			if len(d.pixels) != 0 || d.refreshes != 0 {
				t.Fatal("invalid image modified or refreshed the display")
			}
		}
	}
}
