import { CSSProperties, RefObject, useEffect, useMemo, useRef, useState } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { VideoStatus } from '@/types';
import { getBrowserLayoutViewportSize } from '@/lib/mobile-layout.ts';
import {
  getOrCreatePageZoomBaselineData,
  getPageZoomScale,
  type PageZoomBaseline,
  type PageZoomViewport
} from '@/lib/page-zoom.ts';
import {
  detectPortalEffectiveAreaFromImageData,
  getPortalContentSignature,
  PortalDetectionTracker
} from '@/lib/portal-detection.ts';
import {
  getScreenCropLayout,
  initialScreenCropState,
  isScreenCropReady
} from '@/lib/screen-crop.ts';
import {
  detectedScreenAreaAtom,
  effectiveScreenAreaAtom,
  portalBorderColorAtom,
  portalBorderEnabledAtom,
  portalDetectionPhaseAtom,
  portalModeAtom,
  portalSelectingAtom,
  screenSourceSizeAtom,
  videoStatusAtom
} from '@/jotai/screen.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

const PORTAL_DETECTION_INTERVAL_MS = 1000;
const PORTAL_DETECTION_SAMPLE_MAX = 520;
const PORTAL_SELECTION_SCALE = 0.75;
const SCREEN_BORDER_CLEARANCE = 6;

function getBrowserPageZoomBaseline(): PageZoomBaseline {
  if (typeof window === 'undefined') {
    return {
      devicePixelRatio: 1,
      screenWidth: 1,
      screenHeight: 1,
      viewport: null
    };
  }

  let storage: Storage | undefined;
  try {
    storage = window.sessionStorage;
  } catch {
    storage = undefined;
  }

  return getOrCreatePageZoomBaselineData(
    window.devicePixelRatio,
    { width: window.screen.width, height: window.screen.height },
    storage,
    getPageZoomViewport()
  );
}

function getPageZoomViewport(): PageZoomViewport {
  if (typeof window === 'undefined') {
    return { innerWidth: 1, outerWidth: 1 };
  }

  return {
    innerWidth: window.innerWidth,
    outerWidth: window.outerWidth || window.screen.width
  };
}

function getViewportSize(isMobileDevice = false) {
  if (typeof window === 'undefined') {
    return { width: 1, height: 1 };
  }

  const pageZoomBaseline = getBrowserPageZoomBaseline();
  const pageZoomScale = getPageZoomScale(
    window.devicePixelRatio,
    pageZoomBaseline.devicePixelRatio,
    getPageZoomViewport(),
    pageZoomBaseline.viewport
  );
  const layoutViewport = isMobileDevice
    ? getBrowserLayoutViewportSize()
    : { width: window.innerWidth, height: window.innerHeight };

  return {
    width: layoutViewport.width * pageZoomScale,
    height: layoutViewport.height * pageZoomScale
  };
}

export function useScreenLayout(scale: number) {
  const { isMobileDevice } = useResponsiveDevice();
  const effectiveArea = useAtomValue(effectiveScreenAreaAtom);
  const sourceSize = useAtomValue(screenSourceSizeAtom);
  const borderEnabled = useAtomValue(portalBorderEnabledAtom);
  const borderColor = useAtomValue(portalBorderColorAtom);
  const portalSelecting = useAtomValue(portalSelectingAtom);
  const [viewport, setViewport] = useState(() => getViewportSize(isMobileDevice));

  useEffect(() => {
    const handleResize = () => {
      if (isMobileDevice && isVisualViewportZoomed()) return;
      setViewport(getViewportSize(isMobileDevice));
    };
    handleResize();
    window.addEventListener('resize', handleResize);
    window.visualViewport?.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      window.visualViewport?.removeEventListener('resize', handleResize);
    };
  }, [isMobileDevice]);

  const layoutViewport = useMemo(() => {
    const borderClearance = borderEnabled ? SCREEN_BORDER_CLEARANCE * 2 : 0;
    return {
      width: Math.max(1, viewport.width - borderClearance),
      height: Math.max(1, viewport.height - borderClearance)
    };
  }, [borderEnabled, viewport]);

  const layout = useMemo(() => {
    const area = isScreenCropReady(effectiveArea)
      ? effectiveArea
      : sourceSize.width > 0 && sourceSize.height > 0
        ? {
            sourceWidth: sourceSize.width,
            sourceHeight: sourceSize.height,
            contentRect: { left: 0, top: 0, width: sourceSize.width, height: sourceSize.height },
            contentAspectRatio: sourceSize.width / sourceSize.height,
            updatedAt: 0
          }
        : null;
    return getScreenCropLayout(area, layoutViewport.width, layoutViewport.height);
  }, [effectiveArea, layoutViewport, sourceSize.height, sourceSize.width]);

  const requestedScale = portalSelecting ? PORTAL_SELECTION_SCALE : scale;

  const scaledWrapperStyle = layout?.wrapperStyle
    ? {
        ...layout.wrapperStyle,
        transform: `scale(${requestedScale})`,
        transformOrigin: 'center center',
        outline: borderEnabled ? `3px solid ${borderColor}` : 'none',
        outlineOffset: '2px',
        boxShadow: borderEnabled
          ? `0 0 0 1px #001018, 0 0 14px 2px ${toRgba(borderColor, 0.8)}`
          : 'none'
      }
    : undefined;

  return {
    cropReady: !!layout,
    viewportStyle: {
      width: `${viewport.width}px`,
      height: `${viewport.height}px`,
      position: 'fixed',
      left: '50%',
      top: '50%',
      transform: 'translate(-50%, -50%)'
    } as CSSProperties,
    wrapperStyle: scaledWrapperStyle as CSSProperties | undefined,
    screenScale: requestedScale,
    mediaStyle: (layout?.mediaStyle ?? {
      transform: `scale(${requestedScale})`
    }) as CSSProperties
  };
}

function isVisualViewportZoomed() {
  const scale = window.visualViewport?.scale;
  return typeof scale === 'number' && Number.isFinite(scale) && Math.abs(scale - 1) > 0.01;
}

function toRgba(hex: string, alpha: number) {
  const value = Number.parseInt(hex.slice(1), 16);
  return `rgba(${(value >> 16) & 255}, ${(value >> 8) & 255}, ${value & 255}, ${alpha})`;
}

export function usePortalDetectionFromElement(
  ref: RefObject<HTMLVideoElement | HTMLCanvasElement>
) {
  const portalMode = useAtomValue(portalModeAtom);
  const videoStatus = useAtomValue(videoStatusAtom);
  const setDetectedArea = useSetAtom(detectedScreenAreaAtom);
  const setDetectionPhase = useSetAtom(portalDetectionPhaseAtom);
  const scratchCanvasRef = useRef<HTMLCanvasElement | null>(null);
  const trackerRef = useRef(new PortalDetectionTracker());

  useEffect(() => {
    if (portalMode !== 'auto' || videoStatus !== VideoStatus.Normal) {
      scratchCanvasRef.current = null;
      trackerRef.current.reset();
      setDetectedArea(initialScreenCropState);
      setDetectionPhase('idle');
      return;
    }

    let stopped = false;
    let timer = 0;

    const detect = () => {
      if (stopped) {
        return;
      }

      try {
        const element = ref.current;
        const sourceWidth =
          element instanceof HTMLVideoElement ? element.videoWidth : (element?.width ?? 0);
        const sourceHeight =
          element instanceof HTMLVideoElement ? element.videoHeight : (element?.height ?? 0);

        if (!element || sourceWidth <= 0 || sourceHeight <= 0) {
          return;
        }

        const scale = Math.min(
          1,
          PORTAL_DETECTION_SAMPLE_MAX / Math.max(sourceWidth, sourceHeight)
        );
        const sampleWidth = Math.max(1, Math.round(sourceWidth * scale));
        const sampleHeight = Math.max(1, Math.round(sourceHeight * scale));

        const canvas = scratchCanvasRef.current ?? document.createElement('canvas');
        scratchCanvasRef.current = canvas;
        canvas.width = sampleWidth;
        canvas.height = sampleHeight;

        const ctx = canvas.getContext('2d', { willReadFrequently: true });
        if (!ctx) {
          return;
        }

        ctx.drawImage(element, 0, 0, sampleWidth, sampleHeight);
        const imageData = ctx.getImageData(0, 0, sampleWidth, sampleHeight);
        const nextArea = detectPortalEffectiveAreaFromImageData(
          imageData,
          sampleWidth,
          sampleHeight
        );
        const contentSignature = getPortalContentSignature(imageData, nextArea);

        const tracked = trackerRef.current.update(
          {
            ...nextArea,
            sourceWidth,
            sourceHeight,
            contentRect: nextArea.contentRect
              ? {
                  left: Math.round(nextArea.contentRect.left / scale),
                  top: Math.round(nextArea.contentRect.top / scale),
                  width: Math.round(nextArea.contentRect.width / scale),
                  height: Math.round(nextArea.contentRect.height / scale)
                }
              : null,
            contentAspectRatio: nextArea.contentRect
              ? nextArea.contentRect.width / nextArea.contentRect.height
              : sourceWidth / sourceHeight
          },
          contentSignature
        );
        setDetectedArea(tracked.area);
        setDetectionPhase(tracked.phase);
      } catch (err) {
        console.log(err);
      } finally {
        timer = window.setTimeout(detect, PORTAL_DETECTION_INTERVAL_MS);
      }
    };

    timer = window.setTimeout(detect, 500);

    return () => {
      stopped = true;
      window.clearTimeout(timer);
    };
  }, [portalMode, ref, setDetectedArea, setDetectionPhase, videoStatus]);

  useEffect(() => {
    const tracker = trackerRef.current;
    return () => {
      tracker.reset();
      setDetectedArea(initialScreenCropState);
      setDetectionPhase('idle');
    };
  }, [setDetectedArea, setDetectionPhase]);
}

export function useScreenSourceSize(ref: RefObject<HTMLVideoElement | HTMLCanvasElement>) {
  const setSourceSize = useSetAtom(screenSourceSizeAtom);

  useEffect(() => {
    const update = () => {
      const element = ref.current;
      const width =
        element instanceof HTMLVideoElement ? element.videoWidth : (element?.width ?? 0);
      const height =
        element instanceof HTMLVideoElement ? element.videoHeight : (element?.height ?? 0);
      if (width > 0 && height > 0) {
        setSourceSize({ width, height });
      }
    };

    const element = ref.current;
    update();
    element?.addEventListener('loadedmetadata', update);
    element?.addEventListener('resize', update);

    return () => {
      element?.removeEventListener('loadedmetadata', update);
      element?.removeEventListener('resize', update);
      setSourceSize({ width: 0, height: 0 });
    };
  }, [ref, setSourceSize]);
}
