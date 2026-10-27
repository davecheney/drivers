package epd2in7

import (
	"time"
)

// FastConfig sets the drive for fast updates. Zero values use the defaults.
type FastConfig struct {
	BlackPulses uint8 // scans that drive a pixel which changes to black
	WhitePulses uint8 // scans that drive a pixel which changes to white
	PulseFrames uint8 // frames in each scan, about 3.2 ms each
	TopUpFrames uint8 // frames in a top up pulse
}

// Fast drives only the pixels that change, a fixed number of short pulses
// each. Draw with SetPixel on the Device, then call Scan until it returns
// false. Call Wait before other commands to the Device.
//
// The old and new image planes select one of four LUTs for each pixel. Fast
// loads hold into WW, a short drive into WB and BW, and a white top up into BB.
// Source: EPD_2IN7_4GrayDisplay and the LUTs in the vendor EPD_2in7.c driver.
//
// Short custom waveforms can cause image retention. Call Clean from time to time.
type Fast struct {
	d        *Device
	cfg      FastConfig
	prev     []uint8 // target image of the last scan
	count    []uint8 // remaining pulses for each pixel
	oldPlane []uint8
	newPlane []uint8
	scans    uint8 // refreshes, selects the top up pixels
	busy     bool  // a refresh started by Scan can still be in progress
}

// NewFast returns a fast update driver for a configured Device. Call Clean
// before the first Scan.
func NewFast(d *Device, cfg FastConfig) Fast {
	if cfg.BlackPulses == 0 {
		cfg.BlackPulses = 4
	}
	if cfg.WhitePulses == 0 {
		cfg.WhitePulses = 4
	}
	if cfg.PulseFrames == 0 {
		cfg.PulseFrames = 16
	}
	if cfg.TopUpFrames == 0 {
		cfg.TopUpFrames = 8
	}
	return Fast{
		d:        d,
		cfg:      cfg,
		prev:     make([]uint8, d.bufferLength),
		count:    make([]uint8, d.bufferLength*8),
		oldPlane: make([]uint8, d.bufferLength),
		newPlane: make([]uint8, d.bufferLength),
	}
}

// Clean shows the buffer with the vendor full refresh waveform, then loads
// the fast LUTs. This removes ghosting and gives a known start for Scan.
func (f *Fast) Clean() {
	f.Wait()
	d := f.d
	d.SendCommand(PANEL_SETTING)
	d.SendData(panelSettingKWBfKwrAf)
	d.SendCommand(PLL_CONTROL)
	d.SendData(pllControl100Hz)
	d.SetLUT()
	for i := range f.oldPlane {
		f.oldPlane[i] = 0xFF
	}
	d.sendBytes(DATA_START_TRANSMISSION_1, f.oldPlane)
	d.sendBytes(DATA_START_TRANSMISSION_2, d.buffer)
	d.SendCommand(DISPLAY_REFRESH)
	time.Sleep(100 * time.Millisecond)
	d.WaitUntilIdle()

	copy(f.prev, d.buffer)
	for i := range f.count {
		f.count[i] = 0
	}
	f.setLUT()
}

// Scan sends one pulse to each pixel that has not reached its target. It
// returns false, and does not refresh, when no pixel needs drive. Scan
// returns when the refresh starts, so the next frame can be drawn during it.
//
// Each refresh moves held white pixels a little toward gray. So each scan
// also gives a short white pulse to one scattered pixel in each byte, if it
// is white and at its target. Each pixel gets a top up every 8 scans.
func (f *Fast) Scan() bool {
	stride := int(f.d.logicalWidth) / 8
	phase := 5 * int(f.scans)
	active := false
	for i, t := range f.d.buffer {
		n := f.count[i*8 : i*8+8]
		if c := t ^ f.prev[i]; c != 0 {
			for b := range n {
				m := uint8(0x80) >> b
				if c&m == 0 {
					continue
				}
				if t&m == 0 {
					n[b] = f.cfg.BlackPulses
				} else {
					n[b] = f.cfg.WhitePulses
				}
			}
			f.prev[i] = t
		}
		var drive uint8
		for b := range n {
			if n[b] != 0 {
				n[b]--
				drive |= uint8(0x80) >> b
			}
		}
		if drive != 0 {
			active = true
		}
		topUp := t &^ drive & (0x80 >> ((phase + 3*(i/stride) + 7*(i%stride)) & 7))
		// Hold is old 1 new 1, black is old 1 new 0, white is old 0 new 1,
		// top up is old 0 new 0.
		f.oldPlane[i] = ^(drive&t | topUp)
		f.newPlane[i] = ^(drive&^t | topUp)
	}
	if !active {
		return false
	}
	f.scans++

	f.Wait()
	d := f.d
	d.sendBytes(DATA_START_TRANSMISSION_1, f.oldPlane)
	d.sendBytes(DATA_START_TRANSMISSION_2, f.newPlane)
	d.SendCommand(DISPLAY_REFRESH)
	f.busy = true
	return true
}

// Wait waits until the refresh started by Scan is complete.
func (f *Fast) Wait() {
	if !f.busy {
		return
	}
	for {
		f.d.SendCommand(GET_STATUS)
		if f.d.busy.Get() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	f.busy = false
}

// setLUT loads a single group LUT for each pixel state. Level 01 is VDH and
// drives black, level 10 is VDL and drives white, as the vendor LUTs end.
func (f *Fast) setLUT() {
	// KW mode, where the old and new planes select the LUT for each pixel.
	// Source: EPD_2IN7_Init_4Gray in the vendor EPD_2in7.c driver.
	f.d.SendCommand(PANEL_SETTING)
	f.d.SendData(panelSettingKW)
	// Each refresh has about 8 frames of overhead, so use the shortest frame.
	// 0x19 measured 3.2 ms for each frame on a 2.7in V1 panel.
	f.d.SendCommand(PLL_CONTROL)
	f.d.SendData(pllControlFast)
	group := func(level, frames uint8) []uint8 {
		lut := make([]uint8, 42)
		lut[0] = level
		lut[1] = frames
		lut[5] = 1
		return lut
	}
	frames := f.cfg.PulseFrames
	hold := group(0x00, frames)
	f.d.sendBytes(LUT_FOR_VCOM, append([]uint8{0x00, 0x00}, hold...))
	f.d.sendBytes(LUT_WHITE_TO_WHITE, hold)
	f.d.sendBytes(LUT_BLACK_TO_WHITE, group(0x80, frames))
	f.d.sendBytes(LUT_WHITE_TO_BLACK, group(0x40, frames))
	f.d.sendBytes(LUT_BLACK_TO_BLACK, group(0x80, f.cfg.TopUpFrames))
}

// sendBytes sends a command and its data. The controller drops data sent in
// one CS low transfer, so CS goes high after each byte as in EPD_2in7_SendData.
func (d *Device) sendBytes(command uint8, data []uint8) {
	d.SendCommand(command)
	d.dc.High()
	for _, b := range data {
		d.cs.Low()
		d.bus.Transfer(b)
		d.cs.High()
	}
}
