export type ResponsiveDeviceInput = {
  width: number;
  height: number;
  hasTouch: boolean;
  coarsePointer: boolean;
  userAgent: string;
};

export type ResponsiveDeviceState = {
  isTouchLikeDevice: boolean;
  isMobileDevice: boolean;
  isPortraitViewport: boolean;
  isLandscapeViewport: boolean;
  isMobilePortrait: boolean;
};

const MOBILE_DEVICE_MAX_SHORT_EDGE = 1024;

export function getBrowserLayoutViewportSize() {
  if (typeof window === 'undefined') {
    return { width: 1, height: 1 };
  }

  const documentElement = typeof document === 'undefined' ? null : document.documentElement;
  return {
    width: Math.max(1, documentElement?.clientWidth || window.innerWidth),
    height: Math.max(1, documentElement?.clientHeight || window.innerHeight)
  };
}

export function getResponsiveDeviceState(input: ResponsiveDeviceInput): ResponsiveDeviceState {
  const isMobileUserAgent = /Android|iPhone|iPad|iPod|Mobile|Tablet/i.test(input.userAgent);
  const isTouchLikeDevice = input.hasTouch || input.coarsePointer || isMobileUserAgent;
  const isMobileDevice =
    isMobileUserAgent ||
    (isTouchLikeDevice && Math.min(input.width, input.height) <= MOBILE_DEVICE_MAX_SHORT_EDGE);
  const isPortraitViewport = input.height > input.width;
  const isLandscapeViewport = input.width >= input.height;

  return {
    isTouchLikeDevice,
    isMobileDevice,
    isPortraitViewport,
    isLandscapeViewport,
    isMobilePortrait: isTouchLikeDevice && isPortraitViewport
  };
}
