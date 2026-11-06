package storage

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"NanoKVM-Server/storageguard"
	"github.com/gin-gonic/gin"
)

func allowTestStorageAccess(t *testing.T) {
	t.Helper()
	original := acquireStorageAccess
	acquireStorageAccess = func(bool) (func(), error) { return func() {}, nil }
	t.Cleanup(func() { acquireStorageAccess = original })
}

func TestReadOnlyStorageRejectsLocalMutations(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, handler := range []struct {
		name string
		run  func(*Service, *gin.Context)
	}{
		{"upload", (*Service).UploadImage},
		{"cancel upload", (*Service).CancelImageUpload},
		{"download", (*Service).DownloadImage},
	} {
		t.Run(handler.name, func(t *testing.T) {
			setDownloadHandlerDependencies(t, testDownloadPolicy(100), &http.Client{})
			acquireStorageAccess = func(write bool) (func(), error) {
				if !write {
					t.Fatal("mutation requested a read-only lease")
				}
				return nil, storageguard.ErrReadOnly
			}
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/?uploadId="+testImageUploadID+"&filename=image.iso&chunkSize=8388608&chunkIndex=0&totalChunks=1", strings.NewReader(`{"file":"https://example.com/image.iso"}`))
			ctx.Request.Header.Set("Content-Type", "application/json")
			handler.run(&Service{}, ctx)
			if !strings.Contains(recorder.Body.String(), storageguard.ErrReadOnly.Error()) {
				t.Fatal(recorder.Body.String())
			}
		})
	}
}

func TestDeleteImageDoesNotTouchStorageWhenBusy(t *testing.T) {
	setupDeleteImageHandlerTest(t)
	deleteImageActionRequiresChallenge = func() bool { return false }
	acquireStorageAccess = func(bool) (func(), error) { return nil, storageguard.ErrBusy }
	resolveDeleteImagePath = func(string) (string, bool) { t.Fatal("busy storage was accessed"); return "", false }
	response := performDeleteImageRequest(t, "/data/image.iso", "")
	if response.Msg != storageguard.ErrBusy.Error() {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestImageEnabledRejectsReadOnlyStorageWithoutCreatingProbe(t *testing.T) {
	allowTestStorageAccess(t)
	acquireStorageAccess = func(bool) (func(), error) { return nil, errors.New("read only") }
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	(&Service{}).ImageEnabled(ctx)
	if !strings.Contains(recorder.Body.String(), `"enabled":false`) {
		t.Fatal(recorder.Body.String())
	}
}

func TestDownloadCompletionReleasesStorageBeforeAcknowledgingCancellation(t *testing.T) {
	service := &Service{}
	operation, err := service.beginImageDownload(acquireTestDownloadLease(t))
	if err != nil {
		t.Fatal(err)
	}
	released := 0
	operation.releaseStorage = func() { released++ }
	operation.pause()
	if released != 0 {
		t.Fatal("paused download released storage")
	}
	service.finishImageDownload(operation)
	<-operation.done
	if released != 1 {
		t.Fatal("cancellation acknowledged before storage was released")
	}
	service.finishImageDownload(operation)
	if released != 1 {
		t.Fatal("storage released more than once")
	}
}
