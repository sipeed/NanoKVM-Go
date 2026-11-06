package zerotier

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadVerifiesChecksumAndRemovesFailedPayload(t *testing.T) {
	data := []byte("verified-package")
	hash := sha256.Sum256(data)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "package.deb")
	if err := downloadPackage(context.Background(), server.URL, path, "wrong"); err == nil {
		t.Fatal("bad checksum accepted")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed download left a partial package")
	}
	if err := downloadPackage(context.Background(), server.URL, path, hex.EncodeToString(hash[:])); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, data) {
		t.Fatal("verified download missing")
	}
}

func TestExtractOnlyWhitelistedRegularPayloadFiles(t *testing.T) {
	for _, kind := range []byte{tar.TypeReg, tar.TypeSymlink, tar.TypeLink} {
		t.Run(string(kind), func(t *testing.T) {
			var data bytes.Buffer
			w := tar.NewWriter(&data)
			for _, entry := range []struct {
				name string
				kind byte
			}{{"../../escaped", tar.TypeReg}, {"./usr/sbin/zerotier-one", kind}, {"./usr/share/doc/zerotier-one/copyright", tar.TypeReg}} {
				h := &tar.Header{Name: entry.name, Typeflag: entry.kind, Mode: 0755, Linkname: "/etc/passwd"}
				if entry.kind == tar.TypeReg {
					h.Size = 7
					h.Linkname = ""
				}
				if err := w.WriteHeader(h); err != nil {
					t.Fatal(err)
				}
				if h.Size > 0 {
					_, _ = w.Write([]byte("payload"))
				}
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			err := extractPayload(&data, dir)
			if kind != tar.TypeReg {
				if err == nil {
					t.Fatal("link payload accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil || len(entries) != 2 {
				t.Fatal("unexpected extracted files", err)
			}
			info, err := os.Stat(filepath.Join(dir, "zerotier-one"))
			if err != nil || info.Mode().Perm() != 0755 {
				t.Fatal("client is not executable")
			}
		})
	}
}
