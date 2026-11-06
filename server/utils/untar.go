package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// These limits leave ample room for current NanoKVM update packages and
	// official Tailscale bundles while preventing unbounded disk expansion.
	DefaultTarMaxEntries       = 4096
	DefaultTarMaxFileBytes     = int64(256 << 20)
	DefaultTarMaxExpandedBytes = int64(512 << 20)
	// The envelope also covers tar headers, padding, extended metadata, and
	// bytes after the end-of-archive marker that gzip must consume to verify
	// its checksum. Bounding the whole decompressed stream prevents a small
	// archive followed by an expansion bomb from bypassing the entry limits.
	DefaultTarMaxEnvelopeBytes = DefaultTarMaxExpandedBytes + int64(8<<20)
)

var ErrArchiveLimit = errors.New("archive extraction limit exceeded")

type ArchiveLimitError struct {
	Kind   string
	Limit  int64
	Actual int64
}

func (e *ArchiveLimitError) Error() string {
	return fmt.Sprintf("archive %s limit exceeded: %d > %d", e.Kind, e.Actual, e.Limit)
}

func (e *ArchiveLimitError) Unwrap() error { return ErrArchiveLimit }

type tarExtractionLimits struct {
	maxEntries       int64
	maxFileBytes     int64
	maxExpandedBytes int64
	maxEnvelopeBytes int64
}

var defaultTarExtractionLimits = tarExtractionLimits{
	maxEntries:       DefaultTarMaxEntries,
	maxFileBytes:     DefaultTarMaxFileBytes,
	maxExpandedBytes: DefaultTarMaxExpandedBytes,
	maxEnvelopeBytes: DefaultTarMaxEnvelopeBytes,
}

func UnTarGz(srcFile string, destDir string) (string, error) {
	return unTarGzWithLimits(srcFile, destDir, defaultTarExtractionLimits)
}

func unTarGzWithLimits(srcFile string, destDir string, limits tarExtractionLimits) (string, error) {
	cleanDest, err := filepath.Abs(destDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(cleanDest, 0o755); err != nil {
		return "", err
	}

	stagingDir, err := os.MkdirTemp(cleanDest, ".untar-*")
	if err != nil {
		return "", err
	}
	if err := os.Chmod(stagingDir, 0o700); err != nil {
		_ = os.RemoveAll(stagingDir)
		return "", err
	}
	defer func() {
		_ = os.RemoveAll(stagingDir)
	}()

	fr, err := os.Open(srcFile)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = fr.Close()
	}()

	gr, err := gzip.NewReader(fr)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = gr.Close()
	}()

	envelopeLimit := limits.maxEnvelopeBytes
	if envelopeLimit <= 0 {
		envelopeLimit = limits.maxExpandedBytes + int64(8<<20)
	}
	limitedEnvelope := &io.LimitedReader{R: gr, N: envelopeLimit + 1}
	topLevels, err := extractTar(tar.NewReader(limitedEnvelope), stagingDir, limits)
	if err != nil {
		return "", err
	}
	if err := drainTarEnvelope(limitedEnvelope, envelopeLimit); err != nil {
		return "", err
	}
	if err := gr.Close(); err != nil {
		return "", err
	}
	if len(topLevels) == 0 {
		return "", nil
	}
	if err := commitExtractedArchive(stagingDir, cleanDest, topLevels); err != nil {
		return "", err
	}

	return filepath.Join(cleanDest, topLevels[0]), nil
}

func drainTarEnvelope(envelope *io.LimitedReader, limit int64) error {
	remaining := envelope.N
	if _, err := io.CopyN(io.Discard, envelope, remaining); err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if envelope.N == 0 {
		return &ArchiveLimitError{Kind: "stream size", Limit: limit, Actual: limit + 1}
	}
	return nil
}

func extractTar(tr *tar.Reader, stagingDir string, limits tarExtractionLimits) ([]string, error) {
	var entries int64
	var expanded int64
	topLevels := make([]string, 0, 1)
	seenTopLevels := make(map[string]struct{})

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		entries++
		if entries > limits.maxEntries {
			return nil, &ArchiveLimitError{Kind: "entry count", Limit: limits.maxEntries, Actual: entries}
		}
		if header.Size < 0 {
			return nil, fmt.Errorf("archive entry %q has a negative size", header.Name)
		}
		if header.Size > limits.maxFileBytes {
			return nil, &ArchiveLimitError{Kind: "file size", Limit: limits.maxFileBytes, Actual: header.Size}
		}
		if header.Size > limits.maxExpandedBytes-expanded {
			return nil, &ArchiveLimitError{Kind: "expanded size", Limit: limits.maxExpandedBytes, Actual: expanded + header.Size}
		}
		expanded += header.Size

		filename, cleanName, err := safeArchiveStagingPath(stagingDir, header.Name)
		if err != nil {
			return nil, err
		}
		topLevel := strings.Split(filepath.ToSlash(cleanName), "/")[0]
		if _, ok := seenTopLevels[topLevel]; !ok {
			seenTopLevels[topLevel] = struct{}{}
			topLevels = append(topLevels, topLevel)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(filename, safeTarDirectoryMode(header.FileInfo().Mode())); err != nil {
				return nil, err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := extractTarRegularFile(tr, filename, header); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("refusing to extract unsupported archive entry %q (type %d)", header.Name, header.Typeflag)
		}
	}

	return topLevels, nil
}

func extractTarRegularFile(tr *tar.Reader, filename string, header *tar.Header) error {
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, safeTarFileMode(header.FileInfo().Mode()))
	if err != nil {
		return err
	}

	limited := &io.LimitedReader{R: tr, N: header.Size}
	written, copyErr := io.Copy(file, limited)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if written != header.Size || limited.N != 0 {
		return io.ErrUnexpectedEOF
	}
	return closeErr
}

func commitExtractedArchive(stagingDir, destDir string, topLevels []string) error {
	for _, topLevel := range topLevels {
		if _, err := os.Lstat(filepath.Join(destDir, topLevel)); err == nil {
			return fmt.Errorf("archive destination already exists: %s", topLevel)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	committed := make([]string, 0, len(topLevels))
	for _, topLevel := range topLevels {
		from := filepath.Join(stagingDir, topLevel)
		to := filepath.Join(destDir, topLevel)
		if err := os.Rename(from, to); err != nil {
			for index := len(committed) - 1; index >= 0; index-- {
				name := committed[index]
				_ = os.Rename(filepath.Join(destDir, name), filepath.Join(stagingDir, name))
			}
			return err
		}
		committed = append(committed, topLevel)
	}
	return nil
}

func safeArchiveStagingPath(destDir, name string) (string, string, error) {
	if name == "" || strings.ContainsRune(name, '\x00') || strings.ContainsRune(name, '\\') {
		return "", "", fmt.Errorf("invalid archive path")
	}

	cleanName := filepath.Clean(name)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) || filepath.IsAbs(name) {
		return "", "", fmt.Errorf("invalid archive path")
	}

	fullPath := filepath.Join(destDir, cleanName)
	rel, err := filepath.Rel(destDir, fullPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("archive path escapes destination")
	}

	return fullPath, cleanName, nil
}

func safeArchivePath(destDir string, name string) (string, error) {
	path, _, err := safeArchiveStagingPath(destDir, name)
	return path, err
}

func safeTarDirectoryMode(mode os.FileMode) os.FileMode {
	result := mode.Perm() & 0o755
	if result == 0 {
		return 0o755
	}
	return result
}

func safeTarFileMode(mode os.FileMode) os.FileMode {
	result := mode.Perm() & 0o755
	if result == 0 {
		return 0o644
	}
	return result
}
