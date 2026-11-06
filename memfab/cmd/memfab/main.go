package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"memfab/internal/capture"
	"memfab/internal/config"
	"memfab/internal/daemon"
	"memfab/internal/ingest"
	"memfab/internal/mcp"
	"memfab/internal/paths"
	ocReset "memfab/internal/reset"
	ocSession "memfab/internal/session"
	"memfab/internal/store"
	ocTimeline "memfab/internal/timeline"
)

var version = "0.1.0-memfab"

const (
	ingestReadHeaderTimeout = 5 * time.Second
	ingestReadTimeout       = 15 * time.Second
	ingestWriteTimeout      = 30 * time.Second
	ingestIdleTimeout       = 60 * time.Second
	ingestMaxHeaderBytes    = 16 << 10
)

func main() {
	// Keep every daemon and CLI-created data file private even when SQLite or a
	// dependency chooses its creation mode internally.
	syscall.Umask(0o077)
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	if err := run(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(cmd string, args []string) error {
	switch cmd {
	case "config":
		return configCommand(args, os.Stdin, os.Stdout)
	case "status":
		return status()
	case "ingest-observation":
		return ingestObservation(args)
	case "rebuild-observations-index":
		return rebuildObservationsIndex()
	case "start":
		fs := flag.NewFlagSet("start", flag.ExitOnError)
		foreground := fs.Bool("foreground", false, "run in foreground")
		fs.BoolVar(foreground, "f", false, "run in foreground")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if !*foreground {
			return errors.New("background start is not implemented in Go milestone 1; use start --foreground or systemd")
		}
		return startForeground()
	case "stop":
		return stop()
	case "pause":
		if err := paths.EnsureDirs(); err != nil {
			return err
		}
		return paths.WritePrivateFile(paths.PausedFlag(), []byte(time.Now().Format(time.RFC3339)))
	case "resume":
		if err := os.Remove(paths.PausedFlag()); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	case "mcp":
		return serveMCP()
	case "timeline":
		return timeline(args)
	case "session":
		return session(args)
	case "writer":
		return writer(args)
	case "replay":
		return replay(args)
	case "rebuild-index":
		return rebuildIndex()
	case "cleanup":
		return cleanup()
	case "reset":
		return resetData()
	default:
		usage()
		return fmt.Errorf("unknown command %s", cmd)
	}
}

func usage() {
	fmt.Println("memfab Go")
	fmt.Println("usage: memfab <command> [args]")
	fmt.Println("commands: start --foreground, stop, pause, resume, status, config [get|update|revision|restore], ingest-observation, rebuild-observations-index, rebuild-index, cleanup, reset, timeline, session, writer, replay, mcp")
}

func loadDB() (*store.DB, config.Config, error) {
	if err := paths.EnsureDirs(); err != nil {
		return nil, config.Config{}, err
	}
	if _, err := config.WriteDefaultIfMissing(); err != nil {
		return nil, config.Config{}, err
	}
	cfg, err := config.Load()
	if err != nil {
		return nil, cfg, err
	}
	db, err := store.Open()
	return db, cfg, err
}

func status() error {
	db, cfg, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	pid := readPID()
	observations := count(db, "ocr_observations")
	entries := count(db, "entries")
	files := count(db, "files")
	timelineBlocks := count(db, "timeline_blocks")
	sessions := count(db, "sessions")
	fmt.Printf("Version               %s\n", version)
	fmt.Printf("Root                  %s\n", paths.Root())
	if pid > 0 {
		fmt.Printf("Daemon                running pid %d\n", pid)
	} else {
		fmt.Println("Daemon                stopped")
	}
	fmt.Printf("Observation buffer    %s\n", paths.ObservationBufferDir())
	fmt.Printf("Observations          %d indexed\n", observations)
	fmt.Printf("Memory                %d files, %d entries\n", files, entries)
	fmt.Printf("Timeline              %d blocks\n", timelineBlocks)
	fmt.Printf("Sessions              %d rows\n", sessions)
	fmt.Printf("MCP backend           unix://%s (internal only)\n", paths.ToolsSocket())
	fmt.Printf("Ingest                %s:%d enabled=%v\n", cfg.Ingest.Host, cfg.Ingest.Port, cfg.Ingest.Enabled)
	return nil
}

func count(db *store.DB, table string) int {
	var n int
	_ = db.Conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n)
	return n
}

func ingestObservation(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: memfab ingest-observation <path|->")
	}
	db, _, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	var data []byte
	if args[0] == "-" {
		data, err = os.ReadFile("/dev/stdin")
	} else {
		data, err = os.ReadFile(args[0])
	}
	if err != nil {
		return err
	}
	var o capture.Observation
	if err := json.Unmarshal(data, &o); err != nil {
		return err
	}
	stored, err := ingest.StoreObservation(db, o)
	if err != nil {
		return err
	}
	fmt.Println(stored.Path)
	return nil
}

func rebuildIndex() error {
	db, _, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	files, entries, skipped, err := db.RebuildMemoryIndex(paths.MemoryDir())
	if err != nil {
		return err
	}
	fmt.Printf("Memory index rebuilt: %d files, %d entries, %d skipped.\n", files, entries, skipped)
	return nil
}

func cleanup() error {
	db, cfg, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	now := time.Now()
	latest, found, err := db.LatestTimelineEnd()
	if err != nil {
		return err
	}
	if found {
		stats, err := capture.CleanupBuffer(db, cfg.Capture, latest, now)
		if err != nil {
			return err
		}
		fmt.Printf("cleanup deleted_observations=%d deleted_observation_bytes=%d evicted_observations=%d\n", stats.DeletedObservations, stats.DeletedObservationBytes, stats.EvictedObservations)
	} else {
		fmt.Println("cleanup capture noop: no timeline blocks yet")
	}
	return nil
}

func resetData() error {
	db, _, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	stats, err := ocReset.Run(db)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"ok": true, "reset": stats})
}

func rebuildObservationsIndex() error {
	db, _, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	files, err := filepath.Glob(filepath.Join(paths.ObservationBufferDir(), "*.json"))
	if err != nil {
		return err
	}
	indexed, skipped := 0, 0
	existing := make(map[string]bool, len(files))
	for _, p := range files {
		o, err := capture.ReadFile(p)
		if err != nil {
			skipped++
			continue
		}
		stem := capture.StemFromPath(p)
		if err := db.InsertObservation(stem, o); err != nil {
			return err
		}
		existing[stem] = true
		indexed++
	}
	if err := db.DeleteMissingObservations(existing); err != nil {
		return err
	}
	fmt.Printf("Observations index rebuilt: %d indexed, %d skipped (of %d files).\n", indexed, skipped, len(files))
	return nil
}

func startForeground() error {
	db, cfg, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	if err := paths.WritePrivateFile(paths.PIDFile(), []byte(strconv.Itoa(os.Getpid()))); err != nil {
		return err
	}
	defer os.Remove(paths.PIDFile())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	servers := []*http.Server{}
	toolServer, err := listenToolBackend(db, cfg)
	if err != nil {
		return err
	}
	defer toolServer.Close()
	go func() {
		log.Printf("memory tool backend listening on unix://%s", toolServer.SocketPath())
		if err := toolServer.Serve(); err != nil {
			log.Printf("memory tool backend failed: %v", err)
		}
	}()
	if cfg.Ingest.Enabled {
		addr := net.JoinHostPort(cfg.Ingest.Host, strconv.Itoa(cfg.Ingest.Port))
		srv := newIngestHTTPServer(addr, ingest.Handler(db, cfg))
		servers = append(servers, srv)
		go serve(ctx, srv, "ingest")
	}
	go daemon.Run(ctx, db, cfg)
	fmt.Println("memfab Go running in foreground. Press Ctrl+C to stop.")
	waitSignal()
	for _, srv := range servers {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = srv.Shutdown(shutdownCtx)
		cancel()
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	return toolServer.Shutdown(shutdownCtx)
}

func newIngestHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: ingestReadHeaderTimeout,
		ReadTimeout:       ingestReadTimeout,
		WriteTimeout:      ingestWriteTimeout,
		IdleTimeout:       ingestIdleTimeout,
		MaxHeaderBytes:    ingestMaxHeaderBytes,
	}
}

func serve(ctx context.Context, srv *http.Server, name string) {
	log.Printf("%s server listening on %s", name, srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("%s server failed: %v", name, err)
	}
	<-ctx.Done()
}

func serveMCP() error {
	db, cfg, err := loadDB()
	if err != nil {
		return err
	}
	defer db.Close()
	toolServer, err := listenToolBackend(db, cfg)
	if err != nil {
		return err
	}
	defer toolServer.Close()
	fmt.Println("Memory tool backend:", "unix://"+toolServer.SocketPath())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serveErr := make(chan error, 1)
	go func() { serveErr <- toolServer.Serve() }()
	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return toolServer.Shutdown(shutdownCtx)
	}
}

// listenToolBackend intentionally ignores the legacy public MCP listener
// settings. They remain parseable so existing config files load, but no value
// of host, port, transport, or auto_start can reopen a TCP endpoint.
func listenToolBackend(db *store.DB, cfg config.Config) (*mcp.UnixServer, error) {
	return mcp.ListenUnix(paths.ToolsSocket(), mcp.Handler(db, cfg))
}

func stop() error {
	pid := readPID()
	if pid <= 0 {
		return errors.New("daemon not running")
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
}

func readPID() int {
	b, err := os.ReadFile(paths.PIDFile())
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	if pid <= 0 {
		return 0
	}
	if err := syscall.Kill(pid, 0); err != nil {
		return 0
	}
	return pid
}

func waitSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	signal.Stop(ch)
}

func timeline(args []string) error {
	if len(args) == 0 {
		return errors.New("timeline requires subcommand")
	}
	switch args[0] {
	case "tick":
		fs := flag.NewFlagSet("timeline tick", flag.ExitOnError)
		includeOpen := fs.Bool("include-open", false, "also produce the current open timeline window if missing")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		db, cfg, err := loadDB()
		if err != nil {
			return err
		}
		defer db.Close()
		produced, err := ocTimeline.TickWithOptions(context.Background(), db, cfg, ocTimeline.TickOptions{IncludeOpen: *includeOpen})
		if err != nil {
			return err
		}
		fmt.Printf("Produced %d block(s).\n", produced)
	case "list":
		fs := flag.NewFlagSet("timeline list", flag.ExitOnError)
		limit := fs.Int("n", 24, "number of blocks")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		db, _, err := loadDB()
		if err != nil {
			return err
		}
		defer db.Close()
		blocks, err := db.ListTimelineBlocks(*limit)
		if err != nil {
			return err
		}
		if len(blocks) == 0 {
			fmt.Println("No timeline blocks.")
			return nil
		}
		for _, block := range blocks {
			apps := strings.Join(block.AppsUsed, ", ")
			if apps == "" {
				apps = "-"
			}
			fmt.Printf("%s -> %s apps=%s captures=%d\n", block.StartTime, block.EndTime, apps, block.CaptureCount)
			for _, entry := range block.Entries {
				fmt.Printf("  - %s\n", entry)
			}
		}
	default:
		return fmt.Errorf("unknown timeline command %s", args[0])
	}
	return nil
}

func session(args []string) error {
	if len(args) == 0 {
		return errors.New("session requires subcommand: tick, list, force-end")
	}
	switch args[0] {
	case "tick":
		db, cfg, err := loadDB()
		if err != nil {
			return err
		}
		defer db.Close()
		result, err := ocSession.TickNow(context.Background(), db, cfg)
		if err != nil {
			return err
		}
		active := result.ActiveID
		if active == "" {
			active = "-"
		}
		fmt.Printf("scanned=%d started=%d ended=%d active=%s\n", result.Scanned, result.Started, result.Ended, active)
	case "force-end":
		db, _, err := loadDB()
		if err != nil {
			return err
		}
		defer db.Close()
		ended, err := ocSession.ForceEndActive(context.Background(), db, time.Now())
		if err != nil {
			return err
		}
		if ended {
			fmt.Println("ended=1")
		} else {
			fmt.Println("ended=0")
		}
	case "list":
		fs := flag.NewFlagSet("session list", flag.ExitOnError)
		limit := fs.Int("n", 20, "number of sessions")
		status := fs.String("status", "all", "session status")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		db, _, err := loadDB()
		if err != nil {
			return err
		}
		defer db.Close()
		rows, err := db.ListSessions(*status, *limit)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			fmt.Println("No sessions.")
			return nil
		}
		for _, row := range rows {
			end := row.EndTime
			if end == "" {
				end = "-"
			}
			fmt.Printf("%s %s -> %s %s\n", row.ID, row.StartTime, end, row.Status)
		}
	default:
		return fmt.Errorf("unknown session command %s", args[0])
	}
	return nil
}

func writePretty(v any) {
	b, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(b))
}
