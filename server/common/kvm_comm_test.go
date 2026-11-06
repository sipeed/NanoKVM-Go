package common

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"NanoKVM-Server/utils"

	"github.com/pelletier/go-toml/v2"
)

func TestSetKvmCommVideoModeRestartsWatchedKvmVinProcess(t *testing.T) {
	configPath := setupKvmCommConfigTest(t, KvmCommVideoModeH264)

	var gotOpts utils.RestartWatchedProcessOptions
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		gotOpts = opts
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommVideoMode(KvmCommVideoModeH265, true, 0); err != nil {
		t.Fatalf("set kvmcomm video mode: %v", err)
	}

	if gotOpts.ExecPath != kvmVinExecPath {
		t.Fatalf("expected kvm_vin exec path %q, got %q", kvmVinExecPath, gotOpts.ExecPath)
	}
	if gotOpts.ReadyCheck == nil {
		t.Fatalf("expected ready check")
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get kvmcomm config: %v", err)
	}
	if config.Device.Mode != KvmCommVideoModeH265 {
		t.Fatalf("expected mode %s, got %s", KvmCommVideoModeH265, config.Device.Mode)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}
}

func TestSetKvmCommVideoModeSameModeDoesNotRestart(t *testing.T) {
	setupKvmCommConfigTest(t, KvmCommVideoModeH264)

	calls := 0
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommVideoMode(KvmCommVideoModeH264, true, 0); err != nil {
		t.Fatalf("set same kvmcomm video mode: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected no restart for unchanged mode, got %d calls", calls)
	}
}

func TestSetKvmCommVideoModeRestoresConfigWhenRestartFails(t *testing.T) {
	setupKvmCommConfigTest(t, KvmCommVideoModeH264)

	calls := 0
	restartErr := errors.New("restart failed")
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		if calls == 1 {
			return restartErr
		}
		return nil
	})
	defer restoreRestart()

	err := SetKvmCommVideoMode(KvmCommVideoModeH265, true, 0)
	if !errors.Is(err, ErrKvmCommRestartFailed) {
		t.Fatalf("expected restart failed error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected restart and restore restart calls, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get kvmcomm config after restore: %v", err)
	}
	if config.Device.Mode != KvmCommVideoModeH264 {
		t.Fatalf("expected restored mode %s, got %s", KvmCommVideoModeH264, config.Device.Mode)
	}
}

func TestKvmCommEncoderDefaults(t *testing.T) {
	setupKvmCommConfigTestData(t, `[device]
mode = "h264"

[log]
file = "/var/log/kvmcomm/vin.log"
level = "error"
type = "file"
`)

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}

	want := KvmCommEncoderConfig{
		RateControl: KvmCommEncoderRateControlVBR,
		BitRate:     kvmCommDefaultEncoderBitRate,
		GOP:         kvmCommDefaultEncoderGOP,
		FPS:         kvmCommDefaultEncoderFPS,
	}
	if config.Encoder != want {
		t.Fatalf("unexpected encoder defaults: got %#v want %#v", config.Encoder, want)
	}
}

func TestKvmCommEncoderNormalizationIsReadOnly(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    KvmCommEncoderConfig
	}{
		{
			name: "mixed valid and invalid values",
			content: `[encoder]
rate_control = "cbr"
bitrate = 999
gop = 180
fps = 121
`,
			want: KvmCommEncoderConfig{
				RateControl: KvmCommEncoderRateControlCBR,
				BitRate:     kvmCommDefaultEncoderBitRate,
				GOP:         180,
				FPS:         kvmCommDefaultEncoderFPS,
			},
		},
		{
			name: "wrong field types",
			content: `[encoder]
rate_control = 1
bitrate = "12000"
gop = false
fps = "30"
`,
			want: KvmCommEncoderConfig{
				RateControl: KvmCommEncoderRateControlVBR,
				BitRate:     kvmCommDefaultEncoderBitRate,
				GOP:         kvmCommDefaultEncoderGOP,
				FPS:         kvmCommDefaultEncoderFPS,
			},
		},
		{
			name:    "wrong encoder section type",
			content: "encoder = 1\n",
			want: KvmCommEncoderConfig{
				RateControl: KvmCommEncoderRateControlVBR,
				BitRate:     kvmCommDefaultEncoderBitRate,
				GOP:         kvmCommDefaultEncoderGOP,
				FPS:         kvmCommDefaultEncoderFPS,
			},
		},
		{
			name: "lower boundaries",
			content: `[encoder]
rate_control = "cbr"
bitrate = 1000
gop = 1
fps = 0
`,
			want: KvmCommEncoderConfig{
				RateControl: KvmCommEncoderRateControlCBR,
				BitRate:     kvmCommEncoderBitRateMin,
				GOP:         kvmCommEncoderGOPMin,
				FPS:         kvmCommEncoderFPSMin,
			},
		},
		{
			name: "upper boundaries",
			content: `[encoder]
rate_control = "vbr"
bitrate = 20000
gop = 300
fps = 120
`,
			want: KvmCommEncoderConfig{
				RateControl: KvmCommEncoderRateControlVBR,
				BitRate:     kvmCommEncoderBitRateMax,
				GOP:         kvmCommEncoderGOPMax,
				FPS:         kvmCommEncoderFPSMax,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configPath := setupKvmCommConfigTestData(t, test.content)
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config before get: %v", err)
			}

			config, err := GetKvmCommConfig()
			if err != nil {
				t.Fatalf("get config: %v", err)
			}
			if config.Encoder != test.want {
				t.Fatalf("unexpected encoder config: got %#v want %#v", config.Encoder, test.want)
			}

			after, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config after get: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Fatalf("GetKvmCommConfig rewrote the config:\nbefore: %q\nafter: %q", before, after)
			}
		})
	}
}

func TestKvmCommEncoderSaveNormalizesExistingInvalidFields(t *testing.T) {
	configPath := setupKvmCommConfigTestData(t, `[encoder]
rate_control = "abr"
bitrate = "12000"
gop = 0
fps = 121
vendor_tune = "keep-me"
`)

	if err := SetKvmCommEncoderGOP(180); err != nil {
		t.Fatalf("set GOP: %v", err)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	want := KvmCommEncoderConfig{
		RateControl: KvmCommEncoderRateControlVBR,
		BitRate:     kvmCommDefaultEncoderBitRate,
		GOP:         180,
		FPS:         kvmCommDefaultEncoderFPS,
	}
	if config.Encoder != want {
		t.Fatalf("unexpected normalized encoder config: got %#v want %#v", config.Encoder, want)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	var raw kvmCommConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse saved config: %v", err)
	}
	encoder, err := raw.table(kvmCommEncoderSection)
	if err != nil {
		t.Fatalf("get saved encoder section: %v", err)
	}
	if encoder[kvmCommEncoderRateControl] != KvmCommEncoderRateControlVBR {
		t.Fatalf("saved rate control was not normalized: %#v", encoder[kvmCommEncoderRateControl])
	}
	if bitRate, ok := numericTomlValue[int](encoder[kvmCommEncoderBitRate]); !ok || bitRate != kvmCommDefaultEncoderBitRate {
		t.Fatalf("saved bitrate was not normalized: %#v", encoder[kvmCommEncoderBitRate])
	}
	if gop, ok := numericTomlValue[int](encoder[kvmCommEncoderGOP]); !ok || gop != 180 {
		t.Fatalf("saved GOP is unexpected: %#v", encoder[kvmCommEncoderGOP])
	}
	if fps, ok := numericTomlValue[int](encoder[kvmCommEncoderFPS]); !ok || fps != kvmCommDefaultEncoderFPS {
		t.Fatalf("saved FPS was not normalized: %#v", encoder[kvmCommEncoderFPS])
	}
	if encoder["vendor_tune"] != "keep-me" {
		t.Fatalf("saved encoder extension was not preserved: %#v", encoder["vendor_tune"])
	}
}

func TestSetKvmCommEncoderSettingsWritesFieldsAndPreservesUnknownData(t *testing.T) {
	configPath := setupKvmCommConfigTestData(t, `[device]
enable = true
mode = "h264"
vendor_option = "keep-me"

[encoder]
vendor_tune = "keep-encoder"

[custom]
answer = 42
`)
	if err := os.Chmod(configPath, 0o600); err != nil {
		t.Fatalf("chmod config: %v", err)
	}

	if err := SetKvmCommEncoderRateControl("CBR"); err != nil {
		t.Fatalf("set rate control: %v", err)
	}
	if err := SetKvmCommEncoderBitRate(12345); err != nil {
		t.Fatalf("set bitrate: %v", err)
	}
	if err := SetKvmCommEncoderGOP(180); err != nil {
		t.Fatalf("set GOP: %v", err)
	}
	if err := SetKvmCommEncoderFPS(60); err != nil {
		t.Fatalf("set FPS: %v", err)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	want := KvmCommEncoderConfig{
		RateControl: KvmCommEncoderRateControlCBR,
		BitRate:     12345,
		GOP:         180,
		FPS:         60,
	}
	if config.Encoder != want {
		t.Fatalf("unexpected encoder config: got %#v want %#v", config.Encoder, want)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(data)
	var raw kvmCommConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse saved config: %v", err)
	}
	device, err := raw.table(kvmCommDeviceSection)
	if err != nil {
		t.Fatalf("get device section: %v", err)
	}
	if enabled, _ := device["enable"].(bool); !enabled {
		t.Fatalf("expected device.enable to be preserved, got %#v", device["enable"])
	}
	if got, _ := device["vendor_option"].(string); got != "keep-me" {
		t.Fatalf("expected device vendor option to be preserved, got %q", got)
	}
	encoder, err := raw.table(kvmCommEncoderSection)
	if err != nil {
		t.Fatalf("get encoder section: %v", err)
	}
	if got, _ := encoder["vendor_tune"].(string); got != "keep-encoder" {
		t.Fatalf("expected encoder vendor option to be preserved, got %q", got)
	}
	custom, err := raw.table("custom")
	if err != nil {
		t.Fatalf("get custom section: %v", err)
	}
	if answer, _ := numericTomlValue[int](custom["answer"]); answer != 42 {
		t.Fatalf("expected custom answer to be preserved, got %#v", custom["answer"])
	}
	if !strings.HasSuffix(text, "\n") {
		t.Fatalf("expected config to end with a newline, got %q", text)
	}

	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected config mode 0600, got %04o", got)
	}

	entries, err := os.ReadDir(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("read config directory: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(configPath) {
		t.Fatalf("unexpected files after atomic save: %#v", entries)
	}
}

func TestSetKvmCommEncoderSettingsConcurrentUpdatesPreserveFields(t *testing.T) {
	setupKvmCommConfigTest(t, KvmCommVideoModeH264)

	start := make(chan struct{})
	errCh := make(chan error, 4)
	updates := []func() error{
		func() error { return SetKvmCommEncoderRateControl(KvmCommEncoderRateControlCBR) },
		func() error { return SetKvmCommEncoderBitRate(12000) },
		func() error { return SetKvmCommEncoderGOP(240) },
		func() error { return SetKvmCommEncoderFPS(30) },
	}
	for _, update := range updates {
		update := update
		go func() {
			<-start
			errCh <- update()
		}()
	}
	close(start)

	for range updates {
		if err := <-errCh; err != nil {
			t.Fatalf("concurrent encoder update: %v", err)
		}
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	want := KvmCommEncoderConfig{
		RateControl: KvmCommEncoderRateControlCBR,
		BitRate:     12000,
		GOP:         240,
		FPS:         30,
	}
	if config.Encoder != want {
		t.Fatalf("unexpected concurrent encoder config: got %#v want %#v", config.Encoder, want)
	}
}

func TestSetKvmCommEncoderSettingsRejectInvalidValues(t *testing.T) {
	tests := []struct {
		name string
		set  func() error
	}{
		{name: "rate control", set: func() error { return SetKvmCommEncoderRateControl("abr") }},
		{name: "bitrate", set: func() error { return SetKvmCommEncoderBitRate(999) }},
		{name: "GOP", set: func() error { return SetKvmCommEncoderGOP(0) }},
		{name: "FPS", set: func() error { return SetKvmCommEncoderFPS(121) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.set(); !errors.Is(err, ErrKvmCommInvalidEncoderSetting) {
				t.Fatalf("expected invalid encoder setting error, got %v", err)
			}
		})
	}
}

func TestSetKvmCommEncoderSameValueDoesNotRewriteConfig(t *testing.T) {
	configPath := setupKvmCommConfigTest(t, KvmCommVideoModeH264)

	if err := SetKvmCommEncoderBitRate(kvmCommDefaultEncoderBitRate); err != nil {
		t.Fatalf("write encoder defaults: %v", err)
	}
	before, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config before unchanged update: %v", err)
	}

	if err := SetKvmCommEncoderBitRate(kvmCommDefaultEncoderBitRate); err != nil {
		t.Fatalf("write unchanged bitrate: %v", err)
	}
	after, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config after unchanged update: %v", err)
	}
	if !os.SameFile(before, after) {
		t.Fatalf("expected unchanged encoder value to avoid replacing the config file")
	}
}

func TestSetKvmCommEncoderRestoresSnapshotWhenDirectorySyncFails(t *testing.T) {
	configPath := setupKvmCommConfigTest(t, KvmCommVideoModeH264)
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config before update: %v", err)
	}

	oldSyncDir := syncKvmCommConfigDir
	syncCalls := 0
	syncKvmCommConfigDir = func(string) error {
		syncCalls++
		if syncCalls == 1 {
			return errors.New("directory sync failed")
		}
		return nil
	}
	t.Cleanup(func() {
		syncKvmCommConfigDir = oldSyncDir
	})

	err = SetKvmCommEncoderGOP(240)
	if !errors.Is(err, ErrKvmCommSaveConfigFailed) {
		t.Fatalf("expected save config failure, got %v", err)
	}
	if syncCalls != 2 {
		t.Fatalf("expected failed save and snapshot restore syncs, got %d", syncCalls)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config after failed update: %v", err)
	}
	if !bytes.Equal(after, before) {
		t.Fatalf("expected original config snapshot after failed save:\n%s\ngot:\n%s", before, after)
	}
}

func TestSetKvmCommIdleTimeoutEnabledWritesTimeoutAndRestarts(t *testing.T) {
	setupKvmCommConfigTestWithTimeout(t, KvmCommVideoModeH264, 0)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommIdleTimeoutEnabled(true, true, 0); err != nil {
		t.Fatalf("enable idle timeout: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected restart after enabling idle timeout, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.Device.IdleTimeoutMS != kvmCommDefaultIdleTimeoutMS {
		t.Fatalf("expected idle timeout %d, got %d", kvmCommDefaultIdleTimeoutMS, config.Device.IdleTimeoutMS)
	}
}

func TestSetKvmCommIdleTimeoutDisabledWritesZeroAndRestarts(t *testing.T) {
	setupKvmCommConfigTestWithTimeout(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommIdleTimeoutEnabled(false, true, 0); err != nil {
		t.Fatalf("disable idle timeout: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected restart after disabling idle timeout, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.Device.IdleTimeoutMS != 0 {
		t.Fatalf("expected idle timeout 0, got %d", config.Device.IdleTimeoutMS)
	}
}

func TestSetKvmCommIdleTimeoutSameValueDoesNotRestart(t *testing.T) {
	setupKvmCommConfigTestWithTimeout(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommIdleTimeoutEnabled(true, true, 0); err != nil {
		t.Fatalf("enable unchanged idle timeout: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected no restart for unchanged idle timeout, got %d", calls)
	}
}

func TestSetKvmCommIdleTimeoutRestoresConfigWhenRestartFails(t *testing.T) {
	setupKvmCommConfigTestWithTimeout(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS)

	calls := 0
	restartErr := errors.New("restart failed")
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		if calls == 1 {
			return restartErr
		}
		return nil
	})
	defer restoreRestart()

	err := SetKvmCommIdleTimeoutEnabled(false, true, 0)
	if !errors.Is(err, ErrKvmCommRestartFailed) {
		t.Fatalf("expected restart failed error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected restart and restore restart calls, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config after restore: %v", err)
	}
	if config.Device.IdleTimeoutMS != kvmCommDefaultIdleTimeoutMS {
		t.Fatalf("expected restored idle timeout %d, got %d", kvmCommDefaultIdleTimeoutMS, config.Device.IdleTimeoutMS)
	}
}

func TestSetKvmCommTimelapseEnabledWritesEnableAndRestarts(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommTimelapseEnabled(true, true, 0); err != nil {
		t.Fatalf("enable timelapse: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected restart after enabling timelapse, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if !config.Timelapse.Enable {
		t.Fatalf("expected timelapse enabled")
	}
}

func TestSetKvmCommTimelapseDisabledWritesDisableAndRestarts(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, true)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommTimelapseEnabled(false, true, 0); err != nil {
		t.Fatalf("disable timelapse: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected restart after disabling timelapse, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.Timelapse.Enable {
		t.Fatalf("expected timelapse disabled")
	}
}

func TestSetKvmCommTimelapseSameValueDoesNotRestart(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommTimelapseEnabled(false, true, 0); err != nil {
		t.Fatalf("disable unchanged timelapse: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected no restart for unchanged timelapse, got %d", calls)
	}
}

func TestSetKvmCommTimelapseRestoresConfigWhenRestartFails(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false)

	calls := 0
	restartErr := errors.New("restart failed")
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		if calls == 1 {
			return restartErr
		}
		return nil
	})
	defer restoreRestart()

	err := SetKvmCommTimelapseEnabled(true, true, 0)
	if !errors.Is(err, ErrKvmCommRestartFailed) {
		t.Fatalf("expected restart failed error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected restart and restore restart calls, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config after restore: %v", err)
	}
	if config.Timelapse.Enable {
		t.Fatalf("expected restored timelapse disabled")
	}
}

func TestSetKvmCommTimelapseQPWhileDisabledDoesNotRestart(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false)

	calls := 0
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(context.Context, utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	changed, restarted, err := SetKvmCommTimelapseQP(40, 0)
	if err != nil || !changed || restarted {
		t.Fatalf("set disabled timelapse QP: changed=%t restarted=%t err=%v", changed, restarted, err)
	}
	if calls != 0 {
		t.Fatalf("expected no restart while timelapse is disabled, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.Timelapse.QP != 40 {
		t.Fatalf("timelapse QP=%d, want 40", config.Timelapse.QP)
	}
}

func TestSetKvmCommTimelapseQPWhileEnabledRestarts(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, true)

	calls := 0
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(context.Context, utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	changed, restarted, err := SetKvmCommTimelapseQP(40, 0)
	if err != nil || !changed || !restarted {
		t.Fatalf("set enabled timelapse QP: changed=%t restarted=%t err=%v", changed, restarted, err)
	}
	if calls != 1 {
		t.Fatalf("expected one restart while timelapse is enabled, got %d", calls)
	}
}

func TestSetKvmCommTimelapseQPSameValueDoesNotRestart(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, true)

	calls := 0
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(context.Context, utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	changed, restarted, err := SetKvmCommTimelapseQP(KvmCommTimelapseQPDefault, 0)
	if err != nil || changed || restarted {
		t.Fatalf("set unchanged timelapse QP: changed=%t restarted=%t err=%v", changed, restarted, err)
	}
	if calls != 0 {
		t.Fatalf("expected no restart for unchanged QP, got %d", calls)
	}
}

func TestSetKvmCommTimelapseQPRejectsOutOfRange(t *testing.T) {
	for _, qp := range []int{KvmCommTimelapseQPMin - 1, KvmCommTimelapseQPMax + 1} {
		changed, restarted, err := SetKvmCommTimelapseQP(qp, 0)
		if !errors.Is(err, ErrKvmCommInvalidTimelapseSetting) || changed || restarted {
			t.Fatalf("qp=%d changed=%t restarted=%t err=%v", qp, changed, restarted, err)
		}
	}
}

func TestSetKvmCommTimelapseQPRestoresConfigWhenRestartFails(t *testing.T) {
	setupKvmCommConfigTestWithTimelapse(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, true)

	calls := 0
	restartErr := errors.New("restart failed")
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(context.Context, utils.RestartWatchedProcessOptions) error {
		calls++
		if calls == 1 {
			return restartErr
		}
		return nil
	})
	defer restoreRestart()

	changed, restarted, err := SetKvmCommTimelapseQP(40, 0)
	if !errors.Is(err, ErrKvmCommRestartFailed) || changed || restarted {
		t.Fatalf("restart failure: changed=%t restarted=%t err=%v", changed, restarted, err)
	}
	if calls != 2 {
		t.Fatalf("expected failed restart plus restored restart, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get restored config: %v", err)
	}
	if config.Timelapse.QP != KvmCommTimelapseQPDefault {
		t.Fatalf("restored timelapse QP=%d, want %d", config.Timelapse.QP, KvmCommTimelapseQPDefault)
	}
}

func TestSetKvmCommAIRuntimeEnablesSnapshotOnlyAndRestarts(t *testing.T) {
	setupKvmCommConfigTestWithAIRuntime(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false, false, false)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommAIRuntime(true, false, false, true, 0); err != nil {
		t.Fatalf("set AI runtime: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one restart after changing AI runtime, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if !config.Snapshot.Enable || config.Snapshot.PeriodicEnable {
		t.Fatalf("unexpected snapshot config: %#v", config.Snapshot)
	}
	if config.Timelapse.Enable {
		t.Fatalf("expected timelapse disabled")
	}
}

func TestSetKvmCommAIRuntimeEnablesSnapshotPeriodicAndTimelapseWithSingleRestart(t *testing.T) {
	setupKvmCommConfigTestWithAIRuntime(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false, false, false)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommAIRuntime(true, true, true, true, 0); err != nil {
		t.Fatalf("set AI runtime: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected one restart after changing AI runtime fields, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if !config.Snapshot.Enable || !config.Snapshot.PeriodicEnable || !config.Timelapse.Enable {
		t.Fatalf("expected snapshot and timelapse enabled, got snapshot=%#v timelapse=%#v", config.Snapshot, config.Timelapse)
	}
}

func TestSetKvmCommAIRuntimeSameValueDoesNotRestart(t *testing.T) {
	setupKvmCommConfigTestWithAIRuntime(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, true, false, false)

	var calls int
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		return nil
	})
	defer restoreRestart()

	if err := SetKvmCommAIRuntime(true, false, false, true, 0); err != nil {
		t.Fatalf("set unchanged AI runtime: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected no restart for unchanged AI runtime, got %d", calls)
	}
}

func TestSetKvmCommAIRuntimeRestoresConfigWhenRestartFails(t *testing.T) {
	setupKvmCommConfigTestWithAIRuntime(t, KvmCommVideoModeH264, kvmCommDefaultIdleTimeoutMS, false, false, false)

	calls := 0
	restartErr := errors.New("restart failed")
	restoreRestart := replaceKvmCommRestartWatchedProcess(func(ctx context.Context, opts utils.RestartWatchedProcessOptions) error {
		calls++
		if calls == 1 {
			return restartErr
		}
		return nil
	})
	defer restoreRestart()

	err := SetKvmCommAIRuntime(true, true, true, true, 0)
	if !errors.Is(err, ErrKvmCommRestartFailed) {
		t.Fatalf("expected restart failed error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected restart and restore restart calls, got %d", calls)
	}

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config after restore: %v", err)
	}
	if config.Snapshot.Enable || config.Snapshot.PeriodicEnable || config.Timelapse.Enable {
		t.Fatalf("expected restored AI runtime disabled, got snapshot=%#v timelapse=%#v", config.Snapshot, config.Timelapse)
	}
}

func TestKvmCommConfigDefaultsTimelapseDisabled(t *testing.T) {
	configPath := setupKvmCommConfigTestData(t, `[device]
mode = "h264"
idle_poll_ms = 500

[log]
file = "/var/log/kvmcomm/vin.log"
level = "error"
type = "file"
`)

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.Timelapse.Enable {
		t.Fatalf("expected default timelapse disabled")
	}
	if config.Timelapse.QP != KvmCommTimelapseQPDefault {
		t.Fatalf("default timelapse QP=%d, want %d", config.Timelapse.QP, KvmCommTimelapseQPDefault)
	}
	if config.Snapshot.Enable || config.Snapshot.PeriodicEnable {
		t.Fatalf("unexpected default snapshot config: %#v", config.Snapshot)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config path to exist: %v", err)
	}
}

func TestKvmCommTimelapseQPNormalizationIsReadOnly(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		want  int
	}{
		{name: "minimum", value: "36", want: KvmCommTimelapseQPMin},
		{name: "maximum", value: "51", want: KvmCommTimelapseQPMax},
		{name: "below minimum", value: "35", want: KvmCommTimelapseQPDefault},
		{name: "above maximum", value: "52", want: KvmCommTimelapseQPDefault},
		{name: "wrong type", value: `"40"`, want: KvmCommTimelapseQPDefault},
	} {
		t.Run(test.name, func(t *testing.T) {
			configPath := setupKvmCommConfigTestData(t, "[timelapse]\nenable = true\nqp = "+test.value+"\n")
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config before get: %v", err)
			}

			config, err := GetKvmCommConfig()
			if err != nil {
				t.Fatalf("get config: %v", err)
			}
			if config.Timelapse.QP != test.want {
				t.Fatalf("QP=%d, want %d", config.Timelapse.QP, test.want)
			}

			after, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("read config after get: %v", err)
			}
			if !bytes.Equal(before, after) {
				t.Fatalf("GetKvmCommConfig changed persisted TOML")
			}
		})
	}
}

func TestKvmCommConfigDefaultsSnapshotPeriodicDisabled(t *testing.T) {
	setupKvmCommConfigTestData(t, `[device]
mode = "h264"
idle_poll_ms = 500

[snapshot]
enable = true
interval_ms = 10000

[timelapse]
enable = false

[log]
file = "/var/log/kvmcomm/vin.log"
level = "error"
type = "file"
`)

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if !config.Snapshot.Enable {
		t.Fatalf("expected snapshot enabled")
	}
	if config.Snapshot.PeriodicEnable {
		t.Fatalf("expected default snapshot periodic disabled")
	}
}

func TestKvmCommConfigDefaultsIdleTimeoutEnabled(t *testing.T) {
	configPath := setupKvmCommConfigTestData(t, `[device]
mode = "h264"
idle_poll_ms = 500

[log]
file = "/var/log/kvmcomm/vin.log"
level = "error"
type = "file"
`)

	config, err := GetKvmCommConfig()
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if config.Device.IdleTimeoutMS != kvmCommDefaultIdleTimeoutMS {
		t.Fatalf("expected default idle timeout %d, got %d", kvmCommDefaultIdleTimeoutMS, config.Device.IdleTimeoutMS)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("expected config path to exist: %v", err)
	}
}

func TestSaveKvmCommConfigRemovesDeprecatedFields(t *testing.T) {
	configPath := setupKvmCommConfigTestData(t, `[device]
type = "agent"
mode = "h264"
idle_poll_ms = 500
idle_power_off_lt7911 = false
idle_timeout_ms = 3000

[snapshot]
enable = false
interval_ms = 10000
periodic_enable = false

[timelapse]
enable = false
bitrate = 12000

[log]
file = "/var/log/kvmcomm/vin.log"
level = "error"
type = "file"
`)

	if err := SetKvmCommVideoMode(KvmCommVideoModeH264, false, 0); err != nil {
		t.Fatalf("save unchanged config: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if strings.Contains(string(data), `type = "agent"`) {
		t.Fatalf("expected deprecated device.type to be removed, got:\n%s", string(data))
	}

	var saved map[string]any
	if err := toml.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode saved config: %v", err)
	}
	snapshot, ok := saved[kvmCommSnapshotSection].(map[string]any)
	if !ok {
		t.Fatalf("expected snapshot table, got %#v", saved[kvmCommSnapshotSection])
	}
	if _, exists := snapshot[kvmCommRetiredSnapshotIntervalMS]; exists {
		t.Fatalf("expected deprecated snapshot.interval_ms to be removed, got:\n%s", string(data))
	}
	timelapse, ok := saved[kvmCommTimelapseSection].(map[string]any)
	if !ok {
		t.Fatalf("expected timelapse table, got %#v", saved[kvmCommTimelapseSection])
	}
	if _, exists := timelapse["bitrate"]; exists {
		t.Fatalf("expected deprecated timelapse.bitrate to be removed, got:\n%s", string(data))
	}
	if qp, ok := numericTomlValue[int](timelapse[kvmCommTimelapseQP]); !ok || qp != KvmCommTimelapseQPDefault {
		t.Fatalf("timelapse QP=%d valid=%t, want default %d", qp, ok, KvmCommTimelapseQPDefault)
	}
}

func setupKvmCommConfigTest(t *testing.T, mode string) string {
	return setupKvmCommConfigTestWithTimeout(t, mode, kvmCommDefaultIdleTimeoutMS)
}

func setupKvmCommConfigTestWithTimeout(t *testing.T, mode string, idleTimeoutMS int) string {
	return setupKvmCommConfigTestWithTimelapse(t, mode, idleTimeoutMS, false)
}

func setupKvmCommConfigTestWithTimelapse(t *testing.T, mode string, idleTimeoutMS int, timelapseEnabled bool) string {
	return setupKvmCommConfigTestWithAIRuntime(t, mode, idleTimeoutMS, false, false, timelapseEnabled)
}

func setupKvmCommConfigTestWithAIRuntime(t *testing.T, mode string, idleTimeoutMS int, snapshotEnabled bool, snapshotPeriodicEnabled bool, timelapseEnabled bool) string {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "kvm_vin.toml")
	writeKvmCommConfigForTest(t, configPath, mode, idleTimeoutMS, snapshotEnabled, snapshotPeriodicEnabled, timelapseEnabled)

	setKvmCommConfigPathForTest(t, configPath)
	return configPath
}

func setupKvmCommConfigTestData(t *testing.T, data string) string {
	t.Helper()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "kvm_vin.toml")
	if err := os.WriteFile(configPath, []byte(data), 0o644); err != nil {
		t.Fatalf("write kvmcomm config: %v", err)
	}

	setKvmCommConfigPathForTest(t, configPath)
	return configPath
}

func setKvmCommConfigPathForTest(t *testing.T, configPath string) {
	t.Helper()
	oldPath := kvmCommConfigFilePath
	kvmCommConfigFilePath = configPath
	t.Cleanup(func() {
		kvmCommConfigFilePath = oldPath
	})
}

func replaceKvmCommRestartWatchedProcess(fn func(context.Context, utils.RestartWatchedProcessOptions) error) func() {
	old := restartWatchedProcess
	restartWatchedProcess = fn
	return func() {
		restartWatchedProcess = old
	}
}

func writeKvmCommConfigForTest(t *testing.T, path string, mode string, idleTimeoutMS int, snapshotEnabled bool, snapshotPeriodicEnabled bool, timelapseEnabled bool) {
	t.Helper()

	data := `[device]
mode = "` + mode + `"
idle_poll_ms = 500
idle_power_off_lt7911 = false
idle_timeout_ms = ` + strconv.Itoa(idleTimeoutMS) + `

[log]
file = "/var/log/kvmcomm/vin.log"
level = "error"
type = "file"

[snapshot]
enable = ` + strconv.FormatBool(snapshotEnabled) + `
periodic_enable = ` + strconv.FormatBool(snapshotPeriodicEnabled) + `

[timelapse]
enable = ` + strconv.FormatBool(timelapseEnabled) + `
qp = ` + strconv.Itoa(KvmCommTimelapseQPDefault) + `
`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write kvmcomm config: %v", err)
	}
}
