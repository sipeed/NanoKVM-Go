package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootEnvOverride(t *testing.T) {
	t.Setenv("MEMFAB_ROOT", "/tmp/memfab-test")
	if Root() != "/tmp/memfab-test" {
		t.Fatalf("Root()=%q", Root())
	}
	if ObservationBufferDir() != filepath.Join("/tmp/memfab-test", "observation-buffer") {
		t.Fatalf("ObservationBufferDir()=%q", ObservationBufferDir())
	}
}

func TestToolsSocketDefaultAndOverride(t *testing.T) {
	t.Setenv("MEMFAB_TOOLS_SOCKET", "")
	if got := ToolsSocket(); got != "/run/memfab/tools.sock" {
		t.Fatalf("ToolsSocket()=%q", got)
	}

	want := filepath.Join(t.TempDir(), "tools.sock")
	t.Setenv("MEMFAB_TOOLS_SOCKET", want)
	if got := ToolsSocket(); got != want {
		t.Fatalf("ToolsSocket()=%q, want %q", got, want)
	}
}

func TestEnsureDirsDoesNotCreateScreenshotsDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memfab")
	t.Setenv("MEMFAB_ROOT", root)

	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{Root(), MemoryDir(), ObservationBufferDir(), LogsDir()} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("required directory %s missing or invalid: %v", dir, err)
		}
	}
	if _, err := os.Stat(ScreenshotsDir()); !os.IsNotExist(err) {
		t.Fatalf("screenshots directory should not be created: %v", err)
	}
}

func TestEnsureDirsMigratesLegacyTreeWithoutFollowingSymlinks(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "memfab")
	legacyDir := filepath.Join(root, "screenshots", "2026-08-07")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyFile := filepath.Join(legacyDir, "capture.jpg")
	if err := os.WriteFile(legacyFile, []byte("private capture"), 0o644); err != nil {
		t.Fatal(err)
	}
	timelapseDir := filepath.Join(root, "timelapse", "2026-08-07")
	if err := os.MkdirAll(timelapseDir, 0o755); err != nil {
		t.Fatal(err)
	}
	timelapseFile := filepath.Join(timelapseDir, "screen.mp4")
	if err := os.WriteFile(timelapseFile, []byte("private timelapse"), 0o644); err != nil {
		t.Fatal(err)
	}
	databaseFiles := []string{
		filepath.Join(root, "index.db"),
		filepath.Join(root, "index.db-wal"),
		filepath.Join(root, "index.db-shm"),
	}
	for _, path := range databaseFiles {
		if err := os.WriteFile(path, []byte("legacy sqlite data"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(root, 0o755); err != nil {
		t.Fatal(err)
	}

	externalDir := filepath.Join(parent, "external")
	if err := os.Mkdir(externalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	externalFile := filepath.Join(externalDir, "outside.txt")
	if err := os.WriteFile(externalFile, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalDir, filepath.Join(root, "linked-dir")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(externalFile, filepath.Join(root, "linked-file")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MEMFAB_ROOT", root)
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{root, legacyDir, timelapseDir, MemoryDir(), ObservationBufferDir(), LogsDir()} {
		assertMode(t, path, DirectoryMode)
	}
	assertMode(t, legacyFile, FileMode)
	assertMode(t, timelapseFile, FileMode)
	for _, path := range databaseFiles {
		assertMode(t, path, FileMode)
	}
	assertMode(t, externalDir, 0o755)
	assertMode(t, externalFile, 0o644)
	for _, path := range []string{filepath.Join(root, "linked-dir"), filepath.Join(root, "linked-file")} {
		info, err := os.Lstat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Fatalf("%s is no longer a symlink", path)
		}
	}
}

func TestEnsureObservationBufferDirDoesNotWalkRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "memfab")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	databaseFile := filepath.Join(root, "index.db-wal")
	if err := os.WriteFile(databaseFile, []byte("sqlite data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(databaseFile, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MEMFAB_ROOT", root)
	if err := EnsureObservationBufferDir(); err != nil {
		t.Fatal(err)
	}

	assertMode(t, ObservationBufferDir(), DirectoryMode)
	assertMode(t, databaseFile, 0o644)
}

func TestWritePrivateFileDoesNotFollowDestinationSymlink(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "memfab")
	t.Setenv("MEMFAB_ROOT", root)
	if err := EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(parent, "outside.txt")
	if err := os.WriteFile(external, []byte("unchanged"), 0o644); err != nil {
		t.Fatal(err)
	}
	destination := PausedFlag()
	if err := os.Symlink(external, destination); err != nil {
		t.Fatal(err)
	}
	if err := WritePrivateFile(destination, []byte("paused")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(external)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "unchanged" {
		t.Fatalf("external file changed to %q", got)
	}
	info, err := os.Lstat(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("destination mode=%v, want regular file", info.Mode())
	}
	assertMode(t, destination, FileMode)
}

func TestSecureRegularFilesRejectsSymlink(t *testing.T) {
	parent := t.TempDir()
	external := filepath.Join(parent, "outside.db")
	if err := os.WriteFile(external, []byte("database"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(parent, "index.db")
	if err := os.Symlink(external, link); err != nil {
		t.Fatal(err)
	}
	if err := SecureRegularFiles(link); err == nil {
		t.Fatal("SecureRegularFiles accepted a symlink")
	}
	assertMode(t, external, 0o644)
}

func TestEnsureDirsRejectsIntermediateSymlink(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "memfab")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	external := filepath.Join(parent, "external")
	if err := os.Mkdir(external, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}

	t.Setenv("MEMFAB_ROOT", root)
	if err := ensurePrivateDirectory(filepath.Join(root, "linked", "nested")); err == nil {
		t.Fatal("private directory creation accepted an intermediate symlink")
	}
	if _, err := os.Stat(filepath.Join(external, "nested")); !os.IsNotExist(err) {
		t.Fatalf("intermediate symlink created an external directory: %v", err)
	}
	assertMode(t, external, 0o755)
}

func assertMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode=%#o, want %#o", path, got, want)
	}
}
