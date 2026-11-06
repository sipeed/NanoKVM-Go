package apps

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	maxArchiveBytes = 64 << 20
	maxFiles        = 2048
)

type inspectedArchive struct {
	Manifest Manifest
	Prefix   string
	Reader   *zip.Reader
}

func inspectRepositoryArchive(data []byte) (Catalog, map[string]*inspectedArchive, error) {
	if len(data) == 0 || len(data) > maxArchiveBytes {
		return Catalog{}, nil, errors.New("repository ZIP archive is empty or too large")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return Catalog{}, nil, errors.New("invalid repository ZIP archive")
	}
	if len(zr.File) == 0 || len(zr.File) > maxFiles {
		return Catalog{}, nil, errors.New("repository ZIP archive has an invalid file count")
	}

	files := make(map[string]*zip.File, len(zr.File))
	var expanded uint64
	for _, file := range zr.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return Catalog{}, nil, err
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return Catalog{}, nil, errors.New("ZIP symlinks are not allowed")
		}
		expanded += file.UncompressedSize64
		if expanded > maxArchiveBytes {
			return Catalog{}, nil, errors.New("expanded repository ZIP archive is too large")
		}
		files[clean] = file
	}

	apps := make([]CatalogApp, 0)
	archives := make(map[string]*inspectedArchive)
	packages := make(map[string]bool)
	for name, file := range files {
		parts := strings.Split(name, "/")
		if len(parts) != 3 || parts[2] != "app.json" {
			continue
		}
		prefix := parts[0] + "/" + parts[1] + "/"
		main := files[prefix+"main.py"]
		if main == nil || main.FileInfo().IsDir() {
			continue
		}
		manifestData, err := readZipFile(file, 1<<20)
		if err != nil {
			return Catalog{}, nil, err
		}
		manifest, err := parseManifest(manifestData)
		if err != nil {
			return Catalog{}, nil, fmt.Errorf("invalid %sapp.json: %w", prefix, err)
		}
		clearSecretValues(&manifest)
		id := parts[1]
		if !appIDPattern.MatchString(id) {
			continue
		}
		if err := validateAppIDForDirectory(manifest, id); err != nil {
			return Catalog{}, nil, err
		}
		if _, exists := archives[id]; exists {
			return Catalog{}, nil, fmt.Errorf("duplicate App directory %q in repository", id)
		}
		if packages[manifest.AppID] {
			return Catalog{}, nil, fmt.Errorf("duplicate app_id %q in repository", manifest.AppID)
		}
		packages[manifest.AppID] = true
		archives[id] = &inspectedArchive{Manifest: manifest, Prefix: prefix, Reader: zr}
		apps = append(apps, CatalogApp{Directory: id, Archive: parts[1], Manifest: publicManifest(manifest)})
	}
	if len(apps) == 0 {
		return Catalog{}, nil, errors.New("GitHub repository contains no valid Apps")
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Manifest.Name < apps[j].Manifest.Name })
	return Catalog{Apps: apps}, archives, nil
}

func inspectArchive(data []byte) (*inspectedArchive, error) {
	if len(data) == 0 || len(data) > maxArchiveBytes {
		return nil, errors.New("ZIP archive is empty or too large")
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("invalid ZIP archive")
	}
	if len(zr.File) == 0 || len(zr.File) > maxFiles {
		return nil, errors.New("ZIP archive has an invalid file count")
	}

	manifestFiles := make([]*zip.File, 0, 1)
	var expanded uint64
	for _, file := range zr.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return nil, err
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("ZIP symlinks are not allowed")
		}
		expanded += file.UncompressedSize64
		if expanded > maxArchiveBytes {
			return nil, errors.New("expanded ZIP archive is too large")
		}
		if path.Base(clean) == "app.json" {
			manifestFiles = append(manifestFiles, file)
		}
	}
	if len(manifestFiles) != 1 {
		return nil, errors.New("ZIP archive must contain exactly one app.json")
	}
	manifestPath, _ := cleanArchivePath(manifestFiles[0].Name)
	prefix := strings.TrimSuffix(manifestPath, "app.json")
	if prefix == "" || strings.Count(strings.TrimSuffix(prefix, "/"), "/") > 0 {
		return nil, errors.New("ZIP must contain exactly one top-level App directory with app.json")
	}
	hasMain := false
	for _, file := range zr.File {
		clean, _ := cleanArchivePath(file.Name)
		if prefix != "" && clean != strings.TrimSuffix(prefix, "/") && !strings.HasPrefix(clean, prefix) {
			return nil, errors.New("ZIP entries must belong to the App directory")
		}
		if clean == prefix+"main.py" && !file.FileInfo().IsDir() {
			hasMain = true
		}
	}
	if !hasMain {
		return nil, errors.New("App archive does not contain main.py")
	}
	manifestData, err := readZipFile(manifestFiles[0], 1<<20)
	if err != nil {
		return nil, err
	}
	manifest, err := parseManifest(manifestData)
	if err != nil {
		return nil, err
	}
	clearSecretValues(&manifest)
	directory := strings.TrimSuffix(prefix, "/")
	if !appIDPattern.MatchString(directory) {
		return nil, errors.New("top-level App directory must use lowercase kebab-case")
	}
	if err := validateAppIDForDirectory(manifest, directory); err != nil {
		return nil, err
	}
	return &inspectedArchive{Manifest: manifest, Prefix: prefix, Reader: zr}, nil
}

func cleanArchivePath(name string) (string, error) {
	if strings.ContainsRune(name, '\\') || strings.ContainsRune(name, '\x00') {
		return "", errors.New("invalid ZIP entry path")
	}
	clean := path.Clean(name)
	if clean == "." || strings.HasPrefix(clean, "/") || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("unsafe ZIP entry path")
	}
	return clean, nil
}

func readZipFile(file *zip.File, limit int64) ([]byte, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("app.json is too large")
	}
	return data, nil
}

func extractArchive(archive *inspectedArchive, destination string, manifest Manifest) error {
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	for _, file := range archive.Reader.File {
		clean, _ := cleanArchivePath(file.Name)
		if archive.Prefix != "" && clean != strings.TrimSuffix(archive.Prefix, "/") && !strings.HasPrefix(clean, archive.Prefix) {
			continue
		}
		if archive.Prefix != "" && clean == strings.TrimSuffix(archive.Prefix, "/") {
			continue
		}
		rel := strings.TrimPrefix(clean, archive.Prefix)
		if rel == "" || rel == "." {
			continue
		}
		target := filepath.Join(destination, filepath.FromSlash(rel))
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		r, err := file.Open()
		if err != nil {
			return err
		}
		mode := file.Mode().Perm() & 0o755
		if mode == 0 {
			mode = 0o644
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err == nil {
			_, err = io.Copy(out, r)
			closeErr := out.Close()
			if err == nil {
				err = closeErr
			}
		}
		r.Close()
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(destination, "main.py")); err != nil {
		return errors.New("App archive does not contain main.py")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(destination, "app.json"), data, 0o600)
}

func writeArchive(w io.Writer, appDir, directoryName string, manifest Manifest) error {
	zw := zip.NewWriter(w)
	err := filepath.Walk(appDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(appDir, filePath)
		if err != nil || rel == "." {
			return err
		}
		name := path.Join(directoryName, filepath.ToSlash(rel))
		if info.IsDir() {
			_, err = zw.Create(name + "/")
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = name
		header.Method = zip.Deflate
		entry, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if rel == "app.json" {
			data, err := json.MarshalIndent(publicManifest(manifest), "", "  ")
			if err != nil {
				return err
			}
			_, err = entry.Write(append(data, '\n'))
			return err
		}
		in, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(entry, in)
		return err
	})
	if err != nil {
		_ = zw.Close()
		return fmt.Errorf("create App ZIP: %w", err)
	}
	return zw.Close()
}
