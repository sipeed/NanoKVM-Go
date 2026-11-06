package ai

const mediaProtocolVersion = "nanokvm-webrtc-media/v1"

type PCMContract struct {
	Format     string `json:"format"`
	SampleRate int    `json:"sampleRate"`
	Channels   int    `json:"channels"`
}

type AudioContract struct {
	Codec                     string      `json:"codec"`
	RTPClockRate              int         `json:"rtpClockRate"`
	Channels                  int         `json:"channels"`
	FrameDurationMS           int         `json:"frameDurationMs"`
	SamplesPerChannelPerFrame int         `json:"samplesPerChannelPerFrame"`
	DecodedPCM                PCMContract `json:"decodedPCM"`
	RecommendedFMTP           string      `json:"recommendedFmtp"`
}

type SignalingEndpoint struct {
	Transport     string `json:"transport"`
	Path          string `json:"path"`
	SameHostAsMCP bool   `json:"sameHostAsMCP"`
	HTTPScheme    string `json:"httpScheme"`
	HTTPSScheme   string `json:"httpsScheme"`
}

type TokenUsage struct {
	Location  string `json:"location"`
	Opaque    bool   `json:"opaque"`
	Sensitive bool   `json:"sensitive"`
	Log       bool   `json:"log"`
}

type PlaneHelp struct {
	Transport    string   `json:"transport"`
	Carries      []string `json:"carries"`
	NeverCarries []string `json:"neverCarries,omitempty"`
}

type SignalingMessage struct {
	Direction string `json:"direction"`
	Event     string `json:"event"`
	Data      string `json:"data"`
}

type SignalingHelp struct {
	AudioPath      string             `json:"audioPath"`
	AudioVideoPath string             `json:"audioVideoPath"`
	URLRule        string             `json:"urlRule"`
	Messages       []SignalingMessage `json:"messages"`
	ICE            string             `json:"ice"`
}

type TrackHelp struct {
	NetworkDirection      string `json:"networkDirection"`
	ControlledHostMeaning string `json:"controlledHostMeaning"`
	ClientAction          string `json:"clientAction"`
	Codec                 string `json:"codec"`
}

type RealtimeHelp struct {
	Pacing    []string `json:"pacing"`
	BargeIn   []string `json:"bargeIn"`
	Buffering []string `json:"buffering"`
	AEC       string   `json:"aec"`
}

type LifecycleHelp struct {
	DefaultTTLSeconds int      `json:"defaultTtlSeconds"`
	MaximumTTLSeconds int      `json:"maximumTtlSeconds"`
	Expiration        string   `json:"expiration"`
	Cleanup           []string `json:"cleanup"`
	Reconnect         []string `json:"reconnect"`
}

type IntegrationHelp struct {
	Name     string       `json:"name"`
	Input    *PCMContract `json:"input,omitempty"`
	Output   *PCMContract `json:"output,omitempty"`
	Pipeline []string     `json:"pipeline"`
}

type ClientExamples struct {
	Create            string   `json:"create"`
	PionGo            []string `json:"pionGo"`
	BrowserJavaScript []string `json:"browserJavaScript"`
}

type MediaSessionHelpResult struct {
	ProtocolVersion string               `json:"protocolVersion"`
	Purpose         string               `json:"purpose"`
	ControlPlane    PlaneHelp            `json:"controlPlane"`
	DataPlane       PlaneHelp            `json:"dataPlane"`
	Signaling       SignalingHelp        `json:"signaling"`
	AudioContract   AudioContract        `json:"audioContract"`
	Tracks          map[string]TrackHelp `json:"tracks"`
	Workflow        []string             `json:"workflow"`
	RealtimeRules   RealtimeHelp         `json:"realtimeRules"`
	Lifecycle       LifecycleHelp        `json:"lifecycle"`
	Security        []string             `json:"security"`
	Integrations    []IntegrationHelp    `json:"integrations"`
	Examples        ClientExamples       `json:"examples"`
}

func newAudioContract() AudioContract {
	return AudioContract{Codec: "audio/opus", RTPClockRate: 48000, Channels: 2, FrameDurationMS: 20, SamplesPerChannelPerFrame: 960, DecodedPCM: PCMContract{Format: "s16le", SampleRate: 48000, Channels: 2}, RecommendedFMTP: "minptime=10;useinbandfec=1"}
}

func ptrAudioContract(value AudioContract) *AudioContract { return &value }

func newSignalingEndpoint(path string) *SignalingEndpoint {
	return &SignalingEndpoint{Transport: "websocket", Path: path, SameHostAsMCP: true, HTTPScheme: "ws", HTTPSScheme: "wss"}
}

func newTokenUsage() *TokenUsage {
	return &TokenUsage{Location: "media_session query parameter in signalingPath", Opaque: true, Sensitive: true, Log: false}
}

func newMediaSessionHelp() MediaSessionHelpResult {
	qwenInput := PCMContract{Format: "s16le", SampleRate: 16000, Channels: 1}
	qwenOutput := PCMContract{Format: "s16le", SampleRate: 24000, Channels: 1}
	return MediaSessionHelpResult{
		ProtocolVersion: mediaProtocolVersion,
		Purpose:         "MCP discovers and authorizes media. WebRTC RTP tracks carry continuous audio. An external Agent runtime may connect ASR, tools, web access, and TTS.",
		ControlPlane:    PlaneHelp{Transport: "MCP tools", Carries: []string{"session authorization", "status", "close"}, NeverCarries: []string{"PCM frames", "Opus packets", "RTP audio"}},
		DataPlane:       PlaneHelp{Transport: "WebRTC audio RTP tracks negotiated over WebSocket", Carries: []string{"Opus speaker audio", "Opus microphone audio", "optional video"}, NeverCarries: []string{"MCP API key"}},
		Signaling: SignalingHelp{
			AudioPath: "/api/media/audio/webrtc?media_session=<token>", AudioVideoPath: "/api/media/webrtc?media_session=<token>",
			URLRule:  "Use the same host as the MCP endpoint; replace https with wss or http with ws and append the returned signalingPath.",
			Messages: []SignalingMessage{{"client-to-server", "offer", "JSON-encoded RTCSessionDescription string"}, {"server-to-client", "answer", "JSON-encoded RTCSessionDescription string"}, {"both-directions", "candidate", "JSON-encoded RTCIceCandidateInit string"}},
			ICE:      "Exchange trickle ICE candidates in candidate envelopes. Keep the WebSocket open for the lifetime of the PeerConnection.",
		},
		AudioContract: newAudioContract(),
		Tracks: map[string]TrackHelp{
			"speakerReceive": {"NanoKVM server -> client", "controlled-host HDMI/system speaker audio", "add a recvonly audio transceiver; play, record, or decode for ASR", "Opus"},
			"micSend":        {"client -> NanoKVM server", "input to the controlled host UAC2 virtual microphone", "add a local Opus audio track and send paced 20 ms frames", "Opus"},
		},
		Workflow: []string{"Call media_session_help before implementing the client.", "Call media_session_create with at least one capability.", "Create RTCPeerConnection and configure recvonly speaker and/or local microphone track before creating the offer.", "SetLocalDescription, send the offer envelope, apply the answer with SetRemoteDescription, and exchange ICE candidates.", "Process media continuously; do not transfer audio through MCP or a DataChannel.", "Close local tracks, PeerConnection, WebSocket, queues, and then call media_session_close."},
		RealtimeRules: RealtimeHelp{
			Pacing:    []string{"Encode exactly 960 samples per channel per 20 ms Opus frame.", "Send one frame every 20 ms using a monotonic clock; never burst queued TTS as fast as possible."},
			BargeIn:   []string{"When user speech starts, immediately stop TTS playback.", "Clear pending PCM and Opus output queues.", "Cancel the active model response and never send stale response frames."},
			Buffering: []string{"Use bounded low-latency queues.", "On overflow prefer dropping stale output and mark a discontinuity; never build an unbounded backlog."},
			AEC:       "If rendered TTS can return in captured speaker audio, provide synchronized render/reference and capture streams to an AEC. VAD does not replace AEC.",
		},
		Lifecycle: LifecycleHelp{DefaultTTLSeconds: 600, MaximumTTLSeconds: 600, Expiration: "Expiration rejects new signaling. An already established PeerConnection may continue until either endpoint closes it; there is no token renewal API.", Cleanup: []string{"close media queues", "stop tracks and transceivers", "close RTCPeerConnection", "close signaling WebSocket", "call media_session_close"}, Reconnect: []string{"On WebSocket, ICE, or PeerConnection failure close all local media state.", "Create a new media session; do not reuse an expired token.", "Retry with bounded exponential backoff and jitter, for example 1s, 2s, 4s, 8s."}},
		Security:  []string{"The MCP API key authenticates MCP only; never put it in the signaling URL.", "The media token is opaque, sensitive, capability-scoped, and must not be logged.", "media_session_status does not return the token and cannot recover a lost token.", "Prefer wss and configure proxies not to log the media_session query value."},
		Integrations: []IntegrationHelp{
			{Name: "generic-agent", Pipeline: []string{"speakerReceive Opus -> decode/resample -> VAD/ASR -> Agent with MCP/local/web tools -> TTS -> resample/frame/pace/encode -> micSend"}},
			{Name: "qwen-audio-realtime-example", Input: &qwenInput, Output: &qwenOutput, Pipeline: []string{"NanoKVM Opus 48k stereo -> decode/downmix/resample -> Qwen PCM 16k mono", "Qwen PCM 24k mono -> bounded queue -> resample/upmix to 48k stereo -> 960 samples/channel -> Opus -> one frame per 20ms"}},
		},
		Examples: ClientExamples{
			Create:            `{"speakerReceive":true,"micSend":true,"ttlSeconds":600}`,
			PionGo:            []string{`register Opus capability: MimeType audio/opus, ClockRate 48000, Channels 2, SDPFmtpLine "minptime=10;useinbandfec=1"`, `speakerReceive: pc.AddTransceiverFromKind(audio, recvonly)`, `micSend: NewTrackLocalStaticSample(Opus capability, "microphone", "nanokvm-client") then pc.AddTrack(track)`, `track.WriteSample(media.Sample{Data: opusPacket, Duration: 20*time.Millisecond})`},
			BrowserJavaScript: []string{`pc.addTransceiver("audio", {direction:"recvonly"}) for speakerReceive`, `use a MediaStreamTrack for micSend; browsers do not expose arbitrary pre-encoded Opus payload writes through RTCRtpSender`, `send JSON.stringify({event:"offer", data:JSON.stringify(pc.localDescription)}) and similarly wrap candidates`, `handle ontrack and close tracks, pc, and WebSocket during cleanup`},
		},
	}
}
