import {
  InputSource,
  isInputCaptureDebugEnabled,
  sendBrowserInputCapture,
  withInputCapture,
  type BrowserInputCapture,
  type InputCaptureReference
} from '@/lib/input-router.ts';

export type CapturedPointerEvent = {
  type: 'pointerdown' | 'pointermove' | 'pointerup' | 'pointercancel' | 'pointerleave';
  source: InputSource;
  captureSequence: number;
  capturedAt: number;
  pointerId: number;
  pointerType: string;
  isPrimary: boolean;
  clientX: number;
  clientY: number;
  movementX: number;
  movementY: number;
  offsetX: number;
  offsetY: number;
  button: number;
  buttons: number;
  pressure: number;
  target: HTMLElement;
};

export type CapturedWheelEvent = {
  type: 'wheel';
  source: InputSource;
  captureSequence: number;
  capturedAt: number;
  clientX: number;
  clientY: number;
  offsetX: number;
  offsetY: number;
  deltaX: number;
  deltaY: number;
  deltaZ: number;
  target: HTMLElement;
};

export type CapturedInputEvent = CapturedPointerEvent | CapturedWheelEvent;
export type PointerEventListener = (event: CapturedInputEvent) => void;

const listeners = new Set<PointerEventListener>();
let browserInputCaptureSequence = 0;

export function subscribePointerEvents(listener: PointerEventListener) {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

export function publishPointerEvent(event: CapturedInputEvent) {
  const publish = () => {
    for (const listener of listeners) listener(event);
  };
  if (event.captureSequence === 0) {
    publish();
    return;
  }
  withInputCapture({ sequence: event.captureSequence, capturedAt: event.capturedAt }, publish);
}

export function recordBrowserInputCapture(
  snapshot: (sequence: number, capturedAt: number) => BrowserInputCapture
): InputCaptureReference {
  const capturedAt = performance.now();
  if (!isInputCaptureDebugEnabled()) return { sequence: 0, capturedAt };
  browserInputCaptureSequence = nextCaptureSequence(browserInputCaptureSequence);
  sendBrowserInputCapture(snapshot(browserInputCaptureSequence, capturedAt));
  return { sequence: browserInputCaptureSequence, capturedAt };
}

function nextCaptureSequence(sequence: number) {
  return sequence >= 0xffffffff ? 1 : sequence + 1;
}
