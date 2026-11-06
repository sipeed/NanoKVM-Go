package updateverify

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

type artifactIdentity struct {
	device      uint64
	inode       uint64
	links       uint64
	mode        uint32
	size        int64
	modifiedSec int64
	modifiedNS  int64
	changedSec  int64
	changedNS   int64
}

type artifactSnapshot struct {
	path     string
	label    string
	file     *os.File
	data     []byte
	identity artifactIdentity
}

func openArtifactSnapshot(path, label string, expectedSize, maxSize int64) (*artifactSnapshot, error) {
	pathStat, err := lstatArtifact(path)
	if err != nil {
		return nil, ioError("inspect "+label, err)
	}
	if pathStat.Mode&unix.S_IFMT != unix.S_IFREG {
		return nil, policyError(label+" is not a regular file", nil)
	}

	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, ioError("open "+label, err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, ioError("open "+label, errors.New("failed to create file handle"))
	}

	closeWithError := func(verifyErr error) (*artifactSnapshot, error) {
		if closeErr := file.Close(); closeErr != nil && verifyErr == nil {
			verifyErr = ioError("close "+label, closeErr)
		}
		return nil, verifyErr
	}

	var openedStat unix.Stat_t
	if err := unix.Fstat(fd, &openedStat); err != nil {
		return closeWithError(ioError("stat "+label, err))
	}
	if openedStat.Mode&unix.S_IFMT != unix.S_IFREG {
		return closeWithError(policyError(label+" is not a regular file", nil))
	}
	pathIdentity := identityFromStat(pathStat)
	openedIdentity := identityFromStat(openedStat)
	if pathIdentity != openedIdentity {
		return closeWithError(policyError(label+" changed while it was opened", nil))
	}
	if openedStat.Size <= 0 {
		return closeWithError(policyError(label+" is empty", nil))
	}
	if expectedSize > 0 && openedStat.Size != expectedSize {
		return closeWithError(policyError(label+" size does not match metadata", nil))
	}
	if maxSize > 0 && openedStat.Size > maxSize {
		return closeWithError(policyError(label+" exceeds the verification size limit", nil))
	}
	if uint64(openedStat.Size) > uint64(^uint(0)>>1) {
		return closeWithError(policyError(label+" is too large for this platform", nil))
	}

	data, err := unix.Mmap(fd, 0, int(openedStat.Size), unix.PROT_READ, unix.MAP_PRIVATE)
	if err != nil {
		return closeWithError(ioError("map "+label, err))
	}
	snapshot := &artifactSnapshot{
		path:     path,
		label:    label,
		file:     file,
		data:     data,
		identity: openedIdentity,
	}
	if err := snapshot.VerifyStable(); err != nil {
		_ = snapshot.Close()
		return nil, err
	}
	return snapshot, nil
}

func (snapshot *artifactSnapshot) Bytes() []byte {
	return snapshot.data
}

func (snapshot *artifactSnapshot) File() *os.File {
	return snapshot.file
}

func (snapshot *artifactSnapshot) VerifyStable() error {
	if snapshot == nil || snapshot.file == nil {
		return ioError("verify artifact stability", errors.New("artifact snapshot is closed"))
	}
	var descriptorStat unix.Stat_t
	if err := unix.Fstat(int(snapshot.file.Fd()), &descriptorStat); err != nil {
		return ioError("restat "+snapshot.label, err)
	}
	pathStat, err := lstatArtifact(snapshot.path)
	if err != nil {
		return policyError(snapshot.label+" changed while it was verified", err)
	}
	if descriptorStat.Mode&unix.S_IFMT != unix.S_IFREG || pathStat.Mode&unix.S_IFMT != unix.S_IFREG {
		return policyError(snapshot.label+" changed while it was verified", nil)
	}
	if identityFromStat(descriptorStat) != snapshot.identity || identityFromStat(pathStat) != snapshot.identity {
		return policyError(snapshot.label+" changed while it was verified", nil)
	}
	return nil
}

func (snapshot *artifactSnapshot) Close() error {
	if snapshot == nil {
		return nil
	}
	var result error
	if snapshot.data != nil {
		if err := unix.Munmap(snapshot.data); err != nil {
			result = ioError("unmap "+snapshot.label, err)
		}
		snapshot.data = nil
	}
	if snapshot.file != nil {
		if err := snapshot.file.Close(); err != nil && result == nil {
			result = ioError("close "+snapshot.label, err)
		}
		snapshot.file = nil
	}
	return result
}

func readBoundedRegularFile(path, label string, maxSize int64) ([]byte, error) {
	snapshot, err := openArtifactSnapshot(path, label, 0, maxSize)
	if err != nil {
		return nil, err
	}
	data := append([]byte(nil), snapshot.Bytes()...)
	stableErr := snapshot.VerifyStable()
	closeErr := snapshot.Close()
	if stableErr != nil {
		return nil, stableErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return data, nil
}

func lstatArtifact(path string) (unix.Stat_t, error) {
	var stat unix.Stat_t
	if err := unix.Lstat(path, &stat); err != nil {
		return unix.Stat_t{}, err
	}
	return stat, nil
}

func identityFromStat(stat unix.Stat_t) artifactIdentity {
	return artifactIdentity{
		device:      uint64(stat.Dev),
		inode:       uint64(stat.Ino),
		links:       uint64(stat.Nlink),
		mode:        uint32(stat.Mode),
		size:        stat.Size,
		modifiedSec: int64(stat.Mtim.Sec),
		modifiedNS:  int64(stat.Mtim.Nsec),
		changedSec:  int64(stat.Ctim.Sec),
		changedNS:   int64(stat.Ctim.Nsec),
	}
}
