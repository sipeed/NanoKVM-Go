package storage

import "time"

// Keep ownership between HTTP chunks, including short network interruptions.
// An abandoned browser must not retain the storage lock forever. Once expired,
// an upload must start again, rather than append after USB may have modified it.
const uploadStorageIdleTimeout = 2 * time.Minute

type imageUploadSession struct {
	id         string
	release    func()
	active     bool
	cancelled  bool
	generation uint64
	timer      *time.Timer
}

func (s *Service) beginUploadStorage(metadata imageUploadMetadata) (func(bool), error) {
	s.uploadMu.Lock()
	defer s.uploadMu.Unlock()
	if s.upload != nil && (s.upload.id != metadata.uploadID || s.upload.active) {
		return nil, errImageUploadBusy
	}
	if s.upload == nil {
		if metadata.chunkIndex != 0 {
			return nil, errImageUploadOutOfOrder
		}
		release, err := acquireStorageAccess(true)
		if err != nil {
			return nil, err
		}
		s.upload = &imageUploadSession{id: metadata.uploadID, release: release}
	}
	session := s.upload
	if session.timer != nil {
		session.timer.Stop()
	}
	session.active = true
	session.generation++
	return func(success bool) {
		s.uploadMu.Lock()
		defer s.uploadMu.Unlock()
		if s.upload != session {
			return
		}
		session.active = false
		if !success || metadata.finalChunk() || session.cancelled {
			session.release()
			s.upload = nil
			return
		}
		generation := session.generation
		session.timer = time.AfterFunc(uploadStorageIdleTimeout, func() {
			s.expireUploadStorage(session, generation)
		})
	}, nil
}

func (s *Service) expireUploadStorage(session *imageUploadSession, generation uint64) {
	s.uploadMu.Lock()
	defer s.uploadMu.Unlock()
	if s.upload == session && !session.active && session.generation == generation {
		session.release()
		s.upload = nil
	}
}

func (s *Service) releaseUploadStorage(uploadID string) {
	s.uploadMu.Lock()
	defer s.uploadMu.Unlock()
	if s.upload != nil && s.upload.id == uploadID {
		if s.upload.active {
			s.upload.cancelled = true
			return
		}
		if s.upload.timer != nil {
			s.upload.timer.Stop()
		}
		s.upload.release()
		s.upload = nil
	}
}
