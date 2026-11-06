// Button bit positions
const MouseButtons = {
  Left: 1 << 0,
  Right: 1 << 1,
  Middle: 1 << 2,
  Back: 1 << 3,
  Forward: 1 << 4
} as const;

// Map browser button index to HID bit
function getMouseButtonBit(button: number): number {
  switch (button) {
    case 0:
      return MouseButtons.Left;
    case 1:
      return MouseButtons.Middle;
    case 2:
      return MouseButtons.Right;
    case 3:
      return MouseButtons.Back;
    case 4:
      return MouseButtons.Forward;
    default:
      return 0;
  }
}

/**
 * Relative Mouse Report (4 bytes)
 * Used with /dev/hidg1 (relative mouse)
 *
 * Byte 0: Buttons
 * Byte 1: X movement (-127 to 127)
 * Byte 2: Y movement (-127 to 127)
 * Byte 3: Wheel (-127 to 127)
 */
export class MouseReportRelative {
  private buttons: number = 0;

  buttonDown(button: number): void {
    this.buttons |= getMouseButtonBit(button);
  }

  buttonUp(button: number): void {
    this.buttons &= ~getMouseButtonBit(button);
  }

  /**
   * Build relative mouse report
   * @param deltaX X movement (-127 to 127)
   * @param deltaY Y movement (-127 to 127)
   * @param wheel Scroll wheel (-127 to 127, negative = down)
   */
  buildReport(deltaX: number, deltaY: number, wheel: number = 0): Uint8Array {
    const report = new Uint8Array(4);
    report[0] = this.buttons;
    report[1] = this.clamp(Math.round(deltaX), -127, 127) & 0xff;
    report[2] = this.clamp(Math.round(deltaY), -127, 127) & 0xff;
    report[3] = this.clamp(Math.round(wheel), -127, 127) & 0xff;
    return report;
  }

  /**
   * Build button-only report (no movement)
   */
  buildButtonReport(): Uint8Array {
    return this.buildReport(0, 0, 0);
  }

  reset(): Uint8Array {
    this.buttons = 0;
    return this.buildReport(0, 0, 0);
  }

  private clamp(value: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, value));
  }
}

// /dev/hidg2 exposes two top-level collections; the first report byte selects one.
export const AbsoluteReportID = {
  Mouse: 0x01,
  Stylus: 0x02
} as const;

/**
 * Absolute Mouse Report (7 bytes) - report ID 1 on /dev/hidg2.
 * Generic Desktop absolute mouse: works on Windows/Linux/macOS/iOS,
 * ignored by Android.
 *
 * Byte 0: Report ID (0x01)
 * Byte 1: Buttons
 * Byte 2-3: X position (0 to 32767, Little Endian)
 * Byte 4-5: Y position (0 to 32767, Little Endian)
 * Byte 6: Wheel (-127 to 127)
 */
export class MouseReportAbsolute {
  private buttons: number = 0;

  buttonDown(button: number): void {
    this.buttons |= getMouseButtonBit(button);
  }

  buttonUp(button: number): void {
    this.buttons &= ~getMouseButtonBit(button);
  }

  buildReport(x: number, y: number, wheel: number = 0): Uint8Array {
    const report = new Uint8Array(7);

    report[0] = AbsoluteReportID.Mouse;
    report[1] = this.buttons;
    report[2] = x & 0xff;
    report[3] = (x >> 8) & 0xff;
    report[4] = y & 0xff;
    report[5] = (y >> 8) & 0xff;
    report[6] = this.clamp(Math.round(wheel), -127, 127) & 0xff;

    return report;
  }

  /**
   * Build button-only report (keeps last position)
   */
  buildButtonReport(lastX: number, lastY: number): Uint8Array {
    return this.buildReport(lastX, lastY);
  }

  releaseButtonsAt(lastX: number, lastY: number): Uint8Array {
    this.buttons = 0;
    return this.buildReport(lastX, lastY);
  }

  leaveRangeAt(lastX: number, lastY: number): Uint8Array {
    this.buttons = 0;
    return this.buildReport(lastX, lastY);
  }

  reset(): Uint8Array {
    return this.leaveRangeAt(0, 0);
  }

  private clamp(value: number, min: number, max: number): number {
    return Math.max(min, Math.min(max, value));
  }
}

/**
 * Absolute Stylus Report (7 bytes) - report ID 2 on /dev/hidg2.
 * Digitizer pen: works on Android (move/click/drag, no wheel),
 * ignored where the absolute mouse collection works.
 *
 * Byte 0: Report ID (0x02)
 * Byte 1: State bits: bit0 tip switch, bit1 in range, bit2 barrel switch
 * Byte 2-3: X position (0 to 32767, Little Endian)
 * Byte 4-5: Y position (0 to 32767, Little Endian)
 * Byte 6: Pressure (0 to 255)
 */
export class MouseReportStylus {
  private buttons: number = 0;

  buttonDown(button: number): void {
    this.buttons |= getMouseButtonBit(button);
  }

  buttonUp(button: number): void {
    this.buttons &= ~getMouseButtonBit(button);
  }

  /**
   * Build stylus report from mouse-style button state.
   * Left/middle/back/forward press the stylus tip. Right maps to barrel + tip.
   * The wheel argument is accepted for interface parity and ignored: the
   * stylus collection has no wheel usage.
   */
  buildReport(x: number, y: number, _wheel: number = 0): Uint8Array {
    void _wheel;

    const barrel = (this.buttons & MouseButtons.Right) !== 0;
    const touching = this.buttons !== 0;
    const state = 0x02 | (touching ? 0x01 : 0) | (barrel ? 0x04 : 0);

    const report = new Uint8Array(7);
    report[0] = AbsoluteReportID.Stylus;
    report[1] = state;
    report[2] = x & 0xff;
    report[3] = (x >> 8) & 0xff;
    report[4] = y & 0xff;
    report[5] = (y >> 8) & 0xff;
    report[6] = touching ? 0xff : 0;

    return report;
  }

  /**
   * Build button-only report (keeps last position)
   */
  buildButtonReport(lastX: number, lastY: number): Uint8Array {
    return this.buildReport(lastX, lastY);
  }

  releaseButtonsAt(lastX: number, lastY: number): Uint8Array {
    this.buttons = 0;
    return this.buildReport(lastX, lastY);
  }

  leaveRangeAt(lastX: number, lastY: number): Uint8Array {
    this.buttons = 0;

    const report = new Uint8Array(7);
    report[0] = AbsoluteReportID.Stylus;
    report[1] = 0;
    report[2] = lastX & 0xff;
    report[3] = (lastX >> 8) & 0xff;
    report[4] = lastY & 0xff;
    report[5] = (lastY >> 8) & 0xff;
    report[6] = 0;
    return report;
  }

  reset(): Uint8Array {
    return this.leaveRangeAt(0, 0);
  }
}
