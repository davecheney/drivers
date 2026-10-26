//go:build baremetal && avr

package legacy

import "machine"

const (
	pulldown = machine.PinInput
	pullup   = machine.PinInputPullup
)
