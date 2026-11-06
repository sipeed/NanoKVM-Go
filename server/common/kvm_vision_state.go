package common

import (
	"sync"
	"sync/atomic"
)

type KvmVision struct {
	streamType atomic.Uint32
	frameMu    sync.Mutex
}

func newKvmVision() *KvmVision {
	vision := &KvmVision{}
	vision.SetStreamType(STREAM_TYPE_H264_WEBRTC)
	return vision
}

func (k *KvmVision) StreamType() uint8 {
	return uint8(k.streamType.Load())
}

func (k *KvmVision) SetStreamType(streamType uint8) {
	k.streamType.Store(uint32(streamType))
}
