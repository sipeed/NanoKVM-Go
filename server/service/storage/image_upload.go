package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// The browser still sends an 8 MiB chunk at a time, but the server streams
	// each request directly to disk instead of retaining the chunk in the Go
	// heap. Only one chunk request is admitted at a time.
	imageUploadChunkSizeBytes      int64 = 8 << 20
	imageUploadCopyBufferBytes           = 64 << 10
	imageUploadMaxConcurrentChunks       = 1
	imageUploadIDLength                  = 32
	imageUploadTempPrefix                = ".nanokvm-upload-"
	imageUploadTempSuffix                = ".part"

	// This is the product-level memory budget reserved for image upload. The
	// compile-time constants and tests below keep the bounded upload working set
	// (one chunk of page cache plus the userspace copy buffer) well below 50 MiB.
	imageUploadMemoryBudgetBytes int64 = 50 << 20
	imageUploadBoundedBytes      int64 = imageUploadMaxConcurrentChunks *
		(imageUploadChunkSizeBytes + imageUploadCopyBufferBytes)
)

const _ = uint64(imageUploadMemoryBudgetBytes - imageUploadBoundedBytes)

var (
	errImageUploadBusy          = errors.New("image upload already in progress")
	errImageUploadInvalid       = errors.New("invalid image upload metadata")
	errImageUploadTooLarge      = errors.New("image upload chunk too large")
	errImageUploadInvalidLength = errors.New("invalid image upload chunk length")
	errImageUploadOutOfOrder    = errors.New("image upload chunk out of order")

	// Capacity sentinels stay internal; handlers translate them into the stable
	// response messages below and never expose err.Error() to API clients.
	errImageUploadInsufficientSpace = errors.New("insufficient storage for image upload")
	errImageUploadSizeLimitExceeded = errors.New("image upload exceeds configured size limit")
)

const (
	// imageUploadInsufficientStorageResponseMessage must stay byte-identical to
	// downloadInsufficientStorageResponseMessage: the web client maps this exact
	// message to its localized insufficient-storage copy for both flows.
	imageUploadInsufficientStorageResponseMessage = "insufficient storage"
	imageUploadSizeLimitResponseMessage           = "image exceeds size limit"
)

type imageUploadLimits struct {
	MaxImageBytes     int64
	ReservedFreeBytes int64
}

var (
	imageUploadStatfs = unix.Statfs
	imageUploadPolicy = defaultImageUploadPolicy
)

func defaultImageUploadPolicy() imageUploadLimits {
	conf := config.GetInstance()
	return imageUploadLimits{
		MaxImageBytes:     int64(conf.Security.Upload.MaxImageBytes),
		ReservedFreeBytes: int64(conf.Security.Upload.ReservedFreeBytes),
	}
}

// imageUploadAdmission captures the capacity decision for one chunk request.
// budget is an absolute on-disk byte cap that already accounts for the bytes
// this upload has written so far, so it stays valid as the .part file grows.
// The declared total (totalChunks*chunkSize) is an upper bound on the final
// image size: the last chunk may be shorter, so admission is conservative by
// at most chunkSize-1 bytes.
type imageUploadAdmission struct {
	maxImageBytes     int64
	reservedFreeBytes int64
	budget            int64
}

func admitImageUpload(deps imageUploadDependencies, metadata imageUploadMetadata) (imageUploadAdmission, error) {
	limits := imageUploadPolicy()
	// parseImageUploadMetadata guarantees totalChunks <= math.MaxInt64/chunkSize,
	// so the declared total cannot overflow.
	declaredTotal := metadata.totalChunks * metadata.chunkSize
	if limits.MaxImageBytes <= 0 || declaredTotal > limits.MaxImageBytes {
		return imageUploadAdmission{}, errImageUploadSizeLimitExceeded
	}
	if err := ensureImageDirectory(deps.directory); err != nil {
		return imageUploadAdmission{}, err
	}
	budget, err := imageStorageBudget(deps.directory, imageUploadStatfs, limits.MaxImageBytes, limits.ReservedFreeBytes, metadata.offset)
	switch {
	case errors.Is(err, errImageStorageInsufficientSpace):
		return imageUploadAdmission{}, errImageUploadInsufficientSpace
	case errors.Is(err, errImageStorageSizeLimit):
		return imageUploadAdmission{}, errImageUploadSizeLimitExceeded
	case err != nil:
		return imageUploadAdmission{}, err
	}
	if declaredTotal > budget {
		return imageUploadAdmission{}, errImageUploadInsufficientSpace
	}
	return imageUploadAdmission{
		maxImageBytes:     limits.MaxImageBytes,
		reservedFreeBytes: limits.ReservedFreeBytes,
		budget:            budget,
	}, nil
}

func respondImageUploadError(ctx *gin.Context, rsp *proto.Response, err error) {
	switch {
	case errors.Is(err, errChecksumMismatch):
		rsp.ErrRsp(ctx, -10, errChecksumMismatch.Error())
	case errors.Is(err, errImageUploadTooLarge):
		rsp.ErrRsp(ctx, -3, errImageUploadTooLarge.Error())
	case errors.Is(err, errImageUploadInvalidLength):
		rsp.ErrRsp(ctx, -4, errImageUploadInvalidLength.Error())
	case errors.Is(err, errImageUploadOutOfOrder):
		rsp.ErrRsp(ctx, -5, errImageUploadOutOfOrder.Error())
	case errors.Is(err, errImageUploadSizeLimitExceeded):
		rsp.ErrRsp(ctx, -8, imageUploadSizeLimitResponseMessage)
	case errors.Is(err, errImageUploadInsufficientSpace):
		rsp.ErrRsp(ctx, -9, imageUploadInsufficientStorageResponseMessage)
	default:
		log.Errorf("image upload failed: %s", err)
		rsp.ErrRsp(ctx, -1, "image upload failed")
	}
}

type imageUploadMetadata struct {
	checksum    string
	uploadID    string
	filename    string
	chunkSize   int64
	chunkIndex  int64
	totalChunks int64
	offset      int64
}

func (metadata imageUploadMetadata) finalChunk() bool {
	return metadata.chunkIndex == metadata.totalChunks-1
}

type imageUploadLimiter struct {
	slots chan struct{}
}

func newImageUploadLimiter(limit int) *imageUploadLimiter {
	return &imageUploadLimiter{slots: make(chan struct{}, limit)}
}

func (limiter *imageUploadLimiter) tryAcquire() (func(), bool) {
	select {
	case limiter.slots <- struct{}{}:
		var once sync.Once
		return func() {
			once.Do(func() { <-limiter.slots })
		}, true
	default:
		return nil, false
	}
}

func (limiter *imageUploadLimiter) acquire(ctx context.Context) (func(), error) {
	select {
	case limiter.slots <- struct{}{}:
		var once sync.Once
		return func() {
			once.Do(func() { <-limiter.slots })
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type imageUploadDependencies struct {
	directory string
	limiter   *imageUploadLimiter
	syncFile  func(*os.File) error
	dropCache func(*os.File, int64, int64) error
}

var defaultImageUploadDependencies = imageUploadDependencies{
	directory: imageDirectory,
	limiter:   newImageUploadLimiter(imageUploadMaxConcurrentChunks),
	syncFile: func(file *os.File) error {
		return file.Sync()
	},
	dropCache: func(file *os.File, offset, length int64) error {
		return unix.Fadvise(int(file.Fd()), offset, length, unix.FADV_DONTNEED)
	},
}

func (s *Service) UploadImage(ctx *gin.Context) {
	metadata, err := parseImageUploadMetadata(ctx)
	if err != nil {
		var rsp proto.Response
		rsp.ErrRsp(ctx, -2, errImageUploadInvalid.Error())
		return
	}
	finish, err := s.beginUploadStorage(metadata)
	if err != nil {
		var rsp proto.Response
		rsp.ErrRsp(ctx, -1, err.Error())
		return
	}
	success := false
	defer func() { finish(success) }()
	success = uploadImageChunk(ctx, defaultImageUploadDependencies)
}

func (s *Service) CancelImageUpload(ctx *gin.Context) {
	release, err := acquireStorageAccess(true)
	if err != nil {
		var rsp proto.Response
		rsp.ErrRsp(ctx, -1, err.Error())
		return
	}
	defer release()
	if uploadID := cancelImageUpload(ctx, defaultImageUploadDependencies); uploadID != "" {
		s.releaseUploadStorage(uploadID)
	}
}

func uploadImageChunk(ctx *gin.Context, deps imageUploadDependencies) (success bool) {
	var rsp proto.Response

	metadata, err := parseImageUploadMetadata(ctx)
	if err != nil {
		rsp.ErrRsp(ctx, -2, errImageUploadInvalid.Error())
		return
	}

	if ctx.Request == nil || ctx.Request.Body == nil {
		rsp.ErrRsp(ctx, -3, errImageUploadInvalidLength.Error())
		return
	}
	if ctx.Request.ContentLength > imageUploadChunkSizeBytes {
		rsp.ErrRsp(ctx, -3, errImageUploadTooLarge.Error())
		return
	}

	release, ok := deps.limiter.tryAcquire()
	if !ok {
		rsp.ErrRsp(ctx, -7, errImageUploadBusy.Error())
		return
	}
	defer release()

	admission, err := admitImageUpload(deps, metadata)
	if err != nil {
		respondImageUploadError(ctx, &rsp, err)
		return
	}

	written, err := receiveImageUploadChunk(ctx, deps, metadata, admission)
	if err != nil {
		respondImageUploadError(ctx, &rsp, err)
		return
	}

	log.Infof(
		"successfully uploaded chunk %d/%d of %s (%d bytes)",
		metadata.chunkIndex+1,
		metadata.totalChunks,
		metadata.filename,
		written,
	)
	rsp.OkRspWithData(ctx, gin.H{
		"status":      "chunk_uploaded",
		"filename":    metadata.filename,
		"chunkIndex":  metadata.chunkIndex,
		"totalChunks": metadata.totalChunks,
		"written":     written,
	})
	return true
}

func parseImageUploadMetadata(ctx *gin.Context) (imageUploadMetadata, error) {
	if ctx == nil || ctx.Request == nil {
		return imageUploadMetadata{}, errImageUploadInvalid
	}

	uploadID := strings.TrimSpace(ctx.Query("uploadId"))
	if !validImageUploadID(uploadID) {
		return imageUploadMetadata{}, errImageUploadInvalid
	}

	filename := sanitizeFileName(strings.TrimSpace(ctx.Query("filename")))
	if filename == "" || filename == "." || filename == ".." || filepath.Base(filename) != filename {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	if strings.HasPrefix(filename, ".") {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	if len(filename) > 255 {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".iso" && ext != ".img" {
		return imageUploadMetadata{}, errImageUploadInvalid
	}

	chunkSize, err := strconv.ParseInt(strings.TrimSpace(ctx.Query("chunkSize")), 10, 64)
	if err != nil || chunkSize != imageUploadChunkSizeBytes {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	chunkIndex, err := strconv.ParseInt(strings.TrimSpace(ctx.Query("chunkIndex")), 10, 64)
	if err != nil || chunkIndex < 0 {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	totalChunks, err := strconv.ParseInt(strings.TrimSpace(ctx.Query("totalChunks")), 10, 64)
	if err != nil || totalChunks <= 0 || chunkIndex >= totalChunks {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	if chunkIndex > math.MaxInt64/chunkSize || totalChunks > math.MaxInt64/chunkSize {
		return imageUploadMetadata{}, errImageUploadInvalid
	}

	checksum := strings.TrimSpace(ctx.Query("sha256"))
	if !validChecksum(checksum) {
		return imageUploadMetadata{}, errImageUploadInvalid
	}
	return imageUploadMetadata{
		checksum:    checksum,
		uploadID:    uploadID,
		filename:    filename,
		chunkSize:   chunkSize,
		chunkIndex:  chunkIndex,
		totalChunks: totalChunks,
		offset:      chunkIndex * chunkSize,
	}, nil
}

func validImageUploadID(uploadID string) bool {
	if len(uploadID) != imageUploadIDLength {
		return false
	}
	for _, char := range uploadID {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func imageUploadTemporaryPath(directory, uploadID string) string {
	return filepath.Join(directory, imageUploadTempPrefix+uploadID+imageUploadTempSuffix)
}

func receiveImageUploadChunk(
	ctx *gin.Context,
	deps imageUploadDependencies,
	metadata imageUploadMetadata,
	admission imageUploadAdmission,
) (written int64, err error) {
	if err := ensureImageDirectory(deps.directory); err != nil {
		return 0, err
	}
	// Free space is a moving target (concurrent downloads and other writers
	// share /data), so re-read the cap immediately before writing and honor
	// the tighter of the admission and current values.
	budget := admission.budget
	current, err := imageStorageBudget(deps.directory, imageUploadStatfs, admission.maxImageBytes, admission.reservedFreeBytes, metadata.offset)
	switch {
	case errors.Is(err, errImageStorageInsufficientSpace):
		return 0, errImageUploadInsufficientSpace
	case errors.Is(err, errImageStorageSizeLimit):
		return 0, errImageUploadSizeLimitExceeded
	case err != nil:
		return 0, err
	}
	if current < budget {
		budget = current
	}
	if budget <= metadata.offset {
		return 0, errImageUploadInsufficientSpace
	}
	targetPath := filepath.Join(deps.directory, metadata.filename)
	temporaryPath := imageUploadTemporaryPath(deps.directory, metadata.uploadID)
	file, err := openImageUploadTarget(temporaryPath, metadata)
	if err != nil {
		return 0, err
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	rollback := func() {
		if truncateErr := file.Truncate(metadata.offset); truncateErr != nil {
			log.Warnf("failed to roll back image upload chunk: %s", truncateErr)
		}
		if metadata.offset == 0 {
			if removeErr := os.Remove(temporaryPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				log.Warnf("failed to remove empty image upload: %s", removeErr)
			}
		}
	}

	limitedBody := http.MaxBytesReader(ctx.Writer, ctx.Request.Body, imageUploadChunkSizeBytes)
	defer limitedBody.Close()
	buffer := make([]byte, imageUploadCopyBufferBytes)
	writer := &imageUploadBoundedWriter{
		writer:        file,
		total:         metadata.offset,
		maxImageBytes: admission.maxImageBytes,
		budget:        budget,
	}
	written, err = copyImageUploadChunk(writer, limitedBody, buffer)
	if err != nil {
		rollback()
		return written, mapImageUploadCopyError(err)
	}

	if written <= 0 || written > metadata.chunkSize || (!metadata.finalChunk() && written != metadata.chunkSize) {
		rollback()
		return written, errImageUploadInvalidLength
	}

	if err := deps.syncFile(file); err != nil {
		rollback()
		return written, fmt.Errorf("sync image upload chunk: %w", err)
	}
	if err := deps.dropCache(file, metadata.offset, written); err != nil {
		// FADV_DONTNEED is advisory and may not be implemented by every backing
		// filesystem. The data is already durable and clean after Sync, so a
		// failure here must not corrupt or discard an otherwise valid upload.
		log.Warnf("failed to release image upload page cache: %s", err)
	}
	if metadata.finalChunk() {
		if err := ctx.Request.Context().Err(); err != nil {
			rollback()
			return written, err
		}
		if err := verifyImageChecksum(ctx.Request.Context(), temporaryPath, metadata.checksum); err != nil {
			rollback()
			return written, err
		}
		if err := publishImageUpload(temporaryPath, targetPath); err != nil {
			rollback()
			return written, fmt.Errorf("publish image upload: %w", err)
		}
	}

	return written, nil
}

// imageUploadBoundedWriter caps the absolute on-disk size of an upload at both
// the configured single-image maximum and the current free-space budget, so a
// dishonest chunk plan or a shrinking filesystem cannot write past either line.
type imageUploadBoundedWriter struct {
	writer        io.Writer
	total         int64
	maxImageBytes int64
	budget        int64
}

func (w *imageUploadBoundedWriter) Write(data []byte) (int, error) {
	if w.total > w.maxImageBytes || int64(len(data)) > w.maxImageBytes-w.total {
		return 0, errImageUploadSizeLimitExceeded
	}
	if w.total > w.budget || int64(len(data)) > w.budget-w.total {
		return 0, errImageUploadInsufficientSpace
	}

	n, err := w.writer.Write(data)
	w.total += int64(n)
	return n, err
}

func mapImageUploadCopyError(err error) error {
	var maxBytesError *http.MaxBytesError
	switch {
	case errors.As(err, &maxBytesError):
		return errImageUploadTooLarge
	case errors.Is(err, errImageUploadInsufficientSpace):
		return errImageUploadInsufficientSpace
	case errors.Is(err, errImageUploadSizeLimitExceeded):
		return errImageUploadSizeLimitExceeded
	case errors.Is(err, unix.ENOSPC):
		return errImageUploadInsufficientSpace
	}
	return err
}

func publishImageUpload(temporaryPath, targetPath string) error {
	mediumOperationMu.Lock()
	defer mediumOperationMu.Unlock()
	if err := guardMediaDelete(targetPath); err != nil {
		return err
	}
	info, err := os.Lstat(targetPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("image upload destination is not a regular file")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return os.Rename(temporaryPath, targetPath)
}

func copyImageUploadChunk(writer io.Writer, reader io.Reader, buffer []byte) (int64, error) {
	if len(buffer) == 0 {
		return 0, errors.New("empty image upload copy buffer")
	}

	var written int64
	for {
		read, readErr := reader.Read(buffer)
		if read > 0 {
			copied, writeErr := writer.Write(buffer[:read])
			written += int64(copied)
			if writeErr != nil {
				return written, writeErr
			}
			if copied != read {
				return written, io.ErrShortWrite
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return written, nil
			}
			return written, readErr
		}
	}
}

func openImageUploadTarget(path string, metadata imageUploadMetadata) (*os.File, error) {
	flags := unix.O_WRONLY | unix.O_CLOEXEC | unix.O_NOFOLLOW | unix.O_NONBLOCK
	if metadata.chunkIndex == 0 {
		flags |= unix.O_CREAT | unix.O_EXCL
	}

	fd, err := unix.Open(path, flags, 0o600)
	if err != nil {
		if metadata.chunkIndex > 0 && errors.Is(err, unix.ENOENT) {
			return nil, errImageUploadOutOfOrder
		}
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("create image upload file handle")
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return nil, errors.New("image upload target is not a regular file")
	}
	if info.Size() != metadata.offset {
		_ = file.Close()
		return nil, errImageUploadOutOfOrder
	}
	if _, err := file.Seek(metadata.offset, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}

	return file, nil
}

type cancelImageUploadRequest struct {
	UploadID string `json:"uploadId"`
}

func cancelImageUpload(ctx *gin.Context, deps imageUploadDependencies) (uploadID string) {
	var req cancelImageUploadRequest
	var rsp proto.Response
	if ctx.Request == nil || ctx.Request.Body == nil || ctx.ShouldBindJSON(&req) != nil {
		rsp.ErrRsp(ctx, -2, errImageUploadInvalid.Error())
		return
	}
	if !validImageUploadID(req.UploadID) {
		rsp.ErrRsp(ctx, -2, errImageUploadInvalid.Error())
		return
	}

	release, err := deps.limiter.acquire(ctx.Request.Context())
	if err != nil {
		return
	}
	defer release()

	if err := ensureImageDirectory(deps.directory); err != nil {
		log.Errorf("failed to prepare image upload directory for cancellation: %s", err)
		rsp.ErrRsp(ctx, -1, "failed to cancel image upload")
		return
	}

	temporaryPath := imageUploadTemporaryPath(deps.directory, req.UploadID)
	status := "idle"
	info, err := os.Lstat(temporaryPath)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		log.Errorf("failed to inspect image upload for cancellation: %s", err)
		rsp.ErrRsp(ctx, -1, "failed to cancel image upload")
		return
	case info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular():
		log.Errorf("refusing to cancel invalid image upload path %s", temporaryPath)
		rsp.ErrRsp(ctx, -1, "failed to cancel image upload")
		return
	default:
		if err := os.Remove(temporaryPath); err != nil {
			log.Errorf("failed to remove incomplete image upload: %s", err)
			rsp.ErrRsp(ctx, -1, "failed to cancel image upload")
			return
		}
		status = "cancelled"
	}

	rsp.OkRspWithData(ctx, gin.H{
		"status":   status,
		"uploadId": req.UploadID,
	})
	return req.UploadID
}
