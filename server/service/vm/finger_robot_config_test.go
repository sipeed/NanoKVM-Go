package vm

import (
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestDefaultFingerRobotConfig(t *testing.T) {
	got := DefaultFingerRobotConfig()
	want := FingerRobotConfig{
		Version:   1,
		Revision:  1,
		ActiveArm: FingerRobotArmProfileShortID,
		StrengthByArm: FingerRobotStrengthByArm{
			Long:  3,
			Short: 3,
		},
		PressMS: 800,
		Speed:   FingerRobotSpeedSlow,
	}
	if got != want {
		t.Fatalf("default config = %+v, want %+v", got, want)
	}
	if err := validateFingerRobotConfig(got); err != nil {
		t.Fatalf("default config is invalid: %v", err)
	}

	if speed, err := ResolveFingerRobotSpeedUSPerSecond(FingerRobotSpeedSlow); err != nil || speed != 1000 {
		t.Fatalf("slow speed = %d, %v; want 1000, nil", speed, err)
	}
	if speed, err := ResolveFingerRobotSpeedUSPerSecond(FingerRobotSpeedFast); err != nil || speed != 2000 {
		t.Fatalf("fast speed = %d, %v; want 2000, nil", speed, err)
	}
	if _, err := ResolveFingerRobotSpeedUSPerSecond("medium"); !errors.Is(err, ErrFingerRobotConfigInvalid) {
		t.Fatalf("invalid speed error = %v, want ErrFingerRobotConfigInvalid", err)
	}
}

func TestFingerRobotConfigStoreCreatesMissingDefaultAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kvm", "finger-robot.json")
	store := newFingerRobotConfigStore(path)

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}
	if want := DefaultFingerRobotConfig(); got != want {
		t.Fatalf("created config = %+v, want %+v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat created config: %v", err)
	}
	if gotMode := info.Mode().Perm(); gotMode != 0o600 {
		t.Fatalf("created config mode = %o, want 600", gotMode)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created config: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("created config must end with a newline: %q", data)
	}
	decoded, err := decodeFingerRobotConfig(data)
	if err != nil {
		t.Fatalf("decode created config: %v", err)
	}
	if decoded != got {
		t.Fatalf("persisted config = %+v, want %+v", decoded, got)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read config directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("config directory entries = %v, want only %q", entryNames(entries), filepath.Base(path))
	}
}

func TestFingerRobotConfigStoreDoesNotReplaceInvalidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finger-robot.json")
	original := []byte(`{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow","unknown":true}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("write invalid config: %v", err)
	}

	store := newFingerRobotConfigStore(path)
	if _, err := store.Load(); !errors.Is(err, ErrFingerRobotConfigInvalid) {
		t.Fatalf("load invalid config error = %v, want ErrFingerRobotConfigInvalid", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read invalid config after load: %v", err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("invalid config was replaced:\n got %q\nwant %q", got, original)
	}
}

func TestDecodeFingerRobotConfigStrictValidation(t *testing.T) {
	valid := `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`
	tests := []struct {
		name string
		data string
	}{
		{name: "invalid JSON", data: `{`},
		{name: "unknown root field", data: strings.TrimSuffix(valid, `}`) + `,"unknown":true}`},
		{name: "unknown profile field", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3,"medium":3},"pressMs":800,"speed":"slow"}`},
		{name: "missing version", data: `{"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "null active arm", data: `{"version":1,"revision":1,"activeArm":null,"strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "null strength", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":null,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "wrong version", data: `{"version":2,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "zero revision", data: `{"version":1,"revision":0,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "negative revision", data: `{"version":1,"revision":-1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "unknown active profile", data: `{"version":1,"revision":1,"activeArm":"medium","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "long strength below range", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":0,"short":3},"pressMs":800,"speed":"slow"}`},
		{name: "short strength above range", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":6},"pressMs":800,"speed":"slow"}`},
		{name: "press time below range", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":99,"speed":"slow"}`},
		{name: "press time above range", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":30001,"speed":"slow"}`},
		{name: "unknown speed", data: `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"medium"}`},
		{name: "multiple JSON values", data: valid + ` {}`},
	}

	if _, err := decodeFingerRobotConfig([]byte(valid)); err != nil {
		t.Fatalf("decode valid config: %v", err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := decodeFingerRobotConfig([]byte(test.data)); !errors.Is(err, ErrFingerRobotConfigInvalid) {
				t.Fatalf("decode error = %v, want ErrFingerRobotConfigInvalid", err)
			}
		})
	}
}

func TestFingerRobotConfigStoreUpdateAndNoOp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finger-robot.json")
	store := newFingerRobotConfigStore(path)
	initial, err := store.Load()
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}
	update := FingerRobotConfigUpdate{
		ActiveArm: FingerRobotArmProfileLongID,
		StrengthByArm: FingerRobotStrengthByArm{
			Long:  4,
			Short: 2,
		},
		PressMS: 1200,
		Speed:   FingerRobotSpeedFast,
	}

	updated, changed, err := store.Update(initial.Revision, update)
	if err != nil {
		t.Fatalf("update config: %v", err)
	}
	if !changed || updated.Revision != initial.Revision+1 || updated.userSettings() != update {
		t.Fatalf("updated config = %+v, changed=%t", updated, changed)
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatalf("reload updated config: %v", err)
	}
	if persisted != updated {
		t.Fatalf("persisted config = %+v, want %+v", persisted, updated)
	}

	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before no-op: %v", err)
	}
	noOp, changed, err := store.Update(updated.Revision, update)
	if err != nil {
		t.Fatalf("no-op update: %v", err)
	}
	if changed || noOp != updated {
		t.Fatalf("no-op result = %+v, changed=%t; want unchanged", noOp, changed)
	}
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after no-op: %v", err)
	}
	if !os.SameFile(beforeInfo, afterInfo) {
		t.Fatal("no-op update replaced the config file")
	}
}

func TestFingerRobotConfigStoreRejectsRevisionConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finger-robot.json")
	store := newFingerRobotConfigStore(path)
	current, err := store.Load()
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}
	update := current.userSettings()
	update.PressMS++

	got, changed, err := store.Update(current.Revision+1, update)
	if !errors.Is(err, ErrFingerRobotConfigRevisionConflict) {
		t.Fatalf("conflict error = %v, want ErrFingerRobotConfigRevisionConflict", err)
	}
	if changed || got != current {
		t.Fatalf("conflict result = %+v, changed=%t; want current config", got, changed)
	}
	persisted, loadErr := store.Load()
	if loadErr != nil || persisted != current {
		t.Fatalf("config changed after conflict: %+v, %v", persisted, loadErr)
	}
}

func TestFingerRobotConfigStoreRejectsInvalidUpdateBeforeCreatingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finger-robot.json")
	store := newFingerRobotConfigStore(path)
	update := DefaultFingerRobotConfig().userSettings()
	update.StrengthByArm.Short = 0

	if _, changed, err := store.Update(1, update); !errors.Is(err, ErrFingerRobotConfigInvalid) || changed {
		t.Fatalf("invalid update error = %v, changed=%t; want invalid and unchanged", err, changed)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("invalid update created config file: %v", err)
	}
}

func TestFingerRobotConfigStoreSerializesConcurrentRevisionUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finger-robot.json")
	store := newFingerRobotConfigStore(path)
	current, err := store.Load()
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}
	update := current.userSettings()
	update.PressMS++

	const workers = 16
	start := make(chan struct{})
	type result struct {
		config  FingerRobotConfig
		changed bool
		err     error
	}
	results := make(chan result, workers)
	var wait sync.WaitGroup
	for range workers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			config, changed, err := store.Update(current.Revision, update)
			results <- result{config: config, changed: changed, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)

	changedCount := 0
	conflictCount := 0
	for result := range results {
		switch {
		case result.err == nil && result.changed:
			changedCount++
			if result.config.Revision != current.Revision+1 {
				t.Errorf("winning revision = %d, want %d", result.config.Revision, current.Revision+1)
			}
		case errors.Is(result.err, ErrFingerRobotConfigRevisionConflict) && !result.changed:
			conflictCount++
		default:
			t.Errorf("unexpected concurrent result: %+v", result)
		}
	}
	if changedCount != 1 || conflictCount != workers-1 {
		t.Fatalf("changed=%d conflicts=%d, want 1 and %d", changedCount, conflictCount, workers-1)
	}
	persisted, err := store.Load()
	if err != nil {
		t.Fatalf("load concurrent result: %v", err)
	}
	if persisted.Revision != current.Revision+1 || persisted.userSettings() != update {
		t.Fatalf("persisted concurrent result = %+v", persisted)
	}
}

func TestFingerRobotConfigStoreRejectsRevisionExhaustion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "finger-robot.json")
	current := DefaultFingerRobotConfig()
	current.Revision = math.MaxInt64
	if err := saveFingerRobotConfigToPath(path, current); err != nil {
		t.Fatalf("save maximum revision config: %v", err)
	}
	update := current.userSettings()
	update.PressMS++

	got, changed, err := newFingerRobotConfigStore(path).Update(current.Revision, update)
	if !errors.Is(err, ErrFingerRobotConfigRevisionExhausted) {
		t.Fatalf("revision exhaustion error = %v, want ErrFingerRobotConfigRevisionExhausted", err)
	}
	if changed || got != current {
		t.Fatalf("revision exhaustion result = %+v, changed=%t; want current", got, changed)
	}
}

func TestSaveFingerRobotConfigCleansTemporaryFileAfterRenameFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "finger-robot.json")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("create directory at destination: %v", err)
	}
	if err := saveFingerRobotConfigToPath(path, DefaultFingerRobotConfig()); err == nil {
		t.Fatal("save over directory unexpectedly succeeded")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read config directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("temporary config was not cleaned up: %v", entryNames(entries))
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestFingerRobotConfigJSONShape(t *testing.T) {
	data, err := json.Marshal(DefaultFingerRobotConfig())
	if err != nil {
		t.Fatalf("marshal default config: %v", err)
	}
	want := `{"version":1,"revision":1,"activeArm":"short","strengthByArm":{"long":3,"short":3},"pressMs":800,"speed":"slow"}`
	if string(data) != want {
		t.Fatalf("default JSON = %s, want %s", data, want)
	}
}
