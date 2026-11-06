package mcp

import (
	"strings"

	"memfab/internal/store"
)

func buildCurrentObservationContext(recent []store.ObservationHit, blocks []store.TimelineBlock) map[string]any {
	timelineBlocks := compactTimelineBlocks(blocks)
	if len(recent) == 0 {
		return map[string]any{
			"headline":               "No recent OCR observations",
			"recent_observations":    []map[string]any{},
			"recent_timeline_blocks": timelineBlocks,
			"top_text":               "",
			"apps":                   []string{},
			"titles":                 []string{},
			"keywords":               []string{},
			"observation_count":      0,
			"timeline_block_count":   len(timelineBlocks),
			"latest_captured_at":     "",
		}
	}

	compact := make([]map[string]any, 0, len(recent))
	appsInOrder := make([]string, 0, len(recent))
	titlesInOrder := make([]string, 0, len(recent))
	for _, h := range recent {
		compact = append(compact, compactObservation(h))
		appsInOrder = append(appsInOrder, h.AppName)
		titlesInOrder = append(titlesInOrder, h.Title)
	}
	apps := uniqueNonEmpty(appsInOrder, 8)
	titles := uniqueNonEmpty(titlesInOrder, 8)
	keywords := extractOCRKeywords(recent, 12)

	return map[string]any{
		"headline":               makeObservationHeadline(apps, titles, keywords),
		"recent_observations":    compact,
		"recent_timeline_blocks": timelineBlocks,
		"top_text":               topOCRText(recent, 12, 800),
		"apps":                   apps,
		"titles":                 titles,
		"keywords":               keywords,
		"observation_count":      len(recent),
		"timeline_block_count":   len(timelineBlocks),
		"latest_captured_at":     recent[0].CapturedAt,
	}
}

func compactTimelineBlocks(blocks []store.TimelineBlock) []map[string]any {
	out := make([]map[string]any, 0, len(blocks))
	for _, block := range blocks {
		out = append(out, map[string]any{
			"id":            block.ID,
			"start_time":    block.StartTime,
			"end_time":      block.EndTime,
			"timezone":      block.Timezone,
			"entries":       block.Entries,
			"apps_used":     block.AppsUsed,
			"capture_count": block.CaptureCount,
		})
	}
	return out
}

func compactObservation(h store.ObservationHit) map[string]any {
	return map[string]any{
		"captured_at":   h.CapturedAt,
		"file_stem":     h.ID,
		"source_kind":   h.SourceKind,
		"source_device": h.SourceDevice,
		"source_engine": h.SourceEngine,
		"app_name":      h.AppName,
		"title":         h.Title,
		"url":           h.URL,
		"event_type":    h.EventType,
		"event_reason":  h.EventReason,
		"language":      h.Language,
		"confidence":    h.Confidence,
		"block_count":   h.BlockCount,
		"tags":          h.Tags,
		"snippet":       h.Snippet,
	}
}

func topOCRText(recent []store.ObservationHit, maxLines int, maxRunes int) string {
	lines := make([]string, 0, maxLines)
	for _, h := range recent {
		for _, line := range strings.Split(h.OCRText, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			lines = append(lines, line)
			if len(lines) >= maxLines {
				return preview(strings.Join(lines, "\n"), maxRunes)
			}
		}
	}
	return preview(strings.Join(lines, "\n"), maxRunes)
}

func uniqueNonEmpty(values []string, limit int) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, limit)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, value)
		if len(out) >= limit {
			break
		}
	}
	return out
}

type keywordScore struct {
	term  string
	count int
	order int
}

func extractOCRKeywords(recent []store.ObservationHit, limit int) []string {
	known := []string{"Sipeed", "NanoKVM", "MaixCAM", "MaixZero", "LicheePi", "USB", "RJ45", "OLED", "PoE", "ATX", "电源", "串口"}
	scores := map[string]*keywordScore{}
	order := 0
	add := func(term string, count int) {
		term = strings.TrimSpace(term)
		if term == "" || count <= 0 {
			return
		}
		key := strings.ToLower(term)
		if existing, ok := scores[key]; ok {
			existing.count += count
			return
		}
		scores[key] = &keywordScore{term: term, count: count, order: order}
		order++
	}

	for _, h := range recent {
		text := h.AppName + "\n" + h.Title + "\n" + h.OCRText + "\n" + h.Snippet
		lower := strings.ToLower(text)
		for _, term := range known {
			count := strings.Count(lower, strings.ToLower(term))
			add(term, count)
		}
		for _, token := range strings.FieldsFunc(text, func(r rune) bool {
			return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_')
		}) {
			token = strings.Trim(token, "-_")
			if len([]rune(token)) >= 4 {
				add(token, 1)
			}
		}
	}

	items := make([]keywordScore, 0, len(scores))
	for _, score := range scores {
		items = append(items, *score)
	}
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].count > items[i].count || items[j].count == items[i].count && items[j].order < items[i].order {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
	out := make([]string, 0, limit)
	for _, item := range items {
		out = append(out, item.term)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func makeObservationHeadline(apps, titles, keywords []string) string {
	parts := []string{}
	if len(titles) > 0 {
		parts = append(parts, titles[0])
	} else if len(apps) > 0 {
		parts = append(parts, apps[0])
	}
	if len(keywords) > 0 {
		max := len(keywords)
		if max > 3 {
			max = 3
		}
		parts = append(parts, strings.Join(keywords[:max], ", "))
	}
	if len(parts) == 0 {
		return "Recent OCR observations are available"
	}
	return preview("Recent OCR shows "+strings.Join(parts, " with "), 220)
}

func preview(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
