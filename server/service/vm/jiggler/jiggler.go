package jiggler

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"

	"NanoKVM-Server/service/hid"
	log "github.com/sirupsen/logrus"
)

const idleInterval = 30 * time.Second
const configFile = "/etc/kvm/mouse-jiggler"

var GetJiggler = sync.OnceValue(func() *Jiggler {
	return load(configFile, move, func() time.Duration {
		return time.Duration(rand.IntN(31)) * time.Second
	})
})

type settings struct {
	Mode      string `json:"mode"`
	MaxHours  int32  `json:"maxHours"`
	StartedAt int64  `json:"startedAt"` // Unix milliseconds; zero when off.
}

func (s settings) deadline() time.Time {
	if s.StartedAt == 0 || s.MaxHours == 0 {
		return time.Time{}
	}
	return time.UnixMilli(s.StartedAt + int64(s.MaxHours)*3600000)
}

type Status struct {
	Enabled     bool   `json:"enabled"`
	Mode        string `json:"mode"`
	MaxHours    int32  `json:"maxHours"`
	RemainingMs int64  `json:"remainingMs"`
	ElapsedMs   int64  `json:"elapsedMs"`
}

type Jiggler struct {
	mu        sync.Mutex
	path      string
	config    settings
	timer     *time.Timer
	idleSince time.Time
	moveAt    time.Time
	move      func(string)
	jitter    func() time.Duration
}

func load(path string, move func(string), jitter func() time.Duration) *Jiggler {
	j := &Jiggler{path: path, config: settings{Mode: "relative", MaxHours: 8}, move: move, jitter: jitter}
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Warnf("read mouse jiggler: %v", err)
		}
		return j
	}
	cfg := j.config
	switch string(bytes.TrimSpace(data)) {
	case "", "relative", "absolute":
		// Old firmware saved only the active mode, with no start timestamp.
		if string(bytes.TrimSpace(data)) == "absolute" {
			cfg.Mode = "absolute"
		}
		cfg.StartedAt = time.Now().UnixMilli()
		if err := j.save(cfg); err != nil {
			log.Warnf("migrate mouse jiggler: %v", err)
			return j
		}
	default:
		if json.Unmarshal(data, &cfg) != nil || (cfg.Mode != "relative" && cfg.Mode != "absolute") || cfg.MaxHours < 0 || cfg.StartedAt < 0 {
			log.Warn("invalid mouse jiggler settings")
			return j
		}
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	j.config = cfg
	j.idleSince = time.Now()
	j.scheduleLocked()
	return j
}

func (j *Jiggler) save(cfg settings) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(j.path), ".mouse-jiggler-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(data)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(file.Name(), j.path)
}

func (j *Jiggler) expireLocked(now time.Time) {
	if end := j.config.deadline(); !end.IsZero() && !now.Before(end) {
		j.config.StartedAt = 0
		if err := j.save(j.config); err != nil {
			log.Warnf("expire mouse jiggler: %v", err)
		}
	}
}

func (j *Jiggler) statusLocked() Status {
	now := time.Now().UnixMilli()
	elapsed, remaining := int64(0), int64(0)
	if j.config.StartedAt != 0 {
		elapsed = max(0, now-j.config.StartedAt)
	}
	if end := j.config.deadline(); !end.IsZero() {
		remaining = max(0, end.UnixMilli()-now)
	}
	return Status{Enabled: j.config.StartedAt != 0, Mode: j.config.Mode, MaxHours: j.config.MaxHours, RemainingMs: remaining, ElapsedMs: elapsed}
}

func (j *Jiggler) Status() Status {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.expireLocked(time.Now())
	return j.statusLocked()
}

func (j *Jiggler) Configure(enabled *bool, mode string, hours *int32) (Status, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := time.Now()
	j.expireLocked(now)
	cfg := j.config
	if hours != nil && *hours != cfg.MaxHours {
		cfg.MaxHours = *hours
		if cfg.StartedAt != 0 {
			cfg.StartedAt = now.UnixMilli()
		}
	}
	if mode != "" {
		cfg.Mode = mode
	}
	if cfg.MaxHours < 0 || (cfg.Mode != "absolute" && cfg.Mode != "relative") {
		return Status{}, errors.New("invalid mouse jiggler settings")
	}
	if enabled != nil {
		if !*enabled {
			cfg.StartedAt = 0
		} else if cfg.StartedAt == 0 {
			cfg.StartedAt = now.UnixMilli()
		}
	}
	if cfg != j.config {
		if err := j.save(cfg); err != nil {
			return Status{}, err
		}
		if (j.config.StartedAt == 0) != (cfg.StartedAt == 0) {
			j.idleSince = now
			j.moveAt = time.Time{}
		}
		j.config = cfg
		j.scheduleLocked()
	}
	return j.statusLocked(), nil
}

// Accepted remote input, including input during the random wait, starts a new idle period.
func (j *Jiggler) Update() {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.config.StartedAt != 0 {
		j.idleSince = time.Now()
		j.moveAt = time.Time{}
	}
}

func (j *Jiggler) scheduleLocked() {
	if j.timer != nil {
		j.timer.Stop()
		j.timer = nil
	}
	j.expireLocked(time.Now())
	if j.config.StartedAt == 0 {
		return
	}
	next := j.moveAt
	if next.IsZero() {
		next = j.idleSince.Add(idleInterval)
	}
	if end := j.config.deadline(); !end.IsZero() && end.Before(next) {
		next = end
	}
	var timer *time.Timer
	timer = time.AfterFunc(time.Until(next), func() {
		j.mu.Lock()
		defer j.mu.Unlock()
		if j.timer != timer {
			return
		} // A stop or reconfiguration superseded this callback.
		j.expireLocked(time.Now())
		if j.config.StartedAt != 0 {
			now := time.Now()
			idleAt := j.idleSince.Add(idleInterval)
			if j.moveAt.IsZero() && !now.Before(idleAt) {
				j.moveAt = idleAt.Add(j.jitter())
			}
			if !j.moveAt.IsZero() && !now.Before(j.moveAt) {
				j.move(j.config.Mode)
				j.idleSince = now
				j.moveAt = time.Time{}
			}
		}
		j.scheduleLocked()
	})
	j.timer = timer
}

func move(mode string) {
	defer func() {
		_ = hid.ReleaseInputSession(hid.InternalInputSessionJiggler, "mouse jiggler update")
	}()

	if mode == "absolute" {
		_ = hid.RoutePointerInput(hid.InternalInputSessionJiggler, []byte{hid.AbsoluteReportIDMouse, 0x00, 0x00, 0x3f, 0x00, 0x3f, 0x00})
		time.Sleep(100 * time.Millisecond)
		_ = hid.RoutePointerInput(hid.InternalInputSessionJiggler, []byte{hid.AbsoluteReportIDMouse, 0x00, 0xff, 0x3f, 0xff, 0x3f, 0x00})
	} else {
		_ = hid.RoutePointerInput(hid.InternalInputSessionJiggler, []byte{0x00, 0xa, 0xa, 0x00})
		time.Sleep(100 * time.Millisecond)
		_ = hid.RoutePointerInput(hid.InternalInputSessionJiggler, []byte{0x00, 0xf6, 0xf6, 0x00})
	}
}
