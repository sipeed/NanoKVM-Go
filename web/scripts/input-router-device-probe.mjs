import fs from 'node:fs';
import http from 'node:http';
import https from 'node:https';
import { pathToFileURL } from 'node:url';
import websocket from 'websocket';

import { ensureDevicePointerMode } from './pointer-mode-probe.mjs';

const { w3cwebsocket: W3cWebSocket } = websocket;

const SOURCE = {
  keyboard: 1,
  pointer: 2,
  touchscreen: 3
};

const KIND = {
  keyboard: 1,
  relativePointer: 2,
  absolutePointer: 3,
  stylus: 4,
  touchscreen: 5
};

export function encodeFrame(source, kind, payload, sequence, capturedAt = Date.now()) {
  const frame = Buffer.alloc(13 + payload.length);
  frame[0] = 3;
  frame[1] = 1;
  frame[2] = source;
  frame[3] = kind;
  frame[4] = 0;
  frame.writeUInt32LE(sequence >>> 0, 5);
  frame.writeUInt32LE(capturedAt >>> 0, 9);
  Buffer.from(payload).copy(frame, 13);
  return frame;
}

export function statusDelta(before, after) {
  const beforeRoutes = new Map(before.routes.map((route) => [route.route, route]));
  return {
    received: after.received - before.received,
    dispatched: after.dispatched - before.dispatched,
    rejected: after.rejected - before.rejected,
    decodeRejected: after.decode_rejected - before.decode_rejected,
    dispatchFailed: after.dispatch_failed - before.dispatch_failed,
    backpressureDropped: (after.backpressure_dropped ?? 0) - (before.backpressure_dropped ?? 0),
    routes: after.routes.map((route) => {
      const previous = beforeRoutes.get(route.route) ?? {};
      return {
        route: route.route,
        dispatched: route.dispatched - (previous.dispatched ?? 0),
        failed: route.failed - (previous.failed ?? 0),
        writeRetries:
          (route.write_retries ?? route.release_retries ?? 0) -
          (previous.write_retries ?? previous.release_retries ?? 0),
        backpressureDropped:
          (route.backpressure_dropped ?? 0) - (previous.backpressure_dropped ?? 0)
      };
    })
  };
}

export function hid2ProbeFrame(mode) {
  switch (mode) {
    case 'absolute':
      return {
        source: SOURCE.pointer,
        kind: KIND.absolutePointer,
        payload: absolutePayload(),
        route: 'hidg2/absolute-pointer'
      };
    case 'digitizer-stylus':
      return {
        source: SOURCE.pointer,
        kind: KIND.stylus,
        payload: stylusPayload(),
        route: 'hidg2/stylus'
      };
    case 'touch-screen':
      return {
        source: SOURCE.touchscreen,
        kind: KIND.touchscreen,
        payload: Buffer.from([0]),
        route: 'hidg2/touchscreen'
      };
    default:
      throw new Error(`unsupported active pointer mode: ${mode}`);
  }
}

export function evaluateProbe({ before, middle, after, sent, probeSessionIDs }) {
  const delta = statusDelta(before, after);
  const probeIDs = new Set(probeSessionIDs);
  const beforeSessions = new Map(before.sessions.map((session) => [session.id, session]));
  const middleSessions = new Map(middle.sessions.map((session) => [session.id, session]));
  const afterSessions = new Map(after.sessions.map((session) => [session.id, session]));
  const probeSessions = middle.sessions.filter((session) => probeIDs.has(session.id));
  const remainingSessions = after.sessions.filter((session) => probeIDs.has(session.id));
  const remainingOwners = after.routes.filter((route) => probeIDs.has(route.owner_session));
  const failedRoutes = delta.routes.filter((route) => route.failed !== 0);
  const changedPreexistingSessions = [...beforeSessions.values()].filter((session) => {
    const during = middleSessions.get(session.id);
    const completed = afterSessions.get(session.id);
    return (
      !during ||
      !completed ||
      during.events !== session.events ||
      completed.events !== session.events
    );
  });
  const unexpectedSessionIDs = new Set(
    [...middle.sessions, ...after.sessions]
      .filter((session) => !beforeSessions.has(session.id) && !probeIDs.has(session.id))
      .map((session) => session.id)
  );
  const isolatedWindow = changedPreexistingSessions.length === 0 && unexpectedSessionIDs.size === 0;
  const checks = {
    exactFrameCount: delta.received === sent,
    exactDispatchCount: delta.dispatched === sent,
    noRejectedFrames: delta.rejected === 0,
    noDecodeRejects: delta.decodeRejected === 0,
    noDispatchFailures: delta.dispatchFailed === 0,
    noBackpressureDrops: delta.backpressureDropped === 0,
    noRouteFailures: failedRoutes.length === 0,
    probeSessionsIdentified: probeSessions.length === probeSessionIDs.length,
    probeSessionsReleased: remainingSessions.length === 0,
    probeOwnersReleased: remainingOwners.length === 0,
    isolatedWindow
  };
  const pass = Object.values(checks).every(Boolean);
  return {
    outcome: pass ? 'pass' : isolatedWindow ? 'fail' : 'inconclusive',
    pass,
    checks,
    delta,
    probeSessionIDs: [...probeIDs],
    diagnostics: {
      changedPreexistingSessionIDs: changedPreexistingSessions.map((session) => session.id),
      unexpectedSessionIDs: [...unexpectedSessionIDs],
      middleSessions: middle.sessions.map((session) => ({
        id: session.id,
        events: session.events,
        lastSourceSequence: session.last_source_sequence,
        routes: session.routes,
        sources: session.sources
      }))
    }
  };
}

export function parseOptions(argv, env = process.env) {
  const options = {
    url: env.NANOKVM_URL ?? '',
    token: env.NANOKVM_TOKEN ?? '',
    insecure: false,
    activeTouch: false,
    activeKeyboard: false,
    activePointer: false,
    hid2Frames: 100,
    intervalMS: 16,
    targetOS: 'unknown',
    targetVersion: '',
    notes: '',
    output: ''
  };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--insecure') {
      options.insecure = true;
      continue;
    }
    if (argument === '--active-touch') {
      options.activeTouch = true;
      continue;
    }
    if (argument === '--active-keyboard') {
      options.activeKeyboard = true;
      continue;
    }
    if (argument === '--active-pointer') {
      options.activePointer = true;
      continue;
    }
    const value = argv[++index];
    if (value === undefined) throw new Error(`missing value for ${argument}`);
    switch (argument) {
      case '--url':
        options.url = value;
        break;
      case '--token':
        options.token = value;
        break;
      case '--hid2-frames':
      case '--touch-frames': // Backward-compatible alias.
        options.hid2Frames = parseInteger(value, argument, 2, 10000);
        break;
      case '--interval-ms':
        options.intervalMS = parseInteger(value, argument, 0, 1000);
        break;
      case '--target-os':
        options.targetOS = value;
        break;
      case '--target-version':
        options.targetVersion = value;
        break;
      case '--notes':
        options.notes = value;
        break;
      case '--output':
        options.output = value;
        break;
      default:
        throw new Error(`unknown option: ${argument}`);
    }
  }
  if (!options.url) throw new Error('NANOKVM_URL or --url is required');
  if (!options.token) throw new Error('NANOKVM_TOKEN or --token is required');
  options.url = new URL(options.url).toString().replace(/\/$/, '');
  return options;
}

async function runProbe(options) {
  const pointerMode = await ensureDevicePointerMode(options, 'relative');
  const hid2Frame = hid2ProbeFrame(pointerMode.after.mode);
  if (options.activeTouch && pointerMode.after.mode !== 'touch-screen') {
    throw new Error('--active-touch requires the device touch-screen pointer mode');
  }
  const before = await getRuntimeStatus(options);
  const sockets = [];
  const first = await connect(options);
  sockets.push(first);
  let second;
  try {
    second = await connect(options);
    sockets.push(second);
  } catch (error) {
    await Promise.all(sockets.map(close));
    throw error;
  }
  const firstSequence = createSequence(0x10000000);
  const secondSequence = createSequence(0x90000000);
  let sent = 0;
  let middle;
  let probeSessionIDs;

  try {
    send(first, SOURCE.keyboard, KIND.keyboard, Buffer.alloc(8), firstSequence);
    send(second, SOURCE.pointer, KIND.relativePointer, Buffer.alloc(7), secondSequence);
    sent += 2;
    const markerSequences = [firstSequence.value, secondSequence.value];
    const markerStatus = await waitForRuntimeStatus(options, (status) =>
      markerSequences.every((sequence) => sessionWithSequence(status, sequence))
    );
    probeSessionIDs = markerSequences.map(
      (sequence) => sessionWithSequence(markerStatus, sequence).id
    );

    send(first, SOURCE.pointer, KIND.relativePointer, Buffer.alloc(7), firstSequence);
    sent += 1;

    for (let index = 0; index < options.hid2Frames; index += 1) {
      const target = index % 2 === 0 ? first : second;
      const sequence = index % 2 === 0 ? firstSequence : secondSequence;
      send(target, hid2Frame.source, hid2Frame.kind, hid2Frame.payload, sequence);
      sent += 1;
      if (options.intervalMS > 0) await delay(options.intervalMS);
    }

    if (options.activeTouch) {
      send(first, SOURCE.touchscreen, KIND.touchscreen, activeTouchPayload(0x03), firstSequence);
      sent += 1;
      await delay(100);
      send(first, SOURCE.touchscreen, KIND.touchscreen, activeTouchPayload(0x02), firstSequence);
      sent += 1;
      await delay(16);
      send(first, SOURCE.touchscreen, KIND.touchscreen, Buffer.from([0]), firstSequence);
      sent += 1;
    }

    if (options.activeKeyboard) {
      send(first, SOURCE.keyboard, KIND.keyboard, keyboardModifierPayload(0x02), firstSequence);
      sent += 1;
      await delay(100);
      send(first, SOURCE.keyboard, KIND.keyboard, Buffer.alloc(8), firstSequence);
      sent += 1;
    }

    if (options.activePointer) {
      send(first, SOURCE.pointer, KIND.relativePointer, relativePointerPayload(1), firstSequence);
      sent += 1;
      await delay(50);
      send(first, SOURCE.pointer, KIND.relativePointer, relativePointerPayload(-1), firstSequence);
      sent += 1;
    }

    send(first, SOURCE.keyboard, KIND.keyboard, Buffer.alloc(8), firstSequence);
    send(second, SOURCE.pointer, KIND.relativePointer, Buffer.alloc(7), secondSequence);
    sent += 2;
    const terminalSequences = new Map([
      [probeSessionIDs[0], firstSequence.value],
      [probeSessionIDs[1], secondSequence.value]
    ]);
    middle = await waitForRuntimeStatus(options, (status) =>
      [...terminalSequences].every(([sessionID, sequence]) => {
        const session = status.sessions.find((candidate) => candidate.id === sessionID);
        return session?.last_source_sequence === sequence;
      })
    );
  } finally {
    await Promise.all(sockets.map(close));
  }

  const probeIDs = new Set(probeSessionIDs);
  const after = await waitForRuntimeStatus(
    options,
    (status) =>
      status.sessions.every((session) => !probeIDs.has(session.id)) &&
      status.routes.every((route) => !probeIDs.has(route.owner_session))
  );
  const result = evaluateProbe({ before, middle, after, sent, probeSessionIDs });
  const evidence = {
    schema: 'nanokvm-input-router-probe/v2',
    timestamp: new Date().toISOString(),
    target: {
      os: options.targetOS,
      version: options.targetVersion,
      notes: options.notes
    },
    endpoint: options.url,
    pointerMode,
    sent: {
      frames: sent,
      hid2Frames: options.hid2Frames,
      hid2Route: hid2Frame.route,
      sessions: 2,
      neutralOnly: !options.activeTouch && !options.activeKeyboard && !options.activePointer,
      activeTouch: options.activeTouch,
      activeKeyboard: options.activeKeyboard,
      activePointer: options.activePointer
    },
    result
  };
  const output = `${JSON.stringify(evidence, null, 2)}\n`;
  if (options.output) fs.writeFileSync(options.output, output, { mode: 0o600 });
  process.stdout.write(output);
  if (!result.pass) process.exitCode = 1;
}

function send(socket, source, kind, payload, sequence) {
  sequence.value = (sequence.value + 1) >>> 0;
  socket.send(encodeFrame(source, kind, payload, sequence.value));
}

function createSequence(prefix) {
  const random = Math.floor(Math.random() * 0x0fffff);
  return { value: (prefix | random) >>> 0 };
}

function absolutePayload() {
  return Buffer.from([0, 0x00, 0x80, 0x00, 0x80, 0, 0]);
}

function stylusPayload() {
  return Buffer.from([0, 0x00, 0x80, 0x00, 0x80, 0]);
}

export function keyboardModifierPayload(modifiers) {
  const payload = Buffer.alloc(8);
  payload[0] = modifiers;
  return payload;
}

export function relativePointerPayload(deltaX) {
  const payload = Buffer.alloc(7);
  payload.writeInt16LE(deltaX, 1);
  return payload;
}

export function activeTouchPayload(state) {
  return Buffer.concat([
    Buffer.from([2]),
    touchContactPayload(1, state, 0x5000, 0x7000),
    touchContactPayload(2, state, 0xb000, 0x7000)
  ]);
}

function touchContactPayload(id, state, x, y) {
  const contact = Buffer.alloc(8);
  contact[0] = id;
  contact[1] = state;
  contact.writeUInt16LE(x, 2);
  contact.writeUInt16LE(y, 4);
  contact[6] = state & 1 ? 0xff : 0;
  return contact;
}

function connect(options) {
  const url = new URL('/api/ws', options.url);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return new Promise((resolve, reject) => {
    const socket = new W3cWebSocket(
      url.toString(),
      undefined,
      undefined,
      { Authorization: `Bearer ${options.token}` },
      { rejectUnauthorized: !options.insecure }
    );
    const timeout = setTimeout(() => reject(new Error('WebSocket connection timed out')), 5000);
    socket.onopen = () => {
      clearTimeout(timeout);
      resolve(socket);
    };
    socket.onerror = () => {
      clearTimeout(timeout);
      reject(new Error('WebSocket connection failed'));
    };
  });
}

function close(socket) {
  return new Promise((resolve) => {
    const timeout = setTimeout(resolve, 1000);
    socket.onclose = () => {
      clearTimeout(timeout);
      resolve();
    };
    socket.close();
  });
}

function getRuntimeStatus(options) {
  const url = new URL('/api/hid/input-router/status', options.url);
  const transport = url.protocol === 'https:' ? https : http;
  return new Promise((resolve, reject) => {
    const request = transport.request(
      url,
      {
        method: 'GET',
        headers: { Authorization: `Bearer ${options.token}` },
        rejectUnauthorized: !options.insecure,
        timeout: 5000
      },
      (response) => {
        const chunks = [];
        response.on('data', (chunk) => chunks.push(chunk));
        response.on('end', () => {
          try {
            const payload = JSON.parse(Buffer.concat(chunks).toString('utf8'));
            if (response.statusCode !== 200 || payload.code !== 0) {
              reject(new Error(`management request failed: HTTP ${response.statusCode}`));
              return;
            }
            resolve(payload.data.runtime);
          } catch (error) {
            reject(error);
          }
        });
      }
    );
    request.on('error', reject);
    request.on('timeout', () => request.destroy(new Error('management request timed out')));
    request.end();
  });
}

async function waitForRuntimeStatus(options, predicate, timeoutMS = 5000) {
  const deadline = Date.now() + timeoutMS;
  let status;
  do {
    status = await getRuntimeStatus(options);
    if (predicate(status)) return status;
    await delay(25);
  } while (Date.now() < deadline);
  throw new Error('input router management state did not settle before timeout');
}

function sessionWithSequence(status, sequence) {
  return status.sessions.find((session) => session.last_source_sequence === sequence);
}

function parseInteger(value, option, minimum, maximum) {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isInteger(parsed) || parsed < minimum || parsed > maximum) {
    throw new Error(`${option} must be between ${minimum} and ${maximum}`);
  }
  return parsed;
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  runProbe(parseOptions(process.argv.slice(2))).catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}
