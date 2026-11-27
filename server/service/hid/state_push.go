package hid

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/keyboardlayout"
)

const (
	hidStateURL                 = "http://127.0.0.1:65501/api/v1/hid/state"
	hidStateQueueSize           = 16
	hidMouseStateReportInterval = 100 * time.Millisecond
)

var (
	hidStateClient          = &http.Client{Timeout: 200 * time.Millisecond}
	hidStateOnce            sync.Once
	hidStateQueue           chan []byte
	hidMouseStateMutex      sync.Mutex
	hidMouseStateActive     bool
	hidMouseStateHasActive  bool
	hidMouseStateLastPushAt time.Time
)

type hidStatePayload struct {
	MouseActive     *bool   `json:"mouse_active,omitempty"`
	KeyboardSpecial *string `json:"keyboard_special,omitempty"`
	KeyboardNormal  *string `json:"keyboard_normal,omitempty"`
}

func pushHIDKeyboardState(report []byte) {
	special, normal := hidKeyboardLabels(report)
	pushHIDState(hidStatePayload{
		KeyboardSpecial: &special,
		KeyboardNormal:  &normal,
	})
}

func pushHIDMouseState(report []byte) {
	active := false
	for _, value := range report {
		if value != 0 {
			active = true
			break
		}
	}

	if !shouldPushHIDMouseState(active, time.Now()) {
		return
	}

	pushHIDState(hidStatePayload{MouseActive: &active})
}

func shouldPushHIDMouseState(active bool, now time.Time) bool {
	hidMouseStateMutex.Lock()
	defer hidMouseStateMutex.Unlock()

	if hidMouseStateHasActive &&
		hidMouseStateActive == active &&
		now.Sub(hidMouseStateLastPushAt) < hidMouseStateReportInterval {
		return false
	}

	hidMouseStateActive = active
	hidMouseStateHasActive = true
	hidMouseStateLastPushAt = now
	return true
}

func pushHIDState(payload hidStatePayload) {
	body, err := json.Marshal(payload)
	if err != nil {
		log.Debugf("marshal HID state failed: %v", err)
		return
	}

	hidStateOnce.Do(startHIDStateWorker)
	select {
	case hidStateQueue <- body:
	default:
		select {
		case <-hidStateQueue:
		default:
		}
		select {
		case hidStateQueue <- body:
		default:
		}
	}
}

func startHIDStateWorker() {
	hidStateQueue = make(chan []byte, hidStateQueueSize)
	go func() {
		for body := range hidStateQueue {
			postHIDState(body)
		}
	}()
}

func postHIDState(body []byte) {
	req, err := http.NewRequest(http.MethodPost, hidStateURL, bytes.NewReader(body))
	if err != nil {
		log.Debugf("create HID state request failed: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := hidStateClient.Do(req)
	if err != nil {
		log.Debugf("push HID state failed: %v", err)
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Debugf("push HID state returned status %d", resp.StatusCode)
		return
	}
}

func hidKeyboardLabels(report []byte) (string, string) {
	special := hidKeyboardSpecialLabel(report)
	normal := ""
	shifted := report[0]&(1<<1|1<<5) != 0

	for _, code := range report[2:] {
		if code == 0 {
			continue
		}
		if code == 0x2b {
			if special == "" {
				special = "tab"
			}
			continue
		}

		normal = hidKeyboardNormalLabel(code, shifted)
		if normal != "" {
			break
		}
	}

	return special, normal
}

func hidKeyboardSpecialLabel(report []byte) string {
	modifier := report[0]
	switch {
	case modifier&(1<<0|1<<4) != 0:
		return "ctrl"
	case modifier&(1<<1|1<<5) != 0:
		return "shift"
	case modifier&(1<<2|1<<6) != 0:
		return "alt"
	case modifier&(1<<3|1<<7) != 0:
		return "meta"
	default:
		return ""
	}
}

func hidKeyboardNormalLabel(code byte, shifted bool) string {
	var modifiers byte
	if shifted {
		modifiers = 2
	}
	return keyboardlayout.USLabel(keyboardlayout.Stroke{Modifiers: modifiers, Code: code})
}
