export function isWebrtcSupported() {
  // WebRTC-blocking extensions often keep the property but set it to undefined, so check constructibility instead of property presence.
  return typeof window !== 'undefined' && typeof window.RTCPeerConnection === 'function';
}

export function isWebrtcMode(mode: string | null): mode is 'h264-webrtc' | 'h265-webrtc' {
  return mode === 'h264-webrtc' || mode === 'h265-webrtc';
}

export function isDecoderSupported() {
  return 'VideoDecoder' in window;
}

export function isH265Supported() {
  if (typeof RTCRtpReceiver === 'undefined' || !('getCapabilities' in RTCRtpReceiver)) {
    return false;
  }

  const capabilities = RTCRtpReceiver.getCapabilities('video');
  const h265Codec = capabilities?.codecs?.find(
    (codec) => codec.mimeType.toLowerCase() === 'video/h265'
  );

  return !!h265Codec;
}

export function getSupportedVideoModes() {
  const webrtcSupported = isWebrtcSupported();
  const decoderSupported = isDecoderSupported();
  const h265Supported = isH265Supported();

  const videoModes: string[] = [];

  if (webrtcSupported) {
    videoModes.push('h264-webrtc');

    if (h265Supported) {
      videoModes.push('h265-webrtc');
    }
  }

  if (decoderSupported) {
    videoModes.push('h264-direct');

    if (h265Supported) {
      videoModes.push('h265-direct');
    }
  }

  return videoModes;
}
