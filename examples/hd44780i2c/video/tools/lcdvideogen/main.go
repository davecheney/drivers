// Command lcdvideogen converts a video into frames for a 16x4 HD44780 LCD
// that uses at most 8 custom characters per frame.
//
// The output starts with a 20 byte header. This is the magic "L16D" followed by
// the frame rate numerator, denominator, frame count and data length as
// little-endian uint32 values. Each frame is a delta against the previous one:
//
//	glyph count, then for each glyph: slot, 8 row bytes
//	run count, then for each run: first cell, length, length character codes
//
// Cells are numbered 0 to 63 in row order, and a run never crosses a row.
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"math/bits"
	"os"
	"os/exec"
	"sort"
)

const (
	cols, rows   = 16, 4
	cellW, cellH = 5, 8
	// The panel has a gap of about one dot between cells.
	canvasW = cols*(cellW+1) - 1
	canvasH = rows*(cellH+1) - 1

	charSpace = 0x20
	charFull  = 0xff
	maxGlyphs = 8
)

type tile [cellH]byte

var (
	blankTile tile
	fullTile  = tile{0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f, 0x1f}
)

func dist(a, b tile) int {
	d := 0
	for i := range a {
		d += bits.OnesCount8(a[i] ^ b[i])
	}
	return d
}

func main() {
	input := flag.String("input", "", "source video")
	output := flag.String("output", "video.bin", "output file")
	end := flag.String("end", "", "stop at this source timecode, such as 3:38")
	fps := flag.Int("fps", 10, "output frame rate")
	fit := flag.String("fit", "pad", "crop to fill the display, or pad to show the whole frame")
	trim := flag.Float64("trim", 2, "percent to remove from each edge of the source frame")
	threshold := flag.Int("threshold", 128, "gray levels below this value are dark dots")
	invert := flag.Bool("invert", false, "make light areas dark dots")
	flag.Parse()
	if *input == "" {
		log.Fatal("lcdvideogen: -input is required")
	}
	if *trim < 0 || *trim >= 50 {
		log.Fatal("lcdvideogen: need 0 <= -trim < 50")
	}

	keep := 1 - 2**trim/100
	filter := fmt.Sprintf("fps=%d,crop=iw*%g:ih*%g,", *fps, keep, keep)
	pad := "white"
	if *invert {
		pad = "black"
	}
	switch *fit {
	case "crop":
		filter += fmt.Sprintf("scale=%d:%d:flags=area:force_original_aspect_ratio=increase,crop=%d:%d", canvasW, canvasH, canvasW, canvasH)
	case "pad":
		filter += fmt.Sprintf("scale=%d:%d:flags=area:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:%s", canvasW, canvasH, canvasW, canvasH, pad)
	default:
		log.Fatalf("lcdvideogen: unknown -fit %q", *fit)
	}
	args := []string{"-loglevel", "error", "-i", *input}
	if *end != "" {
		args = append(args, "-to", *end)
	}
	args = append(args, "-vf", filter, "-f", "rawvideo", "-pix_fmt", "gray", "-")
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	f, err := os.Create(*output)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write(make([]byte, 20)); err != nil {
		log.Fatal(err)
	}
	w := bufio.NewWriter(f)

	r := bufio.NewReader(stdout)
	gray := make([]byte, canvasW*canvasH)
	var st state
	for i := range st.screen {
		st.screen[i] = charSpace
	}
	frames, length, merged := 0, 0, 0
	for {
		if _, err := io.ReadFull(r, gray); err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}
		tiles := cut(gray, *threshold, *invert)
		data, m := st.encode(tiles)
		if _, err := w.Write(data); err != nil {
			log.Fatal(err)
		}
		frames++
		length += len(data)
		merged += m
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		log.Fatal(err)
	}
	header := []byte("L16D")
	header = binary.LittleEndian.AppendUint32(header, uint32(*fps))
	header = binary.LittleEndian.AppendUint32(header, 1)
	header = binary.LittleEndian.AppendUint32(header, uint32(frames))
	header = binary.LittleEndian.AppendUint32(header, uint32(length))
	if _, err := f.WriteAt(header, 0); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d frames at %d fps, %d bytes, %.1f bytes per frame, %d frames needed merging\n",
		frames, *fps, 20+length, float64(length)/float64(max(frames, 1)), merged)
}

// cut splits the canvas into cells and drops the gap dots between them.
func cut(gray []byte, threshold int, invert bool) [cols * rows]tile {
	var tiles [cols * rows]tile
	for cy := 0; cy < rows; cy++ {
		for cx := 0; cx < cols; cx++ {
			var t tile
			for y := 0; y < cellH; y++ {
				for x := 0; x < cellW; x++ {
					v := int(gray[(cy*(cellH+1)+y)*canvasW+cx*(cellW+1)+x])
					on := v < threshold
					if invert {
						on = !on
					}
					if on {
						t[y] |= 0x10 >> x
					}
				}
			}
			tiles[cy*cols+cx] = t
		}
	}
	return tiles
}

type state struct {
	screen    [cols * rows]byte
	slots     [maxGlyphs]tile
	slotValid [maxGlyphs]bool
}

type cluster struct {
	rep     tile
	fixed   bool
	char    byte
	members []int // indices into uniq
	weight  int
}

// encode picks the glyphs for one frame and returns its delta.
func (st *state) encode(tiles [cols * rows]tile) ([]byte, int) {
	counts := map[tile]int{}
	for _, t := range tiles {
		counts[t]++
	}
	var uniq []tile
	for t := range counts {
		uniq = append(uniq, t)
	}
	sort.Slice(uniq, func(i, j int) bool { return counts[uniq[i]] > counts[uniq[j]] })

	clusters := []*cluster{
		{rep: blankTile, fixed: true, char: charSpace},
		{rep: fullTile, fixed: true, char: charFull},
	}
	for i, t := range uniq {
		switch t {
		case blankTile:
			clusters[0].members = append(clusters[0].members, i)
			clusters[0].weight += counts[t]
		case fullTile:
			clusters[1].members = append(clusters[1].members, i)
			clusters[1].weight += counts[t]
		default:
			clusters = append(clusters, &cluster{rep: t, members: []int{i}, weight: counts[t]})
		}
	}

	merged := 0
	for len(clusters)-2 > maxGlyphs {
		merged = 1
		bi, bj, best := -1, -1, 1<<30
		for i := 0; i < len(clusters); i++ {
			for j := max(i+1, 2); j < len(clusters); j++ {
				cost := dist(clusters[i].rep, clusters[j].rep) * min(clusters[i].weight, clusters[j].weight)
				if clusters[i].fixed {
					cost = dist(clusters[i].rep, clusters[j].rep) * clusters[j].weight
				}
				if cost < best {
					bi, bj, best = i, j, cost
				}
			}
		}
		a, b := clusters[bi], clusters[bj]
		a.members = append(a.members, b.members...)
		a.weight += b.weight
		if !a.fixed {
			a.rep = majority(a.members, uniq, counts)
		}
		clusters = append(clusters[:bj], clusters[bj+1:]...)
	}

	// Keep slots whose glyph is still used, and fill the others.
	var used [maxGlyphs]bool
	var out []byte
	var glyphWrites []byte
	nWrites := 0
	free := clusters[2:]
	var pending []*cluster
	for _, c := range free {
		found := false
		for s := 0; s < maxGlyphs; s++ {
			if st.slotValid[s] && !used[s] && st.slots[s] == c.rep {
				used[s] = true
				c.char = byte(s)
				found = true
				break
			}
		}
		if !found {
			pending = append(pending, c)
		}
	}
	for _, c := range pending {
		for s := 0; s < maxGlyphs; s++ {
			if !used[s] {
				used[s] = true
				c.char = byte(s)
				st.slots[s] = c.rep
				st.slotValid[s] = true
				glyphWrites = append(glyphWrites, byte(s))
				glyphWrites = append(glyphWrites, c.rep[:]...)
				nWrites++
				break
			}
		}
	}
	out = append(out, byte(nWrites))
	out = append(out, glyphWrites...)

	charOf := map[tile]byte{}
	for _, c := range clusters {
		for _, m := range c.members {
			charOf[uniq[m]] = c.char
		}
	}
	var next [cols * rows]byte
	for i, t := range tiles {
		next[i] = charOf[t]
	}
	// A cell with the same code shows the new glyph after a CGRAM write.
	changed := func(i int) bool { return next[i] != st.screen[i] }

	var runs []byte
	nRuns := 0
	for row := 0; row < rows; row++ {
		x := 0
		for x < cols {
			i := row*cols + x
			if !changed(i) {
				x++
				continue
			}
			start := x
			end := x + 1
			for end < cols {
				if changed(row*cols + end) {
					end++
				} else if end+1 < cols && changed(row*cols+end+1) {
					end += 2
				} else {
					break
				}
			}
			runs = append(runs, byte(row*cols+start), byte(end-start))
			runs = append(runs, next[row*cols+start:row*cols+end]...)
			nRuns++
			x = end
		}
	}
	out = append(out, byte(nRuns))
	out = append(out, runs...)
	st.screen = next
	return out, merged
}

// majority returns the dot by dot weighted vote of the member tiles.
func majority(members []int, uniq []tile, counts map[tile]int) tile {
	var votes [cellH][cellW]int
	total := 0
	for _, m := range members {
		t := uniq[m]
		n := counts[t]
		total += n
		for y := 0; y < cellH; y++ {
			for x := 0; x < cellW; x++ {
				if t[y]&(0x10>>x) != 0 {
					votes[y][x] += n
				}
			}
		}
	}
	var r tile
	for y := 0; y < cellH; y++ {
		for x := 0; x < cellW; x++ {
			if 2*votes[y][x] > total {
				r[y] |= 0x10 >> x
			}
		}
	}
	return r
}
