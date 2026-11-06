package timeline

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/paths"
	"memfab/internal/store"
)

const (
	maxObservationsPerWindow = 30
	maxOCRTextRunes          = 4000
	maxFallbackTextRunes     = 240
)

type llmCaller func(context.Context, config.ModelConfig, []llm.Message) (string, error)

var callJSON llmCaller = llm.CallJSON

type llmEntries struct {
	Entries []string `json:"entries"`
}

type promptOCRBlock struct {
	Text string `json:"text"`
	BBox []int  `json:"bbox"`
}

func FloorToWindow(t time.Time, windowMinutes int) time.Time {
	if windowMinutes <= 0 {
		windowMinutes = 1
	}
	window := time.Duration(windowMinutes) * time.Minute
	return t.Truncate(window)
}

type TickOptions struct {
	IncludeOpen bool
}

func Tick(ctx context.Context, db *store.DB, cfg config.Config) (int, error) {
	now := time.Now()
	return tickUntil(ctx, db, cfg, now)
}

func TickWithOptions(ctx context.Context, db *store.DB, cfg config.Config, opts TickOptions) (int, error) {
	return tickWithOptionsUntil(ctx, db, cfg, opts, time.Now())
}

func tickWithOptionsUntil(ctx context.Context, db *store.DB, cfg config.Config, opts TickOptions, now time.Time) (int, error) {
	produced, err := tickUntil(ctx, db, cfg, now)
	if err != nil || !opts.IncludeOpen {
		return produced, err
	}
	windowMinutes := cfg.Timeline.WindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = 1
	}
	start := FloorToWindow(now, windowMinutes)
	end := start.Add(time.Duration(windowMinutes) * time.Minute)
	inserted, err := ProduceBlockForWindow(ctx, db, cfg, start, end)
	if err != nil {
		return produced, err
	}
	if inserted {
		produced++
	}
	return produced, nil
}

func RunForever(ctx context.Context, db *store.DB, cfg config.Config) {
	_, _ = tickUntil(ctx, db, cfg, time.Now())
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			produced, err := tickUntil(ctx, db, cfg, now)
			if err != nil {
				log.Printf("timeline tick failed: %v", err)
			} else if produced > 0 {
				log.Printf("timeline produced %d block(s)", produced)
			}
		}
	}
}

func tickUntil(ctx context.Context, db *store.DB, cfg config.Config, now time.Time) (int, error) {
	windowMinutes := cfg.Timeline.WindowMinutes
	if windowMinutes <= 0 {
		windowMinutes = 1
	}
	closedEnd := FloorToWindow(now, windowMinutes)
	var start time.Time
	if latest, found, err := db.LatestTimelineEnd(); err != nil {
		return 0, err
	} else if found {
		parsed, err := capture.ParseTimestamp(latest)
		if err != nil {
			return 0, fmt.Errorf("parse latest timeline end: %w", err)
		}
		start = parsed
	} else {
		lookback := cfg.Timeline.ColdLookbackMinutes
		if lookback <= 0 {
			lookback = 30
		}
		start = FloorToWindow(now.Add(-time.Duration(lookback)*time.Minute), windowMinutes)
	}
	if !start.Before(closedEnd) {
		return 0, nil
	}
	produced := 0
	for cursor := start; cursor.Before(closedEnd); cursor = cursor.Add(time.Duration(windowMinutes) * time.Minute) {
		select {
		case <-ctx.Done():
			return produced, ctx.Err()
		default:
		}
		end := cursor.Add(time.Duration(windowMinutes) * time.Minute)
		inserted, err := ProduceBlockForWindow(ctx, db, cfg, cursor, end)
		if err != nil {
			return produced, err
		}
		if inserted {
			produced++
		}
	}
	return produced, nil
}

func ProduceBlockForWindow(ctx context.Context, db *store.DB, cfg config.Config, start, end time.Time) (bool, error) {
	startText := start.Format(time.RFC3339)
	endText := end.Format(time.RFC3339)
	has, err := db.HasTimelineWindow(startText, endText)
	if err != nil || has {
		return false, err
	}
	observations, err := db.ObservationsInWindow(startText, endText, maxObservationsPerWindow)
	if err != nil || len(observations) == 0 {
		return false, err
	}
	observations = dedupeObservations(observations)
	entries, err := entriesFromLLM(ctx, cfg, start, end, observations)
	if err != nil || len(entries) == 0 {
		entries = fallbackEntries(observations)
	}
	if len(entries) == 0 {
		return false, nil
	}
	block := store.TimelineBlock{
		ID:           blockID(start, end),
		StartTime:    startText,
		EndTime:      endText,
		Timezone:     zoneString(start),
		Entries:      entries,
		AppsUsed:     appsUsed(observations),
		CaptureCount: len(observations),
		CreatedAt:    time.Now().Format(time.RFC3339),
	}
	return db.InsertTimelineBlock(block)
}

func entriesFromLLM(ctx context.Context, cfg config.Config, start, end time.Time, observations []store.ObservationHit) ([]string, error) {
	model := cfg.Models.Timeline
	if strings.TrimSpace(model.Model) == "" {
		return nil, fmt.Errorf("timeline model is required")
	}
	out, err := callJSON(ctx, model, []llm.Message{
		{Role: "system", Content: systemPrompt()},
		{Role: "user", Content: renderPrompt(start, end, observations)},
	})
	if err != nil {
		return nil, err
	}
	var parsed llmEntries
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, err
	}
	entries := cleanEntries(parsed.Entries)
	if len(entries) == 0 {
		return nil, fmt.Errorf("timeline llm returned no entries")
	}
	return entries, nil
}

func systemPrompt() string {
	return `You normalize short OCR screen-observation windows into timeline entries. Preserve visible text, URLs, titles, file paths, and proper nouns verbatim. Do not summarize away specific evidence. Do not attribute people, files, or topics across independent apps/titles. Return only JSON with exactly one field: {"entries":[...]}.`
}

func renderPrompt(start, end time.Time, observations []store.ObservationHit) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Window: %s to %s (%d OCR screen snapshots). Records are chronological.\n", start.Format(time.RFC3339), end.Format(time.RFC3339), len(observations))
	b.WriteString("Each entry should look like: [<app>] <title/url/context>: <what was visible or changed>. Quote specific OCR evidence when useful.\n---\n")
	for i, obs := range observations {
		fmt.Fprintf(&b, "%d. [%s] %s", i+1, timeOnly(obs.CapturedAt), valueOr(obs.AppName, "ScreenOCR"))
		if obs.Title != "" {
			fmt.Fprintf(&b, " — %s", obs.Title)
		}
		if obs.URL != "" {
			fmt.Fprintf(&b, " (URL: %s)", obs.URL)
		}
		if obs.EventType != "" || obs.EventReason != "" {
			fmt.Fprintf(&b, " [%s %s]", obs.EventType, obs.EventReason)
		}
		if obs.Language != "" {
			fmt.Fprintf(&b, " lang=%s", obs.Language)
		}
		if obs.Confidence != nil {
			fmt.Fprintf(&b, " confidence=%.2f", *obs.Confidence)
		}
		if obs.BlockCount > 0 {
			fmt.Fprintf(&b, " blocks=%d", obs.BlockCount)
		}
		if layout := compactOCRBlocksJSON(obs.ID); layout != "" {
			fmt.Fprintf(&b, "\n   | layout_blocks=%s", layout)
		}
		text := strings.TrimSpace(truncateRunes(obs.OCRText, maxOCRTextRunes))
		if text != "" {
			fmt.Fprintf(&b, "\n   | %s", text)
		}
		b.WriteByte('\n')
	}
	b.WriteString("---\nOutput only a JSON object with an entries array. Do not use markdown fences.")
	return b.String()
}

func compactOCRBlocksJSON(stem string) string {
	stem = strings.TrimSpace(stem)
	if stem == "" {
		return ""
	}
	obs, err := capture.ReadFile(filepath.Join(paths.ObservationBufferDir(), stem+".json"))
	if err != nil {
		return ""
	}
	blocks := make([]promptOCRBlock, 0, len(obs.OCR.Blocks))
	for _, block := range obs.OCR.Blocks {
		text := strings.TrimSpace(block.Text)
		if text == "" || len(block.BBox) == 0 {
			continue
		}
		blocks = append(blocks, promptOCRBlock{Text: text, BBox: block.BBox})
	}
	if len(blocks) == 0 {
		return ""
	}
	data, err := json.Marshal(blocks)
	if err != nil {
		return ""
	}
	return string(data)
}

func dedupeObservations(observations []store.ObservationHit) []store.ObservationHit {
	seen := map[string]bool{}
	out := make([]store.ObservationHit, 0, len(observations))
	for _, obs := range observations {
		key := strings.ToLower(strings.TrimSpace(obs.AppName)) + "\x00" + strings.ToLower(strings.TrimSpace(obs.Title)) + "\x00"
		if strings.TrimSpace(obs.ImageSHA256) != "" {
			key += "sha:" + strings.TrimSpace(obs.ImageSHA256)
		} else {
			key += "text:" + normalizeText(obs.OCRText)
		}
		if key == "\x00\x00text:" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, obs)
	}
	return out
}

func fallbackEntries(observations []store.ObservationHit) []string {
	groups := make([][]store.ObservationHit, 0)
	for _, obs := range observations {
		if len(groups) == 0 {
			groups = append(groups, []store.ObservationHit{obs})
			continue
		}
		last := groups[len(groups)-1][len(groups[len(groups)-1])-1]
		if strings.EqualFold(last.AppName, obs.AppName) && strings.EqualFold(last.Title, obs.Title) {
			groups[len(groups)-1] = append(groups[len(groups)-1], obs)
		} else {
			groups = append(groups, []store.ObservationHit{obs})
		}
	}
	entries := make([]string, 0, len(groups))
	for _, group := range groups {
		first := group[0]
		app := valueOr(first.AppName, "ScreenOCR")
		context := strings.TrimSpace(first.Title)
		if context == "" {
			context = strings.TrimSpace(first.URL)
		}
		text := bestText(group)
		if text == "" {
			text = strings.TrimSpace(first.EventReason)
		}
		var entry string
		if context != "" {
			entry = fmt.Sprintf("[%s] %s: %s", app, context, text)
		} else {
			entry = fmt.Sprintf("[%s] OCR screen snapshot: %s", app, text)
		}
		entries = append(entries, strings.TrimSpace(entry))
	}
	return cleanEntries(entries)
}

func bestText(group []store.ObservationHit) string {
	best := ""
	for _, obs := range group {
		text := strings.TrimSpace(obs.OCRText)
		if len([]rune(text)) > len([]rune(best)) {
			best = text
		}
	}
	return truncateRunes(oneLine(best), maxFallbackTextRunes)
}

func appsUsed(observations []store.ObservationHit) []string {
	seen := map[string]bool{}
	apps := make([]string, 0)
	for _, obs := range observations {
		app := strings.TrimSpace(obs.AppName)
		if app == "" {
			app = "ScreenOCR"
		}
		key := strings.ToLower(app)
		if !seen[key] {
			seen[key] = true
			apps = append(apps, app)
		}
	}
	sort.Strings(apps)
	return apps
}

func cleanEntries(entries []string) []string {
	out := make([]string, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" || seen[entry] {
			continue
		}
		seen[entry] = true
		out = append(out, entry)
	}
	return out
}

func normalizeText(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

func oneLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func truncateRunes(text string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}

func timeOnly(ts string) string {
	t, err := capture.ParseTimestamp(ts)
	if err != nil {
		return ts
	}
	return t.Format("15:04:05")
}

func valueOr(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func zoneString(t time.Time) string {
	_, offset := t.Zone()
	if offset == 0 {
		return "Z"
	}
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s%02d:%02d", sign, offset/3600, (offset%3600)/60)
}

func blockID(start, end time.Time) string {
	raw := start.Format(time.RFC3339) + "__" + end.Format(time.RFC3339)
	sum := sha1.Sum([]byte(raw))
	return "tl_" + hex.EncodeToString(sum[:8])
}
