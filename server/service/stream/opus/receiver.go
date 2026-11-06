package opus

import (
	"errors"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
	log "github.com/sirupsen/logrus"
)

type rtpReadFunc func([]byte) (int, error)

// ReceiveMicrophoneTrack consumes one browser microphone TrackRemote using the
// ownership-scoped session acquired synchronously by StartMicStream.
func ReceiveMicrophoneTrack(track *webrtc.TrackRemote, session *AudioInputSession) {
	if track == nil {
		session.Stop()
		return
	}

	receiveMicrophonePackets(func(buffer []byte) (int, error) {
		n, _, err := track.Read(buffer)
		return n, err
	}, session)
}

func receiveMicrophonePackets(read rtpReadFunc, session *AudioInputSession) {
	defer session.Stop()

	sampleBuilder := samplebuilder.New(20, &Packet{}, 48000)
	buffer := make([]byte, 1500)
	var dumpSequence uint64

	for {
		n, err := read(buffer)
		if err != nil {
			log.Debugf("audio track read error: %s", err)
			break
		}

		packet, err := unmarshalOwnedPacket(buffer[:n])
		if err != nil {
			log.Debugf("failed to unmarshal RTP packet: %s", err)
			continue
		}

		sampleBuilder.Push(packet)
		for {
			sample := sampleBuilder.Pop()
			if sample == nil {
				break
			}
			dumpSequence++
			microphoneOpusDump.write(
				"NANOKVM_MIC_OPUS_DUMP",
				dumpSequence,
				sample.Duration,
				sample.Data,
			)

			if err := session.Enqueue(sample.Data); err != nil {
				if errors.Is(err, ErrAudioSessionInactive) {
					log.Debug("audio microphone session ended")
					return
				}
				log.Debugf("failed to enqueue microphone audio: %s", err)
			}
		}
	}

	log.Debug("audio microphone track ended")
}

func unmarshalOwnedPacket(raw []byte) (*rtp.Packet, error) {
	packet := &rtp.Packet{}
	if err := packet.Unmarshal(raw); err != nil {
		return nil, err
	}
	// rtp.Packet.Unmarshal keeps Payload backed by raw. SampleBuilder may retain
	// a packet until a later RTP packet arrives, while the next track.Read reuses
	// and overwrites raw. Own the payload before handing it off.
	packet.Payload = append([]byte(nil), packet.Payload...)
	return packet, nil
}
