package webrtc

import (
	"encoding/json"
	"errors"
	"sync"
	"time"

	"NanoKVM-Server/service/media"
	"github.com/gorilla/websocket"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"
)

const signalingWriteTimeout = 5 * time.Second

func NewClient(ws *websocket.Conn, videoConn, audioConn *webrtc.PeerConnection) *Client {
	return NewClientWithCapabilities(ws, videoConn, audioConn, media.Capabilities{SpeakerReceive: true, MicSend: true, VideoReceive: true})
}

func NewClientWithCapabilities(ws *websocket.Conn, videoConn, audioConn *webrtc.PeerConnection, capabilities media.Capabilities) *Client {
	return &Client{
		ws:           ws,
		video:        videoConn,
		audio:        audioConn,
		capabilities: capabilities,
		mutex:        sync.Mutex{},
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

func (c *Client) AddTrack() error {
	var (
		videoTrack      *webrtc.TrackLocalStaticRTP
		videoPacketizer rtp.Packetizer
		audioTrack      *webrtc.TrackLocalStaticSample
		err             error
	)

	if c.capabilities.VideoReceive {
		videoTrack, err = webrtc.NewTrackLocalStaticRTP(
			webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264}, "video", "pion-video")
		if err != nil {
			return err
		}
		videoPacketizer = rtp.NewPacketizer(1200, 100, 0x1234ABCD, &codecs.H264Payloader{}, rtp.NewRandomSequencer(), 90000)
		if videoPacketizer == nil {
			return errors.New("failed to create video packetizer")
		}
		videoSender, err := c.video.AddTrack(videoTrack)
		if err != nil {
			return err
		}
		go startRTCPReader(videoSender, true)
	}

	if c.capabilities.SpeakerReceive {
		audioTrack, err = webrtc.NewTrackLocalStaticSample(
			webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "audio", "pion-audio")
		if err != nil {
			return err
		}
		audioSender, err := c.audio.AddTrack(audioTrack)
		if err != nil {
			return err
		}
		go startRTCPReader(audioSender, false)
	}

	track := &Track{
		videoPacketizer: videoPacketizer,
		video:           videoTrack,
		audio:           audioTrack,
	}

	c.mutex.Lock()
	c.track = track
	c.mutex.Unlock()

	return nil
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
