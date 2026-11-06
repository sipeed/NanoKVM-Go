package storage

import (
	"context"
	"errors"
)

type jobState string

const (
	jobStateIdle       jobState = "idle"
	jobStateInProgress jobState = "in_progress"
	jobStateSuccess    jobState = "success"
	jobStateFailed     jobState = "failed"
)

// maxQueuedJobs bounds the queue: the transfers hold their HTTP connections
// open while they wait, so an unbounded queue would let one client hoard them.
const maxQueuedJobs = 8

var errTooManyJobs = errors.New("too many transfers are already queued")

// activeJobTarget returns the path the running upload writes, if any.
func (s *Service) activeJobTarget() (string, bool) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()

	return s.jobName, s.jobDone != nil
}

// beginJob claims the single media job slot, queueing behind whatever is
// running instead of refusing: the user asked for a file to be written, and
// waiting their turn is what they expect. The queue is first come, first
// served, and a request whose client goes away leaves it.
//
// The transfer lock file that coordinates with the image downloader and the
// application updater is deliberately not taken here. Writing into an image
// only touches blocks that image already owns, so it neither races the
// downloader nor holds it up.
func (s *Service) beginJob(ctx context.Context, name string) (chan struct{}, error) {
	var waiter chan struct{}
	for {
		if err := ctx.Err(); err != nil {
			if waiter != nil {
				s.abandonWaiter(waiter)
			}
			return nil, err
		}
		s.jobMutex.Lock()

		first := len(s.jobWaiters) > 0 && s.jobWaiters[0] == waiter
		if s.jobDone == nil && (len(s.jobWaiters) == 0 || first) {
			if first {
				s.jobWaiters = s.jobWaiters[1:]
			}
			done := make(chan struct{})
			s.jobDone = done
			s.jobState = jobStateInProgress
			s.jobName = name
			s.jobLoaded = 0
			s.jobTotal = 0
			s.jobMessage = ""
			s.jobMutex.Unlock()

			return done, nil
		}

		if len(s.jobWaiters) >= maxQueuedJobs {
			s.jobMutex.Unlock()
			return nil, errTooManyJobs
		}

		waiter = make(chan struct{})
		s.jobWaiters = append(s.jobWaiters, waiter)
		s.jobMutex.Unlock()

		select {
		case <-waiter:
			// Our turn; the loop tries to claim the slot.
		case <-ctx.Done():
			s.abandonWaiter(waiter)
			return nil, ctx.Err()
		}
	}
}

// abandonWaiter drops a waiter whose client went away. A waiter that is still
// queued only has to leave the queue; one that was already handed the turn
// (finishJob closed its channel) has to pass it on, or the transfers waiting
// behind it would sit there until their own clients gave up too.
func (s *Service) abandonWaiter(waiter chan struct{}) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()

	for i, queued := range s.jobWaiters {
		if queued == waiter {
			s.jobWaiters = append(s.jobWaiters[:i], s.jobWaiters[i+1:]...)
			if i == 0 && s.jobDone == nil && len(s.jobWaiters) > 0 {
				close(s.jobWaiters[0])
			}
			return
		}
	}
}

func (s *Service) setJobProgress(done chan struct{}, loaded, total int64) {
	s.jobMutex.Lock()
	defer s.jobMutex.Unlock()

	if s.jobDone == done {
		s.jobLoaded = loaded
		s.jobTotal = total
	}
}

func (s *Service) finishJob(done chan struct{}, state jobState, message string) {
	s.jobMutex.Lock()
	if s.jobDone != done {
		s.jobMutex.Unlock()
		return
	}

	s.jobDone = nil
	s.jobState = state
	s.jobLoaded = 0
	s.jobTotal = 0
	s.jobMessage = message

	// The slot goes to whoever queued first.
	if len(s.jobWaiters) > 0 {
		close(s.jobWaiters[0])
	}
	s.jobMutex.Unlock()

	close(done)
}
