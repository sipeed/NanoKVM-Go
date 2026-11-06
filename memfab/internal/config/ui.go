package config

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/BurntSushi/toml"
	"memfab/internal/paths"
)

var ErrRevisionConflict = errors.New("config revision conflict")

var syncConfigDirectory = func(directory *os.File) error {
	return directory.Sync()
}

const (
	revisionKeyBytes       = 32
	maxModelTokens         = 1_048_576
	maxModelTimeoutSeconds = 3_600
)

type UIModelConfig struct {
	Model            string  `json:"model"`
	BaseURL          string  `json:"baseUrl"`
	APIKeyConfigured bool    `json:"apiKeyConfigured"`
	APIKey           *string `json:"apiKey,omitempty"`
	MaxTokens        int     `json:"maxTokens"`
	TimeoutSeconds   int     `json:"timeoutSeconds"`
}

type UIModelsConfig struct {
	Default UIModelConfig `json:"default"`
}

type UIConfig struct {
	Revision string         `json:"revision"`
	Models   UIModelsConfig `json:"models"`
}

// uiTOMLConfig deliberately describes only the portion of config.toml that
// the UI owns. BurntSushi/toml still parses the complete document, but values
// in hidden sections are left undecoded so their types and ranges cannot make
// the default-model UI unusable.
type uiTOMLModelConfig struct {
	Model          string `toml:"model"`
	BaseURL        string `toml:"base_url"`
	APIKey         string `toml:"api_key"`
	MaxTokens      int    `toml:"max_tokens"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

type uiTOMLModelsConfig struct {
	Default uiTOMLModelConfig `toml:"default"`
}

type uiTOMLConfig struct {
	Models uiTOMLModelsConfig `toml:"models"`
}

type UIUpdateResult struct {
	Config  UIConfig `json:"config"`
	Changed bool     `json:"changed"`
}

// UIUpdateTransaction carries the pre-update snapshot only in process memory.
// Its embedded result is safe to encode as JSON; previousRaw is deliberately
// unexported so config contents and API keys cannot be serialized by accident.
type UIUpdateTransaction struct {
	UIUpdateResult
	previousRaw []byte
}

// Rollback restores the transaction's previous contents only when the config
// still has the revision produced by this update.
func (transaction UIUpdateTransaction) Rollback() error {
	if !transaction.Changed {
		return nil
	}
	if len(transaction.previousRaw) == 0 || transaction.Config.Revision == "" {
		return errors.New("config update transaction cannot be rolled back")
	}
	return Restore(transaction.previousRaw, transaction.Config.Revision)
}

type tomlUpdate struct {
	section string
	key     string
	value   string
	remove  bool
}

func ReadUI() (UIConfig, error) {
	lock, err := lockConfig()
	if err != nil {
		return UIConfig{}, err
	}
	defer unlockConfig(lock)

	data, cfg, err := readRawConfig()
	if err != nil {
		return UIConfig{}, err
	}
	revision, err := configRevision(data)
	if err != nil {
		return UIConfig{}, err
	}
	return uiConfigFromConfig(cfg, revision), nil
}

func UpdateUI(request UIConfig) (UIConfig, bool, error) {
	transaction, err := UpdateUITransaction(request)
	return transaction.Config, transaction.Changed, err
}

// UpdateUITransaction applies a UI update and retains an in-memory snapshot
// that callers can use to CAS-rollback a later output or transport failure.
func UpdateUITransaction(request UIConfig) (UIUpdateTransaction, error) {
	if err := validateUIConfig(request); err != nil {
		return UIUpdateTransaction{}, err
	}

	lock, err := lockConfig()
	if err != nil {
		return UIUpdateTransaction{}, err
	}
	defer unlockConfig(lock)

	current, _, err := readRawConfig()
	if err != nil {
		return UIUpdateTransaction{}, err
	}
	currentRevision, err := configRevision(current)
	if err != nil {
		return UIUpdateTransaction{}, err
	}
	if request.Revision == "" || !hmac.Equal([]byte(request.Revision), []byte(currentRevision)) {
		return UIUpdateTransaction{}, ErrRevisionConflict
	}

	updated := patchTOML(string(current), uiTOMLUpdates(request))
	updatedBytes := []byte(updated)
	updatedConfig, err := decodeRawConfig(updatedBytes)
	if err != nil {
		return UIUpdateTransaction{}, fmt.Errorf("validate updated config: %w", err)
	}
	if string(current) == updated {
		return UIUpdateTransaction{UIUpdateResult: UIUpdateResult{
			Config: uiConfigFromConfig(updatedConfig, currentRevision),
		}}, nil
	}
	updatedRevision, err := configRevision(updatedBytes)
	if err != nil {
		return UIUpdateTransaction{}, err
	}
	if err := writeConfig(updatedBytes); err != nil {
		return UIUpdateTransaction{}, err
	}
	return UIUpdateTransaction{
		UIUpdateResult: UIUpdateResult{
			Config:  uiConfigFromConfig(updatedConfig, updatedRevision),
			Changed: true,
		},
		previousRaw: append([]byte(nil), current...),
	}, nil
}

// Restore performs a compare-and-swap restore. The current config revision is
// checked under the same cross-process lock used for the atomic write.
func Restore(raw []byte, expectedCurrentRevision string) error {
	if len(raw) == 0 {
		return errors.New("config restore payload is empty")
	}
	if expectedCurrentRevision == "" {
		return ErrRevisionConflict
	}
	if _, err := decodeRawConfig(raw); err != nil {
		return fmt.Errorf("validate restore config: %w", err)
	}

	lock, err := lockConfig()
	if err != nil {
		return err
	}
	defer unlockConfig(lock)
	current, _, err := readRawConfig()
	if err != nil {
		return err
	}
	currentRevision, err := configRevision(current)
	if err != nil {
		return err
	}
	if !hmac.Equal([]byte(expectedCurrentRevision), []byte(currentRevision)) {
		return ErrRevisionConflict
	}
	return writeConfig(raw)
}

// Revision validates raw TOML and returns the same opaque, device-bound HMAC
// revision used by the UI APIs. It never exposes a bare hash of config data.
func Revision(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "", errors.New("config revision payload is empty")
	}
	if _, err := decodeRawConfig(raw); err != nil {
		return "", fmt.Errorf("validate revision config: %w", err)
	}
	lock, err := lockConfig()
	if err != nil {
		return "", err
	}
	defer unlockConfig(lock)
	return configRevision(raw)
}

func readRawConfig() ([]byte, uiTOMLConfig, error) {
	if _, err := WriteDefaultIfMissing(); err != nil {
		return nil, uiTOMLConfig{}, err
	}
	data, err := os.ReadFile(paths.ConfigFile())
	if err != nil {
		return nil, uiTOMLConfig{}, err
	}
	cfg, err := decodeRawConfig(data)
	return data, cfg, err
}

func decodeRawConfig(data []byte) (uiTOMLConfig, error) {
	defaults := Default().Models.Default
	cfg := uiTOMLConfig{Models: uiTOMLModelsConfig{Default: uiTOMLModelConfig{
		Model:          defaults.Model,
		BaseURL:        defaults.BaseURL,
		APIKey:         defaults.APIKey,
		MaxTokens:      defaults.MaxTokens,
		TimeoutSeconds: defaults.TimeoutSeconds,
	}}}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return uiTOMLConfig{}, err
	}
	return cfg, nil
}

func uiConfigFromConfig(cfg uiTOMLConfig, revision string) UIConfig {
	return UIConfig{
		Revision: revision,
		Models: UIModelsConfig{
			Default: uiModelFromConfig(cfg.Models.Default),
		},
	}
}

func uiModelFromConfig(cfg uiTOMLModelConfig) UIModelConfig {
	return UIModelConfig{
		Model: cfg.Model, BaseURL: cfg.BaseURL,
		APIKeyConfigured: cfg.APIKey != "", MaxTokens: cfg.MaxTokens, TimeoutSeconds: cfg.TimeoutSeconds,
	}
}

func validateUIConfig(cfg UIConfig) error {
	return validateModel("models.default", cfg.Models.Default)
}

func validateModel(name string, model UIModelConfig) error {
	if strings.TrimSpace(model.Model) == "" {
		return fmt.Errorf("%s.model is required", name)
	}
	if model.BaseURL != "" {
		parsed, err := url.Parse(model.BaseURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
			return fmt.Errorf("%s.baseUrl must be an HTTP or HTTPS URL", name)
		}
	}
	if model.MaxTokens <= 0 || model.MaxTokens > maxModelTokens {
		return fmt.Errorf("%s.maxTokens must be between 1 and %d", name, maxModelTokens)
	}
	if model.TimeoutSeconds <= 0 || model.TimeoutSeconds > maxModelTimeoutSeconds {
		return fmt.Errorf("%s.timeoutSeconds must be between 1 and %d", name, maxModelTimeoutSeconds)
	}
	return nil
}

func uiTOMLUpdates(cfg UIConfig) []tomlUpdate {
	return modelTOMLUpdates("models.default", cfg.Models.Default)
}

func modelTOMLUpdates(section string, model UIModelConfig) []tomlUpdate {
	value := func(key, text string) tomlUpdate {
		return tomlUpdate{section: section, key: key, value: strconv.Quote(text)}
	}
	number := func(key string, number int) tomlUpdate {
		return tomlUpdate{section: section, key: key, value: strconv.Itoa(number)}
	}
	updates := []tomlUpdate{
		value("model", model.Model), value("base_url", model.BaseURL),
		number("max_tokens", model.MaxTokens), number("timeout_seconds", model.TimeoutSeconds),
	}
	if model.BaseURL == "" {
		updates[1].remove = true
	}
	if model.APIKey != nil {
		if *model.APIKey == "" {
			updates = append(updates, tomlUpdate{section: section, key: "api_key", remove: true})
		} else {
			updates = append(updates, tomlUpdate{section: section, key: "api_key", value: strconv.Quote(*model.APIKey)})
		}
	}
	return updates
}

func patchTOML(original string, updates []tomlUpdate) string {
	bySection := make(map[string]map[string]tomlUpdate)
	sectionOrder := make([]string, 0)
	for _, update := range updates {
		if _, ok := bySection[update.section]; !ok {
			bySection[update.section] = make(map[string]tomlUpdate)
			sectionOrder = append(sectionOrder, update.section)
		}
		bySection[update.section][update.key] = update
	}

	lines := strings.Split(strings.TrimSuffix(original, "\n"), "\n")
	output := make([]string, 0, len(lines)+len(updates))
	seenSections := make(map[string]bool)
	seenKeys := make(map[string]bool)
	currentSection := ""
	multiline := tomlMultilineNone
	discardMultiline := false
	flushMissing := func(section string) {
		for _, update := range updates {
			id := update.section + "\x00" + update.key
			if update.section == section && !seenKeys[id] && !update.remove {
				output = append(output, update.key+" = "+update.value)
				seenKeys[id] = true
			}
		}
	}

	for _, line := range lines {
		if multiline != tomlMultilineNone {
			if !discardMultiline {
				output = append(output, line)
			}
			multiline = nextTOMLMultilineState(line, multiline)
			if multiline == tomlMultilineNone {
				discardMultiline = false
			}
			continue
		}
		lineMultiline := nextTOMLMultilineState(line, tomlMultilineNone)
		if section, array, ok := parseTOMLTableHeader(line); ok {
			flushMissing(currentSection)
			if array {
				// Array-of-table entries are independent table bodies. They must
				// never inherit the preceding managed section while patching.
				currentSection = ""
			} else {
				currentSection = section
				seenSections[section] = true
			}
			output = append(output, line)
			multiline = lineMultiline
			continue
		}
		key, ok := parseTOMLKey(line)
		if !ok {
			output = append(output, line)
			multiline = lineMultiline
			continue
		}
		update, managed := bySection[currentSection][key]
		if !managed {
			output = append(output, line)
			multiline = lineMultiline
			continue
		}
		seenKeys[currentSection+"\x00"+key] = true
		if !update.remove {
			output = append(output, replaceTOMLValue(line, update.value))
		}
		multiline = lineMultiline
		discardMultiline = lineMultiline != tomlMultilineNone
	}
	flushMissing(currentSection)

	for _, section := range sectionOrder {
		if seenSections[section] {
			continue
		}
		sectionLines := make([]string, 0)
		for _, update := range updates {
			if update.section == section && !update.remove {
				sectionLines = append(sectionLines, update.key+" = "+update.value)
			}
		}
		if len(sectionLines) == 0 {
			continue
		}
		if len(output) > 0 && output[len(output)-1] != "" {
			output = append(output, "")
		}
		output = append(output, "["+section+"]")
		output = append(output, sectionLines...)
	}
	return strings.Join(output, "\n") + "\n"
}

func parseTOMLTableHeader(line string) (section string, array bool, ok bool) {
	trimmed := strings.TrimSpace(stripTOMLComment(line))
	if len(trimmed) >= 5 && strings.HasPrefix(trimmed, "[[") && strings.HasSuffix(trimmed, "]]") {
		return strings.TrimSpace(trimmed[2 : len(trimmed)-2]), true, true
	}
	if len(trimmed) < 3 || !strings.HasPrefix(trimmed, "[") || !strings.HasSuffix(trimmed, "]") {
		return "", false, false
	}
	return strings.TrimSpace(trimmed[1 : len(trimmed)-1]), false, true
}

type tomlMultilineState uint8

const (
	tomlMultilineNone tomlMultilineState = iota
	tomlMultilineBasic
	tomlMultilineLiteral
)

// nextTOMLMultilineState tracks triple-quoted strings without interpreting
// their contents as tables or key/value assignments. Single-line strings and
// comments are skipped so delimiter-looking text inside them is harmless.
func nextTOMLMultilineState(line string, state tomlMultilineState) tomlMultilineState {
	for index := 0; index < len(line); {
		switch state {
		case tomlMultilineBasic:
			if line[index] == '\\' {
				index += 2
				continue
			}
			if strings.HasPrefix(line[index:], `"""`) {
				state = tomlMultilineNone
				index += 3
				continue
			}
			index++
			continue
		case tomlMultilineLiteral:
			if strings.HasPrefix(line[index:], `'''`) {
				state = tomlMultilineNone
				index += 3
				continue
			}
			index++
			continue
		}

		if line[index] == '#' {
			return state
		}
		if strings.HasPrefix(line[index:], `"""`) {
			state = tomlMultilineBasic
			index += 3
			continue
		}
		if strings.HasPrefix(line[index:], `'''`) {
			state = tomlMultilineLiteral
			index += 3
			continue
		}
		switch line[index] {
		case '"':
			index++
			for index < len(line) {
				if line[index] == '\\' {
					index += 2
					continue
				}
				if line[index] == '"' {
					index++
					break
				}
				index++
			}
		case '\'':
			index++
			for index < len(line) && line[index] != '\'' {
				index++
			}
			if index < len(line) {
				index++
			}
		default:
			index++
		}
	}
	return state
}

func parseTOMLKey(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return "", false
	}
	idx := strings.Index(line, "=")
	if idx < 0 {
		return "", false
	}
	key := strings.TrimSpace(line[:idx])
	if key == "" || strings.ContainsAny(key, "\"'") {
		return "", false
	}
	return key, true
}

func replaceTOMLValue(line, value string) string {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return line
	}
	comment := inlineTOMLComment(line[idx+1:])
	replaced := strings.TrimRight(line[:idx+1], " \t") + " " + value
	if comment != "" {
		replaced += " " + comment
	}
	return replaced
}

func stripTOMLComment(line string) string {
	if idx := tomlCommentIndex(line); idx >= 0 {
		return line[:idx]
	}
	return line
}

func inlineTOMLComment(value string) string {
	if idx := tomlCommentIndex(value); idx >= 0 {
		return strings.TrimSpace(value[idx:])
	}
	return ""
}

func tomlCommentIndex(text string) int {
	var quote rune
	escaped := false
	for index, ch := range text {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && ch == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			quote = ch
			continue
		}
		if ch == '#' {
			return index
		}
	}
	return -1
}

func configRevision(data []byte) (string, error) {
	key, err := loadRevisionKey()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(data); err != nil {
		return "", err
	}
	return "v1:" + hex.EncodeToString(mac.Sum(nil)), nil
}

func loadRevisionKey() ([]byte, error) {
	path := paths.ConfigRevisionKey()
	key, err := readRevisionKey(path)
	if err == nil {
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	candidate := make([]byte, revisionKeyBytes)
	if _, err := rand.Read(candidate); err != nil {
		return nil, fmt.Errorf("generate config revision key: %w", err)
	}
	if _, err := writeFileNoReplace(path, candidate, 0o600); err != nil {
		return nil, fmt.Errorf("create config revision key: %w", err)
	}
	// Always reopen the published path: another process may have won the
	// no-replace race with a different random key.
	return readRevisionKey(path)
}

func readRevisionKey(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("config revision key is not a regular file")
	}
	if info.Mode().Perm() != 0o600 {
		if err := os.Chmod(path, 0o600); err != nil {
			return nil, fmt.Errorf("secure config revision key permissions: %w", err)
		}
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(key) != revisionKeyBytes {
		return nil, fmt.Errorf("config revision key must be %d bytes", revisionKeyBytes)
	}
	return key, nil
}

func writeConfig(data []byte) error {
	dir := filepath.Dir(paths.ConfigFile())
	if err := os.MkdirAll(dir, paths.DirectoryMode); err != nil {
		return err
	}
	previous, readErr := os.ReadFile(paths.ConfigFile())
	previousExists := readErr == nil
	if readErr != nil && !os.IsNotExist(readErr) {
		return readErr
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()

	tmpPath, err := writeConfigTemp(dir, data)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	if err := os.Rename(tmpPath, paths.ConfigFile()); err != nil {
		return err
	}
	if err := syncConfigDirectory(directory); err != nil {
		if rollbackErr := rollbackConfigWrite(dir, directory, previous, previousExists); rollbackErr != nil {
			return fmt.Errorf("sync config directory: %w (rollback failed: %v)", err, rollbackErr)
		}
		return fmt.Errorf("sync config directory: %w", err)
	}
	return nil
}

func writeConfigTemp(dir string, data []byte) (string, error) {
	tmp, err := os.CreateTemp(dir, ".config.toml.*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	failed := true
	defer func() {
		if failed {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	failed = false
	return tmpPath, nil
}

func rollbackConfigWrite(dir string, directory *os.File, previous []byte, previousExists bool) error {
	if !previousExists {
		if err := os.Remove(paths.ConfigFile()); err != nil && !os.IsNotExist(err) {
			return err
		}
		return syncConfigDirectory(directory)
	}
	tmpPath, err := writeConfigTemp(dir, previous)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	if err := os.Rename(tmpPath, paths.ConfigFile()); err != nil {
		return err
	}
	return syncConfigDirectory(directory)
}

func lockConfig() (*os.File, error) {
	if err := paths.EnsureDirs(); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(paths.ConfigFile()+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		lock.Close()
		return nil, err
	}
	return lock, nil
}

func unlockConfig(lock *os.File) {
	_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	_ = lock.Close()
}
