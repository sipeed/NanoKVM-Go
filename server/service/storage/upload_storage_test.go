package storage

import (
	"errors"
	"testing"
)

func TestUploadKeepsStorageUntilFinalChunkOrCancellation(t *testing.T) {
	for _, cancel := range []bool{false, true} {
		t.Run(map[bool]string{false: "finish", true: "cancel"}[cancel], func(t *testing.T) {
			allowTestStorageAccess(t)
			acquired, released := 0, 0
			acquireStorageAccess = func(bool) (func(), error) {
				acquired++
				return func() { released++ }, nil
			}
			service := &Service{}
			metadata := imageUploadMetadata{uploadID: testImageUploadID, totalChunks: 2}
			finish, err := service.beginUploadStorage(metadata)
			if err != nil {
				t.Fatal(err)
			}
			finish(true)
			if acquired != 1 || released != 0 {
				t.Fatal("storage released between chunks")
			}
			if cancel {
				service.releaseUploadStorage(metadata.uploadID)
			} else {
				metadata.chunkIndex = 1
				finish, err = service.beginUploadStorage(metadata)
				if err != nil {
					t.Fatal(err)
				}
				finish(true)
			}
			if acquired != 1 || released != 1 {
				t.Fatalf("leases acquired=%d released=%d", acquired, released)
			}
		})
	}
}

func TestUploadTimeoutCannotReleaseAnActiveOrNewerChunk(t *testing.T) {
	allowTestStorageAccess(t)
	released := 0
	acquireStorageAccess = func(bool) (func(), error) { return func() { released++ }, nil }
	service := &Service{}
	metadata := imageUploadMetadata{uploadID: testImageUploadID, totalChunks: 3}
	finish, err := service.beginUploadStorage(metadata)
	if err != nil {
		t.Fatal(err)
	}
	finish(true)
	session := service.upload
	oldGeneration := session.generation
	metadata.chunkIndex++
	finish, err = service.beginUploadStorage(metadata)
	if err != nil {
		t.Fatal(err)
	}
	service.expireUploadStorage(session, oldGeneration)
	service.expireUploadStorage(session, session.generation)
	if released != 0 {
		t.Fatal("active upload expired")
	}
	finish(true)
	service.expireUploadStorage(session, oldGeneration)
	if released != 0 {
		t.Fatal("old timeout released a newer chunk")
	}
	session.timer.Stop()
	service.expireUploadStorage(session, session.generation)
	if released != 1 {
		t.Fatal("abandoned upload did not release storage")
	}
	if _, err := service.beginUploadStorage(metadata); !errors.Is(err, errImageUploadOutOfOrder) {
		t.Fatalf("expired upload resumed after ownership change: %v", err)
	}
}

func TestUploadCancellationDuringChunkCompletionReleasesStorage(t *testing.T) {
	allowTestStorageAccess(t)
	released := false
	acquireStorageAccess = func(bool) (func(), error) { return func() { released = true }, nil }
	service := &Service{}
	metadata := imageUploadMetadata{uploadID: testImageUploadID, totalChunks: 2}
	finish, err := service.beginUploadStorage(metadata)
	if err != nil {
		t.Fatal(err)
	}
	service.releaseUploadStorage(metadata.uploadID)
	if released {
		t.Fatal("cancellation released storage while chunk was active")
	}
	finish(true)
	if !released {
		t.Fatal("cancelled upload retained storage")
	}
}
