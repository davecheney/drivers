package uc8151

import (
	"bytes"
	"errors"
	"image/color"
	"reflect"
	"testing"
	"time"

	"tinygo.org/x/drivers"
)

func newGray16Test(t *testing.T) (*Grayscale16Device, *graySPI, *grayOutput, *grayInput) {
	t.Helper()
	_, bus, rst, busy := newGrayTest(t)
	return NewGrayscale16(bus, bus.cs, bus.dc, rst, busy), bus, rst, busy
}

func gray16Shade(d *Grayscale16Device, pixel int) byte {
	shift := uint(6 - pixel%4*2)
	return (d.core.buffer[pixel/4]>>shift)&3 | ((d.high[pixel/4]>>shift)&3)<<2
}

func setGray16TestPixels(d *Grayscale16Device) {
	for x := int16(0); x < 16; x++ {
		gray := uint8(x * 17)
		d.SetPixel(x, 0, color.RGBA{gray, gray, gray, 255})
	}
}

func TestGrayscale16New(t *testing.T) {
	d, bus, rst, busy := newGray16Test(t)
	if len(d.core.buffer)+len(d.high) != 18944 {
		t.Fatal("expected an 18,944-byte 4bpp framebuffer")
	}
	for _, plane := range [][]byte{d.core.buffer[:], d.high[:]} {
		if !bytes.Equal(plane, bytes.Repeat([]byte{0xff}, 9472)) {
			t.Fatal("new framebuffer is not white")
		}
	}
	want := [15]byte{2, 4, 6, 8, 10, 12, 14, 17, 19, 21, 23, 25, 27, 29, 32}
	if d.frames != want {
		t.Fatalf("default frames = %v, want %v", d.frames, want)
	}
	for _, operation := range []func() error{d.Display, d.ClearDisplay, d.DeepSleep} {
		if err := operation(); !errors.Is(err, ErrNotConfigured) {
			t.Fatalf("unconfigured operation = %v", err)
		}
	}
	if bus.calls != 0 || busy.reads != 0 || len(rst.changes) != 0 {
		t.Fatal("constructor or unconfigured operation performed hardware I/O")
	}
}

func TestGrayscale16Configure(t *testing.T) {
	for _, custom := range []bool{false, true} {
		d, _, rst, _ := newGray16Test(t)
		d.SetPixel(0, 0, color.RGBA{})
		cfg := Grayscale16Config{Rotation: drivers.Rotation270}
		want := DefaultGrayscale16Frames()
		if custom {
			want[0], want[5] = 1, 11
			cfg.Frames = want
		}
		var stages []string
		cfg.OnInitStage = func(stage string) { stages = append(stages, stage) }
		if err := d.Configure(cfg); err != nil {
			t.Fatal(err)
		}
		if d.frames != want || !d.core.configured || len(stages) != 12 ||
			!reflect.DeepEqual(rst.changes, []bool{false, true}) {
			t.Fatal("incorrect frames, initialization, or hardware reset")
		}
		if w, h := d.Size(); w != 296 || h != 128 || d.Rotation() != drivers.Rotation270 {
			t.Fatal("incorrect landscape dimensions or rotation")
		}
		for i := 0; i < EPD_WIDTH*EPD_HEIGHT; i++ {
			if gray16Shade(d, i) != 15 {
				t.Fatal("Configure did not clear both framebuffer planes")
			}
		}
	}
}

func TestGrayscale16InvalidConfig(t *testing.T) {
	for _, mutate := range []func(*Grayscale16Config){
		func(c *Grayscale16Config) { c.Frames[0] = 0 },
		func(c *Grayscale16Config) { c.Frames[1] = c.Frames[0] },
		func(c *Grayscale16Config) { c.Frames[1] = 1 },
		func(c *Grayscale16Config) { c.Frames[14] = 31 },
		func(c *Grayscale16Config) { c.Frames[14] = 33 },
	} {
		d, bus, rst, _ := newGray16Test(t)
		cfg := Grayscale16Config{Frames: DefaultGrayscale16Frames()}
		mutate(&cfg)
		if err := d.Configure(cfg); !errors.Is(err, ErrInvalidGrayscale16Frames) {
			t.Fatalf("invalid frames %v: %v", cfg.Frames, err)
		}
		if bus.calls != 0 || len(rst.changes) != 0 || d.frames != DefaultGrayscale16Frames() {
			t.Fatal("invalid configuration changed state or performed I/O")
		}
	}
	for _, tc := range []struct {
		cfg Grayscale16Config
		err error
	}{
		{Grayscale16Config{Rotation: 4}, ErrInvalidRotation},
		{Grayscale16Config{BusyTimeout: -time.Second}, ErrInvalidBusyTimeout},
	} {
		d, bus, _, _ := newGray16Test(t)
		if err := d.Configure(tc.cfg); !errors.Is(err, tc.err) || bus.calls != 0 {
			t.Fatalf("invalid controller configuration = %v, want %v without I/O", err, tc.err)
		}
	}
}

func TestGrayscale16Pixels(t *testing.T) {
	d, bus, _, _ := newGray16Test(t)
	setGray16TestPixels(d)
	if !bytes.Equal(d.core.buffer[:4], []byte{0x1b, 0x1b, 0x1b, 0x1b}) ||
		!bytes.Equal(d.high[:4], []byte{0x00, 0x55, 0xaa, 0xff}) {
		t.Fatal("incorrect split 4bpp packing")
	}
	for gray := 0; gray < 256; gray++ {
		c := uint8(gray)
		d.SetPixel(0, 0, color.RGBA{c, c, c, 0})
		if got := gray16Shade(d, 0); got != byte(gray>>4) {
			t.Fatalf("gray %d = shade %d, want %d", gray, got, gray>>4)
		}
		if gray16Shade(d, 1) != 1 {
			t.Fatal("pixel update modified its neighbor")
		}
	}
	for _, tc := range []struct {
		c     color.RGBA
		shade byte
	}{
		{color.RGBA{R: 255}, 4},
		{color.RGBA{G: 255}, 9},
		{color.RGBA{B: 255}, 1},
	} {
		d.SetPixel(0, 0, tc.c)
		if got := gray16Shade(d, 0); got != tc.shade {
			t.Fatalf("luminance of %v = %d, want %d", tc.c, got, tc.shade)
		}
	}
	d.ClearBuffer()
	if gray16Shade(d, 0) != 15 || bus.calls != 0 {
		t.Fatal("buffer operation did not clear or performed hardware I/O")
	}
}

func TestGrayscale16Rotations(t *testing.T) {
	corners := [4][4]int{
		{0, 127, 37760, 37887}, {127, 37887, 0, 37760},
		{37887, 37760, 127, 0}, {37760, 0, 37887, 127},
	}
	d, _, _, _ := newGray16Test(t)
	for rotation := drivers.Rotation(drivers.Rotation0); rotation <= drivers.Rotation270; rotation++ {
		if err := d.SetRotation(rotation); err != nil {
			t.Fatal(err)
		}
		w, h := d.Size()
		if rotation%2 == 0 && (w != 128 || h != 296) || rotation%2 == 1 && (w != 296 || h != 128) {
			t.Fatalf("rotation %d size = %dx%d", rotation, w, h)
		}
		for j, point := range [][2]int16{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}} {
			d.ClearBuffer()
			d.SetPixel(point[0], point[1], color.RGBA{85, 85, 85, 0})
			for i := 0; i < EPD_WIDTH*EPD_HEIGHT; i++ {
				want := byte(15)
				if i == corners[rotation][j] {
					want = 5
				}
				if got := gray16Shade(d, i); got != want {
					t.Fatalf("rotation %d corner %d pixel %d = %d, want %d", rotation, j, i, got, want)
				}
			}
		}
		d.ClearBuffer()
		for _, point := range [][2]int16{{-1, 0}, {0, -1}, {w, 0}, {0, h}} {
			d.SetPixel(point[0], point[1], color.RGBA{})
		}
		for i := 0; i < EPD_WIDTH*EPD_HEIGHT; i++ {
			if gray16Shade(d, i) != 15 {
				t.Fatal("out-of-bounds write changed the framebuffer")
			}
		}
	}
	if err := d.SetRotation(4); !errors.Is(err, ErrInvalidRotation) || d.Rotation() != drivers.Rotation270 {
		t.Fatal("invalid rotation was accepted")
	}
}

func TestGrayscale16Planes(t *testing.T) {
	for pixels := 0; pixels <= 0xffff; pixels++ {
		for first := byte(1); first < 16; first += 3 {
			var low, high uint16
			var old, current byte
			for i := 0; i < 8; i++ {
				shade := byte(pixels>>uint(i%4*4)) & 15
				low = low<<2 | uint16(shade&3)
				high = high<<2 | uint16(shade>>2)
				old <<= 1
				current <<= 1
				switch 15 - shade {
				case first:
				case first + 1:
					old |= 1
					current |= 1
				case first + 2:
					current |= 1
				default:
					old |= 1
				}
			}
			if got := grayscale16Plane(low, high, first, true); got != old {
				t.Fatalf("pixels %04x first %d: old %02x, want %02x", pixels, first, got, old)
			}
			if got := grayscale16Plane(low, high, first, false); got != current {
				t.Fatalf("pixels %04x first %d: new %02x, want %02x", pixels, first, got, current)
			}
		}
	}
}

func TestGrayscale16Display(t *testing.T) {
	d, bus, _, _ := newGray16Test(t)
	d.core.configured = true
	setGray16TestPixels(d)
	low, high := d.core.buffer, d.high
	for attempt := 0; attempt < 2; attempt++ {
		bus.writes = nil
		if err := d.Display(); err != nil {
			t.Fatal(err)
		}
		if len(bus.writes) != 6*13 {
			t.Fatalf("refresh writes = %d, want white reset and five passes", len(bus.writes))
		}
		checkGrayRefresh(t, bus.writes[:13], true, 0, 0)
		var driven [EPD_WIDTH * EPD_HEIGHT]byte
		for pass := 0; pass < 5; pass++ {
			first := byte(pass*3 + 1)
			writes := bus.writes[(pass+1)*13 : (pass+2)*13]
			vcom := [44]byte{0, d.frames[first+1], 0, 0, 0, 1}
			if writes[0].command != LUT_VCOM || !bytes.Equal(writes[0].data, vcom[:]) {
				t.Fatalf("pass %d: incorrect VCOM", pass)
			}
			for lut, duration := range []byte{d.frames[first-1], 0, d.frames[first+1], d.frames[first]} {
				want := [42]byte{}
				if duration != 0 {
					want = [42]byte{0x55, duration, 0, 0, 0, 1}
				}
				if writes[lut+1].command != byte(LUT_WW+lut) || !bytes.Equal(writes[lut+1].data, want[:]) {
					t.Fatalf("pass %d LUT %d = %x, want %x", pass, lut, writes[lut+1].data, want)
				}
			}
			commands := []byte{PON, PTOU, DTM1, DSP, DTM2, DSP, DRF, POF}
			for i, command := range commands {
				if writes[i+5].command != command {
					t.Fatalf("pass %d command %d = %02x, want %02x", pass, i, writes[i+5].command, command)
				}
			}
			if len(writes[7].data) != 4736 || len(writes[9].data) != 4736 {
				t.Fatal("incorrect native bitplane size")
			}
			for pixel := 0; pixel < EPD_WIDTH*EPD_HEIGHT; pixel++ {
				shift := uint(7 - pixel%8)
				old := (writes[7].data[pixel/8] >> shift) & 1
				current := (writes[9].data[pixel/8] >> shift) & 1
				duration := writes[1+old+2*current].data[1]
				darkness := 15 - gray16Shade(d, pixel)
				want := byte(0)
				if darkness >= first && darkness < first+3 {
					want = d.frames[darkness-1]
					driven[pixel]++
				}
				if duration != want {
					t.Fatalf("pass %d pixel %d duration = %d, want %d", pass, pixel, duration, want)
				}
			}
		}
		for pixel, count := range driven {
			want := byte(0)
			if gray16Shade(d, pixel) != 15 {
				want = 1
			}
			if count != want {
				t.Fatalf("pixel %d driven %d times, want %d", pixel, count, want)
			}
		}
		if low != d.core.buffer || high != d.high {
			t.Fatal("Display changed the framebuffer")
		}
	}
}

func TestGrayscale16EmptyPasses(t *testing.T) {
	d, bus, _, _ := newGray16Test(t)
	d.core.configured = true
	if err := d.Display(); err != nil {
		t.Fatal(err)
	}
	if len(bus.writes) != 13 {
		t.Fatal("white framebuffer should only perform the white reset")
	}
	d.SetPixel(0, 0, color.RGBA{})
	bus.writes = nil
	if err := d.Display(); err != nil {
		t.Fatal(err)
	}
	if len(bus.writes) != 26 || bus.writes[16].data[1] != 32 {
		t.Fatal("black-only frame should have one masked pass")
	}
}

func TestGrayscale16SPIErrors(t *testing.T) {
	d, bus, _, _ := newGray16Test(t)
	d.core.configured = true
	setGray16TestPixels(d)
	if err := d.Display(); err != nil {
		t.Fatal(err)
	}
	calls := bus.calls
	for failAt := 1; failAt <= calls; failAt++ {
		d, bus, _, _ := newGray16Test(t)
		d.core.configured = true
		setGray16TestPixels(d)
		bus.failAt = failAt
		if err := d.Display(); !errors.Is(err, bus.err) {
			t.Fatalf("call %d error = %v, want SPI failure", failAt, err)
		}
		if bus.calls != failAt || !bus.cs.high {
			t.Fatalf("call %d continued after error or left CS low", failAt)
		}
	}
	d, bus, _, _ = newGray16Test(t)
	bus.failAt = 1
	if err := d.Configure(Grayscale16Config{}); !errors.Is(err, bus.err) || d.core.configured {
		t.Fatal("configuration failure was not returned")
	}
}

func TestGrayscale16BusyAndSleep(t *testing.T) {
	for _, command := range []byte{PON, DRF, POF} {
		d, bus, _, busy := newGray16Test(t)
		d.core.configured = true
		d.core.timeout = time.Millisecond
		setGray16TestPixels(d)
		seen := 0
		bus.onCommand = func(value byte) {
			if value == command {
				seen++
				if seen == 2 {
					busy.high = false
				}
			}
		}
		if err := d.Display(); !errors.Is(err, ErrBusyTimeout) {
			t.Fatalf("busy after %02x: %v", command, err)
		}
		if bus.writes[len(bus.writes)-1].command != command {
			t.Fatal("continued after BUSY timeout")
		}
	}
	d, _, _, _ := newGray16Test(t)
	d.core.configured = true
	if err := d.DeepSleep(); err != nil {
		t.Fatal(err)
	}
	if err := d.Display(); !errors.Is(err, ErrNotConfigured) {
		t.Fatal("refresh allowed after sleep")
	}
	if err := d.Configure(Grayscale16Config{}); err != nil {
		t.Fatal(err)
	}
	if err := d.Display(); err != nil {
		t.Fatal(err)
	}
}
