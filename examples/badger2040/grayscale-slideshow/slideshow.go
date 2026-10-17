package main

import (
	"embed"
	"errors"
	"io/fs"

	"tinygo.org/x/drivers/internal/grayimage"
)

//go:embed assets/*
var assets embed.FS

func slideshowNames(assets fs.FS) ([]string, error) {
	names, err := grayimage.Names(assets)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, errors.New("no 2bpp images embedded; add .bin files to assets")
	}
	return names, nil
}
