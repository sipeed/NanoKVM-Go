package diskfs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Upload keeps an unpublished file open while its medium retains ownership.
// The caller serializes chunks and closes it before closing the medium.
type Upload struct {
	file    *os.File
	root    string
	rel     string
	target  string
	temp    string
	written int64
}

func beginUpload(root, rel string) (*Upload, error) {
	target, err := ResolveInside(root, rel)
	if err != nil {
		return nil, err
	}
	if _, err := os.Lstat(target); err == nil {
		return nil, fmt.Errorf("%s already exists", rel)
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	temp := filepath.Join(filepath.Dir(target), UploadPrefix+randomSuffix())
	file, err := os.OpenFile(temp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return nil, err
	}
	return &Upload{file: file, root: root, rel: rel, target: target, temp: temp}, nil
}

// WriteChunk never buffers a whole chunk. Sync bounds dirty data to the
// current chunk, and DONTNEED asks the kernel to release its clean file cache.
func (u *Upload) WriteChunk(r io.Reader, size int64) (int64, error) {
	if u.file == nil || size < 0 {
		return 0, errors.New("invalid upload chunk")
	}
	// Hide File.ReadFrom so io.CopyBuffer always uses this bounded buffer.
	written, err := io.CopyBuffer(struct{ io.Writer }{u.file}, io.LimitReader(r, size+1), make([]byte, 64<<10))
	if err == nil && written != size {
		err = fmt.Errorf("invalid upload chunk length: got %d, want %d", written, size)
	}
	if err == nil {
		err = u.file.Sync()
	}
	if err != nil {
		return written, err
	}
	if size > 0 {
		if err := unix.Fadvise(int(u.file.Fd()), u.written, size, unix.FADV_DONTNEED); err != nil {
			log.Warnf("failed to release file upload page cache: %s", err)
		}
	}
	u.written += written
	return written, nil
}

func (u *Upload) Commit() error {
	if u.file == nil {
		return errors.New("upload is closed")
	}
	// Recheck the path and destination before publishing. App-level upload
	// reservations also protect this file and its parents between chunks.
	target, err := ResolveInside(u.root, u.rel)
	if err != nil {
		return err
	}
	if target != u.target {
		return errors.New("upload destination changed")
	}
	if _, err := os.Lstat(target); err == nil {
		return fmt.Errorf("%s already exists", u.rel)
	} else if !os.IsNotExist(err) {
		return err
	}
	err = u.file.Close()
	u.file = nil
	if err != nil {
		return err
	}
	return os.Rename(u.temp, target)
}

func (u *Upload) Close() error {
	var closeErr error
	if u.file != nil {
		closeErr = u.file.Close()
		u.file = nil
	}
	err := os.Remove(u.temp)
	if os.IsNotExist(err) {
		err = nil
	}
	return errors.Join(closeErr, err)
}
