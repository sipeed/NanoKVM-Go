package webrtc

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"
)

const signalingWriteTimeout = 5 * time.Second

func NewClient(ws *websocket.Conn, videoConn, audioConn *webrtc.PeerConnection) *Client {
	return &Client{
		ws:    ws,
		video: videoConn,
		audio: audioConn,
		mutex: sync.Mutex{},
	}
}

func (c *Client) WriteMessage(event string, data string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	message := &Message{
		Event: event,
		Data:  data,
	}

	_ = c.ws.SetWriteDeadline(time.Now().Add(signalingWriteTimeout))
	if err := c.ws.WriteJSON(message); err != nil {
		log.Errorf("failed to send message %s: %v", event, err)
		return err
	}

	log.Debugf("sent message %s", event)
	return nil
}

func (c *Client) ReadMessage() (*Message, error) {
	_, raw, err := c.ws.ReadMessage()
	if err != nil {
		log.Errorf("failed to read message: %v", err)
		return nil, err
	}

	var message Message
	if err := json.Unmarshal(raw, &message); err != nil {
		log.Errorf("failed to unmarshal message: %v", err)
		return nil, nil
	}

	return &message, nil
}

func (c *Client) AddTrack() (*Track, error) {
	// video track
	videoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH265},
		"video",
		"pion-video",
	)
	if err != nil {
		log.Errorf("failed to create video track: %s", err)
		return nil, err
	}

	videoPacketizer := rtp.NewPacketizer(
		1200,
		100,
		0x1234ABCD,
		&codecs.H265Payloader{},
		rtp.NewRandomSequencer(),
		90000,
	)
	if videoPacketizer == nil {
		err := errors.New("failed to create rtp packetizer")
		log.Error(err)
		return nil, err
	}

	videoSender, err := c.video.AddTrack(videoTrack)
	if err != nil {
		log.Errorf("failed to add video track: %s", err)
		return nil, err
	}
	go startRTCPReader(videoSender, true)

	// audio track
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus},
		"audio",
		"pion-audio",
	)
	if err != nil {
		log.Errorf("failed to create audio track: %s", err)
		return nil, err
	}

	audioSender, err := c.audio.AddTrack(audioTrack)
	if err != nil {
		log.Errorf("failed to add audio track: %s", err)
		return nil, err
	}
	go startRTCPReader(audioSender, false)

	return &Track{
		client:          c,
		videoPacketizer: videoPacketizer,
		video:           videoTrack,
		audio:           audioTrack,
	}, nil
}

// startRTCPReader drains inbound RTCP for a sender. For the video sender
// (isVideo), it parses the feedback and asks the manager to force a keyframe
// when a decoder reports picture loss (PLI) or requests a full intra (FIR),
// so corruption recovers within an RTT instead of waiting for the next GOP.
func startRTCPReader(sender *webrtc.RTPSender, isVideo bool) {
	rtcpBuf := make([]byte, 1500)
	for {
		n, _, err := sender.Read(rtcpBuf)
		if err != nil {
			log.Debugf("RTCP reader error: %v", err)
			return
		}

		if !isVideo {
			continue
		}

		packets, err := rtcp.Unmarshal(rtcpBuf[:n])
		if err != nil {
			log.Debugf("failed to unmarshal RTCP: %v", err)
			continue
		}

		for _, p := range packets {
			switch p.(type) {
			case *rtcp.PictureLossIndication:
				getManager().RequestKeyframe()
			case *rtcp.FullIntraRequest:
				getManager().RequestKeyframe()
			}
		}
	}
}
