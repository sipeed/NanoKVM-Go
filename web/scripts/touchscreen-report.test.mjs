import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

import {
  shouldSendTouchscreenReport,
  TOUCHSCREEN_HEARTBEAT_INTERVAL_MS,
  touchscreenReportKey
} from '../src/lib/touchscreen-report.ts';

const touchScreenSource = fs.readFileSync(
  path.resolve(import.meta.dirname, '../src/pages/desktop/mouse/touch-screen.tsx'),
  'utf8'
);

const contact = {
  id: 1,
  touching: true,
  inRange: true,
  x: 0.25,
  y: 0.75,
  pressure: 128
};

test('touchscreen reports send changes immediately and stable state on heartbeat', () => {
  const key = touchscreenReportKey([contact]);

  assert.equal(shouldSendTouchscreenReport(key, '', 0), true);
  assert.equal(shouldSendTouchscreenReport(key, key, TOUCHSCREEN_HEARTBEAT_INTERVAL_MS - 1), false);
  assert.equal(shouldSendTouchscreenReport(key, key, TOUCHSCREEN_HEARTBEAT_INTERVAL_MS), true);
});

test('touchscreen report key includes contact state and position', () => {
  const key = touchscreenReportKey([contact]);

  assert.notEqual(touchscreenReportKey([{ ...contact, touching: false }]), key);
  assert.notEqual(touchscreenReportKey([{ ...contact, x: 0.5 }]), key);
  assert.equal(touchscreenReportKey([]), '');
});

test('touchscreen capture snapshots every active TouchEvent contact on browser frames', () => {
  assert.match(touchScreenSource, /event\.targetTouches/);
  assert.match(touchScreenSource, /event\.changedTouches/);
  assert.match(touchScreenSource, /addEventListener\('touchmove'/);
  assert.match(touchScreenSource, /api: 'TouchEvent'/);
  assert.match(touchScreenSource, /subscribePointerEvents/);
  assert.match(touchScreenSource, /requestAnimationFrame\(reportFrameCallback\)/);
  assert.match(touchScreenSource, /function reportFrameCallback\(\)/);
  assert.doesNotMatch(
    touchScreenSource,
    /setInterval\(sendActiveContacts,\s*TOUCHSCREEN_REPORT_INTERVAL_MS\)/
  );
});
