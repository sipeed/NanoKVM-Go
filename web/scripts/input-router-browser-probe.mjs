import { spawn } from 'node:child_process';
import fs from 'node:fs/promises';
import https from 'node:https';
import { pathToFileURL } from 'node:url';
import websocket from 'websocket';

import { ensureDevicePointerMode, observeDevicePointerMode } from './pointer-mode-probe.mjs';

const { w3cwebsocket: W3cWebSocket } = websocket;

const DIGITIZER_STYLUS_MODE = 'digitizer-stylus';
const LEGACY_DIGITIZER_STYLUS_MODE = 'absolute-android';
const BROWSER_PROBE_MODES = ['touch-screen', 'absolute', DIGITIZER_STYLUS_MODE, 'relative'];

function normalizeProbeMode(value) {
  return value === LEGACY_DIGITIZER_STYLUS_MODE ? DIGITIZER_STYLUS_MODE : value;
}

export function parseOptions(argv, env = process.env) {
  const options = {
    url: env.NANOKVM_URL ?? '',
    token: env.NANOKVM_TOKEN ?? '',
    browser: env.CHROMIUM ?? 'chromium',
    port: 9222,
    insecure: false,
    mode: 'touch-screen',
    switchFrom: '',
    input: 'touch',
    contacts: 2,
    output: ''
  };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === '--insecure') {
      options.insecure = true;
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
      case '--browser':
        options.browser = value;
        break;
      case '--port':
        options.port = parseInteger(value, argument, 1024, 65535);
        break;
      case '--mode':
        if (!BROWSER_PROBE_MODES.includes(normalizeProbeMode(value))) {
          throw new Error(`unsupported browser probe mode: ${value}`);
        }
        options.mode = normalizeProbeMode(value);
        break;
      case '--switch-from':
        if (!BROWSER_PROBE_MODES.includes(normalizeProbeMode(value))) {
          throw new Error(`unsupported initial browser probe mode: ${value}`);
        }
        options.switchFrom = normalizeProbeMode(value);
        break;
      case '--input':
        if (!['touch', 'keyboard', 'mouse'].includes(value)) {
          throw new Error(`unsupported browser probe input: ${value}`);
        }
        options.input = value;
        break;
      case '--contacts':
        options.contacts = parseInteger(value, argument, 1, 2);
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

export function statusDelta(before, after) {
  const beforeRoutes = new Map(before.routes.map((route) => [route.route, route]));
  const beforeSources = new Map(before.sources.map((source) => [source.source, source]));
  return {
    received: after.received - before.received,
    dispatched: after.dispatched - before.dispatched,
    rejected: after.rejected - before.rejected,
    decodeRejected: after.decode_rejected - before.decode_rejected,
    dispatchFailed: after.dispatch_failed - before.dispatch_failed,
    backpressureDropped: (after.backpressure_dropped ?? 0) - (before.backpressure_dropped ?? 0),
    sources: after.sources.map((source) => ({
      source: source.source,
      events: source.events - (beforeSources.get(source.source)?.events ?? 0)
    })),
    routes: after.routes.map((route) => ({
      route: route.route,
      dispatched: route.dispatched - (beforeRoutes.get(route.route)?.dispatched ?? 0),
      failed: route.failed - (beforeRoutes.get(route.route)?.failed ?? 0),
      backpressureDropped:
        (route.backpressure_dropped ?? 0) -
        (beforeRoutes.get(route.route)?.backpressure_dropped ?? 0)
    }))
  };
}

class CdpClient {
  constructor(url) {
    this.socket = new W3cWebSocket(url);
    this.nextID = 1;
    this.pending = new Map();
    this.events = new Map();
  }

  async connect() {
    await new Promise((resolve, reject) => {
      this.socket.onerror = reject;
      this.socket.onopen = resolve;
    });
    this.socket.onmessage = (message) => {
      const data = JSON.parse(message.data);
      if (data.id) {
        const pending = this.pending.get(data.id);
        if (!pending) return;
        this.pending.delete(data.id);
        if (data.error) pending.reject(new Error(data.error.message));
        else pending.resolve(data.result);
        return;
      }
      const listeners = this.events.get(data.method) ?? [];
      listeners.forEach((listener) => listener(data.params ?? {}));
    };
  }

  send(method, params = {}, sessionId) {
    const id = this.nextID++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.socket.send(JSON.stringify({ id, method, params, ...(sessionId ? { sessionId } : {}) }));
    });
  }

  once(method) {
    return new Promise((resolve) => {
      const listeners = this.events.get(method) ?? [];
      const listener = (params) => {
        const current = this.events.get(method) ?? [];
        this.events.set(
          method,
          current.filter((candidate) => candidate !== listener)
        );
        resolve(params);
      };
      listeners.push(listener);
      this.events.set(method, listeners);
    });
  }

  close() {
    this.socket.close();
  }
}

async function runProbe(options) {
  const origin = new URL(options.url).origin;
  const initialBrowserMode = options.switchFrom || options.mode;
  const initialPointerMode = await ensureDevicePointerMode(options, initialBrowserMode);
  const browser = spawn(
    options.browser,
    [
      '--headless=new',
      '--no-sandbox',
      '--disable-gpu',
      '--disable-dev-shm-usage',
      '--ignore-certificate-errors',
      `--remote-debugging-port=${options.port}`,
      `--user-data-dir=/tmp/nanokvm-browser-probe-${process.pid}`,
      'about:blank'
    ],
    { stdio: 'ignore' }
  );
  let cdp;
  let sessionID;
  let targetID;
  try {
    const version = await waitFor(async () =>
      requestJSON(`http://127.0.0.1:${options.port}/json/version`)
    );
    cdp = new CdpClient(version.webSocketDebuggerUrl);
    await cdp.connect();
    const target = await cdp.send('Target.createTarget', { url: 'about:blank' });
    targetID = target.targetId;
    const attached = await cdp.send('Target.attachToTarget', { targetId: targetID, flatten: true });
    sessionID = attached.sessionId;
    await cdp.send('Network.enable', {}, sessionID);
    await cdp.send('Page.enable', {}, sessionID);
    await cdp.send('Runtime.enable', {}, sessionID);
    await cdp.send(
      'Network.setCookie',
      {
        name: 'nano-kvm-token',
        value: options.token,
        url: origin,
        secure: origin.startsWith('https:')
      },
      sessionID
    );
    await navigate(cdp, sessionID, `${origin}/`);
    await evaluate(
      cdp,
      sessionID,
      `(function () {
      localStorage.setItem('nano-kvm-mouse-mode', ${JSON.stringify(options.switchFrom || options.mode)});
      return location.href;
    })()`
    );
    await navigate(cdp, sessionID, `${origin}/`);
    const page = await waitFor(
      async () =>
        evaluate(
          cdp,
          sessionID,
          `({
      href: location.href,
      mouseMode: localStorage.getItem('nano-kvm-mouse-mode'),
      region: (function () {
        const element = document.getElementById('input-region');
        if (!element) return null;
        const rect = element.getBoundingClientRect();
        return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
      })()
    })`
        ),
      (value) =>
        value.region && !value.href.includes('/login') && value.mouseMode === initialBrowserMode
    );
    await delay(1000);
    let finalPointerMode = initialPointerMode;
    if (options.switchFrom) {
      finalPointerMode = await switchMouseMode(cdp, sessionID, options, initialPointerMode.after);
      await delay(300);
    }
    const before = await getRuntimeStatus(options);
    if (options.input === 'touch') {
      await dispatchTouch(cdp, sessionID, page.region, options.contacts);
    } else if (options.input === 'keyboard') {
      await dispatchKeyboard(cdp, sessionID);
    } else {
      await dispatchMouse(cdp, sessionID, page.region);
    }
    await delay(300);
    await cdp.send('Target.closeTarget', { targetId: targetID });
    await delay(300);
    const after = await getRuntimeStatus(options);
    const beforeSessionIDs = new Set(before.sessions.map((session) => session.id));
    const expectedRoute = routeForInput(options.mode, options.input);
    const expectedSource = sourceForInput(options.mode, options.input);
    const result = {
      schema: 'nanokvm-input-router-browser-probe/v2',
      timestamp: new Date().toISOString(),
      browser: options.browser,
      page,
      mode: options.mode,
      switchFrom: options.switchFrom || null,
      input: options.input,
      contacts: options.contacts,
      pointerMode: {
        initial: initialPointerMode,
        final: finalPointerMode
      },
      expectedRoute,
      expectedSource,
      delta: statusDelta(before, after),
      checks: {
        pageLoaded: Boolean(page.region),
        devicePointerModeConfirmed: finalPointerMode.confirmed,
        browserInputEvents:
          (after.sources.find((source) => source.source === expectedSource)?.events ?? 0) >
          (before.sources.find((source) => source.source === expectedSource)?.events ?? 0),
        selectedRouteDispatched:
          (after.routes.find((route) => route.route === expectedRoute)?.dispatched ?? 0) >
          (before.routes.find((route) => route.route === expectedRoute)?.dispatched ?? 0),
        noRejectedFrames: after.rejected === before.rejected,
        noDispatchFailures: after.dispatch_failed === before.dispatch_failed,
        noBackpressureDrops: after.backpressure_dropped === before.backpressure_dropped,
        browserSessionReleased:
          after.active_sessions <= before.active_sessions &&
          after.sessions.every((session) => beforeSessionIDs.has(session.id))
      },
      before,
      after
    };
    result.pass = Object.values(result.checks).every(Boolean);
    return result;
  } finally {
    if (cdp) cdp.close();
    browser.kill('SIGTERM');
  }
}

async function getRuntimeStatus(options) {
  const response = await requestJSON(
    `${options.url}/api/hid/input-router/status`,
    { Authorization: `Bearer ${options.token}` },
    options.insecure
  );
  if (response.code !== 0)
    throw new Error(`management request failed: ${response.msg ?? response.code}`);
  return response.data.runtime;
}

async function navigate(cdp, sessionID, url) {
  const loaded = cdp.once('Page.loadEventFired');
  await cdp.send('Page.navigate', { url }, sessionID);
  await loaded;
}

async function evaluate(cdp, sessionID, expression) {
  const result = await cdp.send('Runtime.evaluate', { expression, returnByValue: true }, sessionID);
  return result.result.value;
}

function touchPoints(region, count) {
  const centerX = region.left + region.width * 0.5;
  const centerY = region.top + region.height * 0.5;
  return Array.from({ length: count }, (_, index) => ({
    x: centerX + (index === 0 ? -24 : 24),
    y: centerY,
    radiusX: 8,
    radiusY: 8,
    force: 1,
    id: index + 1
  }));
}

async function dispatchTouch(cdp, sessionID, region, count) {
  await cdp.send(
    'Emulation.setTouchEmulationEnabled',
    {
      enabled: true,
      maxTouchPoints: count,
      configuration: 'mobile'
    },
    sessionID
  );
  const points = touchPoints(region, count);
  await cdp.send(
    'Input.dispatchTouchEvent',
    { type: 'touchStart', touchPoints: points },
    sessionID
  );
  await delay(100);
  await cdp.send(
    'Input.dispatchTouchEvent',
    {
      type: 'touchMove',
      touchPoints: points.map((point) => ({ ...point, y: point.y + 12 }))
    },
    sessionID
  );
  await delay(100);
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] }, sessionID);
}

async function dispatchKeyboard(cdp, sessionID) {
  await evaluate(cdp, sessionID, 'document.body.focus()');
  await cdp.send(
    'Input.dispatchKeyEvent',
    {
      type: 'keyDown',
      key: 'Shift',
      code: 'ShiftLeft',
      windowsVirtualKeyCode: 16,
      nativeVirtualKeyCode: 16
    },
    sessionID
  );
  await delay(100);
  await cdp.send(
    'Input.dispatchKeyEvent',
    {
      type: 'keyUp',
      key: 'Shift',
      code: 'ShiftLeft',
      windowsVirtualKeyCode: 16,
      nativeVirtualKeyCode: 16
    },
    sessionID
  );
}

async function dispatchMouse(cdp, sessionID, region) {
  const x = region.left + region.width * 0.5;
  const y = region.top + region.height * 0.5;
  await cdp.send('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y }, sessionID);
  await cdp.send(
    'Input.dispatchMouseEvent',
    {
      type: 'mousePressed',
      x,
      y,
      button: 'left',
      clickCount: 1
    },
    sessionID
  );
  await delay(100);
  await cdp.send(
    'Input.dispatchMouseEvent',
    {
      type: 'mouseReleased',
      x,
      y,
      button: 'left',
      clickCount: 1
    },
    sessionID
  );
}

async function switchMouseMode(cdp, sessionID, options, previousDeviceStatus) {
  const mode = options.mode;
  const openedMouseMenu = await evaluate(
    cdp,
    sessionID,
    `(function () {
    const icon = document.querySelector('svg.lucide-mouse');
    const trigger = icon?.closest('div[aria-disabled]');
    if (!trigger) return false;
    trigger.click();
    return true;
  })()`
  );
  if (!openedMouseMenu) throw new Error('mouse menu trigger not found');
  await delay(150);

  const modeMenuRegion = await evaluate(
    cdp,
    sessionID,
    `(function () {
    const icon = document.querySelector('svg.lucide-square-dashed-mouse-pointer');
    const trigger = icon?.closest('div[role="button"], div.cursor-pointer');
    if (!trigger) return null;
    const rect = trigger.getBoundingClientRect();
    return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 };
  })()`
  );
  if (!modeMenuRegion) throw new Error('mouse mode trigger not found');
  await cdp.send(
    'Input.dispatchMouseEvent',
    {
      type: 'mouseMoved',
      x: modeMenuRegion.x,
      y: modeMenuRegion.y
    },
    sessionID
  );
  await delay(300);

  const clicked = await evaluate(
    cdp,
    sessionID,
    `(function () {
    const option = document.querySelector(${JSON.stringify(`[data-mouse-mode="${mode}"]`)});
    if (!option) return false;
    option.click();
    return true;
  })()`
  );
  if (!clicked) throw new Error(`mouse mode option not found: ${mode}`);

  const pointerMode = await observeDevicePointerMode(options, mode, previousDeviceStatus);
  await waitFor(
    () =>
      evaluate(
        cdp,
        sessionID,
        `localStorage.getItem('nano-kvm-mouse-mode') === ${JSON.stringify(mode)}`
      ),
    Boolean,
    45_000
  );

  const closedMouseMenu = await evaluate(
    cdp,
    sessionID,
    `(function () {
    const icon = document.querySelector('svg.lucide-mouse');
    const trigger = icon?.closest('div[aria-disabled]');
    if (!trigger) return false;
    trigger.click();
    return true;
  })()`
  );
  if (!closedMouseMenu) throw new Error('mouse menu trigger disappeared after mode selection');
  return pointerMode;
}

function routeForMode(mode) {
  switch (mode) {
    case 'absolute':
      return 'hidg2/absolute-pointer';
    case 'digitizer-stylus':
      return 'hidg2/stylus';
    case 'relative':
      return 'hidg1/relative-pointer';
    default:
      return 'hidg2/touchscreen';
  }
}

function sourceForMode(mode) {
  return mode === 'relative' ? 'browser-pointer' : 'browser-touchscreen';
}

function routeForInput(mode, input) {
  if (input === 'keyboard') return 'hidg0/keyboard';
  return routeForMode(mode);
}

function sourceForInput(mode, input) {
  if (input === 'keyboard') return 'browser-keyboard';
  if (input === 'mouse' || mode === 'relative') return 'browser-pointer';
  return sourceForMode(mode);
}

async function requestJSON(url, headers = {}, insecure = false) {
  const target = new URL(url);
  const requestOptions = {
    protocol: target.protocol,
    hostname: target.hostname,
    port: target.port || (target.protocol === 'https:' ? 443 : 80),
    path: `${target.pathname}${target.search}`,
    method: 'GET',
    headers,
    ...(target.protocol === 'https:' && insecure ? { rejectUnauthorized: false } : {})
  };
  const transport =
    target.protocol === 'https:'
      ? https
      : await import('node:http').then(({ default: http }) => http);
  return new Promise((resolve, reject) => {
    const request = transport.request(requestOptions, (response) => {
      let body = '';
      response.setEncoding('utf8');
      response.on('data', (chunk) => {
        body += chunk;
      });
      response.on('end', () => {
        if (response.statusCode < 200 || response.statusCode >= 300) {
          reject(new Error(`HTTP ${response.statusCode}: ${body}`));
          return;
        }
        try {
          resolve(JSON.parse(body));
        } catch (error) {
          reject(error);
        }
      });
    });
    request.on('error', reject);
    request.end();
  });
}

async function waitFor(operation, predicate = (value) => Boolean(value), timeoutMS = 10000) {
  const deadline = Date.now() + timeoutMS;
  let lastError;
  while (Date.now() < deadline) {
    try {
      const value = await operation();
      if (predicate(value)) return value;
    } catch (error) {
      lastError = error;
    }
    await delay(100);
  }
  throw lastError ?? new Error('timed out waiting for browser probe');
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

function parseInteger(value, argument, minimum, maximum) {
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < minimum || parsed > maximum) {
    throw new Error(`${argument} must be an integer from ${minimum} to ${maximum}`);
  }
  return parsed;
}

const isMain = process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url;
if (isMain) {
  const options = parseOptions(process.argv.slice(2));
  runProbe(options)
    .then(async (result) => {
      const output = `${JSON.stringify(result, null, 2)}\n`;
      process.stdout.write(output);
      if (options.output) await fs.writeFile(options.output, output);
      process.exitCode = result.pass ? 0 : 1;
    })
    .catch((error) => {
      console.error(error.stack ?? error.message);
      process.exitCode = 1;
    });
}
