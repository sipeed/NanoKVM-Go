package hid

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type PointerMode string

var ErrPointerModeMismatch = errors.New("pointer mode mismatch")

const (
	PointerModeAbsolute        PointerMode = "absolute"
	PointerModeDigitizerStylus PointerMode = "digitizer-stylus"
	PointerModeTouchScreen     PointerMode = "touch-screen"
	PointerModeUnknown         PointerMode = "unknown"
	PointerModeSingleRelative  PointerMode = "single-relative"
	PointerModeSingleAbsolute  PointerMode = "single-absolute"
	PointerModeSingleStylus    PointerMode = "single-digitizer-stylus"
	PointerModeSingleTouch     PointerMode = "single-touch-screen"

	PointerModeFile = "/boot/usb.hid2_mode"
)

const legacyPointerModeAbsoluteAndroid PointerMode = "absolute-android"

type PointerModeStatus struct {
	Mode      PointerMode
	Switching bool
	Degraded  bool
	Error     string
}

type pointerModeManager struct {
	mutex        sync.RWMutex
	modeFile     string
	activeMode   PointerMode
	switching    bool
	degraded     bool
	lastError    string
	quiesceEpoch uint64
}

var (
	pointerModeManagerOnce sync.Once
	defaultPointerMode     *pointerModeManager
)

func newPointerModeManager(modeFile string) *pointerModeManager {
	return &pointerModeManager{
		modeFile:   modeFile,
		activeMode: readPointerModeFile(modeFile),
	}
}

func getPointerModeManager() *pointerModeManager {
	pointerModeManagerOnce.Do(func() {
		defaultPointerMode = newPointerModeManager(PointerModeFile)
	})
	return defaultPointerMode
}

func GetPointerModeStatus() PointerModeStatus {
	return getPointerModeManager().status()
}

func (manager *pointerModeManager) status() PointerModeStatus {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	return PointerModeStatus{
		Mode:      manager.activeMode,
		Switching: manager.switching,
		Degraded:  manager.degraded,
		Error:     manager.lastError,
	}
}

// The persisted profile encodes the layout and descriptor in one atomic marker.
func (mode PointerMode) SingleEndpoint() bool { return strings.HasPrefix(string(mode), "single-") }
func (mode PointerMode) method() PointerMode {
	return PointerMode(strings.TrimPrefix(string(mode), "single-"))
}
func (mode PointerMode) SupportsRelative() bool {
	return !mode.SingleEndpoint() || mode == PointerModeSingleRelative
}

func isKnownPointerMode(mode PointerMode) bool {
	switch mode {
	case PointerModeAbsolute, PointerModeDigitizerStylus, PointerModeTouchScreen,
		PointerModeSingleAbsolute, PointerModeSingleStylus, PointerModeSingleTouch, PointerModeSingleRelative:
		return true
	default:
		return false
	}
}

// beginSwitch marks a manager as changing after the caller has acquired the
// global gadget lock. It deliberately keeps the old active mode visible until
// the new descriptor has been verified and HID files have been reopened.
func (manager *pointerModeManager) beginSwitch(target PointerMode) (PointerModeStatus, bool, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()

	current := manager.statusLocked()
	if manager.switching {
		return current, false, ErrPointerModeSwitchInProgress
	}
	if !isKnownPointerMode(target) {
		return current, false, fmt.Errorf("%w: %q", ErrPointerModeSwitchInvalidMode, target)
	}
	if !manager.degraded && manager.activeMode == target {
		return current, true, nil
	}

	manager.switching = true
	return manager.statusLocked(), false, nil
}

func (manager *pointerModeManager) commitSwitch(mode PointerMode) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.activeMode = mode
	manager.degraded = false
	manager.lastError = ""
}

func (manager *pointerModeManager) finishSwitch() {
	manager.mutex.Lock()
	manager.switching = false
	manager.quiesceEpoch = 0
	manager.mutex.Unlock()
}

func (manager *pointerModeManager) abortSwitch() {
	manager.mutex.Lock()
	manager.switching = false
	manager.mutex.Unlock()
}

func (manager *pointerModeManager) markDegraded(err error, quiesceEpoch uint64) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	manager.activeMode = PointerModeUnknown
	manager.switching = false
	manager.degraded = true
	manager.quiesceEpoch = quiesceEpoch
	manager.lastError = ""
	if err != nil {
		manager.lastError = err.Error()
	}
}

func (manager *pointerModeManager) degradedQuiesceEpoch() (uint64, bool) {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.quiesceEpoch, manager.degraded && manager.quiesceEpoch != 0
}

func (manager *pointerModeManager) statusLocked() PointerModeStatus {
	return PointerModeStatus{
		Mode:      manager.activeMode,
		Switching: manager.switching,
		Degraded:  manager.degraded,
		Error:     manager.lastError,
	}
}

func parsePointerMode(value string) (PointerMode, bool) {
	mode := PointerMode(value)
	if mode == legacyPointerModeAbsoluteAndroid {
		return PointerModeDigitizerStylus, true
	}
	if isKnownPointerMode(mode) {
		return mode, true
	}
	return PointerModeAbsolute, false
}

func pointerModeReportID(mode PointerMode) (byte, bool) {
	switch mode.method() {
	case PointerModeAbsolute:
		return AbsoluteReportIDMouse, true
	case PointerModeDigitizerStylus:
		return AbsoluteReportIDStylus, true
	case PointerModeTouchScreen:
		return TouchScreenReportID, true
	default:
		return 0, false
	}
}

func ValidatePointerModeReportID(mode PointerMode, reportID byte) error {
	want, ok := pointerModeReportID(mode)
	if !ok {
		return fmt.Errorf("%w: active mode %q is invalid", ErrPointerModeMismatch, mode)
	}
	if reportID != want {
		return fmt.Errorf(
			"%w: active mode %q requires HID2 report ID %d, got %d",
			ErrPointerModeMismatch,
			mode,
			want,
			reportID,
		)
	}
	return nil
}

func readPointerModeFile(path string) PointerMode {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return PointerModeSingleAbsolute
	}
	if err != nil {
		log.Warnf("failed to read pointer mode from %s; falling back to absolute: %s", path, err)
		return PointerModeAbsolute
	}

	// Shell command substitution in usbdev.sh removes trailing newlines. Match
	// that behavior while keeping spaces, CR characters, and embedded lines
	// invalid.
	value := strings.TrimRight(string(data), "\n")
	mode, ok := parsePointerMode(value)
	if !ok {
		log.Warnf("invalid pointer mode in %s; falling back to absolute", path)
	}
	return mode
}

func (service *Service) GetPointerMode(c *gin.Context) {
	manager := service.pointerModeManager
	if manager == nil {
		manager = getPointerModeManager()
	}
	status := manager.status()

	var rsp proto.Response
	rsp.OkRspWithData(c, pointerModeStatusResponse(status))
	log.Debugf("get pointer mode: %s (switching: %t, degraded: %t)", status.Mode, status.Switching, status.Degraded)
}
