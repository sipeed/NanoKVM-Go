import type { TouchscreenContact } from './input-router.ts';

export const TOUCHSCREEN_REPORT_INTERVAL_MS = 16;
export const TOUCHSCREEN_HEARTBEAT_INTERVAL_MS = 250;

export function touchscreenReportKey(contacts: TouchscreenContact[]) {
  return contacts
    .map((contact) =>
      [
        contact.id,
        contact.touching ? 1 : 0,
        contact.inRange === false ? 0 : 1,
        contact.x,
        contact.y,
        contact.pressure ?? 0
      ].join(':')
    )
    .join('|');
}

export function shouldSendTouchscreenReport(
  reportKey: string,
  lastReportKey: string,
  elapsedMS: number
) {
  return reportKey !== lastReportKey || elapsedMS >= TOUCHSCREEN_HEARTBEAT_INTERVAL_MS;
}
