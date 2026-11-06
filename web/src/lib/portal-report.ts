import type {
  PortalDetectionPhase,
  PortalMode,
  ScreenContentRect
} from '@/lib/screen-crop.ts';

export type PortalAreaReport = {
  sourceWidth: number;
  sourceHeight: number;
  contentRect: ScreenContentRect | null;
};

export type PortalAreaReportInput = {
  mode: PortalMode;
  phase: PortalDetectionPhase;
  contentRect: ScreenContentRect | null;
  sourceWidth: number;
  sourceHeight: number;
  /** True once this page reported a detected crop. */
  observedCrop: boolean;
  /** Detection ticks seen on this page, used to settle the "no borders" case. */
  detectionTicks: number;
};

/**
 * Detection ticks without any crop that count as "this frame has no black
 * borders". The tracker itself needs four to eight confirmations before it
 * locks, so a shorter window would race the first lock.
 */
export const PORTAL_REPORT_SETTLE_TICKS = 10;

/**
 * The server persists the detected content area so MCP screenshots and clicks
 * use the same crop the browser shows. Returning null means "keep the stored
 * area"; a payload with a null contentRect clears it.
 */
export function portalAreaReport(input: PortalAreaReportInput): PortalAreaReport | null {
  if (input.mode !== 'auto' || input.sourceWidth <= 0 || input.sourceHeight <= 0) {
    return null;
  }

  if (input.contentRect) {
    // The tracker scales a downscaled sample back to source pixels, so an edge can
    // land outside the frame. The reported rect becomes the authoritative crop, so
    // keep it inside.
    const contentRect = clampContentRect(input.contentRect, input.sourceWidth, input.sourceHeight);

    // A detected area that covers the whole frame is positive information: the
    // frame has no borders, so the stored area can be cleared right away.
    if (coversWholeFrame(contentRect, input.sourceWidth, input.sourceHeight)) {
      return {
        sourceWidth: input.sourceWidth,
        sourceHeight: input.sourceHeight,
        contentRect: null
      };
    }

    if (input.phase === 'active') {
      return {
        sourceWidth: input.sourceWidth,
        sourceHeight: input.sourceHeight,
        contentRect
      };
    }

    // While the detector is still learning, the stored area stays authoritative.
    return null;
  }

  if (input.phase !== 'idle') {
    return null;
  }

  const settled = input.observedCrop || input.detectionTicks >= PORTAL_REPORT_SETTLE_TICKS;
  if (!settled) {
    return null;
  }

  return {
    sourceWidth: input.sourceWidth,
    sourceHeight: input.sourceHeight,
    contentRect: null
  };
}

function clampContentRect(
  contentRect: ScreenContentRect,
  sourceWidth: number,
  sourceHeight: number
): ScreenContentRect {
  const left = Math.min(Math.max(0, contentRect.left), sourceWidth - 1);
  const top = Math.min(Math.max(0, contentRect.top), sourceHeight - 1);
  return {
    left,
    top,
    width: Math.min(Math.max(1, contentRect.width), sourceWidth - left),
    height: Math.min(Math.max(1, contentRect.height), sourceHeight - top)
  };
}

function coversWholeFrame(
  contentRect: ScreenContentRect,
  sourceWidth: number,
  sourceHeight: number
): boolean {
  return (
    contentRect.left <= 0 &&
    contentRect.top <= 0 &&
    contentRect.width >= sourceWidth &&
    contentRect.height >= sourceHeight
  );
}

export function portalAreaReportKey(report: PortalAreaReport | null): string {
  if (!report) {
    return '';
  }

  const { contentRect } = report;
  if (!contentRect) {
    return `${report.sourceWidth}x${report.sourceHeight}:none`;
  }

  return `${report.sourceWidth}x${report.sourceHeight}:${contentRect.left},${contentRect.top},${contentRect.width},${contentRect.height}`;
}
