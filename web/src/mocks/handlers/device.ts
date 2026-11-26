import { http } from 'msw';

import {
  fail,
  isRecord,
  ok,
  readBoolean,
  readJsonObject,
  readNumber,
  readString
} from '../response.ts';
import type { MockFingerRobotArmID, MockFingerRobotConfig, MockState } from '../state.ts';

const mockAPKey = 'mock';

function getWifiResponse(state: MockState) {
  return {
    supported: state.network.wifi.supported,
    apMode: state.network.wifi.apMode,
    connected: state.network.wifi.connected,
    wifi: state.network.wifi.current
  };
}

function hasMockAPKey(request: Request): boolean {
  return request.headers.get('X-AP-Key') === mockAPKey;
}

function getWifiScanResponse(state: MockState) {
  return {
    wifiList: state.network.wifi.available.map((wifi) => ({ ...wifi }))
  };
}

function getVirtualDeviceResponse(state: MockState) {
  const audioMode = !state.virtualDevice.isMicEnabled
    ? 'off'
    : state.virtualDevice.isSpeakerEnabled
      ? 'duplex'
      : 'mic';

  return {
    isUsbDisplayEnabled: false,
    usbInEndpointsTotal: 7,
    usbInEndpointsUsed: (state.hid.pointerMode.singleEndpoint ? 2 : 3)
      + 2 * Number(state.virtualDevice.isWebcamEnabled) + 2 * Number(state.virtualDevice.isNetworkEnabled)
      + Number(state.virtualDevice.isMicEnabled) + Number(state.virtualDevice.isDiskEnabled),
    isWebcamEnabled: state.virtualDevice.isWebcamEnabled,
    isNetworkEnabled: state.virtualDevice.isNetworkEnabled,
    isDiskEnabled: state.virtualDevice.isDiskEnabled,
    isSdCardExist: state.virtualDevice.isSdCardExist,
    diskMinSizeMiB: state.virtualDevice.diskMinSizeMiB,
    diskMaxSizeMiB: state.virtualDevice.diskMaxSizeMiB,
    isMicEnabled: state.virtualDevice.isMicEnabled,
    isSpeakerEnabled: state.virtualDevice.isSpeakerEnabled,
    audioMode
  };
}

function getFingerRobotDebugResponse(state: MockState) {
  const debug = state.vm.fingerRobotDebug;
  if (!debug.enabled) {
    return { enabled: debug.enabled, available: debug.available };
  }

  const controller = state.vm.fingerRobot;
  const ownedByDebug = controller.owner === 'debug';

  return {
    ...debug,
    releaseUs: { ...debug.releaseUs },
    pressUs: { ...debug.pressUs },
    count: { ...debug.count },
    pressMs: { ...debug.pressMs },
    pressSpeedUsPerSecond: { ...debug.pressSpeedUsPerSecond },
    running: ownedByDebug ? controller.status.running : false,
    current: ownedByDebug ? debug.current : 0,
    total: ownedByDebug ? debug.total : 0,
    state: ownedByDebug ? controller.status.state : 'idle',
    result: ownedByDebug ? controller.status.result : 'idle',
    error: ownedByDebug ? controller.status.error : ''
  };
}

function syncFingerRobotControllerFromDebug(state: MockState) {
  const debug = state.vm.fingerRobotDebug;
  state.vm.fingerRobot.status = {
    running: debug.running,
    state: debug.state as MockState['vm']['fingerRobot']['status']['state'],
    result: debug.result as MockState['vm']['fingerRobot']['status']['result'],
    error: debug.error
  };
}

function resetFingerRobotController(state: MockState) {
  const debug = state.vm.fingerRobotDebug;
  debug.available = true;
  debug.running = false;
  debug.current = 0;
  debug.total = 0;
  debug.state = 'idle';
  debug.result = 'reset';
  debug.error = '';
  state.vm.fingerRobot.available = true;
  state.vm.fingerRobot.operationRevision += 1;
  state.vm.fingerRobot.owner = 'debug';
  state.vm.fingerRobot.operationKind = 'reset';
  syncFingerRobotControllerFromDebug(state);
}

function cloneFingerRobotConfig(state: MockState) {
  const config = state.vm.fingerRobot.config;
  return {
    ...config,
    strengthByArm: { ...config.strengthByArm }
  };
}

function getFingerRobotResponse(state: MockState) {
  const fingerRobot = state.vm.fingerRobot;
  return {
    enabled: fingerRobot.enabled,
    available: fingerRobot.available,
    config: cloneFingerRobotConfig(state),
    profiles: fingerRobot.profiles.map((profile) => ({ ...profile })),
    strength: { ...fingerRobot.strength },
    pressMs: { ...fingerRobot.pressMs },
    speedOptions: [...fingerRobot.speedOptions],
    defaultSpeed: fingerRobot.defaultSpeed,
    status: { ...fingerRobot.status }
  };
}

async function readStrictJsonObject(request: Request): Promise<Record<string, unknown> | null> {
  try {
    const body: unknown = await request.json();
    return isRecord(body) ? body : null;
  } catch {
    return null;
  }
}

function hasOnlyFields(body: Record<string, unknown>, fields: ReadonlySet<string>): boolean {
  return Object.keys(body).every((field) => fields.has(field));
}

function hasValidOptionalChallenge(body: Record<string, unknown>): boolean {
  return (
    !Object.prototype.hasOwnProperty.call(body, 'challenge') || typeof body.challenge === 'string'
  );
}

function verifyTargetPowerChallenge(state: MockState, challenge: string | null): boolean {
  if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.targetPower) return true;
  if (!state.auth.targetPowerChallenge || challenge !== state.auth.targetPowerChallenge)
    return false;
  state.auth.targetPowerChallenge = null;
  return true;
}

const fingerRobotConfigFields = new Set([
  'version',
  'revision',
  'activeArm',
  'strengthByArm',
  'pressMs',
  'speed',
  'challenge'
]);
const fingerRobotStrengthFields = new Set(['long', 'short']);
// Press, replace-arm, and engineering debug/test verify a target_power challenge;
// config save and release accept the optional field but never consume it.
const fingerRobotActionFields = new Set(['challenge']);

type MockFingerRobotSetResult = {
  config: MockFingerRobotConfig;
  changed: boolean;
};

function readFingerRobotConfigUpdate(state: MockState, body: Record<string, unknown>) {
  if (!hasOnlyFields(body, fingerRobotConfigFields) || !hasValidOptionalChallenge(body)) {
    return null;
  }

  const version = readNumber(body, 'version');
  const revision = readNumber(body, 'revision');
  const activeArm = readString(body, 'activeArm');
  const pressMs = readNumber(body, 'pressMs');
  const speed = readString(body, 'speed');
  const strengths = body.strengthByArm;
  if (
    version !== 1 ||
    !Number.isSafeInteger(revision) ||
    revision === null ||
    revision < 1 ||
    (activeArm !== 'long' && activeArm !== 'short') ||
    !isIntegerInRange(
      pressMs,
      state.vm.fingerRobot.pressMs.min,
      state.vm.fingerRobot.pressMs.max
    ) ||
    (speed !== 'slow' && speed !== 'fast') ||
    !isRecord(strengths) ||
    !hasOnlyFields(strengths, fingerRobotStrengthFields)
  ) {
    return null;
  }

  const longStrength = readNumber(strengths, 'long');
  const shortStrength = readNumber(strengths, 'short');
  if (
    !isIntegerInRange(
      longStrength,
      state.vm.fingerRobot.strength.min,
      state.vm.fingerRobot.strength.max
    ) ||
    !isIntegerInRange(
      shortStrength,
      state.vm.fingerRobot.strength.min,
      state.vm.fingerRobot.strength.max
    )
  ) {
    return null;
  }

  return {
    version: 1 as const,
    revision,
    activeArm: activeArm as MockFingerRobotArmID,
    strengthByArm: { long: longStrength, short: shortStrength },
    pressMs,
    speed: speed as MockFingerRobotConfig['speed']
  };
}

function isSameFingerRobotConfig(
  current: MockState['vm']['fingerRobot']['config'],
  update: NonNullable<ReturnType<typeof readFingerRobotConfigUpdate>>
): boolean {
  return (
    current.activeArm === update.activeArm &&
    current.strengthByArm.long === update.strengthByArm.long &&
    current.strengthByArm.short === update.strengthByArm.short &&
    current.pressMs === update.pressMs &&
    current.speed === update.speed
  );
}

function waitForFingerRobotMock(milliseconds: number) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}

function fingerRobotOperationIsActive(state: MockState, operationRevision: number): boolean {
  const fingerRobot = state.vm.fingerRobot;
  return fingerRobot.operationRevision === operationRevision && fingerRobot.status.running;
}

async function runFingerRobotMockPress(
  state: MockState,
  operationRevision: number,
  pressMs: number
): Promise<boolean> {
  const fingerRobot = state.vm.fingerRobot;

  await waitForFingerRobotMock(15);
  if (!fingerRobotOperationIsActive(state, operationRevision)) return false;
  fingerRobot.status.state = 'pressing';

  await waitForFingerRobotMock(30);
  if (!fingerRobotOperationIsActive(state, operationRevision)) return false;
  fingerRobot.status.state = 'holding';

  const mockHoldMs = Math.min(60, Math.max(20, Math.round(pressMs / 40)));
  await waitForFingerRobotMock(mockHoldMs);
  if (!fingerRobotOperationIsActive(state, operationRevision)) return false;
  fingerRobot.status.state = 'releasing';

  await waitForFingerRobotMock(30);
  if (!fingerRobotOperationIsActive(state, operationRevision)) return false;
  fingerRobot.status.running = false;
  fingerRobot.status.state = 'idle';
  fingerRobot.status.result = 'success';
  fingerRobot.status.error = '';
  return true;
}

function isIntegerInRange(value: number | null, min: number, max: number): value is number {
  return value !== null && Number.isSafeInteger(value) && value >= min && value <= max;
}

const fingerRobotDebugTestFields = new Set([
  'releaseUs',
  'pressUs',
  'count',
  'pressMs',
  'pressSpeedUsPerSecond',
  'challenge'
]);

export function createDeviceHandlers(origin: string, state: MockState) {
  function jigglerStatus() {
    const config = state.vm.mouseJiggler;
    const remainingMs = config.enabled && config.maxHours > 0
      ? Math.max(0, config.startedAt + config.maxHours * 3600000 - Date.now())
      : 0;
    if (config.maxHours > 0 && !remainingMs) config.enabled = false;
    const pointer = state.hid.pointerMode;
    const blockedByPointerMode = !!pointer.singleEndpoint && (pointer.mode === 'touch-screen' || pointer.mode === 'digitizer-stylus');
    return { ...config, blockedByPointerMode, remainingMs, elapsedMs: config.enabled ? Date.now() - config.startedAt : 0 };
  }

  return [
    http.get(`${origin}/api/vm/info`, () =>
      ok({
        image: 'mock-image-2026.01',
        ips: [{ name: 'eth0', addr: '192.0.2.42', version: 'IPv4', type: 'Wired' }],
        mdns: `${state.vm.hostname}.local`,
        application: state.application.current,
        deviceKey: 'mock-development-device',
        pn: 'NanoKVM Mock',
        arch: 'armv7'
      })
    ),
    http.get(`${origin}/api/vm/hardware`, () => ok({ model: 'mock', memory: 2048 })),
    http.get(`${origin}/api/vm/finger-robot`, () => ok(getFingerRobotResponse(state))),
    http.put(`${origin}/api/vm/finger-robot`, async ({ request }) => {
      const body = await readStrictJsonObject(request);
      const update = body === null ? null : readFingerRobotConfigUpdate(state, body);
      if (body === null || update === null) {
        return fail<MockFingerRobotSetResult>(-1, 'invalid arguments');
      }

      const fingerRobot = state.vm.fingerRobot;
      if (update.revision !== fingerRobot.config.revision) {
        return fail<MockFingerRobotSetResult>(-6, 'revision conflict', {
          config: cloneFingerRobotConfig(state),
          changed: false
        });
      }

      const changed = !isSameFingerRobotConfig(fingerRobot.config, update);
      if (changed) {
        fingerRobot.config = {
          ...update,
          revision: fingerRobot.config.revision + 1,
          strengthByArm: { ...update.strengthByArm }
        };
      }
      return ok<MockFingerRobotSetResult>({ config: cloneFingerRobotConfig(state), changed });
    }),
    http.post(`${origin}/api/vm/finger-robot/press`, async ({ request }) => {
      const body = await readStrictJsonObject(request);
      if (
        body === null ||
        !hasOnlyFields(body, fingerRobotActionFields) ||
        !hasValidOptionalChallenge(body)
      ) {
        return fail(-1, 'invalid arguments', null);
      }
      if (!verifyTargetPowerChallenge(state, readString(body, 'challenge'))) {
        return fail(-3, 'sensitive action verification required', null);
      }

      const fingerRobot = state.vm.fingerRobot;
      if (fingerRobot.status.running || state.vm.fingerRobotDebug.running) {
        return fail(-2, 'finger robot operation is busy', null);
      }
      if (!fingerRobot.enabled || !fingerRobot.available) {
        return fail(-5, 'finger robot is unavailable', null);
      }

      const operationRevision = ++fingerRobot.operationRevision;
      fingerRobot.owner = 'normal';
      fingerRobot.operationKind = 'press';
      fingerRobot.status = {
        running: true,
        state: 'releasing',
        result: 'running',
        error: ''
      };
      const completed = await runFingerRobotMockPress(
        state,
        operationRevision,
        fingerRobot.config.pressMs
      );
      return completed ? ok(null) : fail(-5, 'operation failed', null);
    }),
    http.post(`${origin}/api/vm/finger-robot/release`, async ({ request }) => {
      const body = await readStrictJsonObject(request);
      if (
        body === null ||
        !hasOnlyFields(body, fingerRobotActionFields) ||
        !hasValidOptionalChallenge(body)
      ) {
        return fail(-1, 'invalid arguments', null);
      }
      const fingerRobot = state.vm.fingerRobot;
      fingerRobot.operationRevision += 1;
      fingerRobot.owner = 'normal';
      fingerRobot.operationKind = 'release';
      fingerRobot.status = {
        running: true,
        state: 'releasing',
        result: 'running',
        error: ''
      };
      if (state.vm.fingerRobotDebug.running) {
        state.vm.fingerRobotDebug.running = false;
        state.vm.fingerRobotDebug.state = 'idle';
        state.vm.fingerRobotDebug.result = 'cancelled';
        state.vm.fingerRobotDebug.error = '';
      }

      await waitForFingerRobotMock(5);
      fingerRobot.status.running = false;
      fingerRobot.status.state = 'idle';
      fingerRobot.status.result = 'success';
      return ok(null);
    }),
    http.post(`${origin}/api/vm/finger-robot/replace-arm`, async ({ request }) => {
      const body = await readStrictJsonObject(request);
      if (
        body === null ||
        !hasOnlyFields(body, fingerRobotActionFields) ||
        !hasValidOptionalChallenge(body)
      ) {
        return fail(-1, 'invalid arguments', null);
      }
      if (!state.vm.fingerRobot.enabled || !state.vm.fingerRobot.available) {
        return fail(-5, 'finger robot is unavailable', null);
      }
      if (!verifyTargetPowerChallenge(state, readString(body, 'challenge'))) {
        return fail(-3, 'sensitive action verification required', null);
      }

      resetFingerRobotController(state);
      return ok(null);
    }),
    http.get(`${origin}/api/vm/finger-robot/debug`, () => {
      const response = getFingerRobotDebugResponse(state);
      const debug = state.vm.fingerRobotDebug;

      // Advance after snapshotting so mocked pages visibly exercise polling/progress.
      if (debug.running) {
        if (debug.total === debug.count.unlimitedValue) {
          debug.current += 1;
          debug.state = 'releasing';
        } else if (debug.current < debug.total) {
          debug.current += 1;
          debug.state = 'releasing';
        } else {
          debug.running = false;
          debug.state = 'releasing';
          debug.result = 'success';
        }
        syncFingerRobotControllerFromDebug(state);
      }
      return ok(response);
    }),
    http.post(`${origin}/api/vm/finger-robot/debug/test`, async ({ request }) => {
      const body = await readJsonObject(request);
      const challenge = readString(body, 'challenge');
      const debug = state.vm.fingerRobotDebug;

      if (!debug.enabled) {
        return fail(-3, 'finger robot debug mode is disabled', null);
      }
      if (!debug.available) {
        return fail(-3, 'finger robot is unavailable', null);
      }
      if (Object.keys(body).some((field) => !fingerRobotDebugTestFields.has(field))) {
        return fail(-1, 'invalid finger robot debug parameters', null);
      }
      if (state.auth.twoFactorEnabled && state.auth.twoFactorProtection.targetPower) {
        if (!state.auth.targetPowerChallenge || challenge !== state.auth.targetPowerChallenge) {
          return fail(-3, 'sensitive action verification required', null);
        }
        state.auth.targetPowerChallenge = null;
      }
      if (debug.running || state.vm.fingerRobot.status.running) {
        return fail(-2, 'finger robot is busy', null);
      }

      const releaseUs = readNumber(body, 'releaseUs');
      const pressUs = readNumber(body, 'pressUs');
      const count = readNumber(body, 'count');
      const pressMs = readNumber(body, 'pressMs');
      const pressSpeedUsPerSecond = readNumber(body, 'pressSpeedUsPerSecond');
      if (
        !isIntegerInRange(releaseUs, debug.releaseUs.min, debug.releaseUs.max) ||
        !isIntegerInRange(pressUs, debug.pressUs.min, debug.pressUs.max) ||
        pressUs >= releaseUs ||
        (count !== debug.count.unlimitedValue &&
          (count === null || !Number.isSafeInteger(count) || count < debug.count.min)) ||
        !isIntegerInRange(pressMs, debug.pressMs.min, debug.pressMs.max) ||
        !isIntegerInRange(
          pressSpeedUsPerSecond,
          debug.pressSpeedUsPerSecond.min,
          debug.pressSpeedUsPerSecond.max
        )
      ) {
        return fail(-1, 'invalid finger robot debug parameters', null);
      }

      debug.running = true;
      debug.current = 0;
      debug.total = count;
      debug.state = 'pressing';
      debug.result = 'running';
      debug.error = '';
      state.vm.fingerRobot.operationRevision += 1;
      state.vm.fingerRobot.owner = 'debug';
      state.vm.fingerRobot.operationKind = 'test';
      syncFingerRobotControllerFromDebug(state);
      return ok(null);
    }),
    http.post(`${origin}/api/vm/finger-robot/debug/stop`, () => {
      const debug = state.vm.fingerRobotDebug;
      if (!debug.enabled) {
        return fail(-3, 'finger robot debug mode is disabled', null);
      }

      const fingerRobot = state.vm.fingerRobot;
      if (fingerRobot.owner === 'normal' && fingerRobot.status.running) {
        return fail(-2, 'finger robot is busy', null);
      }

      if (!debug.running) return ok(null);

      debug.running = false;
      debug.state = 'idle';
      debug.result = 'cancelled';
      debug.error = '';
      fingerRobot.operationRevision += 1;
      fingerRobot.owner = 'debug';
      fingerRobot.operationKind = 'stop';
      syncFingerRobotControllerFromDebug(state);
      return ok(null);
    }),
    http.post(`${origin}/api/vm/finger-robot/debug/reset`, () => {
      const debug = state.vm.fingerRobotDebug;
      if (!debug.enabled) {
        return fail(-3, 'finger robot debug mode is disabled', null);
      }

      resetFingerRobotController(state);
      return ok(null);
    }),
    http.post(`${origin}/api/vm/system/reboot`, async ({ request }) => {
      const challenge = readString(await readJsonObject(request), 'challenge');
      if (state.auth.twoFactorEnabled && state.auth.twoFactorProtection.nanoKVMRestart) {
        if (
          !state.auth.nanoKVMRestartChallenge ||
          challenge !== state.auth.nanoKVMRestartChallenge
        ) {
          return fail(-3, 'sensitive action verification required', { scheduled: false });
        }
        state.auth.nanoKVMRestartChallenge = null;
      }
      return ok({ scheduled: true });
    }),
    http.post(`${origin}/api/vm/system/reboot-to-flashing`, async ({ request }) => {
      const challenge = readString(await readJsonObject(request), 'challenge');
      if (state.auth.twoFactorEnabled && state.auth.twoFactorProtection.nanoKVMRestart) {
        if (
          !state.auth.nanoKVMRestartChallenge ||
          challenge !== state.auth.nanoKVMRestartChallenge
        ) {
          return fail(-3, 'sensitive action verification required', { scheduled: false });
        }
        state.auth.nanoKVMRestartChallenge = null;
      }
      return ok({ scheduled: true, mode: 'flashing' });
    }),

    http.get(`${origin}/api/vm/hdmi/capture`, () => ok({ enabled: state.vm.hdmiCaptureEnabled })),
    http.post(`${origin}/api/vm/hdmi/capture`, async ({ request }) => {
      const enabled = readBoolean(await readJsonObject(request), 'enabled');
      if (enabled !== null) state.vm.hdmiCaptureEnabled = enabled;
      return ok({ enabled: state.vm.hdmiCaptureEnabled });
    }),
    http.get(`${origin}/api/vm/ssh`, () => ok({ enabled: state.vm.sshEnabled })),
    http.post(`${origin}/api/vm/ssh/enable`, () => {
      state.vm.sshEnabled = true;
      return ok({ enabled: true });
    }),
    http.post(`${origin}/api/vm/ssh/disable`, () => {
      state.vm.sshEnabled = false;
      return ok({ enabled: false });
    }),
    http.get(`${origin}/api/vm/mdns`, () => ok({ enabled: state.vm.mdnsEnabled })),
    http.post(`${origin}/api/vm/mdns/enable`, () => {
      state.vm.mdnsEnabled = true;
      return ok({ enabled: true });
    }),
    http.post(`${origin}/api/vm/mdns/disable`, () => {
      state.vm.mdnsEnabled = false;
      return ok({ enabled: false });
    }),
    http.get(`${origin}/api/vm/mouse-jiggler`, () => ok(jigglerStatus())),
    http.post(`${origin}/api/vm/mouse-jiggler`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const mode = readString(body, 'mode');
      const hours = readNumber(body, 'maxHours');
      const config = state.vm.mouseJiggler;
      const status = jigglerStatus();
      if (status.blockedByPointerMode && enabled === true) {
        return fail(-3, 'Mouse Jiggler requires a USB mouse endpoint.', status);
      }
      if (enabled === true && !config.enabled) config.startedAt = Date.now();
      if (enabled !== null) config.enabled = enabled;
      if (mode) config.mode = mode;
      if (hours !== null && hours !== config.maxHours) {
        config.maxHours = hours;
        if (config.enabled) config.startedAt = Date.now();
      }
      return ok(jigglerStatus());
    }),
    http.get(`${origin}/api/vm/hostname`, () => ok({ hostname: state.vm.hostname })),
    http.post(`${origin}/api/vm/hostname`, async ({ request }) => {
      const hostname = readString(await readJsonObject(request), 'hostname');
      if (hostname !== null) state.vm.hostname = hostname;
      return ok({ hostname: state.vm.hostname });
    }),
    http.get(`${origin}/api/vm/web-title`, () => ok({ title: state.vm.webTitle })),
    http.post(`${origin}/api/vm/web-title`, async ({ request }) => {
      const title = readString(await readJsonObject(request), 'title');
      if (title !== null) state.vm.webTitle = title;
      return ok({ title: state.vm.webTitle });
    }),
    http.get(`${origin}/api/vm/low-power`, () => ok({ enabled: state.vm.lowPowerEnabled })),
    http.post(`${origin}/api/vm/low-power`, async ({ request }) => {
      const enabled = readBoolean(await readJsonObject(request), 'enable');
      if (enabled !== null) state.vm.lowPowerEnabled = enabled;
      return ok({ enabled: state.vm.lowPowerEnabled });
    }),
    http.get(`${origin}/api/vm/swap`, () => ok({ enabled: state.vm.swapEnabled })),
    http.post(`${origin}/api/vm/swap`, async ({ request }) => {
      const enabled = readBoolean(await readJsonObject(request), 'enable');
      if (enabled !== null) state.vm.swapEnabled = enabled;
      return ok({ enabled: state.vm.swapEnabled });
    }),

    http.get(`${origin}/api/vm/edid`, () => ok({ edid: state.stream.edid })),
    http.post(`${origin}/api/vm/edid`, async ({ request }) => {
      const edid = readString(await readJsonObject(request), 'edid');
      if (edid) state.stream.edid = edid;
      return ok({ edid: state.stream.edid });
    }),
    http.get(`${origin}/api/vm/edid/custom`, () => ok({ edidList: state.stream.customEdids })),
    http.post(`${origin}/api/vm/edid/delete`, async ({ request }) => {
      const edid = readString(await readJsonObject(request), 'edid');
      if (edid) state.stream.customEdids = state.stream.customEdids.filter((item) => item !== edid);
      return ok({ edidList: state.stream.customEdids });
    }),
    http.post(`${origin}/api/vm/edid/upload`, () => {
      const file = 'mock-uploaded-display.bin';
      if (!state.stream.customEdids.includes(file)) state.stream.customEdids.push(file);
      return ok({ file });
    }),

    http.get(`${origin}/api/vm/dial-background`, () => ok(state.vm.dialBackground)),
    http.post(`${origin}/api/vm/dial-background`, async ({ request }) => {
      const imageName = readString(await readJsonObject(request), 'imageName');
      if (imageName && state.vm.dialBackground.files.some((file) => file.name === imageName)) {
        state.vm.dialBackground.current = imageName;
      }
      return ok(state.vm.dialBackground);
    }),
    http.post(`${origin}/api/vm/dial-background/upload`, () => {
      const name = 'mock-uploaded-background.bmp';
      const file = {
        name,
        size: 72_000,
        modTime: state.clock.epochMs,
        url: '/mock-uploaded-background.bmp'
      };
      state.vm.dialBackground.files = [
        ...state.vm.dialBackground.files.filter((current) => current.name !== name),
        file
      ];
      return ok({ file });
    }),

    http.get(`${origin}/api/vm/timezone`, () => ok({ timezone: state.vm.timezone })),
    http.post(`${origin}/api/vm/timezone`, async ({ request }) => {
      const timezone = readString(await readJsonObject(request), 'timezone');
      if (timezone) state.vm.timezone = timezone;
      return ok({ timezone: state.vm.timezone });
    }),
    http.post(`${origin}/api/vm/time`, async ({ request }) => {
      const body = await readJsonObject(request);
      const datetime = readString(body, 'datetime');
      if (!datetime) return fail(-2, 'invalid parameters', {});

      const parsed = Date.parse(`${datetime}Z`);
      if (!Number.isFinite(parsed)) return fail(-1, 'invalid parameters', {});

      state.clock.epochMs = parsed;
      return ok({ timestamp: state.clock.epochMs });
    }),
    http.get(`${origin}/api/vm/time/status`, () =>
      ok({ isSynchronized: true, lastSyncTime: state.vm.lastSyncTime })
    ),
    http.post(`${origin}/api/vm/time/sync`, () => {
      state.vm.lastSyncTime = state.clock.epochMs;
      return ok({ isSynchronized: true, lastSyncTime: state.vm.lastSyncTime });
    }),
    http.get(`${origin}/api/vm/menubar`, () => ok({ disabledItems: state.menu.disabledItems })),
    http.post(`${origin}/api/vm/menubar`, async ({ request }) => {
      const value = (await readJsonObject(request)).disabledItems;
      if (Array.isArray(value)) {
        state.menu.disabledItems = value.filter((item): item is string => typeof item === 'string');
      }
      return ok({ disabledItems: state.menu.disabledItems });
    }),
    http.get(`${origin}/api/vm/onboarding`, () =>
      ok({
        completedVersion: state.onboarding.completedVersion,
        mobileCompletedVersion: state.onboarding.mobileCompletedVersion
      })
    ),
    http.post(`${origin}/api/vm/onboarding`, async ({ request }) => {
      // Mirrors the backend contract: both fields optional and independently
      // validated, both empty/missing -> -1, no half-writes on invalid input.
      const body = await readJsonObject(request);
      const completed = readString(body, 'completedVersion') || null;
      const mobile = readString(body, 'mobileCompletedVersion') || null;
      const pattern = /^[a-z0-9][a-z0-9.-]{0,31}$/;
      if (completed === null && mobile === null) {
        return fail(-1, 'invalid arguments');
      }
      if (
        (completed !== null && !pattern.test(completed)) ||
        (mobile !== null && !pattern.test(mobile))
      ) {
        return fail(-1, 'invalid arguments');
      }
      if (completed !== null) state.onboarding.completedVersion = completed;
      if (mobile !== null) state.onboarding.mobileCompletedVersion = mobile;
      return ok(null);
    }),

    http.get(`${origin}/api/stream/encoder`, () =>
      ok({
        rate_control: state.stream.rateControl,
        bitrate: state.stream.bitrate,
        gop: state.stream.gop,
        fps: state.stream.fps
      })
    ),
    http.post(`${origin}/api/stream/rate-control`, async ({ request }) => {
      const mode = readString(await readJsonObject(request), 'mode');
      if (mode === 'cbr' || mode === 'vbr') state.stream.rateControl = mode;
      return ok({ rate_control: state.stream.rateControl });
    }),
    http.post(`${origin}/api/stream/mode`, async ({ request }) => {
      const mode = readString(await readJsonObject(request), 'mode');
      if (mode) state.stream.mode = mode;
      return ok({ mode: state.stream.mode });
    }),
    http.post(`${origin}/api/stream/fps`, async ({ request }) => {
      const fps = readNumber(await readJsonObject(request), 'fps');
      if (fps !== null) state.stream.fps = fps;
      return ok({ fps: state.stream.fps });
    }),
    http.post(`${origin}/api/stream/quality`, async ({ request }) => {
      const bitrate = readNumber(await readJsonObject(request), 'quality');
      if (bitrate !== null) state.stream.bitrate = bitrate;
      return ok({ bitrate: state.stream.bitrate });
    }),
    http.post(`${origin}/api/stream/gop`, async ({ request }) => {
      const gop = readNumber(await readJsonObject(request), 'gop');
      if (gop !== null) state.stream.gop = gop;
      return ok({ gop: state.stream.gop });
    }),

    http.get(`${origin}/api/hid/mode`, () => ok({ mode: state.hid.mode })),
    http.post(`${origin}/api/hid/mode`, async ({ request }) => {
      const mode = readString(await readJsonObject(request), 'mode');
      if (mode) state.hid.mode = mode;
      return ok({ mode: state.hid.mode });
    }),
    http.get(`${origin}/api/hid/pointer-mode`, () => ok(state.hid.pointerMode)),
    http.post(`${origin}/api/hid/pointer-mode`, async ({ request }) => {
      const body = await readJsonObject(request);
      const mode = readString(body, 'mode');
      const singleEndpoint = body.singleEndpoint === true;
      if (!(singleEndpoint && mode === 'relative') && mode !== 'absolute' && mode !== 'digitizer-stylus' && mode !== 'touch-screen') {
        return fail(-1, 'invalid pointer mode', state.hid.pointerMode);
      }
      state.hid.pointerMode = {
        mode,
        singleEndpoint,
        switching: false,
        degraded: false,
        error: ''
      };
      return ok(state.hid.pointerMode);
    }),
    http.post(`${origin}/api/hid/paste`, () => ok({})),
    http.post(`${origin}/api/hid/reset`, () => ok({})),
    http.get(`${origin}/api/hid/shortcuts`, () => ok({ shortcuts: state.hid.shortcuts })),
    http.post(`${origin}/api/hid/shortcut`, async ({ request }) => {
      const keys = (await readJsonObject(request)).keys;
      const shortcut = {
        id: `mock-shortcut-${state.hid.shortcuts.length + 1}`,
        keys: Array.isArray(keys)
          ? keys.filter((key): key is string => typeof key === 'string')
          : []
      };
      state.hid.shortcuts.push(shortcut);
      return ok(shortcut);
    }),
    http.delete(`${origin}/api/hid/shortcut`, async ({ request }) => {
      const id = readString(await readJsonObject(request), 'id');
      if (id) state.hid.shortcuts = state.hid.shortcuts.filter((shortcut) => shortcut.id !== id);
      return ok({ shortcuts: state.hid.shortcuts });
    }),
    http.get(`${origin}/api/hid/shortcut/leader-key`, () => ok({ key: state.hid.leaderKey })),
    http.post(`${origin}/api/hid/shortcut/leader-key`, async ({ request }) => {
      const key = readString(await readJsonObject(request), 'key');
      if (key) state.hid.leaderKey = key;
      return ok({ key: state.hid.leaderKey });
    }),

    http.get(`${origin}/api/network/static-ip`, () => ok(state.vm.staticIp)),
    http.post(`${origin}/api/network/static-ip`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const ip = readString(body, 'ip');
      if (enabled !== null) state.vm.staticIp.enabled = enabled;
      if (ip !== null) state.vm.staticIp.ip = ip;
      return ok(state.vm.staticIp);
    }),
    http.get(`${origin}/api/vm/usb-id`, () => ok(state.vm.usbId)),
    http.post(`${origin}/api/vm/usb-id`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const vid = readString(body, 'vid');
      const pid = readString(body, 'pid');
      const bcdDevice = readString(body, 'bcdDevice');
      const serialnumber = readString(body, 'serialnumber');
      const manufacturer = readString(body, 'manufacturer');
      const product = readString(body, 'product');
      const configurationName = readString(body, 'configurationName');
      if (enabled !== null) state.vm.usbId.enabled = enabled;
      if (vid !== null) state.vm.usbId.vid = vid;
      if (pid !== null) state.vm.usbId.pid = pid;
      if (bcdDevice !== null) state.vm.usbId.bcdDevice = bcdDevice;
      if (serialnumber !== null) state.vm.usbId.serialnumber = serialnumber;
      if (manufacturer !== null) state.vm.usbId.manufacturer = manufacturer;
      if (product !== null) state.vm.usbId.product = product;
      if (configurationName !== null) state.vm.usbId.configurationName = configurationName;
      if (enabled === false) {
        state.vm.usbId.vid = '';
        state.vm.usbId.pid = '';
        state.vm.usbId.bcdDevice = '';
        state.vm.usbId.serialnumber = '';
        state.vm.usbId.manufacturer = '';
        state.vm.usbId.product = '';
        state.vm.usbId.configurationName = '';
      }
      return ok(state.vm.usbId);
    }),
    http.get(`${origin}/api/vm/usb-network`, () => ok(state.vm.usbNetwork)),
    http.post(`${origin}/api/vm/usb-network`, async ({ request }) => {
      const body = await readJsonObject(request);
      const mode = readString(body, 'mode');
      const ipv4 = readString(body, 'ipv4');
      const forwarding = readBoolean(body, 'forwarding');
      if (mode === 'server' || mode === 'client') state.vm.usbNetwork.mode = mode;
      if (ipv4 !== null) state.vm.usbNetwork.ipv4 = ipv4;
      if (forwarding !== null) state.vm.usbNetwork.forwarding = forwarding;
      if (state.vm.usbNetwork.mode === 'client') {
        state.vm.usbNetwork.ipv4 = '';
        state.vm.usbNetwork.forwarding = false;
      }
      return ok(state.vm.usbNetwork);
    }),
    http.get(`${origin}/api/vm/disk-inquiry`, () => ok(state.vm.diskInquiry)),
    http.post(`${origin}/api/vm/disk-inquiry`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const value = readString(body, 'value');
      if (enabled !== null) state.vm.diskInquiry.enabled = enabled;
      if (value !== null) state.vm.diskInquiry.value = value;
      if (enabled === false) state.vm.diskInquiry.value = '';
      return ok(state.vm.diskInquiry);
    }),
    http.get(`${origin}/api/vm/disk-inquiry-cdrom`, () => ok(state.vm.diskInquiryCdrom)),
    http.post(`${origin}/api/vm/disk-inquiry-cdrom`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const value = readString(body, 'value');
      if (enabled !== null) state.vm.diskInquiryCdrom.enabled = enabled;
      if (value !== null) state.vm.diskInquiryCdrom.value = value;
      if (enabled === false) state.vm.diskInquiryCdrom.value = '';
      return ok(state.vm.diskInquiryCdrom);
    }),
    http.get(`${origin}/api/vm/uac2-function-name`, () => ok(state.vm.uac2FunctionName)),
    http.post(`${origin}/api/vm/uac2-function-name`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const value = readString(body, 'value');
      if (enabled !== null) state.vm.uac2FunctionName.enabled = enabled;
      if (value !== null) state.vm.uac2FunctionName.value = value;
      if (enabled === false) state.vm.uac2FunctionName.value = '';
      return ok(state.vm.uac2FunctionName);
    }),
    http.get(`${origin}/api/vm/webcam-function-name`, () => ok(state.vm.webcamFunctionName)),
    http.post(`${origin}/api/vm/webcam-function-name`, async ({ request }) => {
      const body = await readJsonObject(request);
      const enabled = readBoolean(body, 'enabled');
      const value = readString(body, 'value');
      if (enabled !== null) state.vm.webcamFunctionName.enabled = enabled;
      if (value !== null) state.vm.webcamFunctionName.value = value;
      if (enabled === false) state.vm.webcamFunctionName.value = '';
      return ok(state.vm.webcamFunctionName);
    }),
    http.get(`${origin}/api/network/wifi`, () => ok(getWifiResponse(state))),
    http.get(`${origin}/api/network/wifi/scan`, ({ request }) => {
      if (state.network.wifi.apMode && !hasMockAPKey(request)) {
        return fail<ReturnType<typeof getWifiScanResponse>>(-4, 'unauthorized', {
          wifiList: []
        });
      }
      return ok(getWifiScanResponse(state));
    }),
    http.post(`${origin}/api/network/wifi/verify`, ({ request }) => {
      if (!hasMockAPKey(request)) return fail<{ verified: boolean }>(-4, 'unauthorized', { verified: false });
      return ok({ verified: true });
    }),
    http.post(`${origin}/api/network/wifi`, async ({ request }) => {
      if (!hasMockAPKey(request)) {
        return fail<ReturnType<typeof getWifiResponse>>(-4, 'unauthorized', getWifiResponse(state));
      }
      const body = await readJsonObject(request);
      const ssid = readString(body, 'ssid');
      if (ssid) {
        const selected = state.network.wifi.available.find((wifi) => wifi.ssid === ssid) ?? {
          ssid,
          bssid: '02:00:00:00:00:99',
          signal: -62,
          frequency: '2.4 GHz',
          security: 'WPA2'
        };
        state.network.wifi.current = selected;
        state.network.wifi.connected = true;
      }
      return ok(getWifiResponse(state));
    }),
    http.post(`${origin}/api/network/wifi/connect`, async ({ request }) => {
      const body = await readJsonObject(request);
      const ssid = readString(body, 'ssid');
      if (ssid) {
        const selected = state.network.wifi.available.find((wifi) => wifi.ssid === ssid) ?? {
          ssid,
          bssid: '02:00:00:00:00:99',
          signal: -62,
          frequency: '2.4 GHz',
          security: 'WPA2'
        };
        state.network.wifi.current = selected;
        state.network.wifi.connected = true;
      }
      return ok(getWifiResponse(state));
    }),
    http.post(`${origin}/api/network/wifi/disconnect`, () => {
      state.network.wifi.current = null;
      state.network.wifi.connected = false;
      return ok(getWifiResponse(state));
    }),
    http.post(`${origin}/api/network/wifi/preset`, async ({ request }) => {
      const body = await readJsonObject(request);
      const ssid = readString(body, 'ssid');
      if (!ssid) {
        return fail(-1, 'invalid arguments', null);
      }
      // The mock keeps only the SSID; the password is intentionally discarded.
      state.network.wifi.preset = ssid;
      return ok(null);
    }),
    http.get(`${origin}/api/network/wifi/preset`, () => ok({ ssid: state.network.wifi.preset })),
    http.post(`${origin}/api/network/wifi/preset/remove`, () => {
      if (!state.network.wifi.preset) {
        return fail(-1, 'remove wifi preset failed', null);
      }
      state.network.wifi.preset = '';
      return ok(null);
    }),
    http.post(`${origin}/api/network/wol`, () => ok({ sent: true })),
    http.get(`${origin}/api/network/wol/mac`, () => ok({ macs: state.network.wolMacs })),
    http.delete(`${origin}/api/network/wol/mac`, async ({ request }) => {
      const mac = readString(await readJsonObject(request), 'mac');
      if (mac) {
        // Mirrors DeleteMac: lines are "mac" or "mac name"; compare the MAC segment only.
        state.network.wolMacs = state.network.wolMacs.filter((line) => line.split(' ')[0] !== mac);
      }
      return ok(null);
    }),
    http.post(`${origin}/api/network/wol/mac/name`, async ({ request }) => {
      const body = await readJsonObject(request);
      const mac = readString(body, 'mac');
      const name = readString(body, 'name');
      if (!mac || !name) {
        return fail(-1, 'invalid arguments', null);
      }
      // Mirrors SetMacName: rewrite the matching line as "mac name"; an unknown MAC fails.
      const index = state.network.wolMacs.findIndex((line) => line.split(' ')[0] === mac);
      if (index === -1) {
        return fail(-3, 'write failed', null);
      }
      state.network.wolMacs[index] = `${mac} ${name}`;
      return ok(null);
    }),

    http.get(`${origin}/api/vm/device/virtual`, () => ok(getVirtualDeviceResponse(state))),
    http.post(`${origin}/api/vm/device/virtual`, async ({ request }) => {
      const body = await readJsonObject(request);
      const device = readString(body, 'device');
      if (device === 'webcam') {
        const mode = readString(body, 'mode');
        if (mode !== 'on' && mode !== 'off') return fail<ReturnType<typeof getVirtualDeviceResponse>>(-1, 'Invalid webcam mode');
        state.virtualDevice.isWebcamEnabled = mode === 'on';
      } else if (device === 'network') {
        state.virtualDevice.isNetworkEnabled = !state.virtualDevice.isNetworkEnabled;
      } else if (device === 'disk') {
        if (!state.virtualDevice.isSdCardExist) {
          const diskSizeMiB = readNumber(body, 'diskSizeMiB');
          if (
            diskSizeMiB === null ||
            diskSizeMiB < state.virtualDevice.diskMinSizeMiB ||
            diskSizeMiB > state.virtualDevice.diskMaxSizeMiB
          ) {
            return fail<ReturnType<typeof getVirtualDeviceResponse>>(
              -1,
              'invalid virtual disk size'
            );
          }
          state.virtualDevice.isSdCardExist = true;
        }
        state.virtualDevice.isDiskEnabled = !state.virtualDevice.isDiskEnabled;
      } else if (device === 'audio') {
        const mode = readString(body, 'mode');
        if (mode !== 'off' && mode !== 'mic' && mode !== 'duplex') {
          return fail<ReturnType<typeof getVirtualDeviceResponse>>(
            -1,
            'invalid virtual audio mode'
          );
        }
        state.virtualDevice.isMicEnabled = mode !== 'off';
        state.virtualDevice.isSpeakerEnabled = mode === 'duplex';
      } else {
        return fail<ReturnType<typeof getVirtualDeviceResponse>>(-1, 'invalid virtual device');
      }
      return ok(getVirtualDeviceResponse(state));
    }),
    http.post(`${origin}/api/vm/device/virtual/refresh`, () => ok(getVirtualDeviceResponse(state))),
    http.post(`${origin}/api/vm/device/virtual/disk/disable`, () => {
      if (state.virtualDevice.isDiskEnabled) {
        return fail<ReturnType<typeof getVirtualDeviceResponse>>(
          -2,
          'disconnect virtual disk first'
        );
      }
      state.virtualDevice.isSdCardExist = false;
      return ok(getVirtualDeviceResponse(state));
    }),
    http.get(`${origin}/api/storage/image/mounted`, () =>
      ok({
        mounted: !!state.virtualDevice.mountedImage,
        file: state.virtualDevice.mountedImage,
        cdrom: state.virtualDevice.mountedCdrom,
        readOnly: state.virtualDevice.mountedReadOnly
      })
    ),
    http.get(`${origin}/api/storage/image`, () =>
      ok({ files: state.virtualDevice.imageFiles })
    ),
    http.post(`${origin}/api/storage/image/mount`, async ({ request }) => {
      const body = await readJsonObject(request);
      const file = readString(body, 'file');
      state.virtualDevice.mountedImage = file ?? '';
      state.virtualDevice.mountedCdrom = body.cdrom === true;
      state.virtualDevice.mountedReadOnly = body.cdrom === true || body.readOnly === true;
      return ok({ file: state.virtualDevice.mountedImage });
    }),
    http.post(`${origin}/api/storage/image/delete`, async ({ request }) => {
      const body = await readJsonObject(request);
      const challenge = readString(body, 'challenge');
      if (state.auth.twoFactorEnabled && state.auth.twoFactorProtection.deleteImage) {
        if (!state.auth.deleteImageChallenge || challenge !== state.auth.deleteImageChallenge) {
          return fail(-3, 'sensitive action verification required', null);
        }
        state.auth.deleteImageChallenge = null;
      }
      const file = readString(body, 'file');
      if (file) {
        state.virtualDevice.imageFiles = state.virtualDevice.imageFiles.filter(
          (item) => item !== file
        );
      }
      return ok(null);
    }),
    http.post(`${origin}/api/storage/image/upload`, () => {
      const file = 'mock-image.iso';
      if (!state.virtualDevice.imageFiles.includes(file)) {
        state.virtualDevice.imageFiles.push(file);
      }
      return ok({ file });
    }),
    http.post(`${origin}/api/storage/image/upload/cancel`, () => ok({ status: 'cancelled' })),
    http.post(`${origin}/api/storage/download/image`, () => ok({ started: true })),
    http.post(`${origin}/api/storage/download/image/pause`, () => ok({ status: 'paused' })),
    http.post(`${origin}/api/storage/download/image/resume`, () => ok({ status: 'in_progress' })),
    http.post(`${origin}/api/storage/download/image/cancel`, () => ok({ status: 'idle' })),
    http.get(`${origin}/api/storage/download/image/status`, () => ok({ status: 'idle' })),
    http.get(`${origin}/api/storage/download/image/enabled`, () => ok({ enabled: true })),
    http.get(`${origin}/api/vm/portal`, () =>
      ok({
        configured: false,
        version: 1,
        mode: 'auto',
        manualRegions: [],
        selectedRegionId: '',
        resolutions: [],
        selectedResolution: ''
      })
    ),
    http.post(`${origin}/api/vm/portal`, () => ok(null)),
    // MCP reuses the reported area, so mocked pages must accept the report.
    http.post(`${origin}/api/vm/portal/area`, () => ok(null))
  ];
}
