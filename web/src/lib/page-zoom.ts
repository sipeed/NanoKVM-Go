const PAGE_ZOOM_BASELINE_KEY = 'nano-kvm-page-zoom-baseline';
const MIN_PAGE_ZOOM = 0.5;
const MAX_PAGE_ZOOM = 4;

type PageZoomStorage = {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
};

type ScreenSize = {
  width: number;
  height: number;
};

export type PageZoomViewport = {
  innerWidth: number;
  outerWidth: number;
};

export type PageZoomBaseline = {
  devicePixelRatio: number;
  screenWidth: number;
  screenHeight: number;
  viewport: PageZoomViewport | null;
};

const PAGE_ZOOM_SIGNAL_EPSILON = 0.05;

export function getOrCreatePageZoomBaseline(
  currentDevicePixelRatio: number,
  screenSize: ScreenSize,
  storage?: PageZoomStorage
) {
  return getOrCreatePageZoomBaselineData(currentDevicePixelRatio, screenSize, storage)
    .devicePixelRatio;
}

export function getOrCreatePageZoomBaselineData(
  currentDevicePixelRatio: number,
  screenSize: ScreenSize,
  storage?: PageZoomStorage,
  viewport?: PageZoomViewport | null
): PageZoomBaseline {
  const devicePixelRatio = normalizeDevicePixelRatio(currentDevicePixelRatio);
  const baseline = readPageZoomBaseline(storage);

  if (
    baseline &&
    baseline.screenWidth === screenSize.width &&
    baseline.screenHeight === screenSize.height
  ) {
    if (!baseline.viewport && viewport) {
      const migratedBaseline = {
        ...baseline,
        viewport: normalizePageZoomViewport(viewport)
      } satisfies PageZoomBaseline;
      writePageZoomBaseline(storage, migratedBaseline);
      return migratedBaseline;
    }
    return baseline;
  }

  const nextBaseline = {
    devicePixelRatio,
    screenWidth: screenSize.width,
    screenHeight: screenSize.height,
    viewport: normalizePageZoomViewport(viewport)
  } satisfies PageZoomBaseline;
  writePageZoomBaseline(storage, nextBaseline);
  return nextBaseline;
}

export function getPageZoomScale(
  currentDevicePixelRatio: number,
  baselineDevicePixelRatio: number,
  currentViewport?: PageZoomViewport | null,
  baselineViewport?: PageZoomViewport | null,
  userAgent = globalThis.navigator?.userAgent ?? ''
) {
  const deviceScale =
    normalizeDevicePixelRatio(currentDevicePixelRatio) /
    normalizeDevicePixelRatio(baselineDevicePixelRatio);
  // Chromium reports page zoom through DPR. Docked panels also change innerWidth.
  if (
    /\b(?:Chrome|Chromium)\//.test(userAgent) ||
    Math.abs(deviceScale - 1) > PAGE_ZOOM_SIGNAL_EPSILON
  ) {
    return clampPageZoom(deviceScale);
  }

  const viewportScale = getOuterInnerZoomScale(currentViewport, baselineViewport);
  return clampPageZoom(
    viewportScale !== null && Math.abs(viewportScale - 1) > PAGE_ZOOM_SIGNAL_EPSILON
      ? viewportScale
      : deviceScale
  );
}

function readPageZoomBaseline(storage?: PageZoomStorage): PageZoomBaseline | null {
  if (!storage) return null;

  try {
    const value = storage.getItem(PAGE_ZOOM_BASELINE_KEY);
    if (!value) return null;

    const baseline = JSON.parse(value) as Partial<PageZoomBaseline>;
    if (
      typeof baseline.devicePixelRatio !== 'number' ||
      !Number.isFinite(baseline.devicePixelRatio) ||
      baseline.devicePixelRatio <= 0 ||
      typeof baseline.screenWidth !== 'number' ||
      typeof baseline.screenHeight !== 'number'
    ) {
      return null;
    }

    return {
      devicePixelRatio: baseline.devicePixelRatio,
      screenWidth: baseline.screenWidth,
      screenHeight: baseline.screenHeight,
      viewport: normalizePageZoomViewport(baseline.viewport)
    };
  } catch {
    return null;
  }
}

function writePageZoomBaseline(storage: PageZoomStorage | undefined, baseline: PageZoomBaseline) {
  if (!storage) return;

  try {
    storage.setItem(PAGE_ZOOM_BASELINE_KEY, JSON.stringify(baseline));
  } catch {
    return;
  }
}

function normalizeDevicePixelRatio(value: number) {
  return Number.isFinite(value) && value > 0 ? value : 1;
}

function normalizePageZoomViewport(viewport: unknown): PageZoomViewport | null {
  if (!viewport || typeof viewport !== 'object') return null;
  const value = viewport as Partial<PageZoomViewport>;
  if (!validPositiveNumber(value.innerWidth) || !validPositiveNumber(value.outerWidth)) {
    return null;
  }
  return {
    innerWidth: value.innerWidth,
    outerWidth: value.outerWidth
  };
}

function getOuterInnerZoomScale(
  currentViewport: PageZoomViewport | null | undefined,
  baselineViewport: PageZoomViewport | null | undefined
) {
  if (!currentViewport || !baselineViewport) return null;

  const baselineWindowInset = baselineViewport.outerWidth - baselineViewport.innerWidth;
  const unzoomedViewportWidth = currentViewport.outerWidth - baselineWindowInset;
  if (!validPositiveNumber(unzoomedViewportWidth)) {
    return null;
  }

  return unzoomedViewportWidth / currentViewport.innerWidth;
}

function validPositiveNumber(value: unknown): value is number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0;
}

function clampPageZoom(value: number) {
  return Math.max(MIN_PAGE_ZOOM, Math.min(MAX_PAGE_ZOOM, value));
}
