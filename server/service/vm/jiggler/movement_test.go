package jiggler

import (
	"encoding/binary"
	"math"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

func TestCurveReports(t *testing.T) {
	for i := 0; i < 100; i++ {
		for _, mode := range []string{"relative", "absolute"} {
			x, y := []int{0, 16384, 32767}[i%3], []int{0, 16384, 32767}[(i/3)%3]
			const width, height = 2560, 1440
			startPx := point{float64(x) * (width - 1) / 32767, float64(y) * (height - 1) / 32767}
			m := curveMovement(mode, startPx, width, height)
			if m.duration < 5*time.Second || m.duration > 8*time.Second {
				t.Fatal("duration", m.duration)
			}
			start := time.Now()
			m.start = start
			lastX, lastY := x, y
			rel := startPx
			length := 0.0
			for k, r := range m.reports {
				m.index = k
				if mode == "relative" {
					if len(r) != 4 || r[0] != 0 || r[3] != 0 {
						t.Fatal("relative buttons or wheel", r)
					}
					dx, dy := int(int8(r[1])), int(int8(r[2]))
					rel.x += float64(dx)
					rel.y += float64(dy)
					length += math.Hypot(float64(dx), float64(dy))
					if rel.x < 0 || rel.x > width-1 || rel.y < 0 || rel.y > height-1 {
						t.Fatal("relative curve moved beyond screen", rel)
					}
				} else {
					if len(r) != 7 || r[0] != 1 || r[1] != 0 || r[6] != 0 {
						t.Fatal("absolute buttons or wheel", r)
					}
					px, py := int(binary.LittleEndian.Uint16(r[2:])), int(binary.LittleEndian.Uint16(r[4:]))
					if px > 32767 || py > 32767 || (k == 0 && (px != x || py != y)) {
						t.Fatal("absolute coordinates", px, py)
					}
					length += math.Hypot(float64(px-lastX)*(width-1)/32767, float64(py-lastY)*(height-1)/32767)
					lastX, lastY = px, py
				}
			}
			if length < 210 || !m.nextFrame().Equal(start.Add(m.duration)) {
				t.Fatal("path length or end time", length)
			}
		}
	}
}

func TestDisconnectedMovementAndInterruption(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var curved []bool
		var writes, releases int
		j := load(filepath.Join(t.TempDir(), "settings"), func(_ string, useCurve bool) *movement {
			curved = append(curved, useCurve)
			if !useCurve {
				return nil
			}
			return curveMovement("relative", point{}, 1920, 1080)
		}, func() time.Duration { return 0 })
		j.send = func([]byte) error { writes++; return nil }
		j.release = func() { releases++ }
		on := true
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
		j.SetConnected(true)
		time.Sleep(6 * time.Minute)
		synctest.Wait()
		j.SetConnected(false)
		time.Sleep(5 * time.Minute)
		synctest.Wait()
		for _, v := range curved {
			if v {
				t.Fatal("curve before disconnect grace period elapsed")
			}
		}
		time.Sleep(30*time.Second + 200*time.Millisecond)
		synctest.Wait()
		if !curved[len(curved)-1] || writes < 2 {
			t.Fatal("missing curve", writes)
		}
		j.SetConnected(true)
		if releases != 1 {
			t.Fatal("reconnect did not release movement")
		}
		n := writes
		time.Sleep(5 * time.Second)
		if n != writes {
			t.Fatal("curve continued after reconnect")
		}
		j.SetConnected(false)
		time.Sleep(5*time.Minute + 25*time.Second + 200*time.Millisecond)
		synctest.Wait()
		if writes <= n {
			t.Fatal("second curve missing")
		}
		j.Update()
		n = writes
		time.Sleep(5 * time.Second)
		if n != writes {
			t.Fatal("curve continued after input")
		}
		on = false
		if _, err := j.Configure(&on, "", nil); err != nil {
			t.Fatal(err)
		}
	})
}
