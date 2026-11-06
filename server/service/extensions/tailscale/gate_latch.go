package tailscale

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	gateLatchVersion  = 1
	gateLatchMaxBytes = 4 << 10
	// defaultGateLatchPath stores the peer-pairing latch next to the managed
	// identity state. It is a separate file on purpose: the managed identity
	// schema is frozen by its strict decoder (an unknown field would make an
	// older firmware reject the whole identity), so the gate cannot extend it.
	defaultGateLatchPath = "/etc/kvm/remote-network/tailscale-gate.json"
)

// gateLatch records that the managed network identified by NetworkID has at
// least one paired peer device. The LCD gate derives its match flag from it:
// the managed network reaching enabledConnected only means the KVM itself is
// connected, while Ready additionally requires a completed peer pairing.
type gateLatch struct {
	Version    int    `json:"version"`
	NetworkID  string `json:"networkId"`
	PeerPaired bool   `json:"peerPaired"`
	UpdatedAt  string `json:"updatedAt"`
}

// valid reports whether the latch is well-formed. A paired latch must name a
// network; a non-paired latch carries no information and is treated as
// invalid so it can never satisfy the gate.
func (l gateLatch) valid() bool {
	if l.Version != gateLatchVersion || !l.PeerPaired {
		return false
	}
	if !validSipeedStateString(l.NetworkID, 256, false) {
		return false
	}
	if l.UpdatedAt != "" {
		if _, err := time.Parse(time.RFC3339, l.UpdatedAt); err != nil {
			return false
		}
	}
	return true
}

// gateLatchStore persists the peer-pairing latch with the same hardening as
// the managed identity store: a 0700 directory, a regular 0600 file, no
// symlinks, and an atomic same-directory rename with fsync.
type gateLatchStore struct {
	path string
}

func newGateLatchStore(path string) *gateLatchStore {
	if strings.TrimSpace(path) == "" {
		path = defaultGateLatchPath
	}
	return &gateLatchStore{path: filepath.Clean(path)}
}

// paired reports whether a valid latch proves a peer pairing for networkID.
// Every read or validation problem fails closed: a false result only delays
// the Ready stage until a live devices probe can re-establish the fact.
func (store *gateLatchStore) paired(networkID string) bool {
	latch, err := store.load()
	return err == nil && latch.valid() && latch.NetworkID == networkID
}

// mark records a completed peer pairing for networkID.
func (store *gateLatchStore) mark(networkID string, now time.Time) error {
	if !validSipeedStateString(networkID, 256, false) {
		return errors.New("gate latch network id is invalid")
	}
	return store.save(gateLatch{
		Version:    gateLatchVersion,
		NetworkID:  networkID,
		PeerPaired: true,
		UpdatedAt:  now.UTC().Format(time.RFC3339),
	})
}

// clear removes the latch file. A missing file is not an error.
func (store *gateLatchStore) clear() error {
	if store == nil || store.path == "" {
		return nil
	}
	if err := rejectSipeedStateSymlink(filepath.Dir(store.path), true); err != nil {
		return err
	}
	if err := rejectSipeedStateSymlink(store.path, true); err != nil {
		return err
	}
	if err := os.Remove(store.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove gate latch: %w", err)
	}
	return nil
}

func (store *gateLatchStore) load() (gateLatch, error) {
	if store == nil || store.path == "" {
		return gateLatch{}, errors.New("gate latch path is empty")
	}
	directory := filepath.Dir(store.path)
	if err := rejectSipeedStateSymlink(directory, true); err != nil {
		return gateLatch{}, err
	}
	if directoryInfo, err := os.Lstat(directory); err == nil {
		if !directoryInfo.IsDir() || directoryInfo.Mode().Perm() != 0o700 {
			return gateLatch{}, errors.New("gate latch directory permissions are invalid")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return gateLatch{}, fmt.Errorf("inspect gate latch directory: %w", err)
	}
	if err := rejectSipeedStateSymlink(store.path, true); err != nil {
		return gateLatch{}, err
	}

	info, err := os.Lstat(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return gateLatch{}, os.ErrNotExist
	}
	if err != nil {
		return gateLatch{}, fmt.Errorf("inspect gate latch: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 ||
		info.Size() > gateLatchMaxBytes {
		return gateLatch{}, errors.New("gate latch file attributes are invalid")
	}

	file, err := os.OpenFile(store.path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return gateLatch{}, fmt.Errorf("open gate latch: %w", err)
	}
	defer func() { _ = file.Close() }()

	payload, err := io.ReadAll(io.LimitReader(file, gateLatchMaxBytes+1))
	if err != nil || len(payload) > gateLatchMaxBytes {
		return gateLatch{}, errors.New("read gate latch failed")
	}
	var latch gateLatch
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&latch); err != nil {
		return gateLatch{}, errors.New("gate latch contains invalid JSON")
	}
	if err := ensureSipeedJSONEOF(decoder); err != nil {
		return gateLatch{}, err
	}
	if !latch.valid() {
		return gateLatch{}, errors.New("gate latch content is invalid")
	}
	return latch, nil
}

func (store *gateLatchStore) save(latch gateLatch) error {
	if store == nil || store.path == "" {
		return errors.New("gate latch path is empty")
	}
	directory := filepath.Dir(store.path)
	if err := ensurePrivateSipeedStateDirectory(directory); err != nil {
		return err
	}
	if err := rejectSipeedStateSymlink(store.path, true); err != nil {
		return err
	}

	payload, err := json.Marshal(latch)
	if err != nil {
		return errors.New("encode gate latch failed")
	}
	payload = append(payload, '\n')

	randomSuffix := make([]byte, 8)
	if _, err := rand.Read(randomSuffix); err != nil {
		return errors.New("generate gate latch temporary name failed")
	}
	temporaryPath := filepath.Join(directory, ".tailscale-gate-"+hex.EncodeToString(randomSuffix)+".tmp")
	file, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create gate latch temporary file: %w", err)
	}
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("secure gate latch temporary file: %w", err)
	}
	if _, err := file.Write(payload); err != nil {
		return fmt.Errorf("write gate latch: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync gate latch: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close gate latch: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return fmt.Errorf("publish gate latch: %w", err)
	}
	cleanup = false

	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open gate latch directory: %w", err)
	}
	defer func() { _ = directoryFile.Close() }()
	if err := directoryFile.Sync(); err != nil {
		return fmt.Errorf("sync gate latch directory: %w", err)
	}
	return nil
}
