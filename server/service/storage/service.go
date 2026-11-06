package storage

import (
	"context"
	"errors"
	"os"
	"sync"

	"NanoKVM-Server/proto"
)

type Service struct {
	lastDownload proto.StatusImageRsp
	jobMutex     sync.Mutex
	jobState     jobState
	jobName      string
	jobLoaded    int64
	jobTotal     int64
	jobMessage   string
	jobDone      chan struct{}
	jobWaiters   []chan struct{}
	downloadMu   sync.Mutex
	download     *imageDownloadOperation
	uploadMu     sync.Mutex
	upload       *imageUploadSession
	fileUploadMu sync.Mutex
	fileUploads  map[string]*mediaFileUpload
}

type imageDownloadOperation struct {
	ctx            context.Context
	cancel         context.CancelFunc
	lease          *downloadLease
	done           chan struct{}
	finishOnce     sync.Once
	pauseMu        sync.Mutex
	paused         bool
	resumeCh       chan struct{}
	releaseStorage func()
}

func (operation *imageDownloadOperation) pause() bool {
	operation.pauseMu.Lock()
	defer operation.pauseMu.Unlock()
	if operation.ctx.Err() != nil {
		return false
	}
	if operation.paused {
		return true
	}

	operation.paused = true
	operation.resumeCh = make(chan struct{})
	return true
}

func (operation *imageDownloadOperation) resume() bool {
	operation.pauseMu.Lock()
	defer operation.pauseMu.Unlock()
	if operation.ctx.Err() != nil {
		return false
	}
	if !operation.paused {
		return true
	}

	operation.paused = false
	close(operation.resumeCh)
	operation.resumeCh = nil
	return true
}

func (operation *imageDownloadOperation) isPaused() bool {
	operation.pauseMu.Lock()
	defer operation.pauseMu.Unlock()
	return operation.paused
}

func (operation *imageDownloadOperation) waitUntilResumed() error {
	for {
		operation.pauseMu.Lock()
		if !operation.paused {
			err := operation.ctx.Err()
			operation.pauseMu.Unlock()
			return err
		}
		resumeCh := operation.resumeCh
		operation.pauseMu.Unlock()

		select {
		case <-resumeCh:
		case <-operation.ctx.Done():
			return operation.ctx.Err()
		}
	}
}

func NewService() *Service {
	_ = os.Remove(sentinelPath)
	return &Service{jobState: jobStateIdle}
}

func (s *Service) beginImageDownload(lease *downloadLease) (*imageDownloadOperation, error) {
	ctx, cancel := context.WithCancel(context.Background())
	operation := &imageDownloadOperation{
		ctx:    ctx,
		cancel: cancel,
		lease:  lease,
		done:   make(chan struct{}),
	}

	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	if s.download != nil {
		cancel()
		return nil, errors.New("image download is already active")
	}
	s.download = operation
	s.lastDownload = newImageDownloadStatus("idle", "", "")
	return operation, nil
}

func (s *Service) currentImageDownload() *imageDownloadOperation {
	s.downloadMu.Lock()
	defer s.downloadMu.Unlock()
	return s.download
}

func (s *Service) finishImageDownload(operation *imageDownloadOperation) {
	operation.finishOnce.Do(func() {
		operation.cancel()
		releaseDownloadLease(operation.lease)
		if operation.releaseStorage != nil {
			operation.releaseStorage()
		}

		s.downloadMu.Lock()
		if s.download == operation {
			s.download = nil
		}
		s.downloadMu.Unlock()
		close(operation.done)
	})
}
