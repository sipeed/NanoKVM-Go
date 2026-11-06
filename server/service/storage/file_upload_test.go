package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"NanoKVM-Server/proto"
)

type fileChunkResponse struct {
	Code int
	Data struct {
		Written  int64
		Complete bool
	}
}

func fileChunk(t *testing.T, s *Service, name, id string, size, index int64, body []byte) fileChunkResponse {
	t.Helper()
	params := url.Values{
		"type": {proto.MediaTypePartition}, "path": {"/"}, "filename": {name}, "uploadId": {id},
		"size": {fmt.Sprint(size)}, "chunkSize": {fmt.Sprint(imageUploadChunkSizeBytes)},
		"chunkIndex": {fmt.Sprint(index)}, "totalChunks": {fmt.Sprint(max(int64(1), (size+imageUploadChunkSizeBytes-1)/imageUploadChunkSizeBytes))},
	}
	c, r := mediaContext(http.MethodPost, "/api/storage/media/files/upload?"+params.Encode(), bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/octet-stream")
	s.UploadMediaFile(c)
	var response fileChunkResponse
	if err := json.Unmarshal(r.Body.Bytes(), &response); err != nil {
		t.Fatalf("chunk response: %s, %v", r.Body, err)
	}
	return response
}

func cancelFileUpload(t *testing.T, s *Service, id string) {
	t.Helper()
	c, r := mediaContext(http.MethodPost, "/api/storage/media/files/upload/cancel", bytes.NewBufferString(`{"uploadId":"`+id+`"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	s.CancelMediaFileUpload(c)
	if !strings.Contains(r.Body.String(), `"code":0`) {
		t.Fatalf("cancel: %s", r.Body)
	}
}

func TestFileChunksRetainOwnershipAndPublishOnlyOnCompletion(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	first := bytes.Repeat([]byte{0x35}, int(imageUploadChunkSizeBytes))
	tail := []byte("last chunk")
	size := int64(len(first) + len(tail))
	rsp := fileChunk(t, s, "archive.zip", testImageUploadID, size, 0, first)
	if rsp.Code != 0 || rsp.Data.Written != int64(len(first)) || rsp.Data.Complete {
		t.Fatalf("first chunk: %+v", rsp)
	}
	if refs.Load() != 1 || s.jobDone == nil {
		t.Fatal("upload released ownership or the job between chunks")
	}
	if _, err := os.Stat(filepath.Join(mediaDirectory, "archive.zip")); !os.IsNotExist(err) {
		t.Fatal("incomplete file was published")
	}
	c, r := mediaContext(http.MethodGet, "/api/storage/media/files?type=partition&path=/", nil)
	s.ListMediaFiles(c)
	var listing struct{ Data proto.GetMediaFilesRsp }
	if err := json.Unmarshal(r.Body.Bytes(), &listing); err != nil || len(listing.Data.Entries) != 0 {
		t.Fatalf("partial file is visible: %s", r.Body)
	}
	mediumOperationMu.Lock()
	err := guardMediaDelete(mediaDirectory)
	mediumOperationMu.Unlock()
	if err == nil {
		t.Fatal("upload parent was not protected")
	}
	rsp = fileChunk(t, s, "archive.zip", testImageUploadID, size, 1, tail)
	if rsp.Code != 0 || !rsp.Data.Complete || rsp.Data.Written != size {
		t.Fatalf("final chunk: %+v", rsp)
	}
	got, err := os.ReadFile(filepath.Join(mediaDirectory, "archive.zip"))
	if err != nil || !bytes.Equal(got, append(first, tail...)) {
		t.Fatalf("completed file differs: %v", err)
	}
	if refs.Load() != 0 || s.jobDone != nil || len(s.fileUploads) != 0 {
		t.Fatal("completed upload retained resources")
	}
	// A late cancellation can never remove a completed user file.
	cancelFileUpload(t, s, testImageUploadID)
	if _, err := os.Stat(filepath.Join(mediaDirectory, "archive.zip")); err != nil {
		t.Fatal("late cancellation removed the completed file")
	}
}

func TestFileChunksRejectChangedPlansAndOutOfOrderRequests(t *testing.T) {
	s, _ := mediaHandlerFixture(t)
	first := bytes.Repeat([]byte{0x11}, int(imageUploadChunkSizeBytes))
	size := imageUploadChunkSizeBytes + 1
	if rsp := fileChunk(t, s, "keep.bin", testImageUploadID, size, 1, []byte{1}); rsp.Code == 0 {
		t.Fatal("upload started with its final chunk")
	}
	if rsp := fileChunk(t, s, "keep.bin", testImageUploadID, size, 0, first); rsp.Code != 0 {
		t.Fatalf("first chunk: %+v", rsp)
	}
	t.Cleanup(func() { cancelFileUpload(t, s, testImageUploadID) })
	if rsp := fileChunk(t, s, "other.bin", testImageUploadID, size, 1, []byte{1}); rsp.Code == 0 {
		t.Fatal("upload destination changed")
	}
	if rsp := fileChunk(t, s, "keep.bin", testImageUploadID, size, 0, first); rsp.Code == 0 {
		t.Fatal("duplicate chunk appended twice")
	}
	if rsp := fileChunk(t, s, "keep.bin", testImageUploadID, size, 1, []byte{2}); rsp.Code != 0 {
		t.Fatalf("valid continuation failed: %+v", rsp)
	}
}

func TestFileChunkCancellationAndExpiryRemoveOnlyPartialFile(t *testing.T) {
	for _, expire := range []bool{false, true} {
		t.Run(fmt.Sprintf("expire=%v", expire), func(t *testing.T) {
			s, refs := mediaHandlerFixture(t)
			keep := filepath.Join(mediaDirectory, "keep.txt")
			if err := os.WriteFile(keep, []byte("existing"), 0o644); err != nil {
				t.Fatal(err)
			}
			first := bytes.Repeat([]byte{0x22}, int(imageUploadChunkSizeBytes))
			if rsp := fileChunk(t, s, "partial.txt", testImageUploadID, imageUploadChunkSizeBytes+1, 0, first); rsp.Code != 0 {
				t.Fatalf("first chunk: %+v", rsp)
			}
			if expire {
				upload := s.fileUploads[testImageUploadID]
				s.expireFileUpload(upload, upload.generation-1)
				if refs.Load() != 1 {
					t.Fatal("stale timeout released a newer chunk")
				}
				s.expireFileUpload(upload, upload.generation)
			} else {
				cancelFileUpload(t, s, testImageUploadID)
			}
			entries, err := os.ReadDir(mediaDirectory)
			if err != nil || len(entries) != 1 || entries[0].Name() != "keep.txt" || refs.Load() != 0 || s.jobDone != nil {
				t.Fatalf("cleanup retained partial data or locks: %v, %v", entries, err)
			}
		})
	}
}

func TestFileChunksPreserveExistingFilesAndSupportEmptyFiles(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	path := filepath.Join(mediaDirectory, "keep.txt")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if rsp := fileChunk(t, s, "keep.txt", testImageUploadID, 3, 0, []byte("new")); rsp.Code == 0 {
		t.Fatal("existing file was replaced")
	}
	got, _ := os.ReadFile(path)
	if string(got) != "existing" || refs.Load() != 0 {
		t.Fatal("failed upload changed the destination or retained a lock")
	}
	if rsp := fileChunk(t, s, "empty.txt", testImageUploadID, 0, 0, nil); rsp.Code != 0 || !rsp.Data.Complete {
		t.Fatalf("empty file: %+v", rsp)
	}
	if info, err := os.Stat(filepath.Join(mediaDirectory, "empty.txt")); err != nil || info.Size() != 0 {
		t.Fatalf("empty file missing: %v", err)
	}
}

func TestFileChunksRejectLengthsBeforeCreatingFileAndUninitializedStorage(t *testing.T) {
	s, refs := mediaHandlerFixture(t)
	if rsp := fileChunk(t, s, "wrong.txt", testImageUploadID, 3, 0, []byte("long")); rsp.Code == 0 {
		t.Fatal("oversized chunk accepted")
	}
	sharedDiskInitialized = func() bool { return false }
	if rsp := fileChunk(t, s, "new.txt", testImageUploadID, 3, 0, []byte("new")); rsp.Code != codeStorageDisabled {
		t.Fatalf("uninitialized disk: %+v", rsp)
	}
	entries, _ := os.ReadDir(mediaDirectory)
	if len(entries) != 0 || refs.Load() != 0 || s.jobDone != nil {
		t.Fatal("rejected request wrote a file or retained ownership")
	}
}
