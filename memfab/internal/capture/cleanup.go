package capture

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"memfab/internal/config"
	"memfab/internal/paths"
)

type ObservationDeleter interface {
	DeleteObservation(id string) error
}

type CleanupStats struct {
	DeletedObservations     int   `json:"deleted_observations"`
	DeletedObservationBytes int64 `json:"deleted_observation_bytes"`
	EvictedObservations     int   `json:"evicted_observations"`
}

type cleanupCandidate struct {
	Path       string
	Stem       string
	CapturedAt time.Time
	Info       os.FileInfo
	Absorbed   bool
	Deleted    bool
}

func CleanupBuffer(db ObservationDeleter, cfg config.CaptureConfig, processedBefore string, now time.Time) (CleanupStats, error) {
	var stats CleanupStats
	if strings.TrimSpace(processedBefore) == "" {
		return stats, nil
	}
	cutoff, err := ParseTimestamp(processedBefore)
	if err != nil {
		return stats, err
	}
	candidates, err := cleanupCandidates(cutoff)
	if err != nil {
		return stats, err
	}
	bufferRetention := cfg.BufferRetentionHours
	if bufferRetention <= 0 {
		bufferRetention = 168
	}
	bufferCutoff := now.Add(-time.Duration(bufferRetention) * time.Hour)

	for i := range candidates {
		candidate := &candidates[i]
		if !candidate.Absorbed {
			continue
		}
		deleteObservation := candidate.CapturedAt.Before(bufferCutoff) || candidate.Info.ModTime().Before(bufferCutoff)
		if deleteObservation {
			if err := deleteObservationCandidate(db, candidate, &stats, false); err != nil {
				return stats, err
			}
		}
	}

	if cfg.BufferMaxMB > 0 {
		if err := enforceSizeLimit(db, candidates, int64(cfg.BufferMaxMB)*1024*1024, &stats); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

func cleanupCandidates(processedBefore time.Time) ([]cleanupCandidate, error) {
	files, err := filepath.Glob(filepath.Join(paths.ObservationBufferDir(), "*.json"))
	if err != nil {
		return nil, err
	}
	out := make([]cleanupCandidate, 0, len(files))
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		obs, err := ReadFile(path)
		if err != nil {
			continue
		}
		capturedAt, err := ParseTimestamp(obs.CapturedAt)
		if err != nil {
			continue
		}
		out = append(out, cleanupCandidate{Path: path, Stem: StemFromPath(path), CapturedAt: capturedAt, Info: info, Absorbed: capturedAt.Before(processedBefore)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt.Before(out[j].CapturedAt) })
	return out, nil
}

func enforceSizeLimit(db ObservationDeleter, candidates []cleanupCandidate, limitBytes int64, stats *CleanupStats) error {
	total := int64(0)
	for _, candidate := range candidates {
		if !candidate.Deleted {
			total += candidate.Info.Size()
		}
	}
	if total <= limitBytes {
		return nil
	}
	for i := range candidates {
		candidate := &candidates[i]
		if !candidate.Absorbed || candidate.Deleted {
			continue
		}
		if err := deleteObservationCandidate(db, candidate, stats, true); err != nil {
			return err
		}
		total -= candidate.Info.Size()
		if total <= limitBytes {
			break
		}
	}
	return nil
}

func deleteObservationCandidate(db ObservationDeleter, candidate *cleanupCandidate, stats *CleanupStats, evicted bool) error {
	if candidate.Deleted {
		return nil
	}
	if err := os.Remove(candidate.Path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := db.DeleteObservation(candidate.Stem); err != nil {
		return err
	}
	candidate.Deleted = true
	stats.DeletedObservations++
	stats.DeletedObservationBytes += candidate.Info.Size()
	if evicted {
		stats.EvictedObservations++
	}
	return nil
}
