package main

import (
	"errors"
	"image/color"
	"testing"
)

func TestBayerRanks(t *testing.T) {
	var seen [64]bool
	for _, row := range bayer8 {
		for _, rank := range row {
			if rank >= 64 || seen[rank] {
				t.Fatalf("invalid or repeated Bayer rank %d", rank)
			}
			seen[rank] = true
		}
	}
}

func TestBayerGray(t *testing.T) {
	for gray := 0; gray < 256; gray++ {
		var sum int
		for y := int16(0); y < 8; y++ {
			for x := int16(0); x < 8; x++ {
				got := bayerGray(uint8(gray), x, y)
				if got%85 != 0 || int(got/85) < gray/85 || int(got/85) > (gray+84)/85 {
					t.Fatalf("gray %d at %d,%d = %d, not an adjacent shade", gray, x, y, got)
				}
				if gray > 0 && got < bayerGray(uint8(gray-1), x, y) {
					t.Fatalf("gray %d at %d,%d is not monotonic", gray, x, y)
				}
				if got != bayerGray(uint8(gray), x+8, y+8) {
					t.Fatalf("gray %d at %d,%d does not repeat every eight pixels", gray, x, y)
				}
				sum += int(got / 85)
			}
		}
		want := gray/85*64 + (gray%85*64+42)/85
		if sum != want {
			t.Fatalf("gray %d: tile sum = %d, want %d", gray, sum, want)
		}
	}
	for _, tc := range []struct {
		gray uint8
		x, y int16
		want uint8
	}{
		{42, 0, 0, 85},
		{42, 1, 0, 0},
		{127, 0, 0, 170},
		{127, 1, 0, 85},
		{212, 0, 0, 255},
		{212, 1, 0, 170},
	} {
		if got := bayerGray(tc.gray, tc.x, tc.y); got != tc.want {
			t.Fatalf("gray %d at %d,%d = %d, want %d", tc.gray, tc.x, tc.y, got, tc.want)
		}
	}
}

type gradientDisplay struct {
	t         *testing.T
	pixels    [296 * 128]color.RGBA
	refreshes int
	err       error
}

func (d *gradientDisplay) Size() (int16, int16) { return 296, 128 }

func (d *gradientDisplay) SetPixel(x, y int16, c color.RGBA) {
	d.t.Helper()
	if x < 0 || x >= 296 || y < 0 || y >= 128 {
		d.t.Fatalf("pixel outside Badger landscape display: %d,%d", x, y)
	}
	d.pixels[int(y)*296+int(x)] = c
}

func (d *gradientDisplay) Display() error {
	d.refreshes++
	return d.err
}

func TestGradient(t *testing.T) {
	d := &gradientDisplay{t: t}
	if err := showGradient(d); err != nil {
		t.Fatal(err)
	}
	if d.refreshes != 1 {
		t.Fatalf("refreshes = %d, want 1", d.refreshes)
	}
	var shades [4]int
	for y := 0; y < 128; y++ {
		for x := 0; x < 296; x++ {
			got := d.pixels[y*296+x]
			if got.R != got.G || got.R != got.B || got.A != 255 || got.R%85 != 0 {
				t.Fatalf("invalid shade at %d,%d: %v", x, y, got)
			}
			if y >= 32 && y < 96 {
				want := uint8(255)
				if x >= 8 && x < 288 {
					gray := uint8(((287-x)*255 + 139) / 279)
					want = bayerGray(gray, int16(x-8), int16(y-32))
					shades[got.R/85]++
				}
				if got.R != want {
					t.Fatalf("gradient at %d,%d = %d, want %d", x, y, got.R, want)
				}
			} else if y >= 96 && got.R != 255 {
				t.Fatalf("background at %d,%d is not white", x, y)
			}
		}
	}
	for shade, count := range shades {
		if count == 0 {
			t.Fatalf("gradient shade %d is missing", shade)
		}
	}
	for y := 32; y < 96; y++ {
		if d.pixels[y*296+8].R != 255 || d.pixels[y*296+287].R != 0 {
			t.Fatalf("gradient endpoints at row %d are not white and black", y)
		}
	}
}

func TestGradientDisplayError(t *testing.T) {
	d := &gradientDisplay{t: t, err: errors.New("SPI failure")}
	if err := showGradient(d); !errors.Is(err, d.err) {
		t.Fatalf("display error = %v, want %v", err, d.err)
	}
}
