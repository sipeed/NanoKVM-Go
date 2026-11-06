// Picture-in-Picture for the remote screen.
//
// The screen is a <video> (WebRTC modes) or a worker-owned <canvas> (direct
// modes), and the portal crop is applied with CSS, which PiP ignores. Where the
// browser supports insertable streams, each frame is cropped with a
// VideoFrame visibleRect and fed to a hidden proxy <video> that enters PiP, so
// the PiP window shows the same region as the page. Other browsers fall back to
// the uncropped picture. The PiP window is view-only.

import { getFrameCropRect } from '@/lib/picture-in-picture-crop.ts';
import type { ScreenCropState } from '@/lib/screen-crop.ts';

const SCREEN_ELEMENT_ID = 'screen';
const CANVAS_FRAME_RATE = 30;
// A canvas only emits frames when it is repainted; a frozen source would
// otherwise leave play() pending forever. Stay well inside the browser's
// user-activation window so requestPictureInPicture() is still allowed.
const FIRST_FRAME_TIMEOUT_MS = 3000;

type GetCrop = () => ScreenCropState | null;

export type PictureInPictureOptions = {
  /** Read for every frame so the PiP window follows crop changes. */
  getCrop: GetCrop;
  /** Calls back when the screen element is about to be replaced; returns an unsubscribe. */
  subscribeScreenChange: (onChange: () => void) => () => void;
};

type TrackProcessorCtor = new (init: { track: MediaStreamTrack }) => {
  readable: ReadableStream<VideoFrame>;
};
type TrackGeneratorCtor = new (init: { kind: 'video' }) => MediaStreamTrack & {
  writable: WritableStream<VideoFrame>;
};

type Session = {
  video: HTMLVideoElement;
  stop: () => void;
};

let session: Session | null = null;

export function isPictureInPictureSupported(): boolean {
  return typeof document !== 'undefined' && document.pictureInPictureEnabled === true;
}

export function isPictureInPictureActive(): boolean {
  return typeof document !== 'undefined' && document.pictureInPictureElement !== null;
}

function getInsertableStreams(): {
  Processor: TrackProcessorCtor;
  Generator: TrackGeneratorCtor;
} | null {
  const scope = globalThis as unknown as {
    MediaStreamTrackProcessor?: TrackProcessorCtor;
    MediaStreamTrackGenerator?: TrackGeneratorCtor;
  };
  if (!scope.MediaStreamTrackProcessor || !scope.MediaStreamTrackGenerator) return null;
  return { Processor: scope.MediaStreamTrackProcessor, Generator: scope.MediaStreamTrackGenerator };
}

/**
 * Direct modes need insertable streams: without them the canvas is captured
 * through play() before requestPictureInPicture(), and Safari drops the user
 * activation across that wait.
 */
export function canEnterPictureInPicture(isWebrtcMode: boolean): boolean {
  return isPictureInPictureSupported() && (isWebrtcMode || getInsertableStreams() !== null);
}

function getSourceTrack(screen: HTMLVideoElement | HTMLCanvasElement): MediaStreamTrack | null {
  if (screen instanceof HTMLCanvasElement) {
    return screen.captureStream(CANVAS_FRAME_RATE).getVideoTracks()[0] ?? null;
  }

  const stream = screen.srcObject;
  const track = stream instanceof MediaStream ? stream.getVideoTracks()[0] : undefined;
  // Clone so stopping the PiP copy never ends the page's own track.
  return track ? track.clone() : null;
}

function cropFrame(frame: VideoFrame, getCrop: GetCrop): VideoFrame {
  // Coded size may include alignment padding; crop within the visible area.
  const visible = frame.visibleRect;
  const rect = getFrameCropRect(
    getCrop(),
    visible?.width ?? frame.displayWidth,
    visible?.height ?? frame.displayHeight
  );
  if (!rect) return frame;

  try {
    const cropped = new VideoFrame(frame, {
      visibleRect: {
        x: rect.x + (visible?.x ?? 0),
        y: rect.y + (visible?.y ?? 0),
        width: rect.width,
        height: rect.height
      },
      displayWidth: rect.width,
      displayHeight: rect.height
    });
    frame.close();
    return cropped;
  } catch {
    // An unsupported crop for this frame format must not kill the pipeline;
    // show the frame uncropped instead.
    return frame;
  }
}

function cropTrack(source: MediaStreamTrack, getCrop: GetCrop, onEnd: () => void) {
  const streams = getInsertableStreams();
  if (!streams) return null;

  const processor = new streams.Processor({ track: source });
  const generator = new streams.Generator({ kind: 'video' });

  processor.readable
    .pipeThrough(
      new TransformStream<VideoFrame, VideoFrame>({
        transform(frame, controller) {
          controller.enqueue(cropFrame(frame, getCrop));
        }
      })
    )
    .pipeTo(generator.writable)
    .catch(() => undefined)
    // However the pipeline ends, the PiP window would otherwise freeze.
    .finally(onEnd);

  return generator;
}

function createProxyVideo(track: MediaStreamTrack): HTMLVideoElement {
  const video = document.createElement('video');
  video.muted = true;
  video.playsInline = true;
  video.autoplay = true;
  video.setAttribute('aria-hidden', 'true');
  // Safari only allows PiP for videos in the document; keep it rendered but
  // invisible rather than display:none.
  Object.assign(video.style, {
    position: 'fixed',
    left: '0',
    top: '0',
    width: '1px',
    height: '1px',
    opacity: '0',
    pointerEvents: 'none'
  });
  video.srcObject = new MediaStream([track]);
  document.body.appendChild(video);
  return video;
}

function playWithTimeout(video: HTMLVideoElement): Promise<void> {
  return new Promise((resolve, reject) => {
    const timer = window.setTimeout(
      () => reject(new Error('no video frames')),
      FIRST_FRAME_TIMEOUT_MS
    );
    video.play().then(
      () => {
        window.clearTimeout(timer);
        resolve();
      },
      (error: unknown) => {
        window.clearTimeout(timer);
        reject(error);
      }
    );
  });
}

function releaseSession() {
  const current = session;
  session = null;
  current?.stop();
}

/** Exit PiP, but only if `owner` is still the active session. */
function endSession(owner: Session) {
  if (session === owner) void exitPictureInPicture().catch(() => undefined);
}

function startSession(
  screen: HTMLVideoElement | HTMLCanvasElement,
  { getCrop, subscribeScreenChange }: PictureInPictureOptions
): Session | null {
  if (!getInsertableStreams() && screen instanceof HTMLVideoElement) {
    // No insertable streams: show the page's own video, uncropped. Replacing
    // that element exits PiP on its own.
    return null;
  }

  const source = getSourceTrack(screen);
  if (!source) return null;

  // Filled in below; the callbacks only fire after this function returns.
  const owner = {} as Session;
  const cropped = cropTrack(source, getCrop, () => endSession(owner));
  const output = cropped ?? source;
  const video = createProxyVideo(output);

  // The proxy outlives the screen element: leave PiP before it freezes on a
  // removed canvas or a closed peer connection.
  const unsubscribe = subscribeScreenChange(() => endSession(owner));
  const onSourceEnded = () => endSession(owner);
  source.addEventListener('ended', onSourceEnded, { once: true });

  owner.video = video;
  owner.stop = () => {
    unsubscribe();
    source.removeEventListener('ended', onSourceEnded);
    output.stop();
    source.stop();
    video.srcObject = null;
    video.remove();
  };
  video.addEventListener('leavepictureinpicture', releaseSession, { once: true });
  return owner;
}

/** Must be called from a user gesture. Throws when the screen cannot enter PiP. */
export async function enterPictureInPicture(options: PictureInPictureOptions): Promise<void> {
  const screen = document.getElementById(SCREEN_ELEMENT_ID);
  if (!(screen instanceof HTMLVideoElement) && !(screen instanceof HTMLCanvasElement)) {
    throw new Error('screen element not found');
  }

  releaseSession();
  session = startSession(screen, options);
  if (!session && !(screen instanceof HTMLVideoElement)) {
    throw new Error('screen has no video track');
  }
  const video = session?.video ?? (screen as HTMLVideoElement);

  try {
    if (session) await playWithTimeout(video);
    await video.requestPictureInPicture();
  } catch (error) {
    releaseSession();
    throw error;
  }
}

export async function exitPictureInPicture(): Promise<void> {
  if (document.pictureInPictureElement) {
    await document.exitPictureInPicture();
  }
  releaseSession();
}
