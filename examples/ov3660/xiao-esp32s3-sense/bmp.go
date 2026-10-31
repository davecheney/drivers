package main

import (
	"encoding/binary"
	"io"
)

const bmpHeaderSize = 14 + 40 + 12

// writeBMP writes an RGB565 big endian frame as a 16-bit BMP, rotated 90
// degrees counter-clockwise to match the display. rowBuf must hold a whole
// number of output rows.
func writeBMP(w io.Writer, frame []byte, srcW, srcH int, rowBuf []byte) error {
	dstW, dstH := srcH, srcW
	rowBytes := dstW * 2

	var hdr [bmpHeaderSize]byte
	le := binary.LittleEndian
	hdr[0], hdr[1] = 'B', 'M'
	le.PutUint32(hdr[2:], uint32(bmpHeaderSize+rowBytes*dstH))
	le.PutUint32(hdr[10:], bmpHeaderSize)
	le.PutUint32(hdr[14:], 40)
	le.PutUint32(hdr[18:], uint32(dstW))
	le.PutUint32(hdr[22:], uint32(dstH))
	le.PutUint16(hdr[26:], 1)
	le.PutUint16(hdr[28:], 16)
	le.PutUint32(hdr[30:], 3) // BI_BITFIELDS
	le.PutUint32(hdr[34:], uint32(rowBytes*dstH))
	le.PutUint32(hdr[54:], 0xf800)
	le.PutUint32(hdr[58:], 0x07e0)
	le.PutUint32(hdr[62:], 0x001f)
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}

	rows := len(rowBuf) / rowBytes
	n := 0
	// BMP rows go from bottom to top. Output row y is source column srcW-1-y.
	for y := dstH - 1; y >= 0; y-- {
		col := srcW - 1 - y
		out := rowBuf[n*rowBytes : (n+1)*rowBytes]
		for x := 0; x < dstW; x++ {
			i := (x*srcW + col) * 2
			out[x*2] = frame[i+1]
			out[x*2+1] = frame[i]
		}
		n++
		if n == rows || y == 0 {
			if _, err := w.Write(rowBuf[:n*rowBytes]); err != nil {
				return err
			}
			n = 0
		}
	}
	return nil
}
