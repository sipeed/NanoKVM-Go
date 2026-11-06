import { spawn } from 'node:child_process';
import fs from 'node:fs/promises';
import http from 'node:http';
import https from 'node:https';
import { pathToFileURL } from 'node:url';
import websocket from 'websocket';

import { ensureDevicePointerMode } from './pointer-mode-probe.mjs';

const { w3cwebsocket: W3cWebSocket } = websocket;

const DIGITIZER_STYLUS_MODE = 'digitizer-stylus';
const LEGACY_DIGITIZER_STYLUS_MODE = 'absolute-android';
const FIREFOX_PROBE_MODES = ['relative', 'absolute', DIGITIZER_STYLUS_MODE, 'touch-screen'];

function normalizeProbeMode(value) {
  return value === LEGACY_DIGITIZER_STYLUS_MODE ? DIGITIZER_STYLUS_MODE : value;
}

export function parseOptions(argv, env = process.env) {
  const options = {
    url: env.NANOKVM_URL ?? '',
    token: env.NANOKVM_TOKEN ?? '',
    browser: env.FIREFOX ?? 'firefox',
    port: 9223,
    insecure: false,
    mode: 'relative',
    input: 'all',
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
        if (!FIREFOX_PROBE_MODES.includes(normalizeProbeMode(value))) {
          throw new Error(`unsupported Firefox probe mode: ${value}`);
        }
        options.mode = normalizeProbeMode(value);
        break;
      case '--input':
        if (!['all', 'keyboard', 'mouse', 'touch'].includes(value)) {
          throw new Error(`unsupported Firefox probe input: ${value}`);
        }
        options.input = value;
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

export function probeSessionEvidence(baseline, active) {
  const baselineIDs = new Set(baseline.sessions.map((session) => session.id));
  const sessions = active.sessions.filter((session) => !baselineIDs.has(session.id));
  return {
    ids: sessions.map((session) => session.id),
    sources: [
      ...new Set(sessions.flatMap((session) => session.sources.map((source) => source.source)))
    ],
    routes: [...new Set(sessions.flatMap((session) => session.routes))]
  };
}

class BidiClient {
  constructor(url) {
    this.socket = new W3cWebSocket(url);
    this.nextID = 1;
    this.pending = new Map();
  }

  async connect() {
    await new Promise((resolve, reject) => {
      this.socket.onopen = resolve;
      this.socket.onerror = reject;
    });
    this.socket.onmessage = ({ data }) => {
      const message = JSON.parse(data);
      const pending = this.pending.get(message.id);
      if (!pending) return;
      this.pending.delete(message.id);
      if (message.type === 'error')
        pending.reject(new Error(`${message.error}: ${message.message}`));
      else pending.resolve(message.result);
    };
  }

  send(method, params = {}) {
    const id = this.nextID++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
      this.socket.send(JSON.stringify({ id, method, params }));
    });
  }

  close() {
    this.socket.close();
  }
}

export async function runProbe(options) {
  const origin = new URL(options.url).origin;
  const pointerMode = await ensureDevicePointerMode(options, options.mode);
  const profileDir = await fs.mkdtemp('/tmp/nanokvm-firefox-probe-');
  const browser = spawn(
    options.browser,
    [
      '--headless',
      '--remote-debugging-port',
      String(options.port),
      '--remote-allow-hosts',
      '127.0.0.1,localhost',
      '--profile',
      profileDir,
      'about:blank'
    ],
    { stdio: 'ignore' }
  );
  let bidi;
  let context;
  let sessionStarted = false;
  try {
    bidi = await waitFor(async () => {
      const candidate = new BidiClient(`ws://127.0.0.1:${options.port}/session`);
      try {
        await candidate.connect();
        return candidate;
      } catch {
        candidate.close();
        return null;
      }
    }, 10000);
    await bidi.send('session.new', {
      capabilities: { alwaysMatch: { acceptInsecureCerts: true } }
    });
    sessionStarted = true;
    const baseline = await getRuntimeStatus(options);
    context = (await bidi.send('browsingContext.create', { type: 'tab' })).context;
    await bidi.send('browsingContext.navigate', { context, url: `${origin}/`, wait: 'complete' });
    await evaluate(
      bidi,
      context,
      `(function () {
        document.cookie = ${JSON.stringify(
          `nano-kvm-token=${options.token}; Path=/; Secure; SameSite=Lax`
        )};
        localStorage.setItem('nano-kvm-mouse-mode', ${JSON.stringify(options.mode)});
      })()`
    );
    await bidi.send('browsingContext.navigate', { context, url: `${origin}/`, wait: 'complete' });
    const page = await waitFor(
      () =>
        evaluate(
          bidi,
          context,
          `({
            href: location.href,
            ready: document.readyState,
            diagnostics: {
              mouseMode: localStorage.getItem('nano-kvm-mouse-mode'),
              screen: Boolean(document.getElementById('screen')),
              noSignalScreen: Boolean(document.getElementById('no-signal-screen')),
              inputRegion: Boolean(document.getElementById('input-region')),
              screenMetrics: (function () {
                const screen = document.getElementById('screen');
                if (!screen) return null;
                const rect = screen.getBoundingClientRect();
                const parentRect = screen.parentElement?.getBoundingClientRect();
                return {
                  tag: screen.tagName,
                  width: rect.width,
                  height: rect.height,
                  parentWidth: parentRect?.width || 0,
                  parentHeight: parentRect?.height || 0,
                  mediaWidth: screen.videoWidth || screen.width || 0,
                  mediaHeight: screen.videoHeight || screen.height || 0
                };
              })(),
              body: document.body.innerText.slice(0, 160)
            },
            region: (function () {
              const element = document.getElementById('input-region');
              if (!element) return null;
              const rect = element.getBoundingClientRect();
              return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
            })()
          })`
        ),
      10000,
      (value) =>
        value.ready === 'complete' &&
        value.region &&
        !value.href.includes('/login') &&
        value.diagnostics.mouseMode === options.mode
    );
    await delay(1000);
    const before = await getRuntimeStatus(options);
    if (options.input === 'all' || options.input === 'keyboard') {
      await dispatchKeyboard(bidi, context);
    }
    if (options.input === 'all' || options.input === 'mouse') {
      await dispatchMouse(bidi, context, page.region);
    }
    if (options.input === 'touch') {
      await dispatchTouch(bidi, context, page.region);
    }
    await delay(400);
    const active = await getRuntimeStatus(options);
    const sessionEvidence = probeSessionEvidence(baseline, active);
    await bidi.send('browsingContext.close', { context });
    context = undefined;
    await delay(400);
    const after = await getRuntimeStatus(options);
    const delta = statusDelta(before, active);
    const baselineSessionIDs = new Set(baseline.sessions.map((session) => session.id));
    const expectedSources = sourcesForInput(options.input);
    const expectedRoutes = routesForInput(options.mode, options.input);
    const result = {
      schema: 'nanokvm-input-router-firefox-probe/v2',
      timestamp: new Date().toISOString(),
      browser: options.browser,
      page,
      mode: options.mode,
      pointerMode,
      input: options.input,
      expectedSources,
      expectedRoutes,
      sessionEvidence,
      delta,
      checks: {
        authenticatedPage:
          page.ready === 'complete' && Boolean(page.region) && !page.href.includes('/login'),
        devicePointerModeConfirmed: pointerMode.confirmed,
        probeSessionCreated: sessionEvidence.ids.length > 0,
        browserSourcesCaptured: expectedSources.every((source) =>
          sessionEvidence.sources.includes(source)
        ),
        selectedRoutesDispatched: expectedRoutes.every((route) =>
          sessionEvidence.routes.includes(route)
        ),
        noRejectedFrames: delta.rejected === 0,
        noDispatchFailures: delta.dispatchFailed === 0,
        noBackpressureDrops: delta.backpressureDropped === 0,
        browserSessionReleased: after.sessions.every((session) =>
          baselineSessionIDs.has(session.id)
        )
      },
      baseline,
      active,
      before,
      after
    };
    result.pass = Object.values(result.checks).every(Boolean);
    return result;
  } finally {
    if (context && bidi) {
      try {
        await bidi.send('browsingContext.close', { context });
      } catch {}
    }
    if (sessionStarted && bidi) {
      try {
        await bidi.send('session.end');
      } catch {}
    }
    bidi?.close();
    browser.kill('SIGTERM');
    await fs.rm(profileDir, { recursive: true, force: true });
  }
}

async function dispatchKeyboard(bidi, context) {
  await bidi.send('input.performActions', {
    context,
    actions: [
      {
        type: 'key',
        id: 'keyboard',
        actions: [
          { type: 'keyDown', value: '\uE008' },
          { type: 'pause', duration: 100 },
          { type: 'keyUp', value: '\uE008' }
        ]
      }
    ]
  });
}

async function dispatchMouse(bidi, context, region) {
  const x = Math.round(region.left + region.width * 0.5);
  const y = Math.round(region.top + region.height * 0.5);
  await bidi.send('input.performActions', {
    context,
    actions: [
      {
        type: 'pointer',
        id: 'mouse',
        parameters: { pointerType: 'mouse' },
        actions: [
          { type: 'pointerMove', x, y, duration: 0, origin: 'viewport' },
          { type: 'pointerDown', button: 0 },
          { type: 'pause', duration: 100 },
          { type: 'pointerUp', button: 0 }
        ]
      }
    ]
  });
}

async function dispatchTouch(bidi, context, region) {
  const centerX = Math.round(region.left + region.width * 0.5);
  const centerY = Math.round(region.top + region.height * 0.5);
  const contacts = [
    { x: centerX - 24, y: centerY },
    { x: centerX + 24, y: centerY }
  ];
  await bidi.send('input.performActions', {
    context,
    actions: contacts.map((point, index) => ({
      type: 'pointer',
      id: `touch-${index + 1}`,
      parameters: { pointerType: 'touch' },
      actions: [
        { type: 'pointerMove', x: point.x, y: point.y, duration: 0, origin: 'viewport' },
        { type: 'pointerDown', button: 0 },
        { type: 'pause', duration: 100 },
        { type: 'pointerMove', x: point.x, y: point.y + 12, duration: 0, origin: 'viewport' },
        { type: 'pointerUp', button: 0 }
      ]
    }))
  });
}

function sourcesForInput(input) {
  if (input === 'keyboard') return ['browser-keyboard'];
  if (input === 'mouse') return ['browser-pointer'];
  if (input === 'touch') return ['browser-touchscreen'];
  return ['browser-keyboard', 'browser-pointer'];
}

function routesForInput(mode, input) {
  if (input === 'keyboard') return ['hidg0/keyboard'];
  if (input === 'mouse' || input === 'touch') return [routeForMode(mode)];
  return ['hidg0/keyboard', routeForMode(mode)];
}

function routeForMode(mode) {
  if (mode === 'touch-screen') return 'hidg2/touchscreen';
  if (mode === 'absolute') return 'hidg2/absolute-pointer';
  if (mode === 'digitizer-stylus') return 'hidg2/stylus';
  return 'hidg1/relative-pointer';
}

async function evaluate(bidi, context, expression) {
  const result = await bidi.send('script.evaluate', {
    target: { context },
    awaitPromise: false,
    resultOwnership: 'none',
    expression
  });
  return deserialize(result.result);
}

function deserialize(value) {
  if (!value || value.type !== 'object') return value?.value;
  return Object.fromEntries((value.value ?? []).map(([key, item]) => [key, deserialize(item)]));
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

function requestJSON(url, headers, insecure) {
  const target = new URL(url);
  const transport = target.protocol === 'https:' ? https : http;
  return new Promise((resolve, reject) => {
    const request = transport.request(
      target,
      {
        method: 'GET',
        headers,
        rejectUnauthorized: !insecure
      },
      (response) => {
        let body = '';
        response.setEncoding('utf8');
        response.on('data', (chunk) => {
          body += chunk;
        });
        response.on('end', () => {
          try {
            const value = JSON.parse(body);
            if (response.statusCode !== 200) reject(new Error(`HTTP ${response.statusCode}`));
            else resolve(value);
          } catch (error) {
            reject(error);
          }
        });
      }
    );
    request.on('error', reject);
    request.end();
  });
}

async function waitFor(operation, timeoutMS, predicate = (value) => Boolean(value)) {
  const deadline = Date.now() + timeoutMS;
  let lastError;
  let lastValue;
  while (Date.now() < deadline) {
    try {
      const value = await operation();
      if (predicate(value)) return value;
      lastValue = value;
    } catch (error) {
      lastError = error;
    }
    await delay(100);
  }
  if (lastError) throw lastError;
  throw new Error(
    `timed out waiting for Firefox WebDriver BiDi; last value: ${JSON.stringify(lastValue)}`
  );
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
