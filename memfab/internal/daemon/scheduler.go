package daemon

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	ocSession "memfab/internal/session"
	"memfab/internal/store"
	ocTimeline "memfab/internal/timeline"
	"memfab/internal/writer/classifier"
	"memfab/internal/writer/compact"
	"memfab/internal/writer/reducer"
)

type taskFunc func(context.Context, time.Time) error

func Run(ctx context.Context, db *store.DB, cfg config.Config) {
	llmGate := make(chan struct{}, maxInt(1, cfg.Daemon.LLMConcurrency))
	if cfg.Daemon.TimelineEnabled {
		go safeLoop(ctx, "timeline", seconds(cfg.Daemon.TimelineIntervalSecs, 60), true, withGate(llmGate, func(ctx context.Context, now time.Time) error {
			produced, err := ocTimeline.Tick(ctx, db, cfg)
			if err == nil && produced > 0 {
				log.Printf("daemon timeline produced=%d", produced)
			}
			return err
		}))
	}
	if cfg.Daemon.SessionEnabled {
		go safeLoop(ctx, "session", seconds(cfg.Session.TickSeconds, 5), true, func(ctx context.Context, now time.Time) error {
			result, err := ocSession.Tick(ctx, db, cfg, now)
			if err == nil && (result.Started > 0 || result.Ended > 0) {
				log.Printf("daemon session scanned=%d started=%d ended=%d active=%s", result.Scanned, result.Started, result.Ended, result.ActiveID)
			}
			return err
		})
	}
	if cfg.Daemon.ReducerEnabled {
		go safeLoop(ctx, "reducer", seconds(cfg.Daemon.ReducerIntervalSecs, 60), true, withGate(llmGate, func(ctx context.Context, now time.Time) error {
			summary, err := reducer.TickDue(ctx, db, cfg, now, maxInt(1, cfg.Daemon.ReducerLimit))
			if err == nil && (summary.Reduced > 0 || summary.Failed > 0) {
				log.Printf("daemon reducer reduced=%d failed=%d written=%d heuristic=%d", summary.Reduced, summary.Failed, summary.Written, summary.Heuristic)
			}
			return err
		}))
		go safeLoop(ctx, "reducer-flush", minutes(cfg.Session.FlushMinutes, 5), false, withGate(llmGate, func(ctx context.Context, now time.Time) error {
			result, err := reducer.FlushActive(ctx, db, cfg, now)
			if err == nil && result.Written {
				log.Printf("daemon reducer flush session=%s path=%s entry=%s", result.SessionID, result.Path, result.EntryID)
			}
			return err
		}))
	}
	if cfg.Daemon.ClassifierEnabled {
		go safeLoop(ctx, "classifier", minutes(cfg.Classifier.IntervalMinutes, 5), false, withGate(llmGate, func(ctx context.Context, now time.Time) error {
			summary, err := classifier.TickDue(ctx, db, cfg, now, cfg.Classifier.MaxSessionsPerTick)
			if err == nil && (summary.Classified > 0 || summary.Failed > 0 || summary.Skipped > 0) {
				log.Printf("daemon classifier classified=%d failed=%d written=%d skipped=%d", summary.Classified, summary.Failed, summary.Written, summary.Skipped)
			}
			return err
		}))
	}
	if cfg.Daemon.CompactEnabled {
		go safeLoop(ctx, "compact", minutes(cfg.Daemon.CompactIntervalMinutes, 5), false, withGate(llmGate, func(ctx context.Context, now time.Time) error {
			results, err := compact.RunPending(ctx, db, cfg, now, cfg.Compact.MaxFilesPerTick)
			if err == nil && len(results) > 0 {
				accepted, rejected := compactCounts(results)
				log.Printf("daemon compact accepted=%d rejected=%d", accepted, rejected)
			}
			return err
		}))
	}
	if cfg.Daemon.DailySafetyEnabled {
		go dailySafetyLoop(ctx, db, cfg, llmGate)
	}
	<-ctx.Done()
}

func dailySafetyLoop(ctx context.Context, db *store.DB, cfg config.Config, llmGate chan struct{}) {
	var lastRun string
	safeLoop(ctx, "daily-safety", time.Minute, false, withGate(llmGate, func(ctx context.Context, now time.Time) error {
		if now.Hour() != cfg.Reducer.DailyTickHour || now.Minute() != cfg.Reducer.DailyTickMinute {
			return nil
		}
		day := now.Format("2006-01-02")
		if day == lastRun {
			return nil
		}
		lastRun = day
		return RunSafetyNet(ctx, db, cfg, now)
	}))
}

func RunSafetyNet(ctx context.Context, db *store.DB, cfg config.Config, now time.Time) error {
	if _, err := ocSession.ForceEndActive(ctx, db, now); err != nil {
		return err
	}
	if _, err := reducer.TickDue(ctx, db, cfg, now, maxInt(1, cfg.Daemon.ReducerLimit)); err != nil {
		return err
	}
	if _, err := classifier.TickDue(ctx, db, cfg, now, cfg.Classifier.MaxSessionsPerTick); err != nil {
		return err
	}
	if _, err := compact.RunPending(ctx, db, cfg, now, cfg.Compact.MaxFilesPerTick); err != nil {
		return err
	}
	if latest, found, err := db.LatestTimelineEnd(); err != nil {
		return err
	} else if found {
		stats, err := capture.CleanupBuffer(db, cfg.Capture, latest, now)
		if err != nil {
			return err
		}
		if stats.DeletedObservations > 0 || stats.EvictedObservations > 0 {
			log.Printf("daemon cleanup deleted_observations=%d evicted_observations=%d", stats.DeletedObservations, stats.EvictedObservations)
		}
	}
	_, err := db.Conn.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func safeLoop(ctx context.Context, name string, interval time.Duration, immediate bool, task taskFunc) {
	var busy int32
	run := func(now time.Time) {
		if !atomic.CompareAndSwapInt32(&busy, 0, 1) {
			log.Printf("daemon %s skipped: previous run still active", name)
			return
		}
		defer atomic.StoreInt32(&busy, 0)
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("daemon %s panic: %v", name, recovered)
			}
		}()
		if err := task(ctx, now); err != nil {
			log.Printf("daemon %s failed: %v", name, err)
		}
	}
	if immediate {
		run(time.Now())
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			run(now)
		}
	}
}

func withGate(gate chan struct{}, task taskFunc) taskFunc {
	return func(ctx context.Context, now time.Time) error {
		select {
		case gate <- struct{}{}:
			defer func() { <-gate }()
		case <-ctx.Done():
			return ctx.Err()
		}
		return task(ctx, now)
	}
}

func compactCounts(results []compact.Result) (int, int) {
	accepted, rejected := 0, 0
	for _, result := range results {
		if result.Accepted {
			accepted++
		} else {
			rejected++
		}
	}
	return accepted, rejected
}

func seconds(value, fallback int) time.Duration {
	if value <= 0 {
		value = fallback
	}
	if value < 5 {
		value = 5
	}
	return time.Duration(value) * time.Second
}

func minutes(value, fallback int) time.Duration {
	if value <= 0 {
		value = fallback
	}
	if value < 5 {
		value = 5
	}
	return time.Duration(value) * time.Minute
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
