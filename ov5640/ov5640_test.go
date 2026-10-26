package ov5640

import (
	"bytes"
	"errors"
	"testing"
	"time"
)

type transaction struct {
	write []byte
	read  []byte
	err   error
}

type fakeI2C struct {
	t       *testing.T
	address uint16
	steps   []transaction
	next    int
}

func newFakeI2C(t *testing.T, steps ...transaction) *fakeI2C {
	t.Helper()
	bus := &fakeI2C{t: t, address: Address, steps: steps}
	t.Cleanup(func() {
		if bus.next != len(bus.steps) {
			t.Errorf("got %d transactions, want %d", bus.next, len(bus.steps))
		}
	})
	return bus
}

func (bus *fakeI2C) Tx(address uint16, write, read []byte) error {
	bus.t.Helper()
	if address != bus.address {
		bus.t.Fatalf("address = %#x, want %#x", address, bus.address)
	}
	if bus.next >= len(bus.steps) {
		bus.t.Fatalf("unexpected transaction with write %x and read length %d", write, len(read))
	}
	step := bus.steps[bus.next]
	bus.next++
	if !bytes.Equal(write, step.write) || len(read) != len(step.read) {
		bus.t.Fatalf("transaction %d = (%x, %d), want (%x, %d)",
			bus.next, write, len(read), step.write, len(step.read))
	}
	copy(read, step.read)
	return step.err
}

func idTransactions(high, low byte) []transaction {
	return []transaction{
		{write: []byte{0x30, 0x0a}},
		{read: []byte{high}},
		{write: []byte{0x30, 0x0b}},
		{read: []byte{low}},
	}
}

func controlTransactions(before, after byte) []transaction {
	return []transaction{
		{write: []byte{0x30, 0x08}},
		{read: []byte{before}},
		{write: []byte{0x30, 0x08, after}},
	}
}

func TestNew(t *testing.T) {
	dev := New(newFakeI2C(t))
	if dev.Address != Address {
		t.Fatalf("address = %#x, want %#x", dev.Address, Address)
	}
}

func TestRegisterAccess(t *testing.T) {
	bus := newFakeI2C(t,
		transaction{write: []byte{0xab, 0xcd, 0x42}},
		transaction{write: []byte{0xab, 0xcd}},
		transaction{read: []byte{0x42}},
		transaction{write: []byte{0x00, 0x00, 0x00}},
		transaction{write: []byte{0xff, 0xff}},
		transaction{read: []byte{0xff}},
	)
	bus.address = 0x3d
	dev := New(bus)
	dev.Address = bus.address
	if err := dev.WriteRegister(0xabcd, 0x42); err != nil {
		t.Fatal(err)
	}
	if value, err := dev.ReadRegister(0xabcd); err != nil || value != 0x42 {
		t.Fatalf("ReadRegister = (%#x, %v), want (0x42, nil)", value, err)
	}
	if err := dev.WriteRegister(0, 0); err != nil {
		t.Fatal(err)
	}
	if value, err := dev.ReadRegister(0xffff); err != nil || value != 0xff {
		t.Fatalf("ReadRegister = (%#x, %v), want (0xff, nil)", value, err)
	}
}

func TestChipID(t *testing.T) {
	dev := New(newFakeI2C(t, idTransactions(0x56, 0x40)...))
	if id, err := dev.ChipID(); err != nil || id != ID {
		t.Fatalf("ChipID = (%#x, %v), want (%#x, nil)", id, err, ID)
	}
}

func TestCheckID(t *testing.T) {
	for _, tt := range []struct {
		name string
		high byte
		low  byte
		want error
	}{
		{name: "correct", high: 0x56, low: 0x40},
		{name: "wrong high byte", high: 0x55, low: 0x40, want: ErrUnexpectedID},
		{name: "wrong low byte", high: 0x56, low: 0x41, want: ErrUnexpectedID},
		{name: "zero", want: ErrUnexpectedID},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dev := New(newFakeI2C(t, idTransactions(tt.high, tt.low)...))
			if err := dev.CheckID(); !errors.Is(err, tt.want) {
				t.Fatalf("CheckID = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestReset(t *testing.T) {
	for _, value := range []byte{0x02, 0x42, 0x15} {
		dev := New(newFakeI2C(t, controlTransactions(value, value|0x80)...))
		start := time.Now()
		if err := dev.Reset(); err != nil {
			t.Fatal(err)
		}
		if elapsed := time.Since(start); elapsed < time.Millisecond {
			t.Fatalf("reset took %v, want at least 1 ms", elapsed)
		}
	}
}

func TestSetStandby(t *testing.T) {
	for _, tt := range []struct {
		name    string
		enabled bool
		before  byte
		after   byte
	}{
		{name: "enable", enabled: true, before: 0x02, after: 0x42},
		{name: "disable", before: 0x42, after: 0x02},
		{name: "enable twice", enabled: true, before: 0x42, after: 0x42},
		{name: "disable twice", before: 0x02, after: 0x02},
		{name: "preserve bits on enable", enabled: true, before: 0x15, after: 0x55},
		{name: "preserve bits on disable", before: 0x55, after: 0x15},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dev := New(newFakeI2C(t, controlTransactions(tt.before, tt.after)...))
			if err := dev.SetStandby(tt.enabled); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestBusErrors(t *testing.T) {
	for _, tt := range []struct {
		name  string
		steps []transaction
		call  func(*Device) error
	}{
		{
			name: "read",
			steps: []transaction{
				{write: []byte{0x30, 0x08}},
				{read: []byte{0xff}},
			},
			call: func(d *Device) error {
				value, err := d.ReadRegister(0x3008)
				if value != 0 {
					t.Errorf("failed read returned %#x, want 0", value)
				}
				return err
			},
		},
		{
			name:  "write",
			steps: []transaction{{write: []byte{0x30, 0x08, 0x42}}},
			call:  func(d *Device) error { return d.WriteRegister(0x3008, 0x42) },
		},
		{
			name:  "chip ID",
			steps: idTransactions(0x56, 0x40),
			call: func(d *Device) error {
				value, err := d.ChipID()
				if value != 0 {
					t.Errorf("failed chip ID read returned %#x, want 0", value)
				}
				return err
			},
		},
		{name: "check ID", steps: idTransactions(0x56, 0x40), call: (*Device).CheckID},
		{name: "reset", steps: controlTransactions(0x02, 0x82), call: (*Device).Reset},
		{
			name:  "standby on",
			steps: controlTransactions(0x02, 0x42),
			call:  func(d *Device) error { return d.SetStandby(true) },
		},
		{
			name:  "standby off",
			steps: controlTransactions(0x42, 0x02),
			call:  func(d *Device) error { return d.SetStandby(false) },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for failAt := range tt.steps {
				t.Run(string(rune('0'+failAt)), func(t *testing.T) {
					steps := append([]transaction(nil), tt.steps[:failAt+1]...)
					want := errors.New("bus failure")
					steps[failAt].err = want
					dev := New(newFakeI2C(t, steps...))
					if err := tt.call(dev); !errors.Is(err, want) {
						t.Fatalf("error = %v, want %v", err, want)
					}
				})
			}
		})
	}
}
