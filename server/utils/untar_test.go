package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testTarEntry struct {
	name     string
	typeflag byte
	mode     int64
	content  string
	linkname string
}

func TestUnTarGzExtractsArchiveTransactionally(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "payload.tar.gz")
	destDir := filepath.Join(tempDir, "dest")
	writeTestTarGz(t, archivePath, []testTarEntry{
		{name: "bundle", typeflag: tar.TypeDir, mode: 0o777},
		{name: "bundle/bin/tool", typeflag: tar.TypeReg, mode: 0o4755, content: "binary"},
		{name: "bundle/config", typeflag: tar.TypeReg, mode: 0o666, content: "config"},
	})

	target, err := UnTarGz(archivePath, destDir)
	if err != nil {
		t.Fatalf("UnTarGz: %v", err)
	}
	if target != filepath.Join(destDir, "bundle") {
		t.Fatalf("target = %q", target)
	}
	data, err := os.ReadFile(filepath.Join(target, "bin", "tool"))
	if err != nil || string(data) != "binary" {
		t.Fatalf("read extracted tool: data=%q err=%v", data, err)
	}
	assertMode(t, filepath.Join(target, "bin", "tool"), 0o755)
	assertMode(t, filepath.Join(target, "config"), 0o644)
	assertNoTarStaging(t, destDir)
}

func TestUnTarGzRejectsUnsafeAndSpecialEntriesWithoutPartialOutput(t *testing.T) {
	tests := []struct {
		name  string
		entry testTarEntry
	}{
		{name: "traversal", entry: testTarEntry{name: "../escaped", typeflag: tar.TypeReg, mode: 0o644, content: "x"}},
		{name: "absolute", entry: testTarEntry{name: "/tmp/escaped", typeflag: tar.TypeReg, mode: 0o644, content: "x"}},
		{name: "backslash", entry: testTarEntry{name: "bundle\\..\\escaped", typeflag: tar.TypeReg, mode: 0o644, content: "x"}},
		{name: "symlink", entry: testTarEntry{name: "bundle/link", typeflag: tar.TypeSymlink, linkname: "/etc/passwd"}},
		{name: "hardlink", entry: testTarEntry{name: "bundle/link", typeflag: tar.TypeLink, linkname: "bundle/file"}},
		{name: "character device", entry: testTarEntry{name: "bundle/device", typeflag: tar.TypeChar}},
		{name: "block device", entry: testTarEntry{name: "bundle/device", typeflag: tar.TypeBlock}},
		{name: "fifo", entry: testTarEntry{name: "bundle/fifo", typeflag: tar.TypeFifo}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			archivePath := filepath.Join(tempDir, "payload.tar.gz")
			destDir := filepath.Join(tempDir, "dest")
			writeTestTarGz(t, archivePath, []testTarEntry{
				{name: "bundle/first", typeflag: tar.TypeReg, mode: 0o644, content: "first"},
				test.entry,
			})

			if _, err := UnTarGz(archivePath, destDir); err == nil {
				t.Fatal("expected archive to be rejected")
			}
			if _, err := os.Stat(filepath.Join(destDir, "bundle")); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("partial archive output remained: %v", err)
			}
			assertNoTarStaging(t, destDir)
		})
	}
}

func TestUnTarGzEnforcesResourceLimits(t *testing.T) {
	tests := []struct {
		name    string
		entries []testTarEntry
		limits  tarExtractionLimits
		kind    string
	}{
		{
			name: "entry count",
			entries: []testTarEntry{
				{name: "bundle/a", typeflag: tar.TypeReg, mode: 0o644, content: "a"},
				{name: "bundle/b", typeflag: tar.TypeReg, mode: 0o644, content: "b"},
			},
			limits: tarExtractionLimits{maxEntries: 1, maxFileBytes: 10, maxExpandedBytes: 20},
			kind:   "entry count",
		},
		{
			name:    "single file",
			entries: []testTarEntry{{name: "bundle/a", typeflag: tar.TypeReg, mode: 0o644, content: "12345"}},
			limits:  tarExtractionLimits{maxEntries: 2, maxFileBytes: 4, maxExpandedBytes: 20},
			kind:    "file size",
		},
		{
			name: "aggregate expansion",
			entries: []testTarEntry{
				{name: "bundle/a", typeflag: tar.TypeReg, mode: 0o644, content: "123"},
				{name: "bundle/b", typeflag: tar.TypeReg, mode: 0o644, content: "456"},
			},
			limits: tarExtractionLimits{maxEntries: 3, maxFileBytes: 4, maxExpandedBytes: 5},
			kind:   "expanded size",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tempDir := t.TempDir()
			archivePath := filepath.Join(tempDir, "payload.tar.gz")
			destDir := filepath.Join(tempDir, "dest")
			writeTestTarGz(t, archivePath, test.entries)

			_, err := unTarGzWithLimits(archivePath, destDir, test.limits)
			if !errors.Is(err, ErrArchiveLimit) {
				t.Fatalf("expected ErrArchiveLimit, got %v", err)
			}
			var limitErr *ArchiveLimitError
			if !errors.As(err, &limitErr) || limitErr.Kind != test.kind {
				t.Fatalf("unexpected typed limit error: %#v", limitErr)
			}
			if _, statErr := os.Stat(filepath.Join(destDir, "bundle")); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("partial output remained: %v", statErr)
			}
			assertNoTarStaging(t, destDir)
		})
	}
}

func TestUnTarGzRejectsTruncatedContentWithoutPartialOutput(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "payload.tar.gz")
	destDir := filepath.Join(tempDir, "dest")
	writeTestTarGz(t, archivePath, []testTarEntry{{name: "bundle/file", typeflag: tar.TypeReg, mode: 0o644, content: strings.Repeat("payload", 50)}})

	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("read archive: %v", err)
	}
	if err := os.WriteFile(archivePath, data[:len(data)-8], 0o600); err != nil {
		t.Fatalf("truncate archive: %v", err)
	}

	if _, err := UnTarGz(archivePath, destDir); err == nil {
		t.Fatal("expected truncated archive error")
	}
	if _, err := os.Stat(filepath.Join(destDir, "bundle")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial output remained: %v", err)
	}
	assertNoTarStaging(t, destDir)
}

func TestUnTarGzRejectsTrailingExpansionPastEnvelope(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "payload.tar.gz")
	destDir := filepath.Join(tempDir, "dest")
	writeTestTarGzWithTrailing(t, archivePath,
		[]testTarEntry{{name: "bundle/file", typeflag: tar.TypeReg, mode: 0o644, content: "ok"}},
		strings.Repeat("x", 4096),
	)

	limits := tarExtractionLimits{
		maxEntries:       2,
		maxFileBytes:     16,
		maxExpandedBytes: 16,
		maxEnvelopeBytes: 3072,
	}
	_, err := unTarGzWithLimits(archivePath, destDir, limits)
	if !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("expected envelope ErrArchiveLimit, got %v", err)
	}
	var limitErr *ArchiveLimitError
	if !errors.As(err, &limitErr) || limitErr.Kind != "stream size" {
		t.Fatalf("unexpected envelope limit error: %#v", limitErr)
	}
	if _, err := os.Stat(filepath.Join(destDir, "bundle")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial output remained: %v", err)
	}
	assertNoTarStaging(t, destDir)
}

func TestUnTarGzRejectsExistingDestinationWithoutChangingIt(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "payload.tar.gz")
	destDir := filepath.Join(tempDir, "dest")
	existing := filepath.Join(destDir, "bundle")
	if err := os.MkdirAll(existing, 0o755); err != nil {
		t.Fatalf("create existing destination: %v", err)
	}
	if err := os.WriteFile(filepath.Join(existing, "keep"), []byte("original"), 0o600); err != nil {
		t.Fatalf("write existing content: %v", err)
	}
	writeTestTarGz(t, archivePath, []testTarEntry{{name: "bundle/new", typeflag: tar.TypeReg, mode: 0o644, content: "replacement"}})

	if _, err := UnTarGz(archivePath, destDir); err == nil {
		t.Fatal("expected existing destination error")
	}
	data, err := os.ReadFile(filepath.Join(existing, "keep"))
	if err != nil || string(data) != "original" {
		t.Fatalf("existing content changed: data=%q err=%v", data, err)
	}
	if _, err := os.Stat(filepath.Join(existing, "new")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("new archive content was committed: %v", err)
	}
	assertNoTarStaging(t, destDir)
}

func writeTestTarGz(t *testing.T, archivePath string, entries []testTarEntry) {
	writeTestTarGzWithTrailing(t, archivePath, entries, "")
}

func writeTestTarGzWithTrailing(t *testing.T, archivePath string, entries []testTarEntry, trailing string) {
	t.Helper()
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("create archive: %v", err)
	}
	gzWriter := gzip.NewWriter(archiveFile)
	tarWriter := tar.NewWriter(gzWriter)
	for _, entry := range entries {
		header := &tar.Header{
			Name:     entry.name,
			Mode:     entry.mode,
			Size:     int64(len(entry.content)),
			Typeflag: entry.typeflag,
			Linkname: entry.linkname,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("write header %q: %v", entry.name, err)
		}
		if entry.content != "" {
			if _, err := tarWriter.Write([]byte(entry.content)); err != nil {
				t.Fatalf("write content %q: %v", entry.name, err)
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if trailing != "" {
		if _, err := gzWriter.Write([]byte(trailing)); err != nil {
			t.Fatalf("write trailing expansion: %v", err)
		}
	}
	if err := gzWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	if err := archiveFile.Close(); err != nil {
		t.Fatalf("close archive file: %v", err)
	}
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("mode %s = %04o, want %04o", path, got, want)
	}
}

func assertNoTarStaging(t *testing.T, destDir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(destDir, ".untar-*"))
	if err != nil {
		t.Fatalf("glob staging directories: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("staging output remained: %#v", matches)
	}
}
