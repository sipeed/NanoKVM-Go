import { useEffect, useRef, useState } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { applyFourPointCalibration } from '@/lib/input-profile.ts';
import type { FourPointCalibration } from '@/lib/input-profile.ts';
import { getInputRegionElement, subscribeInputRegion } from '@/lib/input-region.ts';
import {
  INPUT_ROUTER_MAX_CONTACTS,
  INPUT_ROUTER_RELEASE_TOUCHSCREEN,
  inputRouter,
  InputSource,
  type BrowserTouchCapture,
  type BrowserTouchPoint,
  type InputCaptureReference,
  type TouchscreenContact
} from '@/lib/input-router.ts';
import { createTouchPort, mapClientPointToTouchPort } from '@/lib/input-viewport.ts';
import type { TouchPort } from '@/lib/input-viewport.ts';
import {
  shouldSendTouchscreenReport,
  TOUCHSCREEN_HEARTBEAT_INTERVAL_MS,
  TOUCHSCREEN_REPORT_INTERVAL_MS,
  touchscreenReportKey
} from '@/lib/touchscreen-report.ts';
import { activeCalibrationAtom } from '@/jotai/active-input-calibration.ts';
import { calibrationRawPointAtom, inputCalibrationSessionAtom } from '@/jotai/input-profile.ts';
import { isPreviewModeAtom } from '@/jotai/preview.ts';
import {
  effectiveScreenAreaAtom,
  portalInputActiveAtom,
  portalInputScreenAreaAtom
} from '@/jotai/screen.ts';

import {
  recordBrowserInputCapture,
  subscribePointerEvents,
  type CapturedInputEvent,
  type CapturedPointerEvent
} from './pointer-events-cap.ts';

const MINIMUM_CONTACT_DURATION_MS = TOUCHSCREEN_REPORT_INTERVAL_MS * 2;

type ActiveContact = TouchscreenContact & {
  source: InputSource;
  slot: number;
  downAt: number;
  releasing: boolean;
  releaseTimer?: ReturnType<typeof setTimeout>;
};

export const TouchScreen = () => {
  const isPreviewMode = useAtomValue(isPreviewModeAtom);
  const effectiveScreenArea = useAtomValue(effectiveScreenAreaAtom);
  const activeCalibration = useAtomValue(activeCalibrationAtom);
  const calibrationSession = useAtomValue(inputCalibrationSessionAtom);
  const setCalibrationRawPoint = useSetAtom(calibrationRawPointAtom);
  const setPortalInputActive = useSetAtom(portalInputActiveAtom);
  const setPortalInputScreenArea = useSetAtom(portalInputScreenAreaAtom);
  const [inputRegion, setInputRegion] = useState<HTMLElement | null>(getInputRegionElement);
  const contactsRef = useRef(new Map<string, ActiveContact>());
  const pendingCaptureRef = useRef<InputCaptureReference | null>(null);
  const isPreviewModeRef = useRef(isPreviewMode);
  const effectiveScreenAreaRef = useRef(effectiveScreenArea);
  const frozenTouchPortRef = useRef<TouchPort | null>(null);
  const frozenCalibrationRef = useRef<FourPointCalibration | null>(null);
  const activeCalibrationRef = useRef<FourPointCalibration | null>(activeCalibration);
  const calibrationSessionRef = useRef(calibrationSession);
  const setCalibrationRawPointRef = useRef(setCalibrationRawPoint);

  isPreviewModeRef.current = isPreviewMode;
  effectiveScreenAreaRef.current = effectiveScreenArea;
  activeCalibrationRef.current = activeCalibration;
  calibrationSessionRef.current = calibrationSession;
  setCalibrationRawPointRef.current = setCalibrationRawPoint;

  useEffect(() => {
    return subscribeInputRegion(setInputRegion);
  }, []);

  useEffect(() => {
    if (isPreviewMode) return;
    if (!inputRegion) return;

    const eventTarget = inputRegion;
    const previousTouchAction = eventTarget.style.touchAction;
    const touchOptions = { passive: false } as const;
    let reportFrame: number | undefined;
    let heartbeatTimer: ReturnType<typeof setInterval> | undefined;
    let lastReportKey = '';
    let lastReportAt = 0;
    let lastSource = InputSource.BrowserTouchscreen;
    eventTarget.style.touchAction = 'none';
    const unsubscribePointerEvents = subscribePointerEvents(handlePointerInput);

    function handleTouchStart(event: TouchEvent) {
      const capture = captureTouchEvent(event);
      disableEvent(event);
      pendingCaptureRef.current = capture.sequence ? capture : null;
      lastSource = InputSource.BrowserTouchscreen;
      freezeScreenArea(eventTarget);
      updateActiveTouches(event.targetTouches, true);
      sendActiveContacts();
      startReports();
    }

    function handleTouchMove(event: TouchEvent) {
      if (contactsRef.current.size === 0) return;
      const capture = captureTouchEvent(event);
      disableEvent(event);
      pendingCaptureRef.current = capture.sequence ? capture : null;
      lastSource = InputSource.BrowserTouchscreen;
      updateActiveTouches(event.targetTouches, false);
    }

    function handleTouchEnd(event: TouchEvent) {
      handleTouchRelease(event, false);
    }

    function handleTouchCancel(event: TouchEvent) {
      handleTouchRelease(event, true);
    }

    function handleTouchRelease(event: TouchEvent, immediate: boolean) {
      const capture = captureTouchEvent(event);
      disableEvent(event);
      pendingCaptureRef.current = capture.sequence ? capture : null;
      lastSource = InputSource.BrowserTouchscreen;
      updateActiveTouches(event.targetTouches, false);
      for (let index = 0; index < event.changedTouches.length; index += 1) {
        const touch = event.changedTouches.item(index);
        if (touch) releaseContact(touchKey(touch.identifier), touch, immediate);
      }
    }

    function handlePointerInput(event: CapturedInputEvent) {
      if (event.type === 'wheel' || event.type === 'pointerleave' || event.pointerType === 'touch') {
        return;
      }
      pendingCaptureRef.current = event.captureSequence
        ? { sequence: event.captureSequence, capturedAt: event.capturedAt }
        : null;
      lastSource = event.source;
      const key = pointerKey(event.pointerId);
      if (event.type === 'pointerdown') {
        freezeScreenArea(event.target);
        updatePointerContact(key, event, true);
        sendActiveContacts();
        startReports();
        return;
      }
      if (event.type === 'pointermove') {
        updatePointerContact(key, event, false);
        return;
      }
      releaseContact(key, event, event.type === 'pointercancel');
    }

    function updateActiveTouches(touches: TouchList, createMissing: boolean) {
      const touchPort = activeTouchPort(eventTarget);
      for (let index = 0; index < touches.length; index += 1) {
        const touch = touches.item(index);
        if (!touch) continue;
        const key = touchKey(touch.identifier);
        let contact = contactsRef.current.get(key);
        if (!contact && createMissing) {
          contact = createContact(InputSource.BrowserTouchscreen);
          if (!contact) continue;
          contactsRef.current.set(key, contact);
        }
        if (!contact || contact.releasing) continue;
        updateContact(contact, touch, touchPort, touchPressure(touch));
      }
    }

    function updatePointerContact(
      key: string,
      event: CapturedPointerEvent,
      createMissing: boolean
    ) {
      let contact = contactsRef.current.get(key);
      if (!contact && createMissing) {
        contact = createContact(event.source);
        if (!contact) return;
        contactsRef.current.set(key, contact);
      }
      if (!contact || contact.releasing) return;
      updateContact(contact, event, activeTouchPort(event.target), pointerPressure(event));
    }

    function createContact(source: InputSource) {
      const slot = allocateSlot();
      if (slot < 0) return undefined;
      return {
        source,
        slot,
        id: slot + 1,
        x: 0,
        y: 0,
        touching: true,
        inRange: true,
        pressure: 255,
        downAt: performance.now(),
        releasing: false
      } satisfies ActiveContact;
    }

    function updateContact(
      contact: ActiveContact,
      point: Pick<Touch, 'clientX' | 'clientY'>,
      touchPort: TouchPort,
      pressure: number
    ) {
      const normalized = normalizedTouchPoint(
        point,
        touchPort,
        frozenCalibrationRef.current ?? activeCalibrationRef.current
      );
      if (calibrationSessionRef.current?.phase === 'capture') {
        setCalibrationRawPointRef.current(normalized.source);
      }
      contact.x = normalized.output.x;
      contact.y = normalized.output.y;
      contact.pressure = pressure;
    }

    function releaseContact(
      key: string,
      point: Pick<Touch, 'clientX' | 'clientY'>,
      immediate: boolean
    ) {
      const contact = contactsRef.current.get(key);
      if (!contact || contact.releasing) return;
      updateContact(contact, point, activeTouchPort(eventTarget), 0);
      contact.releasing = true;
      const remaining = immediate
        ? 0
        : MINIMUM_CONTACT_DURATION_MS - (performance.now() - contact.downAt);
      if (remaining <= 0) {
        finishRelease(key);
      } else {
        contact.releaseTimer = setTimeout(() => finishRelease(key), remaining);
      }
    }

    function finishRelease(key: string) {
      const contact = contactsRef.current.get(key);
      if (!contact) return;
      if (contact.releaseTimer) clearTimeout(contact.releaseTimer);
      contact.touching = false;
      contact.pressure = 0;
      sendActiveContacts();
      contactsRef.current.delete(key);
      sendActiveContacts(true);
      if (contactsRef.current.size === 0) {
        stopReports();
        releaseScreenArea();
      }
    }

    function clearContacts() {
      stopReports();
      for (const contact of contactsRef.current.values()) {
        if (contact.releaseTimer) clearTimeout(contact.releaseTimer);
        contact.touching = false;
        contact.pressure = 0;
      }
      if (contactsRef.current.size > 0) sendActiveContacts(true);
      contactsRef.current.clear();
      pendingCaptureRef.current = null;
      sendActiveContacts(true);
      releaseScreenArea();
    }

    function allocateSlot() {
      const occupied = new Set(Array.from(contactsRef.current.values(), (contact) => contact.slot));
      for (let slot = 0; slot < INPUT_ROUTER_MAX_CONTACTS; slot += 1) {
        if (!occupied.has(slot)) return slot;
      }
      return -1;
    }

    function sendActiveContacts(force = false) {
      if (isPreviewModeRef.current && !force) return;
      const contacts = Array.from(contactsRef.current.values(), (contact) => ({
        id: contact.id,
        touching: contact.touching,
        inRange: contact.inRange,
        x: contact.x,
        y: contact.y,
        pressure: contact.pressure
      }));
      const reportKey = touchscreenReportKey(contacts);
      const now = performance.now();
      if (!force && !shouldSendTouchscreenReport(reportKey, lastReportKey, now - lastReportAt)) {
        return;
      }
      const source =
        Array.from(contactsRef.current.values()).find((contact) => contact.touching)?.source ??
        lastSource;
      const capture = pendingCaptureRef.current;
      const sent = inputRouter.sendTouchscreen(
        source,
        contacts,
        capture?.capturedAt,
        capture ?? undefined
      );
      if (sent && pendingCaptureRef.current === capture) pendingCaptureRef.current = null;
      lastReportKey = reportKey;
      lastReportAt = now;
    }

    function startReports() {
      if (!reportFrame) reportFrame = requestAnimationFrame(reportFrameCallback);
      if (!heartbeatTimer) {
        heartbeatTimer = setInterval(sendActiveContacts, TOUCHSCREEN_HEARTBEAT_INTERVAL_MS);
      }
    }

    function reportFrameCallback() {
      reportFrame = undefined;
      if (contactsRef.current.size === 0) return;
      sendActiveContacts();
      reportFrame = requestAnimationFrame(reportFrameCallback);
    }

    function stopReports() {
      if (reportFrame) cancelAnimationFrame(reportFrame);
      reportFrame = undefined;
      if (heartbeatTimer) clearInterval(heartbeatTimer);
      heartbeatTimer = undefined;
    }

    function activeTouchPort(target: HTMLElement) {
      return (
        frozenTouchPortRef.current ??
        createTouchPort(target.getBoundingClientRect(), effectiveScreenAreaRef.current)
      );
    }

    function freezeScreenArea(target: HTMLElement) {
      if (frozenTouchPortRef.current) return;
      const area = effectiveScreenAreaRef.current;
      const frozenScreenArea = {
        ...area,
        contentRect: area.contentRect ? { ...area.contentRect } : null
      };
      frozenTouchPortRef.current = createTouchPort(
        target.getBoundingClientRect(),
        frozenScreenArea
      );
      frozenCalibrationRef.current = activeCalibrationRef.current;
      setPortalInputScreenArea(frozenScreenArea);
      setPortalInputActive(true);
    }

    function releaseScreenArea() {
      frozenTouchPortRef.current = null;
      frozenCalibrationRef.current = null;
      setPortalInputActive(false);
    }

    eventTarget.addEventListener('touchstart', handleTouchStart, touchOptions);
    eventTarget.addEventListener('touchmove', handleTouchMove, touchOptions);
    eventTarget.addEventListener('touchend', handleTouchEnd, touchOptions);
    eventTarget.addEventListener('touchcancel', handleTouchCancel, touchOptions);
    window.addEventListener('blur', clearContacts);
    window.addEventListener('pagehide', clearContacts);

    return () => {
      unsubscribePointerEvents();
      clearContacts();
      inputRouter.sendPointerRelease(
        InputSource.BrowserTouchscreen,
        INPUT_ROUTER_RELEASE_TOUCHSCREEN
      );
      eventTarget.style.touchAction = previousTouchAction;
      eventTarget.removeEventListener('touchstart', handleTouchStart);
      eventTarget.removeEventListener('touchmove', handleTouchMove);
      eventTarget.removeEventListener('touchend', handleTouchEnd);
      eventTarget.removeEventListener('touchcancel', handleTouchCancel);
      window.removeEventListener('blur', clearContacts);
      window.removeEventListener('pagehide', clearContacts);
    };
  }, [inputRegion, isPreviewMode, setPortalInputActive, setPortalInputScreenArea]);

  return null;
};

function captureTouchEvent(event: TouchEvent) {
  return recordBrowserInputCapture((sequence, capturedAt) =>
    snapshotTouchEvent(event, sequence, capturedAt)
  );
}

function snapshotTouchEvent(
  event: TouchEvent,
  sequence: number,
  capturedAt: number
): BrowserTouchCapture {
  return {
    api: 'TouchEvent',
    sequence,
    capturedAtMS: Math.round(capturedAt) >>> 0,
    type: event.type,
    content: {
      touches: snapshotTouchList(event.touches),
      targetTouches: snapshotTouchList(event.targetTouches),
      changedTouches: snapshotTouchList(event.changedTouches),
      ctrlKey: event.ctrlKey,
      shiftKey: event.shiftKey,
      altKey: event.altKey,
      metaKey: event.metaKey,
      timeStamp: finiteNumber(event.timeStamp),
      cancelable: event.cancelable,
      composed: event.composed,
      targetId: event.currentTarget instanceof HTMLElement ? event.currentTarget.id : ''
    }
  };
}

function snapshotTouchList(touches: TouchList) {
  const points: BrowserTouchPoint[] = [];
  for (let index = 0; index < touches.length; index += 1) {
    const touch = touches.item(index);
    if (!touch) continue;
    points.push({
      identifier: touch.identifier,
      clientX: finiteNumber(touch.clientX),
      clientY: finiteNumber(touch.clientY),
      pageX: finiteNumber(touch.pageX),
      pageY: finiteNumber(touch.pageY),
      screenX: finiteNumber(touch.screenX),
      screenY: finiteNumber(touch.screenY),
      radiusX: finiteNumber(touch.radiusX),
      radiusY: finiteNumber(touch.radiusY),
      rotationAngle: finiteNumber(touch.rotationAngle),
      force: finiteNumber(touch.force)
    });
  }
  return points;
}

function normalizedTouchPoint(
  point: Pick<Touch, 'clientX' | 'clientY'>,
  touchPort: TouchPort,
  calibration: FourPointCalibration | null
) {
  const source = mapClientPointToTouchPort(touchPort, point.clientX, point.clientY).source;
  return { source, output: applyFourPointCalibration(source, calibration) };
}

function touchPressure(touch: Touch) {
  return Math.round(Math.max(0, Math.min(1, touch.force || 1)) * 255);
}

function pointerPressure(pointer: CapturedPointerEvent) {
  if (pointer.pointerType === 'mouse') return 255;
  return Math.round(Math.max(0, Math.min(1, pointer.pressure || 1)) * 255);
}

function touchKey(identifier: number) {
  return `touch:${identifier}`;
}

function pointerKey(pointerId: number) {
  return `pointer:${pointerId}`;
}

function finiteNumber(value: number | undefined) {
  return Number.isFinite(value) ? Number(value) : 0;
}

function disableEvent(event: Event) {
  event.preventDefault();
  event.stopPropagation();
}
