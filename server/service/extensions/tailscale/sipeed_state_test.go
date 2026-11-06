package tailscale

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

func validTestSipeedState(t *testing.T) sipeedPersistedState {
	t.Helper()
	deviceID, err := newSipeedCredential(sipeedDeviceIDBytes)
	if err != nil {
		t.Fatal(err)
	}
	token, err := newSipeedCredential(sipeedDeviceTokenBytes)
	if err != nil {
		t.Fatal(err)
	}
	return sipeedPersistedState{
		Version:      sipeedStateVersion,
		ManagedState: proto.TailscaleManagedEnabledConnected,
		Phase:        sipeedPhaseComplete,
		DeviceID:     deviceID,
		DeviceToken:  token,
		NetworkID:    "net-test",
		NodeID:       "3",
		ProfileID:    "1ab3",
		Hostname:     "nano-kvm",
		IP:           "100.64.0.3",
	}
}

func TestSipeedStateStoreUsesPrivateAtomicFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	state := validTestSipeedState(t)
	if err := store.save(state); err != nil {
		t.Fatalf("save() error = %v", err)
	}

	directoryInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := directoryInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("directory mode = %o, want 700", got)
	}
	fileInfo, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fileInfo.Mode().IsRegular() || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("state file mode = %v, want regular 0600", fileInfo.Mode())
	}

	loaded, err := store.load()
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if loaded != state {
		t.Fatalf("loaded state = %+v, want %+v", loaded, state)
	}

	state.ManagedError = "refreshed"
	if err := store.save(state); err != nil {
		t.Fatalf("second save() error = %v", err)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(path) {
		t.Fatalf("atomic save left unexpected files: %v", entries)
	}
}

func TestSipeedStateStoreRejectsSymlinks(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		directory := t.TempDir()
		target := filepath.Join(directory, "target")
		if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(directory, "tailscale-sipeed.json")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		store := newSipeedStateStore(path)
		if _, err := store.load(); err == nil {
			t.Fatal("load accepted a state-file symlink")
		}
		if err := store.save(validTestSipeedState(t)); err == nil {
			t.Fatal("save accepted a state-file symlink")
		}
		body, err := os.ReadFile(target)
		if err != nil || string(body) != "secret" {
			t.Fatalf("symlink target was modified: %q, %v", body, err)
		}
	})

	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		target := filepath.Join(root, "target")
		if err := os.Mkdir(target, 0o700); err != nil {
			t.Fatal(err)
		}
		linked := filepath.Join(root, "remote-network")
		if err := os.Symlink(target, linked); err != nil {
			t.Fatal(err)
		}
		store := newSipeedStateStore(filepath.Join(linked, "tailscale-sipeed.json"))
		if _, err := store.load(); err == nil {
			t.Fatal("load accepted a state-directory symlink")
		}
		if err := store.save(validTestSipeedState(t)); err == nil {
			t.Fatal("save accepted a state-directory symlink")
		}
	})
}

func TestSipeedStateStoreRejectsInvalidContent(t *testing.T) {
	validID := base64.RawURLEncoding.EncodeToString(make([]byte, sipeedDeviceIDBytes))
	validToken := base64.RawURLEncoding.EncodeToString(make([]byte, sipeedDeviceTokenBytes))
	tests := map[string]string{
		"unknown field": `{"version":1,"managedState":"disabled","surprise":true}`,
		"trailing data": `{"version":1,"managedState":"disabled"} {}`,
		"wrong version": `{"version":2,"managedState":"disabled"}`,
		"bad state":     `{"version":1,"managedState":"connected"}`,
		"bad token":     `{"version":1,"managedState":"degraded","deviceToken":"short"}`,
		"disabled data": `{"version":1,"managedState":"disabled","deviceId":"` + validID + `","deviceToken":"` + validToken + `"}`,
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := newSipeedStateStore(path).load(); err == nil {
				t.Fatalf("load accepted invalid content: %s", body)
			}
		})
	}
}

func TestSipeedStateStoreRejectsLoosePermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remote-network", "tailscale-sipeed.json")
	store := newSipeedStateStore(path)
	if err := store.save(validTestSipeedState(t)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.load(); err == nil {
		t.Fatal("load accepted a world-readable identity file")
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := store.load(); err == nil {
		t.Fatal("load accepted a non-private identity directory")
	}
}

func TestSipeedCredentialLengthsAndEncoding(t *testing.T) {
	for _, size := range []int{sipeedDeviceIDBytes, sipeedDeviceTokenBytes} {
		value, err := newSipeedCredential(size)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(value, "=") || !validRawURLCredential(value, size) {
			t.Fatalf("credential %q is not canonical unpadded base64url", value)
		}
	}
	if validRawURLCredential("not-base64!", sipeedDeviceTokenBytes) {
		t.Fatal("invalid credential was accepted")
	}
}

func TestSipeedStateLoadMissingFileIsDisabled(t *testing.T) {
	state, err := newSipeedStateStore(filepath.Join(t.TempDir(), "missing", "state.json")).load()
	if err != nil {
		t.Fatal(err)
	}
	if state.ManagedState != proto.TailscaleManagedDisabled || state.Version != sipeedStateVersion {
		t.Fatalf("missing file state = %+v", state)
	}
}

func TestSipeedDisabledStateMayRetainOnlyOfficialProfileHint(t *testing.T) {
	state := newDisabledSipeedState()
	state.PreviousProfileID = "official-profile"
	store := newSipeedStateStore(filepath.Join(t.TempDir(), "remote-network", "state.json"))
	if err := store.save(state); err != nil {
		t.Fatalf("save disabled profile hint: %v", err)
	}
	loaded, err := store.load()
	if err != nil || loaded != state {
		t.Fatalf("loaded disabled profile hint = %+v, %v", loaded, err)
	}
}
