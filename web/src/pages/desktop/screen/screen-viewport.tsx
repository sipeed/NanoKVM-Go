import {
  useEffect,
  useLayoutEffect,
  useRef,
  type CSSProperties,
  type ReactNode,
  type PointerEvent as ReactPointerEvent
} from 'react';
import { useAtom } from 'jotai';

import { screenViewportPanAtom } from '@/jotai/screen.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

type ScreenViewportProps = {
  contentStyle: CSSProperties;
  screenScale: number;
  children: ReactNode;
};

type ScreenViewportContentProps = ScreenViewportProps & {
  isMobileDevice: boolean;
};

type PanGesture = {
  pointerId: number;
  clientX: number;
  clientY: number;
  panX: number;
  panY: number;
  minX: number;
  maxX: number;
  minY: number;
  maxY: number;
};

type Rect = {
  left: number;
  top: number;
  width: number;
  height: number;
};

export const ScreenViewport = ({ contentStyle, screenScale, children }: ScreenViewportProps) => {
  const { isMobileDevice } = useResponsiveDevice();

  // Keep the media node mounted while the viewport switches between desktop and mobile layouts.
  return (
    <ScreenViewportContent
      contentStyle={contentStyle}
      screenScale={screenScale}
      isMobileDevice={isMobileDevice}
    >
      {children}
    </ScreenViewportContent>
  );
};

function getMobileContentStyle(contentStyle: CSSProperties): CSSProperties {
  return {
    ...contentStyle,
    position: 'absolute',
    left: '50%',
    top: '50%',
    transform: 'translate(-50%, -50%)'
  };
}

const ScreenViewportContent = ({
  contentStyle,
  screenScale,
  children,
  isMobileDevice
}: ScreenViewportContentProps) => {
  const [pan, setPan] = useAtom(screenViewportPanAtom);
  const viewportRef = useRef<HTMLDivElement>(null);
  const stageRef = useRef<HTMLDivElement>(null);
  const panRef = useRef(pan);
  const gestureRef = useRef<PanGesture | null>(null);
  panRef.current = pan;

  useLayoutEffect(() => {
    if (isMobileDevice) return;
    setPan((current) => clampPan(current, viewportRef.current, stageRef.current));
  }, [contentStyle.height, contentStyle.width, isMobileDevice, screenScale, setPan]);

  useEffect(() => {
    if (isMobileDevice) return;
    const viewport = viewportRef.current;
    const stage = stageRef.current;
    if (!viewport || !stage) return;

    const resizeObserver = new ResizeObserver(() => {
      setPan((current) => clampPan(current, viewport, stage));
    });
    resizeObserver.observe(viewport);
    resizeObserver.observe(stage);
    const screen = stage.querySelector<HTMLElement>('#screen');
    if (screen?.parentElement) resizeObserver.observe(screen.parentElement);
    resizeObserver.observe(screen ?? stage);

    return () => resizeObserver.disconnect();
  }, [isMobileDevice, setPan]);

  function startPan(event: ReactPointerEvent<HTMLDivElement>) {
    const viewport = viewportRef.current;
    const stage = stageRef.current;
    if (!viewport || !stage || event.pointerType !== 'mouse' || event.button !== 0) return;

    const bounds = getPanBounds(viewport, stage, panRef.current);
    gestureRef.current = {
      pointerId: event.pointerId,
      clientX: event.clientX,
      clientY: event.clientY,
      panX: panRef.current.x,
      panY: panRef.current.y,
      ...bounds
    };
    viewport.setPointerCapture(event.pointerId);
    viewport.style.cursor = 'grabbing';
    event.preventDefault();
  }

  function updatePan(event: ReactPointerEvent<HTMLDivElement>) {
    const viewport = viewportRef.current;
    const gesture = gestureRef.current;
    if (!viewport || !gesture || gesture.pointerId !== event.pointerId) return;

    setPan({
      x: clamp(gesture.panX + event.clientX - gesture.clientX, gesture.minX, gesture.maxX),
      y: clamp(gesture.panY + event.clientY - gesture.clientY, gesture.minY, gesture.maxY)
    });
    event.preventDefault();
  }

  function finishPan(event: ReactPointerEvent<HTMLDivElement>) {
    const viewport = viewportRef.current;
    const gesture = gestureRef.current;
    if (!viewport || !gesture || gesture.pointerId !== event.pointerId) return;

    if (viewport.hasPointerCapture(event.pointerId)) {
      viewport.releasePointerCapture(event.pointerId);
    }
    viewport.style.cursor = '';
    gestureRef.current = null;
  }

  return (
    <div
      ref={viewportRef}
      className={
        isMobileDevice
          ? 'relative h-screen w-screen overflow-visible bg-black'
          : 'fixed inset-0 overflow-hidden overscroll-contain bg-black'
      }
      onPointerDown={isMobileDevice ? undefined : startPan}
      onPointerMove={isMobileDevice ? undefined : updatePan}
      onPointerUp={isMobileDevice ? undefined : finishPan}
      onPointerCancel={isMobileDevice ? undefined : finishPan}
    >
      <div
        ref={stageRef}
        className="absolute flex items-center justify-center overflow-visible"
        style={
          isMobileDevice
            ? getMobileContentStyle(contentStyle)
            : {
                ...contentStyle,
                left: '50%',
                top: '50%',
                transform: `translate(-50%, -50%) translate(${pan.x}px, ${pan.y}px)`
              }
        }
      >
        {children}
      </div>
    </div>
  );
};

function getPanBounds(
  viewport: HTMLDivElement,
  stage: HTMLDivElement,
  currentPan: { x: number; y: number }
) {
  const viewportRect = viewport.getBoundingClientRect();
  const screenRect = getScreenRect(stage);
  if (!screenRect) return { minX: 0, maxX: 0, minY: 0, maxY: 0 };

  const baseLeft = screenRect.left - currentPan.x;
  const baseTop = screenRect.top - currentPan.y;
  const hasHorizontalOverflow = screenRect.width > viewportRect.width;
  const hasVerticalOverflow = screenRect.height > viewportRect.height;

  return {
    minX: hasHorizontalOverflow ? viewportRect.right - baseLeft - screenRect.width : 0,
    maxX: hasHorizontalOverflow ? viewportRect.left - baseLeft : 0,
    minY: hasVerticalOverflow ? viewportRect.bottom - baseTop - screenRect.height : 0,
    maxY: hasVerticalOverflow ? viewportRect.top - baseTop : 0
  };
}

function clampPan(
  pan: { x: number; y: number },
  viewport: HTMLDivElement | null,
  stage: HTMLDivElement | null
) {
  if (!viewport || !stage) return pan;
  const bounds = getPanBounds(viewport, stage, pan);
  return {
    x: clamp(pan.x, bounds.minX, bounds.maxX),
    y: clamp(pan.y, bounds.minY, bounds.maxY)
  };
}

function getScreenRect(stage: HTMLDivElement): Rect | null {
  const screen = stage.querySelector<HTMLElement>('#screen');
  if (!screen) return null;

  const parentRect = screen.parentElement?.getBoundingClientRect();
  if (parentRect?.width && parentRect.height) {
    return {
      left: parentRect.left,
      top: parentRect.top,
      width: parentRect.width,
      height: parentRect.height
    };
  }

  const screenRect = screen.getBoundingClientRect();
  if (!screenRect.width || !screenRect.height) return null;
  return {
    left: screenRect.left,
    top: screenRect.top,
    width: screenRect.width,
    height: screenRect.height
  };
}

function clamp(value: number, min: number, max: number) {
  return Math.max(min, Math.min(value, max));
}
