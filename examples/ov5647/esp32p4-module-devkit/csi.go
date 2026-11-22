//go:build esp32p4

package main

import (
	"device/esp"
	"machine"
	"time"
	"unsafe"
)

// The DMA reads frames from the CSI bridge FIFO. ESP-IDF soc/esp32p4 reg_base.h
// MIPI_CSI_BRG_MEM_BASE.
const csiBridgeFIFO = 0x50104000

// hsfreqrange for the D-PHY lane rate. ESP-IDF esp_hal_cam/esp32p4/mipi_csi_periph.c.
func hsFreqSel(mbps uint32) uint32 {
	switch {
	case mbps < 400:
		return 0x15
	case mbps < 450:
		return 0x25
	}
	return 0x06
}

// startCSI powers the D-PHY and sets up the CSI host, bridge and ISP bypass
// for a 2 lane RAW8 stream. Sequence from ESP-IDF esp_driver_cam csi and
// esp_hal_cam mipi_csi_hal.c.
func startCSI(laneRate uint32, width, height int) error {
	if err := machine.LDOConfigure(machine.LDO3, 2500); err != nil {
		return err
	}

	// A DMA reset after the bridge setup breaks the bridge DMA handshake,
	// so the DMA comes first.
	startDMA()

	clk := esp.HP_SYS_CLKRST
	clk.SetREF_CLK_CTRL2_REG_REF_20M_CLK_EN(1)
	clk.SetREF_CLK_CTRL1_REG_REF_240M_CLK_EN(1)

	clk.SetSOC_CLK_CTRL1_REG_CSI_HOST_SYS_CLK_EN(1)
	clk.SetHP_RST_EN0_REG_RST_EN_CSI_HOST(1)
	clk.SetHP_RST_EN0_REG_RST_EN_CSI_HOST(0)

	clk.SetSOC_CLK_CTRL1_REG_CSI_BRG_SYS_CLK_EN(1)
	clk.SetHP_RST_EN0_REG_RST_EN_CSI_BRG(1)
	clk.SetHP_RST_EN0_REG_RST_EN_CSI_BRG(0)
	brg := esp.MIPI_CSI_BRIDGE
	brg.HOST_CTRL.SetBits(1)

	// PLL_F20M is the D-PHY config clock.
	clk.SetPERI_CLK_CTRL03_REG_MIPI_CSI_DPHY_CLK_SRC_SEL(0)
	clk.SetPERI_CLK_CTRL03_REG_MIPI_CSI_DPHY_CFG_CLK_EN(1)

	host := esp.MIPI_CSI_HOST
	host.PHY_SHUTDOWNZ.Set(0)
	host.DPHY_RSTZ.Set(0)
	host.CSI2_RESETN.Set(0)
	host.PHY_TEST_CTRL1.Set(1 << 16)
	host.PHY_TEST_CTRL0.Set(1)
	host.PHY_TEST_CTRL0.Set(0)
	phyWrite(0x44, uint8(hsFreqSel(laneRate)<<1))
	host.PHY_SHUTDOWNZ.Set(1)
	host.DPHY_RSTZ.Set(1)
	host.CSI2_RESETN.Set(1)
	host.N_LANES.Set(2 - 1)
	host.VC_EXTENSION.Set(1)

	// The sensor sends no line start or line end packets.
	frame := brg.FRAME_CFG.Get() &^ (1<<24 | 0xfff<<12 | 0xfff)
	brg.FRAME_CFG.Set(frame | uint32(width)<<12 | uint32(height))
	brg.BUF_FLOW_CTL.Set(brg.BUF_FLOW_CTL.Get()&^0x3fff | 960)
	brg.ENDIAN_MODE.ClearBits(1)
	brg.DMA_REQ_CFG.Set(brg.DMA_REQ_CFG.Get()&^0xfff | 512)
	brg.DATA_TYPE_CFG.Set(brg.DATA_TYPE_CFG.Get()&^(0x3f|0x3f<<8) | 0x12 | 0x2f<<8)

	// The CSI host output goes through the ISP input stage, so the ISP runs
	// in bypass. ESP-IDF esp_driver_isp isp_core.c.
	clk.SetPERI_CLK_CTRL25_REG_ISP_CLK_EN(1)
	clk.SetHP_RST_EN0_REG_RST_EN_ISP(1)
	clk.SetHP_RST_EN0_REG_RST_EN_ISP(0)
	clk.SetPERI_CLK_CTRL25_REG_ISP_CLK_SRC_SEL(1)
	clk.SetPERI_CLK_CTRL26_REG_ISP_CLK_DIV_NUM(0)
	isp := esp.ISP
	isp.CNTL.Set(0)
	isp.CLK_EN.SetBits(1)
	isp.CNTL.Set(1) // MIPI_DATA_EN, input from the CSI host, ISP_EN off.
	isp.FRAME_CFG.Set(uint32((width*8+31)/32-1)<<12 | uint32(height-1))
	return nil
}

// phyWrite writes a D-PHY test code register.
func phyWrite(addr, value uint8) {
	host := esp.MIPI_CSI_HOST
	host.PHY_TEST_CTRL0.Set(0)
	host.PHY_TEST_CTRL1.Set(1<<16 | uint32(addr))
	host.PHY_TEST_CTRL0.Set(2)
	host.PHY_TEST_CTRL0.Set(0)
	host.PHY_TEST_CTRL1.Set(uint32(value))
	host.PHY_TEST_CTRL0.Set(2)
	host.PHY_TEST_CTRL0.Set(0)
}

// startDMA sets up DW-GDMA channel 0 to move data from the bridge into memory,
// with the bridge as flow controller. ESP-IDF esp_driver_dma dw_gdma.c.
func startDMA() {
	esp.HP_SYS.SetPERI_MEM_CLK_FORCE_ON_GDMA_MEM_CLK_FORCE_ON(1)
	clk := esp.HP_SYS_CLKRST
	clk.SetSOC_CLK_CTRL0_REG_GDMA_CPU_CLK_EN(1)
	clk.SetSOC_CLK_CTRL1_REG_GDMA_SYS_CLK_EN(1)
	clk.SetHP_RST_EN0_REG_RST_EN_GDMA(1)
	clk.SetHP_RST_EN0_REG_RST_EN_GDMA(0)

	dma := esp.DMA
	dma.RESET0.Set(1)
	for dma.RESET0.Get()&1 != 0 {
	}
	dma.CFG0.Set(3) // DMAC_EN and INT_EN

	dma.CH1_CFG0.Set(0)
	const (
		flowP2MSrc   = 4
		handshakeCSI = 1
	)
	dma.CH1_CFG1.Set(flowP2MSrc | handshakeCSI<<7 | 1<<17 | 4<<23 | 4<<27)
	dma.CH1_SSTATAR0.Set(csiBridgeFIFO)
	dma.CH1_SSTATAR1.Set(0)
	dma.CH1_INTSTATUS_ENABLE0.Set(0xffffffff)
	dma.CH1_INTSIGNAL_ENABLE0.Set(0)
}

// capture reads one frame into buf, which must be 64 byte aligned and a
// multiple of 8 bytes long.
func capture(buf []byte, timeout time.Duration) bool {
	addr := uintptr(unsafe.Pointer(&buf[0]))
	// Drop cached lines so none are written back over the DMA data.
	invalidate(addr, uintptr(len(buf)))

	dma := esp.DMA
	dma.CH1_SAR0.Set(csiBridgeFIFO)
	dma.CH1_SAR1.Set(0)
	dma.CH1_DAR0.Set(uint32(addr))
	dma.CH1_DAR1.Set(0)
	dma.CH1_BLOCK_TS0.Set(uint32(len(buf)/8 - 1))
	// Source master 0 fixed, destination master 1 incrementing, 64 bit,
	// 512 item bursts. 16 beat AXI reads, single beat AXI writes, because
	// PSRAM stalls on burst writes from the DMA.
	dma.CH1_CTL0.Set(1<<2 | 1<<4 | 3<<8 | 3<<11 | 8<<14 | 8<<18)
	dma.CH1_CTL1.Set(1<<6 | 15<<7 | 1<<15 | 0<<16)
	dma.CH1_INTCLEAR0.Set(0xffffffff)
	dma.CHEN0.Set(0x101)

	brg := esp.MIPI_CSI_BRIDGE
	brg.INT_CLR.Set(0x3f)
	brg.CSI_EN.Set(1)
	deadline := time.Now().Add(timeout)
	done := false
	for time.Now().Before(deadline) {
		if dma.CH1_INTSTATUS0.Get()&2 != 0 {
			done = true
			break
		}
	}
	brg.CSI_EN.Set(0)
	if !done {
		println("capture timeout: bridge int", brg.INT_RAW.Get(), "buffer", brg.BUF_FLOW_CTL.Get()>>16, "dma int", dma.CH1_INTSTATUS0.Get())
		// Clearing the enable bit does not stop a busy channel, so abort it.
		dma.CHEN1.Set(0x101)
		for i := 0; i < 1000000 && dma.CHEN1.Get()&0x101 != 0; i++ {
		}
	}
	dma.CH1_INTCLEAR0.Set(0xffffffff)
	invalidate(addr, uintptr(len(buf)))
	return done
}

// invalidate drops cached lines for a buffer in SRAM or in PSRAM.
func invalidate(addr, size uintptr) {
	if addr >= 0x48000000 && addr < 0x4c000000 {
		machine.CacheInvalidate(addr, size)
		return
	}
	machine.CacheInvalidateL1(addr, size)
}
