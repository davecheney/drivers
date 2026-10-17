package uc8151

// Initialization and reset waveforms follow antirez's Badger driver at cc2a0de.
// https://github.com/antirez/uc8151_micropython/blob/cc2a0de/uc8151.py
var grayscaleInitialization = []struct {
	command byte
	data    []byte
	stage   string
}{
	{PSR, []byte{RESET_SOFT}, "software reset"},
	{PWR, []byte{0x03, 0x00, 0x26, 0x26, 0x03}, "power settings"},
	{BTST, []byte{0x17, 0x17, 0x17}, "booster"},
	{PON, nil, "power on"},
	{PSR, []byte{RES_128x296 | LUT_REG | FORMAT_BW | SHIFT_RIGHT | SCAN_UP | BOOSTER_ON | RESET_NONE}, "panel settings"},
	{PFS, []byte{FRAMES_4}, "power off timing"},
	{TSE, []byte{TEMP_INTERNAL | OFFSET_0}, "temperature sensor"},
	{TCON, []byte{0x22}, "timing"},
	{CDI, []byte{0xcc}, "VCOM interval"},
	{PLL, []byte{HZ_100}, "100Hz waveform clock"},
	{POF, nil, "power off"},
}

func (d *GrayscaleDevice) writeLUTs(white bool) error {
	var vcom [44]byte
	var tables [4]LUTType
	if white {
		vcom = [44]byte{
			0x00, 32, 0, 0, 0, 2,
			0x00, 16, 16, 0, 0, 2,
			0x00, 32, 0, 0, 0, 2,
		}
		for i := range tables {
			tables[i] = LUTType{
				0x40, 32, 0, 0, 0, 2,
				0x90, 16, 16, 0, 0, 1,
				0x80, 32, 0, 0, 0, 2,
			}
		}
	} else {
		vcom = [44]byte{0x00, 24, 0, 0, 0, 1}
		tables[0] = LUTType{0x55, d.lightFrames, 0, 0, 0, 1}
		tables[2] = LUTType{0x55, 32, 0, 0, 0, 1}
		tables[3] = LUTType{0x55, d.darkFrames, 0, 0, 0, 1}
	}
	if err := d.writeRegister(LUT_VCOM, vcom[:]); err != nil {
		return err
	}
	for i := range tables {
		if err := d.writeRegister(byte(LUT_WW+i), tables[i][:]); err != nil {
			return err
		}
	}
	return nil
}
