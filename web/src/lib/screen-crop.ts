export type ScreenContentRect = {
  left: number;
  top: number;
  width: number;
  height: number;
};

export type ScreenCropState = {
  sourceWidth: number;
  sourceHeight: number;
  contentRect: ScreenContentRect | null;
  contentAspectRatio: number;
  updatedAt: number;
};

export type PortalMode = 'off' | 'auto' | 'manual';

export type PortalManualRegion = {
  id: string;
  sourceWidth: number;
  sourceHeight: number;
  contentRect: ScreenContentRect;
};

export type PortalResolution = {
  width: number;
  height: number;
};

export type ScreenSourceSize = {
  width: number;
  height: number;
};

export type PortalConfig = {
  version: number;
  mode: PortalMode;
  manualRegions: PortalManualRegion[];
  selectedRegionId: string;
  resolutions: PortalResolution[];
  selectedResolution: string;
};

export type PortalConfigResponse = PortalConfig & {
  configured: boolean;
};

export type PortalDetectionPhase = 'idle' | 'learning' | 'active';

export type ScreenElement = HTMLVideoElement | HTMLCanvasElement;

export const initialScreenCropState: ScreenCropState = {
  sourceWidth: 0,
  sourceHeight: 0,
  contentRect: null,
  contentAspectRatio: 0,
  updatedAt: 0
};

export const defaultPortalConfig: PortalConfig = {
  version: 1,
  mode: 'auto',
  manualRegions: [],
  selectedRegionId: '',
  resolutions: [],
  selectedResolution: ''
};

export function isScreenCropReady(
  crop: ScreenCropState | null | undefined
): crop is ScreenCropState & { contentRect: ScreenContentRect } {
  return !!crop && crop.sourceWidth > 0 && crop.sourceHeight > 0 && !!crop.contentRect;
}

export function getScreenCropLayout(
  crop: ScreenCropState | null | undefined,
  viewportWidth: number,
  viewportHeight: number
) {
  if (!isScreenCropReady(crop)) {
    return null;
  }

  const { contentRect } = crop;
  const fitScale = Math.min(
    Math.max(1, viewportWidth) / contentRect.width,
    Math.max(1, viewportHeight) / contentRect.height
  );

  const wrapperWidth = Math.max(1, Math.floor(contentRect.width * fitScale));
  const wrapperHeight = Math.max(1, Math.floor(contentRect.height * fitScale));

  return {
    wrapperStyle: {
      width: `${wrapperWidth}px`,
      height: `${wrapperHeight}px`,
      overflow: 'hidden',
      backgroundColor: '#000'
    },
    mediaStyle: {
      width: `${crop.sourceWidth * fitScale}px`,
      height: `${crop.sourceHeight * fitScale}px`,
      minWidth: 0,
      minHeight: 0,
      maxWidth: 'none',
      maxHeight: 'none',
      objectFit: 'fill',
      transform: `translate(${-contentRect.left * fitScale}px, ${-contentRect.top * fitScale}px)`,
      transformOrigin: 'top left'
    }
  };
}

export function getNormalizedScreenPointFromCrop(
  clientX: number,
  clientY: number,
  element: ScreenElement,
  crop: ScreenCropState | null | undefined
) {
  const rect = element.getBoundingClientRect();
  if (!rect.width || !rect.height || !isScreenCropReady(crop)) {
    return { x: 0, y: 0 };
  }

  const sourceX = ((clientX - rect.left) / rect.width) * crop.sourceWidth;
  const sourceY = ((clientY - rect.top) / rect.height) * crop.sourceHeight;

  return {
    x: (sourceX - crop.contentRect.left) / crop.contentRect.width,
    y: (sourceY - crop.contentRect.top) / crop.contentRect.height
  };
}

export function getNormalizedSourcePointFromCrop(
  clientX: number,
  clientY: number,
  element: ScreenElement,
  crop: ScreenCropState | null | undefined
) {
  const rect = element.getBoundingClientRect();
  if (!rect.width || !rect.height || !isScreenCropReady(crop)) {
    return { x: 0, y: 0 };
  }

  return {
    x: (clientX - rect.left) / rect.width,
    y: (clientY - rect.top) / rect.height
  };
}

export function mapNormalizedContentPointToSource(
  x: number,
  y: number,
  crop: ScreenCropState | null | undefined
) {
  if (!isScreenCropReady(crop)) {
    return { x: clamp(x, 0, 1), y: clamp(y, 0, 1) };
  }

  return {
    x: clamp((crop.contentRect.left + x * crop.contentRect.width) / crop.sourceWidth, 0, 1),
    y: clamp((crop.contentRect.top + y * crop.contentRect.height) / crop.sourceHeight, 0, 1)
  };
}

export function getScreenCropFromManualRegion(
  region: PortalManualRegion,
  sourceWidth: number,
  sourceHeight: number
): ScreenCropState {
  if (
    sourceWidth <= 0 ||
    sourceHeight <= 0 ||
    Math.abs(region.sourceWidth / region.sourceHeight - sourceWidth / sourceHeight) > 0.01
  ) {
    return initialScreenCropState;
  }

  const scaleX = sourceWidth / region.sourceWidth;
  const scaleY = sourceHeight / region.sourceHeight;
  const left = clamp(Math.round(region.contentRect.left * scaleX), 0, sourceWidth - 1);
  const top = clamp(Math.round(region.contentRect.top * scaleY), 0, sourceHeight - 1);
  const right = clamp(
    Math.round((region.contentRect.left + region.contentRect.width) * scaleX),
    left + 1,
    sourceWidth
  );
  const bottom = clamp(
    Math.round((region.contentRect.top + region.contentRect.height) * scaleY),
    top + 1,
    sourceHeight
  );
  const contentRect = { left, top, width: right - left, height: bottom - top };

  return {
    sourceWidth,
    sourceHeight,
    contentRect,
    contentAspectRatio: contentRect.width / contentRect.height,
    updatedAt: Date.now()
  };
}

export function getScreenCropFromResolution(
  resolution: PortalResolution,
  sourceWidth: number,
  sourceHeight: number
): ScreenCropState {
  if (resolution.width <= 0 || resolution.height <= 0 || sourceWidth <= 0 || sourceHeight <= 0) {
    return initialScreenCropState;
  }

  const targetRatio = resolution.width / resolution.height;
  const sourceRatio = sourceWidth / sourceHeight;
  const width = Math.round(targetRatio < sourceRatio ? sourceHeight * targetRatio : sourceWidth);
  const height = Math.round(targetRatio < sourceRatio ? sourceHeight : sourceWidth / targetRatio);
  const contentRect = {
    left: Math.floor((sourceWidth - width) / 2),
    top: Math.floor((sourceHeight - height) / 2),
    width,
    height
  };

  return {
    sourceWidth,
    sourceHeight,
    contentRect,
    contentAspectRatio: targetRatio,
    updatedAt: Date.now()
  };
}

export function getRenderedScreenRect(
  elementRect: DOMRect,
  sourceWidth: number,
  sourceHeight: number
) {
  if (!elementRect.width || !elementRect.height || sourceWidth <= 0 || sourceHeight <= 0) {
    return null;
  }

  const sourceRatio = sourceWidth / sourceHeight;
  const elementRatio = elementRect.width / elementRect.height;
  let width = elementRect.width;
  let height = elementRect.height;
  let left = elementRect.left;
  let top = elementRect.top;

  if (sourceRatio > elementRatio) {
    height = width / sourceRatio;
    top += (elementRect.height - height) / 2;
  } else {
    width = height * sourceRatio;
    left += (elementRect.width - width) / 2;
  }

  return { left, top, width, height };
}

export function selectScreenSourceSize(...candidates: ScreenSourceSize[]) {
  return candidates.find((candidate) => candidate.width > 0 && candidate.height > 0) ?? null;
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value));
}
