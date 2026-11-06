package tailscale

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"NanoKVM-Server/proto"
)

const (
	sipeedStateVersion     = 1
	sipeedStateMaxBytes    = 16 << 10
	defaultSipeedStatePath = "/etc/kvm/remote-network/tailscale-sipeed.json"
	sipeedDeviceIDBytes    = 16
	sipeedDeviceTokenBytes = 32
)

type sipeedPhase string

const (
	sipeedPhaseNone       sipeedPhase = ""
	sipeedPhaseIdentity   sipeedPhase = "identity"
	sipeedPhaseEnrolled   sipeedPhase = "enrolled"
	sipeedPhaseConnecting sipeedPhase = "connecting"
	sipeedPhaseReporting  sipeedPhase = "reporting"
	sipeedPhaseComplete   sipeedPhase = "complete"
)

type sipeedPersistedState struct {
	Version           int                         `json:"version"`
	ManagedState      proto.TailscaleManagedState `json:"managedState"`
	ManagedError      string                      `json:"managedError,omitempty"`
	Phase             sipeedPhase                 `json:"phase,omitempty"`
	DeviceID          string                      `json:"deviceId,omitempty"`
	DeviceToken       string                      `json:"deviceToken,omitempty"`
	NetworkID         string                      `json:"networkId,omitempty"`
	NodeID            string                      `json:"nodeId,omitempty"`
	ProfileID         string                      `json:"profileId,omitempty"`
	PreviousProfileID string                      `json:"previousProfileId,omitempty"`
	Hostname          string                      `json:"hostname,omitempty"`
	IP                string                      `json:"ip,omitempty"`
	RetryAfter        string                      `json:"retryAfter,omitempty"`
}

func newDisabledSipeedState() sipeedPersistedState {
	return sipeedPersistedState{
		Version:      sipeedStateVersion,
		ManagedState: proto.TailscaleManagedDisabled,
	}
}

type sipeedStateStore struct {
	path string
}

func newSipeedStateStore(path string) *sipeedStateStore {
	if strings.TrimSpace(path) == "" {
		path = defaultSipeedStatePath
	}
	return &sipeedStateStore{path: filepath.Clean(path)}
}

func (store *sipeedStateStore) load() (sipeedPersistedState, error) {
	state := newDisabledSipeedState()
	if store == nil || store.path == "" {
		return state, errors.New("managed state path is empty")
	}

	directory := filepath.Dir(store.path)
	if err := rejectSipeedStateSymlink(directory, true); err != nil {
		return state, err
	}
	if directoryInfo, err := os.Lstat(directory); err == nil {
		if !directoryInfo.IsDir() || directoryInfo.Mode().Perm() != 0o700 {
			return state, errors.New("managed state directory permissions are invalid")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return state, fmt.Errorf("inspect managed state directory: %w", err)
	}

	info, err := os.Lstat(store.path)
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("inspect managed state: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return state, errors.New("managed state is not a regular file")
	}
	if info.Mode().Perm() != 0o600 {
		return state, errors.New("managed state file permissions are invalid")
	}
	if info.Size() > sipeedStateMaxBytes {
		return state, errors.New("managed state is too large")
	}

	file, err := os.OpenFile(store.path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return state, fmt.Errorf("open managed state: %w", err)
	}
	defer func() { _ = file.Close() }()

	payload, err := io.ReadAll(io.LimitReader(file, sipeedStateMaxBytes+1))
	if err != nil {
		return state, fmt.Errorf("read managed state: %w", err)
	}
	if len(payload) > sipeedStateMaxBytes {
		return state, errors.New("managed state is too large")
	}

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&state); err != nil {
		return newDisabledSipeedState(), errors.New("managed state contains invalid JSON")
	}
	if err := ensureSipeedJSONEOF(decoder); err != nil {
		return newDisabledSipeedState(), err
	}
	if err := validateSipeedState(state); err != nil {
		return newDisabledSipeedState(), err
	}
	return state, nil
}

func ensureSipeedJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("managed state contains trailing data")
}

func (store *sipeedStateStore) save(state sipeedPersistedState) error {
	if store == nil || store.path == "" {
		return errors.New("managed state path is empty")
	}
	if err := validateSipeedState(state); err != nil {
		return err
	}

	directory := filepath.Dir(store.path)
	if err := ensurePrivateSipeedStateDirectory(directory); err != nil {
		return err
	}
	if err := rejectSipeedStateSymlink(store.path, true); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.New("encode managed state failed")
	}
	payload = append(payload, '\n')

	randomSuffix := make([]byte, 8)
	if _, err := rand.Read(randomSuffix); err != nil {
		return errors.New("generate managed state temporary name failed")
	}
	temporaryPath := filepath.Join(directory, ".tailscale-sipeed-"+hex.EncodeToString(randomSuffix)+".tmp")
	file, err := os.OpenFile(temporaryPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create managed state temporary file: %w", err)
	}
	cleanup := true
	defer func() {
		_ = file.Close()
		if cleanup {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := file.Chmod(0o600); err != nil {
		return fmt.Errorf("secure managed state temporary file: %w", err)
	}
	if _, err := file.Write(payload); err != nil {
		return fmt.Errorf("write managed state: %w", err)
	}
	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync managed state: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close managed state: %w", err)
	}
	if err := os.Rename(temporaryPath, store.path); err != nil {
		return fmt.Errorf("publish managed state: %w", err)
	}
	cleanup = false

	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open managed state directory: %w", err)
	}
	defer func() { _ = directoryFile.Close() }()
	if err := directoryFile.Sync(); err != nil {
		return fmt.Errorf("sync managed state directory: %w", err)
	}
	return nil
}

func ensurePrivateSipeedStateDirectory(directory string) error {
	if err := rejectSipeedStateSymlink(directory, true); err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create managed state directory: %w", err)
	}
	if err := rejectSipeedStateSymlink(directory, false); err != nil {
		return err
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return fmt.Errorf("secure managed state directory: %w", err)
	}
	return nil
}

// rejectSipeedStateSymlink accepts a missing path only when allowMissing is
// true. Existing paths must be directories (for directory checks) or regular
// files (for the state-file check performed by the caller), never symlinks.
func rejectSipeedStateSymlink(path string, allowMissing bool) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) && allowMissing {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect managed state path: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("managed state path is a symbolic link")
	}
	return nil
}

func validateSipeedState(state sipeedPersistedState) error {
	if state.Version != sipeedStateVersion {
		return errors.New("unsupported managed state version")
	}
	if !validSipeedManagedState(state.ManagedState) || !validSipeedPhase(state.Phase) {
		return errors.New("managed state contains an invalid state")
	}
	if !validSipeedStateString(state.ManagedError, 64, true) ||
		!validSipeedStateString(state.NetworkID, 256, true) ||
		!validSipeedStateString(state.NodeID, 128, true) ||
		!validSipeedStateString(state.ProfileID, 128, true) ||
		!validSipeedStateString(state.PreviousProfileID, 128, true) ||
		!validSipeedStateString(state.Hostname, 255, true) ||
		!validSipeedStateString(state.IP, 64, true) {
		return errors.New("managed state contains an invalid field")
	}
	if state.ManagedError != "" && !validSipeedErrorCode(state.ManagedError) {
		return errors.New("managed state contains an invalid error code")
	}
	if state.ProfileID != "" && !validTailscaleProfileID(state.ProfileID) {
		return errors.New("managed state contains an invalid profile id")
	}
	if state.PreviousProfileID != "" && !validTailscaleProfileID(state.PreviousProfileID) {
		return errors.New("managed state contains an invalid previous profile id")
	}
	if state.IP != "" {
		ip := net.ParseIP(state.IP)
		if ip == nil || ip.To4() == nil || ip.String() != state.IP {
			return errors.New("managed state contains an invalid IPv4 address")
		}
	}
	if state.DeviceID != "" && !validRawURLCredential(state.DeviceID, sipeedDeviceIDBytes) {
		return errors.New("managed state contains an invalid device id")
	}
	if state.DeviceToken != "" && !validRawURLCredential(state.DeviceToken, sipeedDeviceTokenBytes) {
		return errors.New("managed state contains an invalid device token")
	}
	if state.RetryAfter != "" {
		if _, err := time.Parse(time.RFC3339, state.RetryAfter); err != nil {
			return errors.New("managed state contains an invalid retry deadline")
		}
	}
	if state.ManagedState == proto.TailscaleManagedDisabled &&
		(state.DeviceID != "" || state.DeviceToken != "" || state.NetworkID != "" ||
			state.NodeID != "" || state.ProfileID != "" || state.Hostname != "" ||
			state.IP != "" || state.RetryAfter != "" ||
			state.Phase != sipeedPhaseNone) {
		return errors.New("disabled managed state contains identity data")
	}
	if state.ManagedState == proto.TailscaleManagedInvalidIdentity && state.DeviceToken != "" {
		return errors.New("invalid managed identity still contains a token")
	}
	if state.ManagedState == proto.TailscaleManagedEnabledConnected &&
		(state.Phase != sipeedPhaseComplete || state.DeviceID == "" || state.DeviceToken == "" ||
			state.NetworkID == "" || state.NodeID == "" || state.ProfileID == "" ||
			state.Hostname == "" || state.IP == "") {
		return errors.New("connected managed state is incomplete")
	}
	if state.ManagedState == proto.TailscaleManagedEnabledConnecting &&
		(state.DeviceID == "" || state.DeviceToken == "" || state.NetworkID == "" ||
			(state.Phase != sipeedPhaseConnecting && state.Phase != sipeedPhaseReporting)) {
		return errors.New("connecting managed state is incomplete")
	}
	return nil
}

func validSipeedErrorCode(value string) bool {
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}

func validSipeedManagedState(state proto.TailscaleManagedState) bool {
	switch state {
	case proto.TailscaleManagedDisabled,
		proto.TailscaleManagedEnrolling,
		proto.TailscaleManagedEnabledConnecting,
		proto.TailscaleManagedEnabledConnected,
		proto.TailscaleManagedDegraded,
		proto.TailscaleManagedInvalidIdentity:
		return true
	default:
		return false
	}
}

func validSipeedPhase(phase sipeedPhase) bool {
	switch phase {
	case sipeedPhaseNone, sipeedPhaseIdentity, sipeedPhaseEnrolled,
		sipeedPhaseConnecting, sipeedPhaseReporting, sipeedPhaseComplete:
		return true
	default:
		return false
	}
}

func validSipeedStateString(value string, max int, allowEmpty bool) bool {
	if value == "" {
		return allowEmpty
	}
	if len(value) > max || strings.TrimSpace(value) != value {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func validRawURLCredential(value string, size int) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == size && base64.RawURLEncoding.EncodeToString(decoded) == value
}

func newSipeedCredential(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
