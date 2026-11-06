import { client, MessageEvent } from '@/lib/websocket.ts';

export const INPUT_ROUTER_PROTOCOL_VERSION = 1;
export const INPUT_ROUTER_HEADER_SIZE = 12;
export const INPUT_ROUTER_MAX_CONTACTS = 10;
export const INPUT_ROUTER_CAPTURE_CONTEXT_FLAG = 1 << 0;
export const INPUT_ROUTER_CAPTURE_CONTEXT_SIZE = 8;
export const INPUT_ROUTER_RELEASE_RELATIVE_POINTER = 1 << 0;
export const INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER = 1 << 1;
export const INPUT_ROUTER_RELEASE_STYLUS = 1 << 2;
export const INPUT_ROUTER_RELEASE_TOUCHSCREEN = 1 << 3;

export type InputCaptureReference = {
  sequence: number;
  capturedAt: number;
};

export type BrowserPointerCapture = {
  api: 'PointerEvent';
  sequence: number;
  capturedAtMS: number;
  type: string;
  content: {
    pointerId: number;
    pointerType: string;
    isPrimary: boolean;
    clientX: number;
    clientY: number;
    pageX: number;
    pageY: number;
    screenX: number;
    screenY: number;
    offsetX: number;
    offsetY: number;
    movementX: number;
    movementY: number;
    button: number;
    buttons: number;
    pressure: number;
    tangentialPressure: number;
    width: number;
    height: number;
    tiltX: number;
    tiltY: number;
    twist: number;
    altitudeAngle: number;
    azimuthAngle: number;
    ctrlKey: boolean;
    shiftKey: boolean;
    altKey: boolean;
    metaKey: boolean;
    timeStamp: number;
    coalescedEventCount: number;
    cancelable: boolean;
    composed: boolean;
    targetId: string;
  };
};

export type BrowserTouchPoint = {
  identifier: number;
  clientX: number;
  clientY: number;
  pageX: number;
  pageY: number;
  screenX: number;
  screenY: number;
  radiusX: number;
  radiusY: number;
  rotationAngle: number;
  force: number;
};

export type BrowserTouchCapture = {
  api: 'TouchEvent';
  sequence: number;
  capturedAtMS: number;
  type: string;
  content: {
    touches: BrowserTouchPoint[];
    targetTouches: BrowserTouchPoint[];
    changedTouches: BrowserTouchPoint[];
    ctrlKey: boolean;
    shiftKey: boolean;
    altKey: boolean;
    metaKey: boolean;
    timeStamp: number;
    cancelable: boolean;
    composed: boolean;
    targetId: string;
  };
};

export type BrowserInputCapture = BrowserPointerCapture | BrowserTouchCapture;

let inputRouterEnabled = true;
let inputCaptureDebugEnabled = false;
let activeCaptureReference: InputCaptureReference | null = null;

if (typeof client.on === 'function') {
  client.on('input-debug-capture', (message) => {
    try {
      const payload = JSON.parse(String(message.data)) as { data?: string };
      setInputCaptureDebugEnabled(payload.data === 'true');
    } catch {
      setInputCaptureDebugEnabled(false);
    }
  });
}

if (typeof client.onConnectionState === 'function') {
  client.onConnectionState((state) => {
    if (state !== 'open') setInputCaptureDebugEnabled(false);
  });
}

export function setInputRouterEnabled(enabled: boolean) {
  inputRouterEnabled = enabled;
}

export function isInputRouterEnabled() {
  return inputRouterEnabled;
}

export function setInputCaptureDebugEnabled(enabled: boolean) {
  inputCaptureDebugEnabled = enabled;
}

export function isInputCaptureDebugEnabled() {
  return inputCaptureDebugEnabled;
}

export function withInputCapture<T>(reference: InputCaptureReference, callback: () => T): T {
  const previous = activeCaptureReference;
  activeCaptureReference = reference;
  try {
    return callback();
  } finally {
    activeCaptureReference = previous;
  }
}

export function sendBrowserInputCapture(capture: BrowserInputCapture) {
  if (!inputCaptureDebugEnabled) return false;
  const payload = new TextEncoder().encode(JSON.stringify(capture));
  const frame = new Uint8Array(1 + payload.length);
  frame[0] = MessageEvent.InputCaptureDebug;
  frame.set(payload, 1);
  return client.send(frame);
}

export function sendBrowserPointerCapture(capture: BrowserPointerCapture) {
  return sendBrowserInputCapture(capture);
}

export enum InputSource {
  BrowserKeyboard = 1,
  BrowserPointer = 2,
  BrowserTouchscreen = 3,
  VirtualKeyboard = 16,
  VirtualPointer = 17,
  WebHID = 32
}

export enum InputKind {
  KeyboardState = 1,
  RelativePointer = 2,
  AbsolutePointer = 3,
  Stylus = 4,
  Touchscreen = 5,
  PointerRelease = 6,
  RawHID = 127
}

export type TouchscreenContact = {
  id: number;
  touching: boolean;
  inRange?: boolean;
  x: number;
  y: number;
  pressure?: number;
};

export class PointerButtonState {
  private buttons = 0;

  down(button: number) {
    this.buttons |= browserButtonBit(button);
  }

  up(button: number) {
    this.buttons &= ~browserButtonBit(button);
  }

  reset() {
    this.buttons = 0;
  }

  get value() {
    return this.buttons;
  }
}

export class InputRouterClient {
  private sequence = 0;

  sendKeyboard(
    source: InputSource,
    report: Uint8Array,
    capturedAt?: number,
    capture?: InputCaptureReference
  ) {
    if (report.length !== 8) throw new Error(`invalid keyboard state length: ${report.length}`);
    return this.send(source, InputKind.KeyboardState, report, capturedAt, capture);
  }

  sendRelativePointer(
    source: InputSource,
    buttons: number,
    deltaX = 0,
    deltaY = 0,
    wheel = 0,
    capturedAt?: number,
    capture?: InputCaptureReference
  ) {
    const payload = new Uint8Array(7);
    const view = new DataView(payload.buffer);
    payload[0] = buttons & 0x1f;
    view.setInt16(1, clampSigned16(deltaX), true);
    view.setInt16(3, clampSigned16(deltaY), true);
    view.setInt16(5, clampSigned16(wheel), true);
    return this.send(source, InputKind.RelativePointer, payload, capturedAt, capture);
  }

  sendAbsolutePointer(
    source: InputSource,
    buttons: number,
    x: number,
    y: number,
    wheel = 0,
    capturedAt?: number,
    capture?: InputCaptureReference
  ) {
    const payload = new Uint8Array(7);
    const view = new DataView(payload.buffer);
    payload[0] = buttons & 0x1f;
    view.setUint16(1, normalizedAxis(x), true);
    view.setUint16(3, normalizedAxis(y), true);
    view.setInt16(5, clampSigned16(wheel), true);
    return this.send(source, InputKind.AbsolutePointer, payload, capturedAt, capture);
  }

  sendStylus(
    source: InputSource,
    state: number,
    x: number,
    y: number,
    pressure = 0,
    capturedAt?: number,
    capture?: InputCaptureReference
  ) {
    const payload = new Uint8Array(6);
    const view = new DataView(payload.buffer);
    payload[0] = state & 0x03;
    view.setUint16(1, normalizedAxis(x), true);
    view.setUint16(3, normalizedAxis(y), true);
    payload[5] = clampByte(pressure);
    return this.send(source, InputKind.Stylus, payload, capturedAt, capture);
  }

  sendTouchscreen(
    source: InputSource,
    contacts: TouchscreenContact[],
    capturedAt?: number,
    capture?: InputCaptureReference
  ) {
    if (contacts.length > INPUT_ROUTER_MAX_CONTACTS) {
      throw new Error(`too many touchscreen contacts: ${contacts.length}`);
    }
    const payload = new Uint8Array(1 + contacts.length * 8);
    const view = new DataView(payload.buffer);
    payload[0] = contacts.length;
    contacts.forEach((contact, index) => {
      const offset = 1 + index * 8;
      payload[offset] = contact.id & 0xff;
      payload[offset + 1] = (contact.touching ? 0x01 : 0) | (contact.inRange === false ? 0 : 0x02);
      view.setUint16(offset + 2, normalizedAxis(contact.x), true);
      view.setUint16(offset + 4, normalizedAxis(contact.y), true);
      payload[offset + 6] = clampByte(contact.pressure ?? (contact.touching ? 255 : 0));
    });
    return this.send(source, InputKind.Touchscreen, payload, capturedAt, capture);
  }

  sendPointerRelease(source: InputSource, routes: number) {
    const allRoutes =
      INPUT_ROUTER_RELEASE_RELATIVE_POINTER |
      INPUT_ROUTER_RELEASE_ABSOLUTE_POINTER |
      INPUT_ROUTER_RELEASE_STYLUS |
      INPUT_ROUTER_RELEASE_TOUCHSCREEN;
    if (routes === 0 || (routes & ~allRoutes) !== 0) {
      throw new Error(`invalid pointer release routes: ${routes}`);
    }
    return this.send(source, InputKind.PointerRelease, new Uint8Array([routes]));
  }

  private send(
    source: InputSource,
    kind: InputKind,
    payload: Uint8Array,
    capturedAt?: number,
    capture?: InputCaptureReference
  ) {
    if (!inputRouterEnabled) return false;
    const pendingCapture = capture ?? activeCaptureReference;
    const captureReference = inputCaptureDebugEnabled && pendingCapture?.sequence
      ? pendingCapture
      : null;
    const eventCapturedAt = capturedAt ?? captureReference?.capturedAt ?? performance.now();
    const captureContextSize = captureReference ? INPUT_ROUTER_CAPTURE_CONTEXT_SIZE : 0;
    this.sequence = (this.sequence + 1) >>> 0;
    const frame = new Uint8Array(
      1 + INPUT_ROUTER_HEADER_SIZE + payload.length + captureContextSize
    );
    const view = new DataView(frame.buffer);
    frame[0] = MessageEvent.InputRouter;
    frame[1] = INPUT_ROUTER_PROTOCOL_VERSION;
    frame[2] = source;
    frame[3] = kind;
    frame[4] = captureReference ? INPUT_ROUTER_CAPTURE_CONTEXT_FLAG : 0;
    view.setUint32(5, this.sequence, true);
    view.setUint32(9, Math.round(eventCapturedAt) >>> 0, true);
    frame.set(payload, 1 + INPUT_ROUTER_HEADER_SIZE);
    if (captureReference) {
      const contextOffset = 1 + INPUT_ROUTER_HEADER_SIZE + payload.length;
      view.setUint32(contextOffset, captureReference.sequence >>> 0, true);
      view.setUint32(
        contextOffset + 4,
        Math.max(0, Math.round((performance.now() - captureReference.capturedAt) * 1000)) >>> 0,
        true
      );
    }
    return client.send(frame);
  }
}

export const inputRouter = new InputRouterClient();

function browserButtonBit(button: number) {
  switch (button) {
    case 0:
      return 1 << 0;
    case 1:
      return 1 << 2;
    case 2:
      return 1 << 1;
    case 3:
      return 1 << 3;
    case 4:
      return 1 << 4;
    default:
      return 0;
  }
}

function normalizedAxis(value: number) {
  return Math.round(Math.max(0, Math.min(1, value)) * 0xffff);
}

function clampByte(value: number) {
  return Math.max(0, Math.min(255, Math.round(value)));
}

function clampSigned16(value: number) {
  return Math.max(-0x8000, Math.min(0x7fff, Math.round(value)));
}
