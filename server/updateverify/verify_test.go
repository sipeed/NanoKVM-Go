package updateverify

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseKeyRingRejectsFingerprintMismatch(t *testing.T) {
	publicKey, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(KeyRingFile{Schema: 1, Keys: []Key{{
		KeyID:     "sha256:wrong",
		Algorithm: "ed25519",
		PublicKey: base64.StdEncoding.EncodeToString(publicKey),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseKeyRing(data)
	assertKind(t, err, ErrorInvalid)
}

func TestVerifyTransportArtifactChecksNameSizeAndDigest(t *testing.T) {
	dir := t.TempDir()
	content := []byte("package")
	path := filepath.Join(dir, "update.tar.gz")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha512.Sum512(content)
	metadata := TransportMetadata{
		Name:   filepath.Base(path),
		SHA512: base64.StdEncoding.EncodeToString(digest[:]),
		Size:   int64(len(content)),
	}
	if err := VerifyTransportArtifact(path, metadata); err != nil {
		t.Fatalf("verify valid artifact: %v", err)
	}

	badSize := metadata
	badSize.Size++
	assertKind(t, VerifyTransportArtifact(path, badSize), ErrorPolicy)

	badDigest := metadata
	badDigest.SHA512 = base64.StdEncoding.EncodeToString(make([]byte, sha512.Size))
	assertKind(t, VerifyTransportArtifact(path, badDigest), ErrorPolicy)

	badName := metadata
	badName.Name = "other.tar.gz"
	assertKind(t, VerifyTransportArtifact(path, badName), ErrorPolicy)

	symlinkPath := filepath.Join(dir, "linked.tar.gz")
	if err := os.Symlink(path, symlinkPath); err != nil {
		t.Fatal(err)
	}
	symlinkMetadata := metadata
	symlinkMetadata.Name = filepath.Base(symlinkPath)
	assertKind(t, VerifyTransportArtifact(symlinkPath, symlinkMetadata), ErrorPolicy)
}

func assertKind(t *testing.T, err error, want ErrorKind) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want kind %q", want)
	}
	if got := Kind(err); got != want {
		t.Fatalf("error kind = %q, want %q: %v", got, want, err)
	}
}
