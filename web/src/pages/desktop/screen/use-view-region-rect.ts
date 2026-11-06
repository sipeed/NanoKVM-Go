import { useEffect, useState } from 'react';
import { useAtomValue } from 'jotai';

import { VideoStatus } from '@/types';
import { NO_SIGNAL_SCREEN_ID } from '@/lib/no-signal-screen.ts';
import {
  getRenderedScreenRect,
  isScreenCropReady,
  selectScreenSourceSize
} from '@/lib/screen-crop.ts';
import {
  edidResolutionAtom,
  effectiveScreenAreaAtom,
  screenViewportPanAtom,
  screenSourceSizeAtom,
  videoStatusAtom
} from '@/jotai/screen.ts';

export type ViewRegionRect = { left: number; top: number; width: number; height: number };

export function useViewRegionRect() {
  const videoStatus = useAtomValue(videoStatusAtom);
  const edidResolution = useAtomValue(edidResolutionAtom);
  const effectiveScreenArea = useAtomValue(effectiveScreenAreaAtom);
  const sourceSize = useAtomValue(screenSourceSizeAtom);
  const screenViewportPan = useAtomValue(screenViewportPanAtom);
  const [rect, setRect] = useState<ViewRegionRect | null>(null);

  useEffect(() => {
    let frame = 0;
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let observer: ResizeObserver | undefined;

    const update = () => {
      const isNoSignal = videoStatus === VideoStatus.NoImage;
      const screen = document.getElementById(isNoSignal ? NO_SIGNAL_SCREEN_ID : 'screen');
      if (!screen) {
        retryTimer = setTimeout(update, 100);
        return;
      }

      let nextRect: ViewRegionRect | null;
      if (isNoSignal) {
        nextRect = screen.getBoundingClientRect();
      } else if (isScreenCropReady(effectiveScreenArea) && screen.parentElement) {
        nextRect = screen.parentElement.getBoundingClientRect();
      } else {
        const intrinsicSize = getScreenElementSize(screen);
        const inputSourceSize = selectScreenSourceSize(sourceSize, intrinsicSize, edidResolution);
        nextRect = getRenderedScreenRect(
          screen.getBoundingClientRect(),
          inputSourceSize?.width ?? 0,
          inputSourceSize?.height ?? 0
        );
      }

      if (!nextRect?.width || !nextRect.height) {
        setRect(null);
        return;
      }
      setRect({
        left: nextRect.left,
        top: nextRect.top,
        width: nextRect.width,
        height: nextRect.height
      });

      if (!observer) {
        observer = new ResizeObserver(scheduleUpdate);
        observer.observe(screen);
        if (screen.parentElement) observer.observe(screen.parentElement);
      }
    };

    const scheduleUpdate = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(update);
    };

    update();
    window.addEventListener('resize', scheduleUpdate);
    window.addEventListener('scroll', scheduleUpdate, true);
    window.visualViewport?.addEventListener('resize', scheduleUpdate);
    window.visualViewport?.addEventListener('scroll', scheduleUpdate);
    return () => {
      cancelAnimationFrame(frame);
      if (retryTimer) clearTimeout(retryTimer);
      observer?.disconnect();
      window.removeEventListener('resize', scheduleUpdate);
      window.removeEventListener('scroll', scheduleUpdate, true);
      window.visualViewport?.removeEventListener('resize', scheduleUpdate);
      window.visualViewport?.removeEventListener('scroll', scheduleUpdate);
    };
  }, [
    edidResolution,
    effectiveScreenArea,
    screenViewportPan,
    sourceSize,
    videoStatus
  ]);

  return rect;
}

function getScreenElementSize(element: HTMLElement) {
  if (element instanceof HTMLVideoElement) {
    return { width: element.videoWidth, height: element.videoHeight };
  }
  if (element instanceof HTMLCanvasElement) {
    return {
      width: Number(element.dataset.mediaWidth) || element.width,
      height: Number(element.dataset.mediaHeight) || element.height
    };
  }
  return { width: 0, height: 0 };
}
