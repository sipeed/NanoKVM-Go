import assert from 'node:assert/strict';
import test from 'node:test';

import { sendWebcam } from '../src/pages/desktop/menu/webcam-sender.ts';

function share(t) {
  const reports = [];
  let socket;
  let accept;
  let closed = false;
  class Socket {
    static OPEN = 1;
    readyState = 1;
    bufferedAmount = 0;
    constructor() {
      socket = this;
    }
    send() {}
    close() {}
  }
  const previousSocket = globalThis.WebSocket;
  globalThis.WebSocket = Socket;
  t.after(() => {
    globalThis.WebSocket = previousSocket;
  });
  const source = {
    listen: async (onFrame) => {
      accept = onFrame;
    },
    encode: async () => new Blob([new Uint8Array(100)]),
    close: () => {
      closed = true;
    }
  };
  const sender = sendWebcam('ws://localhost/camera', source, (report) => reports.push(report));
  t.after(sender.stop);
  return {
    grant: (data) => socket.onmessage({ data }),
    frame: () => accept({ close() {} }, performance.now()),
    ended: () => closed,
    reports
  };
}

test('ending a calibration with no delivered frames closes capture and reports an error', (t) => {
  const session = share(t);
  session.grant('calibrate');
  session.grant('idle');
  assert.ok(session.ended());
  assert.match(session.reports.at(-1).error, /No camera frames/);
  assert.ok(!session.reports.some((report) => report.status === 'waiting'));
});

test('the first JPEG acknowledged by the final idle grant is sufficient for Ready', async (t) => {
  const session = share(t);
  session.grant('calibrate');
  session.frame();
  await Promise.resolve();
  session.grant('idle');
  assert.equal(session.ended(), false);
  assert.equal(session.reports.at(-1).status, 'waiting');
});

test('an already-open USB camera with a stalled source exits startup with a useful error', (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  let now = 0;
  t.mock.method(performance, 'now', () => now);
  const session = share(t);
  session.grant('720p30');
  for (let second = 1; second <= 11; second++) {
    now = second * 1000;
    session.grant('720p30');
    t.mock.timers.tick(1000);
  }
  assert.ok(session.ended());
  assert.match(session.reports.at(-1).error, /No camera frames/);
});
