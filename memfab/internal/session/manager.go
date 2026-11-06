package session

import "time"

type Event struct {
	Time time.Time
	App  string
}

type Config struct {
	Gap         time.Duration
	SoftCut     time.Duration
	MaxDuration time.Duration
}

type DecisionKind string

const (
	DecisionStart DecisionKind = "start"
	DecisionEnd   DecisionKind = "end"
)

type Decision struct {
	Kind      DecisionKind
	ID        string
	StartTime time.Time
	EndTime   time.Time
	Reason    string
}

type IDGenerator func(time.Time) string

type Manager struct {
	cfg Config
	id  IDGenerator

	active         bool
	sessionID      string
	sessionStart   time.Time
	lastEventTime  time.Time
	lastApp        string
	appSwitchedAt  time.Time
	recentSwitches []appSwitch
}

type appSwitch struct {
	At  time.Time
	App string
}

func NewManager(cfg Config, id IDGenerator) *Manager {
	if id == nil {
		id = func(t time.Time) string { return "sess_" + t.UTC().Format("20060102T150405") }
	}
	return &Manager{cfg: cfg, id: id}
}

func (m *Manager) Seed(id string, start time.Time) {
	m.active = true
	m.sessionID = id
	m.sessionStart = start
	m.lastEventTime = start
	m.appSwitchedAt = start
}

func (m *Manager) OnEvent(event Event) []Decision {
	if !m.active {
		id := m.id(event.Time)
		m.active = true
		m.sessionID = id
		m.sessionStart = event.Time
		m.lastEventTime = event.Time
		m.lastApp = event.App
		m.appSwitchedAt = event.Time
		m.recentSwitches = []appSwitch{{At: event.Time, App: event.App}}
		return []Decision{{Kind: DecisionStart, ID: id, StartTime: event.Time}}
	}
	if event.App != m.lastApp {
		m.lastApp = event.App
		m.appSwitchedAt = event.Time
		m.recentSwitches = append(m.recentSwitches, appSwitch{At: event.Time, App: event.App})
	}
	m.lastEventTime = event.Time
	m.pruneSwitches(event.Time)
	return nil
}

func (m *Manager) CheckCuts(now time.Time) []Decision {
	if !m.active || m.lastEventTime.IsZero() {
		return nil
	}
	if m.cfg.Gap > 0 && now.Sub(m.lastEventTime) > m.cfg.Gap {
		return m.end(m.lastEventTime, "idle_gap")
	}
	if m.cfg.MaxDuration > 0 && now.Sub(m.sessionStart) > m.cfg.MaxDuration {
		return m.end(m.lastEventTime, "max_duration")
	}
	if m.cfg.SoftCut > 0 && !m.appSwitchedAt.IsZero() && now.Sub(m.appSwitchedAt) > m.cfg.SoftCut && !m.frequentSwitching(now) && m.appSwitchedAt.After(m.sessionStart) {
		return m.end(m.appSwitchedAt, "soft_app_switch")
	}
	return nil
}

func (m *Manager) ForceEnd(now time.Time, reason string) []Decision {
	if !m.active {
		return nil
	}
	end := m.lastEventTime
	if end.IsZero() {
		end = now
	}
	if reason == "" {
		reason = "force_end"
	}
	return m.end(end, reason)
}

func (m *Manager) ActiveID() string {
	if !m.active {
		return ""
	}
	return m.sessionID
}

func (m *Manager) end(end time.Time, reason string) []Decision {
	decision := Decision{Kind: DecisionEnd, ID: m.sessionID, StartTime: m.sessionStart, EndTime: end, Reason: reason}
	m.active = false
	m.sessionID = ""
	m.sessionStart = time.Time{}
	m.lastEventTime = time.Time{}
	m.lastApp = ""
	m.appSwitchedAt = time.Time{}
	m.recentSwitches = nil
	return []Decision{decision}
}

func (m *Manager) frequentSwitching(now time.Time) bool {
	m.pruneSwitches(now)
	apps := map[string]bool{}
	for _, sw := range m.recentSwitches {
		apps[sw.App] = true
	}
	return len(apps) >= 2
}

func (m *Manager) pruneSwitches(now time.Time) {
	cutoff := now.Add(-2 * time.Minute)
	out := m.recentSwitches[:0]
	for _, sw := range m.recentSwitches {
		if !sw.At.Before(cutoff) {
			out = append(out, sw)
		}
	}
	m.recentSwitches = out
}
