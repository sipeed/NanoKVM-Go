package diskfs

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

type boundedUploadReader struct {
	remaining int64
	maxRead   int
}

func (r *boundedUploadReader) Read(buffer []byte) (int, error) {
	if len(buffer) > r.maxRead {
		r.maxRead = len(buffer)
	}
	if r.remaining == 0 {
		return 0, io.EOF
	}
	n := min(int64(len(buffer)), r.remaining)
	clear(buffer[:n])
	r.remaining -= n
	return int(n), nil
}

func TestUploadStreamsAnEightMiBChunkWithBoundedReads(t *testing.T) {
	root := t.TempDir()
	upload, err := beginUpload(root, "large.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer upload.Close()
	source := &boundedUploadReader{remaining: 8 << 20}
	if written, err := upload.WriteChunk(source, 8<<20); err != nil || written != 8<<20 {
		t.Fatalf("write chunk: %d, %v", written, err)
	}
	if source.maxRead > 64<<10 {
		t.Fatalf("read buffer grew to %d bytes", source.maxRead)
	}
	if _, err := os.Stat(filepath.Join(root, "large.bin")); !os.IsNotExist(err) {
		t.Fatal("partial upload was published")
	}
	if err := upload.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := upload.Close(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(root, "large.bin")); err != nil || info.Size() != 8<<20 {
		t.Fatalf("committed file: %v", err)
	}
}

func TestInvalidChunkCleanupPreservesExistingDestination(t *testing.T) {
	root := t.TempDir()
	upload, err := beginUpload(root, "keep.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := upload.WriteChunk(&boundedUploadReader{remaining: 5}, 4); err == nil {
		t.Fatal("oversized chunk accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "keep.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := upload.Commit(); err == nil {
		t.Fatal("upload overwrote a newly created destination")
	}
	if err := upload.Close(); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != "keep.txt" {
		t.Fatalf("temporary file remains: %v, %v", entries, err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "keep.txt"))
	if string(got) != "existing" {
		t.Fatal("cleanup changed the existing file")
	}
}
