package ov3660

import (
	"testing"

	"tinygo.org/x/drivers/tester"
)

func newDevice(t *testing.T) (*Device, *tester.I2CDeviceAddr16) {
	bus := tester.NewI2CBus(t)
	fake := tester.NewI2CDeviceAddr16(t, Address)
	fake.Registers[regChipIDHigh] = 0x36
	fake.Registers[regChipIDLow] = 0x60
	bus.AddDevice(fake)
	return New(bus), fake
}

func reg16(fake *tester.I2CDeviceAddr16, r uint16) uint16 {
	return uint16(fake.Registers[r])<<8 | uint16(fake.Registers[r+1])
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

func TestConfigure240x240(t *testing.T) {
	d, fake := newDevice(t)
	if err := d.Configure(Config{Width: 240, Height: 240}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		reg  uint16
		want uint16
	}{
		{regXAddrStart, 256},
		{regXAddrEnd, 1823},
		{regYAddrEnd, 1547},
		{regXOutputSize, 240},
		{regYOutputSize, 240},
		{regXTotalSize, 2044},
		{regYTotalSize, 783},
		{regXOffset, 8},
		{regYOffset, 2},
	} {
		if got := reg16(fake, c.reg); got != c.want {
			t.Errorf("reg %#04x = %d, want %d", c.reg, got, c.want)
		}
	}
	for _, c := range []struct {
		reg  uint16
		want uint8
	}{
		{regFormatCtrl, 0x01},
		{regFormatCtrl00, 0x61},
		{regISPControl01, 0xa3},
		{regTimingTC20, 0x01},
		{regTimingTC21, 0x01},
		{regXIncrement, 0x31},
		{regPLLCtrl1, 8},
		{regPCLKRatio, 8},
	} {
		if got := fake.Registers[c.reg]; got != c.want {
			t.Errorf("reg %#04x = %#02x, want %#02x", c.reg, got, c.want)
		}
	}
}

func TestConfigureBadSize(t *testing.T) {
	d, _ := newDevice(t)
	if err := d.Configure(Config{Width: 250, Height: 240}); err != ErrFrameSize {
		t.Fatalf("got %v, want ErrFrameSize", err)
	}
}

func TestNotDetected(t *testing.T) {
	d, fake := newDevice(t)
	fake.Registers[regChipIDHigh] = 0
	if err := d.Configure(Config{}); err != ErrNotDetected {
		t.Fatalf("got %v, want ErrNotDetected", err)
	}
}
