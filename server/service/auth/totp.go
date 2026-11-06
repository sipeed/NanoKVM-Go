package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultTOTPSecretFilePath   = "/etc/kvm/totp.secret"
	defaultTOTPHostnameFilePath = "/etc/hostname"
	defaultTOTPIssuer           = "NanoKVM"
	defaultTOTPWindow           = 1
	totpStepSeconds             = int64(30)
	totpDigits                  = 6
	totpSecretByteCount         = 20
	pendingTOTPSetupTTL         = 10 * time.Minute
)

var (
	totpSecretFilePath   = defaultTOTPSecretFilePath
	totpHostnameFilePath = defaultTOTPHostnameFilePath
	totpIssuer           = defaultTOTPIssuer
	totpWindow           = defaultTOTPWindow
	totpNow              = time.Now
)

type totpSettingsValue struct {
	secretFile string
	issuer     string
	window     int
}

type pendingTOTPSetup struct {
	secret    string
	uri       string
	createdAt time.Time
	expiresAt time.Time
}

type secondFactorResult struct {
	OK               bool
	RecoveryCodeUsed bool
	RecoveryCodeID   string
}

var pendingTOTPSetups = struct {
	sync.Mutex
	items map[string]pendingTOTPSetup
}{items: make(map[string]pendingTOTPSetup)}

func totpSettings() totpSettingsValue {
	if hasTOTPTestOverride() {
		return normalizedTOTPSettings(totpSettingsValue{
			secretFile: totpSecretFilePath,
			issuer:     totpIssuer,
			window:     totpWindow,
		})
	}

	settings := totpSettingsValue{
		secretFile: strings.TrimSpace(totpSecretFilePath),
		issuer:     strings.TrimSpace(totpIssuer),
		window:     totpWindow,
	}

	conf := getConfig().TwoFactorAuth
	if settings.secretFile == "" || settings.secretFile == "/etc/kvm/totp.secret" {
		settings.secretFile = strings.TrimSpace(conf.SecretFile)
	}
	if settings.issuer == "" || settings.issuer == "NanoKVM" {
		settings.issuer = strings.TrimSpace(conf.Issuer)
	}
	if settings.window == 1 {
		settings.window = conf.Window
	}

	return normalizedTOTPSettings(settings)
}

func hasTOTPTestOverride() bool {
	return strings.TrimSpace(totpSecretFilePath) != defaultTOTPSecretFilePath ||
		strings.TrimSpace(totpIssuer) != defaultTOTPIssuer ||
		totpWindow != defaultTOTPWindow
}

func normalizedTOTPSettings(settings totpSettingsValue) totpSettingsValue {
	settings.secretFile = strings.TrimSpace(settings.secretFile)
	settings.issuer = normalizedTOTPIssuer(settings.issuer)
	if settings.secretFile == "" {
		settings.secretFile = defaultTOTPSecretFilePath
	}
	if settings.window < 0 || settings.window > 2 {
		settings.window = defaultTOTPWindow
	}

	return settings
}

func normalizedTOTPIssuer(issuer string) string {
	issuer = strings.TrimSpace(issuer)
	if issuer != "" && issuer != defaultTOTPIssuer {
		return issuer
	}
	if hostname := totpHostname(); hostname != "" {
		return hostname
	}
	if issuer != "" {
		return issuer
	}
	return defaultTOTPIssuer
}

func totpHostname() string {
	if data, err := os.ReadFile(totpHostnameFilePath); err == nil {
		if hostname := normalizeHostname(string(data)); hostname != "" {
			return hostname
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return ""
	}
	return normalizeHostname(hostname)
}

func normalizeHostname(value string) string {
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", "")
	return strings.TrimSpace(value)
}

func activeTOTPSecretPath() string {
	return totpSettings().secretFile
}

func isTOTPEnabled() (bool, error) {
	secret, err := readActiveTOTPSecret()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	return secret != "", nil
}

func readActiveTOTPSecret() (string, error) {
	data, err := os.ReadFile(activeTOTPSecretPath())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func writeActiveTOTPSecret(secret string) error {
	path := activeTOTPSecretPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".totp.secret.*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.WriteString(secret); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func clearActiveTOTPSecret() error {
	return writeActiveTOTPSecret("")
}

func generateTOTPSecret() (string, error) {
	buf := make([]byte, totpSecretByteCount)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "="), nil
}

func buildTOTPURI(issuer, account, secret string) string {
	label := url.QueryEscape(issuer + ":" + account)
	return "otpauth://totp/" + label + "?secret=" + url.QueryEscape(secret) + "&issuer=" + url.QueryEscape(issuer)
}

func generateTOTPCode(secret string, at time.Time) (string, error) {
	decoded, err := decodeTOTPSecret(secret)
	if err != nil {
		return "", err
	}

	unixSeconds := at.Unix()
	if unixSeconds < 0 {
		return "", errors.New("totp time is before the Unix epoch")
	}
	// #nosec G115 -- the non-negative check above makes this conversion exact.
	counter := uint64(unixSeconds / totpStepSeconds)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	mac := hmac.New(sha1.New, decoded)
	if _, err := mac.Write(buf[:]); err != nil {
		return "", err
	}
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binaryCode := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)

	code := binaryCode % 1000000
	return fmt.Sprintf("%0*d", totpDigits, code), nil
}

func verifyTOTPCode(secret string, code string, now time.Time, window int) bool {
	if !isSixDigitCode(code) {
		return false
	}
	if window < 0 {
		window = 0
	}

	for offset := -window; offset <= window; offset++ {
		at := now.Add(time.Duration(offset) * time.Duration(totpStepSeconds) * time.Second)
		expected, err := generateTOTPCode(secret, at)
		if err != nil {
			return false
		}
		if subtleStringEqual(expected, code) {
			return true
		}
	}

	return false
}

func verifyCurrentTOTP(code string) bool {
	secret, err := readActiveTOTPSecret()
	if err != nil || secret == "" {
		return false
	}

	settings := totpSettings()
	return verifyTOTPCode(secret, code, totpNow(), settings.window)
}

func verifySecondFactor(code string) secondFactorResult {
	if isSixDigitCode(code) && verifyCurrentTOTP(code) {
		return secondFactorResult{OK: true}
	}

	used, codeID, err := verifyAndConsumeRecoveryCode(code)
	if err != nil || !used {
		return secondFactorResult{}
	}

	return secondFactorResult{
		OK:               true,
		RecoveryCodeUsed: true,
		RecoveryCodeID:   codeID,
	}
}

func createPendingTOTPSetup(username string) (string, string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", "", errors.New("username is required")
	}

	secret, err := generateTOTPSecret()
	if err != nil {
		return "", "", err
	}
	settings := totpSettings()
	uri := buildTOTPURI(settings.issuer, username, secret)
	now := totpNow()

	pendingTOTPSetups.Lock()
	defer pendingTOTPSetups.Unlock()
	cleanupExpiredPendingTOTPSetupsLocked(now)
	pendingTOTPSetups.items[username] = pendingTOTPSetup{
		secret:    secret,
		uri:       uri,
		createdAt: now,
		expiresAt: now.Add(pendingTOTPSetupTTL),
	}

	return secret, uri, nil
}

func cancelPendingTOTPSetup(username string) {
	pendingTOTPSetups.Lock()
	defer pendingTOTPSetups.Unlock()
	delete(pendingTOTPSetups.items, strings.TrimSpace(username))
}

func enablePendingTOTPSetup(username string, code string) error {
	return enablePendingTOTPSetupWithPreCommit(username, code, nil)
}

// enablePendingTOTPSetupWithPreCommit verifies the pending second factor before
// running a security-sensitive commit hook, then persists the active secret.
// This lets callers fail closed on session-key rotation without activating 2FA
// while old, non-2FA sessions remain valid.
func enablePendingTOTPSetupWithPreCommit(username string, code string, preCommit func() error) error {
	username = strings.TrimSpace(username)
	now := totpNow()

	pendingTOTPSetups.Lock()
	pending, ok := pendingTOTPSetups.items[username]
	if ok && now.After(pending.expiresAt) {
		delete(pendingTOTPSetups.items, username)
		ok = false
	}
	pendingTOTPSetups.Unlock()

	if !ok {
		return errors.New("pending totp setup not found")
	}
	if !verifyTOTPCode(pending.secret, code, now, totpSettings().window) {
		return errors.New("invalid totp code")
	}
	if preCommit != nil {
		if err := preCommit(); err != nil {
			return err
		}
	}
	if err := writeActiveTOTPSecret(pending.secret); err != nil {
		return err
	}

	cancelPendingTOTPSetup(username)
	return nil
}

func cleanupExpiredPendingTOTPSetups(now time.Time) {
	pendingTOTPSetups.Lock()
	defer pendingTOTPSetups.Unlock()
	cleanupExpiredPendingTOTPSetupsLocked(now)
}

func cleanupExpiredPendingTOTPSetupsLocked(now time.Time) {
	for username, pending := range pendingTOTPSetups.items {
		if !now.Before(pending.expiresAt) {
			delete(pendingTOTPSetups.items, username)
		}
	}
}

func decodeTOTPSecret(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	if normalized == "" {
		return nil, errors.New("totp secret is empty")
	}
	if padding := len(normalized) % 8; padding != 0 {
		normalized += strings.Repeat("=", 8-padding)
	}

	return base32.StdEncoding.DecodeString(normalized)
}

func isSixDigitCode(code string) bool {
	if len(code) != totpDigits {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func subtleStringEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
