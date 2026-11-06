package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	envRoot        = "MEMFAB_ROOT"
	envToolsSocket = "MEMFAB_TOOLS_SOCKET"
	DirectoryMode  = 0o700
	FileMode       = 0o600
)

func Root() string {
	if root := os.Getenv(envRoot); root != "" {
		return root
	}
	return "/var/lib/memfab"
}

func MemoryDir() string            { return filepath.Join(Root(), "memory") }
func ObservationBufferDir() string { return filepath.Join(Root(), "observation-buffer") }
func ScreenshotsDir() string       { return filepath.Join(Root(), "screenshots") }
func ReplayDir() string            { return filepath.Join(Root(), "replay") }
func LogsDir() string              { return filepath.Join(Root(), "logs") }
func ConfigFile() string           { return filepath.Join(Root(), "config.toml") }
func ConfigRevisionKey() string    { return filepath.Join(Root(), ".config-revision.key") }
func IndexDB() string              { return filepath.Join(Root(), "index.db") }
func PIDFile() string              { return filepath.Join(Root(), ".pid") }
func PausedFlag() string           { return filepath.Join(Root(), ".paused") }
func WriterState() string          { return filepath.Join(Root(), "writer-state.json") }

// ToolsSocket is the local-only bridge used by NanoKVM-Server. Production
// deployments create its parent through systemd's RuntimeDirectory support;
// the environment override is intentionally limited to tests and diagnostics.
func ToolsSocket() string {
	if path := os.Getenv(envToolsSocket); path != "" {
		return path
	}
	return "/run/memfab/tools.sock"
}

func EnsureDirs() error {
	for _, dir := range []string{Root(), MemoryDir(), ObservationBufferDir(), LogsDir()} {
		if err := ensurePrivateDirectory(dir); err != nil {
			return err
		}
	}
	return migratePrivateTree(Root())
}

// EnsureObservationBufferDir avoids walking index.db files while SQLite is open.
func EnsureObservationBufferDir() error {
	return ensurePrivateDirectory(ObservationBufferDir())
}

// WritePrivateFile atomically replaces a regular data file without following a
// pre-existing symlink at the destination.
func WritePrivateFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := ensurePrivateDirectory(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(FileMode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// SecureRegularFiles rejects symlinks and special files, and repairs the mode
// of each existing regular file. Missing files are allowed for SQLite sidecars
// and other lazily created data.
func SecureRegularFiles(files ...string) error {
	for _, path := range files {
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("memfab data path is not a regular file: %s", path)
		}
		if info.Mode().Perm() != FileMode {
			if err := chmodNoFollow(path, info, FileMode); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensurePrivateDirectory(path string) error {
	directory, err := openPrivateDirectory(path)
	if err != nil {
		return err
	}
	return directory.Close()
}

// migratePrivateTree repairs legacy 0755/0644 data without traversing or
// chmodding symlinks. All regular files under MEMFAB_ROOT contain private
// configuration, memory, OCR, process, or diagnostic data.
func migratePrivateTree(root string) error {
	directory, err := openPrivateDirectory(root)
	if err != nil {
		return err
	}
	defer directory.Close()
	return migratePrivateDirectory(directory, root)
}

// openPrivateDirectory resolves every component below MEMFAB_ROOT through a
// directory descriptor. O_NOFOLLOW on each hop prevents an intermediate
// symlink (or a directory swapped for one) from redirecting data operations
// outside the private tree.
func openPrivateDirectory(path string) (*os.File, error) {
	root := filepath.Clean(Root())
	path = filepath.Clean(path)
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("memfab data directory escapes root: %s", path)
	}
	if err := os.MkdirAll(root, DirectoryMode); err != nil {
		return nil, err
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("memfab data path is not a directory: %s", root)
	}

	rootFD, err := syscall.Open(root, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(rootFD), root)
	if current == nil {
		_ = syscall.Close(rootFD)
		return nil, fmt.Errorf("open memfab data directory: %s", root)
	}
	if err := current.Chmod(DirectoryMode); err != nil {
		_ = current.Close()
		return nil, err
	}
	if relative == "." {
		return current, nil
	}

	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		nextFD, openErr := syscall.Openat(int(current.Fd()), component, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
		if errors.Is(openErr, syscall.ENOENT) {
			if mkdirErr := syscall.Mkdirat(int(current.Fd()), component, uint32(DirectoryMode)); mkdirErr != nil && !errors.Is(mkdirErr, syscall.EEXIST) {
				_ = current.Close()
				return nil, mkdirErr
			}
			nextFD, openErr = syscall.Openat(int(current.Fd()), component, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
		}
		if openErr != nil {
			_ = current.Close()
			return nil, fmt.Errorf("open private memfab directory %s: %w", path, openErr)
		}
		next := os.NewFile(uintptr(nextFD), filepath.Join(current.Name(), component))
		if next == nil {
			_ = syscall.Close(nextFD)
			_ = current.Close()
			return nil, fmt.Errorf("open private memfab directory: %s", path)
		}
		if err := next.Chmod(DirectoryMode); err != nil {
			_ = next.Close()
			_ = current.Close()
			return nil, err
		}
		_ = current.Close()
		current = next
	}
	return current, nil
}

const openPath = 0x200000 // Linux O_PATH; not exported by syscall on all supported architectures.

func migratePrivateDirectory(directory *os.File, displayPath string) error {
	names, err := directory.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		probeFD, err := syscall.Openat(int(directory.Fd()), name, openPath|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, 0)
		if err != nil {
			if errors.Is(err, syscall.ENOENT) {
				// Atomic writers may remove a temporary file between readdir
				// and inspection. It no longer needs migration.
				continue
			}
			return fmt.Errorf("inspect memfab data path %s: %w", filepath.Join(displayPath, name), err)
		}
		probe := os.NewFile(uintptr(probeFD), filepath.Join(displayPath, name))
		if probe == nil {
			_ = syscall.Close(probeFD)
			return fmt.Errorf("inspect memfab data path: %s", filepath.Join(displayPath, name))
		}
		probeInfo, statErr := probe.Stat()
		_ = probe.Close()
		if statErr != nil {
			return statErr
		}
		if !probeInfo.IsDir() && !probeInfo.Mode().IsRegular() {
			continue
		}

		flags := syscall.O_RDONLY | syscall.O_CLOEXEC | syscall.O_NOFOLLOW
		mode := os.FileMode(FileMode)
		if probeInfo.IsDir() {
			flags |= syscall.O_DIRECTORY
			mode = os.FileMode(DirectoryMode)
		} else {
			flags |= syscall.O_NONBLOCK
		}
		childFD, err := syscall.Openat(int(directory.Fd()), name, flags, 0)
		if err != nil {
			if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ELOOP) {
				// A removed entry needs no migration; an entry swapped to a
				// symlink is deliberately left untouched.
				continue
			}
			return fmt.Errorf("secure memfab data path %s: %w", filepath.Join(displayPath, name), err)
		}
		child := os.NewFile(uintptr(childFD), filepath.Join(displayPath, name))
		if child == nil {
			_ = syscall.Close(childFD)
			return fmt.Errorf("secure memfab data path: %s", filepath.Join(displayPath, name))
		}
		actual, err := child.Stat()
		if err != nil || !os.SameFile(probeInfo, actual) {
			_ = child.Close()
			if err != nil {
				return err
			}
			return fmt.Errorf("memfab data path changed during permission migration: %s", filepath.Join(displayPath, name))
		}
		if actual.Mode().Perm() != mode {
			if err := child.Chmod(mode); err != nil {
				_ = child.Close()
				return err
			}
		}
		if actual.IsDir() {
			if err := migratePrivateDirectory(child, filepath.Join(displayPath, name)); err != nil {
				_ = child.Close()
				return err
			}
		}
		if err := child.Close(); err != nil {
			return err
		}
	}
	return nil
}

// chmodNoFollow resolves the parent below MEMFAB_ROOT through descriptors,
// opens the final object with O_NOFOLLOW, verifies its identity, and changes
// its mode through the file descriptor.
func chmodNoFollow(path string, expected os.FileInfo, mode os.FileMode) error {
	parent, err := openPrivateDirectory(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer parent.Close()
	flags := syscall.O_RDONLY | syscall.O_CLOEXEC | syscall.O_NOFOLLOW
	if expected.IsDir() {
		flags |= syscall.O_DIRECTORY
	} else {
		flags |= syscall.O_NONBLOCK
	}
	fd, err := syscall.Openat(int(parent.Fd()), filepath.Base(path), flags, 0)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = syscall.Close(fd)
		return fmt.Errorf("secure memfab data path: %s", path)
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(expected, actual) || expected.IsDir() != actual.IsDir() || expected.Mode().IsRegular() != actual.Mode().IsRegular() {
		return fmt.Errorf("memfab data path changed during permission migration: %s", path)
	}
	return file.Chmod(mode)
}
