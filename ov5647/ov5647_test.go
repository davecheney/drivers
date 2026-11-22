package ov5647

import (
	"testing"

	"tinygo.org/x/drivers/tester"
)

func newDevice(t *testing.T) (*Device, *tester.I2CDeviceAddr16) {
	bus := tester.NewI2CBus(t)
	fake := tester.NewI2CDeviceAddr16(t, Address)
	fake.Registers[regChipIDHigh] = 0x56
	fake.Registers[regChipIDLow] = 0x47
	bus.AddDevice(fake)
	return New(bus), fake
}

func TestConnected(t *testing.T) {
	d, fake := newDevice(t)
	if !d.Connected() {
		t.Fatal("expected sensor to be detected")
	}
	fake.Registers[regChipIDLow] = 0x40
	if d.Connected() {
		t.Fatal("expected wrong chip ID to be rejected")
	}
}

func TestStartStop(t *testing.T) {
	d, fake := newDevice(t)
	if err := d.Start(); err != nil {
		t.Fatal(err)
	}
	if got := fake.Registers[regMIPICtrl00]; got != 0x24 {
		t.Fatalf("MIPI ctrl after Start = %#x, want 0x24", got)
	}
	if got := fake.Registers[regModeSelect]; got != 0x01 {
		t.Fatalf("mode select after Start = %#x, want 0x01", got)
	}
	if err := d.Stop(); err != nil {
		t.Fatal(err)
	}
	if got := fake.Registers[regModeSelect]; got != 0x00 {
		t.Fatalf("mode select after Stop = %#x, want 0x00", got)
	}
}

func TestConfigure(t *testing.T) {
	d, fake := newDevice(t)
	if err := d.Configure(Config{}); err != nil {
		t.Fatal(err)
	}
	w := uint16(fake.Registers[0x3808])<<8 | uint16(fake.Registers[0x3809])
	h := uint16(fake.Registers[0x380a])<<8 | uint16(fake.Registers[0x380b])
	if w != 800 || h != 640 {
		t.Fatalf("output size = %dx%d, want 800x640", w, h)
	}
	if got := fake.Registers[0x3034]; got != 0x18 {
		t.Fatalf("0x3034 = %#x, want 0x18 for RAW8", got)
	}
	if got := fake.Registers[regModeSelect]; got != 0x00 {
		t.Fatalf("stream is on after Configure")
	}
}

func TestConfigureFrameSize(t *testing.T) {
	d, _ := newDevice(t)
	for _, c := range []Config{{Width: 801, Height: 640}, {Width: 800, Height: 644}, {Width: 404, Height: 320}, {Width: 400, Height: 0}} {
		if err := d.Configure(c); err != ErrFrameSize {
			t.Fatalf("%dx%d: err = %v, want ErrFrameSize", c.Width, c.Height, err)
		}
	}
}

func TestConfigureCrop(t *testing.T) {
	d, fake := newDevice(t)
	if err := d.Configure(Config{Width: 400, Height: 320}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		reg  uint16
		want uint16
	}{
		{regXOutputSize, 400},
		{regYOutputSize, 320},
		{regXOffset, 208},
		{regYOffset, 160},
	} {
		if got := uint16(fake.Registers[c.reg])<<8 | uint16(fake.Registers[c.reg+1]); got != c.want {
			t.Errorf("reg %#x = %d, want %d", c.reg, got, c.want)
		}
	}
}

func TestConfigureNotDetected(t *testing.T) {
	d, fake := newDevice(t)
	fake.Registers[regChipIDHigh] = 0
	if err := d.Configure(Config{}); err != ErrNotDetected {
		t.Fatalf("err = %v, want ErrNotDetected", err)
	}
}

func TestTestPattern(t *testing.T) {
	d, fake := newDevice(t)
	if err := d.SetTestPattern(true); err != nil {
		t.Fatal(err)
	}
	if got := fake.Registers[regTestPattern]; got != 0x80 {
		t.Fatalf("test pattern = %#x, want 0x80", got)
	}
}
