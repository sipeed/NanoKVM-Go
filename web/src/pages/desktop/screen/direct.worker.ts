import Queue from 'yocto-queue';

import {
  detectPortalEffectiveAreaFromImageData,
  getPortalContentSignature,
  PortalDetectionTracker
} from '../../../lib/portal-detection.ts';

const PORTAL_DETECTION_INTERVAL_MS = 1000;
const PORTAL_DETECTION_SAMPLE_MAX = 520;

let canvas: OffscreenCanvas | null = null;
let ctx: OffscreenCanvasRenderingContext2D | null = null;
let rendering: boolean = false;
let decoder: VideoDecoder | null = null;
let config: { codec: string; optimizeForLatency: boolean } | null = null;
let lastScreenWidth = 0;
let lastScreenHeight = 0;
let lastPortalDetectionAt = 0;
let portalDetectionCanvas: OffscreenCanvas | null = null;
let portalDetectionEnabled = true;
const portalDetectionTracker = new PortalDetectionTracker();

const frameQueue = new Queue<VideoFrame>();

self.onmessage = (event: MessageEvent) => {
  const { type, data, canvas: offscreenCanvas, enabled } = event.data;

  switch (type) {
    case 'init_h264':
    case 'init_h265':
      canvas = offscreenCanvas;
      ctx = canvas!.getContext('2d') as OffscreenCanvasRenderingContext2D;
      lastScreenWidth = 0;
      lastScreenHeight = 0;
      setPortalDetectionEnabled(enabled !== false);
      config = {
        codec: type === 'init_h265' ? 'hev1.1.6.L93.B0' : 'avc1.42E01F',
        optimizeForLatency: true
      };
      break;
    case 'set_portal_detection':
      setPortalDetectionEnabled(enabled === true);
      break;
    case 'ws_message':
      handleWsMessage(data);
      break;
    case 'error':
    case 'close':
      resetDecoder();
      break;
  }
};

function handleWsMessage(message: ArrayBuffer) {
  try {
    if (message.byteLength < 9) {
      return;
    }

    const view = new DataView(message);
    const isKeyFrame = view.getUint8(0) === 1;
    const timestamp = Number(view.getBigUint64(1, true));
    const data = new Uint8Array(message, 9);

    if (!decoder && isKeyFrame) {
      initializeDecoder();
    }

    if (decoder?.state === 'configured') {
      decode(isKeyFrame, timestamp, data);
    }
  } catch (error) {
    console.error('Error processing WebSocket message in worker:', error);
  }
}

function initializeDecoder() {
  if (!self.VideoDecoder) {
    console.log('Error: WebCodecs API not supported in this worker.');
    return;
  }

  if (!config || (decoder && decoder.state !== 'unconfigured')) {
    return;
  }

  const init = {
    output: (frame: VideoFrame) => {
      frameQueue.enqueue(frame);
      if (frameQueue.size >= 10) {
        frameQueue.dequeue()?.close();
      }

      if (!rendering) {
        rendering = true;
        processFrameQueue();
      }
    },
    error: () => {
      resetDecoder();
    }
  };

  try {
    decoder = new VideoDecoder(init);
    decoder.configure(config);
  } catch (err) {
    console.log(err);
    decoder = null;
  }
}

function decode(isKeyFrame: boolean, timestamp: number, data: Uint8Array) {
  const chunk = new EncodedVideoChunk({
    type: isKeyFrame ? 'key' : 'delta',
    timestamp: timestamp,
    data: data
  });

  try {
    decoder?.decode(chunk);
  } catch (err: any) {
    if (err.name === 'TypeError' || err.message.includes('configured')) {
      resetDecoder();
    }
  }
}

function processFrameQueue() {
  const frame = frameQueue.dequeue();
  if (frame) {
    renderFrame(frame);
  }

  if (frameQueue.size > 0) {
    setTimeout(processFrameQueue, 0);
  } else {
    rendering = false;
  }
}

function renderFrame(frame: VideoFrame) {
  if (!canvas || !ctx) {
    frame.close();
    return;
  }

  if (canvas.width !== frame.displayWidth || canvas.height !== frame.displayHeight) {
    canvas.width = frame.displayWidth;
    canvas.height = frame.displayHeight;
  }

  if (lastScreenWidth !== frame.displayWidth || lastScreenHeight !== frame.displayHeight) {
    lastScreenWidth = frame.displayWidth;
    lastScreenHeight = frame.displayHeight;
    self.postMessage({
      type: 'screen_size_changed',
      width: frame.displayWidth,
      height: frame.displayHeight,
      aspectRatio: frame.displayWidth / frame.displayHeight
    });
  }

  ctx.drawImage(frame, 0, 0, canvas.width, canvas.height);
  detectPortalAreaFromCanvas();
  frame.close();
}

function detectPortalAreaFromCanvas() {
  if (!portalDetectionEnabled || !canvas || !ctx || !canvas.width || !canvas.height) {
    return;
  }

  const now = Date.now();
  if (now - lastPortalDetectionAt < PORTAL_DETECTION_INTERVAL_MS) {
    return;
  }
  lastPortalDetectionAt = now;

  try {
    const scale = Math.min(1, PORTAL_DETECTION_SAMPLE_MAX / Math.max(canvas.width, canvas.height));
    const sampleWidth = Math.max(1, Math.round(canvas.width * scale));
    const sampleHeight = Math.max(1, Math.round(canvas.height * scale));

    portalDetectionCanvas ??= new OffscreenCanvas(sampleWidth, sampleHeight);
    if (
      portalDetectionCanvas.width !== sampleWidth ||
      portalDetectionCanvas.height !== sampleHeight
    ) {
      portalDetectionCanvas.width = sampleWidth;
      portalDetectionCanvas.height = sampleHeight;
    }

    const detectionCtx = portalDetectionCanvas.getContext('2d', { willReadFrequently: true });
    if (!detectionCtx) {
      return;
    }

    detectionCtx.drawImage(canvas, 0, 0, sampleWidth, sampleHeight);
    const imageData = detectionCtx.getImageData(0, 0, sampleWidth, sampleHeight);
    const detected = detectPortalEffectiveAreaFromImageData(imageData, sampleWidth, sampleHeight);
    const contentSignature = getPortalContentSignature(imageData, detected);
    const tracked = portalDetectionTracker.update(
      {
        ...detected,
        sourceWidth: canvas.width,
        sourceHeight: canvas.height,
        contentRect: detected.contentRect
          ? {
              left: Math.round(detected.contentRect.left / scale),
              top: Math.round(detected.contentRect.top / scale),
              width: Math.round(detected.contentRect.width / scale),
              height: Math.round(detected.contentRect.height / scale)
            }
          : null,
        contentAspectRatio: detected.contentRect
          ? detected.contentRect.width / detected.contentRect.height
          : canvas.width / canvas.height
      },
      contentSignature
    );

    self.postMessage({
      type: 'portal_effective_area',
      area: tracked.area,
      phase: tracked.phase
    });
  } catch (err) {
    console.log(err);
  }
}

function setPortalDetectionEnabled(enabled: boolean) {
  if (portalDetectionEnabled === enabled) {
    return;
  }

  portalDetectionEnabled = enabled;
  lastPortalDetectionAt = 0;
  portalDetectionTracker.reset();
  if (!enabled) {
    portalDetectionCanvas = null;
  }
}

function resetDecoder() {
  if (decoder && decoder.state !== 'closed') {
    try {
      decoder.close();
    } catch (err) {
      console.log(err);
    }
  }

  decoder = null;
  rendering = false;
  lastScreenWidth = 0;
  lastScreenHeight = 0;
  lastPortalDetectionAt = 0;
  portalDetectionTracker.reset();

  Array.from(frameQueue.drain()).forEach((frame) => frame.close());
}
