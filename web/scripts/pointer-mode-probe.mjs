import http from 'node:http';
import https from 'node:https';

export const DEVICE_POINTER_MODES = ['absolute', 'digitizer-stylus', 'touch-screen'];
const DEVICE_POINTER_MODE_SET = new Set(DEVICE_POINTER_MODES);
const POINTER_MODE_PATH = '/api/hid/pointer-mode';
const DEFAULT_POLL_ATTEMPTS = 90;
const DEFAULT_POLL_INTERVAL_MS = 500;
const DEFAULT_POLL_TIMEOUT_MS = 45_000;

export function requiredDevicePointerMode(browserMode) {
  if (browserMode === 'relative') return null;
  if (!DEVICE_POINTER_MODE_SET.has(browserMode)) {
    throw new Error(`unsupported pointer mode: ${browserMode}`);
  }
  return browserMode;
}

export function parsePointerModeResponse(response) {
  if (!response || response.code !== 0) {
    throw new Error(
      `pointer mode request failed: ${response?.msg ?? response?.code ?? 'invalid response'}`
    );
  }
  const data = response.data;
  if (!data || (!DEVICE_POINTER_MODE_SET.has(data.mode) && data.mode !== 'unknown')) {
    throw new Error(`pointer mode response has invalid mode: ${data?.mode}`);
  }
  if (typeof data.switching !== 'boolean') {
    throw new Error('pointer mode response has invalid switching state');
  }
  if (data.degraded !== undefined && typeof data.degraded !== 'boolean') {
    throw new Error('pointer mode response has invalid degraded state');
  }
  if (data.error !== undefined && typeof data.error !== 'string') {
    throw new Error('pointer mode response has invalid error detail');
  }
  return {
    mode: data.mode,
    switching: data.switching,
    degraded: Boolean(data.degraded) || data.mode === 'unknown',
    error: typeof data.error === 'string' ? data.error : ''
  };
}

export async function ensureDevicePointerMode(options, browserMode, dependencies = {}) {
  const targetMode = requiredDevicePointerMode(browserMode);
  const request = dependencies.request ?? requestJSON;
  const first = await getPointerMode(options, request);
  const before = first.switching
    ? await waitForPointerMode(
        options,
        (status) => !status.switching,
        dependencies,
        'current pointer mode switch did not finish'
      )
    : first;

  if (before.degraded && targetMode === null) {
    throw new Error(`device pointer mode is degraded: ${before.error || before.mode}`);
  }
  if (targetMode === null || (before.mode === targetMode && !before.degraded)) {
    return pointerModeEvidence(browserMode, targetMode, before, before, false, null);
  }

  let postResult = null;
  let postError = '';
  try {
    postResult = await request(
      `${options.url}${POINTER_MODE_PATH}`,
      {
        method: 'POST',
        headers: pointerModeHeaders(options.token, true),
        body: JSON.stringify({ mode: targetMode })
      },
      options.insecure
    );
  } catch (error) {
    // The USB network connection may disappear after the server accepted POST.
    postError = error instanceof Error ? error.message : String(error);
  }

  const after = await waitForPointerMode(
    options,
    (status) => {
      if (status.switching) return false;
      if (status.degraded || status.mode === targetMode) return true;
      return postResult !== null && postResult.code !== 0;
    },
    dependencies,
    `device did not confirm pointer mode ${targetMode}`
  );
  const evidence = pointerModeEvidence(
    browserMode,
    targetMode,
    before,
    after,
    true,
    postResult
      ? { received: true, code: postResult.code, message: postResult.msg ?? '' }
      : { received: false, code: null, message: postError }
  );
  if (!evidence.confirmed) {
    throw new Error(
      `pointer mode ${targetMode} was not activated; final=${JSON.stringify(after)}; ` +
        `post=${JSON.stringify(evidence.post)}`
    );
  }
  return evidence;
}

export async function observeDevicePointerMode(
  options,
  browserMode,
  previousStatus,
  dependencies = {}
) {
  const targetMode = requiredDevicePointerMode(browserMode);
  const after = await waitForPointerMode(
    options,
    (status) => {
      if (status.switching || status.degraded) return status.degraded;
      if (targetMode !== null) return status.mode === targetMode;
      return !previousStatus || status.mode === previousStatus.mode;
    },
    dependencies,
    targetMode === null
      ? 'relative mode unexpectedly changed the device pointer profile'
      : `browser did not activate pointer mode ${targetMode}`
  );
  const evidence = pointerModeEvidence(
    browserMode,
    targetMode,
    previousStatus ?? after,
    after,
    false,
    null
  );
  if (!evidence.confirmed) {
    throw new Error(`browser pointer mode was not confirmed: ${JSON.stringify(evidence)}`);
  }
  return evidence;
}

export async function getPointerMode(options, request = requestJSON) {
  const response = await request(
    `${options.url}${POINTER_MODE_PATH}`,
    { method: 'GET', headers: pointerModeHeaders(options.token, false) },
    options.insecure
  );
  return parsePointerModeResponse(response);
}

async function waitForPointerMode(options, predicate, dependencies, timeoutMessage) {
  const request = dependencies.request ?? requestJSON;
  const sleep = dependencies.sleep ?? delay;
  const attempts = dependencies.attempts ?? DEFAULT_POLL_ATTEMPTS;
  const intervalMs = dependencies.intervalMs ?? DEFAULT_POLL_INTERVAL_MS;
  const deadline = Date.now() + (dependencies.timeoutMs ?? DEFAULT_POLL_TIMEOUT_MS);
  let lastStatus = null;
  let lastError = null;
  for (let attempt = 0; attempt < attempts && Date.now() <= deadline; attempt += 1) {
    try {
      lastStatus = await getPointerMode(options, request);
      lastError = null;
      if (predicate(lastStatus)) return lastStatus;
    } catch (error) {
      lastError = error;
    }
    if (attempt + 1 < attempts && Date.now() < deadline) {
      await sleep(Math.min(intervalMs, Math.max(0, deadline - Date.now())));
    }
  }
  const detail = lastError?.message ?? JSON.stringify(lastStatus);
  throw new Error(`${timeoutMessage}; last=${detail}`);
}

function pointerModeEvidence(browserMode, targetMode, before, after, postAttempted, post) {
  const confirmed =
    !after.switching &&
    !after.degraded &&
    (targetMode === null ? after.mode === before.mode : after.mode === targetMode);
  return {
    browserMode,
    targetMode,
    before,
    after,
    postAttempted,
    post,
    confirmed
  };
}

function pointerModeHeaders(token, json) {
  return {
    Accept: 'application/json',
    Authorization: `Bearer ${token}`,
    ...(json ? { 'Content-Type': 'application/json' } : {})
  };
}

async function requestJSON(url, init = {}, insecure = false) {
  const target = new URL(url);
  const transport = target.protocol === 'https:' ? https : http;
  const body = init.body ?? '';
  const headers = {
    ...(init.headers ?? {}),
    ...(body ? { 'Content-Length': Buffer.byteLength(body) } : {})
  };
  return new Promise((resolve, reject) => {
    const request = transport.request(
      target,
      {
        method: init.method ?? 'GET',
        headers,
        rejectUnauthorized: !insecure
      },
      (response) => {
        let responseBody = '';
        response.setEncoding('utf8');
        response.on('data', (chunk) => {
          responseBody += chunk;
        });
        response.on('end', () => {
          if (!response.statusCode || response.statusCode < 200 || response.statusCode >= 300) {
            reject(new Error(`HTTP ${response.statusCode}: ${responseBody}`));
            return;
          }
          try {
            resolve(JSON.parse(responseBody));
          } catch (error) {
            reject(error);
          }
        });
      }
    );
    const timeoutMs =
      init.timeoutMs ?? (String(init.method).toUpperCase() === 'POST' ? 40_000 : 3_000);
    request.setTimeout(timeoutMs, () =>
      request.destroy(new Error('pointer mode request timed out'))
    );
    request.on('error', reject);
    request.end(body);
  });
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
