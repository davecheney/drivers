package main

import (
	"device/esp"
	"machine"
	"runtime/volatile"
	"time"
	"unsafe"
)

// LCD_CAM camera capture into internal SRAM with GDMA.
// See ESP32-S3 TRM, LCD_CAM and GDMA chapters, and
// https://github.com/espressif/esp32-camera/blob/master/target/esp32s3/ll_cam.c

// GPIO matrix signals from ESP-IDF soc/esp32s3/include/soc/gpio_sig_map.h.
const (
	camData0Idx   = 133
	camPclkIdx    = 149
	camClkIdx     = 149
	camHEnableIdx = 150
	camVSyncIdx   = 152

	dmaPeriLCDCAM = 5
	dmaChannel    = 4
)

type dmaDesc struct {
	dw0  uint32
	buf  uint32
	next uint32
}

type camera struct {
	descs     []dmaDesc
	descBytes int
}

type camPins struct {
	xclk, pclk, vsync, href machine.Pin
	data                    [8]machine.Pin
}

func outFunc(p machine.Pin) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.GPIO.FUNC0_OUT_SEL_CFG), uintptr(p)*4))
}

func inFunc(signal uint32) *volatile.Register32 {
	return (*volatile.Register32)(unsafe.Add(unsafe.Pointer(&esp.GPIO.FUNC0_IN_SEL_CFG), uintptr(signal)*4))
}

func routeIn(p machine.Pin, signal uint32) {
	p.Configure(machine.PinConfig{Mode: machine.PinInput})
	inFunc(signal).Set(esp.GPIO_FUNC_IN_SEL_CFG_SEL | uint32(p))
}

// startXCLK drives the sensor clock from the LCD_CAM 160 MHz source.
func startXCLK(pin machine.Pin, hz uint32) {
	esp.SYSTEM.SetPERIP_CLK_EN1_LCD_CAM_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN1_LCD_CAM_RST(1)
	esp.SYSTEM.SetPERIP_RST_EN1_LCD_CAM_RST(0)

	esp.LCD_CAM.CAM_CTRL.Set(0)
	esp.LCD_CAM.SetCAM_CTRL_CAM_CLKM_DIV_NUM(160_000_000 / hz)
	esp.LCD_CAM.SetCAM_CTRL_CAM_CLK_SEL(3)
	esp.LCD_CAM.SetCAM_CTRL_CAM_VSYNC_FILTER_THRES(4)
	esp.LCD_CAM.SetCAM_CTRL_CAM_VS_EOF_EN(1)
	esp.LCD_CAM.SetCAM_CTRL_CAM_UPDATE(1)

	pin.Configure(machine.PinConfig{Mode: machine.PinOutput})
	outFunc(pin).Set(camClkIdx)
}

// newCamera sets up capture of one frame into buf. descBytes must be a
// multiple of 4 and at most 4095, and must divide len(buf).
func newCamera(pins camPins, buf []byte, descBytes int) *camera {
	routeIn(pins.pclk, camPclkIdx)
	routeIn(pins.vsync, camVSyncIdx)
	routeIn(pins.href, camHEnableIdx)
	for i, p := range pins.data {
		routeIn(p, camData0Idx+uint32(i))
	}

	esp.LCD_CAM.CAM_CTRL1.Set(0)
	esp.LCD_CAM.SetCAM_CTRL1_CAM_VSYNC_FILTER_EN(1)
	esp.LCD_CAM.SetCAM_CTRL1_CAM_REC_DATA_BYTELEN(uint32(descBytes - 1))
	esp.LCD_CAM.CAM_RGB_YUV.Set(0)
	esp.LCD_CAM.SetCAM_CTRL_CAM_UPDATE(1)

	esp.SYSTEM.SetPERIP_CLK_EN1_DMA_CLK_EN(1)
	esp.SYSTEM.SetPERIP_RST_EN1_DMA_RST(0)
	esp.DMA.SetMISC_CONF_CLK_EN(1)

	c := &camera{
		descs:     make([]dmaDesc, len(buf)/descBytes),
		descBytes: descBytes,
	}
	for i := range c.descs {
		c.descs[i].buf = uint32(uintptr(unsafe.Pointer(&buf[i*descBytes])))
		if i+1 < len(c.descs) {
			c.descs[i].next = uint32(uintptr(unsafe.Pointer(&c.descs[i+1])))
		}
	}
	return c
}

// capture waits for the next frame and returns the number of bytes received.
func (c *camera) capture(timeout time.Duration) (int, bool) {
	esp.LCD_CAM.SetCAM_CTRL1_CAM_START(0)
	for i := range c.descs {
		volatile.StoreUint32(&c.descs[i].dw0, 1<<31|uint32(c.descBytes))
	}
	esp.LCD_CAM.SetCAM_CTRL1_CAM_RESET(1)
	esp.LCD_CAM.SetCAM_CTRL1_CAM_RESET(0)
	esp.LCD_CAM.SetCAM_CTRL1_CAM_AFIFO_RESET(1)
	esp.LCD_CAM.SetCAM_CTRL1_CAM_AFIFO_RESET(0)

	esp.DMA.IN_CONF0_CH4.Set(0)
	esp.DMA.SetIN_CONF0_CH4_IN_RST(1)
	esp.DMA.SetIN_CONF0_CH4_IN_RST(0)
	esp.DMA.SetIN_CONF0_CH4_INDSCR_BURST_EN(1)
	esp.DMA.SetIN_CONF0_CH4_IN_DATA_BURST_EN(1)
	esp.DMA.SetIN_CONF1_CH4_IN_CHECK_OWNER(0)
	esp.DMA.SetIN_PERI_SEL_CH4_PERI_IN_SEL(dmaPeriLCDCAM)
	esp.DMA.IN_INT_CLR_CH4.Set(0xffffffff)

	esp.DMA.SetIN_LINK_CH4_INLINK_ADDR(uint32(uintptr(unsafe.Pointer(&c.descs[0]))) & 0xfffff)
	esp.DMA.SetIN_LINK_CH4_INLINK_START(1)
	esp.LCD_CAM.SetCAM_CTRL_CAM_UPDATE(1)
	esp.LCD_CAM.SetCAM_CTRL1_CAM_START(1)

	ok := false
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		raw := esp.DMA.IN_INT_RAW_CH4.Get()
		if raw&(esp.DMA_IN_INT_RAW_CH_IN_SUC_EOF|esp.DMA_IN_INT_RAW_CH_IN_ERR_EOF|
			esp.DMA_IN_INT_RAW_CH_IN_DSCR_ERR|esp.DMA_IN_INT_RAW_CH_IN_DSCR_EMPTY) != 0 {
			ok = raw&esp.DMA_IN_INT_RAW_CH_IN_SUC_EOF != 0
			break
		}
	}
	esp.LCD_CAM.SetCAM_CTRL1_CAM_START(0)

	n := 0
	for i := range c.descs {
		w := volatile.LoadUint32(&c.descs[i].dw0)
		n += int(w>>12) & 0xfff
		if w&(1<<30) != 0 {
			break
		}
	}
	return n, ok
}
