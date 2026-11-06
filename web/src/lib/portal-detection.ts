import { ScreenCropState } from '@/lib/screen-crop.ts';

const MAX_SAMPLE_SIZE = 260;
const LINE_SAMPLE_COUNT = 160;
const BORDER_LINE_RATIO = 0.88;
const COLOR_DISTANCE_LIMIT = 38;
const MAX_BORDER_LUMA = 95;
const MIN_CONTENT_RATIO = 0.15;
const MIN_BORDER_RATIO = 0.05;
const MAX_BORDER_DELTA = 0.02;
const MAX_BORDER_RATIO = 0.425;
const MAX_SIDE_CROP_CONTENT_ASPECT = 1.65;
const MIN_CONTENT_HEIGHT_RATIO = 0.7;
const PORTAL_LOCK_CONFIRMATIONS = 4;
const PORTAL_STATIC_LOCK_CONFIRMATIONS = 8;
const PORTAL_PORTRAIT_LOCK_CONFIRMATIONS = 4;
const PORTAL_UNLOCK_CONFIRMATIONS = 4;
const CONTENT_SIGNATURE_COLUMNS = 7;
const CONTENT_SIGNATURE_ROWS = 5;

type PortalRect = {
  left: number;
  top: number;
  width: number;
  height: number;
};

type RGB = {
  r: number;
  g: number;
  b: number;
};

export function detectPortalEffectiveAreaFromImageData(
  imageData: ImageData,
  sourceWidth: number,
  sourceHeight: number
): ScreenCropState {
  const base = {
    sourceWidth,
    sourceHeight,
    contentRect: null,
    contentAspectRatio: sourceWidth > 0 && sourceHeight > 0 ? sourceWidth / sourceHeight : 0,
    updatedAt: Date.now()
  };

  if (sourceWidth <= 0 || sourceHeight <= 0 || imageData.width <= 0 || imageData.height <= 0) {
    return base;
  }

  const rect = detectPortalRect(imageData, sourceWidth, sourceHeight);
  if (!rect) {
    return base;
  }

  return {
    ...base,
    contentRect: rect,
    contentAspectRatio: rect.width / rect.height
  };
}

function detectPortalRect(imageData: ImageData, sourceWidth: number, sourceHeight: number) {
  const { sampleWidth, sampleHeight } = portalSampleSize(sourceWidth, sourceHeight);
  const borderColor = estimateBorderColor(imageData, sourceWidth, sourceHeight);
  if (luma(borderColor) > MAX_BORDER_LUMA) {
    return null;
  }

  const colRatios = Array.from({ length: sampleWidth }, (_, x) =>
    columnBorderRatioAt(
      imageData,
      sourceWidth,
      sourceHeight,
      sampleWidth,
      sampleHeight,
      borderColor,
      x
    )
  );

  const left = findContentStart(colRatios);
  const right = findContentEnd(colRatios);
  if (left > right) {
    return null;
  }

  const rowRatios = Array.from({ length: sampleHeight }, (_, y) =>
    rowBorderRatioAt(
      imageData,
      sourceWidth,
      sourceHeight,
      sampleWidth,
      sampleHeight,
      borderColor,
      y
    )
  );

  const top = findContentStart(rowRatios);
  const bottom = findContentEnd(rowRatios);
  if (top > bottom) {
    return null;
  }

  const scaleX = sourceWidth / sampleWidth;
  const scaleY = sourceHeight / sampleHeight;
  const detectedRect = clampRect(
    {
      left: Math.floor(left * scaleX),
      top: Math.floor(top * scaleY),
      width: Math.ceil((right - left + 1) * scaleX),
      height: Math.ceil((bottom - top + 1) * scaleY)
    },
    sourceWidth,
    sourceHeight
  );

  const rect = normalizePortalRectCandidate(detectedRect, sourceWidth, sourceHeight);
  if (!rect) {
    return null;
  }

  const minWidth = Math.max(32, Math.floor(sourceWidth * MIN_CONTENT_RATIO));
  const minHeight = Math.max(32, Math.floor(sourceHeight * MIN_CONTENT_RATIO));
  if (rect.width < minWidth || rect.height < minHeight) {
    return null;
  }

  return rect;
}

function normalizePortalRectCandidate(
  rect: PortalRect,
  sourceWidth: number,
  sourceHeight: number
): PortalRect | null {
  const leftRatio = rect.left / sourceWidth;
  const rightRatio = (sourceWidth - (rect.left + rect.width)) / sourceWidth;
  const topRatio = rect.top / sourceHeight;
  const bottomRatio = (sourceHeight - (rect.top + rect.height)) / sourceHeight;
  const cropLeftRight = isSymmetricBorderPair(leftRatio, rightRatio);
  const cropTopBottom = isSymmetricBorderPair(topRatio, bottomRatio);

  if (!cropLeftRight && !cropTopBottom) {
    return null;
  }

  const candidate = clampRect(
    {
      left: cropLeftRight ? rect.left : 0,
      top: cropTopBottom ? rect.top : 0,
      width: cropLeftRight ? rect.width : sourceWidth,
      height: cropTopBottom ? rect.height : sourceHeight
    },
    sourceWidth,
    sourceHeight
  );

  const contentAspect = candidate.width / candidate.height;
  const contentHeightRatio = candidate.height / sourceHeight;
  if (
    (cropLeftRight && !cropTopBottom && contentAspect > MAX_SIDE_CROP_CONTENT_ASPECT) ||
    contentHeightRatio < MIN_CONTENT_HEIGHT_RATIO
  ) {
    return null;
  }

  return candidate;
}

function isSymmetricBorderPair(firstRatio: number, secondRatio: number) {
  return (
    firstRatio >= MIN_BORDER_RATIO &&
    secondRatio >= MIN_BORDER_RATIO &&
    firstRatio <= MAX_BORDER_RATIO &&
    secondRatio <= MAX_BORDER_RATIO &&
    Math.abs(firstRatio - secondRatio) <= MAX_BORDER_DELTA
  );
}

function portalSampleSize(sourceWidth: number, sourceHeight: number) {
  const scale = Math.min(1, MAX_SAMPLE_SIZE / Math.max(sourceWidth, sourceHeight));
  return {
    sampleWidth: Math.max(1, Math.round(sourceWidth * scale)),
    sampleHeight: Math.max(1, Math.round(sourceHeight * scale))
  };
}

function estimateBorderColor(imageData: ImageData, sourceWidth: number, sourceHeight: number) {
  const cornerWidth = Math.max(2, Math.floor(sourceWidth * 0.04));
  const cornerHeight = Math.max(2, Math.floor(sourceHeight * 0.04));
  const corners = [
    { x: 0, y: 0 },
    { x: sourceWidth - cornerWidth, y: 0 },
    { x: 0, y: sourceHeight - cornerHeight },
    { x: sourceWidth - cornerWidth, y: sourceHeight - cornerHeight }
  ];

  let r = 0;
  let g = 0;
  let b = 0;
  let count = 0;

  for (const corner of corners) {
    for (let y = 0; y < cornerHeight; y += 1) {
      for (let x = 0; x < cornerWidth; x += 1) {
        const color = pixel(imageData, corner.x + x, corner.y + y);
        r += color.r;
        g += color.g;
        b += color.b;
        count += 1;
      }
    }
  }

  return { r: r / count, g: g / count, b: b / count };
}

function columnBorderRatioAt(
  imageData: ImageData,
  sourceWidth: number,
  sourceHeight: number,
  sampleWidth: number,
  sampleHeight: number,
  borderColor: RGB,
  index: number
) {
  const step = Math.max(1, Math.floor(sampleHeight / LINE_SAMPLE_COUNT));
  let matches = 0;
  let count = 0;

  for (let offset = 0; offset < sampleHeight; offset += step) {
    const sourceX = sampleCoord(index, sampleWidth, sourceWidth);
    const sourceY = sampleCoord(offset, sampleHeight, sourceHeight);
    if (isBorderColor(pixel(imageData, sourceX, sourceY), borderColor)) {
      matches += 1;
    }
    count += 1;
  }

  return count === 0 ? 0 : matches / count;
}

function rowBorderRatioAt(
  imageData: ImageData,
  sourceWidth: number,
  sourceHeight: number,
  sampleWidth: number,
  sampleHeight: number,
  borderColor: RGB,
  index: number
) {
  const step = Math.max(1, Math.floor(sampleWidth / LINE_SAMPLE_COUNT));
  let matches = 0;
  let count = 0;

  for (let offset = 0; offset < sampleWidth; offset += step) {
    const sourceX = sampleCoord(offset, sampleWidth, sourceWidth);
    const sourceY = sampleCoord(index, sampleHeight, sourceHeight);
    if (isBorderColor(pixel(imageData, sourceX, sourceY), borderColor)) {
      matches += 1;
    }
    count += 1;
  }

  return count === 0 ? 0 : matches / count;
}

function sampleCoord(index: number, sampleSize: number, sourceSize: number) {
  if (sampleSize <= 1) {
    return 0;
  }
  return Math.min(sourceSize - 1, Math.round((index * (sourceSize - 1)) / (sampleSize - 1)));
}

function findContentStart(ratios: number[]) {
  for (let i = 0; i < ratios.length; i += 1) {
    if (hasContentRun(ratios, i, 1)) {
      return findFirstContentLine(ratios, i, 1);
    }
  }
  return ratios.length;
}

function findContentEnd(ratios: number[]) {
  for (let i = ratios.length - 1; i >= 0; i -= 1) {
    if (hasContentRun(ratios, i, -1)) {
      return findFirstContentLine(ratios, i, -1);
    }
  }
  return -1;
}

function findFirstContentLine(ratios: number[], start: number, direction: number) {
  const runLength = Math.min(5, ratios.length);
  for (let i = 0; i < runLength; i += 1) {
    const index = start + i * direction;
    if (index >= 0 && index < ratios.length && ratios[index] < BORDER_LINE_RATIO) {
      return index;
    }
  }
  return start;
}

function hasContentRun(ratios: number[], start: number, direction: number) {
  const runLength = Math.min(5, ratios.length);
  let contentLines = 0;

  for (let i = 0; i < runLength; i += 1) {
    const index = start + i * direction;
    if (index < 0 || index >= ratios.length) {
      continue;
    }
    if (ratios[index] < BORDER_LINE_RATIO) {
      contentLines += 1;
    }
  }

  return contentLines >= Math.ceil(runLength * 0.6);
}

function isBorderColor(color: RGB, borderColor: RGB) {
  const dr = color.r - borderColor.r;
  const dg = color.g - borderColor.g;
  const db = color.b - borderColor.b;
  return dr * dr + dg * dg + db * db <= COLOR_DISTANCE_LIMIT * COLOR_DISTANCE_LIMIT;
}

function pixel(imageData: ImageData, x: number, y: number) {
  const sourceX = Math.max(0, Math.min(imageData.width - 1, Math.round(x)));
  const sourceY = Math.max(0, Math.min(imageData.height - 1, Math.round(y)));
  const offset = (sourceY * imageData.width + sourceX) * 4;
  const data = imageData.data;
  return {
    r: data[offset],
    g: data[offset + 1],
    b: data[offset + 2]
  };
}

function luma(color: RGB) {
  return color.r * 0.2126 + color.g * 0.7152 + color.b * 0.0722;
}

function clampRect(rect: PortalRect, sourceWidth: number, sourceHeight: number) {
  const left = clamp(rect.left, 0, sourceWidth - 1);
  const top = clamp(rect.top, 0, sourceHeight - 1);
  const right = clamp(rect.left + rect.width, left + 1, sourceWidth);
  const bottom = clamp(rect.top + rect.height, top + 1, sourceHeight);
  return {
    left,
    top,
    width: right - left,
    height: bottom - top
  };
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(max, value));
}

export type PortalDetectionUpdate = {
  area: ScreenCropState;
  phase: 'idle' | 'learning' | 'active';
};

export class PortalDetectionTracker {
  private lockedArea: ScreenCropState | null = null;
  private candidateArea: ScreenCropState | null = null;
  private candidateConfirmations = 0;
  private candidateSignatures = new Set<number>();
  private fullFrameConfirmations = 0;

  update(area: ScreenCropState, contentSignature?: number): PortalDetectionUpdate {
    if (!this.sameSource(this.lockedArea, area)) {
      this.reset();
    }

    if (!area.contentRect) {
      this.candidateArea = null;
      this.candidateConfirmations = 0;
      this.candidateSignatures.clear();
      if (!this.lockedArea) {
        return { area, phase: 'idle' };
      }

      this.fullFrameConfirmations += 1;
      if (this.fullFrameConfirmations >= PORTAL_UNLOCK_CONFIRMATIONS) {
        this.reset();
        return { area, phase: 'idle' };
      }

      return { area: this.lockedArea, phase: 'active' };
    }

    this.fullFrameConfirmations = 0;
    if (this.lockedArea && samePortalArea(this.lockedArea, area)) {
      this.candidateArea = null;
      this.candidateConfirmations = 0;
      this.candidateSignatures.clear();
      return { area: this.lockedArea, phase: 'active' };
    }

    if (this.candidateArea && samePortalArea(this.candidateArea, area)) {
      this.candidateConfirmations += 1;
    } else {
      this.candidateArea = area;
      this.candidateConfirmations = 1;
      this.candidateSignatures.clear();
    }
    if (contentSignature !== undefined) {
      this.candidateSignatures.add(contentSignature);
    }

    const staticLockConfirmations = isPortraitPillarbox(area)
      ? PORTAL_PORTRAIT_LOCK_CONFIRMATIONS
      : PORTAL_STATIC_LOCK_CONFIRMATIONS;
    if (
      (this.candidateConfirmations >= PORTAL_LOCK_CONFIRMATIONS &&
        this.candidateSignatures.size >= 2) ||
      this.candidateConfirmations >= staticLockConfirmations
    ) {
      this.lockedArea = area;
      this.candidateArea = null;
      this.candidateConfirmations = 0;
      this.candidateSignatures.clear();
      return { area: this.lockedArea, phase: 'active' };
    }

    return {
      area: this.lockedArea ?? {
        sourceWidth: area.sourceWidth,
        sourceHeight: area.sourceHeight,
        contentRect: null,
        contentAspectRatio: area.sourceWidth / area.sourceHeight,
        updatedAt: area.updatedAt
      },
      phase: 'learning'
    };
  }

  reset() {
    this.lockedArea = null;
    this.candidateArea = null;
    this.candidateConfirmations = 0;
    this.candidateSignatures.clear();
    this.fullFrameConfirmations = 0;
  }

  private sameSource(previous: ScreenCropState | null, next: ScreenCropState) {
    return (
      !previous ||
      (previous.sourceWidth === next.sourceWidth && previous.sourceHeight === next.sourceHeight)
    );
  }
}

function isPortraitPillarbox(area: ScreenCropState) {
  const rect = area.contentRect;
  if (!rect || area.sourceWidth <= 0 || area.sourceHeight <= 0) {
    return false;
  }

  const leftRatio = rect.left / area.sourceWidth;
  const rightRatio = (area.sourceWidth - rect.left - rect.width) / area.sourceWidth;
  return (
    rect.width < rect.height &&
    rect.height / area.sourceHeight >= 0.9 &&
    leftRatio >= 0.25 &&
    rightRatio >= 0.25
  );
}

export function getPortalContentSignature(
  imageData: ImageData,
  area: ScreenCropState
): number | undefined {
  const rect = area.contentRect;
  if (!rect || rect.width <= 0 || rect.height <= 0) {
    return undefined;
  }

  let signature = 2166136261;
  for (let row = 0; row < CONTENT_SIGNATURE_ROWS; row += 1) {
    for (let column = 0; column < CONTENT_SIGNATURE_COLUMNS; column += 1) {
      const x = rect.left + ((column + 0.5) * rect.width) / CONTENT_SIGNATURE_COLUMNS;
      const y = rect.top + ((row + 0.5) * rect.height) / CONTENT_SIGNATURE_ROWS;
      const color = pixel(imageData, x, y);
      const quantizedLuma = Math.round(luma(color) / 16);
      signature ^= quantizedLuma;
      signature = Math.imul(signature, 16777619);
    }
  }

  return signature >>> 0;
}

function samePortalArea(first: ScreenCropState, second: ScreenCropState) {
  if (
    first.sourceWidth !== second.sourceWidth ||
    first.sourceHeight !== second.sourceHeight ||
    !first.contentRect ||
    !second.contentRect
  ) {
    return false;
  }

  const tolerance = Math.max(
    2,
    Math.round(Math.max(first.sourceWidth, first.sourceHeight) * 0.002)
  );
  return (
    Math.abs(first.contentRect.left - second.contentRect.left) <= tolerance &&
    Math.abs(first.contentRect.top - second.contentRect.top) <= tolerance &&
    Math.abs(first.contentRect.width - second.contentRect.width) <= tolerance &&
    Math.abs(first.contentRect.height - second.contentRect.height) <= tolerance
  );
}
