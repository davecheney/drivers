// Package grayimage loads packed four-shade images for e-paper examples.
package grayimage

import (
	"encoding/binary"
	"errors"
	"image/color"
	"io/fs"
	"path"

	"tinygo.org/x/drivers"
)

var (
	ErrHeader     = errors.New("asset header is missing")
	ErrDimensions = errors.New("asset dimensions do not match display")
	ErrLength     = errors.New("asset pixel data length does not match dimensions")
)

// Names returns .bin files in the assets directory in filename order.
func Names(assets fs.FS) ([]string, error) {
	entries, err := fs.ReadDir(assets, "assets")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && path.Ext(entry.Name()) == ".bin" {
			names = append(names, "assets/"+entry.Name())
		}
	}
	return names, nil
}

// Display validates an image before drawing or refreshing the display.
// The header is little-endian uint16 width/height, followed by MSB-first 2bpp.
func Display(display drivers.Displayer, data []byte) error {
	if len(data) < 4 {
		return ErrHeader
	}
	w := int(binary.LittleEndian.Uint16(data[:2]))
	h := int(binary.LittleEndian.Uint16(data[2:4]))
	displayWidth, displayHeight := display.Size()
	if w == 0 || h == 0 || w != int(displayWidth) || h != int(displayHeight) {
		return ErrDimensions
	}
	pixels := data[4:]
	if len(pixels) != (w*h+3)/4 {
		return ErrLength
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			index := y*w + x
			shift := uint(6 - (index%4)*2)
			gray := ((pixels[index/4] >> shift) & 3) * 85
			display.SetPixel(int16(x), int16(y), color.RGBA{gray, gray, gray, 255})
		}
	}
	return display.Display()
}
