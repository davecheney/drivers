package main

import (
	"io/fs"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/grayimage"
)

var (
	errAssetHeader     = grayimage.ErrHeader
	errAssetDimensions = grayimage.ErrDimensions
	errAssetLength     = grayimage.ErrLength
)

func assetNames(assets fs.FS) ([]string, error) {
	return grayimage.Names(assets)
}

func showAsset(display drivers.Displayer, data []byte) error {
	return grayimage.Display(display, data)
}
