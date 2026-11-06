import { useEffect, useRef } from 'react';
import { useAtomValue } from 'jotai';
import { useTranslation } from 'react-i18next';

import { feedback } from '@/lib/feedback.tsx';
import {
  INPUT_ROUTER_RELEASE_RELATIVE_POINTER,
  inputRouter,
  InputSource,
  PointerButtonState
} from '@/lib/input-router.ts';
import { resolvePointerCapture } from '@/lib/pointer-capture.ts';
import { scrollDirectionAtom, scrollIntervalAtom } from '@/jotai/mouse.ts';
import { isPreviewModeAtom } from '@/jotai/preview.ts';

import {
  subscribePointerEvents,
  type CapturedInputEvent,
  type CapturedPointerEvent
} from './pointer-events-cap.ts';

const TOUCHPAD_DRAG_HOLD_DELAY = 800;
const TOUCHPAD_DRAG_BUTTON = 0;

export const Relative = () => {
  const { t } = useTranslation();
  const scrollDirection = useAtomValue(scrollDirectionAtom);
  const scrollInterval = useAtomValue(scrollIntervalAtom);
  const isPreviewMode = useAtomValue(isPreviewModeAtom);
  const pointerCapture = resolvePointerCapture('relative');
  const buttonsRef = useRef(new PointerButtonState());
  const isLockedRef = useRef(false);
  const isPreviewModeRef = useRef(isPreviewMode);
  const lastScrollTimeRef = useRef(0);
  const mouseLastPointRef = useRef<{ x: number; y: number } | null>(null);
  const touchPointersRef = useRef(new Map<number, { x: number; y: number }>());
  const touchStartPointRef = useRef({ x: 0, y: 0 });
  const touchLastPointRef = useRef<{ x: number; y: number } | null>(null);
  const touchMovedRef = useRef(false);
  const touchScrollingRef = useRef(false);
  const touchLongPressRef = useRef(false);
  const touchLongPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  isPreviewModeRef.current = isPreviewMode;

  useEffect(() => {
    if (isPreviewMode) return;

    const useNormalPointer = pointerCapture === 'normal-pointer';
    if (!useNormalPointer) {
      showMessage();
      document.addEventListener('pointerlockchange', handlePointerLockChange);
    } else if (document.pointerLockElement) {
      document.exitPointerLock();
      isLockedRef.current = false;
    }

    const unsubscribe = subscribePointerEvents(handleInputEvent);
    return () => {
      unsubscribe();
      document.removeEventListener('pointerlockchange', handlePointerLockChange);
      releaseState();
      inputRouter.sendPointerRelease(
        InputSource.BrowserPointer,
        INPUT_ROUTER_RELEASE_RELATIVE_POINTER
      );
    };

    function handleInputEvent(event: CapturedInputEvent) {
      if (isPreviewModeRef.current) return;
      if (event.type === 'wheel') {
        handleWheel(event);
        return;
      }
      if (event.type === 'pointerleave') {
        if (buttonsRef.current.value === 0) {
          inputRouter.sendPointerRelease(event.source, INPUT_ROUTER_RELEASE_RELATIVE_POINTER);
        }
        return;
      }

      if (useNormalPointer) {
        handleNormalPointer(event);
      } else {
        handleLockedPointer(event);
      }
    }

    function handleNormalPointer(event: CapturedPointerEvent) {
      if (event.type === 'pointerdown') {
        if (event.pointerType === 'touch') {
          const wasEmpty = touchPointersRef.current.size === 0;
          touchPointersRef.current.set(event.pointerId, {
            x: event.clientX,
            y: event.clientY
          });
          if (wasEmpty) {
            resetTouchGesture();
            touchStartPointRef.current = { x: event.clientX, y: event.clientY };
            touchLastPointRef.current = { x: event.clientX, y: event.clientY };
            touchLongPressTimerRef.current = setTimeout(() => {
              touchLongPressTimerRef.current = null;
              touchLongPressRef.current = true;
              sendMouseButton('down', TOUCHPAD_DRAG_BUTTON, event.source);
              if (navigator.vibrate) navigator.vibrate(50);
            }, TOUCHPAD_DRAG_HOLD_DELAY);
          } else {
            touchScrollingRef.current = true;
            clearTouchLongPressTimer();
          }
          return;
        }
        mouseLastPointRef.current = { x: event.clientX, y: event.clientY };
        if (event.button >= 0) sendMouseButton('down', event.button, event.source);
        return;
      }

      if (event.pointerType === 'touch') {
        handleNormalTouch(event);
        return;
      }
      if (event.type === 'pointermove') {
        const previous = mouseLastPointRef.current ?? { x: event.clientX, y: event.clientY };
        const deltaX = event.movementX || event.clientX - previous.x;
        const deltaY = event.movementY || event.clientY - previous.y;
        mouseLastPointRef.current = { x: event.clientX, y: event.clientY };
        sendRelativeMove(deltaX, deltaY, event.source);
      } else if (event.type === 'pointerup') {
        if (event.button >= 0) sendMouseButton('up', event.button, event.source);
        mouseLastPointRef.current = null;
      } else {
        releaseState();
      }
    }

    function handleNormalTouch(event: CapturedPointerEvent) {
      if (event.type === 'pointerdown') return;
      if (event.type === 'pointercancel') {
        touchPointersRef.current.delete(event.pointerId);
        releaseState();
        return;
      }
      const point = touchPointersRef.current.get(event.pointerId);
      if (!point) return;
      if (event.type === 'pointermove') {
        point.x = event.clientX;
        point.y = event.clientY;
        const current = getTouchCentroid();
        const previous = touchLastPointRef.current ?? current;
        if (touchPointersRef.current.size > 1 || touchScrollingRef.current) {
          touchScrollingRef.current = true;
          clearTouchLongPressTimer();
          sendTouchScroll(current.y - previous.y);
        } else {
          const totalX = current.x - touchStartPointRef.current.x;
          const totalY = current.y - touchStartPointRef.current.y;
          if (Math.hypot(totalX, totalY) > 8) {
            touchMovedRef.current = true;
            clearTouchLongPressTimer();
          }
          sendRelativeMove(current.x - previous.x, current.y - previous.y, event.source);
        }
        touchLastPointRef.current = current;
        return;
      }
      touchPointersRef.current.delete(event.pointerId);
      if (touchPointersRef.current.size > 0) {
        touchLastPointRef.current = getTouchCentroid();
      } else {
        finishTouchGesture(event.source);
      }
    }

    function handleLockedPointer(event: CapturedPointerEvent) {
      if (event.type === 'pointerdown') {
        if (event.button >= 0) sendMouseButton('down', event.button, event.source);
        if (!isLockedRef.current) void event.target.requestPointerLock();
      } else if (event.type === 'pointermove') {
        sendRelativeMove(event.movementX, event.movementY, event.source);
      } else if (event.type === 'pointerup') {
        if (event.button >= 0) sendMouseButton('up', event.button, event.source);
      } else {
        releaseState();
      }
    }

    function handlePointerLockChange() {
      isLockedRef.current = document.pointerLockElement !== null;
    }

    function handleWheel(event: Extract<CapturedInputEvent, { type: 'wheel' }>) {
      if (Math.floor(event.deltaY) === 0) return;
      const now = Date.now();
      if (now - lastScrollTimeRef.current < scrollInterval) return;
      const deltaY = (event.deltaY > 0 ? 1 : -1) * scrollDirection;
      sendMouseEvent({ type: 'wheel', deltaY }, event.source);
      lastScrollTimeRef.current = now;
    }

    function getTouchCentroid() {
      let x = 0;
      let y = 0;
      for (const point of touchPointersRef.current.values()) {
        x += point.x;
        y += point.y;
      }
      const count = Math.max(1, touchPointersRef.current.size);
      return { x: x / count, y: y / count };
    }

    function finishTouchGesture(source: InputSource) {
      clearTouchLongPressTimer();
      if (touchLongPressRef.current) {
        sendMouseButton('up', TOUCHPAD_DRAG_BUTTON, source);
      } else if (!touchMovedRef.current && !touchScrollingRef.current) {
        sendMouseButton('down', 0, source);
        sendMouseButton('up', 0, source);
      }
      resetTouchGesture();
    }

    function resetTouchGesture() {
      clearTouchLongPressTimer();
      touchLastPointRef.current = null;
      touchMovedRef.current = false;
      touchScrollingRef.current = false;
      touchLongPressRef.current = false;
    }

    function clearTouchLongPressTimer() {
      if (!touchLongPressTimerRef.current) return;
      clearTimeout(touchLongPressTimerRef.current);
      touchLongPressTimerRef.current = null;
    }

    function releaseState() {
      clearTouchLongPressTimer();
      touchPointersRef.current.clear();
      mouseLastPointRef.current = null;
      resetTouchGesture();
      sendMouseEvent({ type: 'release' }, InputSource.BrowserPointer);
    }

    function sendRelativeMove(deltaX: number, deltaY: number, source: InputSource) {
      const roundedX = Math.round(deltaX);
      const roundedY = Math.round(deltaY);
      if (roundedX === 0 && roundedY === 0) return;
      sendMouseEvent({ type: 'move', deltaX: roundedX, deltaY: roundedY }, source);
    }

    function sendTouchScroll(rawDeltaY: number) {
      if (Math.round(rawDeltaY) === 0) return;
      const now = Date.now();
      if (now - lastScrollTimeRef.current < scrollInterval) return;
      sendMouseEvent(
        { type: 'wheel', deltaY: (rawDeltaY > 0 ? 1 : -1) * scrollDirection },
        InputSource.BrowserTouchscreen
      );
      lastScrollTimeRef.current = now;
    }

    function sendMouseButton(action: 'down' | 'up', button: number, source: InputSource) {
      sendMouseEvent({ type: action === 'down' ? 'mousedown' : 'mouseup', button }, source);
    }
  }, [isPreviewMode, pointerCapture, scrollDirection, scrollInterval, t]);

  function sendMouseEvent(
    event:
      | { type: 'mousedown'; button: number }
      | { type: 'mouseup'; button: number }
      | { type: 'wheel'; deltaY: number }
      | { type: 'move'; deltaX: number; deltaY: number }
      | { type: 'release' },
    source: InputSource
  ) {
    const buttons = buttonsRef.current;
    switch (event.type) {
      case 'mousedown':
        buttons.down(event.button);
        break;
      case 'mouseup':
        buttons.up(event.button);
        break;
      case 'wheel':
        inputRouter.sendRelativePointer(source, buttons.value, 0, 0, event.deltaY);
        return;
      case 'move':
        inputRouter.sendRelativePointer(source, buttons.value, event.deltaX, event.deltaY);
        return;
      case 'release':
        buttons.reset();
        inputRouter.sendRelativePointer(source, buttons.value);
        return;
    }
    inputRouter.sendRelativePointer(source, buttons.value);
  }

  function showMessage() {
    feedback.info(t('mouse.requestPointer'), {
      key: 'requestPointer',
      style: { marginTop: '40vh' }
    });
  }

  return null;
};
