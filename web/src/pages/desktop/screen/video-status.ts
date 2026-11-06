import { VideoStatus } from '@/types';

export const videoStatusEvent = 'video-status';

export type StreamEvent = {
  event: string;
  data: string;
};

export type VideoStatusUpdate = {
  status: VideoStatus;
  isPlaying: boolean;
};

export const parseStreamEvent = (payload: unknown): StreamEvent | null => {
  if (typeof payload !== 'string') {
    return null;
  }

  try {
    const message: unknown = JSON.parse(payload);
    if (!message || typeof message !== 'object' || !('event' in message) || !('data' in message)) {
      return null;
    }

    const { event, data } = message;
    if (typeof event !== 'string' || typeof data !== 'string') {
      return null;
    }

    return { event, data };
  } catch {
    return null;
  }
};

export const parseVideoStatusUpdate = (message: StreamEvent): VideoStatusUpdate | null => {
  if (message.event !== videoStatusEvent) {
    return null;
  }

  const status = Number(message.data);
  if (!Number.isInteger(status) || String(status) !== message.data) {
    return null;
  }

  switch (status) {
    case VideoStatus.Normal:
      return { status: VideoStatus.Normal, isPlaying: true };
    case VideoStatus.NoImage:
      return { status: VideoStatus.NoImage, isPlaying: false };
    case VideoStatus.InconsistentVideoMode:
      return { status: VideoStatus.InconsistentVideoMode, isPlaying: false };
    case VideoStatus.CaptureDisabled:
      return { status: VideoStatus.CaptureDisabled, isPlaying: false };
    default:
      return null;
  }
};
