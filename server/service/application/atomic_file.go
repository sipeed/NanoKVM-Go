package application

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// publishAtomicFile replaces path with payload through a same-directory
// temporary file, a permission change, a file fsync, an atomic rename, and a
// directory fsync so the rename is durable across a power loss on the device
// filesystems.
//
// It refuses a symlink or a non-regular existing target and removes the
// temporary file on every failure path. The temporary file is named
// ".<base>-*.tmp" in the target directory, so a failed publish can never be
// mistaken for the destination file.
func publishAtomicFile(path string, payload []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := ensurePublishDirectory(directory); err != nil {
		return err
	}

	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("target path is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect target file: %w", err)
	}

	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporary.Name()
	keepTemporary := true
	defer func() {
		_ = temporary.Close()
		if keepTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()

	if err := temporary.Chmod(mode); err != nil {
		return fmt.Errorf("set temporary file permissions: %w", err)
	}
	if _, err := io.WriteString(temporary, string(payload)); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish file: %w", err)
	}
	keepTemporary = false

	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open directory: %w", err)
	}
	defer func() { _ = directoryFile.Close() }()
	if err := directoryFile.Sync(); err != nil {
		return fmt.Errorf("sync directory: %w", err)
	}
	return nil
}

// ensurePublishDirectory creates the parent directory as 0o755 when it is
// missing and rejects a symlink or a non-directory entry.
func ensurePublishDirectory(directory string) error {
	if info, err := os.Lstat(directory); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return errors.New("publish directory is invalid")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect publish directory: %w", err)
	}

	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create publish directory: %w", err)
	}
	info, err := os.Lstat(directory)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("publish directory is invalid")
	}
	return nil
}
