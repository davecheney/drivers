//go:build arduino_due

package main

import "machine"

func init() {
	// The Arduino Due has no onboard WS2812, so drive an external strip.
	neo = machine.D6
}
