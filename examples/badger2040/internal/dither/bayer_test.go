package dither

import "testing"

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

func TestSixteenShades(t *testing.T) {
	for gray := 0; gray < 256; gray++ {
		sum := 0
		for y := int16(0); y < 8; y++ {
			for x := int16(0); x < 8; x++ {
				got := Sixteen(uint8(gray), x, y)
				if got%17 != 0 || int(got/17) < gray/17 || int(got/17) > (gray+16)/17 {
					t.Fatalf("gray %d at %d,%d = %d, not an adjacent shade", gray, x, y, got)
				}
				if gray > 0 && got < Sixteen(uint8(gray-1), x, y) {
					t.Fatalf("gray %d at %d,%d is not monotonic", gray, x, y)
				}
				if got != Sixteen(uint8(gray), x+8, y+8) {
					t.Fatal("pattern does not repeat every eight pixels")
				}
				sum += int(got / 17)
			}
		}
		if want := gray/17*64 + (gray%17*64+8)/17; sum != want {
			t.Fatalf("gray %d tile sum = %d, want %d", gray, sum, want)
		}
	}
}
