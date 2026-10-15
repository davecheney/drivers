package main

import (
	"encoding/binary"
	"errors"
	"image/color"
	"io/fs"
	"path"

	"tinygo.org/x/drivers"
)

var (
	errAssetHeader     = errors.New("asset header is missing")
	errAssetDimensions = errors.New("asset dimensions do not match display")
	errAssetLength     = errors.New("asset pixel data length does not match dimensions")
)

func assetNames(assets fs.FS) ([]string, error) {
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

func showAsset(display drivers.Displayer, data []byte) error {
	if len(data) < 4 {
		return errAssetHeader
	}
	w := int(binary.LittleEndian.Uint16(data[:2]))
	h := int(binary.LittleEndian.Uint16(data[2:4]))
	displayWidth, displayHeight := display.Size()
	if w == 0 || h == 0 || w != int(displayWidth) || h != int(displayHeight) {
		return errAssetDimensions
	}
	pixels := data[4:]
	if len(pixels) != (w*h+3)/4 {
		return errAssetLength
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
