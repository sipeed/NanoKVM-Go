package application

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const offlineApplicationUploadTimeout = 30 * time.Minute

var (
	errOfflineUploadInvalid     = errors.New("invalid offline application upload")
	errOfflineUploadInterrupted = errors.New("offline application upload interrupted")
	errOfflineUploadNoSpace     = errors.New("insufficient space for offline application upload")
)

// offlineApplicationLease keeps the HTTP upload independent from the concrete
// lock implementation while preserving the lock handoff contract. Production
// adapters below always pass the original *applicationUpdateLease to the
// shared worker.
type offlineApplicationLease interface {
	Release()
}

type offlineApplicationUpdateDependencies struct {
	baseDir        string
	uploadTimeout  time.Duration
	budget         func(string) (int64, error)
	beginAdmission func(UpdateSource, string, UpdatePhase) (offlineApplicationLease, ApplicationUpdateStatus, error)
	startOperation func(offlineApplicationLease, ApplicationUpdateStatus, string, string) error
	setVersion     func(string, string) error
	advance        func(string, UpdatePhase, int) error
	fail           func(string, UpdateErrorCode) error
	current        func() (ApplicationUpdateStatus, error)
}

var defaultOfflineApplicationUpdateDependencies = offlineApplicationUpdateDependencies{
	baseDir:       TempDir,
	uploadTimeout: offlineApplicationUploadTimeout,
	budget:        applicationDownloadBudget,
	beginAdmission: func(source UpdateSource, version string, phase UpdatePhase) (offlineApplicationLease, ApplicationUpdateStatus, error) {
		return beginApplicationUpdateAdmission(source, version, phase)
	},
	startOperation: func(lease offlineApplicationLease, status ApplicationUpdateStatus, archivePath, workDir string) error {
		applicationLease, ok := lease.(*applicationUpdateLease)
		if !ok {
			return fmt.Errorf("invalid application update lease")
		}
		return startOfflineApplicationOperation(applicationLease, status, archivePath, workDir)
	},
	setVersion: setApplicationUpdateVersion,
	advance:    advanceApplicationUpdate,
	fail:       failApplicationUpdate,
	current:    currentApplicationUpdateStatus,
}

// UpdateOffline accepts exactly one multipart file and returns as soon as a
// request-independent worker owns the fully persisted archive. Browser names
// never cross the archive trust boundary.
func (s *Service) UpdateOffline(c *gin.Context) {
	updateOfflineApplication(c, defaultOfflineApplicationUpdateDependencies)
}

func updateOfflineApplication(c *gin.Context, deps offlineApplicationUpdateDependencies) {
	var rsp proto.Response

	// Admission deliberately happens before MultipartReader or any other body
	// operation. A busy request therefore cannot consume or stage upload bytes.
	lease, status, err := deps.beginAdmission(UpdateSourceOffline, "", UpdatePhaseUploading)
	if err != nil {
		if lease != nil {
			lease.Release()
		}
		code := UpdateErrorInterrupted
		if errors.Is(err, errApplicationUpdateBusy) {
			code = UpdateErrorBusy
		}
		log.Warnf("offline application update admission failed: %v", err)
		if status.OperationID == "" {
			status = publicFailedUpdateStatus(code)
		}
		rsp.ErrRspWithData(c, -1, string(code), status)
		return
	}

	owned := true
	workDir := ""
	defer func() {
		if !owned {
			return
		}
		if workDir != "" {
			if removeErr := os.RemoveAll(workDir); removeErr != nil {
				log.Warnf("failed to remove offline update staging: %v", removeErr)
			}
		}
		lease.Release()
	}()

	fail := func(code UpdateErrorCode, cause error) {
		if cause != nil {
			log.Warnf("offline application upload failed (%s): %v", code, cause)
		}
		status = failedOfflineApplicationStatus(deps, status, code)
		rsp.ErrRspWithData(c, -1, string(code), status)
	}

	if err := ensureApplicationCacheRoot(deps.baseDir); err != nil {
		fail(classifyOfflineStorageError(err), err)
		return
	}
	workDir, err = os.MkdirTemp(deps.baseDir, ".application-update-offline-*")
	if err != nil {
		fail(classifyOfflineStorageError(err), err)
		return
	}
	if err := os.Chmod(workDir, 0o700); err != nil {
		fail(UpdateErrorInterrupted, err)
		return
	}

	budget, err := deps.budget(workDir)
	if err != nil || budget <= 0 {
		if err == nil {
			err = errOfflineUploadNoSpace
		}
		fail(UpdateErrorNoSpace, err)
		return
	}

	deadline := armOfflineUploadDeadline(c.Request.Body, deps.uploadTimeout)
	archivePath, version, err := receiveOfflineApplicationUpload(
		c.Request,
		workDir,
		budget,
		newOfflineApplicationUploadProgress(c.Request.ContentLength, status.OperationID, deps.advance),
	)
	deadline.Stop()
	if err != nil {
		fail(classifyOfflineUploadError(err), err)
		return
	}

	if err := deps.setVersion(status.OperationID, version); err != nil {
		fail(UpdateErrorInterrupted, err)
		return
	}
	if err := deps.advance(status.OperationID, UpdatePhaseUploading, 100); err != nil {
		fail(UpdateErrorInterrupted, err)
		return
	}
	status, err = deps.current()
	if err != nil {
		fail(UpdateErrorInterrupted, err)
		return
	}

	// A nil return transfers both the lease and the whole staging directory to
	// the controlled worker. From this point request cancellation must not stop
	// verification, cache publication, or installer startup.
	if err := deps.startOperation(lease, status, archivePath, workDir); err != nil {
		fail(UpdateErrorInstallStartFailed, err)
		return
	}
	owned = false
	rsp.OkRspWithData(c, status)
}

func failedOfflineApplicationStatus(
	deps offlineApplicationUpdateDependencies,
	status ApplicationUpdateStatus,
	code UpdateErrorCode,
) ApplicationUpdateStatus {
	if status.OperationID == "" {
		return status
	}
	if err := deps.fail(status.OperationID, code); err != nil {
		log.Warnf("failed to persist offline application update failure: %v", err)
	} else if current, err := deps.current(); err == nil {
		return current
	}
	status.Phase = UpdatePhaseFailed
	status.ErrorCode = code
	return status
}

func classifyOfflineStorageError(err error) UpdateErrorCode {
	if errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT) ||
		errors.Is(err, errOfflineUploadNoSpace) {
		return UpdateErrorNoSpace
	}
	return UpdateErrorInterrupted
}

func classifyOfflineUploadError(err error) UpdateErrorCode {
	switch {
	case errors.Is(err, errOfflineUploadNoSpace), errors.Is(err, unix.ENOSPC), errors.Is(err, unix.EDQUOT):
		return UpdateErrorNoSpace
	case errors.Is(err, errOfflineUploadInterrupted):
		return UpdateErrorUploadInterrupted
	default:
		return UpdateErrorInvalidArchive
	}
}

// receiveOfflineApplicationUpload applies a conservative Content-Length
// preflight and an independent writer-side byte budget. Content-Length includes
// multipart framing and is never trusted as the number of persisted bytes.
func receiveOfflineApplicationUpload(
	request *http.Request,
	workDir string,
	budget int64,
	progress func(int64) error,
) (archivePath, version string, err error) {
	if budget <= 0 {
		return "", "", errOfflineUploadNoSpace
	}
	if request.ContentLength > budget {
		return "", "", errOfflineUploadNoSpace
	}

	reader, err := request.MultipartReader()
	if err != nil {
		return "", "", fmt.Errorf("%w: malformed multipart envelope", errOfflineUploadInvalid)
	}
	part, err := reader.NextPart()
	if errors.Is(err, io.EOF) {
		return "", "", fmt.Errorf("%w: missing file part", errOfflineUploadInvalid)
	}
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", errOfflineUploadInterrupted, err)
	}
	defer part.Close()
	if part.FormName() != "file" || part.FileName() == "" {
		return "", "", fmt.Errorf("%w: expected one file part", errOfflineUploadInvalid)
	}

	file, err := os.CreateTemp(workDir, ".offline-upload-*.tar.gz")
	if err != nil {
		return "", "", err
	}
	uploadPath := file.Name()
	keepArchive := false
	closed := false
	defer func() {
		if !closed {
			_ = file.Close()
		}
		if !keepArchive {
			_ = os.Remove(uploadPath)
		}
	}()
	if err := file.Chmod(0o600); err != nil {
		return "", "", err
	}

	writer := &offlineApplicationUploadWriter{
		writer:    file,
		remaining: budget,
		progress:  progress,
	}
	if err := copyOfflineApplicationUpload(writer, part); err != nil {
		return "", "", err
	}
	extra, err := reader.NextPart()
	if err == nil {
		_ = extra.Close()
		return "", "", fmt.Errorf("%w: unexpected multipart part", errOfflineUploadInvalid)
	}
	if !errors.Is(err, io.EOF) {
		return "", "", fmt.Errorf("%w: %v", errOfflineUploadInterrupted, err)
	}
	if err := file.Sync(); err != nil {
		return "", "", err
	}
	if err := file.Close(); err != nil {
		return "", "", err
	}
	closed = true

	version, err = discoverOfflineApplicationArchiveVersion(uploadPath)
	if err != nil {
		return "", "", fmt.Errorf("%w: %v", errOfflineUploadInvalid, err)
	}
	keepArchive = true
	archivePath = uploadPath
	return archivePath, version, nil
}

func copyOfflineApplicationUpload(writer io.Writer, reader io.Reader) error {
	if _, err := io.Copy(writer, reader); err != nil {
		// Preserve real filesystem quota/space errors so the API can report the
		// stable no_space code. Other copy failures are client/upload interrupts.
		if errors.Is(err, errOfflineUploadNoSpace) || errors.Is(err, unix.ENOSPC) || errors.Is(err, unix.EDQUOT) {
			return err
		}
		return fmt.Errorf("%w: %v", errOfflineUploadInterrupted, err)
	}
	return nil
}

type offlineApplicationUploadWriter struct {
	writer    io.Writer
	remaining int64
	written   int64
	progress  func(int64) error
}

func (writer *offlineApplicationUploadWriter) Write(data []byte) (int, error) {
	if writer.remaining <= 0 {
		return 0, errOfflineUploadNoSpace
	}
	toWrite := data
	exceeded := false
	if int64(len(toWrite)) > writer.remaining {
		toWrite = toWrite[:writer.remaining]
		exceeded = true
	}
	written, err := writer.writer.Write(toWrite)
	writer.remaining -= int64(written)
	writer.written += int64(written)
	if err != nil {
		return written, err
	}
	if written != len(toWrite) {
		return written, io.ErrShortWrite
	}
	if writer.progress != nil {
		if err := writer.progress(writer.written); err != nil {
			return written, err
		}
	}
	if exceeded {
		return written, errOfflineUploadNoSpace
	}
	return written, nil
}

func newOfflineApplicationUploadProgress(
	contentLength int64,
	operationID string,
	advance func(string, UpdatePhase, int) error,
) func(int64) error {
	lastProgress := 0
	return func(written int64) error {
		if contentLength <= 0 || written <= 0 {
			return nil
		}
		progress := int(float64(written) * 100 / float64(contentLength))
		if progress > 99 {
			progress = 99
		}
		if progress <= lastProgress || (progress < 99 && progress-lastProgress < 5) {
			return nil
		}
		if err := advance(operationID, UpdatePhaseUploading, progress); err != nil {
			return err
		}
		lastProgress = progress
		return nil
	}
}

// discoverOfflineApplicationArchiveVersion returns only a candidate version.
// The shared bundle verifier remains authoritative and must confirm it against
// both signed package metadata files before installation.
func discoverOfflineApplicationArchiveVersion(archivePath string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	closed := false
	defer func() {
		if !closed {
			_ = gzipReader.Close()
		}
	}()

	envelope := &io.LimitedReader{R: gzipReader, N: utils.DefaultTarMaxEnvelopeBytes + 1}
	reader := tar.NewReader(envelope)
	seen := make(map[string]struct{})
	root := ""
	rootDirectorySeen := false
	var entries int64
	var expanded int64
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		entries++
		if entries > utils.DefaultTarMaxEntries {
			return "", fmt.Errorf("application archive entry count exceeds the safety limit")
		}
		if header.Size < 0 || header.Size > utils.DefaultTarMaxFileBytes ||
			header.Size > utils.DefaultTarMaxExpandedBytes-expanded {
			return "", fmt.Errorf("application archive entry exceeds the safety limit")
		}
		expanded += header.Size

		isDirectory := header.Typeflag == tar.TypeDir
		if !isDirectory && header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			return "", fmt.Errorf("application archive contains an unsupported entry")
		}
		name, err := canonicalApplicationArchivePath(header.Name, isDirectory)
		if err != nil {
			return "", err
		}
		if _, duplicate := seen[name]; duplicate {
			return "", fmt.Errorf("application archive contains a duplicate path")
		}
		seen[name] = struct{}{}

		topLevel := name
		if index := strings.IndexByte(name, '/'); index >= 0 {
			topLevel = name[:index]
		}
		if root == "" {
			root = topLevel
		} else if root != topLevel {
			return "", fmt.Errorf("application archive contains multiple top-level paths")
		}
		if name == root && isDirectory {
			rootDirectorySeen = true
		}
	}
	if root == "" || !rootDirectorySeen || !strings.HasPrefix(root, updateCacheDirPrefix) {
		return "", fmt.Errorf("application archive has no canonical top-level directory")
	}
	version := strings.TrimPrefix(root, updateCacheDirPrefix)
	if validateLatestVersion(version) != nil || root != updateCacheDirName(version) {
		return "", fmt.Errorf("application archive top-level version is not canonical")
	}
	if _, err := io.Copy(io.Discard, envelope); err != nil {
		return "", err
	}
	if envelope.N == 0 {
		return "", fmt.Errorf("application archive exceeds the stream size limit")
	}
	if err := gzipReader.Close(); err != nil {
		return "", err
	}
	closed = true
	return version, nil
}

type offlineUploadDeadline struct {
	mu    sync.Mutex
	done  bool
	timer *time.Timer
	body  io.Closer
}

func armOfflineUploadDeadline(body io.Closer, timeout time.Duration) *offlineUploadDeadline {
	deadline := &offlineUploadDeadline{body: body}
	if body == nil || timeout <= 0 {
		return deadline
	}
	deadline.timer = time.AfterFunc(timeout, func() {
		deadline.mu.Lock()
		defer deadline.mu.Unlock()
		if deadline.done {
			return
		}
		_ = deadline.body.Close()
	})
	return deadline
}

func (deadline *offlineUploadDeadline) Stop() {
	deadline.mu.Lock()
	defer deadline.mu.Unlock()
	deadline.done = true
	if deadline.timer != nil {
		deadline.timer.Stop()
	}
}
