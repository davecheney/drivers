// This example brings up the ESP32-P4 MIPI DSI host for the Raspberry Pi
// 7-inch touchscreen v1 (TC358762 bridge, ATTINY firmware ID 0xC3). It shows
// the DSI host colour bars for 5 seconds, then a solid red from the bridge.
//
// DSI host sequence from ESP-IDF esp_lcd/dsi and esp_hal_lcd mipi_dsi_hal.c
// (chip revision 3). Panel power from Linux rpi-panel-attiny-regulator.c and
// bridge setup from Linux tc358762.c. The lane rate must be 24 times the DPI
// clock, the escape clock must be slow, and the bridge must leave reset only
// after the DSI lanes are in LP-11.
package main

import (
	"machine"
	"runtime/volatile"
	"time"
	"unsafe"
)

const (
	clkrst = 0x500e6000
	host   = 0x500a0000
	bridge = 0x500a0800

	attiny = 0x45
)

func reg(base, off uintptr) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Pointer(base + off))
}

func field(r *volatile.Register32, shift, width uint32, v uint32) {
	mask := uint32(1)<<width - 1
	r.Set(r.Get()&^(mask<<shift) | (v&mask)<<shift)
}

var bus = machine.I2C0

func main() {
	time.Sleep(time.Second)
	bus.Configure(machine.I2CConfig{SDA: machine.SDA_PIN, SCL: machine.SCL_PIN, Frequency: 100_000})

	// Panel power through the ATTINY, firmware ver 2 (ID 0xC3). Sequence from
	// Linux rpi-panel-attiny-regulator.c.
	println("attiny id", i2cRead(0x80), "port c before", i2cRead(0x83))
	i2cWrite(0x83, 0) // bridge, LCD and touch in reset
	time.Sleep(10 * time.Millisecond)
	i2cWrite(0x81, 1<<2) // PA_LCD_LR
	time.Sleep(10 * time.Millisecond)
	i2cWrite(0x82, 1<<7) // PB_LCD_MAIN
	time.Sleep(10 * time.Millisecond)
	i2cWrite(0x83, 1<<0) // PC_LED_EN, bridge still in reset
	time.Sleep(80 * time.Millisecond)
	println("port a", i2cRead(0x81), "port b", i2cRead(0x82), "port c", i2cRead(0x83))

	if err := machine.LDOConfigure(machine.LDO3, 2500); err != nil {
		println("ldo:", err.Error())
	}

	// Clocks and resets.
	field(reg(clkrst, 0x2c), 8, 1, 1)  // REF_20M_CLK_EN
	field(reg(clkrst, 0x28), 30, 1, 1) // REF_240M_CLK_EN
	field(reg(clkrst, 0x18), 12, 1, 1) // DSI_SYS_CLK_EN
	field(reg(clkrst, 0xc0), 26, 1, 1) // RST_EN_DSI_BRG
	field(reg(clkrst, 0xc0), 26, 1, 0)
	field(reg(clkrst, 0x38), 30, 2, 0) // DPHY_CLK_SRC_SEL = PLL_F20M
	field(reg(clkrst, 0x3c), 0, 1, 1)  // DPHY_CFG_CLK_EN
	field(reg(clkrst, 0x3c), 16, 3, 0) // DPHY_PLL_REFCLK_SRC_SEL = XTAL
	field(reg(clkrst, 0x3c), 19, 8, 0) // DPHY_PLL_REFCLK_DIV_NUM = /1
	field(reg(clkrst, 0x3c), 1, 1, 1)  // DPHY_PLL_REFCLK_EN
	field(reg(clkrst, 0x3c), 5, 2, 1)  // DPICLK_SRC_SEL = PLL_F240M
	field(reg(clkrst, 0x3c), 8, 8, 8)  // DPICLK_DIV_NUM = /9, 26.67 MHz
	field(reg(clkrst, 0x3c), 7, 1, 1)  // DPICLK_EN

	// Host and D-PHY.
	reg(host, 0xa4).Set(0) // PHY_IF_CFG, 1 lane
	reg(host, 0x04).Set(1) // PWR_UP
	reg(host, 0xa0).Set(1) // PHY_RSTZ: SHUTDOWNZ
	reg(host, 0xa0).Set(3) // RSTZ
	reg(host, 0xa0).Set(7) // ENABLECLK
	reg(host, 0xa0).Set(0xf)
	field(reg(bridge, 0x04), 1, 1, 1) // DSI_BRIG_RST
	field(reg(bridge, 0x04), 1, 1, 0)

	// D-PHY PLL for 640 Mbps from the 40 MHz XTAL: N = 4, M = 64. This is 24
	// times the 26.67 MHz DPI clock, so the TC358762 pixel clock matches.
	phyWrite(0x44, 0x08<<1)
	phyWrite(0x22, 0x80)
	phyWrite(0x19, 0x30)
	phyWrite(0x17, 4-1)
	phyWrite(0x18, (64-1)&0x1f)
	phyWrite(0x18, 0x80|((64-1)>>5)&0x0f)

	if !waitBits(reg(host, 0xb0), 1<<0) {
		println("phy lock timeout, status", reg(host, 0xb0).Get())
	}
	if !waitBits(reg(host, 0xb0), 1<<2|1<<4) {
		println("phy stopstate timeout, status", reg(host, 0xb0).Get())
	}
	println("phy status", reg(host, 0xb0).Get())

	reg(host, 0x34).Set(1)          // MODE_CFG: command mode
	reg(host, 0x94).Set(1)          // LPCLK_CTRL: continuous HS clock
	reg(host, 0x9c).Set(0x00320068) // PHY_TMR_CFG
	reg(host, 0x98).Set(0x002e0080) // PHY_TMR_LPCLK_CFG
	reg(host, 0x2c).Set(0x1d)       // PCKHDL_CFG: with BTA for reads
	reg(host, 0x08).Set(0x0810)     // CLKMGR_CFG: escape clock 640/8/16 = 5 MHz
	reg(host, 0x78).Set(0x7fff)     // TO_CNT_CFG
	reg(host, 0xf4).Set(6000)       // PHY_TMR_RD_CFG
	reg(host, 0xa4).Set(0x3f00)     // PHY_IF_CFG: stop wait, 1 lane
	reg(host, 0x68).Set(0x010f7f00) // CMD_MODE_CFG: all LP, no ack

	// Release the bridge reset only now, while the DSI lanes are in LP-11,
	// as Linux does with pre_enable_prev_first.
	i2cWrite(0x83, 1<<0|1<<2|1<<3) // RST_LCD_N and RST_BRIDGE_N high
	time.Sleep(8 * time.Millisecond)
	// TC358762 SYSPMCTRL (0x047c) = 0, through the ATTINY.
	i2cWrite(0x8d, 0x04)
	time.Sleep(8 * time.Millisecond)
	i2cWrite(0x8c, 0x7c)
	time.Sleep(8 * time.Millisecond)
	i2cWrite(0x90, 0x00)
	time.Sleep(8 * time.Millisecond)
	i2cWrite(0x91, 0x00)
	time.Sleep(100 * time.Millisecond)
	i2cWrite(0x83, 1<<0|1<<1|1<<2|1<<3) // RST_TP_N high
	println("bridge out of reset, port c", i2cRead(0x83), "phy status", reg(host, 0xb0).Get())

	// TC358762 setup.
	tc(0x0210, 0x3) // DSI_LANEENABLE: clock and D0
	tc(0x0164, 0x05)
	tc(0x0168, 0x05)
	tc(0x0144, 0x00)
	tc(0x0148, 0x00)
	tc(0x0114, 0x03)
	tc(0x0450, 0x00)
	tc(0x0420, 0x00100150)
	tc(0x0464, 0x040f)
	time.Sleep(100 * time.Millisecond)
	tc(0x0104, 0x01)
	tc(0x0204, 0x01)
	time.Sleep(100 * time.Millisecond)
	println("cmd status", reg(host, 0x74).Get(), "int st0", reg(host, 0xbc).Get(), "int st1", reg(host, 0xc0).Get())
	id, ok := tcRead(0x04a0)
	println("tc358762 idreg", id, "ok", ok, "int st0", reg(host, 0xbc).Get(), "int st1", reg(host, 0xc0).Get())
	sys, ok := tcRead(0x0460)
	println("tc358762 sysstat", sys, "ok", ok)

	// Video timing: 800x480, HFP 23, HSYNC 2, HBP 46, VFP 7, VSYNC 2, VBP 21.
	// The lane byte clock is exactly 3 times the DPI clock.
	reg(host, 0x0c).Set(0)       // DPI_VCID
	reg(host, 0x10).Set(5)       // DPI_COLOR_CODING: RGB888
	reg(host, 0x14).Set(0)       // DPI_CFG_POL
	reg(host, 0x38).Set(vidMode) // VID_MODE_CFG
	reg(host, 0x3c).Set(800)     // VID_PKT_SIZE
	reg(host, 0x40).Set(0)
	reg(host, 0x44).Set(0)
	reg(host, 0x48).Set(2 * 3)   // VID_HSA_TIME
	reg(host, 0x4c).Set(46 * 3)  // VID_HBP_TIME
	reg(host, 0x50).Set(872 * 3) // VID_HLINE_TIME: even line length
	reg(host, 0x54).Set(2)
	reg(host, 0x58).Set(21)
	reg(host, 0x5c).Set(7)
	reg(host, 0x60).Set(480)

	// Bridge DPI timing, DPI off.
	reg(bridge, 0x38).Set(872 | 800<<16) // DPI_H_CFG0: HFP 24
	reg(bridge, 0x3c).Set(46 | 2<<16)    // DPI_H_CFG1
	reg(bridge, 0x30).Set(510 | 480<<16) // DPI_V_CFG0
	reg(bridge, 0x34).Set(21 | 2<<16)    // DPI_V_CFG1
	reg(bridge, 0x0c).Set(144000 | 1<<31)
	reg(bridge, 0x40).Set(510 << 4) // DPI_MISC_CONFIG
	reg(bridge, 0x18).Set(0)        // PIXEL_TYPE: RGB888 in and out
	reg(bridge, 0x88).Set(1 << 4)   // DMA_FLOW_CTRL: DMAC
	field(reg(bridge, 0x6c), 28, 1, 0)
	reg(bridge, 0x08).Set(256)
	reg(bridge, 0x8c).Set(768)
	reg(bridge, 0x28).Set(0x3ff << 20) // DPI_RSV_DPI_DATA: red, if the layout is 10 bit R, G, B
	field(reg(bridge, 0x04), 0, 1, 1)  // DSI_EN
	reg(bridge, 0x44).Set(1)

	// Backlight on.
	i2cWrite(0x86, 0xff)
	i2cWrite(0x81, 1<<2)

	// Colour bars from the host pattern generator.
	reg(host, 0x38).Set(vidMode | 1<<16)
	reg(host, 0x34).Set(0) // video mode
	println("colour bars")
	time.Sleep(5 * time.Second)

	// Solid colour: the bridge sends DPI_RSV_DPI_DATA when it has no pixels.
	reg(host, 0x38).Set(vidMode)
	field(reg(bridge, 0x40), 0, 1, 1) // DPI_EN
	reg(bridge, 0x44).Set(1)
	println("solid colour")
	for {
		time.Sleep(2 * time.Second)
		println("bridge int raw", reg(bridge, 0x58).Get(), "host int st0", reg(host, 0xbc).Get(), "st1", reg(host, 0xc0).Get())
	}
}

// VID_MODE_CFG: non-burst with sync pulses, HS blanking packets, LP only for
// commands, no frame ack.
const vidMode = 1 << 15

// phyWrite writes a D-PHY test code register.
func phyWrite(addr, val uint8) {
	c0, c1 := reg(host, 0xb4), reg(host, 0xb8)
	c0.Set(0)
	c1.Set(1<<16 | uint32(addr))
	c0.Set(2)
	c0.Set(0)
	c1.Set(uint32(val))
	c0.Set(2)
	c0.Set(0)
}

func waitBits(r *volatile.Register32, bits uint32) bool {
	for i := 0; i < 100000; i++ {
		if r.Get()&bits == bits {
			return true
		}
	}
	return false
}

// tc writes a TC358762 register with a DSI generic long write.
func tc(addr uint16, val uint32) {
	msg := [6]byte{byte(addr), byte(addr >> 8), byte(val), byte(val >> 8), byte(val >> 16), byte(val >> 24)}
	genericLongWrite(msg[:])
}

func genericLongWrite(data []byte) {
	status := reg(host, 0x74)
	for i := 0; i < len(data); i += 4 {
		var w uint32
		for j := 0; j < 4 && i+j < len(data); j++ {
			w |= uint32(data[i+j]) << (8 * j)
		}
		for status.Get()&(1<<3) != 0 { // GEN_PLD_W_FULL
		}
		reg(host, 0x70).Set(w)
	}
	for status.Get()&(1<<1) != 0 { // GEN_CMD_FULL
	}
	n := uint32(len(data))
	reg(host, 0x6c).Set((n>>8)<<16 | (n&0xff)<<8 | 0x29)
	// Wait until the command and payload FIFOs are empty.
	for i := 0; i < 100000 && status.Get()&(1<<0|1<<2) != 1<<0|1<<2; i++ {
	}
}

// tcRead reads a TC358762 register with a DSI generic read (2 parameters),
// after it sets the maximum return packet size to 4.
func tcRead(addr uint16) (uint32, bool) {
	status := reg(host, 0x74)
	hdr := reg(host, 0x6c)
	for status.Get()&(1<<1) != 0 {
	}
	hdr.Set(4<<8 | 0x37) // set maximum return packet size
	for status.Get()&(1<<1) != 0 {
	}
	hdr.Set(uint32(addr>>8)<<16 | uint32(addr&0xff)<<8 | 0x24)
	// Wait for the read to finish, then for data.
	for i := 0; i < 200000; i++ {
		s := status.Get()
		if s&(1<<6) == 0 && s&(1<<4) == 0 { // not busy, read FIFO not empty
			return reg(host, 0x70).Get(), true
		}
	}
	return status.Get(), false
}

func i2cRead(r uint8) uint8 {
	var v [1]byte
	bus.Tx(attiny, []byte{r}, v[:])
	return v[0]
}

func i2cWrite(r, v uint8) {
	bus.Tx(attiny, []byte{r, v}, nil)
}
