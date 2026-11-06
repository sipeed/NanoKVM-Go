package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"memfab/internal/config"
	"memfab/internal/store"
	"memfab/internal/writer/classifier"
	"memfab/internal/writer/compact"
	"memfab/internal/writer/reducer"
)

func writer(args []string) error {
	if len(args) == 0 {
		return errors.New("writer requires run or classify")
	}
	db, cfg, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	now := time.Now()
	switch args[0] {
	case "run":
		reducerSummary, err := reducer.TickDue(context.Background(), db, cfg, now, 20)
		if err != nil {
			return err
		}
		classifierSummary, err := classifier.TickDue(context.Background(), db, cfg, now, cfg.Classifier.MaxSessionsPerTick)
		if err != nil {
			return err
		}
		compactResults, err := compact.RunPending(context.Background(), db, cfg, now, cfg.Compact.MaxFilesPerTick)
		if err != nil {
			return err
		}
		accepted, rejected := compactCounts(compactResults)
		fmt.Printf("reduced=%d reducer_failed=%d event_written=%d heuristic=%d classified=%d classifier_failed=%d durable_written=%d classifier_skipped=%d compact_accepted=%d compact_rejected=%d\n", reducerSummary.Reduced, reducerSummary.Failed, reducerSummary.Written, reducerSummary.Heuristic, classifierSummary.Classified, classifierSummary.Failed, classifierSummary.Written, classifierSummary.Skipped, accepted, rejected)
	case "classify":
		summary, err := classifier.TickDue(context.Background(), db, cfg, now, cfg.Classifier.MaxSessionsPerTick)
		if err != nil {
			return err
		}
		fmt.Printf("classified=%d classifier_failed=%d durable_written=%d classifier_skipped=%d\n", summary.Classified, summary.Failed, summary.Written, summary.Skipped)
	case "compact":
		return writerCompact(db, cfg, now, args[1:])
	default:
		return fmt.Errorf("unknown writer command %s", args[0])
	}
	return nil
}

func writerCompact(db *store.DB, cfg config.Config, now time.Time, args []string) error {
	fs := flag.NewFlagSet("writer compact", flag.ExitOnError)
	pending := fs.Bool("pending", false, "compact pending files")
	limit := fs.Int("n", cfg.Compact.MaxFilesPerTick, "number of pending files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *pending {
		results, err := compact.RunPending(context.Background(), db, cfg, now, *limit)
		if err != nil {
			return err
		}
		for _, result := range results {
			printCompactResult(result)
		}
		accepted, rejected := compactCounts(results)
		fmt.Printf("compact_accepted=%d compact_rejected=%d\n", accepted, rejected)
		return nil
	}
	if fs.NArg() != 1 {
		return errors.New("usage: memfab writer compact [--pending -n N] <memory-file>")
	}
	result, err := compact.File(context.Background(), db, cfg, fs.Arg(0), now)
	if err != nil {
		return err
	}
	printCompactResult(result)
	return nil
}

func printCompactResult(result compact.Result) {
	status := "rejected"
	if result.Accepted {
		status = "accepted"
	}
	fmt.Printf("%s %s before_tokens=%d after_tokens=%d preservation=%.2f note=%s\n", result.Path, status, result.BeforeTokens, result.AfterTokens, result.PreservationRatio, result.Note)
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
