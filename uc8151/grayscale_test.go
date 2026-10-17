package uc8151

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"reflect"
	"testing"
	"time"

	"tinygo.org/x/drivers"
)

type grayOutput struct {
	high    bool
	changes []bool
}

func (p *grayOutput) Set(high bool) {
	p.high = high
	p.changes = append(p.changes, high)
}

type grayInput struct {
	high   bool
	levels []bool
	reads  int
}

func (p *grayInput) Get() bool {
	p.reads++
	if len(p.levels) != 0 {
		level := p.levels[0]
		p.levels = p.levels[1:]
		return level
	}
	return p.high
}

type grayWrite struct {
	command byte
	data    []byte
}

type graySPI struct {
	t         *testing.T
	cs, dc    *grayOutput
	writes    []grayWrite
	calls     int
	failAt    int
	err       error
	onCommand func(byte)
}

func (s *graySPI) check(data bool) error {
	s.t.Helper()
	if s.cs.high || s.dc.high != data {
		s.t.Fatal("incorrect CS or DC during transfer")
	}
	s.calls++
	if s.calls == s.failAt {
		return s.err
	}
	return nil
}

func (s *graySPI) Transfer(value byte) (byte, error) {
	if err := s.check(false); err != nil {
		return 0, err
	}
	s.writes = append(s.writes, grayWrite{command: value})
	if s.onCommand != nil {
		s.onCommand(value)
	}
	return 0, nil
}

func (s *graySPI) Tx(write, read []byte) error {
	if err := s.check(true); err != nil {
		return err
	}
	if read != nil || len(s.writes) == 0 {
		s.t.Fatal("unexpected SPI read or data without a command")
	}
	last := &s.writes[len(s.writes)-1]
	last.data = append(last.data, write...)
	return nil
}

func newGrayTest(t *testing.T) (*GrayscaleDevice, *graySPI, *grayOutput, *grayInput) {
	t.Helper()
	cs, dc, rst := &grayOutput{}, &grayOutput{}, &grayOutput{}
	busy := &grayInput{high: true}
	bus := &graySPI{t: t, cs: cs, dc: dc, err: errors.New("SPI failure")}
	return NewGrayscale(bus, cs, dc, rst, busy), bus, rst, busy
}

func grayConfigured(t *testing.T) (*GrayscaleDevice, *graySPI, *grayInput) {
	t.Helper()
	d, bus, _, busy := newGrayTest(t)
	d.configured = true
	return d, bus, busy
}

func TestGrayscaleNew(t *testing.T) {
	d, bus, rst, busy := newGrayTest(t)
	if w, h := d.Size(); w != 128 || h != 296 {
		t.Fatalf("size = %dx%d, want 128x296", w, h)
	}
	if len(d.buffer) != 9472 || !bytes.Equal(d.buffer[:], bytes.Repeat([]byte{0xff}, 9472)) {
		t.Fatal("expected a white 9472-byte 2bpp framebuffer")
	}
	if bus.calls != 0 || busy.reads != 0 || len(bus.cs.changes)+len(bus.dc.changes)+len(rst.changes) != 0 {
		t.Fatal("constructor performed hardware I/O")
	}
	for _, operation := range []func() error{d.Display, d.ClearDisplay, d.DeepSleep} {
		if err := operation(); !errors.Is(err, ErrNotConfigured) {
			t.Fatalf("unconfigured operation = %v", err)
		}
	}
	if bus.calls != 0 {
		t.Fatal("unconfigured operations performed I/O")
	}
	if d.lightFrames != 5 || d.darkFrames != 9 {
		t.Fatal("incorrect default gray pulse lengths")
	}
}

func TestGrayscaleConfigure(t *testing.T) {
	d, bus, rst, _ := newGrayTest(t)
	var stages []string
	if err := d.Configure(GrayscaleConfig{
		Rotation: drivers.Rotation270,
		OnInitStage: func(stage string) {
			stages = append(stages, stage)
		},
	}); err != nil {
		t.Fatal(err)
	}
	want := []grayWrite{
		{0x00, []byte{0x00}},
		{0x01, []byte{0x03, 0x00, 0x26, 0x26, 0x03}},
		{0x06, []byte{0x17, 0x17, 0x17}},
		{0x04, nil},
		{0x00, []byte{0xbf}},
		{0x03, []byte{0x30}},
		{0x41, []byte{0x00}},
		{0x60, []byte{0x22}},
		{0x50, []byte{0xcc}},
		{0x30, []byte{0x3a}},
		{0x02, nil},
	}
	if !reflect.DeepEqual(bus.writes, want) {
		t.Fatalf("initialization = %#v, want %#v", bus.writes, want)
	}
	wantStages := []string{"reset", "software reset", "power settings", "booster", "power on", "panel settings",
		"power off timing", "temperature sensor", "timing", "VCOM interval", "100Hz waveform clock", "power off"}
	if !reflect.DeepEqual(stages, wantStages) {
		t.Fatalf("stages = %v, want %v", stages, wantStages)
	}
	if !reflect.DeepEqual(rst.changes, []bool{false, true}) || !d.configured || d.timeout != 30*time.Second {
		t.Fatal("incorrect reset, ready state, or default timeout")
	}
	if w, h := d.Size(); w != 296 || h != 128 {
		t.Fatalf("landscape size = %dx%d, want 296x128", w, h)
	}
}

func TestGrayscaleInvalidConfig(t *testing.T) {
	for _, tc := range []struct {
		cfg GrayscaleConfig
		err error
	}{
		{GrayscaleConfig{Rotation: 4}, ErrInvalidRotation},
		{GrayscaleConfig{BusyTimeout: -time.Second}, ErrInvalidBusyTimeout},
		{GrayscaleConfig{LightGrayFrames: 10, DarkGrayFrames: 10}, ErrInvalidGrayscaleFrames},
		{GrayscaleConfig{LightGrayFrames: 11, DarkGrayFrames: 10}, ErrInvalidGrayscaleFrames},
		{GrayscaleConfig{DarkGrayFrames: 32}, ErrInvalidGrayscaleFrames},
		{GrayscaleConfig{DarkGrayFrames: 255}, ErrInvalidGrayscaleFrames},
		{GrayscaleConfig{LightGrayFrames: 255}, ErrInvalidGrayscaleFrames},
	} {
		d, bus, rst, _ := newGrayTest(t)
		before := d.buffer
		if err := d.Configure(tc.cfg); !errors.Is(err, tc.err) {
			t.Fatalf("error = %v, want %v", err, tc.err)
		}
		if bus.calls != 0 || len(rst.changes) != 0 || d.buffer != before {
			t.Fatal("invalid configuration changed state or performed I/O")
		}
	}
}

func TestGrayscalePacking(t *testing.T) {
	d, bus, _, _ := newGrayTest(t)
	for x := int16(0); x < 8; x++ {
		gray := uint8(x%4) * 85
		d.SetPixel(x, 0, color.RGBA{gray, gray, gray, 255})
	}
	if !bytes.Equal(d.buffer[:3], []byte{0x1b, 0x1b, 0xff}) {
		t.Fatalf("packed pixels = %x, want 1b1bff", d.buffer[:3])
	}
	d.SetPixel(3, 0, color.RGBA{85, 85, 85, 0})
	if d.buffer[0] != 0x19 {
		t.Fatal("overwriting a pixel changed its neighbors or used alpha")
	}
	d.SetPixel(127, 0, color.RGBA{})
	d.SetPixel(0, 1, color.RGBA{})
	d.SetPixel(127, 295, color.RGBA{})
	if d.buffer[31] != 0xfc || d.buffer[32] != 0x3f || d.buffer[9471] != 0xfc {
		t.Fatal("incorrect row or framebuffer boundary")
	}
	for _, tc := range []struct {
		color color.RGBA
		shade byte
	}{
		{color.RGBA{R: 255}, 1},
		{color.RGBA{G: 255}, 2},
		{color.RGBA{B: 255}, 0},
		{color.RGBA{63, 63, 63, 255}, 0},
		{color.RGBA{64, 64, 64, 255}, 1},
		{color.RGBA{127, 127, 127, 255}, 1},
		{color.RGBA{128, 128, 128, 255}, 2},
		{color.RGBA{191, 191, 191, 255}, 2},
		{color.RGBA{192, 192, 192, 255}, 3},
	} {
		d.SetPixel(0, 0, tc.color)
		if d.buffer[0]>>6 != tc.shade {
			t.Fatalf("luminance of %v = %d, want %d", tc.color, d.buffer[0]>>6, tc.shade)
		}
	}
	d.ClearBuffer()
	if !bytes.Equal(d.buffer[:], bytes.Repeat([]byte{0xff}, len(d.buffer))) || bus.calls != 0 {
		t.Fatal("buffer operations performed I/O or did not clear to white")
	}
}

func TestGrayscaleRotations(t *testing.T) {
	for rotation := drivers.Rotation(drivers.Rotation0); rotation <= drivers.Rotation270; rotation++ {
		t.Run(fmt.Sprint(rotation), func(t *testing.T) {
			d, _, _, _ := newGrayTest(t)
			if err := d.SetRotation(rotation); err != nil {
				t.Fatal(err)
			}
			if d.Rotation() != rotation {
				t.Fatal("rotation was not retained")
			}
			w, h := d.Size()
			if rotation%2 == 0 && (w != 128 || h != 296) || rotation%2 == 1 && (w != 296 || h != 128) {
				t.Fatalf("incorrect dimensions %dx%d", w, h)
			}
			for _, point := range [][2]int16{{-1, 0}, {0, -1}, {w, 0}, {0, h}, {w, h}} {
				d.SetPixel(point[0], point[1], color.RGBA{})
			}
			if !bytes.Equal(d.buffer[:], bytes.Repeat([]byte{0xff}, len(d.buffer))) {
				t.Fatal("out of bounds writes modified the buffer")
			}
			corners := [][2]int16{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}}
			indices := [4][4]int{{0, 127, 37760, 37887}, {127, 37887, 0, 37760},
				{37887, 37760, 127, 0}, {37760, 0, 37887, 127}}
			for i, point := range corners {
				d.ClearBuffer()
				d.SetPixel(point[0], point[1], color.RGBA{})
				index := indices[rotation][i]
				for j, value := range d.buffer {
					want := byte(0xff)
					if j == index/4 {
						want &^= 3 << uint(6-index%4*2)
					}
					if value != want {
						t.Fatalf("corner %d byte %d = %02x, want %02x", i, j, value, want)
					}
				}
			}
			if err := d.SetRotation(4); !errors.Is(err, ErrInvalidRotation) || d.Rotation() != rotation {
				t.Fatal("invalid rotation was accepted")
			}
		})
	}
}

func TestGrayscalePlanesExhaustive(t *testing.T) {
	for pixels := 0; pixels <= 0xffff; pixels++ {
		var old, current byte
		for i := 7; i >= 0; i-- {
			shade := (pixels >> (2 * i)) & 3
			old <<= 1
			current <<= 1
			switch shade {
			case 0:
				current |= 1
			case 1:
				old |= 1
				current |= 1
			case 3:
				old |= 1
			}
		}
		if got := grayscalePlane(byte(pixels>>8), byte(pixels), true); got != old {
			t.Fatalf("old plane for %04x = %02x, want %02x", pixels, got, old)
		}
		if got := grayscalePlane(byte(pixels>>8), byte(pixels), false); got != current {
			t.Fatalf("new plane for %04x = %02x, want %02x", pixels, got, current)
		}
	}
}

func checkGrayRefresh(t *testing.T, writes []grayWrite, white bool, old, current byte) {
	t.Helper()
	commands := []byte{0x20, 0x21, 0x22, 0x23, 0x24, 0x04, 0x92, 0x10, 0x11, 0x13, 0x11, 0x12, 0x02}
	if len(writes) != len(commands) {
		t.Fatalf("got %d commands, want %d", len(writes), len(commands))
	}
	for i, command := range commands {
		if writes[i].command != command {
			t.Fatalf("command %d = %02x, want %02x", i, writes[i].command, command)
		}
	}
	wantVCOM := make([]byte, 44)
	wantTables := [4][42]byte{}
	if white {
		copy(wantVCOM, []byte{0, 32, 0, 0, 0, 2, 0, 16, 16, 0, 0, 2, 0, 32, 0, 0, 0, 2})
		for i := range wantTables {
			copy(wantTables[i][:], []byte{0x40, 32, 0, 0, 0, 2, 0x90, 16, 16, 0, 0, 1, 0x80, 32, 0, 0, 0, 2})
		}
	} else {
		copy(wantVCOM, []byte{0, 24, 0, 0, 0, 1})
		copy(wantTables[0][:], []byte{0x55, 5, 0, 0, 0, 1})
		copy(wantTables[2][:], []byte{0x55, 32, 0, 0, 0, 1})
		copy(wantTables[3][:], []byte{0x55, 9, 0, 0, 0, 1})
	}
	if !bytes.Equal(writes[0].data, wantVCOM) {
		t.Fatalf("VCOM LUT = %x, want %x", writes[0].data, wantVCOM)
	}
	for i := range wantTables {
		if !bytes.Equal(writes[i+1].data, wantTables[i][:]) {
			t.Fatalf("LUT %d = %x, want %x", i, writes[i+1].data, wantTables[i])
		}
	}
	if !bytes.Equal(writes[7].data, bytes.Repeat([]byte{old}, 4736)) ||
		!bytes.Equal(writes[9].data, bytes.Repeat([]byte{current}, 4736)) {
		t.Fatal("incorrect native 128x296 old/new bitplanes")
	}
	for _, i := range []int{5, 6, 8, 10, 11, 12} {
		if len(writes[i].data) != 0 {
			t.Fatalf("unexpected payload for command %02x", commands[i])
		}
	}
}

func TestGrayscaleDisplayAndClear(t *testing.T) {
	d, bus, _ := grayConfigured(t)
	for i := range d.buffer {
		d.buffer[i] = 0x1b
	}
	before := d.buffer
	for frame := 0; frame < 2; frame++ {
		value, old, current := byte(0x1b), byte(0x55), byte(0xcc)
		if frame == 1 {
			value, old, current = 0xe4, 0xaa, 0x33
		}
		for i := range d.buffer {
			d.buffer[i] = value
		}
		before = d.buffer
		bus.writes = nil
		if err := d.Display(); err != nil {
			t.Fatal(err)
		}
		if len(bus.writes) != 26 {
			t.Fatalf("frame %d did not include exactly two refreshes", frame)
		}
		checkGrayRefresh(t, bus.writes[:13], true, 0, 0)
		checkGrayRefresh(t, bus.writes[13:], false, old, current)
		if d.buffer != before {
			t.Fatal("Display modified framebuffer")
		}
	}
	bus.writes = nil
	if err := d.ClearDisplay(); err != nil {
		t.Fatal(err)
	}
	checkGrayRefresh(t, bus.writes, true, 0, 0)
	if d.buffer != before {
		t.Fatal("ClearDisplay modified framebuffer")
	}
}

func TestGrayscaleSPIErrors(t *testing.T) {
	for _, method := range []string{"configure", "display", "sleep"} {
		t.Run(method, func(t *testing.T) {
			run := func(d *GrayscaleDevice) error {
				switch method {
				case "configure":
					return d.Configure(GrayscaleConfig{})
				case "display":
					return d.Display()
				default:
					return d.DeepSleep()
				}
			}
			d, bus, _ := grayConfigured(t)
			if err := run(d); err != nil {
				t.Fatal(err)
			}
			calls := bus.calls
			for failAt := 1; failAt <= calls; failAt++ {
				d, bus, _ := grayConfigured(t)
				bus.failAt = failAt
				if err := run(d); !errors.Is(err, bus.err) {
					t.Fatalf("call %d error = %v, want SPI error", failAt, err)
				}
				if !bus.cs.high || bus.calls != failAt {
					t.Fatalf("call %d left CS asserted or continued after error", failAt)
				}
				if method == "configure" && d.configured {
					t.Fatal("failed configuration left device ready")
				}
			}
		})
	}
}

func TestGrayscaleBusy(t *testing.T) {
	d, bus, busy := grayConfigured(t)
	d.timeout = 30 * time.Millisecond
	busy.levels = []bool{false, false, true}
	if err := d.waitReady(); err != nil || busy.reads != 3 {
		t.Fatalf("BUSY was not polled until HIGH: %v, reads %d", err, busy.reads)
	}
	d.timeout = 2 * time.Millisecond
	for _, operation := range []func() error{
		d.Display, d.DeepSleep,
		func() error { return d.Configure(GrayscaleConfig{BusyTimeout: 2 * time.Millisecond}) },
	} {
		busy.high = false
		bus.writes = nil
		if err := operation(); !errors.Is(err, ErrBusyTimeout) {
			t.Fatalf("error = %v, want BUSY timeout", err)
		}
		if len(bus.writes) != 0 {
			t.Fatal("sent commands while BUSY was LOW")
		}
	}
}

func TestGrayscaleSleepAndWake(t *testing.T) {
	d, bus, _ := grayConfigured(t)
	if err := d.DeepSleep(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(bus.writes, []grayWrite{{0x02, nil}, {0x07, []byte{0xa5}}}) || d.configured {
		t.Fatalf("incorrect sleep sequence: %#v", bus.writes)
	}
	if err := d.Display(); !errors.Is(err, ErrNotConfigured) {
		t.Fatal("display allowed after deep sleep")
	}
	if err := d.Configure(GrayscaleConfig{}); err != nil {
		t.Fatal(err)
	}
	if err := d.Display(); err != nil {
		t.Fatal(err)
	}
}

func TestGrayscaleBusyDuringRefresh(t *testing.T) {
	for _, command := range []byte{PON, DRF, POF} {
		t.Run(fmt.Sprintf("%02x", command), func(t *testing.T) {
			d, bus, busy := grayConfigured(t)
			d.timeout = time.Millisecond
			bus.onCommand = func(value byte) {
				if value == command {
					busy.high = false
				}
			}
			if err := d.Display(); !errors.Is(err, ErrBusyTimeout) {
				t.Fatalf("error = %v, want BUSY timeout", err)
			}
			if got := bus.writes[len(bus.writes)-1].command; got != command {
				t.Fatalf("sent command %02x while busy after %02x", got, command)
			}
		})
	}
}

func TestGrayscaleCustomPulseLengths(t *testing.T) {
	for _, cfg := range []GrayscaleConfig{
		{},
		{LightGrayFrames: 1, DarkGrayFrames: 2},
		{LightGrayFrames: 10, DarkGrayFrames: 21},
		{LightGrayFrames: 30, DarkGrayFrames: 31},
		{LightGrayFrames: 2},
		{DarkGrayFrames: 12},
	} {
		d, bus, _, _ := newGrayTest(t)
		if err := d.Configure(cfg); err != nil {
			t.Fatal(err)
		}
		light, dark := cfg.LightGrayFrames, cfg.DarkGrayFrames
		if light == 0 {
			light = 5
		}
		if dark == 0 {
			dark = 9
		}
		bus.writes = nil
		if err := d.writeLUTs(false); err != nil {
			t.Fatal(err)
		}
		if bus.writes[1].data[1] != light || bus.writes[4].data[1] != dark ||
			bus.writes[2].data[1] != 0 || bus.writes[3].data[1] != 32 {
			t.Fatal("custom gray pulses changed white/black or were not applied")
		}
	}
}
