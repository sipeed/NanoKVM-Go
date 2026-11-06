import { useCallback, useEffect, useState, type CSSProperties } from 'react';
import { useAtomValue } from 'jotai';
import { createPortal } from 'react-dom';

import { VideoStatus } from '@/types';
import {
  INPUT_REGION_HOST_ATTRIBUTE,
  INPUT_REGION_ID,
  registerInputRegion
} from '@/lib/input-region.ts';
import { NO_SIGNAL_SCREEN_ID } from '@/lib/no-signal-screen.ts';
import { isPreviewModeAtom } from '@/jotai/preview.ts';
import { videoStatusAtom } from '@/jotai/screen.ts';

import { useViewRegionRect, type ViewRegionRect } from '../screen/use-view-region-rect.ts';

export const InputRegion = () => {
  const rect = useViewRegionRect();
  const isPreviewMode = useAtomValue(isPreviewModeAtom);
  const videoStatus = useAtomValue(videoStatusAtom);
  const [host, setHost] = useState<HTMLElement | null>(null);

  const inputRegionRef = useCallback((element: HTMLDivElement | null) => {
    registerInputRegion(element);
  }, []);

  useEffect(() => {
    let retryTimer: ReturnType<typeof setTimeout> | undefined;
    let observer: MutationObserver | undefined;

    const update = () => {
      if (retryTimer) clearTimeout(retryTimer);
      const nextHost = findInputRegionHost(videoStatus);
      setHost((currentHost) => (currentHost === nextHost ? currentHost : nextHost));
      if (!nextHost) retryTimer = setTimeout(update, 100);
    };

    update();
    if (typeof MutationObserver !== 'undefined') {
      observer = new MutationObserver(update);
      observer.observe(document.body, { childList: true, subtree: true });
    }

    return () => {
      if (retryTimer) clearTimeout(retryTimer);
      observer?.disconnect();
    };
  }, [videoStatus]);

  if (!rect || !host || isPreviewMode) return null;

  return createPortal(
    <div
      ref={inputRegionRef}
      id={INPUT_REGION_ID}
      className="pointer-events-auto absolute z-20 bg-transparent"
      style={getInputRegionStyle(host, rect)}
    />,
    host
  );
};

function findInputRegionHost(videoStatus: VideoStatus) {
  const targetId = videoStatus === VideoStatus.NoImage ? NO_SIGNAL_SCREEN_ID : 'screen';
  const target = document.getElementById(targetId);
  if (!(target instanceof HTMLElement)) return null;
  if (target.hasAttribute(INPUT_REGION_HOST_ATTRIBUTE)) return target;
  return target.closest<HTMLElement>(`[${INPUT_REGION_HOST_ATTRIBUTE}]`);
}

function getInputRegionStyle(host: HTMLElement, rect: ViewRegionRect): CSSProperties {
  const hostRect = host.getBoundingClientRect();
  const scaleX = getElementScale(hostRect.width, host.clientWidth);
  const scaleY = getElementScale(hostRect.height, host.clientHeight);

  return {
    left: `${(rect.left - hostRect.left) / scaleX}px`,
    top: `${(rect.top - hostRect.top) / scaleY}px`,
    width: `${rect.width / scaleX}px`,
    height: `${rect.height / scaleY}px`
  };
}

function getElementScale(renderedLength: number, layoutLength: number) {
  const scale = layoutLength > 0 ? renderedLength / layoutLength : 1;
  return Number.isFinite(scale) && scale > 0 ? scale : 1;
}
