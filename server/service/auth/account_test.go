package auth

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func useTempAccountFile(t *testing.T) string {
	t.Helper()

	originalPath := accountFilePath
	path := filepath.Join(t.TempDir(), "pwd")
	accountFilePath = path
	t.Cleanup(func() {
		accountFilePath = originalPath
	})

	return path
}

func TestAccountDefaultAccountWhenFileMissing(t *testing.T) {
	useTempAccountFile(t)

	account, err := GetAccount()
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if account == nil {
		t.Fatal("GetAccount returned nil account")
	}
	if account.Username != "admin" {
		t.Fatalf("expected default username admin, got %q", account.Username)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(account.Password), []byte("admin")); err != nil {
		t.Fatalf("default password hash does not match admin: %v", err)
	}
}

func TestCompareAccountAfterSetAccount(t *testing.T) {
	useTempAccountFile(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}

	if err := SetAccount("admin", string(hash)); err != nil {
		t.Fatalf("SetAccount returned error: %v", err)
	}

	if !CompareAccount("admin", "secret") {
		t.Fatal("CompareAccount rejected the correct password")
	}
	if CompareAccount("admin", "wrong") {
		t.Fatal("CompareAccount accepted the wrong password")
	}
}

func TestAccountFileMode(t *testing.T) {
	path := useTempAccountFile(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}

	if err := SetAccount("admin", string(hash)); err != nil {
		t.Fatalf("SetAccount returned error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat returned error: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected account file mode 0600, got %04o", mode)
	}
}

func TestGetAccountRepairsOverlyPermissiveMode(t *testing.T) {
	path := useTempAccountFile(t)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	content := []byte(`{"username":"admin","password":"` + string(hash) + `"}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write legacy account: %v", err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod legacy account: %v", err)
	}

	account, err := GetAccount()
	if err != nil {
		t.Fatalf("GetAccount returned error: %v", err)
	}
	if account.Username != "admin" {
		t.Fatalf("unexpected account: %#v", account)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat migrated account: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected migrated account mode 0600, got %04o", mode)
	}
}

func TestSetAccountReplacesExistingFileAtomicallyAt0600(t *testing.T) {
	path := useTempAccountFile(t)
	if err := os.WriteFile(path, []byte("legacy"), 0o666); err != nil {
		t.Fatalf("write legacy account: %v", err)
	}
	if err := os.Chmod(path, 0o666); err != nil {
		t.Fatalf("chmod legacy account: %v", err)
	}

	if err := SetAccount("admin", "hash"); err != nil {
		t.Fatalf("SetAccount returned error: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat account: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Fatalf("expected rewritten account mode 0600, got %04o", mode)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".pwd.*"))
	if err != nil {
		t.Fatalf("glob temporary files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("atomic account write left temporary files: %#v", matches)
	}
}
