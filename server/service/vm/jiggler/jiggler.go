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
	return load(configFile, planMovement, func() time.Duration {
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
	mu             sync.Mutex
	path           string
	config         settings
	timer          *time.Timer
	idleSince      time.Time
	moveAt         time.Time
	move           func(string, bool) *movement
	motion         *movement
	disconnectedAt time.Time
	send           func([]byte) error
	release        func()
	jitter         func() time.Duration
}

func load(path string, move func(string, bool) *movement, jitter func() time.Duration) *Jiggler {
	j := &Jiggler{path: path, config: settings{Mode: "relative", MaxHours: 8}, move: move, jitter: jitter, disconnectedAt: time.Now(),
		send:    func(report []byte) error { return hid.RoutePointerInput(hid.InternalInputSessionJiggler, report) },
		release: func() { _ = hid.ReleaseInputSession(hid.InternalInputSessionJiggler, "mouse jiggler movement") },
	}
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
		j.stopMovementLocked()
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
		if cfg.StartedAt == 0 || cfg.Mode != j.config.Mode {
			j.stopMovementLocked()
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
		j.stopMovementLocked()
		j.idleSince = time.Now()
		j.moveAt = time.Time{}
	}
}

// SetConnected receives transitions of the desktop control WebSocket registry.
func (j *Jiggler) SetConnected(connected bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if connected {
		j.disconnectedAt = time.Time{}
		j.stopMovementLocked()
	} else if j.disconnectedAt.IsZero() {
		j.disconnectedAt = time.Now()
	}
}

func (j *Jiggler) stopMovementLocked() {
	if j.motion != nil {
		j.release()
		j.motion = nil
	}
}

func (j *Jiggler) advanceLocked(now time.Time) {
	if j.motion == nil {
		idleAt := j.idleSince.Add(idleInterval)
		if now.Before(idleAt) {
			return
		}
		if j.moveAt.IsZero() {
			j.moveAt = idleAt.Add(j.jitter())
		}
		if now.Before(j.moveAt) {
			return
		}

		curved := !j.disconnectedAt.IsZero() && now.Sub(j.disconnectedAt) > 5*time.Minute
		j.motion = j.move(j.config.Mode, curved)
		j.idleSince = now
		j.moveAt = time.Time{}
		if j.motion == nil {
			return
		}
		j.motion.start = now
	}
	m := j.motion
	if now.Before(m.nextFrame()) {
		return
	}
	if err := j.send(m.reports[m.index]); err != nil {
		log.Debugf("mouse jiggler report: %v", err)
		j.stopMovementLocked()
		return
	}
	m.index++
	if m.index == len(m.reports) {
		j.stopMovementLocked()
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
	if j.motion != nil {
		next = j.motion.nextFrame()
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
			j.advanceLocked(time.Now())
		}
		j.scheduleLocked()
	})
	j.timer = timer
}
