package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/service/storage/diskfs"
	"NanoKVM-Server/storageguard"
	"github.com/gin-gonic/gin"
)

func TestReadDiskStatusUsesBothGoLUNs(t *testing.T) {
	root := t.TempDir()
	write := func(name, value string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("UDC", "controller")
	for _, disk := range []string{"disk0", "disk1"} {
		write("configs/c.1/mass_storage."+disk, "link")
		write("functions/mass_storage."+disk+"/lun.0/file", "\n")
	}
	write("functions/mass_storage.disk0/lun.0/ro", "1")
	write("functions/mass_storage.disk0/lun.0/cdrom", "1")
	status, err := readDiskStatusAt(root, false)
	if err != nil || status.State != diskStateNone {
		t.Fatalf("empty: %+v, %v", status, err)
	}
	write("functions/mass_storage.disk1/lun.0/file", partitionDevice)
	status, err = readDiskStatusAt(root, false)
	if err != nil || status.State != diskStatePartition {
		t.Fatalf("shared disk: %+v, %v", status, err)
	}
	write("functions/mass_storage.disk1/lun.0/file", "\n")
	write("functions/mass_storage.disk0/lun.0/file", "/data/win.iso")
	status, err = readDiskStatusAt(root, false)
	if err != nil || status.State != diskStateImage || !status.Cdrom || !status.ReadOnly {
		t.Fatalf("image: %+v, %v", status, err)
	}
	write("functions/mass_storage.disk0/lun.0/ro", "0")
	write("functions/mass_storage.disk0/lun.0/cdrom", "0")
	status, err = readDiskStatusAt(root, false)
	if err != nil || status.State != diskStateImage || status.Cdrom || status.ReadOnly {
		t.Fatalf("writable image: %+v, %v", status, err)
	}
	write("functions/mass_storage.disk1/lun.0/file", partitionDevice)
	if _, err := readDiskStatusAt(root, false); err == nil {
		t.Fatal("accepted two attached LUNs")
	}
	status, err = readDiskStatusAt(root, true)
	if err != nil || status.State != diskStateUnsupported || !status.HidOnly {
		t.Fatalf("HID-only: %+v, %v", status, err)
	}
}

func mediaHandlerFixture(t *testing.T) (*Service, *atomic.Int32) {
	t.Helper()
	oldDir, oldAcquire, oldChallenge, oldStatus := mediaDirectory, acquireStorageAccess, deleteImageActionRequiresChallenge, readDiskStatus
	oldInitialized := sharedDiskInitialized
	sharedDiskInitialized = func() bool { return true }
	mediaDirectory = t.TempDir()
	readDiskStatus = func() (diskStatus, error) { return diskStatus{State: diskStateNone}, nil }
	var refs atomic.Int32
	acquireStorageAccess = func(bool) (func(), error) { refs.Add(1); return func() { refs.Add(-1) }, nil }
	deleteImageActionRequiresChallenge = func() bool { return false }
	t.Cleanup(func() {
		mediaDirectory = oldDir
		acquireStorageAccess = oldAcquire
		deleteImageActionRequiresChallenge = oldChallenge
		readDiskStatus = oldStatus
		sharedDiskInitialized = oldInitialized
		if refs.Load() != 0 {
			t.Errorf("leaked storage locks: %d", refs.Load())
		}
	})
	return &Service{jobState: jobStateIdle}, &refs
}
func mediaContext(method, url string, body *bytes.Buffer) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	r := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(r)
	if body == nil {
		body = &bytes.Buffer{}
	}
	c.Request = httptest.NewRequest(method, url, body)
	return c, r
}

func TestMediaStateReportsImageAccessMode(t *testing.T) {
	s, _ := mediaHandlerFixture(t)
	for _, test := range []struct {
		name     string
		cdrom    bool
		readOnly bool
		want     bool
	}{
		{name: "writable disk"},
		{name: "read-only disk", readOnly: true, want: true},
		{name: "optical medium", cdrom: true, want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			readDiskStatus = func() (diskStatus, error) {
				return diskStatus{State: diskStateImage, Backing: filepath.Join(mediaDirectory, "rescue.img"), Cdrom: test.cdrom, ReadOnly: test.readOnly}, nil
			}
			c, r := mediaContext(http.MethodGet, "/api/storage/media/state", nil)
			s.GetMediaState(c)
			var response struct {
				Code int
				Data proto.MediaStateRsp
			}
			if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != 0 || response.Data.ReadOnly != test.want || response.Data.Cdrom != test.cdrom {
				t.Fatalf("access mode: %s, %v", r.Body, err)
			}
		})
	}
}

func TestMediaStateReportsActualLocalWriteAccess(t *testing.T) {
	for _, test := range []struct {
		name, state string
		accessErr   error
		wantError   string
	}{
		{name: "host eject recovered", state: "none"},
		{name: "USB owns storage", state: "partition", accessErr: storageguard.ErrReadOnly},
		{name: "recovery waits for transfer", state: "none", accessErr: storageguard.ErrBusy},
		{name: "local filesystem stayed read-only", state: "none", accessErr: storageguard.ErrReadOnly, wantError: "local storage is read-only"},
		{name: "recovery failed", state: "none", accessErr: errors.New("restore failed"), wantError: "restore failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			s, _ := mediaHandlerFixture(t)
			held := false
			acquireStorageAccess = func(writable bool) (func(), error) {
				if !writable {
					t.Fatal("state did not check actual write access")
				}
				if test.accessErr != nil {
					return nil, test.accessErr
				}
				held = true
				return func() { held = false }, nil
			}
			readDiskStatus = func() (diskStatus, error) {
				if test.accessErr == nil && !held {
					t.Fatal("USB state was read after releasing local ownership")
				}
				return diskStatus{State: diskState(test.state)}, nil
			}
			c, r := mediaContext(http.MethodGet, "/api/storage/media/state", nil)
			s.GetMediaState(c)
			var response struct {
				Code int
				Data proto.MediaStateRsp
			}
			if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != 0 {
				t.Fatalf("state response: %s, %v", r.Body, err)
			}
			if response.Data.LocalWritable != (test.accessErr == nil) || response.Data.LocalStorageError != test.wantError || held {
				t.Fatalf("state granted incorrect write access: %+v, lease held=%v", response.Data, held)
			}
		})
	}
}

func TestSharedDiskAvailabilityDoesNotHideLocalImages(t *testing.T) {
	s, _ := mediaHandlerFixture(t)
	if err := os.WriteFile(filepath.Join(mediaDirectory, "rescue.img"), []byte("local image"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, initialized := range []bool{false, true} {
		sharedDiskInitialized = func() bool { return initialized }
		items, err := listMedia()
		if err != nil {
			t.Fatal(err)
		}
		images, partitions := 0, 0
		for _, item := range items {
			if item.Type == proto.MediaTypeImage {
				images++
			}
			if item.Type == proto.MediaTypePartition {
				partitions++
			}
		}
		if images != 1 || (partitions == 1) != initialized || partitions > 1 {
			t.Fatalf("initialized=%v: images=%d, shared disks=%d", initialized, images, partitions)
		}
		c, r := mediaContext(http.MethodGet, "/api/storage/media/state", nil)
		s.GetMediaState(c)
		var response struct {
			Code int
			Data proto.MediaStateRsp
		}
		if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != 0 || response.Data.SharedDiskInitialized != initialized {
			t.Fatalf("initialized=%v: %s, %v", initialized, r.Body, err)
		}
	}
}

func TestUninitializedSharedDiskRejectsFileAccess(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	sharedDiskInitialized = func() bool { return false }
	keep := filepath.Join(mediaDirectory, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	var upload bytes.Buffer
	form := multipart.NewWriter(&upload)
	part, err := form.CreateFormFile("file", "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("must not reach the root filesystem"))
	_ = form.Close()
	for _, test := range []struct {
		name, method, url, body, contentType string
		handler                              func(*gin.Context)
	}{
		{"list", http.MethodGet, "/api/storage/media/files?type=partition&path=/", "", "", s.ListMediaFiles},
		{"upload", http.MethodPost, "/api/storage/media/files/upload?type=partition&path=/", upload.String(), form.FormDataContentType(), s.UploadMediaFile},
		{"mkdir", http.MethodPost, "/api/storage/media/files/mkdir", `{"type":"partition","path":"new-folder"}`, "application/json", s.MkdirMediaFile},
		{"download", http.MethodGet, "/api/storage/media/files/download?type=partition&path=keep.txt", "", "", s.DownloadMediaFile},
		{"delete", http.MethodPost, "/api/storage/media/files/delete", `{"type":"partition","path":"keep.txt"}`, "application/json", s.DeleteMediaFile},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, r := mediaContext(test.method, test.url, bytes.NewBufferString(test.body))
			c.Request.Header.Set("Content-Type", test.contentType)
			test.handler(c)
			var response struct{ Code int }
			if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != codeStorageDisabled {
				t.Fatalf("uninitialized shared disk: %s, %v", r.Body, err)
			}
			if refs.Load() != 0 {
				t.Fatal("rejected access retained a storage lock")
			}
		})
	}
	entries, err := os.ReadDir(mediaDirectory)
	if err != nil || len(entries) != 1 || entries[0].Name() != "keep.txt" {
		t.Fatalf("uninitialized operations changed the rootfs directory: %v, %v", entries, err)
	}
	if contents, err := os.ReadFile(keep); err != nil || string(contents) != "keep" {
		t.Fatalf("existing file changed: %q, %v", contents, err)
	}
}

func TestSharedFileUploadListDownloadAndDelete(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte("hello world"))
	_ = form.Close()
	c, r := mediaContext(http.MethodPost, "/api/storage/media/files/upload?type=partition&path=/", &body)
	c.Request.Header.Set("Content-Type", form.FormDataContentType())
	s.UploadMediaFile(c)
	var response struct {
		Code int
		Data proto.GetMediaFilesRsp
	}
	if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != 0 {
		t.Fatalf("upload: %s, %v", r.Body, err)
	}
	c, r = mediaContext(http.MethodGet, "/api/storage/media/files?type=partition&path=/", nil)
	s.ListMediaFiles(c)
	if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != 0 || len(response.Data.Entries) != 1 || response.Data.Entries[0].Name != "hello.txt" {
		t.Fatalf("listing: %s, %v", r.Body, err)
	}
	c, r = mediaContext(http.MethodGet, "/api/storage/media/files/download?type=partition&path=hello.txt", nil)
	c.Request.Header.Set("Range", "bytes=6-10")
	s.DownloadMediaFile(c)
	if r.Code != http.StatusPartialContent || r.Body.String() != "world" {
		t.Fatalf("range download: %d %s", r.Code, r.Body)
	}
	c, r = mediaContext(http.MethodPost, "/api/storage/media/files/delete", bytes.NewBufferString(`{"type":"partition","path":"hello.txt"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.DeleteMediaFile(c)
	if _, err := os.Stat(filepath.Join(mediaDirectory, "hello.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("delete: %s, %v", r.Body, err)
	}
	if refs.Load() != 0 {
		t.Fatal("handler retained storage lock")
	}
}
func TestSharedFileMkdirListsAsDirectory(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	mkdir := func(path string) int {
		c, r := mediaContext(http.MethodPost, "/api/storage/media/files/mkdir", bytes.NewBufferString(`{"type":"partition","path":"`+path+`"}`))
		c.Request.Header.Set("Content-Type", "application/json")
		s.MkdirMediaFile(c)
		var response struct {
			Code int
		}
		if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil {
			t.Fatalf("mkdir %s: %s, %v", path, r.Body, err)
		}
		return response.Code
	}
	if code := mkdir("docs"); code != 0 {
		t.Fatalf("mkdir docs: code %d", code)
	}
	info, err := os.Stat(filepath.Join(mediaDirectory, "docs"))
	if err != nil || !info.IsDir() {
		t.Fatalf("created dir: %v", err)
	}
	if code := mkdir("docs/inner"); code != 0 {
		t.Fatalf("nested mkdir: code %d", code)
	}
	c, r := mediaContext(http.MethodGet, "/api/storage/media/files?type=partition&path=/", nil)
	s.ListMediaFiles(c)
	var listing struct {
		Code int
		Data proto.GetMediaFilesRsp
	}
	if err := json.Unmarshal(r.Body.Bytes(), &listing); err != nil || listing.Code != 0 || len(listing.Data.Entries) != 1 || !listing.Data.Entries[0].IsDir || listing.Data.Entries[0].Name != "docs" {
		t.Fatalf("listing: %s, %v", r.Body, err)
	}
	if code := mkdir("docs"); code == 0 {
		t.Fatal("duplicate mkdir succeeded")
	}
	if code := mkdir("bad?name"); code == 0 {
		t.Fatal("invalid name accepted")
	}
	if code := mkdir("../escape"); code == 0 {
		t.Fatal("escaping path accepted")
	}
	if refs.Load() != 0 {
		t.Fatal("handler retained storage lock")
	}
}

func TestSharedImageDownloadRejectsAttachedImage(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	backing := filepath.Join(mediaDirectory, "images", "rescue.img")
	if err := os.Mkdir(filepath.Dir(backing), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backing, []byte("image content"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, readOnly := range []bool{false, true} {
		readDiskStatus = func() (diskStatus, error) {
			return diskStatus{State: diskStateImage, Backing: backing, ReadOnly: readOnly}, nil
		}
		for _, requestPath := range []string{"images/rescue.img", "IMAGES/RESCUE.IMG"} {
			c, r := mediaContext(http.MethodGet, "/api/storage/media/files/download?type=partition&path="+requestPath, nil)
			s.DownloadMediaFile(c)
			var response struct{ Code int }
			if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil || response.Code != codeMediumBusy {
				t.Fatalf("attached image readOnly=%v, path=%s: %s, %v", readOnly, requestPath, r.Body, err)
			}
		}
	}
	if refs.Load() != 0 || len(diskfs.BusyImages()) != 0 {
		t.Fatal("rejected image download retained a lock or read pin")
	}
}

type mediaDownloadWriter struct {
	*httptest.ResponseRecorder
	onWrite func()
}

func (w *mediaDownloadWriter) Write(data []byte) (int, error) {
	w.onWrite()
	return w.ResponseRecorder.Write(data)
}

func TestSharedImageDownloadPinsImageUntilStreamCloses(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	backing := filepath.Join(mediaDirectory, "rescue.img")
	if err := os.WriteFile(backing, []byte("image content"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRecorder()
	writes := 0
	c, _ := gin.CreateTestContext(&mediaDownloadWriter{ResponseRecorder: r, onWrite: func() {
		writes++
		mediumOperationMu.Lock()
		err := guardMediaDelete(backing)
		mediumOperationMu.Unlock()
		if !errors.Is(err, errMediumBusy) || refs.Load() == 0 {
			t.Errorf("stream did not retain its read pin and storage lock: %v", err)
		}
	}})
	c.Request = httptest.NewRequest(http.MethodGet, "/api/storage/media/files/download?type=partition&path=rescue.img", nil)
	c.Request.Header.Set("Range", "bytes=6-12")
	s.DownloadMediaFile(c)
	if r.Code != http.StatusPartialContent || r.Body.String() != "content" || writes == 0 {
		t.Fatalf("image range download: %d %s", r.Code, r.Body)
	}
	if refs.Load() != 0 || len(diskfs.BusyImages()) != 0 {
		t.Fatal("finished image download retained a lock or read pin")
	}
}

func TestMediaDeleteRequiresExistingProtectionGate(t *testing.T) {
	s, _ := mediaHandlerFixture(t)
	path := filepath.Join(mediaDirectory, "keep.iso")
	if err := os.WriteFile(path, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	deleteImageActionRequiresChallenge = func() bool { return true }
	oldVerify := verifyDeleteImageActionChallenge
	verifyDeleteImageActionChallenge = func(*gin.Context, string, string) error { return errors.New("missing challenge") }
	t.Cleanup(func() { verifyDeleteImageActionChallenge = oldVerify })
	c, r := mediaContext(http.MethodPost, "/api/storage/media/files/delete", bytes.NewBufferString(`{"type":"partition","path":"keep.iso"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.DeleteMediaFile(c)
	if !strings.Contains(r.Body.String(), "sensitive action verification required") {
		t.Fatalf("response: %s", r.Body)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("protected image was deleted")
	}
}
func TestDeleteGuardProtectsMountedAndDownloadedImages(t *testing.T) {
	if err := deleteGuard("/data/IMAGES", "/data/images/win.iso", nil); !errors.Is(err, errMediumAttached) {
		t.Fatal("attached image parent was deletable")
	}
	image := filepath.Join(t.TempDir(), "image.img")
	release, err := diskfs.AcquireReadPin(image)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := deleteGuard(filepath.Dir(image), "", diskfs.BusyImages()); !errors.Is(err, errMediumBusy) {
		t.Fatal("downloaded image parent was deletable")
	}
	if err := deleteGuard("/data/images-other", "/data/images/a.iso", nil); err != nil {
		t.Fatal("unrelated directory blocked")
	}
}
func TestImageChecksumVerificationAndCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "temporary")
	data := []byte("complete image")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if err := verifyImageChecksum(context.Background(), path, strings.ToUpper(hex.EncodeToString(sum[:]))); err != nil {
		t.Fatal(err)
	}
	if err := verifyImageChecksum(context.Background(), path, strings.Repeat("0", 64)); !errors.Is(err, errChecksumMismatch) {
		t.Fatalf("mismatch: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := verifyImageChecksum(ctx, path, hex.EncodeToString(sum[:])); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}

func TestImageUploadChecksumPreservesExistingImage(t *testing.T) {
	for _, match := range []bool{true, false} {
		t.Run(fmt.Sprintf("match=%v", match), func(t *testing.T) {
			_, _ = mediaHandlerFixture(t)
			installImageUploadTestCapacity(t)
			directory := t.TempDir()
			target := filepath.Join(directory, "disk.img")
			if err := os.WriteFile(target, []byte("old image"), 0o644); err != nil {
				t.Fatal(err)
			}
			data := []byte("new complete image")
			sum := sha256.Sum256(data)
			checksum := hex.EncodeToString(sum[:])
			if !match {
				checksum = strings.Repeat("0", 64)
			}
			request := imageUploadRequest("disk.img", 0, 1, bytes.NewReader(data), int64(len(data)))
			query := request.URL.Query()
			query.Set("sha256", checksum)
			request.URL.RawQuery = query.Encode()
			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Request = request
			uploadImageChunk(c, imageUploadTestDependencies(directory))
			response := decodeImageUploadResponse(t, r)
			content, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if match {
				if response.Code != 0 || !bytes.Equal(content, data) {
					t.Fatalf("valid image was not published: %+v %q", response, content)
				}
			} else if response.Msg != "sha256 mismatch" || string(content) != "old image" {
				t.Fatalf("checksum mismatch replaced the existing image: %+v %q", response, content)
			}
		})
	}
}

func TestSharedUploadsProtectUnpublishedTargetAndParent(t *testing.T) {
	_, _ = mediaHandlerFixture(t)
	release, err := reserveMediaUpload(proto.MediaTypePartition, "folder/new.img")
	if err != nil {
		t.Fatal(err)
	}
	mediumOperationMu.Lock()
	err = guardMediaDelete(filepath.Join(mediaDirectory, "folder"))
	mediumOperationMu.Unlock()
	if !errors.Is(err, errMediumBusy) {
		t.Fatal("upload parent was not protected")
	}
	release()
	mediumOperationMu.Lock()
	err = guardMediaDelete(filepath.Join(mediaDirectory, "folder"))
	mediumOperationMu.Unlock()
	if err != nil {
		t.Fatalf("completed upload retained destination: %v", err)
	}
}

func TestMediaJobsPreserveQueueAfterCancellation(t *testing.T) {
	s := &Service{}
	initial, err := s.beginJob(context.Background(), "initial")
	if err != nil {
		t.Fatal(err)
	}
	type started struct {
		name string
		done chan struct{}
		err  error
	}
	results := make(chan started, 3)
	start := func(ctx context.Context, name string) {
		go func() { done, err := s.beginJob(ctx, name); results <- started{name, done, err} }()
	}
	waitQueued := func(want int) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			s.jobMutex.Lock()
			n := len(s.jobWaiters)
			s.jobMutex.Unlock()
			if n == want {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Fatal("media transfer did not join the queue")
	}
	receive := func() started {
		t.Helper()
		select {
		case result := <-results:
			return result
		case <-time.After(2 * time.Second):
			t.Fatal("queued transfer did not resume")
			return started{}
		}
	}
	start(context.Background(), "first")
	waitQueued(1)
	ctx, cancel := context.WithCancel(context.Background())
	start(ctx, "cancelled")
	waitQueued(2)
	start(context.Background(), "last")
	waitQueued(3)
	cancel()
	if result := receive(); result.name != "cancelled" || !errors.Is(result.err, context.Canceled) {
		t.Fatalf("cancelled queue entry: %+v", result)
	}
	s.finishJob(initial, jobStateSuccess, "")
	first := receive()
	if first.name != "first" || first.err != nil {
		t.Fatalf("FIFO first: %+v", first)
	}
	s.finishJob(first.done, jobStateSuccess, "")
	last := receive()
	if last.name != "last" || last.err != nil {
		t.Fatalf("FIFO last: %+v", last)
	}
	s.finishJob(last.done, jobStateSuccess, "")
}
