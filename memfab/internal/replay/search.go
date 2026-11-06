package replay

import (
	"errors"
	"fmt"
	"strings"

	"memfab/internal/capture"
	"memfab/internal/store"
)

const MaxSearchLimit = 5000

type SearchOptions struct {
	Query string
	Since string
	Until string
	Limit int
}

type SearchHit struct {
	CapturedAt string `json:"captured_at"`
	CaptureID  string `json:"capture_id"`
	OCRPreview string `json:"ocr_preview,omitempty"`
}

type SearchResult struct {
	Query   string      `json:"query"`
	Count   int         `json:"count"`
	Results []SearchHit `json:"results"`
}

func SearchTimesByOCR(db *store.DB, opts SearchOptions) (SearchResult, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return SearchResult{Results: []SearchHit{}}, errors.New("query is required")
	}
	if opts.Since != "" {
		if _, err := capture.ParseTimestamp(opts.Since); err != nil {
			return SearchResult{Query: query, Results: []SearchHit{}}, fmt.Errorf("invalid since: %w", err)
		}
	}
	if opts.Until != "" {
		if _, err := capture.ParseTimestamp(opts.Until); err != nil {
			return SearchResult{Query: query, Results: []SearchHit{}}, fmt.Errorf("invalid until: %w", err)
		}
	}

	resultLimit := normalizeSearchLimit(opts.Limit)
	rows, err := db.SearchOCRHits(query, store.OCRSearchOptions{
		Since: opts.Since,
		Until: opts.Until,
		Limit: MaxSearchLimit,
	})
	if err != nil {
		return SearchResult{Query: query, Results: []SearchHit{}}, err
	}

	result := SearchResult{
		Query:   query,
		Results: make([]SearchHit, 0, len(rows)),
	}
	for _, row := range rows {
		result.Results = append(result.Results, SearchHit{
			CapturedAt: row.CapturedAt,
			CaptureID:  row.CaptureID,
			OCRPreview: row.OCRPreview,
		})
		if len(result.Results) >= resultLimit {
			break
		}
	}
	result.Count = len(result.Results)
	return result, nil
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 || limit > MaxSearchLimit {
		return MaxSearchLimit
	}
	return limit
}
