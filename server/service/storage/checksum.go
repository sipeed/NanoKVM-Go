package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

var errChecksumMismatch = errors.New("sha256 mismatch")

func validChecksum(sum string) bool {
	if sum == "" {
		return true
	}
	decoded, err := hex.DecodeString(sum)
	return err == nil && len(decoded) == sha256.Size
}

// Hash only when requested, with bounded memory and cancellation between
// reads. Never publish a temporary whose expected checksum does not match.
func verifyImageChecksum(ctx context.Context, path, expected string) error {
	if expected == "" {
		return nil
	}
	file, err := os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	buffer := make([]byte, 64<<10)
	var offset int64
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := file.Read(buffer)
		if n > 0 {
			_, _ = hash.Write(buffer[:n])
			_ = unix.Fadvise(int(file.Fd()), offset, int64(n), unix.FADV_DONTNEED)
			offset += int64(n)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected) {
		return errChecksumMismatch
	}
	return nil
}
