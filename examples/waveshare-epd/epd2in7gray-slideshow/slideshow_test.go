package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image/color"
	"io/fs"
	"os"
	"slices"
	"testing"
	"testing/fstest"
)

type fakeDisplay struct {
	width, height int16
	pixels        []color.RGBA
	writes        int
	refreshes     int
	err           error
}

func (d *fakeDisplay) Size() (int16, int16) {
	return d.width, d.height
}

func (d *fakeDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.pixels[int(y)*int(d.width)+int(x)] = c
	d.writes++
}

func (d *fakeDisplay) Display() error {
	d.refreshes++
	return d.err
}

func newDisplay(width, height int16) *fakeDisplay {
	return &fakeDisplay{
		width: width, height: height,
		pixels: make([]color.RGBA, int(width)*int(height)),
	}
}

func TestAssetNames(t *testing.T) {
	files := fstest.MapFS{
		"assets/01-second.bin":      {Data: nil},
		"assets/00-first.bin":       {Data: nil},
		"assets/README.md":          {Data: nil},
		"assets/photo.png":          {Data: nil},
		"assets/ignored.bin/child":  {Data: nil},
		"assets/subdir/another.bin": {Data: nil},
	}
	got, err := assetNames(files)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"assets/00-first.bin", "assets/01-second.bin"}
	if !slices.Equal(got, want) {
		t.Fatalf("asset names = %v, want %v", got, want)
	}
}

func TestAssetNamesWithoutImages(t *testing.T) {
	names, err := assetNames(fstest.MapFS{"assets/README.md": {Data: nil}})
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 0 {
		t.Fatalf("non-image files were included: %v", names)
	}
}

func TestAssetNamesReadError(t *testing.T) {
	if _, err := assetNames(fstest.MapFS{}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error = %v, want missing directory", err)
	}
}

func TestShowAsset(t *testing.T) {
	display := newDisplay(8, 2)
	data := []byte{8, 0, 2, 0, 0x1b, 0x1b, 0xe4, 0xe4}
	if err := showAsset(display, data); err != nil {
		t.Fatal(err)
	}
	want := []uint8{0, 85, 170, 255, 0, 85, 170, 255, 255, 170, 85, 0, 255, 170, 85, 0}
	for i, gray := range want {
		if got := display.pixels[i]; got != (color.RGBA{gray, gray, gray, 255}) {
			t.Fatalf("pixel %d = %v, want gray %d", i, got, gray)
		}
	}
	if display.writes != 16 || display.refreshes != 1 {
		t.Fatalf("pixel writes = %d, refreshes = %d", display.writes, display.refreshes)
	}
}

func TestShowAssetNativeLandscape(t *testing.T) {
	display := newDisplay(264, 176)
	data := make([]byte, 4+11616)
	binary.LittleEndian.PutUint16(data[:2], 264)
	binary.LittleEndian.PutUint16(data[2:4], 176)
	copy(data[4:], bytes.Repeat([]byte{0x1b}, 11616))
	if err := showAsset(display, data); err != nil {
		t.Fatal(err)
	}
	for i, got := range display.pixels {
		gray := uint8(i%4) * 85
		if got != (color.RGBA{gray, gray, gray, 255}) {
			t.Fatalf("pixel %d = %v, want gray %d", i, got, gray)
		}
	}
	if display.writes != 46464 || display.refreshes != 1 {
		t.Fatalf("pixel writes = %d, refreshes = %d", display.writes, display.refreshes)
	}
}

func TestShowAssetRowCrossesByteBoundary(t *testing.T) {
	display := newDisplay(3, 2)
	if err := showAsset(display, []byte{3, 0, 2, 0, 0x1b, 0x10}); err != nil {
		t.Fatal(err)
	}
	for i, gray := range []uint8{0, 85, 170, 255, 0, 85} {
		if got := display.pixels[i]; got != (color.RGBA{gray, gray, gray, 255}) {
			t.Fatalf("pixel %d = %v, want gray %d", i, got, gray)
		}
	}
}

func TestShowAssetRejectsInvalidData(t *testing.T) {
	valid := []byte{8, 0, 2, 0, 0x1b, 0x1b, 0xe4, 0xe4}
	for _, tc := range []struct {
		name string
		data []byte
		err  error
	}{
		{"empty", nil, errAssetHeader},
		{"short header", valid[:3], errAssetHeader},
		{"zero width", []byte{0, 0, 2, 0}, errAssetDimensions},
		{"zero height", []byte{8, 0, 0, 0}, errAssetDimensions},
		{"wrong width", []byte{9, 0, 2, 0}, errAssetDimensions},
		{"wrong height", []byte{8, 0, 3, 0}, errAssetDimensions},
		{"large width", []byte{0xff, 0xff, 2, 0}, errAssetDimensions},
		{"no pixels", valid[:4], errAssetLength},
		{"truncated pixels", valid[:7], errAssetLength},
		{"extra pixels", append(bytes.Clone(valid), 0), errAssetLength},
	} {
		t.Run(tc.name, func(t *testing.T) {
			display := newDisplay(8, 2)
			if err := showAsset(display, tc.data); !errors.Is(err, tc.err) {
				t.Fatalf("error = %v, want %v", err, tc.err)
			}
			if display.writes != 0 || display.refreshes != 0 {
				t.Fatal("invalid asset modified the framebuffer or refreshed the display")
			}
		})
	}
}

func TestShowAssetDisplayError(t *testing.T) {
	display := newDisplay(4, 1)
	display.err = errors.New("SPI failure")
	if err := showAsset(display, []byte{4, 0, 1, 0, 0x1b}); !errors.Is(err, display.err) {
		t.Fatalf("error = %v, want display error", err)
	}
}

func TestSlideshowAssets(t *testing.T) {
	files := os.DirFS(".")
	names, err := assetNames(files)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("no slideshow images found")
	}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			data, err := fs.ReadFile(files, name)
			if err != nil {
				t.Fatal(err)
			}
			display := newDisplay(264, 176)
			if err := showAsset(display, data); err != nil {
				t.Fatal(err)
			}
			if display.writes != 46464 || display.refreshes != 1 {
				t.Fatalf("pixel writes = %d, refreshes = %d", display.writes, display.refreshes)
			}
		})
	}
}
