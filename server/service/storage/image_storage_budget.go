package storage

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"
)

var (
	errImageStorageInsufficientSpace = errors.New("insufficient storage")
	errImageStorageSizeLimit         = errors.New("image exceeds configured size limit")
)

// imageStorageBudget returns the absolute on-disk byte cap for a single image
// stored in directory: the smaller of maxImageBytes and the filesystem's
// available bytes minus reservedFreeBytes, plus existingBytes. existingBytes
// is the number of bytes the caller has already written for this same
// operation (e.g. a partial upload): those blocks are already subtracted from
// the filesystem's available count, so they are added back to keep the cap
// independent of transfer progress. Callers that have written nothing pass 0.
// The statfs seam keeps the calculation host-testable.
func imageStorageBudget(
	directory string,
	statfs func(string, *unix.Statfs_t) error,
	maxImageBytes int64,
	reservedFreeBytes int64,
	existingBytes int64,
) (int64, error) {
	if maxImageBytes <= 0 || reservedFreeBytes < 0 || existingBytes < 0 {
		return 0, errImageStorageSizeLimit
	}

	var stat unix.Statfs_t
	if err := statfs(directory, &stat); err != nil {
		return 0, err
	}
	if stat.Bsize <= 0 {
		return 0, fmt.Errorf("invalid image filesystem capacity")
	}

	availableBlocks := uint64(stat.Bavail)
	blockSize := uint64(stat.Bsize)
	if availableBlocks > ^uint64(0)/blockSize {
		return 0, fmt.Errorf("available image storage overflows")
	}
	availableBytes := availableBlocks * blockSize
	if uint64(existingBytes) > ^uint64(0)-availableBytes {
		return 0, fmt.Errorf("image storage accounting overflows")
	}
	effectiveBytes := availableBytes + uint64(existingBytes)
	reserved := uint64(reservedFreeBytes)
	if effectiveBytes <= reserved {
		return 0, errImageStorageInsufficientSpace
	}

	budget := effectiveBytes - reserved
	if budget > uint64(maxImageBytes) {
		return maxImageBytes, nil
	}
	if budget > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1), nil
	}
	return int64(budget), nil
}
