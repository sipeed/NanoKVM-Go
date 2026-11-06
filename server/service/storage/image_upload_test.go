package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sys/unix"
)

const testImageUploadID = "0123456789abcdef0123456789abcdef"

type imageUploadTestResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Filename    string `json:"filename"`
		ChunkIndex  int64  `json:"chunkIndex"`
		TotalChunks int64  `json:"totalChunks"`
		Written     int64  `json:"written"`
		Status      string `json:"status"`
		UploadID    string `json:"uploadId"`
	} `json:"data"`
}

func TestImageUploadMemoryBudget(t *testing.T) {
	if imageUploadBoundedBytes > imageUploadMemoryBudgetBytes {
		t.Fatalf(
			"bounded upload bytes = %d, budget = %d",
			imageUploadBoundedBytes,
			imageUploadMemoryBudgetBytes,
		)
	}
	if imageUploadCopyBufferBytes > 64<<10 {
		t.Fatalf("copy buffer = %d, want at most 64 KiB", imageUploadCopyBufferBytes)
	}
	if imageUploadMaxConcurrentChunks != 1 {
		t.Fatalf("concurrent chunks = %d, want 1", imageUploadMaxConcurrentChunks)
	}
}

func setImageUploadPolicyForTest(t *testing.T, limits imageUploadLimits) {
	t.Helper()
	original := imageUploadPolicy
	imageUploadPolicy = func() imageUploadLimits { return limits }
	t.Cleanup(func() { imageUploadPolicy = original })
}

func setImageUploadStatfsForTest(t *testing.T, statfs func(string, *unix.Statfs_t) error) {
	t.Helper()
	original := imageUploadStatfs
	imageUploadStatfs = statfs
	t.Cleanup(func() { imageUploadStatfs = original })
}

func setImageUploadStatfsAvailability(t *testing.T, availableBlocks uint64, blockSize int64) {
	t.Helper()
	setImageUploadStatfsForTest(t, func(_ string, stat *unix.Statfs_t) error {
		*stat = unix.Statfs_t{Bavail: availableBlocks, Bsize: blockSize}
		return nil
	})
}

// installImageUploadTestCapacity gives handler tests a generous, deterministic
// capacity policy so admission and the in-flight budget stay out of the way of
// behavior those tests exercise.
func installImageUploadTestCapacity(t *testing.T) {
	t.Helper()
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 40,
		ReservedFreeBytes: 1 << 20,
	})
	setImageUploadStatfsAvailability(t, 1<<24, 4096)
}

// setImageUploadStatfsTrackingPart installs a Statfs seam whose available
// block count shrinks as the upload's .part file grows, mirroring how a real
// filesystem accounts the partially written image. freeSequence supplies the
// base free bytes (before subtracting the .part size) for successive Statfs
// calls; the last entry is reused once the sequence is exhausted.
func setImageUploadStatfsTrackingPart(t *testing.T, directory string, freeSequence []int64) {
	t.Helper()
	calls := 0
	setImageUploadStatfsForTest(t, func(_ string, stat *unix.Statfs_t) error {
		baseFree := freeSequence[calls]
		if calls < len(freeSequence)-1 {
			calls++
		}
		var partSize int64
		if info, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); err == nil {
			partSize = info.Size()
		}
		available := baseFree - partSize
		if available < 0 {
			available = 0
		}
		*stat = unix.Statfs_t{Bavail: uint64(available) / 4096, Bsize: 4096}
		return nil
	})
}

func TestUploadImageStreamsRawChunksInOrder(t *testing.T) {
	installImageUploadTestCapacity(t)
	directory := t.TempDir()
	var cacheRanges [][2]int64
	deps := imageUploadDependencies{
		directory: directory,
		limiter:   newImageUploadLimiter(1),
		syncFile: func(file *os.File) error {
			return file.Sync()
		},
		dropCache: func(_ *os.File, offset, length int64) error {
			cacheRanges = append(cacheRanges, [2]int64{offset, length})
			return nil
		},
	}

	first := bytes.Repeat([]byte{0x11}, int(imageUploadChunkSizeBytes))
	second := []byte("final image bytes")

	response := performImageUploadRequest(t, deps, "disk.iso", 0, 2, first, int64(len(first)))
	if response.Code != 0 || response.Data.Written != int64(len(first)) {
		t.Fatalf("first chunk response = %+v", response)
	}
	if _, err := os.Stat(filepath.Join(directory, "disk.iso")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("incomplete image was published: %v", err)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); err != nil {
		t.Fatalf("incomplete image temporary file: %v", err)
	}
	response = performImageUploadRequest(t, deps, "disk.iso", 1, 2, second, int64(len(second)))
	if response.Code != 0 || response.Data.Written != int64(len(second)) {
		t.Fatalf("second chunk response = %+v", response)
	}

	content, err := os.ReadFile(filepath.Join(directory, "disk.iso"))
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]byte(nil), first...), second...)
	if !bytes.Equal(content, want) {
		t.Fatalf("uploaded image length = %d, want %d", len(content), len(want))
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed upload left temporary file: %v", err)
	}
	wantRanges := [][2]int64{
		{0, int64(len(first))},
		{imageUploadChunkSizeBytes, int64(len(second))},
	}
	if len(cacheRanges) != len(wantRanges) {
		t.Fatalf("cache ranges = %v, want %v", cacheRanges, wantRanges)
	}
	for index := range wantRanges {
		if cacheRanges[index] != wantRanges[index] {
			t.Fatalf("cache ranges = %v, want %v", cacheRanges, wantRanges)
		}
	}
}

func TestUploadImageCreatesMissingImageDirectory(t *testing.T) {
	installImageUploadTestCapacity(t)
	directory := filepath.Join(t.TempDir(), "data")
	response := performImageUploadRequest(
		t,
		imageUploadTestDependencies(directory),
		"disk.iso",
		0,
		1,
		[]byte("image bytes"),
		int64(len("image bytes")),
	)
	if response.Code != 0 {
		t.Fatalf("response = %+v", response)
	}
	content, err := os.ReadFile(filepath.Join(directory, "disk.iso"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "image bytes" {
		t.Fatalf("uploaded content = %q", content)
	}
}

func TestUploadImageRejectsSymlinkImageDirectory(t *testing.T) {
	installImageUploadTestCapacity(t)
	base := t.TempDir()
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(base, "data")
	if err := os.Symlink(outside, directory); err != nil {
		t.Fatal(err)
	}

	response := performImageUploadRequest(
		t,
		imageUploadTestDependencies(directory),
		"disk.iso",
		0,
		1,
		[]byte("image bytes"),
		int64(len("image bytes")),
	)
	if response.Code == 0 {
		t.Fatalf("symlink image directory accepted: %+v", response)
	}
	if _, err := os.Stat(filepath.Join(outside, "disk.iso")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside image stat error = %v", err)
	}
}

func TestUploadImageDoesNotParseOrRetainWholeChunk(t *testing.T) {
	installImageUploadTestCapacity(t)
	directory := t.TempDir()
	reader := &maxReadImageUploadReader{
		remaining: imageUploadChunkSizeBytes,
		maxRead:   imageUploadCopyBufferBytes,
	}
	request := imageUploadRequest("stream.iso", 0, 1, reader, imageUploadChunkSizeBytes)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	uploadImageChunk(ctx, imageUploadTestDependencies(directory))

	response := decodeImageUploadResponse(t, recorder)
	if response.Code != 0 || response.Data.Written != imageUploadChunkSizeBytes {
		t.Fatalf("response = %+v", response)
	}
	if request.MultipartForm != nil {
		t.Fatal("raw image upload unexpectedly parsed a multipart form")
	}
	info, err := os.Stat(filepath.Join(directory, "stream.iso"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != imageUploadChunkSizeBytes {
		t.Fatalf("uploaded size = %d, want %d", info.Size(), imageUploadChunkSizeBytes)
	}
}

func TestUploadImageRejectsOversizedAndShortChunks(t *testing.T) {
	tests := []struct {
		name          string
		body          []byte
		contentLength int64
		wantCode      int
	}{
		{
			name:          "declared oversized",
			body:          []byte("not read"),
			contentLength: imageUploadChunkSizeBytes + 1,
			wantCode:      -3,
		},
		{
			name:          "streamed oversized with unknown length",
			body:          bytes.Repeat([]byte{0x22}, int(imageUploadChunkSizeBytes)+1),
			contentLength: -1,
			wantCode:      -3,
		},
		{
			name:          "short non-final chunk",
			body:          []byte("short"),
			contentLength: int64(len("short")),
			wantCode:      -4,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			installImageUploadTestCapacity(t)
			directory := t.TempDir()
			deps := imageUploadTestDependencies(directory)
			response := performImageUploadRequest(
				t,
				deps,
				"disk.img",
				0,
				2,
				test.body,
				test.contentLength,
			)
			if response.Code != test.wantCode {
				t.Fatalf("response = %+v, want code %d", response, test.wantCode)
			}

			info, err := os.Stat(filepath.Join(directory, "disk.img"))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				t.Fatal(err)
			}
			if err == nil && info.Size() != 0 {
				t.Fatalf("failed chunk left %d bytes", info.Size())
			}
		})
	}
}

func TestUploadImageRejectsOutOfOrderChunk(t *testing.T) {
	installImageUploadTestCapacity(t)
	directory := t.TempDir()
	response := performImageUploadRequest(
		t,
		imageUploadTestDependencies(directory),
		"disk.img",
		1,
		2,
		[]byte("final"),
		int64(len("final")),
	)
	if response.Code != -5 || response.Msg != errImageUploadOutOfOrder.Error() {
		t.Fatalf("response = %+v", response)
	}
}

func TestUploadImageRejectsSymlinkTarget(t *testing.T) {
	installImageUploadTestCapacity(t)
	directory := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.iso")
	if err := os.WriteFile(outside, []byte("preserve me"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(directory, "disk.iso")); err != nil {
		t.Fatal(err)
	}

	response := performImageUploadRequest(
		t,
		imageUploadTestDependencies(directory),
		"disk.iso",
		0,
		1,
		[]byte("replacement"),
		int64(len("replacement")),
	)
	if response.Code == 0 {
		t.Fatalf("symlink target accepted: %+v", response)
	}
	content, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "preserve me" {
		t.Fatalf("outside target changed to %q", content)
	}
}

func TestUploadImageRejectsConcurrentChunkBeforeReadingBody(t *testing.T) {
	limiter := newImageUploadLimiter(1)
	release, ok := limiter.tryAcquire()
	if !ok {
		t.Fatal("reserve image upload limiter")
	}
	defer release()

	body := &trackingImageUploadBody{reader: strings.NewReader("must not be read")}
	request := imageUploadRequest("disk.iso", 0, 1, body, int64(len("must not be read")))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	deps := imageUploadTestDependencies(t.TempDir())
	deps.limiter = limiter
	uploadImageChunk(ctx, deps)

	response := decodeImageUploadResponse(t, recorder)
	if response.Code != -7 || response.Msg != errImageUploadBusy.Error() {
		t.Fatalf("response = %+v", response)
	}
	if body.reads != 0 {
		t.Fatalf("busy request body read %d times", body.reads)
	}
}

func TestParseImageUploadMetadataRejectsUnapprovedValues(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		chunkSize   string
		chunkIndex  string
		totalChunks string
	}{
		{name: "wrong extension", filename: "disk.txt", chunkSize: strconv.FormatInt(imageUploadChunkSizeBytes, 10), chunkIndex: "0", totalChunks: "1"},
		{name: "bin extension no longer accepted", filename: "disk.bin", chunkSize: strconv.FormatInt(imageUploadChunkSizeBytes, 10), chunkIndex: "0", totalChunks: "1"},
		{name: "hidden upload", filename: ".disk.iso", chunkSize: strconv.FormatInt(imageUploadChunkSizeBytes, 10), chunkIndex: "0", totalChunks: "1"},
		{name: "wrong chunk size", filename: "disk.iso", chunkSize: "1048576", chunkIndex: "0", totalChunks: "1"},
		{name: "negative index", filename: "disk.iso", chunkSize: strconv.FormatInt(imageUploadChunkSizeBytes, 10), chunkIndex: "-1", totalChunks: "1"},
		{name: "index outside total", filename: "disk.iso", chunkSize: strconv.FormatInt(imageUploadChunkSizeBytes, 10), chunkIndex: "1", totalChunks: "1"},
		{name: "overflow", filename: "disk.iso", chunkSize: strconv.FormatInt(imageUploadChunkSizeBytes, 10), chunkIndex: strconv.FormatInt(int64(^uint64(0)>>1)/imageUploadChunkSizeBytes+1, 10), totalChunks: strconv.FormatInt(int64(^uint64(0)>>1)/imageUploadChunkSizeBytes+2, 10)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/storage/image/upload", nil)
			query := request.URL.Query()
			query.Set("uploadId", testImageUploadID)
			query.Set("filename", test.filename)
			query.Set("chunkSize", test.chunkSize)
			query.Set("chunkIndex", test.chunkIndex)
			query.Set("totalChunks", test.totalChunks)
			request.URL.RawQuery = query.Encode()
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = request

			if _, err := parseImageUploadMetadata(ctx); !errors.Is(err, errImageUploadInvalid) {
				t.Fatalf("error = %v, want %v", err, errImageUploadInvalid)
			}
		})
	}
}

func TestParseImageUploadMetadataRejectsInvalidUploadID(t *testing.T) {
	request := imageUploadRequest("disk.iso", 0, 1, strings.NewReader("image"), int64(len("image")))
	query := request.URL.Query()
	query.Set("uploadId", "../../disk.iso")
	request.URL.RawQuery = query.Encode()
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = request

	if _, err := parseImageUploadMetadata(ctx); !errors.Is(err, errImageUploadInvalid) {
		t.Fatalf("error = %v, want %v", err, errImageUploadInvalid)
	}
}

func TestCancelImageUploadRemovesOnlyIncompleteUpload(t *testing.T) {
	directory := t.TempDir()
	deps := imageUploadTestDependencies(directory)
	temporaryPath := imageUploadTemporaryPath(directory, testImageUploadID)
	if err := os.WriteFile(temporaryPath, []byte("partial upload"), 0o600); err != nil {
		t.Fatal(err)
	}
	completedPath := filepath.Join(directory, "completed.iso")
	if err := os.WriteFile(completedPath, []byte("completed image"), 0o600); err != nil {
		t.Fatal(err)
	}
	unrelatedID := "fedcba9876543210fedcba9876543210"
	unrelatedPath := imageUploadTemporaryPath(directory, unrelatedID)
	if err := os.WriteFile(unrelatedPath, []byte("other partial upload"), 0o600); err != nil {
		t.Fatal(err)
	}

	response := performImageUploadCancelRequest(t, deps, testImageUploadID)
	if response.Code != 0 || response.Data.Status != "cancelled" {
		t.Fatalf("cancel response = %+v", response)
	}
	if _, err := os.Stat(temporaryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled upload still exists: %v", err)
	}
	content, err := os.ReadFile(completedPath)
	if err != nil || string(content) != "completed image" {
		t.Fatalf("completed image changed: content=%q err=%v", content, err)
	}
	content, err = os.ReadFile(unrelatedPath)
	if err != nil || string(content) != "other partial upload" {
		t.Fatalf("unrelated upload changed: content=%q err=%v", content, err)
	}
}

func TestCancelImageUploadCannotDeleteCompletedImage(t *testing.T) {
	directory := t.TempDir()
	completedPath := filepath.Join(directory, "disk.iso")
	if err := os.WriteFile(completedPath, []byte("published"), 0o600); err != nil {
		t.Fatal(err)
	}

	response := performImageUploadCancelRequest(
		t,
		imageUploadTestDependencies(directory),
		testImageUploadID,
	)
	if response.Code != 0 || response.Data.Status != "idle" {
		t.Fatalf("cancel response = %+v", response)
	}
	content, err := os.ReadFile(completedPath)
	if err != nil || string(content) != "published" {
		t.Fatalf("completed image changed: content=%q err=%v", content, err)
	}
}

func TestCancelImageUploadRejectsInvalidUploadID(t *testing.T) {
	response := performImageUploadCancelRequest(
		t,
		imageUploadTestDependencies(t.TempDir()),
		"../../disk.iso",
	)
	if response.Code != -2 || response.Msg != errImageUploadInvalid.Error() {
		t.Fatalf("cancel response = %+v", response)
	}
}

func TestCancelImageUploadWaitsForActiveChunkAndCleansUp(t *testing.T) {
	installImageUploadTestCapacity(t)
	directory := t.TempDir()
	deps := imageUploadTestDependencies(directory)
	started := make(chan struct{})
	releaseBody := make(chan struct{})
	body := &blockingImageUploadReader{
		reader:  bytes.NewReader(bytes.Repeat([]byte{0x44}, int(imageUploadChunkSizeBytes))),
		started: started,
		release: releaseBody,
	}
	uploadRecorder := httptest.NewRecorder()
	uploadCtx, _ := gin.CreateTestContext(uploadRecorder)
	uploadCtx.Request = imageUploadRequest(
		"disk.iso",
		0,
		2,
		body,
		imageUploadChunkSizeBytes,
	)
	uploadDone := make(chan struct{})
	go func() {
		uploadImageChunk(uploadCtx, deps)
		close(uploadDone)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("upload did not start")
	}

	cancelBody, err := json.Marshal(cancelImageUploadRequest{UploadID: testImageUploadID})
	if err != nil {
		t.Fatal(err)
	}
	cancelRecorder := httptest.NewRecorder()
	cancelCtx, _ := gin.CreateTestContext(cancelRecorder)
	cancelCtx.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/storage/image/upload/cancel",
		bytes.NewReader(cancelBody),
	)
	cancelCtx.Request.Header.Set("Content-Type", "application/json")
	cancelDone := make(chan struct{})
	go func() {
		cancelImageUpload(cancelCtx, deps)
		close(cancelDone)
	}()

	select {
	case <-cancelDone:
		t.Fatal("cancellation completed while the chunk was still active")
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseBody)

	for name, done := range map[string]<-chan struct{}{
		"upload": uploadDone,
		"cancel": cancelDone,
	} {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("%s did not finish", name)
		}
	}

	uploadResponse := decodeImageUploadResponse(t, uploadRecorder)
	if uploadResponse.Code != 0 {
		t.Fatalf("upload response = %+v", uploadResponse)
	}
	cancelResponse := decodeImageUploadResponse(t, cancelRecorder)
	if cancelResponse.Code != 0 || cancelResponse.Data.Status != "cancelled" {
		t.Fatalf("cancel response = %+v", cancelResponse)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled upload still exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "disk.iso")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("incomplete image was published: %v", err)
	}
}

func TestCopyImageUploadChunkUsesProvidedBoundedBuffer(t *testing.T) {
	reader := &maxReadImageUploadReader{
		remaining: imageUploadChunkSizeBytes,
		maxRead:   imageUploadCopyBufferBytes,
	}
	written, err := copyImageUploadChunk(io.Discard, reader, make([]byte, imageUploadCopyBufferBytes))
	if err != nil {
		t.Fatal(err)
	}
	if written != imageUploadChunkSizeBytes {
		t.Fatalf("written = %d, want %d", written, imageUploadChunkSizeBytes)
	}
}

func TestUploadImageRejectsDeclaredOversizeBeforeReadingBody(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     imageUploadChunkSizeBytes,
		ReservedFreeBytes: 1 << 20,
	})
	setImageUploadStatfsAvailability(t, 1<<24, 4096)

	directory := t.TempDir()
	body := &trackingImageUploadBody{reader: strings.NewReader("must not be read")}
	request := imageUploadRequest("disk.iso", 0, 2, body, int64(len("must not be read")))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	uploadImageChunk(ctx, imageUploadTestDependencies(directory))

	response := decodeImageUploadResponse(t, recorder)
	if response.Code != -8 || response.Msg != imageUploadSizeLimitResponseMessage {
		t.Fatalf("response = %+v, want code -8 %q", response, imageUploadSizeLimitResponseMessage)
	}
	if body.reads != 0 {
		t.Fatalf("oversized upload body read %d times", body.reads)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("oversized upload left temporary file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "disk.iso")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("oversized upload published an image: %v", err)
	}
}

func TestUploadImageRejectsInsufficientSpaceAtFirstChunk(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 40,
		ReservedFreeBytes: 1 << 20,
	})
	// One available block is far below the reserved bytes.
	setImageUploadStatfsAvailability(t, 1, 4096)

	directory := t.TempDir()
	body := &trackingImageUploadBody{reader: strings.NewReader("must not be read")}
	request := imageUploadRequest("disk.iso", 0, 1, body, int64(len("must not be read")))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	uploadImageChunk(ctx, imageUploadTestDependencies(directory))

	response := decodeImageUploadResponse(t, recorder)
	if response.Code != -9 || response.Msg != imageUploadInsufficientStorageResponseMessage {
		t.Fatalf("response = %+v, want code -9 %q", response, imageUploadInsufficientStorageResponseMessage)
	}
	if body.reads != 0 {
		t.Fatalf("insufficient-space upload body read %d times", body.reads)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("insufficient-space upload left temporary file: %v", err)
	}
}

func TestUploadImageRejectsDeclaredTotalAboveBudget(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 30,
		ReservedFreeBytes: 1 << 20,
	})
	// 20 MiB available minus 1 MiB reserve leaves a 19 MiB cap, below the
	// declared 24 MiB total of three 8 MiB chunks.
	setImageUploadStatfsAvailability(t, 5120, 4096)

	directory := t.TempDir()
	body := &trackingImageUploadBody{reader: strings.NewReader("must not be read")}
	request := imageUploadRequest("disk.iso", 0, 3, body, int64(len("must not be read")))
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	uploadImageChunk(ctx, imageUploadTestDependencies(directory))

	response := decodeImageUploadResponse(t, recorder)
	if response.Code != -9 || response.Msg != imageUploadInsufficientStorageResponseMessage {
		t.Fatalf("response = %+v, want code -9 %q", response, imageUploadInsufficientStorageResponseMessage)
	}
	if body.reads != 0 {
		t.Fatalf("over-budget upload body read %d times", body.reads)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("over-budget upload left temporary file: %v", err)
	}
}

// A partially written .part file already consumes filesystem blocks, so the
// capacity check must not count those bytes twice: a 16 MiB image fits a
// 20 MiB filesystem with a 1 MiB reserve even though the first chunk leaves
// only 12 MiB free.
func TestUploadImageSucceedsWhenImageExceedsHalfOfFreeSpace(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 40,
		ReservedFreeBytes: 1 << 20,
	})
	directory := t.TempDir()
	setImageUploadStatfsTrackingPart(t, directory, []int64{20 << 20, 20 << 20, 20 << 20, 20 << 20})

	deps := imageUploadTestDependencies(directory)
	first := bytes.Repeat([]byte{0x11}, int(imageUploadChunkSizeBytes))
	if response := performImageUploadRequest(t, deps, "disk.iso", 0, 2, first, int64(len(first))); response.Code != 0 {
		t.Fatalf("first chunk response = %+v", response)
	}
	final := []byte("final image bytes")
	response := performImageUploadRequest(t, deps, "disk.iso", 1, 2, final, int64(len(final)))
	if response.Code != 0 || response.Data.Written != int64(len(final)) {
		t.Fatalf("final chunk response = %+v", response)
	}

	content, err := os.ReadFile(filepath.Join(directory, "disk.iso"))
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]byte(nil), first...), final...)
	if !bytes.Equal(content, want) {
		t.Fatalf("uploaded image length = %d, want %d", len(content), len(want))
	}
}

func TestUploadImageAbortsWhenBudgetShrinksMidUpload(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 40,
		ReservedFreeBytes: 1 << 20,
	})
	// Statfs is called once at admission and once before each chunk write:
	// [chunk0 admission, chunk0 write, chunk1 admission, chunk1 write]. The
	// final answer reports a base free of 13 MiB, so the absolute cap drops to
	// 12 MiB: chunk 1 may start at offset 8 MiB but must stop 4 MiB in.
	directory := t.TempDir()
	setImageUploadStatfsTrackingPart(t, directory, []int64{64 << 30, 64 << 30, 64 << 30, 13 << 20})

	deps := imageUploadTestDependencies(directory)
	first := bytes.Repeat([]byte{0x11}, int(imageUploadChunkSizeBytes))
	response := performImageUploadRequest(t, deps, "disk.iso", 0, 2, first, int64(len(first)))
	if response.Code != 0 {
		t.Fatalf("first chunk response = %+v", response)
	}

	second := bytes.Repeat([]byte{0x22}, int(imageUploadChunkSizeBytes))
	response = performImageUploadRequest(t, deps, "disk.iso", 1, 2, second, int64(len(second)))
	if response.Code != -9 || response.Msg != imageUploadInsufficientStorageResponseMessage {
		t.Fatalf("shrunk-budget response = %+v, want code -9 %q", response, imageUploadInsufficientStorageResponseMessage)
	}

	info, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID))
	if err != nil {
		t.Fatalf("committed chunk should survive rollback: %v", err)
	}
	if info.Size() != imageUploadChunkSizeBytes {
		t.Fatalf("rolled back temporary size = %d, want %d", info.Size(), imageUploadChunkSizeBytes)
	}
	if _, err := os.Stat(filepath.Join(directory, "disk.iso")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("aborted upload published an image: %v", err)
	}
}

func TestUploadImageRetryAfterCapacityRollback(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 40,
		ReservedFreeBytes: 1 << 20,
	})
	// Chunk 1 sees a shrunk cap at write time (fourth call); the retry
	// afterwards gets generous space again.
	directory := t.TempDir()
	setImageUploadStatfsTrackingPart(t, directory, []int64{64 << 30, 64 << 30, 64 << 30, 13 << 20, 64 << 30, 64 << 30})

	deps := imageUploadTestDependencies(directory)
	first := bytes.Repeat([]byte{0x11}, int(imageUploadChunkSizeBytes))
	if response := performImageUploadRequest(t, deps, "disk.iso", 0, 2, first, int64(len(first))); response.Code != 0 {
		t.Fatalf("first chunk response = %+v", response)
	}
	second := bytes.Repeat([]byte{0x22}, int(imageUploadChunkSizeBytes))
	if response := performImageUploadRequest(t, deps, "disk.iso", 1, 2, second, int64(len(second))); response.Code != -9 {
		t.Fatalf("shrunk-budget response = %+v, want code -9", response)
	}

	final := []byte("final image bytes")
	response := performImageUploadRequest(t, deps, "disk.iso", 1, 2, final, int64(len(final)))
	if response.Code != 0 || response.Data.Written != int64(len(final)) {
		t.Fatalf("retry response = %+v", response)
	}
	content, err := os.ReadFile(filepath.Join(directory, "disk.iso"))
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]byte(nil), first...), final...)
	if !bytes.Equal(content, want) {
		t.Fatalf("retried image length = %d, want %d", len(content), len(want))
	}
}

func TestUploadImageRejectsShrunkBudgetAtFirstChunkWithoutTemporaryFile(t *testing.T) {
	setImageUploadPolicyForTest(t, imageUploadLimits{
		MaxImageBytes:     1 << 40,
		ReservedFreeBytes: 1 << 20,
	})
	// Admission sees generous space, but the write-time re-read reports the
	// reserve exhausted, so the chunk must fail before any file is created.
	directory := t.TempDir()
	setImageUploadStatfsTrackingPart(t, directory, []int64{64 << 30, 4096})
	response := performImageUploadRequest(
		t,
		imageUploadTestDependencies(directory),
		"disk.iso",
		0,
		1,
		[]byte("image bytes"),
		int64(len("image bytes")),
	)
	if response.Code != -9 || response.Msg != imageUploadInsufficientStorageResponseMessage {
		t.Fatalf("response = %+v, want code -9 %q", response, imageUploadInsufficientStorageResponseMessage)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected first chunk left temporary file: %v", err)
	}
}

func TestUploadImageRejectsInvalidFilesystemCapacity(t *testing.T) {
	tests := []struct {
		name      string
		available uint64
		blockSize int64
	}{
		{name: "available bytes overflow", available: ^uint64(0), blockSize: 2},
		{name: "invalid block size", available: 100, blockSize: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setImageUploadPolicyForTest(t, imageUploadLimits{
				MaxImageBytes:     1 << 40,
				ReservedFreeBytes: 1 << 20,
			})
			setImageUploadStatfsAvailability(t, test.available, test.blockSize)

			directory := t.TempDir()
			response := performImageUploadRequest(
				t,
				imageUploadTestDependencies(directory),
				"disk.iso",
				0,
				1,
				[]byte("image bytes"),
				int64(len("image bytes")),
			)
			if response.Code != -1 || response.Msg != "image upload failed" {
				t.Fatalf("capacity response = %+v, want generic -1 failure", response)
			}
			if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("capacity rejection left temporary file: %v", err)
			}
		})
	}
}

func TestUploadImageMapsDiskFullReadToInsufficientStorageAndCleansUp(t *testing.T) {
	installImageUploadTestCapacity(t)

	directory := t.TempDir()
	body := &errorAfterImageUploadReader{
		remaining: 4,
		err:       fmt.Errorf("write image chunk: %w", unix.ENOSPC),
	}
	request := imageUploadRequest("disk.iso", 0, 1, body, imageUploadChunkSizeBytes)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	uploadImageChunk(ctx, imageUploadTestDependencies(directory))

	response := decodeImageUploadResponse(t, recorder)
	if response.Code != -9 || response.Msg != imageUploadInsufficientStorageResponseMessage {
		t.Fatalf("response = %+v, want code -9 %q", response, imageUploadInsufficientStorageResponseMessage)
	}
	if _, err := os.Stat(imageUploadTemporaryPath(directory, testImageUploadID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("disk-full upload left temporary file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "disk.iso")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("disk-full upload published an image: %v", err)
	}
}

func TestMapImageUploadCopyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "disk full maps to insufficient storage",
			err:  fmt.Errorf("write image chunk: %w", unix.ENOSPC),
			want: errImageUploadInsufficientSpace,
		},
		{
			name: "budget sentinel passes through",
			err:  errImageUploadInsufficientSpace,
			want: errImageUploadInsufficientSpace,
		},
		{
			name: "size limit sentinel passes through",
			err:  errImageUploadSizeLimitExceeded,
			want: errImageUploadSizeLimitExceeded,
		},
		{
			name: "unrelated error passes through",
			err:  errors.New("boom"),
			want: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := mapImageUploadCopyError(test.err)
			if test.want == nil {
				if got != test.err {
					t.Fatalf("mapImageUploadCopyError = %v, want original error", got)
				}
				return
			}
			if !errors.Is(got, test.want) {
				t.Fatalf("mapImageUploadCopyError = %v, want %v", got, test.want)
			}
		})
	}
}

func TestImageUploadBoundedWriterLimits(t *testing.T) {
	t.Run("writes up to the budget are allowed", func(t *testing.T) {
		var destination bytes.Buffer
		writer := &imageUploadBoundedWriter{
			writer:        &destination,
			total:         4,
			maxImageBytes: 10,
			budget:        6,
		}
		if n, err := writer.Write([]byte("ab")); err != nil || n != 2 {
			t.Fatalf("Write = (%d, %v), want (2, nil)", n, err)
		}
		if writer.total != 6 {
			t.Fatalf("total = %d, want 6", writer.total)
		}
	})

	t.Run("crossing the budget reports insufficient storage", func(t *testing.T) {
		var destination bytes.Buffer
		writer := &imageUploadBoundedWriter{
			writer:        &destination,
			total:         4,
			maxImageBytes: 10,
			budget:        6,
		}
		if _, err := writer.Write([]byte("abc")); !errors.Is(err, errImageUploadInsufficientSpace) {
			t.Fatalf("Write error = %v, want insufficient storage", err)
		}
		if destination.Len() != 0 {
			t.Fatalf("destination bytes = %d, want 0", destination.Len())
		}
	})

	t.Run("crossing the image maximum reports the size limit", func(t *testing.T) {
		var destination bytes.Buffer
		writer := &imageUploadBoundedWriter{
			writer:        &destination,
			total:         4,
			maxImageBytes: 10,
			budget:        1 << 40,
		}
		if _, err := writer.Write([]byte("1234567")); !errors.Is(err, errImageUploadSizeLimitExceeded) {
			t.Fatalf("Write error = %v, want size limit", err)
		}
	})

	t.Run("a full budget rejects further writes as insufficient storage", func(t *testing.T) {
		var destination bytes.Buffer
		writer := &imageUploadBoundedWriter{
			writer:        &destination,
			total:         6,
			maxImageBytes: 10,
			budget:        6,
		}
		if _, err := writer.Write([]byte("a")); !errors.Is(err, errImageUploadInsufficientSpace) {
			t.Fatalf("Write error = %v, want insufficient storage", err)
		}
	})
}

func imageUploadTestDependencies(directory string) imageUploadDependencies {
	return imageUploadDependencies{
		directory: directory,
		limiter:   newImageUploadLimiter(1),
		syncFile:  func(*os.File) error { return nil },
		dropCache: func(*os.File, int64, int64) error { return nil },
	}
}

func performImageUploadRequest(
	t *testing.T,
	deps imageUploadDependencies,
	filename string,
	chunkIndex int64,
	totalChunks int64,
	body []byte,
	contentLength int64,
) imageUploadTestResponse {
	t.Helper()
	request := imageUploadRequest(filename, chunkIndex, totalChunks, bytes.NewReader(body), contentLength)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	uploadImageChunk(ctx, deps)
	return decodeImageUploadResponse(t, recorder)
}

func imageUploadRequest(
	filename string,
	chunkIndex int64,
	totalChunks int64,
	body io.Reader,
	contentLength int64,
) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/storage/image/upload", body)
	query := request.URL.Query()
	query.Set("uploadId", testImageUploadID)
	query.Set("filename", filename)
	query.Set("chunkSize", strconv.FormatInt(imageUploadChunkSizeBytes, 10))
	query.Set("chunkIndex", strconv.FormatInt(chunkIndex, 10))
	query.Set("totalChunks", strconv.FormatInt(totalChunks, 10))
	request.URL.RawQuery = query.Encode()
	request.Header.Set("Content-Type", "application/octet-stream")
	request.ContentLength = contentLength
	return request
}

func performImageUploadCancelRequest(
	t *testing.T,
	deps imageUploadDependencies,
	uploadID string,
) imageUploadTestResponse {
	t.Helper()
	body, err := json.Marshal(cancelImageUploadRequest{UploadID: uploadID})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/storage/image/upload/cancel",
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	cancelImageUpload(ctx, deps)
	return decodeImageUploadResponse(t, recorder)
}

func decodeImageUploadResponse(t *testing.T, recorder *httptest.ResponseRecorder) imageUploadTestResponse {
	t.Helper()
	var response imageUploadTestResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v body=%s", err, recorder.Body.String())
	}
	return response
}

type errorAfterImageUploadReader struct {
	remaining int64
	err       error
}

func (reader *errorAfterImageUploadReader) Read(buffer []byte) (int, error) {
	if reader.remaining <= 0 {
		return 0, reader.err
	}
	count := len(buffer)
	if int64(count) > reader.remaining {
		count = int(reader.remaining)
	}
	for index := 0; index < count; index++ {
		buffer[index] = 0x55
	}
	reader.remaining -= int64(count)
	return count, nil
}

type trackingImageUploadBody struct {
	reader io.Reader
	reads  int
}

func (body *trackingImageUploadBody) Read(buffer []byte) (int, error) {
	body.reads++
	return body.reader.Read(buffer)
}

func (body *trackingImageUploadBody) Close() error { return nil }

type maxReadImageUploadReader struct {
	remaining int64
	maxRead   int
	mu        sync.Mutex
}

type blockingImageUploadReader struct {
	reader  io.Reader
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (reader *blockingImageUploadReader) Read(buffer []byte) (int, error) {
	reader.once.Do(func() {
		close(reader.started)
		<-reader.release
	})
	return reader.reader.Read(buffer)
}

func (reader *maxReadImageUploadReader) Read(buffer []byte) (int, error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	if len(buffer) > reader.maxRead {
		return 0, errors.New("reader received an oversized buffer")
	}
	if reader.remaining == 0 {
		return 0, io.EOF
	}
	count := len(buffer)
	if int64(count) > reader.remaining {
		count = int(reader.remaining)
	}
	for index := 0; index < count; index++ {
		buffer[index] = byte(index)
	}
	reader.remaining -= int64(count)
	return count, nil
}
