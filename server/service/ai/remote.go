package ai

import (
	"fmt"
	"sync"
	"time"

	"NanoKVM-Server/service/hid"
)

type HIDWriter interface {
	WriteKeyboardReport([]byte) error
	WriteRelativeMouseReport([]byte) error
	WriteAbsoluteMouseReport([]byte) error
}

type Remote struct {
	hid     HIDWriter
	hidMu   sync.Mutex
	release func() error
}

func NewRemote() *Remote {
	return &Remote{
		hid: &routedHIDWriter{session: hid.InternalInputSessionAI},
		release: func() error {
			return hid.ReleaseInputSession(hid.InternalInputSessionAI, "AI action complete")
		},
	}
}

func newRemoteWithHID(writer HIDWriter) *Remote {
	return &Remote{hid: writer}
}

type routedHIDWriter struct {
	session uint64
}

func (writer *routedHIDWriter) WriteKeyboardReport(report []byte) error {
	return hid.RouteKeyboardInput(writer.session, report)
}

func (writer *routedHIDWriter) WriteRelativeMouseReport(report []byte) error {
	return hid.RoutePointerInput(writer.session, report)
}

func (writer *routedHIDWriter) WriteAbsoluteMouseReport(report []byte) error {
	return hid.RoutePointerInput(writer.session, report)
}

func (r *Remote) releaseInput() error {
	if r.release != nil {
		return r.release()
	}
	return nil
}

func (r *Remote) TypeText(text string, delay time.Duration) ([]rune, error) {
	reports, skipped := buildTypeReports(text)
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	for _, report := range reports {
		if err := r.hid.WriteKeyboardReport(report); err != nil {
			return skipped, err
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	return skipped, nil
}

func (r *Remote) PressKeys(keys []string, hold time.Duration) error {
	report, err := buildKeyComboReport(keys)
	if err != nil {
		return err
	}
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	if err := r.hid.WriteKeyboardReport(report); err != nil {
		return err
	}
	if hold > 0 {
		time.Sleep(hold)
	}
	return r.hid.WriteKeyboardReport(keyUpReport)
}

func (r *Remote) MoveAbsolute(x float64, y float64) error {
	report := buildAbsolutePointerReport(absoluteCoordinate(x), absoluteCoordinate(y), false, false)
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	return r.hid.WriteAbsoluteMouseReport(report)
}

func (r *Remote) TouchAbsolute(x float64, y float64, hold time.Duration) error {
	return r.TouchAbsoluteButton(x, y, "left", hold)
}

func (r *Remote) TouchAbsoluteButton(x float64, y float64, button string, hold time.Duration) error {
	hidX := absoluteCoordinate(x)
	hidY := absoluteCoordinate(y)
	secondary := button == "right"
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	if err := r.hid.WriteAbsoluteMouseReport(buildAbsolutePointerReport(hidX, hidY, true, secondary)); err != nil {
		return err
	}
	if hold > 0 {
		time.Sleep(hold)
	}
	return r.hid.WriteAbsoluteMouseReport(buildAbsolutePointerReport(hidX, hidY, false, false))
}

func (r *Remote) MoveRelative(deltaX int, deltaY int) error {
	reports, err := buildRelativeMoveReports(deltaX, deltaY)
	if err != nil {
		return err
	}
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	for _, report := range reports {
		if err := r.hid.WriteRelativeMouseReport(report); err != nil {
			return err
		}
	}
	return nil
}

func (r *Remote) Click(button string, clicks int, delay time.Duration) error {
	bit, err := mouseButtonBit(button)
	if err != nil {
		return err
	}
	clicks = clampInt(clicks, 1, 5)
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	for i := 0; i < clicks; i++ {
		if err := r.hid.WriteRelativeMouseReport(buildRelativeMouseReport(0, 0, bit, 0)); err != nil {
			return err
		}
		if delay > 0 {
			time.Sleep(delay)
		}
		if err := r.hid.WriteRelativeMouseReport(buildRelativeMouseReport(0, 0, 0, 0)); err != nil {
			return err
		}
		if delay > 0 {
			time.Sleep(delay)
		}
	}
	return nil
}

func (r *Remote) Scroll(deltaY int) error {
	if reportsForDelta(deltaY) > maxRelativeMouseReports {
		return fmt.Errorf("scroll exceeds %d reports", maxRelativeMouseReports)
	}
	r.hidMu.Lock()
	defer r.hidMu.Unlock()
	defer r.releaseInput()

	for deltaY != 0 {
		wheel := clampInt(deltaY, -127, 127)
		if err := r.hid.WriteRelativeMouseReport(buildRelativeMouseReport(0, 0, 0, wheel)); err != nil {
			return err
		}
		deltaY -= wheel
	}
	return nil
}
