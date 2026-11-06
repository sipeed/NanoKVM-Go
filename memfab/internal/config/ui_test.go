package config

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"memfab/internal/paths"
)

func TestReadUIRedactsAPIKeyAndOmitsEnvironmentName(t *testing.T) {
	writeUIConfigFixture(t, strings.ReplaceAll(DefaultTOML(),
		`# base_url = ""`, `base_url = "https://example.test/v1"`))
	data, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(data), `# api_key = ""`, `api_key = "top-secret"`, 1)
	if err := os.WriteFile(paths.ConfigFile(), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Revision == "" {
		t.Fatal("expected config revision")
	}
	if cfg.Models.Default.APIKey != nil {
		t.Fatal("API key must not be returned")
	}
	if !cfg.Models.Default.APIKeyConfigured {
		t.Fatal("expected API key configured flag")
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("apiKeyEnv")) {
		t.Fatalf("API key environment name must not be returned: %s", encoded)
	}
	if cfg.Models.Default.BaseURL != "https://example.test/v1" {
		t.Fatalf("baseUrl=%q", cfg.Models.Default.BaseURL)
	}
	if !strings.HasPrefix(cfg.Revision, "v1:") {
		t.Fatalf("revision=%q, want opaque v1 revision", cfg.Revision)
	}
	raw, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	bare := sha256.Sum256(raw)
	if cfg.Revision == hex.EncodeToString(bare[:]) || cfg.Revision == "v1:"+hex.EncodeToString(bare[:]) {
		t.Fatal("revision must not expose the config's bare SHA-256")
	}
	computed, err := Revision(raw)
	if err != nil {
		t.Fatal(err)
	}
	if computed != cfg.Revision {
		t.Fatalf("Revision(raw)=%q, ReadUI revision=%q", computed, cfg.Revision)
	}
	key, err := os.ReadFile(paths.ConfigRevisionKey())
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != revisionKeyBytes || bytes.Equal(key, make([]byte, revisionKeyBytes)) {
		t.Fatalf("unexpected revision key length/content: %d bytes", len(key))
	}
	keyInfo, err := os.Stat(paths.ConfigRevisionKey())
	if err != nil {
		t.Fatal(err)
	}
	if keyInfo.Mode().Perm() != 0o600 {
		t.Fatalf("revision key mode=%o", keyInfo.Mode().Perm())
	}
}

func TestRevisionKeyConcurrentCreationUsesOneAtomicKey(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)

	const workers = 24
	keys := make(chan []byte, workers)
	errs := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			key, err := loadRevisionKey()
			if err != nil {
				errs <- err
				return
			}
			keys <- key
		}()
	}
	group.Wait()
	close(keys)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var first []byte
	for key := range keys {
		if first == nil {
			first = key
			continue
		}
		if !bytes.Equal(key, first) {
			t.Fatal("concurrent creators returned different revision keys")
		}
	}
	if len(first) != revisionKeyBytes {
		t.Fatalf("revision key length=%d", len(first))
	}
	info, err := os.Stat(paths.ConfigRevisionKey())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("revision key mode=%o", info.Mode().Perm())
	}
	temps, err := filepath.Glob(filepath.Join(root, ".config-revision.key.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Fatalf("temporary revision key files left behind: %v", temps)
	}
}

func TestWriteDefaultIfMissingPublishesOneCompleteFile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)

	const workers = 24
	created := make(chan bool, workers)
	errs := make(chan error, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			ok, err := WriteDefaultIfMissing()
			if err != nil {
				errs <- err
				return
			}
			created <- ok
		}()
	}
	group.Wait()
	close(created)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	createdCount := 0
	for result := range created {
		if result {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created count=%d, want 1", createdCount)
	}
	raw, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != DefaultTOML() {
		t.Fatal("atomically published default config is incomplete")
	}
	info, err := os.Stat(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode=%o", info.Mode().Perm())
	}
	temps, err := filepath.Glob(filepath.Join(root, ".config.toml.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temps) != 0 {
		t.Fatalf("temporary config files left behind: %v", temps)
	}
}

func TestUpdateUIOnlyChangesDefaultModelAndIgnoresHiddenConfigLimits(t *testing.T) {
	text := hiddenTypeMismatchTOML(strings.Replace(DefaultTOML(), `# api_key = ""`, `api_key = "keep-secret"`, 1))
	const apiKeyEnvLine = `api_key_env   =   "not a valid environment name" # keep formatting`
	text = strings.Replace(text, `api_key_env = "OPENAI_API_KEY"`, apiKeyEnvLine, 1)
	text = strings.Replace(text, "buffer_max_mb = 16", "buffer_max_mb = 999999 # outside the old UI limit", 1)
	text = strings.Replace(text, "soft_limit_tokens = 8000", "soft_limit_tokens = 2000000", 1)
	text = strings.Replace(text, "hard_limit_tokens = 20000", "hard_limit_tokens = 1", 1)
	text = strings.Replace(text, "reducer_limit = 10", "reducer_limit = 999999", 1)
	text += "\n# keep custom section\n[custom]\nvalue = \"keep\"\n"
	writeUIConfigFixture(t, text)

	revision, err := Revision([]byte(text))
	if err != nil {
		t.Fatalf("Revision rejected type-incompatible hidden config: %v", err)
	}
	cfg, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Revision != revision {
		t.Fatalf("ReadUI revision=%q, Revision=%q", cfg.Revision, revision)
	}
	cfg.Models.Default.Model = "gpt-test"
	updated, changed, err := UpdateUI(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed || updated.Revision == cfg.Revision {
		t.Fatalf("changed=%t old=%s new=%s", changed, cfg.Revision, updated.Revision)
	}
	if updated.Models.Default.APIKey != nil || !updated.Models.Default.APIKeyConfigured {
		t.Fatalf("unexpected API key response: %+v", updated.Models.Default)
	}

	raw, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	for _, want := range []string{
		`model = "gpt-test"`,
		`api_key = "keep-secret"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("updated config missing %q:\n%s", want, got)
		}
	}
	if bytes.Count(raw, []byte(apiKeyEnvLine+"\n")) != 1 {
		t.Fatalf("API key environment line was not preserved byte-for-byte:\n%s", raw)
	}
	if hiddenTOML(got) != hiddenTOML(text) {
		t.Fatalf("hidden config changed while updating the default model\nbefore:\n%s\nafter:\n%s", hiddenTOML(text), hiddenTOML(got))
	}
	info, err := os.Stat(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

func TestReadUIRejectsTypeMismatchInDefaultModel(t *testing.T) {
	text := strings.Replace(DefaultTOML(), `model = "gpt-5.5"`, `model = 123`, 1)
	writeUIConfigFixture(t, text)

	if _, err := ReadUI(); err == nil {
		t.Fatal("expected models.default.model type mismatch to be rejected")
	}
}

func TestPatchTOMLSkipsMultilineStringsAndTreatsArrayTablesAsBoundaries(t *testing.T) {
	original := `[capture]
heartbeat_minutes = 10
unknown_basic = """
[daemon]
llm_concurrency = 901
heartbeat_minutes = 902
[[custom.fake]]
"""
unknown_literal = '''
[capture]
heartbeat_minutes = 903
'''

[[custom.items]]
heartbeat_minutes = 904
llm_concurrency = 905

[daemon]
llm_concurrency = 1
`
	updated := patchTOML(original, []tomlUpdate{
		{section: "capture", key: "heartbeat_minutes", value: "12"},
		{section: "daemon", key: "llm_concurrency", value: "2"},
	})
	for _, preserved := range []string{
		"llm_concurrency = 901",
		"heartbeat_minutes = 902",
		"heartbeat_minutes = 903",
		"heartbeat_minutes = 904",
		"llm_concurrency = 905",
	} {
		if !strings.Contains(updated, preserved) {
			t.Fatalf("multiline/array content %q was changed:\n%s", preserved, updated)
		}
	}
	if strings.Count(updated, "heartbeat_minutes = 12") != 1 {
		t.Fatalf("managed capture key was not changed exactly once:\n%s", updated)
	}
	if strings.Count(updated, "llm_concurrency = 2") != 1 {
		t.Fatalf("managed daemon key was not changed exactly once:\n%s", updated)
	}
	if _, err := decodeRawConfig([]byte(updated)); err != nil {
		t.Fatalf("patched TOML is invalid: %v\n%s", err, updated)
	}
}

func TestPatchTOMLReplacesEntireManagedMultilineValues(t *testing.T) {
	original := `[models.default]
model = """
old-model
[daemon]
llm_concurrency = 999
"""
api_key = '''
old-secret
[capture]
heartbeat_minutes = 999
'''
max_tokens = 1024
timeout_seconds = 60
`
	updated := patchTOML(original, []tomlUpdate{
		{section: "models.default", key: "model", value: `"new-model"`},
		{section: "models.default", key: "api_key", remove: true},
	})
	for _, removed := range []string{"old-model", "old-secret", "llm_concurrency = 999", "heartbeat_minutes = 999", `api_key = '''`} {
		if strings.Contains(updated, removed) {
			t.Fatalf("managed multiline content %q was left behind:\n%s", removed, updated)
		}
	}
	if strings.Count(updated, `model = "new-model"`) != 1 {
		t.Fatalf("managed multiline model was not replaced exactly once:\n%s", updated)
	}
	if _, err := decodeRawConfig([]byte(updated)); err != nil {
		t.Fatalf("patched TOML is invalid: %v\n%s", err, updated)
	}
}

func TestUpdateUIReplacesAndClearsAPIKey(t *testing.T) {
	writeUIConfigFixture(t, DefaultTOML())
	cfg, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	newKey := "replacement-secret"
	cfg.Models.Default.APIKey = &newKey
	updated, _, err := UpdateUI(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Models.Default.APIKeyConfigured || updated.Models.Default.APIKey != nil {
		t.Fatalf("unexpected updated key state: %+v", updated.Models.Default)
	}
	raw, _ := os.ReadFile(paths.ConfigFile())
	if !strings.Contains(string(raw), `api_key = "replacement-secret"`) {
		t.Fatalf("replacement key missing:\n%s", raw)
	}

	clear := ""
	updated.Models.Default.APIKey = &clear
	cleared, _, err := UpdateUI(updated)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Models.Default.APIKeyConfigured {
		t.Fatal("expected direct API key to be cleared")
	}
	raw, _ = os.ReadFile(paths.ConfigFile())
	if strings.Contains(string(raw), "\napi_key =") {
		t.Fatalf("API key line should be removed:\n%s", raw)
	}
}

func TestUpdateUIUnchangedDefaultDoesNotRewrite(t *testing.T) {
	writeUIConfigFixture(t, DefaultTOML())
	before, _ := os.ReadFile(paths.ConfigFile())
	cfg, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	updated, changed, err := UpdateUI(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if changed || updated.Revision != cfg.Revision {
		t.Fatalf("changed=%t old=%s new=%s", changed, cfg.Revision, updated.Revision)
	}
	after, _ := os.ReadFile(paths.ConfigFile())
	if string(after) != string(before) {
		t.Fatalf("unchanged update rewrote config:\n%s", after)
	}
}

func TestUpdateUIRejectsRevisionConflictWithoutWriting(t *testing.T) {
	writeUIConfigFixture(t, DefaultTOML())
	before, _ := os.ReadFile(paths.ConfigFile())
	cfg, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Revision = "stale"
	cfg.Models.Default.Model = "must-not-write"
	if _, _, err := UpdateUI(cfg); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("err=%v, want revision conflict", err)
	}
	after, _ := os.ReadFile(paths.ConfigFile())
	if string(after) != string(before) {
		t.Fatalf("config changed after conflict:\n%s", after)
	}
}

func TestUpdateUIRejectsInvalidDefaultModelWithoutWriting(t *testing.T) {
	writeUIConfigFixture(t, DefaultTOML())
	before, _ := os.ReadFile(paths.ConfigFile())
	cfg, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Models.Default.Model = ""
	if _, _, err := UpdateUI(cfg); err == nil || !strings.Contains(err.Error(), "models.default.model") {
		t.Fatalf("unexpected error: %v", err)
	}
	after, _ := os.ReadFile(paths.ConfigFile())
	if string(after) != string(before) {
		t.Fatal("invalid update changed config")
	}
}

func TestUpdateUIRejectsUnsafeUpperBoundsWithoutWriting(t *testing.T) {
	writeUIConfigFixture(t, DefaultTOML())
	before, _ := os.ReadFile(paths.ConfigFile())
	base, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		field  string
		mutate func(*UIConfig)
	}{
		{"model timeout duration", "models.default.timeoutSeconds", func(cfg *UIConfig) { cfg.Models.Default.TimeoutSeconds = maxModelTimeoutSeconds + 1 }},
		{"model token count", "models.default.maxTokens", func(cfg *UIConfig) { cfg.Models.Default.MaxTokens = maxModelTokens + 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := base
			test.mutate(&request)
			if _, _, err := UpdateUI(request); err == nil || !strings.Contains(err.Error(), test.field) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
	after, _ := os.ReadFile(paths.ConfigFile())
	if string(after) != string(before) {
		t.Fatal("unsafe update changed config")
	}
}

func TestWriteConfigRollsBackWhenDirectorySyncFails(t *testing.T) {
	original := DefaultTOML() + "# keep old config\n"
	writeUIConfigFixture(t, original)
	updated := strings.Replace(original, `model = "gpt-5.5"`, `model = "must-roll-back"`, 1)

	originalSync := syncConfigDirectory
	defer func() { syncConfigDirectory = originalSync }()
	injected := errors.New("injected directory sync failure")
	calls := 0
	syncConfigDirectory = func(directory *os.File) error {
		calls++
		if calls == 1 {
			return injected
		}
		return directory.Sync()
	}
	if err := writeConfig([]byte(updated)); !errors.Is(err, injected) {
		t.Fatalf("writeConfig err=%v, want injected sync failure", err)
	}
	if calls < 2 {
		t.Fatalf("directory sync calls=%d, want rollback sync", calls)
	}
	raw, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != original {
		t.Fatalf("config remained committed after sync error:\n%s", raw)
	}
}

func TestRestoreUsesRevisionCASAndRestoresExactConfig(t *testing.T) {
	original := hiddenTypeMismatchTOML(strings.Replace(DefaultTOML(), `# api_key = ""`, `api_key = "rollback-secret"`, 1)) + "\n# exact backup\n"
	writeUIConfigFixture(t, original)
	before, err := ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	if err := Restore([]byte("not toml = ["), before.Revision); err == nil {
		t.Fatal("expected invalid restore error")
	}
	request := before
	request.Models.Default.Model = "changed-before-rollback"
	transaction, err := UpdateUITransaction(request)
	if err != nil {
		t.Fatal(err)
	}
	if !transaction.Changed {
		t.Fatal("expected changed transaction")
	}
	encoded, err := json.Marshal(transaction)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("rollback-secret")) {
		t.Fatalf("transaction JSON leaked previous raw config: %s", encoded)
	}
	if err := Restore([]byte(original), before.Revision); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale restore err=%v, want revision conflict", err)
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != original {
		t.Fatalf("restored config differs:\n%s", raw)
	}
	if err := transaction.Rollback(); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("second rollback err=%v, want revision conflict", err)
	}
}

func writeUIConfigFixture(t *testing.T, text string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("MEMFAB_ROOT", root)
	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ConfigFile(), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hiddenTOML(text string) string {
	const defaultSection = "[models.default]"
	start := strings.Index(text, defaultSection)
	if start < 0 {
		return text
	}
	afterHeader := start + len(defaultSection)
	nextSection := strings.Index(text[afterHeader:], "\n[")
	if nextSection < 0 {
		return text[:start]
	}
	end := afterHeader + nextSection + 1
	return text[:start] + text[end:]
}

func hiddenTypeMismatchTOML(text string) string {
	text = strings.Replace(text, "[models.compact]\n", `[models.compact]
model = 123
api_key = ["hidden-secret"]
max_tokens = 9999999
timeout_seconds = "hidden-timeout"
`, 1)
	text = strings.Replace(text, "heartbeat_minutes = 10", `heartbeat_minutes = "hidden-capture"`, 1)
	return strings.Replace(text, "llm_concurrency = 1", `llm_concurrency = { hidden = true }`, 1)
}
