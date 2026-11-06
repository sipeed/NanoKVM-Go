import { PointerEvent as ReactPointerEvent, useEffect, useMemo, useRef, useState } from 'react';
import { Button } from 'antd';
import { useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { feedback } from '@/lib/feedback.tsx';
import {
  calibrationTargetsFromRect,
  createCalibrationFromPointPairs,
  isFourPointCalibrationUsable
} from '@/lib/input-profile.ts';
import type {
  CalibrationCorner,
  CalibrationTargetRect,
  NormalizedPoint
} from '@/lib/input-profile.ts';
import { getInputRegionElement, subscribeInputRegion } from '@/lib/input-region.ts';
import { createInputViewport, mapContentPointToTouchPort } from '@/lib/input-viewport.ts';
import { LAYER_OVERLAY, LAYER_OVERLAY_BACKDROP, LAYER_OVERLAY_HINT } from '@/lib/layers.ts';
import {
  calibrationRawPointAtom,
  inputCalibrationScreenAreaAtom,
  inputCalibrationSessionAtom,
  inputProfileStoreAtom,
  setProfileCalibration
} from '@/jotai/input-profile.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { effectiveScreenAreaAtom } from '@/jotai/screen.ts';

type FrameRect = { left: number; top: number; width: number; height: number };
type ResizeHandle = 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w' | 'nw';
type Operation =
  | { type: 'move'; startX: number; startY: number; initial: CalibrationTargetRect }
  | {
      type: 'resize';
      startX: number;
      startY: number;
      initial: CalibrationTargetRect;
      handle: ResizeHandle;
    };

const CORNERS: CalibrationCorner[] = ['top-left', 'top-right', 'bottom-right', 'bottom-left'];
const MINIMUM_SIZE = 0.1;

export const CalibrationOverlay = () => {
  const { t } = useTranslation();
  const session = useAtomValue(inputCalibrationSessionAtom);
  const rawPoint = useAtomValue(calibrationRawPointAtom);
  const store = useAtomValue(inputProfileStoreAtom);
  const effectiveScreenArea = useAtomValue(effectiveScreenAreaAtom);
  const setSession = useSetAtom(inputCalibrationSessionAtom);
  const setRawPoint = useSetAtom(calibrationRawPointAtom);
  const setStore = useSetAtom(inputProfileStoreAtom);
  const setKeyboardEnabled = useSetAtom(isKeyboardEnableAtom);
  const setCalibrationScreenArea = useSetAtom(inputCalibrationScreenAreaAtom);
  const [frameRect, setFrameRect] = useState<FrameRect | null>(null);
  const [inputRegion, setInputRegion] = useState<HTMLElement | null>(getInputRegionElement);
  const [error, setError] = useState('');
  const rootRef = useRef<HTMLDivElement>(null);
  const operationRef = useRef<Operation | null>(null);
  const sessionRef = useRef(session);
  const sessionWasActiveRef = useRef(false);
  const screenAreaRef = useRef(effectiveScreenArea);
  const confirmSampleRef = useRef<() => void>(() => undefined);
  const sessionActive = Boolean(session);
  sessionRef.current = session;
  if (!sessionActive || !sessionWasActiveRef.current) {
    screenAreaRef.current = effectiveScreenArea;
  }
  sessionWasActiveRef.current = sessionActive;

  const targetPoints = useMemo(
    () => (session ? calibrationTargetsFromRect(session.targetRect) : null),
    [session]
  );

  useEffect(() => {
    if (!sessionActive) return;
    const screenArea = {
      ...screenAreaRef.current,
      contentRect: screenAreaRef.current.contentRect
        ? { ...screenAreaRef.current.contentRect }
        : null
    };
    screenAreaRef.current = screenArea;
    setCalibrationScreenArea(screenArea);
    setKeyboardEnabled(false);

    return () => {
      setCalibrationScreenArea(null);
      setKeyboardEnabled(true);
    };
  }, [sessionActive, setCalibrationScreenArea, setKeyboardEnabled]);

  useEffect(() => {
    return subscribeInputRegion(setInputRegion);
  }, []);

  useEffect(() => {
    if (!sessionActive || !inputRegion) return;

    const update = () => {
      const rect = inputRegion.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) {
        setFrameRect({ left: rect.left, top: rect.top, width: rect.width, height: rect.height });
      }
    };

    update();
    const observer = new ResizeObserver(update);
    observer.observe(inputRegion);
    window.addEventListener('resize', update);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', update);
      setFrameRect(null);
    };
  }, [sessionActive, inputRegion]);

  useEffect(() => {
    if (!sessionActive) return;
    const handleKey = (event: KeyboardEvent) => {
      const current = sessionRef.current;
      if (!current) return;
      const confirms = current.phase === 'capture' && (event.key === 'Enter' || event.key === ' ');
      if (event.key !== 'Escape' && !confirms) return;
      event.preventDefault();
      event.stopPropagation();
      event.stopImmediatePropagation();
      if (event.type === 'keyup') return;
      if (event.repeat) return;
      if (event.key === 'Escape') {
        setError('');
        setRawPoint(null);
        setSession(null);
      } else confirmSampleRef.current();
    };

    window.addEventListener('keydown', handleKey, true);
    window.addEventListener('keyup', handleKey, true);
    return () => {
      window.removeEventListener('keydown', handleKey, true);
      window.removeEventListener('keyup', handleKey, true);
    };
  }, [sessionActive, setRawPoint, setSession]);

  if (!session || !frameRect || !targetPoints) return null;

  function cancel() {
    setError('');
    setRawPoint(null);
    setSession(null);
  }

  function beginCapture() {
    if (!session) return;
    const viewport = createInputViewport(screenAreaRef.current);
    const contentTargets = calibrationTargetsFromRect(session.targetRect);
    const controlPoints = {} as Record<CalibrationCorner, NormalizedPoint>;
    for (const corner of CORNERS) {
      const contentPoint = contentTargets[corner];
      controlPoints[corner] = mapContentPointToTouchPort(
        viewport,
        contentPoint.x,
        contentPoint.y
      ).source;
    }
    setError('');
    setRawPoint(null);
    setSession({
      ...session,
      phase: 'capture',
      cornerIndex: 0,
      controlPoints,
      hidSamples: {}
    });
  }

  function confirmSample() {
    const current = sessionRef.current;
    if (!current || current.phase !== 'capture' || !rawPoint) return;
    const corner = CORNERS[current.cornerIndex];
    const hidSamples = { ...current.hidSamples, [corner]: { ...rawPoint } };
    if (current.cornerIndex < CORNERS.length - 1) {
      setRawPoint(null);
      setSession({ ...current, cornerIndex: current.cornerIndex + 1, hidSamples });
      return;
    }

    const profile = store.profiles.find((item) => item.id === current.profileId);
    if (!profile || profile.origin !== 'user' || !current.controlPoints) {
      setError(t('mouse.calibration.customRequired'));
      return;
    }
    const calibration = createCalibrationFromPointPairs(
      current.controlPoints,
      hidSamples as Record<CalibrationCorner, NormalizedPoint>
    );
    if (!isFourPointCalibrationUsable(calibration)) {
      setError(t('mouse.calibration.invalidSamples'));
      return;
    }
    const nextStore = setProfileCalibration(store, profile.id, current.mode, calibration);
    setStore(nextStore);
    setRawPoint(null);
    setSession(null);
    feedback.success(t('mouse.calibration.saved'));
  }
  confirmSampleRef.current = confirmSample;

  function capturePointer(event: ReactPointerEvent) {
    rootRef.current?.setPointerCapture(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
  }

  function handleMoveStart(event: ReactPointerEvent<HTMLDivElement>) {
    const current = sessionRef.current;
    if (!current) return;
    operationRef.current = {
      type: 'move',
      startX: event.clientX,
      startY: event.clientY,
      initial: current.targetRect
    };
    capturePointer(event);
  }

  function handleResizeStart(event: ReactPointerEvent<HTMLDivElement>, handle: ResizeHandle) {
    const current = sessionRef.current;
    if (!current) return;
    operationRef.current = {
      type: 'resize',
      startX: event.clientX,
      startY: event.clientY,
      initial: current.targetRect,
      handle
    };
    capturePointer(event);
  }

  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    const operation = operationRef.current;
    const current = sessionRef.current;
    if (!operation || !current || !frameRect) return;
    const deltaX = (event.clientX - operation.startX) / frameRect.width;
    const deltaY = (event.clientY - operation.startY) / frameRect.height;
    const targetRect =
      operation.type === 'move'
        ? moveRect(operation.initial, deltaX, deltaY)
        : resizeRect(operation.initial, operation.handle, deltaX, deltaY);
    setSession({ ...current, targetRect });
  }

  function handlePointerEnd(event: ReactPointerEvent<HTMLDivElement>) {
    operationRef.current = null;
    if (rootRef.current?.hasPointerCapture(event.pointerId)) {
      rootRef.current.releasePointerCapture(event.pointerId);
    }
  }

  const currentCorner = CORNERS[session.cornerIndex];
  const selectionStyle = {
    left: session.targetRect.left * frameRect.width,
    top: session.targetRect.top * frameRect.height,
    width: session.targetRect.width * frameRect.width,
    height: session.targetRect.height * frameRect.height
  };

  return (
    <>
      {outsideBlockers(frameRect).map((style, index) => (
        <div
          key={index}
          className="fixed touch-none bg-transparent"
          style={{ ...style, zIndex: LAYER_OVERLAY_BACKDROP }}
          onPointerDown={blockEvent}
          onPointerMove={blockEvent}
          onPointerUp={blockEvent}
        />
      ))}
      {session.phase === 'position' ? (
        <div
          ref={rootRef}
          className="fixed touch-none bg-black/20"
          style={{ ...frameRect, zIndex: LAYER_OVERLAY }}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerEnd}
          onPointerCancel={handlePointerEnd}
        >
          <div
            className="absolute cursor-move border-2 border-amber-400 bg-amber-400/10 shadow-[0_0_18px_rgba(251,191,36,0.75)]"
            style={selectionStyle}
            onPointerDown={handleMoveStart}
          >
            {resizeHandles.map(({ handle, className, cursor, label }) => (
              <div
                key={handle}
                className={`absolute flex items-center justify-center rounded-full border-2 border-white font-bold text-white ${
                  label ? 'h-7 w-7 bg-red-600 text-xs' : 'h-3 w-3 bg-amber-500 text-[0px]'
                } ${className}`}
                style={{ cursor }}
                onPointerDown={(event) => handleResizeStart(event, handle)}
              >
                {label}
              </div>
            ))}
          </div>
        </div>
      ) : (
        <div className="pointer-events-none fixed" style={{ ...frameRect, zIndex: LAYER_OVERLAY }}>
          {CORNERS.map((corner, index) => {
            const point = targetPoints[corner];
            const active = corner === currentCorner;
            return (
              <div
                key={corner}
                className={`absolute flex h-8 w-8 -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full border-2 text-sm font-bold shadow-lg ${
                  active
                    ? 'border-white bg-red-600 text-white shadow-red-500/80'
                    : 'border-white/80 bg-neutral-800/70 text-white/70'
                }`}
                style={{ left: point.x * frameRect.width, top: point.y * frameRect.height }}
              >
                {index + 1}
              </div>
            );
          })}
        </div>
      )}

      <div
        className="fixed bottom-5 left-1/2 w-[min(92vw,680px)] -translate-x-1/2 rounded-lg bg-neutral-900/95 px-4 py-3 text-center text-white shadow-xl"
        style={{ zIndex: LAYER_OVERLAY_HINT }}
      >
        <div className="text-sm font-medium">
          {session.phase === 'position'
            ? t('mouse.calibration.positionHint')
            : t('mouse.calibration.captureHint', {
                current: session.cornerIndex + 1,
                total: CORNERS.length
              })}
        </div>
        {session.phase === 'capture' && (
          <>
            <div className="mt-1 text-xs text-amber-300">
              {t('mouse.calibration.feedbackRequired')}
            </div>
            <div className="mt-1 text-xs text-neutral-300">
              {t('mouse.calibration.confirmHint')}{' '}
              {rawPoint
                ? `HID (${rawPoint.x.toFixed(4)}, ${rawPoint.y.toFixed(4)})`
                : t('mouse.calibration.waitingInput')}
            </div>
          </>
        )}
        {error && <div className="mt-2 text-sm text-red-400">{error}</div>}
        <div className="mt-3 flex justify-center gap-2">
          <Button onClick={cancel}>{t('auth.cancel')}</Button>
          {session.phase === 'position' ? (
            <Button type="primary" onClick={beginCapture}>
              {t('mouse.calibration.next')}
            </Button>
          ) : (
            <Button type="primary" disabled={!rawPoint} onClick={confirmSample}>
              {t('mouse.calibration.confirmPoint')}
            </Button>
          )}
        </div>
      </div>
    </>
  );
};

const resizeHandles: Array<{
  handle: ResizeHandle;
  className: string;
  cursor: string;
  label?: number;
}> = [
  { handle: 'nw', className: '-left-3.5 -top-3.5', cursor: 'nwse-resize', label: 1 },
  { handle: 'n', className: 'left-1/2 -top-1.5 -translate-x-1/2', cursor: 'ns-resize' },
  { handle: 'ne', className: '-right-3.5 -top-3.5', cursor: 'nesw-resize', label: 2 },
  { handle: 'e', className: '-right-1.5 top-1/2 -translate-y-1/2', cursor: 'ew-resize' },
  { handle: 'se', className: '-bottom-3.5 -right-3.5', cursor: 'nwse-resize', label: 3 },
  { handle: 's', className: '-bottom-1.5 left-1/2 -translate-x-1/2', cursor: 'ns-resize' },
  { handle: 'sw', className: '-bottom-3.5 -left-3.5', cursor: 'nesw-resize', label: 4 },
  { handle: 'w', className: '-left-1.5 top-1/2 -translate-y-1/2', cursor: 'ew-resize' }
];

function moveRect(rect: CalibrationTargetRect, deltaX: number, deltaY: number) {
  return {
    ...rect,
    left: clamp(rect.left + deltaX, 0, 1 - rect.width),
    top: clamp(rect.top + deltaY, 0, 1 - rect.height)
  };
}

function resizeRect(
  rect: CalibrationTargetRect,
  handle: ResizeHandle,
  deltaX: number,
  deltaY: number
) {
  let left = rect.left;
  let top = rect.top;
  let right = rect.left + rect.width;
  let bottom = rect.top + rect.height;
  if (handle.includes('w')) left += deltaX;
  if (handle.includes('e')) right += deltaX;
  if (handle.includes('n')) top += deltaY;
  if (handle.includes('s')) bottom += deltaY;
  left = clamp(left, 0, right - MINIMUM_SIZE);
  right = clamp(right, left + MINIMUM_SIZE, 1);
  top = clamp(top, 0, bottom - MINIMUM_SIZE);
  bottom = clamp(bottom, top + MINIMUM_SIZE, 1);
  return { left, top, width: right - left, height: bottom - top };
}

function clamp(value: number, minimum: number, maximum: number) {
  return Math.max(minimum, Math.min(maximum, value));
}

function outsideBlockers(frame: FrameRect): FrameRect[] {
  const right = frame.left + frame.width;
  const bottom = frame.top + frame.height;
  return [
    { left: 0, top: 0, width: window.innerWidth, height: frame.top },
    { left: 0, top: bottom, width: window.innerWidth, height: window.innerHeight - bottom },
    { left: 0, top: frame.top, width: frame.left, height: frame.height },
    { left: right, top: frame.top, width: window.innerWidth - right, height: frame.height }
  ].filter((rect) => rect.width > 0 && rect.height > 0);
}

function blockEvent(event: ReactPointerEvent) {
  event.preventDefault();
  event.stopPropagation();
}
