import { useEffect, useState } from 'react';
import { useAtomValue } from 'jotai';

import { getInputRegionElement, subscribeInputRegion } from '@/lib/input-region.ts';
import { InputSource, type BrowserPointerCapture } from '@/lib/input-router.ts';
import { mouseModeAtom } from '@/jotai/mouse.ts';
import { isPreviewModeAtom } from '@/jotai/preview.ts';

import {
  publishPointerEvent,
  recordBrowserInputCapture,
  type CapturedInputEvent,
  type CapturedPointerEvent
} from './pointer-events-cap.ts';

export const PointerEventsCap = () => {
  const isPreviewMode = useAtomValue(isPreviewModeAtom);
  const mouseMode = useAtomValue(mouseModeAtom);
  const [inputRegion, setInputRegion] = useState<HTMLElement | null>(getInputRegionElement);

  useEffect(() => {
    return subscribeInputRegion(setInputRegion);
  }, []);

  useEffect(() => {
    if (isPreviewMode) return;
    if (!inputRegion) return;

    const eventTarget = inputRegion;
    const previousTouchAction = eventTarget.style.touchAction;
    const useNativeTouchEvents = mouseMode === 'touch-screen';
    const allowNativeTouchGestures = !useNativeTouchEvents;
    const activePointers = new Map<number, PointerEvent>();
    const eventOptions = { passive: false } as const;
    eventTarget.style.touchAction = allowNativeTouchGestures ? 'pinch-zoom' : 'none';

    function emit(event: CapturedInputEvent) {
      publishPointerEvent(event);
    }

    function handlePointerEvent(event: PointerEvent) {
      if (useNativeTouchEvents && event.pointerType === 'touch') return;
      const capture = recordBrowserInputCapture((sequence, capturedAt) =>
        snapshotPointerEvent(event, sequence, capturedAt)
      );
      if (event.pointerType === 'touch' && allowNativeTouchGestures) {
        event.stopPropagation();
      } else {
        preventEvent(event);
      }
      if (event.type === 'pointerdown') {
        activePointers.set(event.pointerId, event);
        capturePointer(event.pointerId);
      } else if (event.type === 'pointermove' && activePointers.has(event.pointerId)) {
        activePointers.set(event.pointerId, event);
      } else if (event.type === 'pointerup' || event.type === 'pointercancel') {
        activePointers.delete(event.pointerId);
      }
      emit({
        type: event.type as CapturedPointerEvent['type'],
        source:
          event.pointerType === 'touch'
            ? InputSource.BrowserTouchscreen
            : InputSource.BrowserPointer,
        captureSequence: capture.sequence,
        capturedAt: capture.capturedAt,
        pointerId: event.pointerId,
        pointerType: event.pointerType,
        isPrimary: event.isPrimary,
        clientX: event.clientX,
        clientY: event.clientY,
        movementX: event.movementX,
        movementY: event.movementY,
        offsetX: event.offsetX,
        offsetY: event.offsetY,
        button: event.button,
        buttons: event.buttons,
        pressure: event.pressure,
        target: eventTarget
      });
      if (event.type === 'pointerup' || event.type === 'pointercancel') {
        releasePointer(event.pointerId);
      }
    }

    function handleWheel(event: WheelEvent) {
      const capturedAt = performance.now();
      preventEvent(event);
      emit({
        type: 'wheel',
        source: InputSource.BrowserPointer,
        captureSequence: 0,
        capturedAt,
        clientX: event.clientX,
        clientY: event.clientY,
        offsetX: event.offsetX,
        offsetY: event.offsetY,
        deltaX: event.deltaX,
        deltaY: event.deltaY,
        deltaZ: event.deltaZ,
        deltaMode: event.deltaMode,
        target: eventTarget
      });
    }

    function releaseActivePointers() {
      for (const event of activePointers.values()) {
        emit({
          type: 'pointercancel',
          source:
            event.pointerType === 'touch'
              ? InputSource.BrowserTouchscreen
              : InputSource.BrowserPointer,
          captureSequence: 0,
          capturedAt: performance.now(),
          pointerId: event.pointerId,
          pointerType: event.pointerType,
          isPrimary: event.isPrimary,
          clientX: event.clientX,
          clientY: event.clientY,
          movementX: 0,
          movementY: 0,
          offsetX: event.offsetX,
          offsetY: event.offsetY,
          button: -1,
          buttons: 0,
          pressure: 0,
          target: eventTarget
        });
        releasePointer(event.pointerId);
      }
      activePointers.clear();
    }

    function capturePointer(pointerId: number) {
      try {
        if (typeof eventTarget.setPointerCapture === 'function') {
          eventTarget.setPointerCapture(pointerId);
        }
      } catch {
        return;
      }
    }

    function releasePointer(pointerId: number) {
      try {
        if (
          typeof eventTarget.releasePointerCapture === 'function' &&
          (typeof eventTarget.hasPointerCapture !== 'function' ||
            eventTarget.hasPointerCapture(pointerId))
        ) {
          eventTarget.releasePointerCapture(pointerId);
        }
      } catch {
        return;
      }
    }

    eventTarget.addEventListener('pointerdown', handlePointerEvent, eventOptions);
    eventTarget.addEventListener('pointermove', handlePointerEvent, eventOptions);
    eventTarget.addEventListener('pointerup', handlePointerEvent, eventOptions);
    eventTarget.addEventListener('pointercancel', handlePointerEvent, eventOptions);
    eventTarget.addEventListener('pointerleave', handlePointerEvent, eventOptions);
    eventTarget.addEventListener('wheel', handleWheel, eventOptions);
    eventTarget.addEventListener('click', preventEvent);
    eventTarget.addEventListener('contextmenu', preventEvent);
    window.addEventListener('blur', releaseActivePointers);
    window.addEventListener('pagehide', releaseActivePointers);

    return () => {
      releaseActivePointers();
      eventTarget.style.touchAction = previousTouchAction;
      eventTarget.removeEventListener('pointerdown', handlePointerEvent);
      eventTarget.removeEventListener('pointermove', handlePointerEvent);
      eventTarget.removeEventListener('pointerup', handlePointerEvent);
      eventTarget.removeEventListener('pointercancel', handlePointerEvent);
      eventTarget.removeEventListener('pointerleave', handlePointerEvent);
      eventTarget.removeEventListener('wheel', handleWheel);
      eventTarget.removeEventListener('click', preventEvent);
      eventTarget.removeEventListener('contextmenu', preventEvent);
      window.removeEventListener('blur', releaseActivePointers);
      window.removeEventListener('pagehide', releaseActivePointers);
    };
  }, [inputRegion, isPreviewMode, mouseMode]);

  return null;
};

function preventEvent(event: Event) {
  event.preventDefault();
  event.stopPropagation();
}

function snapshotPointerEvent(
  event: PointerEvent,
  sequence: number,
  capturedAt: number
): BrowserPointerCapture {
  return {
    api: 'PointerEvent',
    sequence,
    capturedAtMS: Math.round(capturedAt) >>> 0,
    type: event.type,
    content: {
      pointerId: event.pointerId,
      pointerType: event.pointerType,
      isPrimary: event.isPrimary,
      clientX: finiteNumber(event.clientX),
      clientY: finiteNumber(event.clientY),
      pageX: finiteNumber(event.pageX),
      pageY: finiteNumber(event.pageY),
      screenX: finiteNumber(event.screenX),
      screenY: finiteNumber(event.screenY),
      offsetX: finiteNumber(event.offsetX),
      offsetY: finiteNumber(event.offsetY),
      movementX: finiteNumber(event.movementX),
      movementY: finiteNumber(event.movementY),
      button: event.button,
      buttons: event.buttons,
      pressure: finiteNumber(event.pressure),
      tangentialPressure: finiteNumber(event.tangentialPressure),
      width: finiteNumber(event.width),
      height: finiteNumber(event.height),
      tiltX: event.tiltX,
      tiltY: event.tiltY,
      twist: event.twist,
      altitudeAngle: finiteNumber(event.altitudeAngle),
      azimuthAngle: finiteNumber(event.azimuthAngle),
      ctrlKey: event.ctrlKey,
      shiftKey: event.shiftKey,
      altKey: event.altKey,
      metaKey: event.metaKey,
      timeStamp: finiteNumber(event.timeStamp),
      coalescedEventCount: coalescedEventCount(event),
      cancelable: event.cancelable,
      composed: event.composed,
      targetId: event.currentTarget instanceof HTMLElement ? event.currentTarget.id : ''
    }
  };
}

function finiteNumber(value: number | undefined) {
  return Number.isFinite(value) ? Number(value) : 0;
}

function coalescedEventCount(event: PointerEvent) {
  try {
    return typeof event.getCoalescedEvents === 'function' ? event.getCoalescedEvents().length : 0;
  } catch {
    return 0;
  }
}
