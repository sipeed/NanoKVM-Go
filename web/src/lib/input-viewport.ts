import type { ScreenCropState } from './screen-crop.ts';

export type ViewportSize = {
  width: number;
  height: number;
};

export type ViewportRect = ViewportSize & {
  left: number;
  top: number;
};

export type NormalizedPoint = {
  x: number;
  y: number;
};

export type InputViewport = {
  sourceSize: ViewportSize;
  contentRect: ViewportRect;
  contentAspectRatio: number;
};

export type TouchPort = {
  clientRect: ViewportRect;
  viewport: InputViewport;
};

export type TouchPortPoint = {
  content: NormalizedPoint;
  source: NormalizedPoint;
};

export function createInputViewport(
  crop: ScreenCropState | null | undefined,
  fallbackSourceSize?: ViewportSize
): InputViewport {
  if (
    crop &&
    crop.sourceWidth > 0 &&
    crop.sourceHeight > 0 &&
    crop.contentRect &&
    crop.contentRect.width > 0 &&
    crop.contentRect.height > 0
  ) {
    const contentRect = { ...crop.contentRect };
    return {
      sourceSize: { width: crop.sourceWidth, height: crop.sourceHeight },
      contentRect,
      contentAspectRatio: contentRect.width / contentRect.height
    };
  }

  const sourceSize = validSize(fallbackSourceSize)
    ? { width: fallbackSourceSize.width, height: fallbackSourceSize.height }
    : { width: 1, height: 1 };
  return {
    sourceSize,
    contentRect: { left: 0, top: 0, ...sourceSize },
    contentAspectRatio: sourceSize.width / sourceSize.height
  };
}

export function createTouchPort(
  clientRect: ViewportRect,
  crop: ScreenCropState | null | undefined
): TouchPort {
  const rect = normalizedRect(clientRect);
  return {
    clientRect: rect,
    viewport: createInputViewport(crop, rect)
  };
}

export function mapClientPointToTouchPort(
  touchPort: TouchPort,
  clientX: number,
  clientY: number
): TouchPortPoint {
  return mapContentPointToTouchPort(
    touchPort.viewport,
    normalizeClientAxis(clientX, touchPort.clientRect.left, touchPort.clientRect.width),
    normalizeClientAxis(clientY, touchPort.clientRect.top, touchPort.clientRect.height)
  );
}

export function mapLocalPointToTouchPort(
  touchPort: TouchPort,
  localX: number,
  localY: number,
  localWidth: number,
  localHeight: number
): TouchPortPoint {
  return mapContentPointToTouchPort(
    touchPort.viewport,
    normalizeClientAxis(localX, 0, localWidth),
    normalizeClientAxis(localY, 0, localHeight)
  );
}

export function mapContentPointToTouchPort(
  viewport: InputViewport,
  x: number,
  y: number
): TouchPortPoint {
  const content = { x: clamp(x), y: clamp(y) };
  const { contentRect, sourceSize } = viewport;
  return {
    content,
    source: {
      x: clamp((contentRect.left + content.x * contentRect.width) / sourceSize.width),
      y: clamp((contentRect.top + content.y * contentRect.height) / sourceSize.height)
    }
  };
}

function normalizeClientAxis(value: number, offset: number, length: number) {
  return clamp((value - offset) / Math.max(1, length - 1));
}

function normalizedRect(rect: ViewportRect): ViewportRect {
  return {
    left: finiteOr(rect.left, 0),
    top: finiteOr(rect.top, 0),
    width: Math.max(1, finiteOr(rect.width, 1)),
    height: Math.max(1, finiteOr(rect.height, 1))
  };
}

function validSize(size: ViewportSize | undefined): size is ViewportSize {
  return (
    !!size &&
    Number.isFinite(size.width) &&
    size.width > 0 &&
    Number.isFinite(size.height) &&
    size.height > 0
  );
}

function finiteOr(value: number, fallback: number) {
  return Number.isFinite(value) ? value : fallback;
}

function clamp(value: number) {
  return Math.max(0, Math.min(1, value));
}
