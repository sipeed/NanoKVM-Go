package hid

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeHIDFile struct {
	writeErrs       []error
	defaultWriteErr error
	shortWrite      bool
	writes          [][]byte
	deadlines       []time.Time
	syncCalls       int
	closeCalls      int
}

func (f *fakeHIDFile) Write(data []byte) (int, error) {
	f.writes = append(f.writes, bytes.Clone(data))

	err := f.defaultWriteErr
	if len(f.writeErrs) > 0 {
		err = f.writeErrs[0]
		f.writeErrs = f.writeErrs[1:]
	}
	if err != nil {
		return 0, err
	}
	if f.shortWrite {
		return len(data) - 1, nil
	}

	return len(data), nil
}

func (f *fakeHIDFile) SetWriteDeadline(deadline time.Time) error {
	f.deadlines = append(f.deadlines, deadline)
	return nil
}

func (f *fakeHIDFile) Sync() error {
	f.syncCalls++
	return nil
}

func (f *fakeHIDFile) Close() error {
	f.closeCalls++
	return nil
}

type fakeHIDOpenResult struct {
	file hidFile
	err  error
}

type fakeHIDOpener struct {
	t       *testing.T
	results []fakeHIDOpenResult
	paths   []string
}

func (o *fakeHIDOpener) open(path string) (hidFile, error) {
	o.t.Helper()
	o.paths = append(o.paths, path)
	if len(o.results) == 0 {
		o.t.Fatalf("unexpected open of %s", path)
	}

	result := o.results[0]
	o.results = o.results[1:]
	return result.file, result.err
}

func wrappedHIDTimeout() error {
	return fmt.Errorf("fake HID write: %w", os.ErrDeadlineExceeded)
}

func TestWriteKeyboardReportReturnsWriteError(t *testing.T) {
	writeErr := errors.New("keyboard write failed")
	h := &Hid{g0: hidEndpoint{file: &fakeHIDFile{defaultWriteErr: writeErr}}}

	err := h.WriteKeyboardReport([]byte{0x00, 0x00, 0x04, 0x00, 0x00, 0x00, 0x00, 0x00})
	if !errors.Is(err, writeErr) {
		t.Fatalf("WriteKeyboardReport error = %v, want %v", err, writeErr)
	}
}

func TestWriteAbsoluteMouseReportDoesNotStoreFailedPosition(t *testing.T) {
	seed := []byte{AbsoluteReportIDMouse, 0x03, 0x34, 0x12, 0x78, 0x56, 0xff}
	next := []byte{AbsoluteReportIDMouse, 0x03, 0xab, 0xcd, 0xef, 0x01, 0}
	writeErr := errors.New("absolute write failed")

	tests := []struct {
		name string
		file *fakeHIDFile
	}{
		{name: "write error", file: &fakeHIDFile{defaultWriteErr: writeErr}},
		{name: "short write", file: &fakeHIDFile{shortWrite: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Hid{g2: hidEndpoint{file: tt.file}}
			h.storeAbsolutePointerPosition(seed)

			if err := h.WriteAbsoluteMouseReport(next); err == nil {
				t.Fatal("WriteAbsoluteMouseReport unexpectedly succeeded")
			}

			release, ok := h.absolutePointerReleaseReport(AbsoluteReportIDMouse)
			if !ok {
				t.Fatal("failed write cleared the previously observed position")
			}
			want := []byte{AbsoluteReportIDMouse, 0, seed[2], seed[3], seed[4], seed[5], 0}
			if !bytes.Equal(release, want) {
				t.Fatalf("release report after failed write = %v, want %v", release, want)
			}
		})
	}
}

func TestWriteAbsoluteMouseReportStoresSuccessfulPosition(t *testing.T) {
	report := []byte{AbsoluteReportIDMouse, 0x03, 0xab, 0xcd, 0xef, 0x01, 0}
	h := &Hid{g2: hidEndpoint{file: &fakeHIDFile{}}}

	if err := h.WriteAbsoluteMouseReport(report); err != nil {
		t.Fatalf("WriteAbsoluteMouseReport: %v", err)
	}

	release, ok := h.absolutePointerReleaseReport(AbsoluteReportIDMouse)
	if !ok {
		t.Fatal("successful write did not record the absolute position")
	}
	want := []byte{AbsoluteReportIDMouse, 0, report[2], report[3], report[4], report[5], 0}
	if !bytes.Equal(release, want) {
		t.Fatalf("release report after successful write = %v, want %v", release, want)
	}
}

func TestAbsolutePointerReleaseReportWithoutKnownPosition(t *testing.T) {
	h := &Hid{}
	for _, reportID := range []byte{AbsoluteReportIDMouse, AbsoluteReportIDStylus} {
		report, ok := h.absolutePointerReleaseReport(reportID)
		if ok || report != nil {
			t.Fatalf("release report ID %d = %v, %v; want nil, false", reportID, report, ok)
		}
	}
}

func TestAbsolutePointerReleaseReportPreservesLastPosition(t *testing.T) {
	h := &Hid{}
	h.storeAbsolutePointerPosition([]byte{AbsoluteReportIDMouse, 0x03, 0x34, 0x12, 0x78, 0x56, 0xff})

	for _, reportID := range []byte{AbsoluteReportIDMouse, AbsoluteReportIDStylus} {
		report, ok := h.absolutePointerReleaseReport(reportID)
		if !ok {
			t.Fatalf("release report ID %d missing after seeded position", reportID)
		}
		want := []byte{reportID, 0, 0x34, 0x12, 0x78, 0x56, 0}
		if !bytes.Equal(report, want) {
			t.Fatalf("release report ID %d = %v, want %v", reportID, report, want)
		}
	}
}

func TestAbsolutePointerReleaseReportSkipsStatelessWrite(t *testing.T) {
	for _, reportID := range []byte{AbsoluteReportIDMouse, AbsoluteReportIDStylus} {
		h := &Hid{}
		h.storeAbsolutePointerPosition([]byte{reportID, 0x00, 0x34, 0x12, 0x78, 0x56, 0xff})

		report, ok := h.absolutePointerReleaseReport(reportID)
		if ok || report != nil {
			t.Fatalf("stateless release report ID %d = %v, %v; want nil, false", reportID, report, ok)
		}
	}
}

func TestAbsolutePointerReleaseReportKeepsStatefulRelease(t *testing.T) {
	tests := []struct {
		name  string
		state byte
	}{
		{name: "mouse button", state: 0x01},
		{name: "stylus in range", state: 0x02},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seed := []byte{AbsoluteReportIDMouse, tt.state, 0x34, 0x12, 0x78, 0x56, 0xff}
			h := &Hid{}
			h.storeAbsolutePointerPosition(seed)

			for _, reportID := range []byte{AbsoluteReportIDMouse, AbsoluteReportIDStylus} {
				report, ok := h.absolutePointerReleaseReport(reportID)
				if !ok {
					t.Fatalf("release report ID %d missing after stateful write", reportID)
				}
				want := []byte{reportID, 0, seed[2], seed[3], seed[4], seed[5], 0}
				if !bytes.Equal(report, want) {
					t.Fatalf("release report ID %d = %v, want %v", reportID, report, want)
				}
			}
		})
	}
}

func TestAbsolutePointerReleaseStateEvidenceSurvivesFailedWrites(t *testing.T) {
	seed := []byte{AbsoluteReportIDMouse, 0x03, 0x34, 0x12, 0x78, 0x56, 0xff}
	next := []byte{AbsoluteReportIDMouse, 0x00, 0xab, 0xcd, 0xef, 0x01, 0}
	writeErr := errors.New("absolute write failed")

	tests := []struct {
		name string
		file *fakeHIDFile
	}{
		{name: "write error", file: &fakeHIDFile{defaultWriteErr: writeErr}},
		{name: "short write", file: &fakeHIDFile{shortWrite: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Hid{g2: hidEndpoint{file: tt.file}}
			h.storeAbsolutePointerPosition(seed)

			if err := h.WriteAbsoluteMouseReport(next); err == nil {
				t.Fatal("WriteAbsoluteMouseReport unexpectedly succeeded")
			}

			release, ok := h.absolutePointerReleaseReport(AbsoluteReportIDMouse)
			if !ok {
				t.Fatal("failed stateless write cleared the previously observed state")
			}
			want := []byte{AbsoluteReportIDMouse, 0, seed[2], seed[3], seed[4], seed[5], 0}
			if !bytes.Equal(release, want) {
				t.Fatalf("release report after failed write = %v, want %v", release, want)
			}
		})
	}
}

func TestWriteTouchScreenReport(t *testing.T) {
	file := &fakeHIDFile{}
	h := &Hid{
		g2:                  hidEndpoint{file: file},
		pointerModeStatusFn: fixedPointerModeStatus(PointerModeTouchScreen),
	}
	report := touchScreenReleaseReport()
	report[1] = 0x03
	report[2] = 0x01
	report[len(report)-1] = 0x01

	if err := h.WriteTouchScreenReport(report); err != nil {
		t.Fatalf("WriteTouchScreenReport: %v", err)
	}
	if len(file.writes) != 1 || !bytes.Equal(file.writes[0], report) {
		t.Fatalf("touch screen writes = %v, want [%v]", file.writes, report)
	}
}

func TestWriteTouchScreenReportRejectsInvalidFrames(t *testing.T) {
	tests := [][]byte{
		make([]byte, TouchScreenReportSize-1),
		append([]byte{0xff}, make([]byte, TouchScreenReportSize-1)...),
		func() []byte {
			report := touchScreenReleaseReport()
			report[len(report)-1] = TouchScreenMaxContacts + 1
			return report
		}(),
	}

	h := &Hid{pointerModeStatusFn: fixedPointerModeStatus(PointerModeTouchScreen)}
	for _, report := range tests {
		if err := h.WriteTouchScreenReport(report); err == nil {
			t.Fatalf("WriteTouchScreenReport(%v) unexpectedly succeeded", report)
		}
	}
}

func TestHID2WritersRejectReportsOutsideActiveMode(t *testing.T) {
	reports := []struct {
		name string
		id   byte
		data []byte
	}{
		{name: "absolute", id: AbsoluteReportIDMouse, data: []byte{AbsoluteReportIDMouse, 0, 0x00, 0x40, 0x00, 0x40, 0}},
		{name: "stylus", id: AbsoluteReportIDStylus, data: []byte{AbsoluteReportIDStylus, 0, 0x00, 0x40, 0x00, 0x40, 0}},
		{name: "touchscreen", id: TouchScreenReportID, data: touchScreenReleaseReport()},
	}
	modes := []PointerMode{PointerModeAbsolute, PointerModeDigitizerStylus, PointerModeTouchScreen}

	for _, mode := range modes {
		wantID, _ := pointerModeReportID(mode)
		for _, report := range reports {
			t.Run(string(mode)+"/"+report.name, func(t *testing.T) {
				file := &fakeHIDFile{}
				h := &Hid{
					g2:                  hidEndpoint{file: file},
					pointerModeStatusFn: fixedPointerModeStatus(mode),
				}

				var err error
				if report.id == TouchScreenReportID {
					err = h.WriteTouchScreenReport(report.data)
				} else {
					err = h.WriteAbsoluteMouseReport(report.data)
				}

				if report.id == wantID {
					if err != nil {
						t.Fatalf("active report rejected: %v", err)
					}
					if len(file.writes) != 1 || !bytes.Equal(file.writes[0], report.data) {
						t.Fatalf("active report writes = %v, want [%v]", file.writes, report.data)
					}
					return
				}

				if !errors.Is(err, ErrPointerModeMismatch) {
					t.Fatalf("inactive report error = %v, want pointer mode mismatch", err)
				}
				if len(file.writes) != 0 {
					t.Fatalf("inactive report reached HID2: %v", file.writes)
				}
			})
		}
	}
}

func TestReleaseAllInputsUsesOnlyActiveHID2Profile(t *testing.T) {
	seed := []byte{AbsoluteReportIDMouse, 0x03, 0x34, 0x12, 0x78, 0x56, 0xff}
	modes := []PointerMode{PointerModeAbsolute, PointerModeDigitizerStylus, PointerModeTouchScreen}
	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			keyboard := &fakeHIDFile{}
			relative := &fakeHIDFile{}
			hid2 := &fakeHIDFile{}
			h := &Hid{
				g0:                  hidEndpoint{file: keyboard},
				g1:                  hidEndpoint{file: relative},
				g2:                  hidEndpoint{file: hid2},
				pointerModeStatusFn: fixedPointerModeStatus(mode),
			}
			h.storeAbsolutePointerPosition(seed)

			if err := h.ReleaseAllInputs(); err != nil {
				t.Fatalf("ReleaseAllInputs: %v", err)
			}
			if len(keyboard.writes) != 1 || !bytes.Equal(keyboard.writes[0], make([]byte, 8)) {
				t.Fatalf("keyboard releases = %v", keyboard.writes)
			}
			if len(relative.writes) != 1 || !bytes.Equal(relative.writes[0], make([]byte, 4)) {
				t.Fatalf("relative releases = %v", relative.writes)
			}
			if len(hid2.writes) != 1 {
				t.Fatalf("HID2 releases = %v, want one active-profile report", hid2.writes)
			}
			wantID, _ := pointerModeReportID(mode)
			release := hid2.writes[0]
			if release[0] != wantID {
				t.Fatalf("HID2 release ID = %d, want %d", release[0], wantID)
			}
			if mode == PointerModeTouchScreen {
				return
			}
			if release[1] != 0 {
				t.Fatalf("HID2 release buttons = 0x%02x, want 0", release[1])
			}
			if !bytes.Equal(release[2:6], seed[2:6]) {
				t.Fatalf("HID2 release position = %v, want seeded %v", release[2:6], seed[2:6])
			}
		})
	}
}

func TestReleaseAllInputsWithoutKnownPositionSkipsHID2(t *testing.T) {
	keyboard := &fakeHIDFile{}
	relative := &fakeHIDFile{}
	hid2 := &fakeHIDFile{}
	h := &Hid{
		g0:                  hidEndpoint{file: keyboard},
		g1:                  hidEndpoint{file: relative},
		g2:                  hidEndpoint{file: hid2},
		pointerModeStatusFn: fixedPointerModeStatus(PointerModeAbsolute),
	}

	if err := h.ReleaseAllInputs(); err != nil {
		t.Fatalf("ReleaseAllInputs: %v", err)
	}
	if len(keyboard.writes) != 1 || !bytes.Equal(keyboard.writes[0], make([]byte, 8)) {
		t.Fatalf("keyboard releases = %v", keyboard.writes)
	}
	if len(relative.writes) != 1 || !bytes.Equal(relative.writes[0], make([]byte, 4)) {
		t.Fatalf("relative releases = %v", relative.writes)
	}
	if len(hid2.writes) != 0 {
		t.Fatalf("HID2 releases without known position = %v, want none", hid2.writes)
	}
}

func TestReleaseAllInputsSkipsStatelessHID2(t *testing.T) {
	keyboard := &fakeHIDFile{}
	relative := &fakeHIDFile{}
	hid2 := &fakeHIDFile{}
	h := &Hid{
		g0:                  hidEndpoint{file: keyboard},
		g1:                  hidEndpoint{file: relative},
		g2:                  hidEndpoint{file: hid2},
		pointerModeStatusFn: fixedPointerModeStatus(PointerModeAbsolute),
	}
	h.storeAbsolutePointerPosition([]byte{AbsoluteReportIDMouse, 0x00, 0x34, 0x12, 0x78, 0x56, 0xff})

	if err := h.ReleaseAllInputs(); err != nil {
		t.Fatalf("ReleaseAllInputs: %v", err)
	}
	if len(keyboard.writes) != 1 || !bytes.Equal(keyboard.writes[0], make([]byte, 8)) {
		t.Fatalf("keyboard releases = %v", keyboard.writes)
	}
	if len(relative.writes) != 1 || !bytes.Equal(relative.writes[0], make([]byte, 4)) {
		t.Fatalf("relative releases = %v", relative.writes)
	}
	if len(hid2.writes) != 0 {
		t.Fatalf("stateless HID2 releases = %v, want none", hid2.writes)
	}
}

func TestReleaseAllInputsWithUnknownModeSkipsHID2(t *testing.T) {
	keyboard := &fakeHIDFile{}
	relative := &fakeHIDFile{}
	hid2 := &fakeHIDFile{}
	h := &Hid{
		g0:                  hidEndpoint{file: keyboard},
		g1:                  hidEndpoint{file: relative},
		g2:                  hidEndpoint{file: hid2},
		pointerModeStatusFn: fixedPointerModeStatus(PointerMode("unknown")),
	}

	err := h.ReleaseAllInputs()
	if !errors.Is(err, ErrPointerModeMismatch) {
		t.Fatalf("ReleaseAllInputs error = %v, want pointer mode mismatch", err)
	}
	if len(keyboard.writes) != 1 || len(relative.writes) != 1 {
		t.Fatalf("unknown mode did not release keyboard/relative: keyboard=%v relative=%v", keyboard.writes, relative.writes)
	}
	if len(hid2.writes) != 0 {
		t.Fatalf("unknown mode guessed a HID2 release: %v", hid2.writes)
	}
}

func fixedPointerModeStatus(mode PointerMode) func() PointerModeStatus {
	return func() PointerModeStatus {
		return PointerModeStatus{Mode: mode}
	}
}

func TestTouchScreenReleaseReport(t *testing.T) {
	report := touchScreenReleaseReport()
	if len(report) != TouchScreenReportSize || report[0] != TouchScreenReportID {
		t.Fatalf("touch screen release report = %v", report)
	}
	if report[len(report)-1] != TouchScreenMaxContacts {
		t.Fatalf("touch screen release contact count = %d, want %d", report[len(report)-1], TouchScreenMaxContacts)
	}
	for index := 0; index < TouchScreenMaxContacts; index++ {
		offset := 1 + index*6
		if report[offset] != 0 || report[offset+1] != byte(index+1) {
			t.Fatalf("touch screen release slot %d = %v", index, report[offset:offset+6])
		}
		for _, value := range report[offset+2 : offset+6] {
			if value != 0 {
				t.Fatalf("touch screen release slot %d = %v", index, report[offset:offset+6])
			}
		}
	}
}

func TestHIDWriteReopensAfterConsecutiveTimeoutThreshold(t *testing.T) {
	timeoutFile := &fakeHIDFile{defaultWriteErr: wrappedHIDTimeout()}
	replacement := &fakeHIDFile{}
	opener := &fakeHIDOpener{
		t: t,
		results: []fakeHIDOpenResult{
			{file: timeoutFile},
			{file: replacement},
		},
	}
	fixedNow := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	h := &Hid{
		openFileFn: opener.open,
		nowFn:      func() time.Time { return fixedNow },
	}
	report := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	for attempt := 1; attempt < hidTimeoutReopenThreshold; attempt++ {
		err := h.write(&h.g0, HID0, report, &h.kbMutex)
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("write %d error = %v, want deadline exceeded", attempt, err)
		}
		if h.g0.timeoutStreak != attempt {
			t.Fatalf("write %d timeout streak = %d, want %d", attempt, h.g0.timeoutStreak, attempt)
		}
		if len(opener.paths) != 1 {
			t.Fatalf("write %d open calls = %d, want 1", attempt, len(opener.paths))
		}
		if timeoutFile.closeCalls != 0 {
			t.Fatalf("write %d close calls = %d, want 0", attempt, timeoutFile.closeCalls)
		}
	}

	err := h.write(&h.g0, HID0, report, &h.kbMutex)
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("threshold write error = %v, want deadline exceeded", err)
	}
	if len(opener.paths) != 2 {
		t.Fatalf("threshold open calls = %d, want 2", len(opener.paths))
	}
	if h.g0.file != replacement {
		t.Fatalf("threshold endpoint file = %T, want replacement", h.g0.file)
	}
	if h.g0.timeoutStreak != 0 {
		t.Fatalf("threshold timeout streak = %d, want 0", h.g0.timeoutStreak)
	}
	if timeoutFile.syncCalls != 0 || timeoutFile.closeCalls != 1 {
		t.Fatalf(
			"threshold old file sync/close calls = %d/%d, want 0/1",
			timeoutFile.syncCalls,
			timeoutFile.closeCalls,
		)
	}
	if len(timeoutFile.writes) != hidTimeoutReopenThreshold {
		t.Fatalf("old file writes = %d, want %d", len(timeoutFile.writes), hidTimeoutReopenThreshold)
	}
	if len(replacement.writes) != 0 {
		t.Fatalf("replacement writes at threshold = %d, want 0", len(replacement.writes))
	}

	if err := h.write(&h.g0, HID0, report, &h.kbMutex); err != nil {
		t.Fatalf("write after reopen failed: %v", err)
	}
	if len(replacement.writes) != 1 {
		t.Fatalf("replacement writes after recovery = %d, want 1", len(replacement.writes))
	}
	if len(replacement.deadlines) != 1 || !replacement.deadlines[0].Equal(fixedNow.Add(hidWriteTimeout)) {
		t.Fatalf("replacement deadlines = %v, want [%v]", replacement.deadlines, fixedNow.Add(hidWriteTimeout))
	}
}

func TestHIDWriteResetsTimeoutStreak(t *testing.T) {
	interruptErr := errors.New("non-timeout write failure")
	tests := []struct {
		name          string
		interruptErr  error
		wantInterrupt error
	}{
		{
			name: "successful write",
		},
		{
			name:          "non-timeout error",
			interruptErr:  interruptErr,
			wantInterrupt: interruptErr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writeErrs := make([]error, 0, 2*hidTimeoutReopenThreshold)
			for range hidTimeoutReopenThreshold - 1 {
				writeErrs = append(writeErrs, wrappedHIDTimeout())
			}
			writeErrs = append(writeErrs, test.interruptErr)
			for range hidTimeoutReopenThreshold {
				writeErrs = append(writeErrs, wrappedHIDTimeout())
			}

			first := &fakeHIDFile{writeErrs: writeErrs}
			replacement := &fakeHIDFile{}
			opener := &fakeHIDOpener{
				t: t,
				results: []fakeHIDOpenResult{
					{file: first},
					{file: replacement},
				},
			}
			h := &Hid{openFileFn: opener.open}
			report := []byte{0, 0, 0, 0, 0, 0, 0, 0}

			for attempt := 0; attempt < hidTimeoutReopenThreshold-1; attempt++ {
				err := h.write(&h.g0, HID0, report, &h.kbMutex)
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					t.Fatalf("initial timeout %d error = %v, want deadline exceeded", attempt+1, err)
				}
			}

			err := h.write(&h.g0, HID0, report, &h.kbMutex)
			if !errors.Is(err, test.wantInterrupt) {
				t.Fatalf("interrupt error = %v, want %v", err, test.wantInterrupt)
			}
			if h.g0.timeoutStreak != 0 {
				t.Fatalf("timeout streak after interrupt = %d, want 0", h.g0.timeoutStreak)
			}

			for attempt := 1; attempt < hidTimeoutReopenThreshold; attempt++ {
				err := h.write(&h.g0, HID0, report, &h.kbMutex)
				if !errors.Is(err, os.ErrDeadlineExceeded) {
					t.Fatalf("post-reset timeout %d error = %v, want deadline exceeded", attempt, err)
				}
				if len(opener.paths) != 1 {
					t.Fatalf("post-reset timeout %d open calls = %d, want 1", attempt, len(opener.paths))
				}
			}

			err = h.write(&h.g0, HID0, report, &h.kbMutex)
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				t.Fatalf("post-reset threshold error = %v, want deadline exceeded", err)
			}
			if len(opener.paths) != 2 {
				t.Fatalf("post-reset threshold open calls = %d, want 2", len(opener.paths))
			}
			if h.g0.file != replacement {
				t.Fatalf("post-reset endpoint did not use replacement file")
			}
		})
	}
}

func TestHIDWriteTimeoutStreaksArePerEndpoint(t *testing.T) {
	keyboard := &fakeHIDFile{defaultWriteErr: wrappedHIDTimeout()}
	relativeMouse := &fakeHIDFile{defaultWriteErr: wrappedHIDTimeout()}
	keyboardReplacement := &fakeHIDFile{}
	relativeMouseReplacement := &fakeHIDFile{}
	openedPaths := make([]string, 0, 2)
	h := &Hid{
		g0: hidEndpoint{file: keyboard},
		g1: hidEndpoint{file: relativeMouse},
		openFileFn: func(path string) (hidFile, error) {
			openedPaths = append(openedPaths, path)
			switch path {
			case HID0:
				return keyboardReplacement, nil
			case HID1:
				return relativeMouseReplacement, nil
			default:
				t.Fatalf("unexpected open path %s", path)
				return nil, nil
			}
		},
	}
	keyboardReport := []byte{0, 0, 0, 0, 0, 0, 0, 0}
	mouseReport := []byte{0, 0, 0, 0}

	for attempt := 1; attempt < hidTimeoutReopenThreshold; attempt++ {
		if err := h.write(&h.g0, HID0, keyboardReport, &h.kbMutex); !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("keyboard timeout %d error = %v, want deadline exceeded", attempt, err)
		}
		if err := h.write(&h.g1, HID1, mouseReport, &h.mouseMutex); !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("mouse timeout %d error = %v, want deadline exceeded", attempt, err)
		}
	}

	if err := h.write(&h.g0, HID0, keyboardReport, &h.kbMutex); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("keyboard threshold error = %v, want deadline exceeded", err)
	}
	if h.g0.file != keyboardReplacement {
		t.Fatalf("keyboard endpoint did not use replacement file")
	}
	if h.g1.file != relativeMouse {
		t.Fatalf("mouse endpoint changed when keyboard reopened")
	}
	if h.g1.timeoutStreak != hidTimeoutReopenThreshold-1 {
		t.Fatalf("mouse timeout streak = %d, want %d", h.g1.timeoutStreak, hidTimeoutReopenThreshold-1)
	}
	if len(openedPaths) != 1 || openedPaths[0] != HID0 {
		t.Fatalf("opened paths after keyboard threshold = %v, want [%s]", openedPaths, HID0)
	}

	if err := h.write(&h.g1, HID1, mouseReport, &h.mouseMutex); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("mouse threshold error = %v, want deadline exceeded", err)
	}
	if h.g1.file != relativeMouseReplacement {
		t.Fatalf("mouse endpoint did not use replacement file")
	}
	if len(openedPaths) != 2 || openedPaths[1] != HID1 {
		t.Fatalf("opened paths after both thresholds = %v, want [%s %s]", openedPaths, HID0, HID1)
	}
}

func TestHIDWriteRecoversAfterTimeoutReopenFailure(t *testing.T) {
	reopenErr := errors.New("reopen failed")
	timeoutFile := &fakeHIDFile{defaultWriteErr: wrappedHIDTimeout()}
	replacement := &fakeHIDFile{}
	opener := &fakeHIDOpener{
		t: t,
		results: []fakeHIDOpenResult{
			{err: reopenErr},
			{file: replacement},
		},
	}
	h := &Hid{
		g0:         hidEndpoint{file: timeoutFile},
		openFileFn: opener.open,
	}
	report := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	for attempt := 1; attempt < hidTimeoutReopenThreshold; attempt++ {
		err := h.write(&h.g0, HID0, report, &h.kbMutex)
		if !errors.Is(err, os.ErrDeadlineExceeded) {
			t.Fatalf("write %d error = %v, want deadline exceeded", attempt, err)
		}
	}

	err := h.write(&h.g0, HID0, report, &h.kbMutex)
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Fatalf("threshold error = %v, want deadline exceeded", err)
	}
	if !errors.Is(err, reopenErr) {
		t.Fatalf("threshold error = %v, want reopen error", err)
	}
	if h.g0.file != nil {
		t.Fatalf("endpoint file after failed reopen = %T, want nil", h.g0.file)
	}
	if h.g0.timeoutStreak != 0 {
		t.Fatalf("timeout streak after failed reopen = %d, want 0", h.g0.timeoutStreak)
	}
	if timeoutFile.syncCalls != 0 || timeoutFile.closeCalls != 1 {
		t.Fatalf(
			"old file sync/close calls = %d/%d, want 0/1",
			timeoutFile.syncCalls,
			timeoutFile.closeCalls,
		)
	}

	if err := h.write(&h.g0, HID0, report, &h.kbMutex); err != nil {
		t.Fatalf("lazy-open recovery write failed: %v", err)
	}
	if h.g0.file != replacement {
		t.Fatalf("endpoint file after lazy-open recovery = %T, want replacement", h.g0.file)
	}
	if len(opener.paths) != 2 || opener.paths[0] != HID0 || opener.paths[1] != HID0 {
		t.Fatalf("open paths = %v, want [%s %s]", opener.paths, HID0, HID0)
	}
	if len(replacement.writes) != 1 {
		t.Fatalf("replacement writes = %d, want 1", len(replacement.writes))
	}
}

func TestHIDOpenAndCloseResetTimeoutStreaks(t *testing.T) {
	oldKeyboard := &fakeHIDFile{}
	oldRelativeMouse := &fakeHIDFile{}
	oldAbsolutePointer := &fakeHIDFile{}
	newKeyboard := &fakeHIDFile{}
	newAbsolutePointer := &fakeHIDFile{}
	openedPaths := make([]string, 0, 3)
	h := &Hid{
		pointerModeFile: filepath.Join(t.TempDir(), "usb.hid2_mode"),
		g0:              hidEndpoint{file: oldKeyboard, timeoutStreak: hidTimeoutReopenThreshold - 1},
		g1:              hidEndpoint{file: oldRelativeMouse, timeoutStreak: hidTimeoutReopenThreshold - 1},
		g2:              hidEndpoint{file: oldAbsolutePointer, timeoutStreak: hidTimeoutReopenThreshold - 1},
		openFileFn: func(path string) (hidFile, error) {
			openedPaths = append(openedPaths, path)
			switch path {
			case HID0:
				return newKeyboard, nil
			case HID2:
				return newAbsolutePointer, nil
			default:
				t.Fatalf("unexpected open path %s", path)
				return nil, nil
			}
		},
	}

	h.OpenNoLock()

	if h.g0.file != newKeyboard || h.g1.file != nil || h.g2.file != newAbsolutePointer {
		t.Fatalf("open files = [%T %T %T], want replacement handles", h.g0.file, h.g1.file, h.g2.file)
	}
	if h.g0.timeoutStreak != 0 || h.g1.timeoutStreak != 0 || h.g2.timeoutStreak != 0 {
		t.Fatalf(
			"timeout streaks after open = [%d %d %d], want all zero",
			h.g0.timeoutStreak,
			h.g1.timeoutStreak,
			h.g2.timeoutStreak,
		)
	}
	if len(openedPaths) != 2 || openedPaths[0] != HID0 || openedPaths[1] != HID2 {
		t.Fatalf("opened paths = %v, want [%s %s]", openedPaths, HID0, HID2)
	}
	for name, file := range map[string]*fakeHIDFile{
		HID0: oldKeyboard,
		HID1: oldRelativeMouse,
		HID2: oldAbsolutePointer,
	} {
		if file.syncCalls != 1 || file.closeCalls != 1 {
			t.Fatalf("%s old file sync/close calls = %d/%d, want 1/1", name, file.syncCalls, file.closeCalls)
		}
	}

	h.g0.timeoutStreak = hidTimeoutReopenThreshold - 1
	h.g1.timeoutStreak = hidTimeoutReopenThreshold - 1
	h.g2.timeoutStreak = hidTimeoutReopenThreshold - 1
	h.CloseNoLock()

	if h.g0.file != nil || h.g1.file != nil || h.g2.file != nil {
		t.Fatalf("files after close = [%T %T %T], want all nil", h.g0.file, h.g1.file, h.g2.file)
	}
	if h.g0.timeoutStreak != 0 || h.g1.timeoutStreak != 0 || h.g2.timeoutStreak != 0 {
		t.Fatalf(
			"timeout streaks after close = [%d %d %d], want all zero",
			h.g0.timeoutStreak,
			h.g1.timeoutStreak,
			h.g2.timeoutStreak,
		)
	}
}

func TestHIDStrictOpenClosesPartialEndpointSet(t *testing.T) {
	keyboard := &fakeHIDFile{}
	hid2 := &fakeHIDFile{}
	openFailure := errors.New("relative mouse unavailable")
	opener := &fakeHIDOpener{
		t: t,
		results: []fakeHIDOpenResult{
			{file: keyboard},
			{err: openFailure},
			{file: hid2},
		},
	}
	h := &Hid{pointerModeFile: dualPointerModeFile(t), openFileFn: opener.open}

	if err := h.openNoLockWithError(); !errors.Is(err, openFailure) {
		t.Fatalf("strict open error = %v, want %v", err, openFailure)
	}
	if h.g0.file != nil || h.g1.file != nil || h.g2.file != nil {
		t.Fatalf("strict open retained partial endpoints: [%T %T %T]", h.g0.file, h.g1.file, h.g2.file)
	}
	if keyboard.closeCalls != 1 || hid2.closeCalls != 1 {
		t.Fatalf("partial endpoint close calls = keyboard:%d hid2:%d", keyboard.closeCalls, hid2.closeCalls)
	}
}

func TestHIDLegacyOpenKeepsAvailableEndpoints(t *testing.T) {
	keyboard := &fakeHIDFile{}
	hid2 := &fakeHIDFile{}
	opener := &fakeHIDOpener{
		t: t,
		results: []fakeHIDOpenResult{
			{file: keyboard},
			{err: errors.New("relative mouse unavailable")},
			{file: hid2},
		},
	}
	h := &Hid{pointerModeFile: dualPointerModeFile(t), openFileFn: opener.open}

	h.OpenNoLock()
	if h.g0.file != keyboard || h.g1.file != nil || h.g2.file != hid2 {
		t.Fatalf("legacy open endpoints = [%T %T %T]", h.g0.file, h.g1.file, h.g2.file)
	}
}

func dualPointerModeFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "usb.hid2_mode")
	if err := writePointerModeMarker(path, PointerModeAbsolute); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestHIDWriteClosedFileRetryUsesFreshDeadline(t *testing.T) {
	first := &fakeHIDFile{defaultWriteErr: fmt.Errorf("fake closed write: %w", os.ErrClosed)}
	replacement := &fakeHIDFile{}
	opener := &fakeHIDOpener{
		t:       t,
		results: []fakeHIDOpenResult{{file: replacement}},
	}
	firstNow := time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)
	secondNow := firstNow.Add(time.Second)
	nowValues := []time.Time{firstNow, secondNow}
	h := &Hid{
		g0:         hidEndpoint{file: first, timeoutStreak: hidTimeoutReopenThreshold - 1},
		openFileFn: opener.open,
		nowFn: func() time.Time {
			if len(nowValues) == 0 {
				t.Fatal("unexpected currentTime call")
			}
			now := nowValues[0]
			nowValues = nowValues[1:]
			return now
		},
	}
	report := []byte{0, 0, 0, 0, 0, 0, 0, 0}

	if err := h.write(&h.g0, HID0, report, &h.kbMutex); err != nil {
		t.Fatalf("write after closed-file retry failed: %v", err)
	}
	if len(first.deadlines) != 1 || !first.deadlines[0].Equal(firstNow.Add(hidWriteTimeout)) {
		t.Fatalf("first deadlines = %v, want [%v]", first.deadlines, firstNow.Add(hidWriteTimeout))
	}
	if len(replacement.deadlines) != 1 || !replacement.deadlines[0].Equal(secondNow.Add(hidWriteTimeout)) {
		t.Fatalf(
			"replacement deadlines = %v, want [%v]",
			replacement.deadlines,
			secondNow.Add(hidWriteTimeout),
		)
	}
	if first.syncCalls != 0 || first.closeCalls != 1 {
		t.Fatalf("first file sync/close calls = %d/%d, want 0/1", first.syncCalls, first.closeCalls)
	}
	if len(first.writes) != 1 || len(replacement.writes) != 1 {
		t.Fatalf("first/replacement writes = %d/%d, want 1/1", len(first.writes), len(replacement.writes))
	}
	if h.g0.file != replacement {
		t.Fatalf("endpoint file after closed retry = %T, want replacement", h.g0.file)
	}
	if h.g0.timeoutStreak != 0 {
		t.Fatalf("timeout streak after closed retry = %d, want 0", h.g0.timeoutStreak)
	}
	if len(nowValues) != 0 {
		t.Fatalf("unused now values = %v", nowValues)
	}
}

func TestEstimatedCursorTracksDeliveredMotion(t *testing.T) {
	relative := &fakeHIDFile{}
	h := &Hid{g1: hidEndpoint{file: relative}, g2: hidEndpoint{file: &fakeHIDFile{}}}
	check := func(wantX, wantY float64) {
		t.Helper()
		x, y, w, height := h.EstimatedMousePosition(2560, 1440)
		if math.Abs(x-wantX) > 0.01 || math.Abs(y-wantY) > 0.01 || w != 2560 || height != 1440 {
			t.Fatalf("position (%v,%v), want (%v,%v)", x, y, wantX, wantY)
		}
	}
	check(1279.5, 719.5)
	if err := h.WriteAbsoluteMouseReport([]byte{AbsoluteReportIDMouse, 0, 0xff, 0x7f, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	check(2559, 0)
	if err := h.WriteRelativeMouseReport([]byte{0, 100, 0xff, 0}); err != nil {
		t.Fatal(err)
	}
	check(2559, 0)
	if err := h.WriteRelativeMouseReport([]byte{0, 0xf6, 30, 0}); err != nil {
		t.Fatal(err)
	}
	check(2549, 30)
	relative.defaultWriteErr = errors.New("failed")
	if err := h.WriteRelativeMouseReport([]byte{0, 6, 15, 0}); err == nil {
		t.Fatal("expected failure")
	}
	check(2549, 30)
}
