package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"memfab/internal/paths"
)

func TestLoadAppliesLLMTraceDefaultsAndStages(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LLMTrace.Enabled {
		t.Fatal("llm_trace should be disabled by default")
	}
	if !cfg.Replay.Enabled {
		t.Fatal("replay should be enabled by default")
	}
	if cfg.MCP.Transport != "unix" || cfg.MCP.Host != "" || cfg.MCP.Port != 0 {
		t.Fatalf("default MCP config=%+v, want internal Unix backend", cfg.MCP)
	}
	if cfg.Models.Default.Model != "gpt-5.5" || cfg.Models.Timeline.Model != "gpt-5.5" {
		t.Fatalf("default model default=%q timeline=%q", cfg.Models.Default.Model, cfg.Models.Timeline.Model)
	}
	wantTracePath := filepath.Join(paths.LogsDir(), "llm-trace.jsonl")
	if cfg.LLMTrace.Path != wantTracePath {
		t.Fatalf("trace path=%q, want %q", cfg.LLMTrace.Path, wantTracePath)
	}
	stages := map[string]ModelConfig{
		"timeline":   cfg.Models.Timeline,
		"reducer":    cfg.Models.Reducer,
		"classifier": cfg.Models.Classifier,
		"compact":    cfg.Models.Compact,
	}
	for want, model := range stages {
		if model.Stage != want {
			t.Fatalf("%s stage=%q", want, model.Stage)
		}
		if model.Trace.Path != wantTracePath || model.Trace.Enabled {
			t.Fatalf("%s trace=%+v", want, model.Trace)
		}
	}
}

func TestLoadKeepsLegacyMCPListenerFieldsParseable(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	text := `
[mcp]
auto_start = true
transport = "streamable-http"
host = "0.0.0.0"
port = 8742
`
	if err := os.WriteFile(paths.ConfigFile(), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.MCP.AutoStart || cfg.MCP.Transport != "streamable-http" || cfg.MCP.Host != "0.0.0.0" || cfg.MCP.Port != 8742 {
		t.Fatalf("legacy MCP fields were not preserved: %+v", cfg.MCP)
	}
}

func TestLoadRejectsSymlinkedConfigWithoutChangingTarget(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "memfab")
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(parent, "outside.toml")
	if err := os.WriteFile(target, []byte("[replay]\nenabled = false\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(target, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, paths.ConfigFile()); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load accepted a symlinked config file")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("external config mode=%#o, want %#o", got, os.FileMode(0o644))
	}
}

func TestLoadHonorsLLMTraceConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	text := `
	[models.default]
	model = "qwen-portal/qwen3.6-plus"
	api_key_env = "OPENAI_API_KEY"

[llm_trace]
enabled = true
path = "/tmp/memfab-llm-trace.jsonl"
`
	if err := os.WriteFile(paths.ConfigFile(), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.LLMTrace.Enabled || cfg.LLMTrace.Path != "/tmp/memfab-llm-trace.jsonl" {
		t.Fatalf("trace=%+v", cfg.LLMTrace)
	}
	if cfg.Models.Timeline.Model != "qwen-portal/qwen3.6-plus" {
		t.Fatalf("timeline model=%q", cfg.Models.Timeline.Model)
	}
	if !cfg.Models.Timeline.Trace.Enabled || cfg.Models.Timeline.Trace.Path != "/tmp/memfab-llm-trace.jsonl" {
		t.Fatalf("timeline trace=%+v", cfg.Models.Timeline.Trace)
	}
}

func TestLoadHonorsReplayEnabled(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	text := `
[replay]
enabled = false
`
	if err := os.WriteFile(paths.ConfigFile(), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Replay.Enabled {
		t.Fatal("replay should honor enabled=false")
	}
}

func TestDefaultTOMLIncludesReplayConfig(t *testing.T) {
	text := DefaultTOML()
	for _, want := range []string{
		"[replay]",
		`enabled = true`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("DefaultTOML missing %q", want)
		}
	}
	for _, removed := range []string{
		`output_dir =`,
		`default_fps =`,
		`default_speed =`,
		`default_context_minutes =`,
		`default_sample_every =`,
		`max_frames =`,
		`allow_external_paths =`,
		`artifact_retention_hours =`,
	} {
		if strings.Contains(text, removed) {
			t.Fatalf("DefaultTOML should not include replay field %q", removed)
		}
	}
}

func TestDefaultTOMLUsesOnlyInternalMCPTransport(t *testing.T) {
	text := DefaultTOML()
	for _, want := range []string{"[mcp]", `transport = "unix"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("DefaultTOML missing %q", want)
		}
	}
	for _, retired := range []string{`host = "0.0.0.0"`, "port = 8742", `transport = "streamable-http"`} {
		if strings.Contains(text, retired) {
			t.Fatalf("DefaultTOML advertises retired MCP setting %q", retired)
		}
	}
}

func TestDefaultTOMLExcludesRemovedScreenshotConfig(t *testing.T) {
	text := DefaultTOML()
	for _, removed := range []string{
		`screenshot_retention_hours =`,
		`screenshot_dir =`,
		`include_screenshot =`,
		`screenshot_max_width =`,
		`screenshot_jpeg_quality =`,
		`ax_depth =`,
		`ax_timeout_seconds =`,
	} {
		if strings.Contains(text, removed) {
			t.Fatalf("DefaultTOML should not include screenshot field %q", removed)
		}
	}
}
