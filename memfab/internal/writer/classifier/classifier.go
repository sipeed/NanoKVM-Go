package classifier

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/llm"
	"memfab/internal/memory"
	"memfab/internal/paths"
	"memfab/internal/store"
	wtools "memfab/internal/writer/tools"
)

var callTools = llm.CallTools

var retryBackoffMinutes = []int{5, 15, 30, 60, 120}

type Result struct {
	SessionID     string
	Committed     bool
	WrittenIDs    []string
	CreatedPaths  []string
	Iterations    int
	SkippedReason string
	Failed        bool
	Error         string
}

type Summary struct {
	Classified int
	Failed     int
	Written    int
	Skipped    int
}

func TickDue(ctx context.Context, db *store.DB, cfg config.Config, now time.Time, limit int) (Summary, error) {
	var summary Summary
	if !cfg.Classifier.Enabled {
		return summary, nil
	}
	if limit <= 0 {
		limit = cfg.Classifier.MaxSessionsPerTick
	}
	if limit <= 0 {
		limit = 4
	}
	rows, err := db.ListSessionsNeedingClassification(now.Format(time.RFC3339), limit)
	if err != nil {
		return summary, err
	}
	for _, row := range rows {
		result, err := ClassifySession(ctx, db, cfg, row, now)
		if err != nil {
			return summary, err
		}
		if result.Failed {
			summary.Failed++
		}
		if result.Committed {
			summary.Classified++
		}
		if len(result.WrittenIDs) > 0 || len(result.CreatedPaths) > 0 {
			summary.Written++
		}
		if result.SkippedReason != "" {
			summary.Skipped++
		}
	}
	return summary, nil
}

func ClassifySession(ctx context.Context, db *store.DB, cfg config.Config, row store.SessionRow, now time.Time) (Result, error) {
	result := Result{SessionID: row.ID}
	if !cfg.Classifier.Enabled || row.Status != string(store.SessionReduced) {
		return result, nil
	}
	targetEnd := classificationTargetEnd(row)
	if targetEnd == "" {
		result.SkippedReason = "session has no reduced end"
		return result, nil
	}
	windowStart := row.StartTime
	if row.ClassifiedEnd != "" && after(row.ClassifiedEnd, windowStart) {
		windowStart = row.ClassifiedEnd
	}
	startTime, err := capture.ParseTimestamp(windowStart)
	if err != nil {
		return result, err
	}
	endTime, err := capture.ParseTimestamp(targetEnd)
	if err != nil {
		return result, err
	}
	if !endTime.After(startTime) && !endTime.Equal(startTime) {
		result.SkippedReason = "classification window already complete"
		return result, nil
	}
	eventName := memory.EventDailyName(startTime)
	entries, err := focusEntries(eventName, row.ID, startTime, endTime)
	if err != nil {
		return result, err
	}
	if len(entries) == 0 {
		if err := db.SetSessionClassifiedEnd(row.ID, targetEnd, now.Format(time.RFC3339)); err != nil {
			return result, err
		}
		result.Committed = true
		result.SkippedReason = "no session entries in window"
		return result, nil
	}
	blocks, err := db.TimelineBlocksIntersecting(windowStart, targetEnd, 100)
	if err != nil {
		return result, err
	}
	committed, state, iterations, err := runToolLoop(ctx, db, cfg, row.ID, eventName, entries, blocks, renderPriorDay(startTime), now)
	result.Iterations = iterations
	if err != nil || !committed {
		message := "classifier did not commit"
		if err != nil {
			message = err.Error()
		}
		nextRetry := ""
		if row.ClassifierRetryCount < len(retryBackoffMinutes) {
			nextRetry = now.Add(time.Duration(retryBackoffMinutes[row.ClassifierRetryCount]) * time.Minute).Format(time.RFC3339)
		}
		if err := db.MarkSessionClassificationFailed(row.ID, message, nextRetry, now.Format(time.RFC3339), row.ClassifierRetryCount+1); err != nil {
			return result, err
		}
		result.Failed = true
		result.Error = message
		return result, nil
	}
	if err := db.SetSessionClassifiedEnd(row.ID, targetEnd, now.Format(time.RFC3339)); err != nil {
		return result, err
	}
	result.Committed = true
	result.WrittenIDs = append(result.WrittenIDs, state.WrittenIDs...)
	result.CreatedPaths = append(result.CreatedPaths, state.CreatedPaths...)
	return result, nil
}

func runToolLoop(ctx context.Context, db *store.DB, cfg config.Config, sessionID, eventName string, entries []memory.ParsedEntry, blocks []store.TimelineBlock, priorDay string, now time.Time) (bool, wtools.CommitState, int, error) {
	model := cfg.Models.Classifier
	if strings.TrimSpace(model.Model) == "" {
		return false, wtools.CommitState{}, 0, fmt.Errorf("classifier llm is not configured")
	}
	maxIterations := cfg.Classifier.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 8
	}
	state := &wtools.CommitState{}
	messages := []llm.Message{
		{Role: "system", Content: classifierPrompt()},
		{Role: "user", Content: renderUserPrompt(db, sessionID, eventName, entries, blocks, priorDay)},
	}
	dispatcher := wtools.Dispatcher{DB: db, MemoryDir: paths.MemoryDir(), SoftLimitTokens: cfg.Writer.SoftLimitTokens, Now: now, State: state}
	tools := wtools.ToolDefinitions()
	for iteration := 0; iteration < maxIterations; iteration++ {
		resp, err := callTools(ctx, model, messages, tools)
		if err != nil {
			return false, *state, iteration + 1, err
		}
		assistant := llm.Message{Role: "assistant", Content: resp.Content, ToolCalls: resp.ToolCalls}
		messages = append(messages, assistant)
		if len(resp.ToolCalls) == 0 {
			return false, *state, iteration + 1, nil
		}
		for _, call := range resp.ToolCalls {
			out := dispatcher.Dispatch(ctx, call.Function.Name, call.Function.Arguments)
			data, _ := json.Marshal(out)
			messages = append(messages, llm.Message{Role: "tool", Name: call.Function.Name, ToolCallID: call.ID, Content: string(data)})
		}
		if state.Committed {
			return true, *state, iteration + 1, nil
		}
	}
	return false, *state, maxIterations, nil
}

func focusEntries(eventName, sessionID string, start, end time.Time) ([]memory.ParsedEntry, error) {
	path, err := memory.MemoryPath(paths.MemoryDir(), eventName)
	if err != nil {
		return nil, err
	}
	parsed, err := memory.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sid := "sid:" + sessionID
	var allSID []memory.ParsedEntry
	var matches []memory.ParsedEntry
	for _, entry := range parsed.Entries {
		if !hasTag(entry.Tags, sid) {
			continue
		}
		allSID = append(allSID, entry)
		ts, err := capture.ParseTimestamp(entry.Timestamp)
		if err != nil || (!ts.Before(start) && !ts.After(end)) {
			matches = append(matches, entry)
		}
	}
	if len(matches) == 0 && len(allSID) > 0 {
		return allSID, nil
	}
	return matches, nil
}

func renderUserPrompt(db *store.DB, sessionID, eventName string, entries []memory.ParsedEntry, blocks []store.TimelineBlock, priorDay string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Schema\n\n%s\n\n", memory.SchemaText())
	fmt.Fprintf(&b, "# Memory index\n\n%s\n\n", renderIndex(db))
	fmt.Fprintf(&b, "# Event-daily context\n\nSource file: %s\n\n", eventName)
	b.WriteString("## Session entries (focus — classify these)\n")
	for _, entry := range entries {
		fmt.Fprintf(&b, "### [%s] {id: %s}\n%s\n\n", entry.Timestamp, entry.ID, strings.TrimSpace(entry.Body))
	}
	b.WriteString("## Timeline blocks covering this session\n")
	b.WriteString("These are the verbatim-preserving activity slices the reducer compressed. Use them to ground durable facts.\n\n")
	if len(blocks) == 0 {
		b.WriteString("(no timeline blocks recorded for this session)\n\n")
	} else {
		for _, block := range blocks {
			fmt.Fprintf(&b, "[%s-%s] apps=%s\n", hm(block.StartTime), hm(block.EndTime), strings.Join(block.AppsUsed, ", "))
			for _, entry := range block.Entries {
				fmt.Fprintf(&b, "  - %s\n", entry)
			}
		}
		b.WriteString("\n")
	}
	if strings.TrimSpace(priorDay) != "" {
		fmt.Fprintf(&b, "## Preceding day (context, dedup anchor)\n\n%s\n\n", priorDay)
	}
	fmt.Fprintf(&b, "Source file (do NOT write to it): %s\nSession being classified: %s\n", eventName, sessionID)
	return b.String()
}

func renderIndex(db *store.DB) string {
	files, err := db.ListMemoryFiles(false, false)
	if err != nil || len(files) == 0 {
		return "(no non-event memory files yet — create them as needed)"
	}
	lines := []string{"Active non-event memory files:"}
	for _, file := range files {
		if strings.HasPrefix(file.Path, "event-") {
			continue
		}
		lines = append(lines, fmt.Sprintf("- %s  # %s  (tags: %s; entries: %d; updated: %s)", file.Path, file.Description, strings.Join(file.Tags, ", "), file.EntryCount, file.Updated))
		if len(lines) >= 31 {
			break
		}
	}
	if len(lines) == 1 {
		return "(no non-event memory files yet — create them as needed)"
	}
	return strings.Join(lines, "\n")
}

func renderPriorDay(start time.Time) string {
	name := memory.EventDailyName(start.AddDate(0, 0, -1))
	path, err := memory.MemoryPath(paths.MemoryDir(), name)
	if err != nil {
		return ""
	}
	parsed, err := memory.ReadFile(path)
	if err != nil || len(parsed.Entries) == 0 {
		return ""
	}
	entries := parsed.Entries
	if len(entries) > 8 {
		entries = entries[len(entries)-8:]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "From %s (last %d entries):\n\n", name, len(entries))
	for _, entry := range entries {
		fmt.Fprintf(&b, "### [%s] {id: %s}\n%s\n\n", entry.Timestamp, entry.ID, strings.TrimSpace(entry.Body))
	}
	return strings.TrimSpace(b.String())
}

func classifierPrompt() string {
	for _, candidate := range []string{
		"src/memfab/prompts/classifier.md",
		filepath.Join("..", "..", "src", "memfab", "prompts", "classifier.md"),
	} {
		if data, err := os.ReadFile(candidate); err == nil {
			return string(data)
		}
	}
	return strings.TrimSpace(`You are the Classifier module of memfab. Extract only durable long-term facts from event memory into non-event memory files. Event-daily files are owned by the reducer; never write to event-*.md. The default action is to write nothing. Use search_memory before appending, read_memory before updating an existing file, and call commit exactly once at the end. If in doubt, skip.`)
}

func classificationTargetEnd(row store.SessionRow) string {
	if row.FlushEnd != "" {
		return row.FlushEnd
	}
	return row.EndTime
}

func hasTag(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func after(a, b string) bool {
	ta, errA := capture.ParseTimestamp(a)
	tb, errB := capture.ParseTimestamp(b)
	return errA == nil && errB == nil && ta.After(tb)
}

func hm(ts string) string {
	parsed, err := capture.ParseTimestamp(ts)
	if err != nil {
		return ts
	}
	return parsed.Format("15:04")
}
