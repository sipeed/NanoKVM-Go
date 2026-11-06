package application

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"github.com/gin-gonic/gin"
	"golang.org/x/sys/unix"
)

type offlineTestLease struct {
	mu       sync.Mutex
	released int
}

func (lease *offlineTestLease) Release() {
	lease.mu.Lock()
	defer lease.mu.Unlock()
	lease.released++
}

func (lease *offlineTestLease) releases() int {
	lease.mu.Lock()
	defer lease.mu.Unlock()
	return lease.released
}

func offlineTestDirectory(t *testing.T) string {
	t.Helper()
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate offline update test source")
	}
	parent := filepath.Join(filepath.Dir(source), "..", "..", "..", "build", "test-tmp")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(parent, ".application-offline-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(directory); err != nil {
			t.Errorf("remove offline update test directory: %v", err)
		}
	})
	return directory
}

func offlineTestArchive(t *testing.T, version string) []byte {
	t.Helper()
	var body bytes.Buffer
	gzipWriter := gzip.NewWriter(&body)
	tarWriter := tar.NewWriter(gzipWriter)
	root := updateCacheDirName(version)
	if err := tarWriter.WriteHeader(&tar.Header{Name: root + "/", Typeflag: tar.TypeDir, Mode: 0o755}); err != nil {
		t.Fatal(err)
	}
	content := []byte("candidate bytes")
	if err := tarWriter.WriteHeader(&tar.Header{Name: root + "/payload", Typeflag: tar.TypeReg, Mode: 0o600, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

type offlineMultipartPart struct {
	fieldName string
	fileName  string
	content   []byte
}

func offlineMultipartRequest(t *testing.T, parts ...offlineMultipartPart) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, item := range parts {
		var part io.Writer
		var err error
		if item.fileName == "" {
			part, err = writer.CreateFormField(item.fieldName)
		} else {
			part, err = writer.CreateFormFile(item.fieldName, item.fileName)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(item.content); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/application/update/offline", bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func TestReceiveOfflineApplicationUploadIsStrictAndIgnoresBrowserFilename(t *testing.T) {
	archive := offlineTestArchive(t, "0.0.25")
	tests := []struct {
		name        string
		parts       []offlineMultipartPart
		wantErr     error
		wantVersion string
	}{
		{
			name:        "browser filename cannot select version or path",
			parts:       []offlineMultipartPart{{fieldName: "file", fileName: "../../firmware_9.9.9.tar.gz", content: archive}},
			wantVersion: "0.0.25",
		},
		{
			name:    "missing file",
			parts:   nil,
			wantErr: errOfflineUploadInvalid,
		},
		{
			name:    "wrong field",
			parts:   []offlineMultipartPart{{fieldName: "archive", fileName: "nanokvm-go_0.0.25.tar.gz", content: archive}},
			wantErr: errOfflineUploadInvalid,
		},
		{
			name:    "ordinary form field",
			parts:   []offlineMultipartPart{{fieldName: "file", content: archive}},
			wantErr: errOfflineUploadInvalid,
		},
		{
			name: "duplicate file",
			parts: []offlineMultipartPart{
				{fieldName: "file", fileName: "one.tar.gz", content: archive},
				{fieldName: "file", fileName: "two.tar.gz", content: archive},
			},
			wantErr: errOfflineUploadInvalid,
		},
		{
			name: "extra field",
			parts: []offlineMultipartPart{
				{fieldName: "file", fileName: "one.tar.gz", content: archive},
				{fieldName: "version", content: []byte("9.9.9")},
			},
			wantErr: errOfflineUploadInvalid,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			workDir := offlineTestDirectory(t)
			request := offlineMultipartRequest(t, test.parts...)
			path, version, err := receiveOfflineApplicationUpload(request, workDir, int64(len(archive))+4096, nil)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("error = %v, want %v", err, test.wantErr)
			}
			if test.wantErr != nil {
				entries, readErr := os.ReadDir(workDir)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if len(entries) != 0 {
					t.Fatalf("failed upload left %d staged entries", len(entries))
				}
				return
			}
			if version != test.wantVersion {
				t.Fatalf("version = %q, want %q", version, test.wantVersion)
			}
			if filepath.Dir(path) != workDir || filepath.Base(path) == filepath.Base(test.parts[0].fileName) {
				t.Fatalf("staged path trusts browser filename: %q", path)
			}
			info, statErr := os.Lstat(path)
			if statErr != nil {
				t.Fatal(statErr)
			}
			if !info.Mode().IsRegular() || info.Mode().Perm() != 0o600 {
				t.Fatalf("archive mode = %v", info.Mode())
			}
		})
	}
}

func TestReceiveOfflineApplicationUploadEnforcesIndependentBudget(t *testing.T) {
	archive := offlineTestArchive(t, "0.0.25")
	request := offlineMultipartRequest(t, offlineMultipartPart{
		fieldName: "file", fileName: "nanokvm-go_0.0.25.tar.gz", content: archive,
	})
	request.ContentLength = -1 // writer-side enforcement cannot use preflight
	workDir := offlineTestDirectory(t)
	if _, _, err := receiveOfflineApplicationUpload(request, workDir, int64(len(archive))-1, nil); !errors.Is(err, errOfflineUploadNoSpace) {
		t.Fatalf("error = %v, want no space", err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("budget failure left %d staged entries", len(entries))
	}
}

type failingOfflineWriter struct {
	err error
}

func (writer failingOfflineWriter) Write([]byte) (int, error) {
	return 0, writer.err
}

func TestCopyOfflineApplicationUploadPreservesStorageErrors(t *testing.T) {
	for _, storageErr := range []error{unix.ENOSPC, unix.EDQUOT} {
		pathErr := &os.PathError{Op: "write", Path: "offline-archive", Err: storageErr}
		err := copyOfflineApplicationUpload(failingOfflineWriter{err: pathErr}, strings.NewReader("archive"))
		if !errors.Is(err, storageErr) {
			t.Fatalf("copy error = %v, want preserved %v", err, storageErr)
		}
		if code := classifyOfflineUploadError(err); code != UpdateErrorNoSpace {
			t.Fatalf("copy error %v classified as %s", storageErr, code)
		}
	}

	interrupted := errors.New("client disconnected")
	err := copyOfflineApplicationUpload(failingOfflineWriter{err: interrupted}, strings.NewReader("archive"))
	if !errors.Is(err, errOfflineUploadInterrupted) {
		t.Fatalf("ordinary copy error = %v, want upload interruption", err)
	}
	if code := classifyOfflineUploadError(err); code != UpdateErrorUploadInterrupted {
		t.Fatalf("ordinary copy error classified as %s", code)
	}
}

func TestReceiveOfflineApplicationUploadUsesContentLengthOnlyForPreflight(t *testing.T) {
	request := offlineMultipartRequest(t, offlineMultipartPart{
		fieldName: "file", fileName: "nanokvm-go_0.0.25.tar.gz", content: offlineTestArchive(t, "0.0.25"),
	})
	workDir := offlineTestDirectory(t)
	if _, _, err := receiveOfflineApplicationUpload(request, workDir, request.ContentLength-1, nil); !errors.Is(err, errOfflineUploadNoSpace) {
		t.Fatalf("error = %v, want no space", err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("preflight created %d staged entries", len(entries))
	}
}

type failingOfflineReader struct {
	data []byte
	err  error
}

func (reader *failingOfflineReader) Read(buffer []byte) (int, error) {
	if len(reader.data) == 0 {
		return 0, reader.err
	}
	written := copy(buffer, reader.data)
	reader.data = reader.data[written:]
	return written, nil
}

func TestReceiveOfflineApplicationUploadClassifiesInterruptedMultipart(t *testing.T) {
	request := offlineMultipartRequest(t, offlineMultipartPart{
		fieldName: "file", fileName: "nanokvm-go_0.0.25.tar.gz", content: offlineTestArchive(t, "0.0.25"),
	})
	body, err := io.ReadAll(request.Body)
	if err != nil {
		t.Fatal(err)
	}
	request.Body = io.NopCloser(&failingOfflineReader{data: body[:len(body)/2], err: errors.New("client disconnected")})
	request.ContentLength = -1
	workDir := offlineTestDirectory(t)
	if _, _, err := receiveOfflineApplicationUpload(request, workDir, int64(len(body))+4096, nil); !errors.Is(err, errOfflineUploadInterrupted) {
		t.Fatalf("error = %v, want interrupted", err)
	}
	entries, err := os.ReadDir(workDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("interrupted upload left %d staged entries", len(entries))
	}
}

func TestDiscoverOfflineApplicationArchiveVersionRequiresCanonicalSingleRoot(t *testing.T) {
	tests := []struct {
		name    string
		entries []struct {
			name      string
			directory bool
		}
		want string
	}{
		{
			name: "canonical",
			entries: []struct {
				name      string
				directory bool
			}{{"nanokvm-go_1.2.3/", true}, {"nanokvm-go_1.2.3/file", false}},
			want: "1.2.3",
		},
		{name: "leading zero", entries: []struct {
			name      string
			directory bool
		}{{"nanokvm-go_01.2.3/", true}}},
		{name: "missing explicit root", entries: []struct {
			name      string
			directory bool
		}{{"nanokvm-go_1.2.3/file", false}}},
		{name: "multiple roots", entries: []struct {
			name      string
			directory bool
		}{{"nanokvm-go_1.2.3/", true}, {"extra/", true}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(offlineTestDirectory(t), "archive.tar.gz")
			var body bytes.Buffer
			gzipWriter := gzip.NewWriter(&body)
			tarWriter := tar.NewWriter(gzipWriter)
			for _, entry := range test.entries {
				header := &tar.Header{Name: entry.name, Typeflag: tar.TypeReg, Mode: 0o600}
				if entry.directory {
					header.Typeflag = tar.TypeDir
					header.Mode = 0o700
				}
				if err := tarWriter.WriteHeader(header); err != nil {
					t.Fatal(err)
				}
			}
			if err := tarWriter.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gzipWriter.Close(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(archivePath, body.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			version, err := discoverOfflineApplicationArchiveVersion(archivePath)
			if test.want == "" && err == nil {
				t.Fatalf("invalid archive returned version %q", version)
			}
			if test.want != "" && (err != nil || version != test.want) {
				t.Fatalf("version = %q, error = %v, want %q", version, err, test.want)
			}
		})
	}
}

func TestUpdateOfflineAdmitsBeforeReadingBodyAndHandsOffPrivateStaging(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseDir := offlineTestDirectory(t)
	lease := &offlineTestLease{}
	status := ApplicationUpdateStatus{
		OperationID: "offline-operation", Source: UpdateSourceOffline, Phase: UpdatePhaseUploading,
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano), UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	current := status
	var handedArchive, handedWorkDir string
	deps := offlineApplicationUpdateDependencies{
		baseDir:       baseDir,
		uploadTimeout: time.Minute,
		budget:        func(string) (int64, error) { return 1 << 20, nil },
		beginAdmission: func(source UpdateSource, version string, phase UpdatePhase) (offlineApplicationLease, ApplicationUpdateStatus, error) {
			if source != UpdateSourceOffline || version != "" || phase != UpdatePhaseUploading {
				t.Fatalf("unexpected admission: %q %q %q", source, version, phase)
			}
			return lease, status, nil
		},
		startOperation: func(gotLease offlineApplicationLease, gotStatus ApplicationUpdateStatus, archivePath, workDir string) error {
			if gotLease != lease || gotStatus.Version != "0.0.25" {
				t.Fatalf("unexpected handoff: lease=%v status=%+v", gotLease, gotStatus)
			}
			handedArchive, handedWorkDir = archivePath, workDir
			return nil
		},
		setVersion: func(operationID, version string) error {
			current.Version = version
			return nil
		},
		advance: func(operationID string, phase UpdatePhase, progress int) error {
			current.Phase, current.Progress = phase, progress
			return nil
		},
		fail: func(operationID string, code UpdateErrorCode) error {
			current.Phase, current.ErrorCode = UpdatePhaseFailed, code
			return nil
		},
		current: func() (ApplicationUpdateStatus, error) { return current, nil },
	}
	request := offlineMultipartRequest(t, offlineMultipartPart{
		fieldName: "file", fileName: "malicious-name.tar.gz", content: offlineTestArchive(t, "0.0.25"),
	})
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	updateOfflineApplication(context, deps)

	var response proto.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != 0 || handedArchive == "" || handedWorkDir == "" {
		t.Fatalf("response=%+v archive=%q work=%q", response, handedArchive, handedWorkDir)
	}
	if filepath.Dir(handedWorkDir) != baseDir || filepath.Dir(handedArchive) != handedWorkDir {
		t.Fatalf("handoff escaped base directory: %q %q", handedWorkDir, handedArchive)
	}
	workInfo, err := os.Stat(handedWorkDir)
	if err != nil || workInfo.Mode().Perm() != 0o700 {
		t.Fatalf("work directory mode/error = %v/%v", workInfo, err)
	}
	archiveInfo, err := os.Stat(handedArchive)
	if err != nil || archiveInfo.Mode().Perm() != 0o600 {
		t.Fatalf("archive mode/error = %v/%v", archiveInfo, err)
	}
	if lease.releases() != 0 {
		t.Fatalf("successful handoff released lease %d times", lease.releases())
	}
}

type trackingOfflineBody struct {
	reader *strings.Reader
	reads  int
}

func (body *trackingOfflineBody) Read(buffer []byte) (int, error) {
	body.reads++
	return body.reader.Read(buffer)
}

func (body *trackingOfflineBody) Close() error { return nil }

func TestUpdateOfflineBusyDoesNotReadRequestBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := &trackingOfflineBody{reader: strings.NewReader("must not be read")}
	request := httptest.NewRequest(http.MethodPost, "/api/application/update/offline", body)
	request.Header.Set("Content-Type", "multipart/form-data; boundary=unused")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	deps := offlineApplicationUpdateDependencies{
		beginAdmission: func(UpdateSource, string, UpdatePhase) (offlineApplicationLease, ApplicationUpdateStatus, error) {
			return nil, ApplicationUpdateStatus{}, errApplicationUpdateBusy
		},
	}
	updateOfflineApplication(context, deps)
	if body.reads != 0 {
		t.Fatalf("busy request body read %d times", body.reads)
	}
	var response struct {
		Code int                     `json:"code"`
		Data ApplicationUpdateStatus `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code == 0 || response.Data.ErrorCode != UpdateErrorBusy || response.Data.Phase != UpdatePhaseFailed {
		t.Fatalf("unexpected busy response: %+v", response)
	}
}

func TestUpdateOfflineRejectsSymlinkCacheRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	parent := offlineTestDirectory(t)
	realRoot := filepath.Join(parent, "real-root")
	if err := os.Mkdir(realRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	cacheRoot := filepath.Join(parent, "cache-root")
	if err := os.Symlink(realRoot, cacheRoot); err != nil {
		t.Fatal(err)
	}

	lease := &offlineTestLease{}
	status := ApplicationUpdateStatus{
		OperationID: "offline-operation", Source: UpdateSourceOffline, Phase: UpdatePhaseUploading,
	}
	current := status
	deps := offlineApplicationUpdateDependencies{
		baseDir:       cacheRoot,
		uploadTimeout: time.Minute,
		beginAdmission: func(UpdateSource, string, UpdatePhase) (offlineApplicationLease, ApplicationUpdateStatus, error) {
			return lease, status, nil
		},
		fail: func(operationID string, code UpdateErrorCode) error {
			current.Phase, current.ErrorCode = UpdatePhaseFailed, code
			return nil
		},
		current: func() (ApplicationUpdateStatus, error) { return current, nil },
	}
	body := &trackingOfflineBody{reader: strings.NewReader("must not be read")}
	request := httptest.NewRequest(http.MethodPost, "/api/application/update/offline", body)
	request.Header.Set("Content-Type", "multipart/form-data; boundary=unused")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	updateOfflineApplication(context, deps)

	if body.reads != 0 {
		t.Fatalf("unsafe cache root request body read %d times", body.reads)
	}
	if lease.releases() != 1 {
		t.Fatalf("unsafe cache root released lease %d times", lease.releases())
	}
	entries, err := os.ReadDir(realRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("unsafe cache root created %d entries through symlink", len(entries))
	}
	if current.Phase != UpdatePhaseFailed || current.ErrorCode != UpdateErrorInterrupted {
		t.Fatalf("unsafe cache root status = %+v", current)
	}
}

func TestUpdateOfflineFailureCleansStagingAndReleasesLease(t *testing.T) {
	gin.SetMode(gin.TestMode)
	baseDir := offlineTestDirectory(t)
	lease := &offlineTestLease{}
	status := ApplicationUpdateStatus{OperationID: "offline-operation", Source: UpdateSourceOffline, Phase: UpdatePhaseUploading}
	current := status
	deps := offlineApplicationUpdateDependencies{
		baseDir:       baseDir,
		uploadTimeout: time.Minute,
		budget:        func(string) (int64, error) { return 1 << 20, nil },
		beginAdmission: func(UpdateSource, string, UpdatePhase) (offlineApplicationLease, ApplicationUpdateStatus, error) {
			return lease, status, nil
		},
		setVersion: func(string, string) error { return nil },
		advance:    func(string, UpdatePhase, int) error { return nil },
		fail: func(operationID string, code UpdateErrorCode) error {
			current.Phase, current.ErrorCode = UpdatePhaseFailed, code
			return nil
		},
		current: func() (ApplicationUpdateStatus, error) { return current, nil },
	}
	request := offlineMultipartRequest(t,
		offlineMultipartPart{fieldName: "file", fileName: "one.tar.gz", content: offlineTestArchive(t, "0.0.25")},
		offlineMultipartPart{fieldName: "extra", content: []byte("unexpected")},
	)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	updateOfflineApplication(context, deps)
	if lease.releases() != 1 {
		t.Fatalf("failure released lease %d times", lease.releases())
	}
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failure left %d staging entries", len(entries))
	}
	if current.ErrorCode != UpdateErrorInvalidArchive {
		t.Fatalf("failure code = %q", current.ErrorCode)
	}
}
