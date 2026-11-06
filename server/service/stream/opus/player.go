package opus

import (
	"context"
	"errors"
	"sync"
	"time"

	"NanoKVM-Server/common"

	log "github.com/sirupsen/logrus"
)

const (
	defaultQueueCapacity  = 6
	defaultWriteTimeout   = 100 * time.Millisecond
	defaultRetryInitial   = 100 * time.Millisecond
	defaultRetryMaximum   = 2 * time.Second
	defaultErrorLogPeriod = 5 * time.Second
)

var (
	// ErrAudioSessionInactive means this TrackRemote no longer owns playback.
	ErrAudioSessionInactive = errors.New("audio input session is inactive")
	errInvalidOpusFrame     = errors.New("invalid Opus frame")
)

// OpusSink is the synchronous playback boundary owned by the writer goroutine.
// Implementations must allow Close followed by another Open.
type OpusSink interface {
	Open() error
	WriteOpus(frame []byte, timeout time.Duration, discontinuity bool) error
	Close()
}

type opusSinkFactory func() OpusSink
type backoffWaitFunc func(context.Context, time.Duration) bool

type audioPlayerConfig struct {
	queueCapacity  int
	writeTimeout   time.Duration
	retryInitial   time.Duration
	retryMaximum   time.Duration
	errorLogPeriod time.Duration
	waitBackoff    backoffWaitFunc
}

func defaultAudioPlayerConfig() audioPlayerConfig {
	return audioPlayerConfig{
		queueCapacity:  defaultQueueCapacity,
		writeTimeout:   defaultWriteTimeout,
		retryInitial:   defaultRetryInitial,
		retryMaximum:   defaultRetryMaximum,
		errorLogPeriod: defaultErrorLogPeriod,
		waitBackoff:    waitForBackoff,
	}
}

type libkvmOpusSink struct {
	playback *common.AudioPlayback
}

func newLibkvmOpusSink() OpusSink {
	return &libkvmOpusSink{}
}

func (s *libkvmOpusSink) Open() error {
	if s.playback != nil {
		return nil
	}

	playback, err := common.OpenAudioPlayback()
	if err != nil {
		return err
	}
	s.playback = playback
	return nil
}

func (s *libkvmOpusSink) WriteOpus(
	frame []byte,
	timeout time.Duration,
	discontinuity bool,
) error {
	if s.playback == nil {
		return &common.AudioPlaybackError{Code: common.AudioPlaybackErrInvalidState}
	}

	timeoutMS, ok := durationMilliseconds(timeout)
	if !ok {
		return &common.AudioPlaybackError{Code: common.AudioPlaybackErrInvalidArgument}
	}

	flags := common.AudioPlaybackFlagNone
	if discontinuity {
		flags = common.AudioPlaybackFlagDiscontinuity
	}

	return s.playback.WriteOpus(frame, timeoutMS, flags)
}

func (s *libkvmOpusSink) Close() {
	if s.playback == nil {
		return
	}

	s.playback.Close()
	s.playback = nil
}

func durationMilliseconds(duration time.Duration) (uint32, bool) {
	if duration <= 0 {
		return 0, false
	}

	milliseconds := (duration + time.Millisecond - 1) / time.Millisecond
	if milliseconds <= 0 || milliseconds > time.Duration(common.AudioPlaybackMaxTimeoutMS) {
		return 0, false
	}

	return uint32(milliseconds), true
}

type queuedOpusFrame struct {
	data          []byte
	discontinuity bool
}

// opusFrameQueue keeps only a small amount of current audio. Its mutex makes
// dropping the oldest frame and applying discontinuity to the next pop atomic.
type opusFrameQueue struct {
	mutex                sync.Mutex
	frames               []queuedOpusFrame
	capacity             int
	discontinuityPending bool
	notify               chan struct{}
}

func newOpusFrameQueue(capacity int) *opusFrameQueue {
	return &opusFrameQueue{
		frames:   make([]queuedOpusFrame, 0, capacity),
		capacity: capacity,
		notify:   make(chan struct{}, 1),
	}
}

func (q *opusFrameQueue) push(frame []byte) bool {
	q.mutex.Lock()
	dropped := len(q.frames) == q.capacity
	if dropped {
		copy(q.frames, q.frames[1:])
		q.frames[len(q.frames)-1] = queuedOpusFrame{}
		q.frames = q.frames[:len(q.frames)-1]
		q.discontinuityPending = true
	}
	q.frames = append(q.frames, queuedOpusFrame{data: frame})
	q.mutex.Unlock()

	select {
	case q.notify <- struct{}{}:
	default:
	}

	return dropped
}

func (q *opusFrameQueue) pop(ctx context.Context) (queuedOpusFrame, bool) {
	for {
		q.mutex.Lock()
		if ctx.Err() != nil {
			q.mutex.Unlock()
			return queuedOpusFrame{}, false
		}
		if len(q.frames) != 0 {
			frame := q.frames[0]
			copy(q.frames, q.frames[1:])
			q.frames[len(q.frames)-1] = queuedOpusFrame{}
			q.frames = q.frames[:len(q.frames)-1]
			if q.discontinuityPending {
				frame.discontinuity = true
				q.discontinuityPending = false
			}
			q.mutex.Unlock()
			return frame, true
		}
		q.mutex.Unlock()

		select {
		case <-ctx.Done():
			return queuedOpusFrame{}, false
		case <-q.notify:
		}
	}
}

func (q *opusFrameQueue) clear(markDiscontinuity bool) {
	q.mutex.Lock()
	for index := range q.frames {
		q.frames[index] = queuedOpusFrame{}
	}
	q.frames = q.frames[:0]
	if markDiscontinuity {
		q.discontinuityPending = true
	}
	q.mutex.Unlock()
}

func (q *opusFrameQueue) markDiscontinuity() {
	q.mutex.Lock()
	q.discontinuityPending = true
	q.mutex.Unlock()
}

type AudioInputPlayer struct {
	transition     sync.Mutex
	mutex          sync.Mutex
	active         bool
	playbackPaused bool
	ownerToken     uint64
	nextToken      uint64
	queue          *opusFrameQueue
	cancel         context.CancelFunc
	done           chan struct{}
	sinkFactory    opusSinkFactory
	config         audioPlayerConfig
}

// AudioInputSession is the ownership token for one TrackRemote. The newest
// session preempts the previous one; stale sessions cannot enqueue or stop the
// current playback writer.
type AudioInputSession struct {
	player   *AudioInputPlayer
	token    uint64
	stopOnce sync.Once
}

var (
	audioInputPlayer     *AudioInputPlayer
	audioInputPlayerOnce sync.Once
)

func GetAudioInputPlayer() *AudioInputPlayer {
	audioInputPlayerOnce.Do(func() {
		audioInputPlayer = newAudioInputPlayer(newLibkvmOpusSink, defaultAudioPlayerConfig())
	})
	return audioInputPlayer
}

func newAudioInputPlayer(factory opusSinkFactory, config audioPlayerConfig) *AudioInputPlayer {
	if config.queueCapacity <= 0 {
		config.queueCapacity = defaultQueueCapacity
	}
	if config.writeTimeout <= 0 {
		config.writeTimeout = defaultWriteTimeout
	}
	if config.retryInitial <= 0 {
		config.retryInitial = defaultRetryInitial
	}
	if config.retryMaximum < config.retryInitial {
		config.retryMaximum = config.retryInitial
	}
	if config.errorLogPeriod <= 0 {
		config.errorLogPeriod = defaultErrorLogPeriod
	}
	if config.waitBackoff == nil {
		config.waitBackoff = waitForBackoff
	}

	return &AudioInputPlayer{
		sinkFactory: factory,
		config:      config,
	}
}

// StartSession gives the newest TrackRemote exclusive ownership. Takeover
// invalidates the previous token, stops its bounded writer, and creates a new
// libkvm AudioPlayer session with an independent Opus decoder.
func (p *AudioInputPlayer) StartSession() (*AudioInputSession, error) {
	p.transition.Lock()
	defer p.transition.Unlock()

	p.mutex.Lock()
	if p.sinkFactory == nil {
		p.mutex.Unlock()
		return nil, errors.New("audio input player has no Opus sink")
	}

	p.mutex.Unlock()

	preempted := p.stopSessionLocked(0, true)

	p.mutex.Lock()
	token := p.nextSessionTokenLocked()
	paused := p.playbackPaused
	p.active = true
	p.ownerToken = token
	var ctx context.Context
	var queue *opusFrameQueue
	var done chan struct{}
	if !paused {
		ctx, p.cancel = context.WithCancel(context.Background())
		queue = newOpusFrameQueue(p.config.queueCapacity)
		done = make(chan struct{})
		p.queue = queue
		p.done = done
	}
	p.mutex.Unlock()

	if !paused {
		go p.runWriter(ctx, queue, done, false)
	}
	if preempted {
		log.Debugf("audio input session %d took ownership", token)
	} else {
		log.Debugf("audio input session %d started", token)
	}
	if paused {
		log.Debugf("audio input session %d is waiting for playback to resume", token)
	}
	return &AudioInputSession{player: p, token: token}, nil
}

func (p *AudioInputPlayer) nextSessionTokenLocked() uint64 {
	p.nextToken++
	if p.nextToken == 0 {
		p.nextToken++
	}
	return p.nextToken
}

// Enqueue copies one complete Opus frame into the latest-wins queue. It never
// waits for Opus decoding, PCM writes, device recovery, or reconnect backoff.
func (s *AudioInputSession) Enqueue(opusData []byte) error {
	if len(opusData) == 0 || uint64(len(opusData)) > uint64(common.AudioPlaybackMaxOpusPacketBytes) {
		return errInvalidOpusFrame
	}
	if s == nil || s.player == nil || s.token == 0 {
		return ErrAudioSessionInactive
	}

	p := s.player
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if !p.active || p.ownerToken != s.token {
		return ErrAudioSessionInactive
	}
	if p.playbackPaused {
		return nil
	}
	if p.queue == nil {
		return ErrAudioSessionInactive
	}

	frame := append([]byte(nil), opusData...)
	p.queue.push(frame)
	return nil
}

// Stop only affects the writer still owned by this session token. It is safe to
// call after a newer TrackRemote has taken ownership.
func (s *AudioInputSession) Stop() {
	if s == nil || s.player == nil || s.token == 0 {
		return
	}

	s.stopOnce.Do(func() {
		s.player.stopSession(s.token)
	})
}

func (s *AudioInputSession) IsActive() bool {
	if s == nil || s.player == nil || s.token == 0 {
		return false
	}

	s.player.mutex.Lock()
	defer s.player.mutex.Unlock()
	return s.player.active && s.player.ownerToken == s.token
}

func (p *AudioInputPlayer) stopSession(token uint64) {
	p.transition.Lock()
	defer p.transition.Unlock()
	p.stopSessionLocked(token, false)
}

// PausePlayback synchronously stops the current writer and closes its sink
// without invalidating the TrackRemote session that owns it. Frames received
// while paused are dropped so the receiver stays alive without reopening the
// UAC playback device during a USB gadget rebuild.
func (p *AudioInputPlayer) PausePlayback() {
	if p == nil {
		return
	}

	p.transition.Lock()
	defer p.transition.Unlock()

	p.mutex.Lock()
	p.playbackPaused = true
	cancel := p.cancel
	done := p.done
	queue := p.queue
	if queue != nil {
		queue.clear(false)
	}
	p.queue = nil
	p.cancel = nil
	p.done = nil
	if !p.active {
		p.ownerToken = 0
	}
	p.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// ResumePlayback allows the current TrackRemote session to play again. The
// replacement writer opens its sink lazily on the first new frame, so merely
// completing a gadget rebuild does not reacquire an idle UAC playback device.
func (p *AudioInputPlayer) ResumePlayback() {
	if p == nil {
		return
	}

	p.transition.Lock()
	defer p.transition.Unlock()

	p.mutex.Lock()
	if !p.playbackPaused {
		p.mutex.Unlock()
		return
	}
	p.playbackPaused = false
	if !p.active || p.ownerToken == 0 || p.done != nil {
		p.mutex.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	queue := newOpusFrameQueue(p.config.queueCapacity)
	done := make(chan struct{})
	p.queue = queue
	p.cancel = cancel
	p.done = done
	token := p.ownerToken
	p.mutex.Unlock()

	go p.runWriter(ctx, queue, done, true)
	log.Debugf("audio input session %d playback resumed", token)
}

// stopSessionLocked requires transition to be held. A forced stop is used by
// StartSession to preempt or clean up the previous owner before starting a new
// writer. A token stop becomes a no-op once ownership has changed.
func (p *AudioInputPlayer) stopSessionLocked(token uint64, force bool) bool {
	p.mutex.Lock()
	if (!p.active && p.done == nil) || (!force && p.ownerToken != token) {
		p.mutex.Unlock()
		return false
	}

	stoppedToken := p.ownerToken
	p.active = false
	p.ownerToken = 0
	cancel := p.cancel
	done := p.done
	queue := p.queue
	if queue != nil {
		queue.clear(false)
	}
	p.cancel = nil
	p.mutex.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
		p.clearFinishedSession(done)
	}

	if stoppedToken != 0 {
		log.Debugf("audio input session %d stopped", stoppedToken)
	}
	return true
}

func (p *AudioInputPlayer) IsActive() bool {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.active
}

func (p *AudioInputPlayer) clearFinishedSession(done chan struct{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.done != done {
		return
	}

	if p.queue != nil {
		p.queue.clear(false)
	}
	p.queue = nil
	p.cancel = nil
	p.done = nil
	p.active = false
	p.ownerToken = 0
}

func (p *AudioInputPlayer) runWriter(
	ctx context.Context,
	queue *opusFrameQueue,
	done chan struct{},
	openLazily bool,
) {
	var sink OpusSink
	defer func() {
		if sink != nil {
			sink.Close()
		}
		queue.clear(false)

		p.mutex.Lock()
		if p.done == done {
			p.active = false
			p.cancel = nil
		}
		p.mutex.Unlock()
		close(done)
	}()

	sink = p.sinkFactory()
	if sink == nil {
		log.Error("audio input player Opus sink factory returned nil")
		return
	}

	opened := false
	retryDelay := p.config.retryInitial
	var lastErrorLog time.Time
	var pendingFrame *queuedOpusFrame
	if openLazily {
		frame, ok := queue.pop(ctx)
		if !ok {
			return
		}
		pendingFrame = &frame
	}

	for {
		if !opened {
			if ctx.Err() != nil {
				return
			}

			if err := sink.Open(); err != nil {
				sink.Close()
				queue.clear(true)
				p.logWriterError(&lastErrorLog, "failed to open libkvm audio playback: %v", err)
				if isPermissionError(err) {
					<-ctx.Done()
					return
				}
				if !p.config.waitBackoff(ctx, retryDelay) {
					return
				}
				retryDelay = nextBackoff(retryDelay, p.config.retryMaximum)
				continue
			}

			opened = true
			log.Debug("libkvm audio playback opened")
		}

		var frame queuedOpusFrame
		if pendingFrame != nil {
			frame = *pendingFrame
			pendingFrame = nil
		} else {
			var ok bool
			frame, ok = queue.pop(ctx)
			if !ok {
				return
			}
		}

		err := sink.WriteOpus(frame.data, p.config.writeTimeout, frame.discontinuity)
		if err == nil {
			retryDelay = p.config.retryInitial
			continue
		}

		code, classified := common.AudioPlaybackErrorCodeOf(err)
		if classified && (code == common.AudioPlaybackErrCodec || code == common.AudioPlaybackErrInvalidArgument) {
			queue.markDiscontinuity()
			p.logWriterError(&lastErrorLog, "libkvm rejected an Opus frame: %v", err)
			continue
		}

		p.logWriterError(&lastErrorLog, "libkvm audio playback failed; reopening: %v", err)
		sink.Close()
		opened = false
		queue.clear(true)
		if isPermissionError(err) {
			<-ctx.Done()
			return
		}
		if !p.config.waitBackoff(ctx, retryDelay) {
			return
		}
		retryDelay = nextBackoff(retryDelay, p.config.retryMaximum)
	}
}

func (p *AudioInputPlayer) logWriterError(last *time.Time, format string, args ...any) {
	now := time.Now()
	if !last.IsZero() && now.Sub(*last) < p.config.errorLogPeriod {
		return
	}

	log.Warnf(format, args...)
	*last = now
}

func waitForBackoff(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func nextBackoff(current, maximum time.Duration) time.Duration {
	if current >= maximum {
		return maximum
	}
	if current > maximum/2 {
		return maximum
	}
	return current * 2
}

func isPermissionError(err error) bool {
	code, ok := common.AudioPlaybackErrorCodeOf(err)
	return ok && code == common.AudioPlaybackErrPermission
}
