package netbird

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadRequiresPinnedChecksum(t *testing.T) {
	data := []byte("release package")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "package.tar.gz")
	if downloadPackage(context.Background(), server.URL, path, "bad") == nil {
		t.Fatal("accepted bad checksum")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid download was kept")
	}
	hash := sha256.Sum256(data)
	if err := downloadPackage(context.Background(), server.URL, path, hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
}

func TestExtractOnlyRegularExpectedFilesAndKeepLicenses(t *testing.T) {
	for _, kind := range []byte{tar.TypeReg, tar.TypeSymlink, tar.TypeLink} {
		t.Run(string(kind), func(t *testing.T) {
			var archive bytes.Buffer
			tw := tar.NewWriter(&archive)
			for _, name := range append([]string{"../escape"}, packageFiles...) {
				typ := byte(tar.TypeReg)
				size := int64(1)
				if name == "netbird" {
					typ = kind
					if kind != tar.TypeReg {
						size = 0
					}
				}
				if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: typ, Mode: 0755, Size: size, Linkname: "/etc/passwd"}); err != nil {
					t.Fatal(err)
				}
				if size > 0 {
					_, _ = tw.Write([]byte("x"))
				}
			}
			_ = tw.Close()
			dir := t.TempDir()
			err := extractPayload(&archive, dir)
			if kind != tar.TypeReg {
				if err == nil {
					t.Fatal("accepted link executable")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range packageFiles {
				if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
					t.Fatal(name, err)
				}
			}
			if _, err := os.Stat(filepath.Join(dir, "..", "escape")); !os.IsNotExist(err) {
				t.Fatal("path traversal")
			}
		})
	}
}
