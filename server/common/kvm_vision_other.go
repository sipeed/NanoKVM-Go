//go:build !linux || !cgo

package common

const (
	RATE_CONTROL_CBR uint8 = iota
	RATE_CONTROL_VBR
)

const (
	IMG_H264_TYPE_SPS uint8 = 1
	IMG_H264_TYPE_PPS uint8 = 2
	IMG_H264_TYPE_IF  uint8 = 3
	IMG_H264_TYPE_PF  uint8 = 4
	IMG_H265_TYPE_SPS uint8 = 5
	IMG_H265_TYPE_PPS uint8 = 6
	IMG_H265_TYPE_IF  uint8 = 7
	IMG_H265_TYPE_PF  uint8 = 8
)

// kvmv_read_img result codes (negative = error/no-data).
const (
	IMG_BUFFER_FULL = -3
	IMG_VENC_ERROR  = -2
	IMG_NOT_EXIST   = -1
)

const (
	STREAM_TYPE_H264_WEBRTC = iota
	STREAM_TYPE_H264_DIRECT
	STREAM_TYPE_H265_WEBRTC
	STREAM_TYPE_H265_DIRECT
)

var kvmVision = newKvmVision()

func GetKvmVision() *KvmVision {
	return kvmVision
}

func (k *KvmVision) SetRateControl(mode uint8) int {
	if mode != RATE_CONTROL_CBR && mode != RATE_CONTROL_VBR {
		return -1
	}

	return -1
}

func (k *KvmVision) ReadH264(width uint16, height uint16, bitRate uint16) (data []byte, result int) {
	return nil, -1
}

func (k *KvmVision) ReadH265(width uint16, height uint16, bitRate uint16) (data []byte, result int) {
	return nil, -1
}

func (k *KvmVision) ReadAudio() (data []byte, result int) {
	return nil, -1
}

func (k *KvmVision) ResetAudioCapture() {}

func (k *KvmVision) SetFps(fps uint8) int {
	return -1
}

func (k *KvmVision) GetFps() int {
	return -1
}

func (k *KvmVision) SetHDMI(enable bool) int {
	return -1
}

func (k *KvmVision) SetGop(gop uint16) int {
	return -1
}

func (k *KvmVision) SetBitRate(bitRate uint16) int {
	return -1
}

func (k *KvmVision) ForceIDR() int {
	return -1
}

func (k *KvmVision) CloseVideoStream() {}

func (k *KvmVision) Close() {}
