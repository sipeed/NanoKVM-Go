package reducer

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
)

var callJSON = llm.CallJSON

var retryBackoffMinutes = []int{5, 15, 30, 60, 120}

type Result struct {
	SessionID string
	Reduced   bool
	Written   bool
	Heuristic bool
	Failed    bool
	EntryID   string
	Path      string
	Error     string
}

type Summary struct {
	Reduced    int
	Failed     int
	Written    int
	Heuristic  int
	Classified int
}

type llmOutput struct {
	Summary  string   `json:"summary"`
	SubTasks []string `json:"sub_tasks"`
}

func TickDue(ctx context.Context, db *store.DB, cfg config.Config, now time.Time, limit int) (Summary, error) {
	var summary Summary
	if !cfg.Reducer.Enabled {
		return summary, nil
	}
	rows, err := db.ListDueReductions(now.Format(time.RFC3339), limit)
	if err != nil {
		return summary, err
	}
	for _, row := range rows {
		result, err := ReduceSession(ctx, db, cfg, row, now)
		if err != nil {
			return summary, err
		}
		if result.Reduced {
			summary.Reduced++
		}
		if result.Failed {
			summary.Failed++
		}
		if result.Written {
			summary.Written++
		}
		if result.Heuristic {
			summary.Heuristic++
		}
	}
	return summary, nil
}

func RunForever(ctx context.Context, db *store.DB, cfg config.Config) {
	tick := func(now time.Time) {
		summary, err := TickDue(ctx, db, cfg, now, 10)
		if err != nil {
			log.Printf("reducer tick failed: %v", err)
			return
		}
		if summary.Reduced > 0 || summary.Failed > 0 {
			log.Printf("reducer tick reduced=%d failed=%d written=%d heuristic=%d", summary.Reduced, summary.Failed, summary.Written, summary.Heuristic)
		}
	}
	tick(time.Now())
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			tick(now)
		}
	}
}

func ReduceSession(ctx context.Context, db *store.DB, cfg config.Config, row store.SessionRow, now time.Time) (Result, error) {
	result := Result{SessionID: row.ID}
	if !cfg.Reducer.Enabled || row.Status == string(store.SessionReduced) {
		return result, nil
	}
	if row.EndTime == "" {
		return result, nil
	}
	start := row.StartTime
	if row.FlushEnd != "" && after(row.FlushEnd, start) {
		start = row.FlushEnd
	}
	blocks, err := db.TimelineBlocksIntersecting(start, row.EndTime, 100)
	if err != nil {
		return result, err
	}
	if len(blocks) == 0 {
		if err := db.MarkSessionReduced(row.ID, row.EndTime, now.Format(time.RFC3339)); err != nil {
			return result, err
		}
		result.Reduced = true
		return result, nil
	}
	startTime, err := capture.ParseTimestamp(row.StartTime)
	if err != nil {
		return result, err
	}
	endTime, err := capture.ParseTimestamp(row.EndTime)
	if err != nil {
		return result, err
	}
	output, llmErr := reduceWithLLM(ctx, cfg, row, blocks, recentEventContext(startTime))
	if llmErr != nil {
		if row.RetryCount < len(retryBackoffMinutes) {
			next := now.Add(time.Duration(retryBackoffMinutes[row.RetryCount]) * time.Minute).Format(time.RFC3339)
			if err := db.MarkSessionFailed(row.ID, llmErr.Error(), next, now.Format(time.RFC3339), row.RetryCount+1); err != nil {
				return result, err
			}
			result.Failed = true
			result.Error = llmErr.Error()
			return result, nil
		}
		output = heuristicOutput(blocks)
		result.Heuristic = true
	}
	entryID, path, err := writeEventEntry(db, output, startTime, endTime, row.ID, now, result.Heuristic, nil)
	if err != nil {
		return result, err
	}
	if err := db.MarkSessionReduced(row.ID, row.EndTime, now.Format(time.RFC3339)); err != nil {
		return result, err
	}
	result.Reduced = true
	result.Written = true
	result.EntryID = entryID
	result.Path = path
	return result, nil
}

func FlushActive(ctx context.Context, db *store.DB, cfg config.Config, now time.Time) (Result, error) {
	result := Result{}
	if !cfg.Reducer.Enabled {
		return result, nil
	}
	row, err := db.GetActiveSession()
	if err != nil || row == nil {
		return result, err
	}
	result.SessionID = row.ID
	start := row.StartTime
	if row.FlushEnd != "" && after(row.FlushEnd, start) {
		start = row.FlushEnd
	}
	end := now.Format(time.RFC3339)
	blocks, err := db.TimelineBlocksIntersecting(start, end, 100)
	if err != nil {
		return result, err
	}
	if len(blocks) == 0 {
		return result, nil
	}
	startTime, err := capture.ParseTimestamp(start)
	if err != nil {
		return result, err
	}
	output, err := reduceWithLLM(ctx, cfg, store.SessionRow{ID: row.ID, StartTime: start, EndTime: end}, blocks, recentEventContext(startTime))
	if err != nil {
		result.Failed = true
		result.Error = err.Error()
		return result, nil
	}
	entryID, path, err := writeEventEntry(db, output, startTime, now, row.ID, now, false, []string{"flush"})
	if err != nil {
		return result, err
	}
	if err := db.SetSessionFlushEnd(row.ID, end, now.Format(time.RFC3339)); err != nil {
		return result, err
	}
	result.Written = true
	result.EntryID = entryID
	result.Path = path
	return result, nil
}

func writeEventEntry(db *store.DB, output llmOutput, start, end time.Time, sessionID string, now time.Time, heuristic bool, extraTags []string) (string, string, error) {
	entryID := makeEntryID(start, sessionID)
	path := memory.EventDailyName(start)
	body := renderEventBody(output, start, end, sessionID, heuristic)
	entry := memory.ParsedEntry{ID: entryID, Timestamp: end.Format(time.RFC3339), Tags: []string{"event", "session", "sid:" + sessionID}, Body: body}
	entry.Tags = append(entry.Tags, extraTags...)
	if heuristic {
		entry.Tags = append(entry.Tags, "heuristic")
	}
	parsed, err := memory.EnsureEventDailyFile(paths.MemoryDir(), path, start, now)
	if err != nil {
		return "", "", err
	}
	parsed, err = memory.AppendEntry(paths.MemoryDir(), path, entry, now)
	if err != nil {
		return "", "", err
	}
	if err := db.UpsertMemoryFile(store.FileRow{Path: parsed.Path, Prefix: "event", Description: parsed.Description, Tags: parsed.Tags, Status: parsed.Status, EntryCount: parsed.EntryCount, Created: parsed.Created, Updated: parsed.Updated, NeedsCompact: parsed.NeedsCompact}); err != nil {
		return "", "", err
	}
	if err := db.InsertMemoryEntry(store.MemoryEntryRow{ID: entry.ID, Path: parsed.Path, Prefix: "event", Timestamp: entry.Timestamp, Tags: strings.Join(entry.Tags, " "), Content: entry.Body}); err != nil {
		return "", "", err
	}
	return entryID, path, nil
}

func reduceWithLLM(ctx context.Context, cfg config.Config, row store.SessionRow, blocks []store.TimelineBlock, recent string) (llmOutput, error) {
	model := cfg.Models.Reducer
	if strings.TrimSpace(model.Model) == "" {
		return llmOutput{}, fmt.Errorf("reducer llm is not configured")
	}
	out, err := callJSON(ctx, model, []llm.Message{{Role: "user", Content: renderPrompt(row, blocks, recent)}})
	if err != nil {
		return llmOutput{}, err
	}
	var parsed llmOutput
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return llmOutput{}, err
	}
	parsed.Summary = strings.TrimSpace(parsed.Summary)
	parsed.SubTasks = cleanStrings(parsed.SubTasks)
	if parsed.Summary == "" || len(parsed.SubTasks) == 0 {
		return llmOutput{}, fmt.Errorf("reducer llm returned empty output")
	}
	return parsed, nil
}

func renderPrompt(row store.SessionRow, blocks []store.TimelineBlock, recent string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Reduce this memfab session into event memory JSON.\nSession: %s to %s, id %s.\n", row.StartTime, row.EndTime, row.ID)
	if strings.TrimSpace(recent) != "" {
		fmt.Fprintf(&b, "Recent event context:\n%s\n", recent)
	}
	b.WriteString("Timeline blocks:\n")
	for _, block := range blocks {
		fmt.Fprintf(&b, "- %s to %s apps=%s\n", block.StartTime, block.EndTime, strings.Join(block.AppsUsed, ", "))
		for _, entry := range block.Entries {
			fmt.Fprintf(&b, "  * %s\n", entry)
		}
	}
	b.WriteString("Return only JSON: {\"summary\": string, \"sub_tasks\": [string, ...]}. Preserve URLs, filenames, proper nouns, and quoted evidence.")
	return b.String()
}

func heuristicOutput(blocks []store.TimelineBlock) llmOutput {
	apps := map[string]bool{}
	subtasks := make([]string, 0, len(blocks))
	for _, block := range blocks {
		for _, app := range block.AppsUsed {
			apps[app] = true
		}
		if len(block.Entries) > 0 {
			subtasks = append(subtasks, fmt.Sprintf("[%s-%s] %s", hm(block.StartTime), hm(block.EndTime), strings.Join(block.Entries, "; ")))
		}
	}
	if len(subtasks) == 0 {
		subtasks = append(subtasks, "No detailed timeline entries were available.")
	}
	appList := make([]string, 0, len(apps))
	for app := range apps {
		if app != "" {
			appList = append(appList, app)
		}
	}
	summary := "Reduced session from timeline blocks"
	if len(appList) > 0 {
		summary += " involving " + strings.Join(appList, ", ")
	}
	return llmOutput{Summary: summary + ".", SubTasks: subtasks}
}

func renderEventBody(output llmOutput, start, end time.Time, sessionID string, heuristic bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Session: %s-%s (%s)\n\n", start.Format("15:04"), end.Format("15:04"), sessionID)
	fmt.Fprintf(&b, "Summary: %s\n\n", output.Summary)
	b.WriteString("Sub-tasks:\n")
	for _, task := range output.SubTasks {
		fmt.Fprintf(&b, "- %s\n", task)
	}
	if heuristic {
		b.WriteString("\nReducer note: heuristic fallback after retry exhaustion.\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func recentEventContext(day time.Time) string {
	path, err := memory.MemoryPath(paths.MemoryDir(), memory.EventDailyName(day))
	if err != nil {
		return ""
	}
	parsed, err := memory.ReadFile(path)
	if err != nil || len(parsed.Entries) == 0 {
		return ""
	}
	entries := parsed.Entries
	if len(entries) > 6 {
		entries = entries[len(entries)-6:]
	}
	parts := make([]string, 0, len(entries))
	for _, entry := range entries {
		parts = append(parts, fmt.Sprintf("[%s] %s", entry.Timestamp, entry.Body))
	}
	return strings.Join(parts, "\n")
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func after(a, b string) bool {
	ta, errA := capture.ParseTimestamp(a)
	tb, errB := capture.ParseTimestamp(b)
	return errA == nil && errB == nil && ta.After(tb)
}

func makeEntryID(start time.Time, sessionID string) string {
	sum := sha1.Sum([]byte(start.Format(time.RFC3339Nano) + sessionID))
	return start.Format("20060102-1504-") + hex.EncodeToString(sum[:3])
}

func hm(ts string) string {
	parsed, err := capture.ParseTimestamp(ts)
	if err != nil {
		return ts
	}
	return parsed.Format("15:04")
}
