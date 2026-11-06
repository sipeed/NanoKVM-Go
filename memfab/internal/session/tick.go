package session

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/store"
)

const observationBatchLimit = 200

type Result struct {
	Scanned  int
	Started  int
	Ended    int
	ActiveID string
}

func TickNow(ctx context.Context, db *store.DB, cfg config.Config) (Result, error) {
	return Tick(ctx, db, cfg, time.Now())
}

func ForceEndActive(ctx context.Context, db *store.DB, now time.Time) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	active, err := db.GetActiveSession()
	if err != nil || active == nil {
		return false, err
	}
	start, err := capture.ParseTimestamp(active.StartTime)
	if err != nil {
		return false, fmt.Errorf("parse active session start: %w", err)
	}
	if err := persistDecision(db, Decision{Kind: DecisionEnd, ID: active.ID, StartTime: start, EndTime: now, Reason: "manual_force_end"}, now); err != nil {
		return false, err
	}
	return true, nil
}

func Tick(ctx context.Context, db *store.DB, cfg config.Config, now time.Time) (Result, error) {
	manager := NewManager(configFrom(cfg), sessionID)
	result := Result{}
	active, err := db.GetActiveSession()
	if err != nil {
		return result, err
	}
	startCursor := ""
	if active != nil {
		start, err := capture.ParseTimestamp(active.StartTime)
		if err != nil {
			return result, fmt.Errorf("parse active session start: %w", err)
		}
		manager.Seed(active.ID, start)
		startCursor = active.StartTime
	} else if latest, found, err := db.LatestSessionEnd(); err != nil {
		return result, err
	} else if found {
		startCursor = latest
	} else if earliest, found, err := db.EarliestObservationTime(); err != nil {
		return result, err
	} else if found {
		startCursor = ""
		_ = earliest
	} else {
		return result, nil
	}

	cursor := startCursor
	for {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}
		observations, err := db.ObservationsSince(cursor, observationBatchLimit)
		if err != nil {
			return result, err
		}
		if len(observations) == 0 {
			break
		}
		for _, obs := range observations {
			eventTime, err := capture.ParseTimestamp(obs.CapturedAt)
			if err != nil {
				return result, fmt.Errorf("parse observation timestamp: %w", err)
			}
			for _, decision := range manager.CheckCuts(eventTime) {
				if err := persistDecision(db, decision, now); err != nil {
					return result, err
				}
				if decision.Kind == DecisionEnd {
					result.Ended++
				}
			}
			for _, decision := range manager.OnEvent(Event{Time: eventTime, App: obs.AppName}) {
				if err := persistDecision(db, decision, now); err != nil {
					return result, err
				}
				if decision.Kind == DecisionStart {
					result.Started++
				}
			}
			result.Scanned++
			cursor = obs.CapturedAt
		}
		if len(observations) < observationBatchLimit {
			break
		}
	}
	for _, decision := range manager.CheckCuts(now) {
		if err := persistDecision(db, decision, now); err != nil {
			return result, err
		}
		if decision.Kind == DecisionEnd {
			result.Ended++
		}
	}
	result.ActiveID = manager.ActiveID()
	return result, nil
}

func RunForever(ctx context.Context, db *store.DB, cfg config.Config) {
	if result, err := TickNow(ctx, db, cfg); err != nil {
		log.Printf("session tick failed: %v", err)
	} else if result.Started > 0 || result.Ended > 0 {
		log.Printf("session tick scanned=%d started=%d ended=%d active=%s", result.Scanned, result.Started, result.Ended, valueOrDash(result.ActiveID))
	}
	seconds := cfg.Session.TickSeconds
	if seconds < 5 {
		seconds = 5
	}
	ticker := time.NewTicker(time.Duration(seconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			result, err := Tick(ctx, db, cfg, now)
			if err != nil {
				log.Printf("session tick failed: %v", err)
			} else if result.Started > 0 || result.Ended > 0 {
				log.Printf("session tick scanned=%d started=%d ended=%d active=%s", result.Scanned, result.Started, result.Ended, valueOrDash(result.ActiveID))
			}
		}
	}
}

func persistDecision(db *store.DB, decision Decision, now time.Time) error {
	switch decision.Kind {
	case DecisionStart:
		_, err := db.InsertSession(store.SessionRow{
			ID:        decision.ID,
			StartTime: decision.StartTime.Format(time.RFC3339),
			Status:    string(store.SessionActive),
			CreatedAt: now.Format(time.RFC3339),
			UpdatedAt: now.Format(time.RFC3339),
		})
		return err
	case DecisionEnd:
		return db.MarkSessionEnded(decision.ID, decision.EndTime.Format(time.RFC3339), now.Format(time.RFC3339))
	default:
		return nil
	}
}

func configFrom(cfg config.Config) Config {
	gap := cfg.Session.GapMinutes
	if gap <= 0 {
		gap = 5
	}
	soft := cfg.Session.SoftCutMinutes
	if soft <= 0 {
		soft = 3
	}
	maxHours := cfg.Session.MaxSessionHours
	if maxHours <= 0 {
		maxHours = 2
	}
	return Config{Gap: time.Duration(gap) * time.Minute, SoftCut: time.Duration(soft) * time.Minute, MaxDuration: time.Duration(maxHours) * time.Hour}
}

func sessionID(t time.Time) string {
	raw := t.UTC().Format(time.RFC3339Nano)
	sum := sha1.Sum([]byte(raw))
	return "sess_" + hex.EncodeToString(sum[:6])
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
