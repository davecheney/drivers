//go:build avr

package ws2812

// This file implements the WS2812 protocol for AVR microcontrollers.

import (
	"image/color"
	"machine"
	"runtime/interrupt"
	"unsafe"
)

/*
#include <stdint.h>

__attribute__((always_inline))
void ws2812_writeBytes16(const uint8_t *buf, uint16_t len, volatile uint8_t *port, uint8_t maskSet, uint8_t maskClear) {
	// See:
	// https://wp.josh.com/2014/05/13/ws2812-neopixels-are-not-so-finicky-once-you-get-to-know-them/
	// T0H: 4  cycles or 250ns
	// T0L: 14 cycles or 875ns -> together 18 cycles or 1125ns
	// T1H: 9  cycles or 562ns
	// T1L: 8  cycles or 500ns -> together 17 cycles or 1062ns
	// Moving to the next byte adds 5 cycles to the last low period.
	uint8_t c, i;
	__asm__ __volatile__(
		"0:\n"
		"\t  ld    %[c], X+              ; [2]   load the next byte\n"
		"\t  ldi   %[i], 8               ; [1]\n"
		"1:\n"
		"\t  st    Z, %[maskSet]         ; [2]   set output high\n"
		"\t  lsl   %[c]                  ; [1]   shift off the next bit, store it in C\n"
		"\t  brcs  2f                    ; [1/2] branch if this bit is high (long pulse)\n"
		"\t  st    Z, %[maskClear]       ; [2]   set output low (short pulse)\n"
		"\t2:\n"
		"\t  nop                         ; [4]   wait before changing the output again\n"
		"\t  nop\n"
		"\t  nop\n"
		"\t  nop\n"
		"\t  st    Z, %[maskClear]       ; [2]   set output low (end of pulse)\n"
		"\t  nop                         ; [3]\n"
		"\t  nop\n"
		"\t  nop\n"
		"\t  subi  %[i], 1               ; [1]   subtract one (for the loop)\n"
		"\t  brne  1b                    ; [1/2] send the next bit, if not at the end of the byte\n"
		"\t  sbiw  %[len], 1             ; [2]\n"
		"\t  brne  0b                    ; [1/2] send the next byte, if not at the end of the buffer\n"
	: [c]"=&r"(c),
	  [i]"=&d"(i),
	  [buf]"+x"(buf),
	  [len]"+w"(len)
	: [port]"z"(port),
	  [maskSet]"r"(maskSet),
	  [maskClear]"r"(maskClear)
	: "memory");
}
*/
import "C"

// frame holds the bytes of the last WriteColors call, so a whole frame can be
// sent without gaps. A gap longer than a few microseconds latches the strip.
var (
	frame []byte
	one   [1]byte
)

// Send a single byte using the WS2812 protocol.
func (d Device) WriteByte(c byte) error {
	one[0] = c
	return d.writeBytes(one[:])
}

func (d Device) writeBytes(buf []byte) error {
	if machine.CPUFrequency() != 16e6 {
		return errUnknownClockSpeed
	}
	if len(buf) == 0 {
		return nil
	}
	port, maskSet := d.Pin.PortMaskSet()
	_, maskClear := d.Pin.PortMaskClear()
	mask := interrupt.Disable()
	C.ws2812_writeBytes16((*C.uint8_t)(unsafe.Pointer(&buf[0])), C.uint16_t(len(buf)), (*C.uint8_t)(unsafe.Pointer(port)), C.uint8_t(maskSet), C.uint8_t(maskClear))
	interrupt.Restore(mask)
	return nil
}

func (d Device) writePixels(buf []color.RGBA, brightness uint8, alpha bool) error {
	size := 3
	if alpha {
		size = 4
	}
	n := len(buf) * size
	if cap(frame) < n {
		frame = make([]byte, n)
	}
	frame = frame[:n]
	i := 0
	for _, color := range buf {
		r, g, b := applyBrightness(color, brightness)
		frame[i], frame[i+1], frame[i+2] = g, r, b
		if alpha {
			frame[i+3] = color.A
		}
		i += size
	}
	return d.writeBytes(frame)
}
