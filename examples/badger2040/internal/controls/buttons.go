package controls

import (
	"time"

	"tinygo.org/x/drivers/internal/pin"
)

type Buttons uint8

const (
	Up Buttons = 1 << iota
	Down
	A
	B
)

// Pins are the active-high Up, Down, A and B inputs, in that order.
type Pins [4]pin.Input

func (pins Pins) Read() Buttons {
	var pressed Buttons
	for i, pin := range pins {
		if pin.Get() {
			pressed |= 1 << uint(i)
		}
	}
	return pressed
}

type Debouncer struct {
	last, stable Buttons
	changed      time.Time
}

// Update emits press edges after 30ms of stable input, without held-key repeats.
func (b *Debouncer) Update(raw Buttons, now time.Time) Buttons {
	if raw != b.last {
		b.last = raw
		b.changed = now
	}
	if now.Sub(b.changed) < 30*time.Millisecond {
		return 0
	}
	pressed := raw &^ b.stable
	b.stable = raw
	return pressed
}
