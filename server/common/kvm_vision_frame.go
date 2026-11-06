package common

func IsH264FrameType(result int) bool {
	switch uint8(result) {
	case IMG_H264_TYPE_SPS, IMG_H264_TYPE_PPS, IMG_H264_TYPE_IF, IMG_H264_TYPE_PF:
		return true
	default:
		return false
	}
}

func IsH265FrameType(result int) bool {
	switch uint8(result) {
	case IMG_H265_TYPE_SPS, IMG_H265_TYPE_PPS, IMG_H265_TYPE_IF, IMG_H265_TYPE_PF:
		return true
	default:
		return false
	}
}

func IsH264KeyFrame(result int) bool {
	return uint8(result) == IMG_H264_TYPE_SPS || uint8(result) == IMG_H264_TYPE_IF
}

func IsH265KeyFrame(result int) bool {
	return uint8(result) == IMG_H265_TYPE_SPS || uint8(result) == IMG_H265_TYPE_IF
}
