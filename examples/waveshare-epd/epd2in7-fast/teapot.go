package main

import (
	"image/color"

	"tinygo.org/x/drivers/waveshare-epd/epd2in7"
)

const (
	angleSteps = 1024 // len(sinTable)
	fracBits   = 14

	cameraDistance = 10 * 2048 // model units, same x2048 scale as the mesh
	focalLength    = 320       // pixels

	yawPeriodUS   = 24_000_000
	pitchPeriodUS = 36_000_000
	rollPeriodUS  = 52_000_000
)

type point struct{ x, y int16 }

// orientation is a Q14 rotation matrix.
type orientation [3][3]int32

func axisAngle(us, periodUS uint64) uint32 {
	return uint32(us % periodUS * angleSteps / periodUS)
}

func sinCos(angle uint32) (s, c int32) {
	return sinTable[angle%angleSteps], sinTable[(angle+angleSteps/4)%angleSteps]
}

func q(a, b int32) int32 { return a * b >> fracBits }

// rotationAt returns yaw (about y) * pitch (about x) * roll (about z) at time
// us, with pitch reversed.
func rotationAt(us uint64) orientation {
	sy, cy := sinCos(axisAngle(us, yawPeriodUS))
	sp, cp := sinCos((angleSteps - axisAngle(us, pitchPeriodUS)) % angleSteps)
	sr, cr := sinCos(axisAngle(us, rollPeriodUS))
	return orientation{
		{q(cy, cr) + q(q(sy, sp), sr), q(q(sy, sp), cr) - q(cy, sr), q(sy, cp)},
		{q(cp, sr), q(cp, cr), -sp},
		{q(q(cy, sp), sr) - q(sy, cr), q(sy, sr) + q(q(cy, sp), cr), q(cy, cp)},
	}
}

// projectVertex rotates v by r and applies a perspective projection centered
// on cx, cy.
func projectVertex(v *[3]int16, r *orientation, cx, cy int32) point {
	x, y, z := int32(v[0]), int32(v[1]), int32(v[2])
	x1 := (r[0][0]*x + r[0][1]*y + r[0][2]*z) >> fracBits
	y1 := (r[1][0]*x + r[1][1]*y + r[1][2]*z) >> fracBits
	z1 := (r[2][0]*x + r[2][1]*y + r[2][2]*z) >> fracBits

	den := max(z1+cameraDistance, cameraDistance/4)
	return point{
		x: int16(cx + x1*focalLength/den),
		y: int16(cy - y1*focalLength/den),
	}
}

// drawLine draws a Bresenham line. SetPixel clips pixels outside the display.
func drawLine(d *epd2in7.Device, a, b point, c color.RGBA) {
	x0, y0, x1, y1 := a.x, a.y, b.x, b.y
	dx, sx := x1-x0, int16(1)
	if dx < 0 {
		dx, sx = -dx, -1
	}
	dy, sy := y1-y0, int16(1)
	if dy < 0 {
		dy, sy = -dy, -1
	}
	err := dx - dy
	for {
		d.SetPixel(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

var pts [len(teapotVerts)]point

// drawTeapot draws the wireframe at time us into the display buffer.
func drawTeapot(d *epd2in7.Device, us uint64) {
	w, h := d.Size()
	r := rotationAt(us)
	for i := range teapotVerts {
		pts[i] = projectVertex(&teapotVerts[i], &r, int32(w)/2, int32(h)/2)
	}
	for _, e := range teapotEdges {
		drawLine(d, pts[e[0]], pts[e[1]], black)
	}
}
