package compact

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"memfab/internal/config"
	"memfab/internal/memory"
)

var uniqueTokenRE = regexp.MustCompile(`[A-Za-z][A-Za-z0-9_-]{3,}`)

type validationStats struct {
	BeforeUnique      int
	AfterUnique       int
	PreservationRatio float64
}

func validateAndPrepare(name, original, candidate string, originalParsed memory.ParsedFile, cfg config.Config, now time.Time) (string, memory.ParsedFile, validationStats, error) {
	candidate = strings.TrimSpace(candidate)
	stats := preservationStats(original, candidate)
	if !strings.HasPrefix(candidate, "---\n") && !strings.HasPrefix(candidate, "---\r\n") {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("compact output missing frontmatter")
	}
	if !hasClosingFrontmatter(candidate) {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("compact output frontmatter is not closed")
	}
	parsed := memory.Parse(name, candidate)
	if len(originalParsed.Entries) > 0 && len(parsed.Entries) == 0 {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("compact output removed all entries")
	}
	if err := validateFrontmatter(originalParsed, parsed); err != nil {
		return "", memory.ParsedFile{}, stats, err
	}
	if struckCount(parsed.Entries) < struckCount(originalParsed.Entries) {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("compact output dropped struck-through history")
	}
	if !containsAll(supersededRefs(parsed.Entries), supersededRefs(originalParsed.Entries)) {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("compact output dropped supersede references")
	}
	threshold := cfg.Compact.PreservationRatio
	if threshold <= 0 {
		threshold = 0.95
	}
	if stats.PreservationRatio < threshold {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("unique-token preservation %.2f below %.2f", stats.PreservationRatio, threshold)
	}
	var err error
	candidate, err = memory.SetFrontmatterBool(candidate, "needs_compact", false)
	if err != nil {
		return "", memory.ParsedFile{}, stats, err
	}
	candidate, err = memory.SetFrontmatterInt(candidate, "entry_count", len(parsed.Entries))
	if err != nil {
		return "", memory.ParsedFile{}, stats, err
	}
	candidate, err = memory.SetFrontmatterScalar(candidate, "updated", now.Format(time.RFC3339))
	if err != nil {
		return "", memory.ParsedFile{}, stats, err
	}
	parsed = memory.Parse(name, candidate)
	if parsed.EntryCount != len(parsed.Entries) {
		return "", memory.ParsedFile{}, stats, fmt.Errorf("compact output entry_count mismatch")
	}
	return candidate + "\n", parsed, stats, nil
}

func hasClosingFrontmatter(text string) bool {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Index(text[4:], "\n---") >= 0
}

func validateFrontmatter(before, after memory.ParsedFile) error {
	if before.Description != after.Description {
		return fmt.Errorf("compact output changed description")
	}
	if before.Status != after.Status {
		return fmt.Errorf("compact output changed status")
	}
	if before.Created != after.Created {
		return fmt.Errorf("compact output changed created timestamp")
	}
	if !sameStrings(before.Tags, after.Tags) {
		return fmt.Errorf("compact output changed frontmatter tags")
	}
	return nil
}

func preservationStats(before, after string) validationStats {
	beforeSet := uniqueTerms(before)
	afterSet := uniqueTerms(after)
	kept := 0
	for term := range beforeSet {
		if afterSet[term] {
			kept++
		}
	}
	ratio := 1.0
	if len(beforeSet) > 0 {
		ratio = float64(kept) / float64(len(beforeSet))
	}
	return validationStats{BeforeUnique: len(beforeSet), AfterUnique: len(afterSet), PreservationRatio: ratio}
}

func uniqueTerms(text string) map[string]bool {
	matches := uniqueTokenRE.FindAllString(text, -1)
	out := make(map[string]bool, len(matches))
	for _, match := range matches {
		out[strings.ToLower(match)] = true
	}
	return out
}

func struckCount(entries []memory.ParsedEntry) int {
	count := 0
	for _, entry := range entries {
		body := strings.TrimSpace(entry.Body)
		if strings.Contains(body, "~~") {
			count++
		}
	}
	return count
}

func supersededRefs(entries []memory.ParsedEntry) map[string]bool {
	refs := map[string]bool{}
	for _, entry := range entries {
		if entry.SupersededBy != "" {
			refs[entry.SupersededBy] = true
		}
		for _, tag := range entry.Tags {
			if strings.HasPrefix(tag, "superseded-by:") {
				refs[strings.TrimPrefix(tag, "superseded-by:")] = true
			}
		}
	}
	return refs
}

func containsAll(have, want map[string]bool) bool {
	for key := range want {
		if !have[key] {
			return false
		}
	}
	return true
}

func sameStrings(a, b []string) bool {
	aa := append([]string(nil), a...)
	bb := append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	if len(aa) != len(bb) {
		return false
	}
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
