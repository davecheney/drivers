package main

import (
	"encoding/binary"
	"errors"
	"image/color"
	"io/fs"
	"reflect"
	"testing"
	"testing/fstest"

	"tinygo.org/x/drivers"
	"tinygo.org/x/drivers/internal/grayimage"
	"tinygo.org/x/drivers/uc8151"
)

type assetDisplay struct {
	pixels    []color.RGBA
	writes    int
	refreshes int
	err       error
}

func (d *assetDisplay) Size() (int16, int16) { return 296, 128 }

func (d *assetDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.pixels[int(y)*296+int(x)] = c
	d.writes++
}

func (d *assetDisplay) Display() error {
	d.refreshes++
	return d.err
}

func TestSlideshowNames(t *testing.T) {
	want := []string{
		"assets/00-macintosh-classic.bin", "assets/01-guinea-pig.bin",
		"assets/02-cccp-stamp.bin", "assets/03-kanagawa.bin",
	}
	names, err := slideshowNames(assets)
	if err != nil || !reflect.DeepEqual(names, want) {
		t.Fatalf("names = %v, error = %v, want %v", names, err, want)
	}
	if _, err := slideshowNames(fstest.MapFS{"assets/README.md": {Data: nil}}); err == nil {
		t.Fatal("empty slideshow was accepted")
	}
	if _, err := slideshowNames(fstest.MapFS{}); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing assets error = %v", err)
	}
}

func TestBadgerAssets(t *testing.T) {
	names, err := slideshowNames(assets)
	if err != nil {
		t.Fatal(err)
	}
	contentWidths := []int{169, 192, 178, 185}
	for i, name := range names {
		t.Run(name, func(t *testing.T) {
			data, err := assets.ReadFile(name)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) != 9476 || binary.LittleEndian.Uint16(data[:2]) != 296 ||
				binary.LittleEndian.Uint16(data[2:4]) != 128 {
				t.Fatal("expected 296x128 packed 2bpp image with a four-byte header")
			}
			display := &assetDisplay{pixels: make([]color.RGBA, 296*128)}
			if err := grayimage.Display(display, data); err != nil {
				t.Fatal(err)
			}
			if display.writes != 37888 || display.refreshes != 1 {
				t.Fatalf("writes = %d, refreshes = %d", display.writes, display.refreshes)
			}
			var shades [4]int
			left := (296 - contentWidths[i]) / 2
			for y := 0; y < 128; y++ {
				for x := 0; x < 296; x++ {
					p := display.pixels[y*296+x]
					if p.R != p.G || p.R != p.B || p.A != 255 || p.R%85 != 0 {
						t.Fatalf("invalid shade at %d,%d: %v", x, y, p)
					}
					shades[p.R/85]++
					if (x < left || x >= left+contentWidths[i]) && p.R != 255 {
						t.Fatalf("white aspect-ratio padding changed at %d,%d", x, y)
					}
				}
			}
			for shade, count := range shades {
				if count == 0 {
					t.Fatalf("shade %d is missing", shade)
				}
			}
		})
	}
}

func TestBadgerDimensionsAndInvalidAssets(t *testing.T) {
	panel := uc8151.NewGrayscale(nil, nil, nil, nil, nil)
	if err := panel.SetRotation(drivers.Rotation270); err != nil {
		t.Fatal(err)
	}
	if w, h := panel.Size(); w != 296 || h != 128 {
		t.Fatalf("rotated panel size = %dx%d", w, h)
	}
	data := make([]byte, 9476)
	binary.LittleEndian.PutUint16(data[:2], 296)
	binary.LittleEndian.PutUint16(data[2:4], 128)
	for _, tc := range []struct {
		data []byte
		err  error
	}{
		{nil, grayimage.ErrHeader},
		{data[:3], grayimage.ErrHeader},
		{[]byte{8, 1, 176, 0}, grayimage.ErrDimensions},
		{[]byte{0, 0, 128, 0}, grayimage.ErrDimensions},
		{data[:9475], grayimage.ErrLength},
		{append(append([]byte{}, data...), 0), grayimage.ErrLength},
	} {
		display := &assetDisplay{pixels: make([]color.RGBA, 296*128)}
		if err := grayimage.Display(display, tc.data); !errors.Is(err, tc.err) {
			t.Fatalf("error = %v, want %v", err, tc.err)
		}
		if display.writes != 0 || display.refreshes != 0 {
			t.Fatal("invalid asset modified the display")
		}
	}
	display := &assetDisplay{pixels: make([]color.RGBA, 296*128), err: errors.New("display failure")}
	if err := grayimage.Display(display, data); !errors.Is(err, display.err) {
		t.Fatalf("display error = %v", err)
	}
	if err := grayimage.Display(panel, data); !errors.Is(err, uc8151.ErrNotConfigured) {
		t.Fatalf("UC8151 error was not propagated: %v", err)
	}
}
