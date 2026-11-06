//go:build !linux || !cgo

package common

type AudioPlayback struct{}

func OpenAudioPlayback() (*AudioPlayback, error) {
	return nil, newAudioPlaybackError(int(AudioPlaybackErrUnavailable))
}

func (p *AudioPlayback) WriteOpus(frame []byte, timeoutMS uint32, flags AudioPlaybackFlags) error {
	if p == nil {
		return newAudioPlaybackError(int(AudioPlaybackErrInvalidState))
	}
	if len(frame) == 0 || uint64(len(frame)) > uint64(AudioPlaybackMaxOpusPacketBytes) ||
		timeoutMS == 0 || timeoutMS > AudioPlaybackMaxTimeoutMS ||
		flags&^AudioPlaybackFlagValidMask != 0 {
		return newAudioPlaybackError(int(AudioPlaybackErrInvalidArgument))
	}
	return newAudioPlaybackError(int(AudioPlaybackErrInvalidState))
}

func (p *AudioPlayback) Close() {}
