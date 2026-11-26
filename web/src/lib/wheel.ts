/**
 * Turns browser wheel deltas into HID wheel notches.
 *
 * A HID wheel report carries whole notches, and the host scrolls a fixed distance
 * (plus its own acceleration) per notch. Browsers instead deliver many small pixel
 * deltas for touchpads and smooth scrolling, so forwarding one notch per event
 * makes touchpads scroll many times faster than the gesture. This accumulator adds
 * up the deltas and emits a notch only once a notch's worth of distance has built up.
 *
 * Only the vertical axis is reported (the gadget descriptors declare no AC Pan), so
 * events that are mostly horizontal are dropped rather than leaking their small
 * vertical component into vertical scrolling.
 */

const DOM_DELTA_PIXEL = 0;
const DOM_DELTA_LINE = 1;
const DOM_DELTA_PAGE = 2;

/** Pixels of browser scroll that make one HID wheel notch. */
export const WHEEL_PIXELS_PER_NOTCH = 25;
export const WHEEL_PIXELS_PER_NOTCH_MIN = 10;
export const WHEEL_PIXELS_PER_NOTCH_MAX = 120;
const LINE_PIXELS = 16;
const PAGE_PIXELS = 800;
/** A pause longer than this starts a new gesture with an empty accumulator. */
const IDLE_RESET_MS = 250;

// Acceleration: the distance each event contributes is scaled by the swipe speed,
// so slow swipes scroll precisely and fast swipes cover more ground. A swipe at
// REFERENCE_SPEED (a normal two-finger scroll, ~8 px every 7 ms in Firefox) is unscaled.
const REFERENCE_SPEED = 1.2; // px per ms
const ACCELERATION_EXPONENT = 0.7;
const MIN_GAIN = 0.5;
/** The first notch of a gesture needs only this fraction of a notch, unscaled, so scrolling starts promptly. */
const FIRST_NOTCH_FRACTION = 1 / 3;
/** Notches at the start of a gesture that are never scaled down by acceleration. */
const RAMP_NOTCHES = 3;
const MAX_GAIN = 4;
const SPEED_SMOOTHING = 0.3; // weight of the newest event in the running speed
const MIN_EVENT_GAP_MS = 4;
/** Cap per event, so a fast swipe can send several notches in one report. */
const MAX_NOTCHES_PER_EVENT = 3;

export type WheelAccumulatorOptions = {
  /** Scale deltas by swipe speed (default true). */
  acceleration?: boolean;
};

/** Running totals for the #wheel-debug readout. */
export const wheelStats = {
  events: 0,
  notches: 0,
  /** Notches that passed the Wheel speed throttle and were sent. */
  sent: 0,
  pixelsPerNotch: 0,
  gain: 1,
  /** Wheel direction setting in effect (1 or -1) at the last sent notch. */
  direction: 0
};

export type WheelInput = {
  deltaX: number;
  deltaY: number;
  deltaMode?: number;
};

function toPixels(delta: number, deltaMode: number | undefined): number {
  switch (deltaMode) {
    case DOM_DELTA_LINE:
      return delta * LINE_PIXELS;
    case DOM_DELTA_PAGE:
      return delta * PAGE_PIXELS;
    case DOM_DELTA_PIXEL:
    default:
      return delta;
  }
}

export function clampPixelsPerNotch(value: number): number {
  if (!Number.isFinite(value)) return WHEEL_PIXELS_PER_NOTCH;
  return Math.min(WHEEL_PIXELS_PER_NOTCH_MAX, Math.max(WHEEL_PIXELS_PER_NOTCH_MIN, value));
}

export class WheelAccumulator {
  private accumulated = 0;
  private lastAt = -Infinity;
  private speed = 0;
  private sentThisGesture = false;
  private notchesThisGesture = 0;
  private pixelsPerNotch: number;
  private readonly acceleration: boolean;

  constructor(pixelsPerNotch = WHEEL_PIXELS_PER_NOTCH, options: WheelAccumulatorOptions = {}) {
    this.pixelsPerNotch = clampPixelsPerNotch(pixelsPerNotch);
    this.acceleration = options.acceleration ?? true;
  }

  /** Changes the distance per notch (the touchpad scroll setting). */
  setPixelsPerNotch(pixelsPerNotch: number): void {
    this.pixelsPerNotch = clampPixelsPerNotch(pixelsPerNotch);
  }

  /**
   * Adds one wheel event and returns the notches to send now (0 to ±3), with the
   * browser's sign (positive = content scrolls down). The first notch of a gesture
   * comes after half a notch of travel and is always a single notch, so scrolling
   * starts promptly and a single click of a wheel mouse stays one notch.
   */
  push(event: WheelInput, now: number): number {
    const x = toPixels(event.deltaX, event.deltaMode);
    const y = toPixels(event.deltaY, event.deltaMode);
    wheelStats.events++;
    wheelStats.pixelsPerNotch = this.pixelsPerNotch;

    const gap = now - this.lastAt;
    const newGesture = gap > IDLE_RESET_MS;
    if (newGesture) {
      this.accumulated = 0;
      this.sentThisGesture = false;
      this.notchesThisGesture = 0;
    }
    this.lastAt = now;

    // Mostly horizontal: nothing to report on a vertical-only wheel.
    if (Math.abs(x) > Math.abs(y) || y === 0) return 0;

    // Reversing direction discards what was built up the other way.
    if (this.accumulated !== 0 && Math.sign(this.accumulated) !== Math.sign(y)) {
      this.accumulated = 0;
    }
    const gain = this.gain(Math.abs(y), newGesture ? Infinity : gap);
    // At the start of a gesture, never scale down: the start of a swipe is slow, and
    // acceleration would otherwise make the first notches need a long finger travel.
    const ramping = this.notchesThisGesture < RAMP_NOTCHES;
    this.accumulated += y * (ramping ? Math.max(1, gain) : gain);

    if (!this.sentThisGesture) {
      if (Math.abs(this.accumulated) < this.pixelsPerNotch * FIRST_NOTCH_FRACTION) return 0;
      const first = Math.sign(this.accumulated);
      this.accumulated = 0;
      this.sentThisGesture = true;
      this.notchesThisGesture = 1;
      wheelStats.notches++;
      return first;
    }

    if (Math.abs(this.accumulated) < this.pixelsPerNotch) return 0;
    const direction = Math.sign(this.accumulated);
    const count = Math.min(
      MAX_NOTCHES_PER_EVENT,
      Math.trunc(Math.abs(this.accumulated) / this.pixelsPerNotch)
    );
    this.accumulated -= direction * count * this.pixelsPerNotch;
    // Keep at most one notch of backlog so a fast flick does not keep scrolling
    // after the fingers stop.
    if (Math.abs(this.accumulated) > this.pixelsPerNotch) {
      this.accumulated = direction * this.pixelsPerNotch;
    }
    wheelStats.notches += count;
    this.notchesThisGesture += count;
    return direction * count;
  }

  reset(): void {
    this.accumulated = 0;
    this.lastAt = -Infinity;
    this.speed = 0;
    this.sentThisGesture = false;
    this.notchesThisGesture = 0;
  }

  private gain(distance: number, gap: number): number {
    if (!this.acceleration) return 1;
    // The first event of a gesture has no gap to measure; assume a typical frame.
    if (!Number.isFinite(gap)) {
      // First event of a gesture: no gap to measure a speed from, so leave it unscaled.
      this.speed = REFERENCE_SPEED;
      return 1;
    }
    const instant = distance / Math.max(MIN_EVENT_GAP_MS, gap);
    this.speed = this.speed * (1 - SPEED_SMOOTHING) + instant * SPEED_SMOOTHING;
    const gain = Math.min(
      MAX_GAIN,
      Math.max(MIN_GAIN, (this.speed / REFERENCE_SPEED) ** ACCELERATION_EXPONENT)
    );
    wheelStats.gain = gain;
    return gain;
  }
}
