package webrtc

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"NanoKVM-Server/service/authsession"
	"NanoKVM-Server/service/media"
	"NanoKVM-Server/service/stream"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	pionmedia "github.com/pion/webrtc/v4/pkg/media"
	log "github.com/sirupsen/logrus"
)

// ConnectMediaAudio is the capability-scoped, audio-only WebRTC data plane.
// Signaling messages are {event:"offer"|"candidate", data:"<json>"}; replies
// are {event:"answer"|"candidate", data:"<json>"}.
func ConnectMediaAudio(c *gin.Context) {
	value, ok := c.Get("mediaSession")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "media session required"})
		return
	}
	session, ok := value.(media.Session)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid media session"})
		return
	}

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	unregister := authsession.Register(func(code int, text string) {
		_ = ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(2*time.Second))
		_ = ws.Close()
	})
	defer unregister()

	pc, err := createPeerConnection(createICEServers(), nil, false)
	if err != nil {
		return
	}
	defer pc.Close()

	var speaker *webrtc.TrackLocalStaticSample
	if session.Capabilities.SpeakerReceive {
		speaker, err = webrtc.NewTrackLocalStaticSample(
			webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus}, "speaker", "nanokvm-speaker")
		if err != nil {
			return
		}
		sender, err := pc.AddTrack(speaker)
		if err != nil {
			return
		}
		go startRTCPReader(sender, false)
	}

	done := make(chan struct{})
	defer close(done)
	if speaker != nil {
		go writeMediaSpeaker(done, pc, speaker)
	}
	if session.Capabilities.MicSend {
		pc.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
			if track.Kind() == webrtc.RTPCodecTypeAudio {
				getManager().StartMicStream(track)
			}
		})
	}
	pc.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		data, err := json.Marshal(candidate.ToJSON())
		if err == nil {
			_ = ws.WriteJSON(Message{Event: "candidate", Data: string(data)})
		}
	})

	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			return
		}
		var message Message
		if json.Unmarshal(raw, &message) != nil {
			continue
		}
		switch message.Event {
		case "offer":
			if err := handleMediaOffer(pc, ws, message.Data); err != nil {
				log.Debugf("media audio offer: %v", err)
				return
			}
		case "candidate":
			var candidate webrtc.ICECandidateInit
			if json.Unmarshal([]byte(message.Data), &candidate) == nil {
				_ = pc.AddICECandidate(candidate)
			}
		}
	}
}

func handleMediaOffer(pc *webrtc.PeerConnection, ws *websocket.Conn, raw string) error {
	if pc.SignalingState() != webrtc.SignalingStateStable {
		return errors.New("signaling is not stable")
	}
	var offer webrtc.SessionDescription
	if err := json.Unmarshal([]byte(raw), &offer); err != nil {
		return err
	}
	if err := pc.SetRemoteDescription(offer); err != nil {
		return err
	}
	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err := pc.SetLocalDescription(answer); err != nil {
		return err
	}
	data, err := json.Marshal(answer)
	if err != nil {
		return err
	}
	return ws.WriteJSON(Message{Event: "answer", Data: string(data)})
}

func writeMediaSpeaker(done <-chan struct{}, pc *webrtc.PeerConnection, speaker *webrtc.TrackLocalStaticSample) {
	subscription := stream.DefaultAudioBroker().Subscribe()
	defer subscription.Close()
	for {
		select {
		case <-done:
			return
		case frame := <-subscription.C:
			if pc.ConnectionState() == webrtc.PeerConnectionStateClosed {
				return
			}
			_ = speaker.WriteSample(pionmedia.Sample{Data: frame, Duration: 20 * time.Millisecond})
		}
	}
}
