package webrtc

import (
	"github.com/pion/webrtc/v4/pkg/media"
	log "github.com/sirupsen/logrus"
)

func (t *Track) writeVideoSample(sample media.Sample) (err error) {
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
		log.Errorf("failed to write h265 video: %s", err)
	}
}

func (t *Track) writeAudio(sample media.Sample) {
	err := t.audio.WriteSample(sample)
	if err != nil {
		log.Errorf("failed to send h265 audio: %s", err)
	}
}
