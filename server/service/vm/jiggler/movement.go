package jiggler

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"time"

	"NanoKVM-Server/common"
	"NanoKVM-Server/service/hid"
)

// The volatile appliance-side override permits watching curves with a viewer connected.
const curvePreviewFile = "/run/nanokvm-jiggler-curve-preview"

type movement struct {
	reports  [][]byte
	duration time.Duration
	start    time.Time
	index    int
}

func (m *movement) nextFrame() time.Time {
	return m.start.Add(m.duration * time.Duration(m.index) / time.Duration(len(m.reports)-1))
}

func absoluteReport(x, y int) []byte {
	r := []byte{hid.AbsoluteReportIDMouse, 0, 0, 0, 0, 0, 0}
	binary.LittleEndian.PutUint16(r[2:4], uint16(x))
	binary.LittleEndian.PutUint16(r[4:6], uint16(y))
	return r
}

func planMovement(mode string, curved bool) *movement {
	if !curved {
		reports := [][]byte{{0, 10, 10, 0}, {0, 246, 246, 0}}
		if mode == "absolute" {
			reports = [][]byte{absoluteReport(0x3f00, 0x3f00), absoluteReport(0x3fff, 0x3fff)}
		}
		return &movement{reports: reports, duration: 100 * time.Millisecond}
	}
	width, height := common.HDMIInputSize()
	x, y, w, h := hid.GetHid().EstimatedMousePosition(width, height)
	return curveMovement(mode, point{x, y}, w, h)
}

type point struct{ x, y float64 }

// Give the center a mild preference in the interior. In the outer
// quarter, always head inward so an edge cannot flatten the entire stroke.
func curveDirection(position, span float64) float64 {
	fraction := position / span
	if fraction <= 0.25 {
		return 1
	}
	if fraction >= 0.75 || rand.Float64() < fraction {
		return -1
	}
	return 1
}

func curveMovement(mode string, start point, width, height int) *movement {
	// One cubic, starting at (0,0). Vary how far the endpoint doubles back
	// relative to the outer control point to produce both arcs and hooks.
	shortSide := float64(min(width-1, height-1))
	scale := shortSide * (0.55 + 0.45*rand.Float64())
	if mode == "relative" {
		// Each per-frame HID delta must fit in a signed byte, including on very large displays.
		scale = min(scale, 3200)
	}
	p1 := point{scale * (0.35 + rand.Float64()*0.6), scale * rand.Float64() * 0.12}
	p2 := point{scale * (0.7 + rand.Float64()*0.75), scale * (0.12 + rand.Float64()*0.6)}
	end := point{scale * rand.Float64(), scale * (0.4 + rand.Float64()*0.55)}
	if rand.IntN(2) == 0 {
		p1, p2, end = point{p1.y, p1.x}, point{p2.y, p2.x}, point{end.y, end.x}
	}
	screenX, screenY := float64(width-1), float64(height-1)
	dx, dy := curveDirection(start.x, screenX), curveDirection(start.y, screenY)
	roomX, roomY := start.x, start.y
	if dx > 0 {
		roomX = screenX - start.x
	}
	if dy > 0 {
		roomY = screenY - start.y
	}
	// A cubic stays inside its control polygon. Leave a little space at
	// the far edge, applying one scale factor to preserve the shape.
	gain := min(1.0, 0.93*roomX/max(p1.x, p2.x, end.x), 0.93*roomY/max(p1.y, p2.y, end.y))
	orient := func(p point) point { return point{p.x * gain * dx, p.y * gain * dy} }
	p1, p2, end = orient(p1), orient(p2), orient(end)

	at := func(t float64) point {
		u := 1 - t
		a, b, c := 3*u*u*t, 3*u*t*t, t*t*t
		return point{a*p1.x + b*p2.x + c*end.x, a*p1.y + b*p2.y + c*end.y}
	}
	length := 0.0
	var prev point
	for i := 1; i <= 24; i++ {
		p := at(float64(i) / 24)
		length += math.Hypot(p.x-prev.x, p.y-prev.y)
		prev = p
	}
	seconds := 5 + min(2.0, 2*length/shortSide) + rand.Float64()

	m := &movement{duration: time.Duration(seconds * float64(time.Second))}
	steps := int(math.Ceil(seconds * 40))
	// Scale the sampled stroke after choosing its duration, so the larger
	// movement also travels faster. Fit the actual path into the available room.
	points := make([]point, steps+1)
	var reach point
	for i := range points {
		t := float64(i) / float64(steps)
		p := at(t * t * (3 - 2*t))
		points[i] = p
		reach.x = max(reach.x, math.Abs(p.x))
		reach.y = max(reach.y, math.Abs(p.y))
	}
	// Some strokes are edge-limited. This ceiling gives about 1.5x mean travel after fitting.
	fit := func(x, y float64) float64 { return min(1.88, 0.93*roomX/x, 0.93*roomY/y) }
	stretch := fit(reach.x, reach.y)
	turned := fit(reach.y, reach.x)
	turn := turned > stretch
	if turn {
		stretch = turned
	}

	lastX, lastY := 0, 0
	for _, p := range points {
		if turn {
			p = point{math.Copysign(p.y, dx), math.Copysign(p.x, dy)}
		}
		p.x *= stretch
		p.y *= stretch
		x, y := int(math.Round(p.x)), int(math.Round(p.y))
		if mode == "absolute" {
			m.reports = append(m.reports, absoluteReport(int(math.Round((start.x+p.x)*32767/screenX)), int(math.Round((start.y+p.y)*32767/screenY))))
		} else {
			m.reports = append(m.reports, []byte{0, byte(int8(x - lastX)), byte(int8(y - lastY)), 0})
		}
		lastX, lastY = x, y
	}
	return m
}
