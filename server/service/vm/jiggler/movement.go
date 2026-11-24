package jiggler

import (
	"encoding/binary"
	"math"
	"math/rand/v2"
	"time"

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
	x, y := 16384, 16384
	if mode == "absolute" {
		if lastX, lastY, ok := hid.GetHid().AbsoluteMousePosition(); ok {
			x, y = int(lastX), int(lastY)
		}
	}
	return curveMovement(mode, x, y)
}

type point struct{ x, y float64 }

func curveMovement(mode string, startX, startY int) *movement {
	// One cubic, starting at (0,0). Vary how far the endpoint doubles back
	// relative to the outer control point to produce both arcs and hooks.
	scale := 140 + rand.Float64()*400
	p1 := point{scale * (0.35 + rand.Float64()*0.6), scale * rand.Float64() * 0.12}
	p2 := point{scale * (0.7 + rand.Float64()*0.75), scale * (0.12 + rand.Float64()*0.6)}
	end := point{scale * rand.Float64(), scale * (0.4 + rand.Float64()*0.55)}
	if mode == "absolute" {
		// Aim into the larger free quadrant. Keeping every control point
		// within the HID coordinate range keeps the whole curve on screen.
		if rand.IntN(2) == 0 {
			p1, p2, end = point{p1.y, p1.x}, point{p2.y, p2.x}, point{end.y, end.x}
		}
		roomX, roomY := max(startX, 32767-startX), max(startY, 32767-startY)
		gain := min(24.0, float64(roomX)/max(p1.x, p2.x, end.x), float64(roomY)/max(p1.y, p2.y, end.y))
		dx, dy := gain, gain
		if startX >= 16384 {
			dx = -dx
		}
		if startY >= 16384 {
			dy = -dy
		}
		orient := func(p point) point { return point{p.x * dx, p.y * dy} }
		p1, p2, end = orient(p1), orient(p2), orient(end)
	} else {
		sin, cos := math.Sincos(rand.Float64() * 2 * math.Pi)
		orient := func(p point) point { return point{p.x*cos - p.y*sin, p.x*sin + p.y*cos} }
		p1, p2, end = orient(p1), orient(p2), orient(end)
	}

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
	if mode == "absolute" {
		length /= 24
	}
	seconds := min(5, max(2, 1.55+length/(150+100*rand.Float64())+0.45*rand.Float64()))
	m := &movement{duration: time.Duration(seconds * float64(time.Second))}
	steps := int(math.Ceil(seconds * 40))
	lastX, lastY := 0, 0
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		p := at(t * t * (3 - 2*t)) // Accelerate into the bend and slow down at the endpoint.
		x, y := int(math.Round(p.x)), int(math.Round(p.y))
		if mode == "absolute" {
			m.reports = append(m.reports, absoluteReport(startX+x, startY+y))
		} else {
			m.reports = append(m.reports, []byte{0, byte(int8(x - lastX)), byte(int8(y - lastY)), 0})
		}
		lastX, lastY = x, y
	}
	return m
}
