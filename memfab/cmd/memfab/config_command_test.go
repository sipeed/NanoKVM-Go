package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"memfab/internal/config"
	"memfab/internal/paths"
)

func TestConfigCommandGetJSONIsStructuredAndSecretSafe(t *testing.T) {
	setupConfigCommandRoot(t, strings.Replace(config.DefaultTOML(), `# api_key = ""`, `api_key = "secret-value"`, 1))

	var output bytes.Buffer
	if err := configCommand([]string{"get", "--json"}, strings.NewReader(""), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), "secret-value") {
		t.Fatalf("secret leaked in output: %s", output.String())
	}
	if strings.Contains(output.String(), "apiKeyEnv") {
		t.Fatalf("API key environment name leaked in output: %s", output.String())
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(output.Bytes(), &object); err != nil {
		t.Fatal(err)
	}
	if len(object) != 2 || object["revision"] == nil || object["models"] == nil {
		t.Fatalf("unexpected top-level UI config fields: %s", output.String())
	}
	var models map[string]json.RawMessage
	if err := json.Unmarshal(object["models"], &models); err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models["default"] == nil {
		t.Fatalf("unexpected UI model fields: %s", object["models"])
	}
	var got config.UIConfig
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Models.Default.APIKeyConfigured || got.Models.Default.APIKey != nil {
		t.Fatalf("unexpected key state: %+v", got.Models.Default)
	}
}

func TestConfigCommandUpdateJSONUsesStdinAndDoesNotEchoSecret(t *testing.T) {
	const apiKeyEnvLine = `api_key_env   =   "DEVICE_MANAGED_KEY" # preserve exact value`
	original := strings.Replace(config.DefaultTOML(), `api_key_env = "OPENAI_API_KEY"`, apiKeyEnvLine, 1)
	setupConfigCommandRoot(t, original)
	request, err := config.ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	request.Models.Default.Model = "gpt-updated"
	secret := "new-secret-value"
	request.Models.Default.APIKey = &secret
	input, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := configCommand([]string{"update", "--json"}, bytes.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), secret) {
		t.Fatalf("secret leaked in output: %s", output.String())
	}
	var result config.UIUpdateResult
	if err := json.Unmarshal(output.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Config.Models.Default.Model != "gpt-updated" || !result.Config.Models.Default.APIKeyConfigured {
		t.Fatalf("unexpected update result: %+v", result)
	}
	raw, _ := os.ReadFile(paths.ConfigFile())
	if !strings.Contains(string(raw), `api_key = "new-secret-value"`) {
		t.Fatalf("secret was not persisted:\n%s", raw)
	}
	if bytes.Count(raw, []byte(apiKeyEnvLine+"\n")) != 1 {
		t.Fatalf("API key environment line was not preserved byte-for-byte:\n%s", raw)
	}
}

func TestConfigCommandUpdateRollsBackWhenOutputFails(t *testing.T) {
	setupConfigCommandRoot(t, config.DefaultTOML())
	request, err := config.ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	request.Models.Default.Model = "must-roll-back"
	input, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}

	err = configCommand([]string{"update", "--json"}, bytes.NewReader(input), failingConfigWriter{})
	if err == nil || !strings.Contains(err.Error(), "encode config update result") {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, readErr := os.ReadFile(paths.ConfigFile())
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != config.DefaultTOML() {
		t.Fatalf("config was not rolled back after output failure:\n%s", raw)
	}
}

func TestConfigCommandRevisionRawMatchesUIRevisionWithoutEchoingSecret(t *testing.T) {
	secret := "revision-secret-value"
	raw := strings.Replace(config.DefaultTOML(), `# api_key = ""`, `api_key = "`+secret+`"`, 1)
	setupConfigCommandRoot(t, raw)
	ui, err := config.ReadUI()
	if err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := configCommand([]string{"revision", "--raw"}, strings.NewReader(raw), &output); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output.String(), secret) {
		t.Fatalf("secret leaked in revision output: %s", output.String())
	}
	if strings.TrimSpace(output.String()) != ui.Revision {
		t.Fatalf("revision=%q want=%q", strings.TrimSpace(output.String()), ui.Revision)
	}
}

func TestConfigCommandRejectsUnknownJSONFields(t *testing.T) {
	setupConfigCommandRoot(t, config.DefaultTOML())
	for name, request := range map[string]string{
		"top-level config section": `{"revision":"x","capture":{}}`,
		"non-default model":        `{"revision":"x","models":{"compact":{}}}`,
		"API key environment":      `{"revision":"x","models":{"default":{"apiKeyEnv":"OPENAI_API_KEY"}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			err := configCommand([]string{"update", "--json"}, strings.NewReader(request), &output)
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigCommandRestoreRaw(t *testing.T) {
	setupConfigCommandRoot(t, config.DefaultTOML())
	current, err := config.ReadUI()
	if err != nil {
		t.Fatal(err)
	}
	backup := strings.Replace(config.DefaultTOML(), `model = "gpt-5.5"`, `model = "restored-model"`, 1)
	if err := configCommand(
		[]string{"restore", "--raw", "--if-revision", current.Revision},
		strings.NewReader(backup),
		&bytes.Buffer{},
	); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != backup {
		t.Fatalf("restore mismatch:\n%s", raw)
	}
}

func TestConfigCommandRestoreRejectsRevisionConflict(t *testing.T) {
	setupConfigCommandRoot(t, config.DefaultTOML())
	backup := strings.Replace(config.DefaultTOML(), `model = "gpt-5.5"`, `model = "must-not-restore"`, 1)

	err := configCommand(
		[]string{"restore", "--raw", "--if-revision", strings.Repeat("0", 64)},
		strings.NewReader(backup),
		&bytes.Buffer{},
	)
	if !errors.Is(err, config.ErrRevisionConflict) {
		t.Fatalf("err=%v, want revision conflict", err)
	}
	raw, readErr := os.ReadFile(paths.ConfigFile())
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(raw) != config.DefaultTOML() {
		t.Fatalf("config changed after restore conflict:\n%s", raw)
	}
}

func setupConfigCommandRoot(t *testing.T, text string) {
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

type failingConfigWriter struct{}

func (failingConfigWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
