package webrtc

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/service/authsession"
	"NanoKVM-Server/service/media"
	"NanoKVM-Server/service/stream"
	"NanoKVM-Server/service/stream/playoutdelay"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/dtls/v3"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	log "github.com/sirupsen/logrus"
)

var (
	upgrader      = websocket.Upgrader{CheckOrigin: utils.CheckWebSocketOrigin}
	globalManager *WebRTCManager
	managerOnce   sync.Once
)

func getManager() *WebRTCManager {
	managerOnce.Do(func() {
		globalManager = NewWebRTCManager()
		modeChanges := stream.DefaultVideoModeBroker().Subscribe()
		go globalManager.watchVideoMode(modeChanges)
	})
	return globalManager
}

func Connect(c *gin.Context) {
	// create WebSocket connection
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Errorf("failed to create h264 websocket: %s", err)
		return
	}
	defer func() {
		_ = wsConn.Close()
		log.Debugf("h264 websocket disconnected: %s", c.ClientIP())
	}()
	log.Debugf("h264 websocket connected: %s", c.ClientIP())
	unregister := authsession.Register(func(code int, text string) {
		_ = wsConn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, text), time.Now().Add(2*time.Second))
		_ = wsConn.Close()
	})
	defer unregister()

	var zeroTime time.Time
	_ = wsConn.SetReadDeadline(zeroTime)

	// create video connection
	iceServers := createICEServers()

	mediaEngine, err := createMediaEngine()
	if err != nil {
		log.Errorf("failed to create h264 media engine: %s", err)
		return
	}

	videoConn, err := createPeerConnection(iceServers, mediaEngine, true)
	if err != nil {
		log.Errorf("failed to create h264 video peer connection: %s", err)
		return
	}
	defer func() {
		_ = videoConn.Close()
		log.Debugf("h264 video peer disconnected: %s", c.ClientIP())
	}()

	// create audio connection
	audioConn, err := createPeerConnection(iceServers, nil, false)
	if err != nil {
		log.Errorf("failed to create h264 audio peer connection: %s", err)
		return
	}
	defer func() {
		_ = audioConn.Close()
		log.Debugf("h264 audio peer disconnected: %s", c.ClientIP())
	}()

	// create client
	capabilities := media.Capabilities{SpeakerReceive: true, MicSend: true, VideoReceive: true}
	if value, ok := c.Get("mediaSession"); ok {
		capabilities = value.(media.Session).Capabilities
	}
	client := NewClientWithCapabilities(wsConn, videoConn, audioConn, capabilities)
	manager := getManager()
	if err := client.AddTrack(); err != nil {
		log.Errorf("failed to add track: %s", err)
		return
	}

	manager.AddClient(wsConn, client)
	defer manager.RemoveClient(wsConn)

	// handle signaling
	signalingHandler := NewSignalingHandler(client)
	signalingHandler.RegisterCallbacks()

	// read and wait
	for {
		message, err := client.ReadMessage()
		if err != nil {
			return
		}
		if message != nil {
			if err := signalingHandler.HandleMessage(message); err != nil {
				log.Errorf("failed to handle signaling message: %s", err)
			}
		}
	}
}

func createICEServers() []webrtc.ICEServer {
	var iceServers []webrtc.ICEServer

	conf := config.GetInstance()

	if conf.Stun != "" && conf.Stun != "disable" {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs: []string{"stun:" + conf.Stun},
		})
	}

	if conf.Turn.TurnAddr != "" && conf.Turn.TurnUser != "" && conf.Turn.TurnCred != "" {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{"turn:" + conf.Turn.TurnAddr},
			Username:   conf.Turn.TurnUser,
			Credential: conf.Turn.TurnCred,
		})
	}

	return iceServers
}

func createMediaEngine() (*webrtc.MediaEngine, error) {
	mediaEngine := &webrtc.MediaEngine{}

	h264Codec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeH264,
			ClockRate:   90000,
			SDPFmtpLine: "level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42001f",
			RTCPFeedback: []webrtc.RTCPFeedback{
				{Type: "nack"},
				{Type: "nack", Parameter: "pli"},
				{Type: "ccm", Parameter: "fir"},
			},
		},
		PayloadType: 102,
	}
	if err := mediaEngine.RegisterCodec(h264Codec, webrtc.RTPCodecTypeVideo); err != nil {
		log.Errorf("failed to register H.264 codec: %s", err)
		return nil, err
	}

	rtxCodec := webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeRTX,
			ClockRate:   90000,
			SDPFmtpLine: "apt=102",
		},
		PayloadType: 103,
	}
	if err := mediaEngine.RegisterCodec(rtxCodec, webrtc.RTPCodecTypeVideo); err != nil {
		log.Errorf("failed to register H.264 RTX codec: %s", err)
		return nil, err
	}

	if err := mediaEngine.RegisterHeaderExtension(
		webrtc.RTPHeaderExtensionCapability{URI: playoutdelay.URI},
		webrtc.RTPCodecTypeVideo,
	); err != nil {
		log.Errorf("failed to register header extension: %s", err)
		return nil, err
	}

	return mediaEngine, nil
}

func createPeerConnection(
	iceServers []webrtc.ICEServer,
	mediaEngine *webrtc.MediaEngine,
	withPlayoutDelay bool,
) (*webrtc.PeerConnection, error) {
	settingEngine := webrtc.SettingEngine{}
	settingEngine.SetSRTPProtectionProfiles(
		dtls.SRTP_AEAD_AES_128_GCM,
		dtls.SRTP_AES128_CM_HMAC_SHA1_80,
	)

	apiOptions := []func(api *webrtc.API){
		webrtc.WithSettingEngine(settingEngine),
	}
	if mediaEngine != nil {
		apiOptions = append(apiOptions, webrtc.WithMediaEngine(mediaEngine))
	}
	if withPlayoutDelay {
		if mediaEngine == nil {
			return nil, errors.New("playout-delay requires a media engine")
		}

		registry := &interceptor.Registry{}
		if err := webrtc.RegisterDefaultInterceptors(mediaEngine, registry); err != nil {
			return nil, fmt.Errorf("register default interceptors: %w", err)
		}

		factory, err := playoutdelay.NewFactory(0, 0)
		if err != nil {
			return nil, fmt.Errorf("create playout-delay interceptor: %w", err)
		}
		registry.Add(factory)
		apiOptions = append(apiOptions, webrtc.WithInterceptorRegistry(registry))
	}

	api := webrtc.NewAPI(apiOptions...)

	return api.NewPeerConnection(webrtc.Configuration{
		ICEServers:   iceServers,
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
	})
}
