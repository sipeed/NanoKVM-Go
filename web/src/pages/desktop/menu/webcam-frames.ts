type TrackProcessor = new (options: { track: MediaStreamTrack; maxBufferSize: number }) => {
  readable: ReadableStream<VideoFrame>;
};

export function cameraFrameStream(track: MediaStreamTrack) {
  const Processor = (window as typeof window & { MediaStreamTrackProcessor?: TrackProcessor })
    .MediaStreamTrackProcessor;
  return Processor ? new Processor({ track, maxBufferSize: 1 }).readable : undefined;
}

export type CameraFrame = VideoFrame | HTMLVideoElement;

export function releaseFrame(frame?: CameraFrame) {
  if (frame && 'close' in frame) frame.close();
}

export function webcamFrames(input: MediaStreamTrack | ReadableStream<VideoFrame>) {
  const reader = 'getReader' in input ? input.getReader() : undefined;
  const video = 'getReader' in input ? undefined : document.createElement('video');
  const canvas =
    typeof OffscreenCanvas === 'function'
      ? new OffscreenCanvas(1280, 720)
      : document.createElement('canvas');
  canvas.width = 1280;
  canvas.height = 720;
  const ctx = canvas.getContext('2d', { alpha: false }) as
    | CanvasRenderingContext2D
    | OffscreenCanvasRenderingContext2D
    | null;
  let closed = false;
  let callback = 0;
  let timer: ReturnType<typeof setTimeout>;

  async function listen(accept: (frame: CameraFrame, timeMs: number) => void) {
    if (reader) {
      while (!closed) {
        const { value, done } = await reader.read();
        if (done) {
          if (!closed) throw new Error('The camera disconnected.');
          return;
        }
        if (closed) value.close();
        else accept(value, value.timestamp / 1000);
      }
    } else if (video) {
      // The fallback observes the camera independently of upload grants, too.
      if (typeof video.requestVideoFrameCallback === 'function') {
        const frame: Parameters<HTMLVideoElement['requestVideoFrameCallback']>[0] = (now) => {
          if (closed) return;
          // WebKit can report mediaTime = 0 on every live frame.
          accept(video, now);
          callback = video.requestVideoFrameCallback(frame);
        };
        callback = video.requestVideoFrameCallback(frame);
      } else {
        let previous = -1;
        const poll = () => {
          if (closed) return;
          const number = video.getVideoPlaybackQuality().totalVideoFrames;
          if (number !== previous && number > 0) {
            previous = number;
            accept(video, performance.now());
          }
          timer = setTimeout(poll, 10);
        };
        poll();
      }
    }
  }

  async function encode(source: CameraFrame, quality: number) {
    if (!ctx) throw new Error('The browser could not prepare the camera frame.');
    const width = 'displayWidth' in source ? source.displayWidth : source.videoWidth;
    const height = 'displayHeight' in source ? source.displayHeight : source.videoHeight;
    if (!width || !height) return;
    const scale = Math.min(canvas.width / width, canvas.height / height);
    const w = Math.round(width * scale);
    const h = Math.round(height * scale);
    ctx.fillStyle = '#000';
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(source, (canvas.width - w) / 2, (canvas.height - h) / 2, w, h);
    if ('convertToBlob' in canvas) return canvas.convertToBlob({ type: 'image/jpeg', quality });
    return new Promise<Blob>((resolve, reject) =>
      canvas.toBlob(
        (blob) => (blob ? resolve(blob) : reject(new Error('JPEG encoding failed.'))),
        'image/jpeg',
        quality
      )
    );
  }

  return {
    async prepare() {
      if (video && !('getReader' in input)) {
        video.muted = true;
        video.playsInline = true;
        video.srcObject = new MediaStream([input]);
        await video.play();
      }
    },
    listen,
    encode,
    close() {
      if (closed) return;
      closed = true;
      if (reader) void reader.cancel().catch(() => {});
      if (video) {
        if (callback) video.cancelVideoFrameCallback(callback);
        clearTimeout(timer);
        video.srcObject = null;
      }
    }
  };
}
