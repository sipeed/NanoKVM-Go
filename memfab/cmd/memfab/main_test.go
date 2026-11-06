package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/paths"
	ocReplay "memfab/internal/replay"
	"memfab/internal/store"
)

func TestToolBackendIgnoresLegacyPublicMCPConfig(t *testing.T) {
	setupCLITestRoot(t)
	socketPath := filepath.Join(t.TempDir(), "runtime", "tools.sock")
	t.Setenv("MEMFAB_TOOLS_SOCKET", socketPath)
	db := openCLITestDB(t)

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("sandbox does not permit socket creation: %v", err)
		}
		t.Fatal(err)
	}
	legacyAddr := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatal(err)
	}
	_, portText, err := net.SplitHostPort(legacyAddr)
	if err != nil {
		t.Fatal(err)
	}
	var legacyPort int
	if _, err := fmt.Sscanf(portText, "%d", &legacyPort); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.MCP.AutoStart = false
	cfg.MCP.Transport = "streamable-http"
	cfg.MCP.Host = "0.0.0.0"
	cfg.MCP.Port = legacyPort
	server, err := listenToolBackend(db, cfg)
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("sandbox does not permit socket creation: %v", err)
		}
		t.Fatal(err)
	}
	defer server.Close()
	if server.SocketPath() != socketPath {
		t.Fatalf("socket=%q, want %q", server.SocketPath(), socketPath)
	}

	tcpListener, err := net.Listen("tcp", legacyAddr)
	if err != nil {
		t.Fatalf("legacy TCP address was unexpectedly opened: %v", err)
	}
	_ = tcpListener.Close()
}

func TestStatusReportsOnlyInternalToolBackend(t *testing.T) {
	setupCLITestRoot(t)
	socketPath := filepath.Join(t.TempDir(), "runtime", "tools.sock")
	t.Setenv("MEMFAB_TOOLS_SOCKET", socketPath)

	out, err := captureRun(t, "status", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "MCP backend") || !strings.Contains(out, "unix://"+socketPath) {
		t.Fatalf("status missing internal backend:\n%s", out)
	}
	for _, forbidden := range []string{"0.0.0.0", ":8742", "streamable-http"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("status advertises retired %q:\n%s", forbidden, out)
		}
	}
}

func TestSystemdUnitSecuresToolRuntimeDirectory(t *testing.T) {
	paths := []string{
		"../../packaging/systemd/memfab.service",
		"../../../debhelper/packages/nanokvm-go/debian/memfab.service",
	}
	var canonical []byte
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		unit := string(data)
		for _, want := range []string{"User=root", "Group=root", "UMask=0077", "RuntimeDirectory=memfab", "RuntimeDirectoryMode=0700", "ExecStart=/usr/bin/memfab start --foreground"} {
			if !strings.Contains(unit, want) {
				t.Fatalf("systemd unit %s missing %q:\n%s", path, want, unit)
			}
		}
		if canonical == nil {
			canonical = data
			continue
		}
		if string(data) != string(canonical) {
			t.Fatalf("systemd unit %s drifted from %s", path, paths[0])
		}
	}
}

func TestIngestHTTPServerHasFiniteResourceBounds(t *testing.T) {
	handler := http.NewServeMux()
	server := newIngestHTTPServer("127.0.0.1:8743", handler)
	if server.Addr != "127.0.0.1:8743" || server.Handler != handler {
		t.Fatalf("server wiring changed: addr=%q handler=%T", server.Addr, server.Handler)
	}
	if server.ReadHeaderTimeout != ingestReadHeaderTimeout || server.ReadTimeout != ingestReadTimeout || server.WriteTimeout != ingestWriteTimeout || server.IdleTimeout != ingestIdleTimeout {
		t.Fatalf("unexpected ingest timeouts: header=%s read=%s write=%s idle=%s", server.ReadHeaderTimeout, server.ReadTimeout, server.WriteTimeout, server.IdleTimeout)
	}
	if server.MaxHeaderBytes != ingestMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes=%d, want %d", server.MaxHeaderBytes, ingestMaxHeaderBytes)
	}
	if server.ReadTimeout <= 0 || server.WriteTimeout <= 0 || server.IdleTimeout <= 0 || server.MaxHeaderBytes <= 0 {
		t.Fatal("ingest server has an unbounded resource setting")
	}
}

func TestReplaySearchReturnsTimesFromOCRText(t *testing.T) {
	setupCLITestRoot(t)
	db := openCLITestDB(t)
	insertCLIObservation(t, db, "2026-05-19T10:00:00Z", "ScreenOCR", "Plain", "PicoClaw appears in OCR text", "session-id:42")
	insertCLIObservation(t, db, "2026-05-19T10:01:00Z", "ScreenOCR", "PicoClaw title only", "ordinary OCR text")

	out, err := captureRun(t, "replay", []string{"search", "--query", "PicoClaw"})
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeReplaySearchOutput(t, out)
	if decoded.Count != 1 || len(decoded.Results) != 1 {
		t.Fatalf("decoded=%+v", decoded)
	}
	if decoded.Results[0].CapturedAt != "2026-05-19T10:00:00Z" {
		t.Fatalf("hit=%+v", decoded.Results[0])
	}
	if decoded.Results[0].CaptureID != "session-id:42" || !strings.Contains(out, `"capture_id": "session-id:42"`) {
		t.Fatalf("capture_id not present in CLI output: %+v\n%s", decoded.Results[0], out)
	}
	if !strings.Contains(decoded.Results[0].OCRPreview, "[PicoClaw]") {
		t.Fatalf("preview=%q", decoded.Results[0].OCRPreview)
	}
}

func TestReplaySearchNoMatchesUsesEmptyResults(t *testing.T) {
	setupCLITestRoot(t)
	openCLITestDB(t)

	out, err := captureRun(t, "replay", []string{"search", "--query", "missing"})
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeReplaySearchOutput(t, out)
	if decoded.Count != 0 || len(decoded.Results) != 0 {
		t.Fatalf("decoded=%+v", decoded)
	}
	if !strings.Contains(out, `"results": []`) {
		t.Fatalf("results should encode as empty array:\n%s", out)
	}
}

func TestReplaySearchHonorsTimeRangeAndLimit(t *testing.T) {
	setupCLITestRoot(t)
	db := openCLITestDB(t)
	insertCLIObservation(t, db, "2026-05-19T10:00:00Z", "ScreenOCR", "First", "target")
	insertCLIObservation(t, db, "2026-05-19T10:01:00Z", "ScreenOCR", "Second", "target")

	out, err := captureRun(t, "replay", []string{
		"search",
		"--query", "target",
		"--since", "2026-05-19T10:00:30Z",
		"--until", "2026-05-19T10:02:00Z",
		"--limit", "1",
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded := decodeReplaySearchOutput(t, out)
	if decoded.Count != 1 || decoded.Results[0].CapturedAt != "2026-05-19T10:01:00Z" {
		t.Fatalf("decoded=%+v", decoded)
	}
}

func TestReplaySearchValidationAndRemovedCommands(t *testing.T) {
	setupCLITestRoot(t)
	if _, err := captureRun(t, "replay", nil); err == nil || !strings.Contains(err.Error(), "replay requires subcommand: search") {
		t.Fatalf("err=%v, want search subcommand error", err)
	}
	if _, err := captureRun(t, "replay", []string{"search"}); err == nil || !strings.Contains(err.Error(), "query is required") {
		t.Fatalf("err=%v, want query required", err)
	}
	for _, cmd := range []string{"frames", "timelapse", "clean", "decode-smoke", "encode-smoke"} {
		_, err := captureRun(t, "replay", []string{cmd})
		if err == nil || !strings.Contains(err.Error(), "unknown replay command") {
			t.Fatalf("%s err=%v, want unknown command", cmd, err)
		}
	}
}

func TestReplaySearchDisabled(t *testing.T) {
	setupCLITestRoot(t)
	if err := os.WriteFile(paths.ConfigFile(), []byte("[replay]\nenabled = false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := captureRun(t, "replay", []string{"search", "--query", "PicoClaw"})
	if err == nil || !strings.Contains(err.Error(), "replay is disabled") {
		t.Fatalf("err=%v, want replay disabled", err)
	}
}

func TestCleanupNoLongerRunsReplayCleanup(t *testing.T) {
	setupCLITestRoot(t)
	openCLITestDB(t)

	out, err := captureRun(t, "cleanup", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "cleanup capture noop: no timeline blocks yet") || strings.Contains(out, "replay_cleanup") {
		t.Fatalf("unexpected cleanup output:\n%s", out)
	}
}

func TestResetCommandClearsMemfabData(t *testing.T) {
	root := setupCLITestRoot(t)
	db := openCLITestDB(t)
	touchCLITestFile(t, filepath.Join(paths.MemoryDir(), "project-cli-reset.md"))
	touchCLITestFile(t, paths.WriterState())
	screenshotPath(t, root, "reset.jpg")
	insertCLIObservation(t, db, "2026-06-18T10:00:00Z", "ScreenOCR", "Reset", "cli reset target")
	if _, err := db.InsertTimelineBlock(store.TimelineBlock{ID: "tl-cli-reset", StartTime: "2026-06-18T10:00:00Z", EndTime: "2026-06-18T10:01:00Z", Entries: []string{"[ScreenOCR] reset"}, AppsUsed: []string{"ScreenOCR"}, CaptureCount: 1, CreatedAt: "2026-06-18T10:01:00Z"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.InsertSession(store.SessionRow{ID: "sess-cli-reset", StartTime: "2026-06-18T10:00:00Z", EndTime: "2026-06-18T10:01:00Z", Status: string(store.SessionEnded), CreatedAt: "2026-06-18T10:01:00Z", UpdatedAt: "2026-06-18T10:01:00Z"}); err != nil {
		t.Fatal(err)
	}

	out, err := captureRun(t, "reset", nil)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		OK    bool `json:"ok"`
		Reset struct {
			DeletedMemoryFiles    int  `json:"deleted_memory_files"`
			DeletedObservations   int  `json:"deleted_observations"`
			DeletedScreenshots    int  `json:"deleted_screenshots"`
			DeletedTimelineBlocks int  `json:"deleted_timeline_blocks"`
			DeletedSessions       int  `json:"deleted_sessions"`
			DeletedWriterState    bool `json:"deleted_writer_state"`
		} `json:"reset"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("decode reset output: %v\n%s", err, out)
	}
	if !decoded.OK || decoded.Reset.DeletedMemoryFiles != 1 || decoded.Reset.DeletedObservations != 1 || decoded.Reset.DeletedScreenshots != 1 || decoded.Reset.DeletedTimelineBlocks != 1 || decoded.Reset.DeletedSessions != 1 || !decoded.Reset.DeletedWriterState {
		t.Fatalf("decoded=%+v output=%s", decoded, out)
	}
	for _, table := range []string{"ocr_observations", "timeline_blocks", "sessions"} {
		if got := count(db, table); got != 0 {
			t.Fatalf("%s rows=%d, want 0", table, got)
		}
	}
	if _, err := os.Stat(paths.WriterState()); !os.IsNotExist(err) {
		t.Fatalf("writer state exists or unexpected err=%v", err)
	}
}

func setupCLITestRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	return root
}

func openCLITestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func insertCLIObservation(t *testing.T, db *store.DB, ts, app, title, text string, captureID ...string) string {
	t.Helper()
	id := "test:" + ts
	if len(captureID) > 0 {
		id = captureID[0]
	}
	o := capture.Observation{
		SchemaVersion: capture.SchemaVersion,
		Type:          capture.ObservationType,
		CapturedAt:    ts,
		CaptureID:     id,
		Source:        capture.Source{Kind: "screen_ocr", Device: "test"},
		Context:       capture.Context{AppName: app, Title: title},
		Event:         capture.Event{Type: "ocr_capture", Reason: "test"},
		OCR:           capture.OCR{Text: text},
		Tags:          []string{"ocr"},
	}
	if err := o.Normalize(nil); err != nil {
		t.Fatal(err)
	}
	stem, err := capture.SafeFilename(o.CapturedAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.InsertObservation(stem, o); err != nil {
		t.Fatal(err)
	}
	return stem
}

func screenshotPath(t *testing.T, root, name string) string {
	t.Helper()
	return touchCLITestFile(t, filepath.Join(root, "screenshots", "2026-05-19", name))
}

func touchCLITestFile(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("jpeg"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func captureRun(t *testing.T, cmd string, args []string) (string, error) {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	runErr := run(cmd, args)
	_ = w.Close()
	os.Stdout = oldStdout
	out, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(out), runErr
}

func decodeReplaySearchOutput(t *testing.T, text string) ocReplay.SearchResult {
	t.Helper()
	var out ocReplay.SearchResult
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("decode output: %v\n%s", err, text)
	}
	return out
}
