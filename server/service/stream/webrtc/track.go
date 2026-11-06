package webrtc

import (
	"github.com/pion/webrtc/v4/pkg/media"
	log "github.com/sirupsen/logrus"
)

func (t *Track) writeVideoSample(sample media.Sample) (err error) {
	if t.video == nil || t.videoPacketizer == nil {
		return nil
	}
	samples := uint32(sample.Duration.Seconds() * 90000)
	packets := t.videoPacketizer.Packetize(sample.Data, samples)

	for _, p := range packets {
		err = t.video.WriteRTP(p)
		if err != nil {
			log.Errorf("failed to write RTP: %v", err)
			return err
		}
	}

	return nil
}

func (t *Track) writeVideo(sample media.Sample) {
	err := t.writeVideoSample(sample)
	if err != nil {
		log.Errorf("failed to write h264 video: %s", err)
	}
}

func (t *Track) writeAudio(sample media.Sample) {
	if t.audio == nil {
		return
	}
	err := t.audio.WriteSample(sample)
	if err != nil {
		log.Errorf("failed to send h264 audio: %s", err)
	}
}
