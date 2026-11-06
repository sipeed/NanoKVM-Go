/**
 * Browser capability and platform detection for the desktop fullscreen action.
 *
 * The functions in this file deliberately accept a small environment object.
 * Besides making the iOS heuristics explicit, that keeps the decision logic
 * testable without constructing a browser DOM in Node.
 */

export type FullscreenMode = 'native' | 'guide' | 'hidden';

export interface PlatformDocument {
  fullscreenEnabled?: boolean;
  fullscreenElement?: unknown;
  documentElement?: {
    requestFullscreen?: () => Promise<void> | void;
  } | null;
  exitFullscreen?: () => Promise<void> | void;
}

export interface PlatformNavigator {
  userAgent?: string;
  platform?: string;
  maxTouchPoints?: number;
  standalone?: boolean;
  keyboard?: {
    lock?: () => Promise<void> | void;
    unlock?: () => Promise<void> | void;
  };
}

export interface PlatformWindow {
  innerWidth?: number;
  innerHeight?: number;
  screen?: {
    width?: number;
    height?: number;
  };
  matchMedia?: (query: string) => { matches: boolean };
}

export interface PlatformEnvironment {
  document?: PlatformDocument;
  navigator?: PlatformNavigator;
  window?: PlatformWindow;
}

export interface FullscreenCapability {
  mode: FullscreenMode;
  isIPhone: boolean;
  isIPad: boolean;
  isStandalone: boolean;
  supportsNativeFullscreen: boolean;
}

function getBrowserEnvironment(environment?: PlatformEnvironment): PlatformEnvironment {
  if (environment) return environment;

  return {
    document: typeof document === 'undefined' ? undefined : document,
    navigator: typeof navigator === 'undefined' ? undefined : navigator,
    window: typeof window === 'undefined' ? undefined : window
  };
}

function getUserAgent(environment?: PlatformEnvironment): string {
  return getBrowserEnvironment(environment).navigator?.userAgent ?? '';
}

function isMacIntelTouchDevice(environment?: PlatformEnvironment): boolean {
  const navigatorObject = getBrowserEnvironment(environment).navigator;
  return navigatorObject?.platform === 'MacIntel' && (navigatorObject.maxTouchPoints ?? 0) > 1;
}

/**
 * iPhone Safari can send a desktop-site user agent. In that mode it looks like
 * a touch-enabled Mac, while an iPad still reports a much larger physical
 * short side (including the 744px iPad mini). Requiring both dimensions lets
 * us support the former without exposing the iPhone guide on iPadOS.
 */
export function isSmallMacIntelTouchDevice(environment?: PlatformEnvironment): boolean {
  if (!isMacIntelTouchDevice(environment)) return false;

  const browserEnvironment = getBrowserEnvironment(environment);
  const screen = browserEnvironment.window?.screen;
  const width = screen?.width ?? browserEnvironment.window?.innerWidth;
  const height = screen?.height ?? browserEnvironment.window?.innerHeight;

  if (!Number.isFinite(width) || !Number.isFinite(height)) return false;
  return Math.min(width as number, height as number) < 600;
}

/** Detect iPad before applying the small MacIntel touch-device fallback. */
export function isIPad(environment?: PlatformEnvironment): boolean {
  const userAgent = getUserAgent(environment);
  if (/\biPad\b/i.test(userAgent)) return true;

  // iPadOS desktop-site mode uses Macintosh/MacIntel + touch points. Treat an
  // unknown-size device as iPad rather than guessing it is an iPhone.
  return isMacIntelTouchDevice(environment) && !isSmallMacIntelTouchDevice(environment);
}

export function isIPhone(environment?: PlatformEnvironment): boolean {
  const userAgent = getUserAgent(environment);
  if (/\biPad\b/i.test(userAgent)) return false;
  if (/\b(?:iPhone|iPod)\b/i.test(userAgent)) return true;
  return isSmallMacIntelTouchDevice(environment) && !isIPad(environment);
}

/**
 * Check both halves of the Fullscreen API contract. Some iOS browsers expose
 * requestFullscreen as a stub while advertising fullscreenEnabled=false.
 */
export function supportsNativeFullscreen(environment?: PlatformEnvironment): boolean {
  const browserEnvironment = getBrowserEnvironment(environment);
  return (
    browserEnvironment.document?.fullscreenEnabled === true &&
    typeof browserEnvironment.document.documentElement?.requestFullscreen === 'function'
  );
}

export function isStandalone(environment?: PlatformEnvironment): boolean {
  const browserEnvironment = getBrowserEnvironment(environment);
  const mediaQueryMatches = browserEnvironment.window?.matchMedia?.(
    '(display-mode: standalone)'
  )?.matches;

  return mediaQueryMatches === true || browserEnvironment.navigator?.standalone === true;
}

export function getFullscreenMode(environment?: PlatformEnvironment): FullscreenMode {
  const browserEnvironment = getBrowserEnvironment(environment);
  const iphone = isIPhone(browserEnvironment);
  const standalone = isStandalone(browserEnvironment);

  // A standalone iPhone already has the browser chrome removed. Keep the
  // toolbar compact and do not offer an action that cannot improve the view.
  if (iphone && standalone) return 'hidden';
  if (supportsNativeFullscreen(browserEnvironment)) return 'native';
  if (iphone) return 'guide';
  return 'hidden';
}

export function getFullscreenCapability(environment?: PlatformEnvironment): FullscreenCapability {
  const browserEnvironment = getBrowserEnvironment(environment);
  const iphone = isIPhone(browserEnvironment);
  const ipad = isIPad(browserEnvironment);
  const standalone = isStandalone(browserEnvironment);
  const native = supportsNativeFullscreen(browserEnvironment);

  return {
    mode: getFullscreenMode(browserEnvironment),
    isIPhone: iphone,
    isIPad: ipad,
    isStandalone: standalone,
    supportsNativeFullscreen: native
  };
}

/** Enter fullscreen, locking the keyboard only after the request succeeds. */
export async function enterNativeFullscreen(environment?: PlatformEnvironment): Promise<void> {
  const browserEnvironment = getBrowserEnvironment(environment);
  const element = browserEnvironment.document?.documentElement;
  if (typeof element?.requestFullscreen !== 'function') {
    throw new Error('Fullscreen API is unavailable');
  }

  await element.requestFullscreen();

  // Keyboard Lock is an optional enhancement and can be denied independently
  // after fullscreen succeeds. Do not report a successful fullscreen request
  // as failed solely because the lock is unavailable.
  try {
    void Promise.resolve(browserEnvironment.navigator?.keyboard?.lock?.()).catch(() => undefined);
  } catch {
    // Ignore an optional Keyboard Lock rejection.
  }
}

/** Exit fullscreen and release a previously acquired keyboard lock. */
export async function exitNativeFullscreen(environment?: PlatformEnvironment): Promise<void> {
  const browserEnvironment = getBrowserEnvironment(environment);
  if (typeof browserEnvironment.document?.exitFullscreen !== 'function') {
    throw new Error('Fullscreen API is unavailable');
  }

  await browserEnvironment.document.exitFullscreen();

  try {
    await browserEnvironment.navigator?.keyboard?.unlock?.();
  } catch {
    // Ignore an optional Keyboard Lock release failure.
  }
}
