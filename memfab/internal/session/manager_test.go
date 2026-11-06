package session

import (
	"testing"
	"time"
)

func testManager() *Manager {
	return NewManager(Config{Gap: 5 * time.Minute, SoftCut: 3 * time.Minute, MaxDuration: 2 * time.Hour}, func(t time.Time) string { return "sess_test" })
}

func at(minute int) time.Time {
	return time.Date(2026, 5, 14, 10, minute, 0, 0, time.UTC)
}

func TestManagerAutoStartsOnFirstEvent(t *testing.T) {
	m := testManager()
	decisions := m.OnEvent(Event{Time: at(0), App: "A"})
	if len(decisions) != 1 || decisions[0].Kind != DecisionStart || decisions[0].StartTime != at(0) || m.ActiveID() != "sess_test" {
		t.Fatalf("decisions=%+v active=%s", decisions, m.ActiveID())
	}
}

func TestManagerHardCutAfterIdleGap(t *testing.T) {
	m := testManager()
	m.OnEvent(Event{Time: at(0), App: "A"})
	m.OnEvent(Event{Time: at(1), App: "A"})
	decisions := m.CheckCuts(at(7))
	if len(decisions) != 1 || decisions[0].Reason != "idle_gap" || decisions[0].EndTime != at(1) {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestManagerNoHardCutBelowThreshold(t *testing.T) {
	m := testManager()
	m.OnEvent(Event{Time: at(0), App: "A"})
	if decisions := m.CheckCuts(at(4)); len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestManagerTimeoutCut(t *testing.T) {
	m := NewManager(Config{Gap: 24 * time.Hour, MaxDuration: time.Hour}, func(t time.Time) string { return "sess_test" })
	m.OnEvent(Event{Time: at(0), App: "A"})
	m.OnEvent(Event{Time: at(10), App: "A"})
	decisions := m.CheckCuts(at(0).Add(61 * time.Minute))
	if len(decisions) != 1 || decisions[0].Reason != "max_duration" || decisions[0].EndTime != at(10) {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestManagerSoftCutAfterAppSwitch(t *testing.T) {
	m := testManager()
	m.OnEvent(Event{Time: at(0), App: "A"})
	m.OnEvent(Event{Time: at(1), App: "B"})
	decisions := m.CheckCuts(at(5))
	if len(decisions) != 1 || decisions[0].Reason != "soft_app_switch" || decisions[0].EndTime != at(1) {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestManagerFrequentSwitchingPreventsSoftCut(t *testing.T) {
	m := testManager()
	m.OnEvent(Event{Time: at(0), App: "A"})
	m.OnEvent(Event{Time: at(1), App: "B"})
	m.OnEvent(Event{Time: at(2), App: "A"})
	if decisions := m.CheckCuts(at(4)); len(decisions) != 0 {
		t.Fatalf("decisions=%+v", decisions)
	}
}

func TestManagerForceEnd(t *testing.T) {
	m := testManager()
	m.OnEvent(Event{Time: at(0), App: "A"})
	m.OnEvent(Event{Time: at(1), App: "A"})
	decisions := m.ForceEnd(at(2), "shutdown")
	if len(decisions) != 1 || decisions[0].Reason != "shutdown" || decisions[0].StartTime != at(0) || decisions[0].EndTime != at(1) || m.ActiveID() != "" {
		t.Fatalf("decisions=%+v active=%s", decisions, m.ActiveID())
	}
}
