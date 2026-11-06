package main

import (
	"errors"
	"flag"
	"fmt"

	ocReplay "memfab/internal/replay"
)

func replay(args []string) error {
	if len(args) == 0 {
		return errors.New("replay requires subcommand: search")
	}
	switch args[0] {
	case "search":
		return replaySearch(args[1:])
	default:
		return fmt.Errorf("unknown replay command %s", args[0])
	}
}

func replaySearch(args []string) error {
	db, cfg, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	if !cfg.Replay.Enabled {
		return errors.New("replay is disabled")
	}

	fs := flag.NewFlagSet("replay search", flag.ExitOnError)
	query := fs.String("query", "", "full OCR text query")
	since := fs.String("since", "", "RFC3339 start time")
	until := fs.String("until", "", "RFC3339 end time")
	limit := fs.Int("limit", 100, "maximum OCR hits")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := ocReplay.SearchTimesByOCR(db, ocReplay.SearchOptions{
		Query: *query,
		Since: *since,
		Until: *until,
		Limit: *limit,
	})
	if err != nil {
		return err
	}
	writePretty(result)
	return nil
}
