export type VideoMode = 'h265-webrtc' | 'h265-direct' | 'h264-webrtc' | 'h264-direct';

export interface VideoParameters {
  rateControlMode: string; // cbr | vbr;
  bitrate: number;
  gop: number;
  fps: number;
  scale: number;
}

export enum VideoStatus {
  Normal = 1,
  NoImage = -1,
  VencError = -2,
  ImageBufferFull = -3,
  InconsistentVideoMode = -4,
  CaptureDisabled = -5
}
