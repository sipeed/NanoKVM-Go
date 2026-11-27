package hid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Hid struct {
	g0                           hidEndpoint
	g1                           hidEndpoint
	g2                           hidEndpoint
	kbMutex                      sync.Mutex
	mouseMutex                   sync.Mutex
	cursor                       cursorEstimate // protected by mouseMutex
	absolutePointerPositionMutex sync.Mutex
	lastAbsolutePointerPosition  [4]byte
	hasAbsolutePointerPosition   bool
	lastAbsolutePointerState     byte
	hasAbsolutePointerState      bool
	pointerModeFile              string
	pointerModeStatusFn          func() PointerModeStatus
	openFileFn                   func(string) (hidFile, error)
	nowFn                        func() time.Time
}

type hidFile interface {
	Write([]byte) (int, error)
	SetWriteDeadline(time.Time) error
	Sync() error
	Close() error
}

type hidEndpoint struct {
	file          hidFile
	timeoutStreak int
}

const (
	HID0 = "/dev/hidg0" // Keyboard
	HID1 = "/dev/hidg1" // Mouse (Relative Mode)
	HID2 = "/dev/hidg2" // Active absolute, stylus, or touchscreen profile.

	hidWriteTimeout           = 16 * time.Millisecond
	hidTimeoutReopenThreshold = 10

	AbsoluteReportIDMouse  = 0x01
	AbsoluteReportIDStylus = 0x02
	TouchScreenReportID    = 0x03
	TouchScreenReportSize  = 62
	TouchScreenMaxContacts = 10
)

var (
	hid     *Hid
	hidOnce sync.Once
)

func GetHid() *Hid {
	hidOnce.Do(func() {
		hid = &Hid{pointerModeFile: PointerModeFile, pointerModeStatusFn: GetPointerModeStatus}
	})
	return hid
}

func (h *Hid) Lock() {
	h.kbMutex.Lock()
	h.mouseMutex.Lock()
}

func (h *Hid) Unlock() {
	h.kbMutex.Unlock()
	h.mouseMutex.Unlock()
}

func (h *Hid) OpenNoLock() {
	h.CloseNoLock()
	_ = h.openEndpointsNoLock()
}

// openNoLockWithError reopens the complete HID composite while the caller
// holds both HID mutexes. Unlike OpenNoLock, it reports failures so a gadget
// rebuild transaction can roll back instead of declaring a half-open device
// healthy. Any partial open is closed before returning an error.
func (h *Hid) openNoLockWithError() error {
	h.CloseNoLock()
	if err := h.openEndpointsNoLock(); err != nil {
		h.CloseNoLock()
		return err
	}
	return nil
}

func (h *Hid) openEndpointsNoLock() error {
	var openErrors []error
	endpoints := []struct {
		endpoint *hidEndpoint
		path     string
	}{
		{endpoint: &h.g0, path: HID0},
		{endpoint: &h.g1, path: HID1},
		{endpoint: &h.g2, path: HID2},
	}
	// During a rebuild the new marker is installed before the manager commits.
	mode := readPointerModeFile(h.pointerModeFile)
	for _, item := range endpoints {
		if (item.path == HID1 && !mode.SupportsRelative()) || (item.path == HID2 && mode == PointerModeSingleRelative) {
			continue
		}
		if err := h.openEndpointNoLock(item.endpoint, item.path); err != nil {
			log.Errorf("open %s failed: %s", item.path, err)
			openErrors = append(openErrors, fmt.Errorf("open %s: %w", item.path, err))
		}
	}
	return errors.Join(openErrors...)
}

func (h *Hid) CloseNoLock() {
	closeEndpointNoLock(&h.g0)
	closeEndpointNoLock(&h.g1)
	closeEndpointNoLock(&h.g2)
}

func closeFile(file *hidFile) {
	if *file == nil {
		return
	}

	_ = (*file).Sync()
	_ = (*file).Close()
	*file = nil
}

func closeFileWithoutSync(file *hidFile) {
	if *file == nil {
		return
	}

	_ = (*file).Close()
	*file = nil
}

func (h *Hid) Open() {
	h.kbMutex.Lock()
	defer h.kbMutex.Unlock()
	h.mouseMutex.Lock()
	defer h.mouseMutex.Unlock()

	h.CloseNoLock()

	h.OpenNoLock()
}

func (h *Hid) Close() {
	h.kbMutex.Lock()
	defer h.kbMutex.Unlock()
	h.mouseMutex.Lock()
	defer h.mouseMutex.Unlock()

	h.CloseNoLock()
}

func (h *Hid) WriteKeyboardReport(data []byte) error {
	if len(data) != 8 {
		return errors.New("invalid keyboard report length")
	}

	h.kbMutex.Lock()
	defer h.kbMutex.Unlock()
	return h.writeKeyboardReportNoLock(data)
}

// withKeyboardLock keeps a multi-report keyboard sequence contiguous with
// respect to all keyboard report producers.
func (h *Hid) withKeyboardLock(fn func()) {
	h.kbMutex.Lock()
	defer h.kbMutex.Unlock()

	fn()
}

// writeHid0NoLock writes one keyboard report while kbMutex is already held.
func (h *Hid) writeHid0NoLock(data []byte) error {
	if err := h.writeKeyboardReportNoLock(data); err != nil {
		log.Errorf("write to %s failed: %s", HID0, err)
		return err
	}

	log.Debugf("write to %s: %v", HID0, data)
	return nil
}

func (h *Hid) writeKeyboardReportNoLock(data []byte) error {
	if len(data) != 8 {
		return errors.New("invalid keyboard report length")
	}
	if err := h.writeNoLock(&h.g0, HID0, data); err != nil {
		return err
	}
	pushHIDKeyboardState(data)
	return nil
}

func (h *Hid) WriteRelativeMouseReport(data []byte) error {
	if len(data) != 4 {
		return errors.New("invalid relative mouse report length")
	}
	h.mouseMutex.Lock()
	defer h.mouseMutex.Unlock()
	if !h.currentPointerModeStatus().Mode.SupportsRelative() {
		return fmt.Errorf("%w: relative pointer is disabled", ErrPointerModeMismatch)
	}
	if err := h.writeNoLock(&h.g1, HID1, data); err != nil {
		return err
	}
	h.cursor.move(int8(data[1]), int8(data[2]))
	pushHIDMouseState(data)
	return nil
}

func (h *Hid) WriteAbsoluteMouseReport(data []byte) error {
	if len(data) != 7 {
		return errors.New("invalid absolute pointer report length")
	}
	if data[0] != AbsoluteReportIDMouse && data[0] != AbsoluteReportIDStylus {
		return errors.New("invalid absolute pointer report id")
	}
	h.mouseMutex.Lock()
	defer h.mouseMutex.Unlock()
	if err := ValidatePointerModeReportID(h.currentPointerModeStatus().Mode, data[0]); err != nil {
		return err
	}
	if err := h.writeNoLock(&h.g2, HID2, data); err != nil {
		return err
	}
	h.storeAbsolutePointerPosition(data)
	h.cursor.anchor(binary.LittleEndian.Uint16(data[2:4]), binary.LittleEndian.Uint16(data[4:6]))
	pushHIDMouseState(data[1:])
	return nil
}

func (h *Hid) WriteTouchScreenReport(data []byte) error {
	if len(data) != TouchScreenReportSize {
		return errors.New("invalid touch screen report length")
	}
	if data[0] != TouchScreenReportID {
		return errors.New("invalid touch screen report id")
	}
	if data[len(data)-1] > TouchScreenMaxContacts {
		return errors.New("invalid touch screen contact count")
	}
	h.mouseMutex.Lock()
	defer h.mouseMutex.Unlock()
	if err := ValidatePointerModeReportID(h.currentPointerModeStatus().Mode, data[0]); err != nil {
		return err
	}
	if err := h.writeNoLock(&h.g2, HID2, data); err != nil {
		return err
	}
	pushHIDMouseState(data[len(data)-1:])
	return nil
}

func (h *Hid) ReleaseAllInputs() error {
	return errors.Join(
		h.WriteKeyboardReport(make([]byte, 8)),
		h.releaseMouseButtons(),
	)
}

func (h *Hid) releaseMouseButtons() error {
	var releaseErrors []error
	mode := h.currentPointerModeStatus().Mode
	if mode == PointerModeSingleRelative {
		return h.WriteRelativeMouseReport([]byte{0, 0, 0, 0})
	}
	reportID, ok := pointerModeReportID(mode)
	if !ok {
		releaseErrors = append(releaseErrors, fmt.Errorf("release %s: %w", HID2, ValidatePointerModeReportID(mode, 0)))
	} else if reportID == TouchScreenReportID {
		if err := h.WriteTouchScreenReport(touchScreenReleaseReport()); err != nil {
			releaseErrors = append(releaseErrors, fmt.Errorf("release %s touchscreen: %w", HID2, err))
		}
	} else {
		if report, ok := h.absolutePointerReleaseReport(reportID); ok {
			if err := h.WriteAbsoluteMouseReport(report); err != nil {
				releaseErrors = append(releaseErrors, fmt.Errorf("release %s absolute pointer: %w", HID2, err))
			}
		}
	}

	if !mode.SupportsRelative() {
		return errors.Join(releaseErrors...)
	}
	if err := h.WriteRelativeMouseReport([]byte{0, 0, 0, 0}); err != nil {
		releaseErrors = append(releaseErrors, fmt.Errorf("release %s relative pointer: %w", HID1, err))
	}
	return errors.Join(releaseErrors...)
}

func touchScreenReleaseReport() []byte {
	report := make([]byte, TouchScreenReportSize)
	report[0] = TouchScreenReportID
	for index := 0; index < TouchScreenMaxContacts; index++ {
		report[1+index*6+1] = byte(index + 1)
	}
	report[len(report)-1] = TouchScreenMaxContacts
	return report
}

// EstimatedMousePosition returns pixel coordinates and the dimensions they use.
func (h *Hid) EstimatedMousePosition(width, height uint16) (float64, float64, int, int) {
	h.mouseMutex.Lock()
	defer h.mouseMutex.Unlock()
	if width > 1 && height > 1 {
		h.cursor.width, h.cursor.height = int(width), int(height)
	}
	w, hgt := h.cursor.dimensions()
	return (0.5 + h.cursor.x) * float64(w-1), (0.5 + h.cursor.y) * float64(hgt-1), w, hgt
}

// Offsets from the center in fractions of the captured screen. USB relative
// reports are counts; treating one count as one pixel approximates host motion.
type cursorEstimate struct {
	x, y          float64
	width, height int
}

func (c *cursorEstimate) dimensions() (int, int) {
	if c.width <= 1 || c.height <= 1 {
		return 1920, 1080
	}
	return c.width, c.height
}

func (c *cursorEstimate) anchor(x, y uint16) {
	c.x = float64(min(x, 32767))/32767 - 0.5
	c.y = float64(min(y, 32767))/32767 - 0.5
}

func (c *cursorEstimate) move(dx, dy int8) {
	w, h := c.dimensions()
	c.x = max(-0.5, min(0.5, c.x+float64(dx)/float64(w-1)))
	c.y = max(-0.5, min(0.5, c.y+float64(dy)/float64(h-1)))
}

func (h *Hid) storeAbsolutePointerPosition(data []byte) {
	h.absolutePointerPositionMutex.Lock()
	defer h.absolutePointerPositionMutex.Unlock()

	copy(h.lastAbsolutePointerPosition[:], data[2:6])
	h.hasAbsolutePointerPosition = true
	h.lastAbsolutePointerState = data[1]
	h.hasAbsolutePointerState = true
}

// absolutePointerReleaseReport builds a buttons-up report at the last
// absolute position this process actually wrote. ok is false when no
// position was ever observed, or when the last successful HID2 absolute
// write carried no state bits, in which case the caller must skip the HID2
// write instead of fabricating a coordinate the host never received.
func (h *Hid) absolutePointerReleaseReport(reportID byte) (report []byte, ok bool) {
	h.absolutePointerPositionMutex.Lock()
	defer h.absolutePointerPositionMutex.Unlock()

	if !h.hasAbsolutePointerPosition || !h.hasAbsolutePointerState || h.lastAbsolutePointerState == 0 {
		return nil, false
	}
	position := h.lastAbsolutePointerPosition
	return []byte{reportID, 0, position[0], position[1], position[2], position[3], 0}, true
}

func (h *Hid) currentPointerModeStatus() PointerModeStatus {
	if h.pointerModeStatusFn != nil {
		return h.pointerModeStatusFn()
	}
	return PointerModeStatus{Mode: PointerModeAbsolute}
}

func (h *Hid) write(endpoint *hidEndpoint, path string, data []byte, mu *sync.Mutex) error {
	mu.Lock()
	defer mu.Unlock()
	return h.writeNoLock(endpoint, path, data)
}

func (h *Hid) writeNoLock(endpoint *hidEndpoint, path string, data []byte) error {
	if endpoint.file == nil {
		if err := h.openEndpointNoLock(endpoint, path); err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
	}

	err := h.writeOnce(endpoint.file, data)

	if err != nil {
		if errors.Is(err, os.ErrClosed) {
			if reopenErr := h.reopenEndpointNoLock(endpoint, path); reopenErr != nil {
				return fmt.Errorf(
					"reopen %s after closed write: %w",
					path,
					errors.Join(err, reopenErr),
				)
			}

			err = h.writeOnce(endpoint.file, data)
			if err == nil {
				endpoint.timeoutStreak = 0
				return nil
			}
		}

		if !errors.Is(err, os.ErrDeadlineExceeded) {
			endpoint.timeoutStreak = 0
			return fmt.Errorf("write %s: %w", path, err)
		}

		endpoint.timeoutStreak++
		if endpoint.timeoutStreak >= hidTimeoutReopenThreshold {
			log.Warnf(
				"write to %s timed out %d consecutive times; reopening file handle",
				path,
				hidTimeoutReopenThreshold,
			)

			if reopenErr := h.reopenEndpointNoLock(endpoint, path); reopenErr != nil {
				return fmt.Errorf(
					"write %s timed out %d consecutive times; reopen failed: %w",
					path,
					hidTimeoutReopenThreshold,
					errors.Join(err, reopenErr),
				)
			}
		}

		return fmt.Errorf("write %s: %w", path, err)
	}

	endpoint.timeoutStreak = 0
	return nil
}

func (h *Hid) writeOnce(file hidFile, data []byte) error {
	_ = file.SetWriteDeadline(h.currentTime().Add(hidWriteTimeout))
	written, err := file.Write(data)
	if err == nil && written != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func (h *Hid) openEndpointNoLock(endpoint *hidEndpoint, path string) error {
	endpoint.timeoutStreak = 0

	opened, err := h.openFile(path)
	if err != nil {
		endpoint.file = nil
		return err
	}

	endpoint.file = opened
	return nil
}

func (h *Hid) reopenEndpointNoLock(endpoint *hidEndpoint, path string) error {
	closeFileWithoutSync(&endpoint.file)
	endpoint.timeoutStreak = 0
	return h.openEndpointNoLock(endpoint, path)
}

func closeEndpointNoLock(endpoint *hidEndpoint) {
	closeFile(&endpoint.file)
	endpoint.timeoutStreak = 0
}

func (h *Hid) openFile(path string) (hidFile, error) {
	if h.openFileFn != nil {
		return h.openFileFn(path)
	}

	return os.OpenFile(path, os.O_WRONLY, 0o666)
}

func (h *Hid) currentTime() time.Time {
	if h.nowFn != nil {
		return h.nowFn()
	}

	return time.Now()
}
