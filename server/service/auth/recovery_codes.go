package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	defaultRecoveryCodesFilePath = "/etc/kvm/recovery_codes.json"
	recoveryCodePrefix           = "nkv"
	recoveryCodeRawLength        = 12
	recoveryCodeGroupLength      = 4
	recoveryCodeVersion          = 1
	defaultRecoveryCodeCount     = 8
)

const recoveryCodeCharset = "2346789ABCDEFGHJKMNPQRTUVWXYZ"

var (
	recoveryCodesFilePath   = defaultRecoveryCodesFilePath
	recoveryCodeBcryptCost  = bcrypt.DefaultCost
	recoveryCodesNow        = time.Now
	recoveryCodesMu         sync.Mutex
	errInvalidRecoveryCount = errors.New("recovery code count must be positive")
)

type recoveryCodesFile struct {
	Enabled bool                `json:"enabled"`
	Version int                 `json:"version"`
	Codes   []recoveryCodeEntry `json:"codes"`
}

type recoveryCodeEntry struct {
	ID        string  `json:"id"`
	Hash      string  `json:"hash"`
	UsedAt    *string `json:"usedAt"`
	CreatedAt string  `json:"createdAt"`
}

func generateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 {
		return nil, errInvalidRecoveryCount
	}

	codes := make([]string, count)
	for i := range codes {
		raw := make([]byte, recoveryCodeRawLength)
		for j := range raw {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(recoveryCodeCharset))))
			if err != nil {
				return nil, err
			}
			raw[j] = recoveryCodeCharset[idx.Int64()]
		}
		codes[i] = formatRecoveryCode(string(raw))
	}

	return codes, nil
}

func normalizeRecoveryCode(input string) (string, bool) {
	value := strings.ToUpper(strings.TrimSpace(input))
	if !strings.HasPrefix(value, strings.ToUpper(recoveryCodePrefix)) {
		return "", false
	}

	body := strings.TrimPrefix(value, strings.ToUpper(recoveryCodePrefix))
	var raw string
	if strings.HasPrefix(body, "-") {
		parts := strings.Split(strings.TrimPrefix(body, "-"), "-")
		if len(parts) != 3 {
			return "", false
		}
		for _, part := range parts {
			if len(part) != recoveryCodeGroupLength {
				return "", false
			}
			raw += part
		}
	} else {
		if strings.Contains(body, "-") {
			return "", false
		}
		raw = body
	}

	if len(raw) != recoveryCodeRawLength {
		return "", false
	}
	for _, ch := range raw {
		if !strings.ContainsRune(recoveryCodeCharset, ch) {
			return "", false
		}
	}

	return formatRecoveryCode(raw), true
}

func formatRecoveryCode(raw string) string {
	return fmt.Sprintf("%s-%s-%s-%s", recoveryCodePrefix, raw[0:4], raw[4:8], raw[8:12])
}

func loadRecoveryCodes() (recoveryCodesFile, error) {
	recoveryCodesMu.Lock()
	defer recoveryCodesMu.Unlock()

	return loadRecoveryCodesLocked()
}

func loadRecoveryCodesLocked() (recoveryCodesFile, error) {
	data, err := os.ReadFile(recoveryCodesFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return disabledRecoveryCodesFile(), nil
		}
		return recoveryCodesFile{}, err
	}

	var file recoveryCodesFile
	if err := json.Unmarshal(data, &file); err != nil {
		return recoveryCodesFile{}, err
	}
	if file.Version == 0 {
		file.Version = recoveryCodeVersion
	}

	return file, nil
}

func saveRecoveryCodes(file recoveryCodesFile) error {
	recoveryCodesMu.Lock()
	defer recoveryCodesMu.Unlock()

	return saveRecoveryCodesLocked(file)
}

func saveRecoveryCodesLocked(file recoveryCodesFile) error {
	if file.Version == 0 {
		file.Version = recoveryCodeVersion
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	dir := filepath.Dir(recoveryCodesFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".recovery_codes.json.*")
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
	if _, err := tmp.Write(data); err != nil {
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
	if err := os.Rename(tmpPath, recoveryCodesFilePath); err != nil {
		return err
	}

	dirFile, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFile.Close()
	return dirFile.Sync()
}

func clearRecoveryCodes() error {
	recoveryCodesMu.Lock()
	defer recoveryCodesMu.Unlock()

	return saveRecoveryCodesLocked(disabledRecoveryCodesFile())
}

func recoveryCodesStatus() (enabled bool, remaining int, err error) {
	recoveryCodesMu.Lock()
	defer recoveryCodesMu.Unlock()

	file, err := loadRecoveryCodesLocked()
	if err != nil {
		return false, 0, err
	}
	if !file.Enabled {
		return false, 0, nil
	}

	for _, code := range file.Codes {
		if code.UsedAt == nil {
			remaining++
		}
	}

	return true, remaining, nil
}

func verifyAndConsumeRecoveryCode(code string) (used bool, codeID string, err error) {
	normalized, ok := normalizeRecoveryCode(code)
	if !ok {
		return false, "", nil
	}

	recoveryCodesMu.Lock()
	defer recoveryCodesMu.Unlock()

	file, err := loadRecoveryCodesLocked()
	if err != nil {
		return false, "", err
	}
	if !file.Enabled {
		return false, "", nil
	}

	for i := range file.Codes {
		entry := &file.Codes[i]
		if entry.UsedAt != nil {
			continue
		}
		if bcrypt.CompareHashAndPassword([]byte(entry.Hash), []byte(normalized)) != nil {
			continue
		}

		usedAt := recoveryCodesNow().UTC().Format(time.RFC3339)
		entry.UsedAt = &usedAt
		if err := saveRecoveryCodesLocked(file); err != nil {
			return false, "", err
		}

		return true, entry.ID, nil
	}

	return false, "", nil
}

func disabledRecoveryCodesFile() recoveryCodesFile {
	return recoveryCodesFile{
		Enabled: false,
		Version: recoveryCodeVersion,
		Codes:   []recoveryCodeEntry{},
	}
}

func buildRecoveryCodesFile(codes []string) (recoveryCodesFile, error) {
	now := recoveryCodesNow().UTC().Format(time.RFC3339)
	entries := make([]recoveryCodeEntry, 0, len(codes))
	for _, code := range codes {
		normalized, ok := normalizeRecoveryCode(code)
		if !ok {
			return recoveryCodesFile{}, errors.New("invalid recovery code")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(normalized), recoveryCodeBcryptCost)
		if err != nil {
			return recoveryCodesFile{}, err
		}
		id, err := randomRecoveryCodeID()
		if err != nil {
			return recoveryCodesFile{}, err
		}
		entries = append(entries, recoveryCodeEntry{
			ID:        id,
			Hash:      string(hash),
			CreatedAt: now,
		})
	}

	return recoveryCodesFile{
		Enabled: true,
		Version: recoveryCodeVersion,
		Codes:   entries,
	}, nil
}

func generateAndSaveRecoveryCodes(count int) ([]string, error) {
	codes, err := generateRecoveryCodes(count)
	if err != nil {
		return nil, err
	}
	file, err := buildRecoveryCodesFile(codes)
	if err != nil {
		return nil, err
	}
	if err := saveRecoveryCodes(file); err != nil {
		return nil, err
	}

	return codes, nil
}

func randomRecoveryCodeID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf[:]), nil
}
