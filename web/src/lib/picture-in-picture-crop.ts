import type { ScreenCropState } from '@/lib/screen-crop.ts';

export type PixelRect = {
  x: number;
  y: number;
  width: number;
  height: number;
};

// Chroma-subsampled frames (I420/NV12) need even offsets and sizes.
const alignDown = (value: number) => Math.floor(value / 2) * 2;

/**
 * Map the on-screen crop (in source pixels) onto a decoded frame, which may be
 * scaled relative to the source. Returns null when the full frame should be
 * shown.
 */
export function getFrameCropRect(
  crop: ScreenCropState | null | undefined,
  frameWidth: number,
  frameHeight: number
): PixelRect | null {
  const rect = crop?.contentRect;
  if (!crop || !rect || crop.sourceWidth <= 0 || crop.sourceHeight <= 0) return null;
  if (frameWidth <= 0 || frameHeight <= 0) return null;

  const scaleX = frameWidth / crop.sourceWidth;
  const scaleY = frameHeight / crop.sourceHeight;

  const left = Math.max(0, Math.min(frameWidth, rect.left * scaleX));
  const top = Math.max(0, Math.min(frameHeight, rect.top * scaleY));
  const right = Math.max(left, Math.min(frameWidth, (rect.left + rect.width) * scaleX));
  const bottom = Math.max(top, Math.min(frameHeight, (rect.top + rect.height) * scaleY));

  const x = alignDown(left);
  const y = alignDown(top);
  const width = alignDown(right - x);
  const height = alignDown(bottom - y);
  if (width < 2 || height < 2) return null;
  if (x === 0 && y === 0 && width >= alignDown(frameWidth) && height >= alignDown(frameHeight)) {
    return null;
  }

  return { x, y, width, height };
}
