package frp

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestDownloadRejectsIncorrectChecksum(t *testing.T) {
	data := []byte("not an executable")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(data) }))
	defer server.Close()
	digest := sha256.Sum256(data)
	for _, tc := range []struct {
		checksum string
		success  bool
	}{
		{hex.EncodeToString(digest[:]), true}, {releaseChecksums["arm_hf"], false},
	} {
		path := filepath.Join(t.TempDir(), "archive")
		err := downloadArchive(context.Background(), server.URL, path, tc.checksum)
		if (err == nil) != tc.success {
			t.Fatal("unexpected checksum result", err)
		}
	}
}

func TestDownloadRetriesInterruptedTransfer(t *testing.T) {
	data := []byte("verified complete archive")
	digest := sha256.Sum256(data)
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		if attempts.Add(1) == 1 {
			_, _ = w.Write(data[:3])
			return
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "archive")
	if err := downloadArchive(context.Background(), server.URL, path, hex.EncodeToString(digest[:])); err != nil {
		t.Fatal(err)
	}
	if attempts.Load() != 2 {
		t.Fatal("interrupted download was not retried")
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(data) {
		t.Fatal("incomplete archive accepted", err)
	}
}

func TestExtractionUsesOnlyExpectedRegularFiles(t *testing.T) {
	for _, kind := range []string{"valid", "symlink", "duplicate", "missing"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "archive.tgz")
			file, err := os.Create(archive)
			if err != nil {
				t.Fatal(err)
			}
			gz := gzip.NewWriter(file)
			tw := tar.NewWriter(gz)
			add := func(name string, flag byte) {
				header := &tar.Header{Name: name, Mode: 0755, Typeflag: flag}
				if flag == tar.TypeReg {
					header.Size = 4
				} else {
					header.Linkname = "/etc/shadow"
				}
				if err := tw.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
				if flag == tar.TypeReg {
					if _, err := tw.Write([]byte("data")); err != nil {
						t.Fatal(err)
					}
				}
			}
			add("../../escape", tar.TypeReg)
			add("package/frps", tar.TypeReg)
			if kind == "symlink" {
				add("package/frpc", tar.TypeSymlink)
			} else {
				add("package/frpc", tar.TypeReg)
			}
			if kind == "duplicate" {
				add("package/frpc", tar.TypeReg)
			}
			if kind != "missing" {
				add("package/LICENSE", tar.TypeReg)
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(dir, "out")
			if err := os.Mkdir(output, 0700); err != nil {
				t.Fatal(err)
			}
			err = extractClient(archive, output, "package")
			if (err == nil) != (kind == "valid") {
				t.Fatal("unexpected extraction result", err)
			}
			entries, _ := os.ReadDir(output)
			for _, entry := range entries {
				if entry.Name() != "frpc" && entry.Name() != "LICENSE" {
					t.Fatal("unexpected extracted entry", entry.Name())
				}
			}
		})
	}
}

func TestUninstallDoesNotRemoveExecutableWhenStopFails(t *testing.T) {
	s := testService(t)
	s.system.stop = func() error { return os.ErrPermission }
	if err := s.uninstall(); err == nil || !s.installed() {
		t.Fatal("failed stop still removed service", err)
	}
}
