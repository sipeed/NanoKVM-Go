import { useEffect, useLayoutEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { applyFourPointCalibration } from '@/lib/input-profile.ts';
import type { FourPointCalibration } from '@/lib/input-profile.ts';
import {
  INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER,
  INPUT_ROUTER_RELEASE_RELATIVE_POINTER,
  INPUT_ROUTER_RELEASE_STYLUS,
  inputRouter,
  InputSource,
  PointerButtonState
} from '@/lib/input-router.ts';
import {
  createInputViewport,
  createTouchPort,
  mapClientPointToTouchPort,
  mapContentPointToTouchPort,
  mapLocalPointToTouchPort
} from '@/lib/input-viewport.ts';
import type { TouchPort } from '@/lib/input-viewport.ts';
import { activeCalibrationAtom } from '@/jotai/active-input-calibration.ts';
import { calibrationRawPointAtom, inputCalibrationSessionAtom } from '@/jotai/input-profile.ts';
import { mouseModeAtom, scrollDirectionAtom, scrollIntervalAtom } from '@/jotai/mouse.ts';
import { isPreviewModeAtom } from '@/jotai/preview.ts';
import {
  effectiveScreenAreaAtom,
  portalInputActiveAtom,
  portalInputScreenAreaAtom
} from '@/jotai/screen.ts';

import {
  subscribePointerEvents,
  type CapturedInputEvent,
  type CapturedPointerEvent
} from './pointer-events-cap.ts';

enum MouseButton {
  Left = 0,
  Middle = 1,
  Right = 2,
  Back = 3,
  Forward = 4
}

export const Absolute = () => {
  const mouseMode = useAtomValue(mouseModeAtom);
  const scrollDirection = useAtomValue(scrollDirectionAtom);
  const scrollInterval = useAtomValue(scrollIntervalAtom);
  const isPreviewMode = useAtomValue(isPreviewModeAtom);
  const effectiveScreenArea = useAtomValue(effectiveScreenAreaAtom);
  const activeCalibration = useAtomValue(activeCalibrationAtom);
  const calibrationSession = useAtomValue(inputCalibrationSessionAtom);
  const setCalibrationRawPoint = useSetAtom(calibrationRawPointAtom);
  const setPortalInputActive = useSetAtom(portalInputActiveAtom);
  const setPortalInputScreenArea = useSetAtom(portalInputScreenAreaAtom);
  const useStylusReports = mouseMode === 'digitizer-stylus';

  const buttonsRef = useRef(new PointerButtonState());
  const activePointerIdRef = useRef<number | null>(null);
  const mouseModeRef = useRef(mouseMode);
  const sendPointerStateRef = useRef(sendPointerState);
  const outputStylusRef = useRef(useStylusReports);
  const isPreviewModeRef = useRef(isPreviewMode);
  const effectiveScreenAreaRef = useRef(effectiveScreenArea);
  const latestTouchPortRef = useRef<TouchPort | null>(null);
  const frozenTouchPortRef = useRef<TouchPort | null>(null);
  const activeCalibrationRef = useRef<FourPointCalibration | null>(activeCalibration);
  const calibrationSessionRef = useRef(calibrationSession);
  const setCalibrationRawPointRef = useRef(setCalibrationRawPoint);
  const frozenCalibrationRef = useRef<FourPointCalibration | null>(null);
  const scrollDirectionRef = useRef(scrollDirection);
  const scrollIntervalRef = useRef(scrollInterval);
  const lastScrollTimeRef = useRef(0);
  // null until a real pointer event provides a position; force-sent release
  // reports must never fabricate a coordinate (e.g. screen center) that the
  // host never observed.
  const lastPosRef = useRef<{ x: number; y: number } | null>(null);
  const mousePressedButtonsRef = useRef(new Set<number>());
  const relativeProxyButtonsRef = useRef(0);
  const touchStartTimeRef = useRef(0);
  const longPressTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const isLongPressRef = useRef(false);
  const hasMoveRef = useRef(false);
  const isDraggingRef = useRef(false);
  const pressedButtonRef = useRef<MouseButton | null>(null);
  const touchStartPosRef = useRef({ x: 0, y: 0 });

  mouseModeRef.current = mouseMode;
  sendPointerStateRef.current = sendPointerState;
  isPreviewModeRef.current = isPreviewMode;
  effectiveScreenAreaRef.current = effectiveScreenArea;
  activeCalibrationRef.current = activeCalibration;
  calibrationSessionRef.current = calibrationSession;
  setCalibrationRawPointRef.current = setCalibrationRawPoint;
  scrollDirectionRef.current = scrollDirection;
  scrollIntervalRef.current = scrollInterval;

  useLayoutEffect(() => {
    const wasStylus = outputStylusRef.current;
    if (wasStylus === useStylusReports) return;

    clearLongPressTimer();
    buttonsRef.current.reset();
    mousePressedButtonsRef.current.clear();
    pressedButtonRef.current = null;
    activePointerIdRef.current = null;
    isLongPressRef.current = false;
    hasMoveRef.current = false;
    isDraggingRef.current = false;
    releaseRelativePointerProxy();
    if (wasStylus && lastPosRef.current) {
      sendPointerStateRef.current(
        wasStylus,
        InputSource.BrowserPointer,
        lastPosRef.current.x,
        lastPosRef.current.y,
        0,
        false,
        true
      );
    }
    frozenTouchPortRef.current = null;
    frozenCalibrationRef.current = null;
    setPortalInputActive(false);
    outputStylusRef.current = useStylusReports;
  }, [setPortalInputActive, useStylusReports]);

  useEffect(() => {
    if (isPreviewMode) return;

    const unsubscribe = subscribePointerEvents(handleInputEvent);
    return () => {
      unsubscribe();
      releaseAllMouseButtons();
      clearLongPressTimer();
      inputRouter.sendPointerRelease(InputSource.BrowserPointer, relinquishPointerRoutes());
    };

    function handleInputEvent(event: CapturedInputEvent) {
      if (isPreviewModeRef.current) return;
      if (event.type === 'wheel') {
        handleWheel(event);
        return;
      }

      if (event.type === 'pointerleave') {
        if (activePointerIdRef.current === null && event.pointerType !== 'touch') {
          releaseAbsolutePointer();
          inputRouter.sendPointerRelease(event.source, relinquishPointerRoutes());
        }
        return;
      }

      if (event.type === 'pointerdown') {
        if (event.pointerType === 'touch' && activePointerIdRef.current !== null) return;
        activePointerIdRef.current = event.pointerId;
        freezeScreenArea(event.target);
        updateLastPosition(event);
        if (event.pointerType === 'touch') {
          startTouchPointer(event);
        } else if (event.button >= 0) {
          pressMouseButton(event.button, event.source);
        }
        return;
      }

      if (event.type === 'pointermove') {
        if (event.pointerType === 'touch') {
          if (activePointerIdRef.current !== event.pointerId) return;
          moveTouchPointer(event);
          return;
        }
        syncMouseButtonsFromEvent(event.buttons, event.source);
        const { x, y } = getCoordinate(event);
        handleMouseEvent({ type: 'move', x, y }, event.source);
        return;
      }

      if (event.pointerType === 'touch' && activePointerIdRef.current !== event.pointerId) return;
      updateLastPosition(event);
      if (event.pointerType === 'touch') {
        if (event.type === 'pointercancel') cancelTouchPointer();
        else finishTouchPointer(event.source);
      } else if (event.button >= 0) {
        releaseMouseButton(event.button, event.source);
      } else {
        releaseAllMouseButtons();
      }
      activePointerIdRef.current = null;
      releaseFrozenScreenAreaIfIdle();
    }
  }, [isPreviewMode, setPortalInputActive, setPortalInputScreenArea]);

  function handleMouseEvent(
    event:
      | { type: 'mousedown'; button: number }
      | { type: 'mouseup'; button: number }
      | { type: 'wheel'; deltaY: number }
      | { type: 'move'; x: number; y: number }
      | { type: 'release' },
    source: InputSource = InputSource.BrowserPointer
  ) {
    const stylus = outputStylusRef.current;
    const buttons = buttonsRef.current;
    switch (event.type) {
      case 'mousedown':
        buttons.down(event.button);
        // EXPERIMENT(hybrid click): send button transitions through the relative
        // pointer; some hosts (iPadOS AssistiveTouch, some BIOS) ignore or
        // mishandle buttons carried by the absolute device
        if (!stylus) {
          sendRelativePointerProxy(source, buttons.value);
          return;
        }
        break;
      case 'mouseup':
        buttons.up(event.button);
        if (!stylus) {
          sendRelativePointerProxy(source, buttons.value);
          return;
        }
        break;
      case 'wheel':
        if (!stylus) {
          sendRelativePointerProxy(source, buttons.value, 0, 0, event.deltaY > 0 ? 1 : -1);
        }
        return;
      case 'move': {
        lastPosRef.current = { x: event.x, y: event.y };
        sendPointerState(stylus, source, event.x, event.y);
        return;
      }
      default:
        buttons.reset();
        if (!stylus) {
          releaseRelativePointerProxy(source);
        }
        if (stylus && lastPosRef.current) {
          sendPointerState(
            stylus,
            source,
            lastPosRef.current.x,
            lastPosRef.current.y,
            0,
            false,
            true
          );
        }
        return;
    }

    if (lastPosRef.current) {
      sendPointerState(stylus, source, lastPosRef.current.x, lastPosRef.current.y);
    }
  }

  function sendPointerState(
    stylus: boolean,
    source: InputSource,
    x: number,
    y: number,
    wheel = 0,
    inRange = true,
    force = false
  ) {
    if (isPreviewModeRef.current && !force) return;
    // EXPERIMENT(hybrid click): the absolute report always carries buttons=0;
    // button state lives on the relative pointer (see handleMouseEvent)
    const buttons = stylus ? buttonsRef.current.value : 0;
    const viewport =
      (frozenTouchPortRef.current ?? latestTouchPortRef.current)?.viewport ??
      createInputViewport(effectiveScreenAreaRef.current);
    const point = mapContentPointToTouchPort(viewport, x, y).source;
    const calibration = frozenCalibrationRef.current ?? activeCalibrationRef.current;
    publishCalibrationPoint(point, inRange);
    const calibrated = applyFourPointCalibration(point, calibration);

    if (stylus) {
      const touching = buttons !== 0;
      const barrel = (buttons & 0x02) !== 0;
      const state = inRange ? 0x02 | (touching ? 0x01 : 0) | (barrel ? 0x04 : 0) : 0;
      inputRouter.sendStylus(source, state, calibrated.x, calibrated.y, touching ? 0xff : 0);
    } else {
      inputRouter.sendAbsolutePointer(source, buttons, calibrated.x, calibrated.y, wheel);
    }
  }

  function startTouchPointer(event: CapturedPointerEvent) {
    touchStartTimeRef.current = Date.now();
    isLongPressRef.current = false;
    hasMoveRef.current = false;
    isDraggingRef.current = false;
    pressedButtonRef.current = null;
    touchStartPosRef.current = { x: event.clientX, y: event.clientY };
    clearLongPressTimer();

    const { x, y } = getCoordinate(event);
    handleMouseEvent({ type: 'move', x, y }, event.source);
    handleMouseEvent({ type: 'mousedown', button: MouseButton.Left }, event.source);
    pressedButtonRef.current = MouseButton.Left;
    longPressTimerRef.current = setTimeout(() => {
      longPressTimerRef.current = null;
      handleMouseEvent({ type: 'mouseup', button: MouseButton.Left }, event.source);
      isLongPressRef.current = true;
      pressedButtonRef.current = MouseButton.Right;
      if (navigator.vibrate) navigator.vibrate(50);
      handleMouseEvent({ type: 'mousedown', button: MouseButton.Right }, event.source);
    }, 800);
  }

  function moveTouchPointer(event: CapturedPointerEvent) {
    const deltaX = Math.abs(event.clientX - touchStartPosRef.current.x);
    const deltaY = Math.abs(event.clientY - touchStartPosRef.current.y);
    const distance = Math.hypot(deltaX, deltaY);
    const timeDelta = Date.now() - touchStartTimeRef.current;
    const velocity = timeDelta > 0 ? distance / timeDelta : 0;
    const shouldStartDrag = distance > 10 || (distance > 8 && velocity > 0.3);

    if (shouldStartDrag && !isDraggingRef.current && !isLongPressRef.current) {
      hasMoveRef.current = true;
      clearLongPressTimer();
      if (pressedButtonRef.current === null) {
        isDraggingRef.current = true;
        pressedButtonRef.current = MouseButton.Left;
        handleMouseEvent({ type: 'mousedown', button: MouseButton.Left }, event.source);
      }
    }
    if (distance > 8) hasMoveRef.current = true;
    const { x, y } = getCoordinate(event);
    handleMouseEvent({ type: 'move', x, y }, event.source);
  }

  function finishTouchPointer(source: InputSource) {
    clearLongPressTimer();
    if (pressedButtonRef.current !== null) {
      handleMouseEvent({ type: 'mouseup', button: pressedButtonRef.current }, source);
    }
    resetTouchState();
  }

  function cancelTouchPointer() {
    clearLongPressTimer();
    if (pressedButtonRef.current !== null) {
      handleMouseEvent(
        { type: 'mouseup', button: pressedButtonRef.current },
        InputSource.BrowserTouchscreen
      );
    }
    resetTouchState();
  }

  function resetTouchState() {
    isLongPressRef.current = false;
    hasMoveRef.current = false;
    isDraggingRef.current = false;
    pressedButtonRef.current = null;
    frozenTouchPortRef.current = null;
    frozenCalibrationRef.current = null;
    setPortalInputActive(false);
  }

  function handleWheel(event: Extract<CapturedInputEvent, { type: 'wheel' }>) {
    if (Math.floor(event.deltaY) === 0) return;
    const now = Date.now();
    if (now - lastScrollTimeRef.current < scrollIntervalRef.current) return;
    updateLastPosition(event);
    handleMouseEvent(
      { type: 'wheel', deltaY: event.deltaY * scrollDirectionRef.current },
      event.source
    );
    lastScrollTimeRef.current = now;
  }

  function pressMouseButton(button: number, source: InputSource) {
    mousePressedButtonsRef.current.add(button);
    handleMouseEvent({ type: 'mousedown', button }, source);
  }

  function releaseMouseButton(button: number, source: InputSource) {
    mousePressedButtonsRef.current.delete(button);
    handleMouseEvent({ type: 'mouseup', button }, source);
  }

  function releaseAllMouseButtons() {
    for (const button of mousePressedButtonsRef.current) {
      mousePressedButtonsRef.current.delete(button);
      handleMouseEvent({ type: 'mouseup', button });
    }
    handleMouseEvent({ type: 'release' });
    resetTouchState();
    activePointerIdRef.current = null;
  }

  function getCoordinate(
    event: Pick<CapturedInputEvent, 'clientX' | 'clientY' | 'target'> &
      Partial<Pick<CapturedPointerEvent, 'offsetX' | 'offsetY'>>
  ) {
    const touchPort =
      frozenTouchPortRef.current ??
      createTouchPort(event.target.getBoundingClientRect(), effectiveScreenAreaRef.current);
    latestTouchPortRef.current = touchPort;
    if (
      event.offsetX !== undefined &&
      event.offsetY !== undefined &&
      event.target.clientWidth > 0 &&
      event.target.clientHeight > 0
    ) {
      return mapLocalPointToTouchPort(
        touchPort,
        event.offsetX,
        event.offsetY,
        event.target.clientWidth,
        event.target.clientHeight
      ).content;
    }
    return mapClientPointToTouchPort(touchPort, event.clientX, event.clientY).content;
  }

  function updateLastPosition(
    event: Pick<CapturedInputEvent, 'clientX' | 'clientY' | 'target'> &
      Partial<Pick<CapturedPointerEvent, 'offsetX' | 'offsetY'>>
  ) {
    lastPosRef.current = getCoordinate(event);
  }

  function freezeScreenArea(target: HTMLElement) {
    if (frozenTouchPortRef.current) return;
    const screenArea = effectiveScreenAreaRef.current;
    const frozenScreenArea = {
      ...screenArea,
      contentRect: screenArea.contentRect ? { ...screenArea.contentRect } : null
    };
    frozenTouchPortRef.current = createTouchPort(target.getBoundingClientRect(), frozenScreenArea);
    latestTouchPortRef.current = frozenTouchPortRef.current;
    frozenCalibrationRef.current = activeCalibrationRef.current;
    setPortalInputScreenArea(frozenScreenArea);
    setPortalInputActive(true);
  }

  function releaseFrozenScreenAreaIfIdle() {
    if (mousePressedButtonsRef.current.size === 0 && pressedButtonRef.current === null) {
      frozenTouchPortRef.current = null;
      frozenCalibrationRef.current = null;
      setPortalInputActive(false);
    }
  }

  function syncMouseButtonsFromEvent(buttons: number, source: InputSource) {
    if (buttons === 0) {
      if (mousePressedButtonsRef.current.size > 0) releaseAllMouseButtons();
      return;
    }
    for (const button of mousePressedButtonsRef.current) {
      if (!isMouseButtonPressed(button, buttons)) releaseMouseButton(button, source);
    }
  }

  function isMouseButtonPressed(button: number, buttons: number) {
    const masks = [1, 4, 2, 8, 16];
    return (buttons & (masks[button] ?? 0)) !== 0;
  }

  function clearLongPressTimer() {
    if (!longPressTimerRef.current) return;
    clearTimeout(longPressTimerRef.current);
    longPressTimerRef.current = null;
  }

  function releaseAbsolutePointer() {
    buttonsRef.current.reset();
    releaseRelativePointerProxy();
    if (!outputStylusRef.current || !lastPosRef.current) return;
    sendPointerState(
      outputStylusRef.current,
      InputSource.BrowserPointer,
      lastPosRef.current.x,
      lastPosRef.current.y,
      0,
      false,
      true
    );
  }

  function relinquishPointerRoutes() {
    return (
      INPUT_ROUTER_RELEASE_RELATIVE_POINTER |
      (outputStylusRef.current
        ? INPUT_ROUTER_RELEASE_STYLUS
        : INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER)
    );
  }

  function sendRelativePointerProxy(
    source: InputSource,
    buttons: number,
    deltaX = 0,
    deltaY = 0,
    wheel = 0
  ) {
    relativeProxyButtonsRef.current = buttons;
    return inputRouter.sendRelativePointer(source, buttons, deltaX, deltaY, wheel);
  }

  function releaseRelativePointerProxy(source: InputSource = InputSource.BrowserPointer) {
    if (relativeProxyButtonsRef.current === 0) return;

    relativeProxyButtonsRef.current = 0;
    inputRouter.sendRelativePointer(source, 0);
  }

  function publishCalibrationPoint(point: { x: number; y: number }, inRange: boolean) {
    const session = calibrationSessionRef.current;
    if (!inRange || session?.phase !== 'capture' || session.mode !== mouseModeRef.current) return;
    setCalibrationRawPointRef.current(point);
  }

  return null;
};
