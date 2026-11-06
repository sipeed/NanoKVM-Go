package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"memfab/internal/paths"
)

type ModelConfig struct {
	Model          string         `toml:"model"`
	BaseURL        string         `toml:"base_url"`
	APIKeyEnv      string         `toml:"api_key_env"`
	APIKey         string         `toml:"api_key"`
	MaxTokens      int            `toml:"max_tokens"`
	TimeoutSeconds int            `toml:"timeout_seconds"`
	Stage          string         `toml:"-"`
	Trace          LLMTraceConfig `toml:"-"`
}

type LLMTraceConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"`
}

type ReplayConfig struct {
	Enabled bool `toml:"enabled"`
}

type CaptureConfig struct {
	EventDriven            bool    `toml:"event_driven"`
	HeartbeatMinutes       int     `toml:"heartbeat_minutes"`
	DebounceSeconds        float64 `toml:"debounce_seconds"`
	MinCaptureGapSeconds   float64 `toml:"min_capture_gap_seconds"`
	DedupIntervalSeconds   float64 `toml:"dedup_interval_seconds"`
	SameWindowDedupSeconds float64 `toml:"same_window_dedup_seconds"`
	BufferRetentionHours   int     `toml:"buffer_retention_hours"`
	BufferMaxMB            int     `toml:"buffer_max_mb"`
}

type TimelineConfig struct {
	WindowMinutes       int `toml:"window_minutes"`
	ColdLookbackMinutes int `toml:"cold_lookback_minutes"`
	RecentContextBlocks int `toml:"recent_context_blocks"`
}

type WriterConfig struct {
	SoftLimitTokens            int `toml:"soft_limit_tokens"`
	HardLimitTokens            int `toml:"hard_limit_tokens"`
	DedupWindowHours           int `toml:"dedup_window_hours"`
	ColdStartConservativeHours int `toml:"cold_start_conservative_hours"`
}

type SessionConfig struct {
	GapMinutes      int `toml:"gap_minutes"`
	SoftCutMinutes  int `toml:"soft_cut_minutes"`
	MaxSessionHours int `toml:"max_session_hours"`
	TickSeconds     int `toml:"tick_seconds"`
	FlushMinutes    int `toml:"flush_minutes"`
}

type ReducerConfig struct {
	Enabled         bool `toml:"enabled"`
	DailyTickHour   int  `toml:"daily_tick_hour"`
	DailyTickMinute int  `toml:"daily_tick_minute"`
}

type ClassifierConfig struct {
	Enabled            bool `toml:"enabled"`
	IntervalMinutes    int  `toml:"interval_minutes"`
	MaxIterations      int  `toml:"max_iterations"`
	MaxSessionsPerTick int  `toml:"max_sessions_per_tick"`
}

type CompactConfig struct {
	Enabled            bool    `toml:"enabled"`
	MaxFilesPerTick    int     `toml:"max_files_per_tick"`
	PreservationRatio  float64 `toml:"preservation_ratio"`
	MinEstimatedTokens int     `toml:"min_estimated_tokens"`
}

type DaemonConfig struct {
	TimelineEnabled        bool `toml:"timeline_enabled"`
	SessionEnabled         bool `toml:"session_enabled"`
	ReducerEnabled         bool `toml:"reducer_enabled"`
	ClassifierEnabled      bool `toml:"classifier_enabled"`
	CompactEnabled         bool `toml:"compact_enabled"`
	DailySafetyEnabled     bool `toml:"daily_safety_enabled"`
	LLMConcurrency         int  `toml:"llm_concurrency"`
	TimelineIntervalSecs   int  `toml:"timeline_interval_seconds"`
	ReducerIntervalSecs    int  `toml:"reducer_interval_seconds"`
	CompactIntervalMinutes int  `toml:"compact_interval_minutes"`
	ReducerLimit           int  `toml:"reducer_limit"`
}

type MemoryConfig struct {
	AutoDormantDays int `toml:"auto_dormant_days"`
}

type SearchConfig struct {
	DefaultTopK               int  `toml:"default_top_k"`
	FilterSupersededByDefault bool `toml:"filter_superseded_by_default"`
}

type MCPConfig struct {
	AutoStart bool   `toml:"auto_start"`
	Transport string `toml:"transport"`
	Host      string `toml:"host"`
	Port      int    `toml:"port"`
}

type IngestConfig struct {
	Enabled      bool   `toml:"enabled"`
	Host         string `toml:"host"`
	Port         int    `toml:"port"`
	RequireToken bool   `toml:"require_token"`
	Token        string `toml:"token"`
	MaxBatchSize int    `toml:"max_batch_size"`
	MaxBodyBytes int64  `toml:"max_body_bytes"`
}

type ModelsConfig struct {
	Default    ModelConfig `toml:"default"`
	Compact    ModelConfig `toml:"compact"`
	Timeline   ModelConfig `toml:"timeline"`
	Reducer    ModelConfig `toml:"reducer"`
	Classifier ModelConfig `toml:"classifier"`
}

type Config struct {
	Models     ModelsConfig     `toml:"models"`
	LLMTrace   LLMTraceConfig   `toml:"llm_trace"`
	Capture    CaptureConfig    `toml:"capture"`
	Timeline   TimelineConfig   `toml:"timeline"`
	Writer     WriterConfig     `toml:"writer"`
	Session    SessionConfig    `toml:"session"`
	Reducer    ReducerConfig    `toml:"reducer"`
	Classifier ClassifierConfig `toml:"classifier"`
	Compact    CompactConfig    `toml:"compact"`
	Daemon     DaemonConfig     `toml:"daemon"`
	Memory     MemoryConfig     `toml:"memory"`
	Search     SearchConfig     `toml:"search"`
	MCP        MCPConfig        `toml:"mcp"`
	Ingest     IngestConfig     `toml:"ingest"`
	Replay     ReplayConfig     `toml:"replay"`
}

func Default() Config {
	return Config{
		Models: ModelsConfig{Default: ModelConfig{
			Model: "gpt-5.5", APIKeyEnv: "OPENAI_API_KEY", MaxTokens: 1024, TimeoutSeconds: 60,
		}},
		Capture: CaptureConfig{
			EventDriven: true, HeartbeatMinutes: 10, DebounceSeconds: 3, MinCaptureGapSeconds: 2,
			DedupIntervalSeconds: 1, SameWindowDedupSeconds: 5, BufferRetentionHours: 168,
			BufferMaxMB: 16,
		},
		Timeline:   TimelineConfig{WindowMinutes: 1, ColdLookbackMinutes: 30, RecentContextBlocks: 120},
		Writer:     WriterConfig{SoftLimitTokens: 8000, HardLimitTokens: 20000, DedupWindowHours: 24},
		Session:    SessionConfig{GapMinutes: 5, SoftCutMinutes: 3, MaxSessionHours: 2, TickSeconds: 30, FlushMinutes: 5},
		Reducer:    ReducerConfig{Enabled: true, DailyTickHour: 23, DailyTickMinute: 55},
		Classifier: ClassifierConfig{Enabled: true, IntervalMinutes: 30, MaxIterations: 8, MaxSessionsPerTick: 4},
		Compact:    CompactConfig{Enabled: true, MaxFilesPerTick: 2, PreservationRatio: 0.95},
		Daemon:     DaemonConfig{TimelineEnabled: true, SessionEnabled: true, ReducerEnabled: true, ClassifierEnabled: true, CompactEnabled: true, DailySafetyEnabled: true, LLMConcurrency: 1, TimelineIntervalSecs: 60, ReducerIntervalSecs: 60, CompactIntervalMinutes: 30, ReducerLimit: 10},
		Memory:     MemoryConfig{AutoDormantDays: 30},
		Search:     SearchConfig{DefaultTopK: 5, FilterSupersededByDefault: true},
		MCP:        MCPConfig{AutoStart: true, Transport: "unix"},
		Ingest:     IngestConfig{Enabled: true, Host: "127.0.0.1", Port: 8743, MaxBatchSize: 16, MaxBodyBytes: 1 << 20},
		Replay:     ReplayConfig{Enabled: true},
	}
}

func WriteDefaultIfMissing() (bool, error) {
	if _, err := os.Lstat(paths.ConfigFile()); err == nil {
		if err := paths.SecureRegularFiles(paths.ConfigFile()); err != nil {
			return false, err
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := paths.EnsureDirs(); err != nil {
		return false, err
	}
	return writeFileNoReplace(paths.ConfigFile(), []byte(DefaultTOML()), 0o600)
}

// writeFileNoReplace publishes a fully written and synced file without ever
// exposing a partial destination. A hard link supplies atomic no-replace
// semantics for concurrent creators on the same filesystem.
func writeFileNoReplace(path string, data []byte, mode os.FileMode) (bool, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, paths.DirectoryMode); err != nil {
		return false, err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return false, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	closeWithError := func(cause error) (bool, error) {
		if closeErr := tmp.Close(); cause == nil {
			cause = closeErr
		}
		return false, cause
	}
	if err := tmp.Chmod(mode); err != nil {
		return closeWithError(err)
	}
	if _, err := tmp.Write(data); err != nil {
		return closeWithError(err)
	}
	if err := tmp.Sync(); err != nil {
		return closeWithError(err)
	}
	if err := tmp.Close(); err != nil {
		return false, err
	}
	if err := os.Link(tmpPath, path); err != nil {
		if errors.Is(err, os.ErrExist) {
			return false, nil
		}
		return false, err
	}
	if err := os.Remove(tmpPath); err != nil {
		return true, err
	}
	if err := syncDirectory(dir); err != nil {
		return true, err
	}
	return true, nil
}

func syncDirectory(dir string) error {
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return err
	}
	return directory.Close()
}

func Load() (Config, error) {
	cfg := Default()
	if _, err := WriteDefaultIfMissing(); err != nil {
		return cfg, err
	}
	if _, err := toml.DecodeFile(paths.ConfigFile(), &cfg); err != nil {
		return cfg, err
	}
	cfg.applyLLMTraceDefaults()
	cfg.applyModelInheritance()
	return cfg, nil
}

func (c *Config) applyLLMTraceDefaults() {
	if strings.TrimSpace(c.LLMTrace.Path) == "" {
		c.LLMTrace.Path = filepath.Join(paths.LogsDir(), "llm-trace.jsonl")
	}
}

func (c *Config) applyModelInheritance() {
	base := c.Models.Default
	merge := func(stageName string, stage *ModelConfig) {
		if stage.Model == "" {
			stage.Model = base.Model
		}
		if stage.BaseURL == "" {
			stage.BaseURL = base.BaseURL
		}
		if stage.APIKeyEnv == "" {
			stage.APIKeyEnv = base.APIKeyEnv
		}
		if stage.APIKey == "" {
			stage.APIKey = base.APIKey
		}
		if stage.MaxTokens == 0 {
			stage.MaxTokens = base.MaxTokens
		}
		if stage.TimeoutSeconds == 0 {
			stage.TimeoutSeconds = base.TimeoutSeconds
		}
		stage.Stage = stageName
		stage.Trace = c.LLMTrace
	}
	merge("compact", &c.Models.Compact)
	merge("timeline", &c.Models.Timeline)
	merge("reducer", &c.Models.Reducer)
	merge("classifier", &c.Models.Classifier)
}

func (m ModelConfig) ResolvedAPIKey() string {
	if m.APIKey != "" {
		return m.APIKey
	}
	if m.APIKeyEnv != "" {
		return os.Getenv(m.APIKeyEnv)
	}
	return ""
}

func DefaultTOML() string {
	return strings.TrimSpace(`# memfab Go configuration for low-memory Linux devices

[models.default]
model = "gpt-5.5"
api_key_env = "OPENAI_API_KEY"
max_tokens = 1024
timeout_seconds = 60
# base_url = ""
# api_key = ""

[models.compact]

[models.timeline]

[models.reducer]

[models.classifier]

[llm_trace]
enabled = false
path = ""

[replay]
enabled = true

[capture]
event_driven = true
heartbeat_minutes = 10
debounce_seconds = 3.0
min_capture_gap_seconds = 2.0
dedup_interval_seconds = 1.0
same_window_dedup_seconds = 5.0
buffer_retention_hours = 168
buffer_max_mb = 16

[timeline]
window_minutes = 1
cold_lookback_minutes = 30
recent_context_blocks = 120

[writer]
soft_limit_tokens = 8000
hard_limit_tokens = 20000
dedup_window_hours = 24
cold_start_conservative_hours = 0

[session]
gap_minutes = 5
soft_cut_minutes = 3
max_session_hours = 2
tick_seconds = 30
flush_minutes = 5

[reducer]
enabled = true
daily_tick_hour = 23
daily_tick_minute = 55

[classifier]
enabled = true
interval_minutes = 30
max_iterations = 8
max_sessions_per_tick = 4

[compact]
enabled = true
max_files_per_tick = 2
preservation_ratio = 0.95
min_estimated_tokens = 0

[daemon]
timeline_enabled = true
session_enabled = true
reducer_enabled = true
classifier_enabled = true
compact_enabled = true
daily_safety_enabled = true
llm_concurrency = 1
timeline_interval_seconds = 60
reducer_interval_seconds = 60
compact_interval_minutes = 30
reducer_limit = 10

[memory]
auto_dormant_days = 30

[search]
default_top_k = 5
filter_superseded_by_default = true

[mcp]
auto_start = true
transport = "unix"

[ingest]
enabled = true
host = "127.0.0.1"
port = 8743
require_token = false
max_batch_size = 16
max_body_bytes = 1048576
`) + "\n"
}

func RedactedTOML() (string, error) {
	b, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		return "", err
	}
	text := string(b)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "api_key") || strings.HasPrefix(trim, "token") {
			if idx := strings.Index(line, "="); idx >= 0 {
				lines[i] = line[:idx+1] + " \"REDACTED\""
			}
		}
	}
	return fmt.Sprintf("%s\n%s", paths.ConfigFile(), strings.Join(lines, "\n")), nil
}
