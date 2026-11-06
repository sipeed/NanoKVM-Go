package common

import (
	"sync"
	"testing"
)

func TestKvmVisionStreamTypeConcurrentAccess(t *testing.T) {
	t.Parallel()

	vision := newKvmVision()
	start := make(chan struct{})
	var workers sync.WaitGroup

	for mode := uint8(STREAM_TYPE_H264_WEBRTC); mode <= STREAM_TYPE_H265_DIRECT; mode++ {
		mode := mode
		workers.Add(2)

		go func() {
			defer workers.Done()
			<-start
			for range 1000 {
				vision.SetStreamType(mode)
			}
		}()

		go func() {
			defer workers.Done()
			<-start
			for range 1000 {
				if got := vision.StreamType(); got > STREAM_TYPE_H265_DIRECT {
					t.Errorf("unexpected stream type: %d", got)
					return
				}
			}
		}()
	}

	close(start)
	workers.Wait()

	vision.SetStreamType(STREAM_TYPE_H264_DIRECT)
	if got := vision.StreamType(); got != STREAM_TYPE_H264_DIRECT {
		t.Fatalf("stream type = %d, want %d", got, STREAM_TYPE_H264_DIRECT)
	}
}
