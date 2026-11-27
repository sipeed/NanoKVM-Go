import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { after, before, test } from 'node:test';
import { setImmediate } from 'node:timers/promises';
import { createStore } from 'jotai/vanilla';
import { http, HttpResponse } from 'msw';
import { setupServer } from 'msw/node';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
const origin = 'http://localhost:3001';
const alternateOrigin = 'http://127.0.0.1:4173';
const aiSettingsRoot = path.join(webRoot, 'src/pages/desktop/menu/settings/ai');
const aiSettingsSource = fs.readFileSync(path.join(aiSettingsRoot, 'index.tsx'), 'utf8');
const mcpPanelSource = fs.readFileSync(path.join(aiSettingsRoot, 'mcp.tsx'), 'utf8');
const memoryFabricPanelSource = fs.readFileSync(
  path.join(aiSettingsRoot, 'memory-fabric.tsx'),
  'utf8'
);
const portalMenuSource = fs.readFileSync(
  path.join(webRoot, 'src/pages/desktop/menu/screen/cropping.tsx'),
  'utf8'
);
const submenuCounterSource = fs.readFileSync(
  path.join(webRoot, 'src/hooks/useSubmenuOpenCounter.ts'),
  'utf8'
);
const appSource = fs.readFileSync(path.join(webRoot, 'src/app.tsx'), 'utf8');
const layersSource = fs.readFileSync(path.join(webRoot, 'src/lib/layers.ts'), 'utf8');
const menuItemSource = fs.readFileSync(path.join(webRoot, 'src/components/menu-item.tsx'), 'utf8');
const layeringConsumerSources = [
  'src/components/menu-item.tsx',
  'src/pages/desktop/menu/index.tsx',
  'src/pages/desktop/menu/screen/cropping.tsx',
  'src/pages/desktop/message.tsx',
  'src/pages/desktop/virtual-keyboard/index.tsx',
  'src/pages/desktop/keyboard/recorder.tsx',
  'src/pages/desktop/mouse/calibration-overlay.tsx',
  'src/pages/desktop/screen/portal-selection-overlay.tsx',
  'src/pages/desktop/menu/settings/ai/screen-timelapse.tsx',
  'src/hooks/useProtectionChallengeGate.tsx',
  'src/pages/desktop/menu/settings/remote-network/components/tailscale/uninstall.tsx'
].map((file) => [file, fs.readFileSync(path.join(webRoot, file), 'utf8')]);
let vite;
let server;
let modules;

async function fetchJson(url, init) {
  const headers =
    init?.body instanceof FormData
      ? init.headers
      : { 'content-type': 'application/json', ...(init?.headers ?? {}) };
  const response = await fetch(url, {
    headers,
    ...init
  });
  return response.json();
}

function installHandlers(baseUrl = origin) {
  const state = modules.createMockState();
  const hubs = modules.createMockHubs();
  server.resetHandlers(...modules.createHttpHandlers(baseUrl, state, hubs));
  return { state, hubs };
}

before(async () => {
  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false, ws: false }
  });

  const [runtimeMode, settings, target, state, hubs, handlers, memoryFabricAccess] =
    await Promise.all([
      vite.ssrLoadModule('/src/lib/runtime-mode.ts'),
      vite.ssrLoadModule('/src/jotai/settings.ts'),
      vite.ssrLoadModule('/src/mocks/target.ts'),
      vite.ssrLoadModule('/src/mocks/state.ts'),
      vite.ssrLoadModule('/src/mocks/hubs.ts'),
      vite.ssrLoadModule('/src/mocks/handlers/index.ts'),
      vite.ssrLoadModule('/src/pages/desktop/menu/settings/ai/memory-fabric-remote-access.ts')
    ]);

  modules = {
    ...runtimeMode,
    ...settings,
    ...target,
    ...state,
    ...hubs,
    ...handlers,
    ...memoryFabricAccess
  };
  server = setupServer();
  server.listen({ onUnhandledRequest: 'error' });
});

after(async () => {
  server.close();
  await vite.close();
});

test('storage media keeps shared files readable and blocks writes until eject', async () => {
  installHandlers();
  let response = await fetchJson(`${origin}/api/storage/media/state`);
  assert.equal(response.data.state, 'partition');
  assert.equal(response.data.localWritable, false);
  response = await fetchJson(`${origin}/api/storage/media/files?type=partition&path=/`);
  assert.ok(response.data.entries.some((entry) => entry.name === 'hello.txt'));
  const upload = () => {
    const form = new FormData();
    form.append('file', new File(['new data'], 'new.txt'));
    return fetchJson(`${origin}/api/storage/media/files/upload?type=partition&path=/`, {
      method: 'POST',
      body: form
    });
  };
  assert.equal((await upload()).code, -12);
  assert.equal(
    (await fetchJson(`${origin}/api/storage/media/eject`, { method: 'POST', body: '{}' })).code,
    0
  );
  response = await fetchJson(`${origin}/api/storage/media/state`);
  assert.equal(response.data.state, 'none');
  assert.equal(response.data.localWritable, true);
  assert.equal((await upload()).code, 0);
  response = await fetchJson(`${origin}/api/storage/media/files?type=partition&path=/`);
  assert.ok(response.data.entries.some((entry) => entry.name === 'new.txt'));
  assert.equal((await upload()).code, -2);
});

test('shared file chunks remain hidden until complete and release reservations on cancel', async () => {
  const { state } = installHandlers();
  state.virtualDevice.isDiskEnabled = false;
  const chunkSize = 8 * 1024 * 1024;
  const uploadId = '0123456789abcdef0123456789abcdef';
  const params = new URLSearchParams({
    type: 'partition',
    path: '/',
    uploadId,
    filename: 'chunked.zip',
    size: String(chunkSize + 3),
    chunkSize: String(chunkSize),
    totalChunks: '2'
  });
  const chunk = (index, body) =>
    fetchJson(`${origin}/api/storage/media/files/upload?${params}&chunkIndex=${index}`, {
      method: 'POST',
      headers: { 'content-type': 'application/octet-stream' },
      body
    });
  const list = () => fetchJson(`${origin}/api/storage/media/files?type=partition&path=/`);
  let response = await chunk(0, new Uint8Array(chunkSize));
  assert.equal(response.code, 0);
  assert.equal(response.data.complete, false);
  assert.equal(response.data.written, chunkSize);
  assert.ok(!(await list()).data.entries.some((entry) => entry.name === 'chunked.zip'));
  const mount = () =>
    fetchJson(`${origin}/api/storage/media/mount`, {
      method: 'POST',
      body: JSON.stringify({ type: 'partition', cdrom: false })
    });
  assert.equal((await mount()).code, -12);
  assert.equal((await chunk(0, new Uint8Array(chunkSize))).code, -1);
  response = await chunk(1, new Uint8Array(3));
  assert.equal(response.code, 0);
  assert.equal(response.data.complete, true);
  assert.equal(
    (await list()).data.entries.find((entry) => entry.name === 'chunked.zip').size,
    chunkSize + 3
  );

  params.set('filename', 'cancelled.zip');
  assert.equal((await chunk(0, new Uint8Array(chunkSize))).code, 0);
  response = await fetchJson(`${origin}/api/storage/media/files/upload/cancel`, {
    method: 'POST',
    body: JSON.stringify({ uploadId })
  });
  assert.equal(response.code, 0);
  assert.equal((await chunk(1, new Uint8Array(3))).code, -1);
  assert.ok(!(await list()).data.entries.some((entry) => entry.name === 'cancelled.zip'));
  assert.equal((await mount()).code, 0);
});

test('file sharing stays unavailable until the virtual disk is initialized', async () => {
  const { state } = installHandlers();
  state.virtualDevice.isSdCardExist = false;
  state.virtualDevice.isDiskEnabled = false;
  let status = await fetchJson(`${origin}/api/storage/media/state`);
  assert.equal(status.data.sharedDiskInitialized, false);
  let media = await fetchJson(`${origin}/api/storage/media`);
  assert.ok(media.data.media.some((item) => item.type === 'image'));
  assert.ok(!media.data.media.some((item) => item.type === 'partition'));

  const upload = () => {
    const form = new FormData();
    form.append('file', new File(['new data'], 'new.txt'));
    return fetchJson(`${origin}/api/storage/media/files/upload?type=partition&path=/`, {
      method: 'POST',
      body: form
    });
  };
  assert.equal((await upload()).code, -10);
  for (const action of ['mkdir', 'delete']) {
    const response = await fetchJson(`${origin}/api/storage/media/files/${action}`, {
      method: 'POST',
      body: JSON.stringify({ type: 'partition', path: 'hello.txt' })
    });
    assert.equal(response.code, -10);
  }
  assert.equal(
    (await fetchJson(`${origin}/api/storage/media/files?type=partition&path=/`)).code,
    -10
  );
  assert.equal(
    (await fetchJson(`${origin}/api/storage/media/files/download?type=partition&path=hello.txt`))
      .code,
    -10
  );

  state.virtualDevice.isSdCardExist = true;
  status = await fetchJson(`${origin}/api/storage/media/state`);
  assert.equal(status.data.sharedDiskInitialized, true);
  media = await fetchJson(`${origin}/api/storage/media`);
  assert.ok(media.data.media.some((item) => item.type === 'partition'));
  assert.equal((await upload()).code, 0);
  const listing = await fetchJson(`${origin}/api/storage/media/files?type=partition&path=/`);
  assert.ok(listing.data.entries.some((entry) => entry.name === 'new.txt'));
  assert.ok(listing.data.entries.some((entry) => entry.name === 'hello.txt'));
});

test('storage media preserves deletion protection and rejects invalid optical mounts', async () => {
  const { state } = installHandlers();
  await fetchJson(`${origin}/api/storage/media/eject`, { method: 'POST', body: '{}' });
  state.auth.twoFactorEnabled = true;
  state.auth.twoFactorProtection.deleteImage = true;
  const remove = (challenge) =>
    fetchJson(`${origin}/api/storage/media/files/delete`, {
      method: 'POST',
      body: JSON.stringify({ type: 'partition', path: 'hello.txt', challenge })
    });
  assert.equal((await remove()).code, -3);
  state.auth.deleteImageChallenge = 'storage-challenge';
  assert.equal((await remove('storage-challenge')).code, 0);
  const listing = await fetchJson(`${origin}/api/storage/media/files?type=partition&path=/`);
  assert.ok(!listing.data.entries.some((entry) => entry.name === 'hello.txt'));
  const mount = await fetchJson(`${origin}/api/storage/media/mount`, {
    method: 'POST',
    body: JSON.stringify({ type: 'image', rel: 'rescue.img', cdrom: true })
  });
  assert.equal(mount.code, -15);
});

test('storage media reports disk access mode and keeps attached images unavailable locally', async () => {
  installHandlers();
  for (const options of [
    { cdrom: false, readOnly: false },
    { cdrom: false, readOnly: true },
    { cdrom: false },
    { cdrom: true, readOnly: false }
  ]) {
    const rel = options.cdrom ? 'rescue.iso' : 'rescue.img';
    const mounted = await fetchJson(`${origin}/api/storage/media/mount`, {
      method: 'POST',
      body: JSON.stringify({ type: 'image', rel, ...options })
    });
    assert.equal(mounted.code, 0);
    const status = await fetchJson(`${origin}/api/storage/media/state`);
    assert.equal(status.data.cdrom, options.cdrom);
    assert.equal(status.data.readOnly, options.cdrom || options.readOnly !== false);
    const files = await fetchJson(`${origin}/api/storage/media/files?type=image&rel=${rel}&path=/`);
    assert.equal(files.code, -12);
    const sharedDownload = await fetchJson(
      `${origin}/api/storage/media/files/download?type=partition&path=/${rel.toUpperCase()}`
    );
    assert.equal(sharedDownload.code, -12);
    const imageDownload = await fetchJson(
      `${origin}/api/storage/media/download?type=image&rel=${rel}`
    );
    assert.equal(imageDownload.code, -12);
  }
  await fetchJson(`${origin}/api/storage/media/eject`, { method: 'POST', body: '{}' });
  const download = await fetch(
    `${origin}/api/storage/media/files/download?type=partition&path=/rescue.img`
  );
  assert.equal(download.headers.get('Content-Type'), 'application/octet-stream');
});

test('mocked development requires both Vite development and mocked mode', () => {
  assert.equal(modules.isMockedDevelopment('mocked', true), true);
  assert.equal(modules.isMockedDevelopment('development', true), false);
  assert.equal(modules.isMockedDevelopment('mocked', false), false);
  assert.equal(modules.isMockedDevelopment('production', false), false);
});

test('pointer mode mock exposes an authoritative HID2 profile and strict mutation', async () => {
  const { state, hubs } = installHandlers();

  const initial = await fetchJson(`${origin}/api/hid/pointer-mode`);
  assert.deepEqual(initial.data, {
    mode: 'absolute',
    singleEndpoint: true,
    switching: false,
    degraded: false,
    error: ''
  });

  const switched = await fetchJson(`${origin}/api/hid/pointer-mode`, {
    method: 'POST',
    body: JSON.stringify({ mode: 'digitizer-stylus' })
  });
  assert.equal(switched.code, 0);
  assert.equal(switched.data.mode, 'digitizer-stylus');
  assert.equal(state.hid.pointerMode.mode, 'digitizer-stylus');

  const invalid = await fetchJson(`${origin}/api/hid/pointer-mode`, {
    method: 'POST',
    body: JSON.stringify({ mode: 'absolute-android' })
  });
  assert.equal(invalid.code, -1);
  assert.equal((await fetchJson(`${origin}/api/hid/pointer-mode`)).data.mode, 'digitizer-stylus');

  hubs.dispose();
});

test('MCP and Memory Fabric mocks preserve all four independent switch combinations', async () => {
  const { hubs } = installHandlers();

  for (const [mcpEnabled, memoryFabricEnabled, expectedAccess] of [
    [false, false, 'disabled'],
    [true, false, 'disabled'],
    [false, true, 'requiresMcp'],
    [true, true, 'available']
  ]) {
    await fetchJson(`${origin}/api/ai/config`, {
      method: 'POST',
      body: JSON.stringify({ mcpEnabled })
    });
    await fetchJson(`${origin}/api/ai/memory-fabric`, {
      method: 'POST',
      body: JSON.stringify({ enabled: memoryFabricEnabled })
    });

    const mcp = await fetchJson(`${origin}/api/ai/config`);
    const memoryFabric = await fetchJson(`${origin}/api/ai/memory-fabric`);
    assert.equal(mcp.data.mcpEnabled, mcpEnabled);
    assert.equal(memoryFabric.data.enabled, memoryFabricEnabled);
    assert.equal(
      modules.getMemoryFabricRemoteAccess(mcpEnabled, memoryFabricEnabled),
      expectedAccess
    );
  }

  hubs.dispose();
  assert.equal('mcpApiKey' in (await fetchJson(`${origin}/api/ai/config`)).data, false);
});

test('AI settings keep one MCP endpoint and share gateway state with Memory Fabric', () => {
  assert.match(mcpPanelSource, /window\.location\.origin}\/api\/mcp/);
  // Key reveal modal (every viewport): description copy removed; failure hint stays inline.
  assert.doesNotMatch(mcpPanelSource, /settings\.ai\.apiKeyDesc/);
  // The key panel must never scroll: it wraps freely and grows with the key.
  assert.doesNotMatch(mcpPanelSource, /overflow-y-auto|max-h-\[40dvh\]/);
  // The last-line spacer and the docked button are load-bearing: removing them lets
  // the button escape the panel or cover key characters (measured via CDP geometry).
  assert.match(mcpPanelSource, /inline-block h-5 min-h-\[26px\] w-10 min-w-\[28px\]/);
  assert.match(mcpPanelSource, /absolute bottom-2 right-2/);
  // Without `relative` on the panel the absolute button docks to the modal and escapes.
  assert.match(mcpPanelSource, /relative rounded-lg/);
  // Icon-only MCP buttons must expose i18n accessible names (component-guidelines mandate).
  const countOccurrences = (pattern) => (mcpPanelSource.match(pattern) ?? []).length;
  assert.equal(countOccurrences(/aria-label=\{t\('settings\.ai\.copyApiKey'\)\}/g), 2);
  assert.equal(countOccurrences(/aria-label=\{t\('settings\.ai\.copyEndpoint'\)\}/g), 1);
  assert.equal(countOccurrences(/aria-label=\{t\('settings\.ai\.regenerate'\)\}/g), 1);
  // The eye button only opens the wrapping key modal on every viewport; the inline-reveal
  // hide state is gone, so showApiKey appears exactly once and the old branches are dead.
  assert.equal(countOccurrences(/aria-label=\{t\('settings\.ai\.showApiKey'\)\}/g), 1);
  assert.doesNotMatch(
    mcpPanelSource,
    /settings\.ai\.hideApiKey|isInlineKeyVisible|isMobilePortrait|EyeOffIcon|isKeyVisible|displayKey/
  );
  // Regeneration stacks the key modal on the closing confirm on every viewport.
  assert.match(mcpPanelSource, /window\.setTimeout\(\(\) => showKeyModal\(newKey\), 0\)/);
  // antd's `.ant-btn-icon > svg { color: inherit }` (0,2,1) beats a plain `.text-green-500`
  // (0,1,0), so the success tick needs the important modifier to actually render green.
  assert.equal(countOccurrences(/!text-green-500/g), 3);
  assert.doesNotMatch(mcpPanelSource, /className="text-green-500"/);
  assert.doesNotMatch(memoryFabricPanelSource, /8742|getMemoryFabricEndpoint|copyEndpoint/);
  assert.doesNotMatch(memoryFabricPanelSource, /navigator\.clipboard|settings\.ai\.endpoint/);
  assert.match(aiSettingsSource, /onEnabledChange=\{setMCPEnabled}/);
  assert.match(aiSettingsSource, /mcpEnabled=\{mcpEnabled}/);
});

test('Cropping uses Screen’s submenu and dismisses the toolbar before area selection', () => {
  assert.match(portalMenuSource, /<MenuSubmenu/);
  assert.match(portalMenuSource, /onOpenChange: setOpen/);
  assert.match(portalMenuSource, /function startSelection\(\) \{\s*dismissMenus\(\);\s*setSelecting\(true\)/);

  assert.match(submenuCounterSource, /if \(isOpenRef\.current === open\) \{\s*return;/);
  assert.match(
    submenuCounterSource,
    /setSubmenuOpenCount\(\(count\) => Math\.max\(0, count - 1\)\)/
  );
  assert.match(submenuCounterSource, /return \(\) => \{[\s\S]*?hasOpenCountRef\.current = false/);
});

test('global layering scale is centralized in lib/layers.ts', () => {
  // Canonical constants keep the documented values.
  assert.match(layersSource, /export const LAYER_VIDEO_STATUS_MESSAGE = 998;/);
  assert.match(layersSource, /export const LAYER_DESKTOP_MENU_POPOVER = 999;/);
  assert.match(layersSource, /export const LAYER_VIRTUAL_KEYBOARD = 999;/);
  assert.match(layersSource, /export const LAYER_MENU_RAIL = 1000;/);
  assert.match(layersSource, /export const LAYER_MOBILE_MENU_POPOVER = 1100;/);
  assert.match(layersSource, /export const LAYER_OVERLAY_BACKDROP = 1199;/);
  assert.match(layersSource, /export const LAYER_OVERLAY = 1200;/);
  assert.match(layersSource, /export const LAYER_OVERLAY_HINT = 1201;/);
  assert.match(layersSource, /export const LAYER_ONBOARDING_BACKDROP = 1202;/);
  assert.match(layersSource, /export const LAYER_ONBOARDING = 1210;/);
  assert.match(layersSource, /export const LAYER_RECORDER = 9999;/);

  // Static message/notification holders receive the dark theme and the 1100 token.
  assert.match(appSource, /ConfigProvider\.config\(\{ theme: themeConfig \}\)/);
  assert.match(appSource, /Message: \{[\s\S]*?zIndexPopup: 1100/);
  assert.match(appSource, /Notification: \{[\s\S]*?zIndexPopup: 1100/);

  // Menu popover layers come from lib/layers.ts; menu-item re-exports them.
  assert.match(menuItemSource, /from '@\/lib\/layers\.ts'/);
  assert.match(
    menuItemSource,
    /export const desktopMenuPopoverZIndex = LAYER_DESKTOP_MENU_POPOVER;/
  );
  assert.match(menuItemSource, /export const mobileMenuPopoverZIndex = LAYER_MOBILE_MENU_POPOVER;/);

  // Consumers reference constants instead of bare numbers (screen-local z-10/z-20
  // utilities and app.tsx token objects are intentionally out of this pattern).
  const bareLayerValue = /z-\[\d+\]|zIndex=\{\d+\}/;
  for (const [file, source] of layeringConsumerSources) {
    assert.doesNotMatch(source, bareLayerValue, `${file} must not contain bare z-index values`);
  }
});

test('all settings targets parse and seed coherent Jotai state', () => {
  for (const tab of modules.settingTabIDs) {
    const hash = `/#/?mockTarget=settings:${tab}`;
    assert.deepEqual(modules.parseMockTarget(hash), { type: 'settings', tab });

    const store = createStore();
    const result = modules.applyMockTarget(hash, store, () => assert.fail('unexpected warning'));
    assert.deepEqual(result, { type: 'settings', tab });
    assert.equal(store.get(modules.settingTabAtom), tab);
    assert.equal(store.get(modules.isSettingsOpenAtom), true);
    assert.equal(store.get(modules.isSettingsDetailOpenAtom), true);
    assert.equal(store.get(modules.submenuOpenCountAtom), 1);
  }
});

test('the renamed remote network tab still accepts the legacy tailscale target', () => {
  const store = createStore();

  assert.deepEqual(modules.parseMockTarget('/#/?mockTarget=settings:tailscale'), {
    type: 'settings',
    tab: 'remoteNetwork'
  });
  assert.deepEqual(
    modules.applyMockTarget('/#/?mockTarget=settings:tailscale', store, () =>
      assert.fail('unexpected warning')
    ),
    { type: 'settings', tab: 'remoteNetwork' }
  );
  assert.equal(store.get(modules.settingTabAtom), 'remoteNetwork');
});

test('dev-only mockFeature selectors are parsed strictly', () => {
  assert.deepEqual(modules.parseMockFeatures('/#/?mockFeature=tailscale_cn:off'), {
    tailscaleCnEnabled: false
  });
  assert.deepEqual(modules.parseMockFeatures('/#/?mockFeature=tailscale_cn:on'), {
    tailscaleCnEnabled: true
  });
  assert.deepEqual(modules.parseMockFeatures('/#/?mockFeature=tailscale_cn:maybe'), {});
  assert.deepEqual(modules.parseMockFeatures('/#/?mockFeature=unknown:off'), {});
  assert.deepEqual(modules.parseMockFeatures('/#/?mockFeature'), {});
  assert.deepEqual(modules.parseMockFeatures('/#/'), {});
  assert.deepEqual(
    modules.parseMockFeatures('/#/?mockTarget=settings:remoteNetwork&mockFeature=tailscale_cn:off'),
    { tailscaleCnEnabled: false }
  );
});

test('invalid target is ignored and warns only once', () => {
  const store = createStore();
  const warnings = [];
  modules.resetMockTargetWarningForTests();

  assert.equal(
    modules.applyMockTarget('/#/?mockTarget=settings:unknown', store, (msg) => warnings.push(msg)),
    null
  );
  assert.equal(
    modules.applyMockTarget('/#/?mockTarget=other:value', store, (msg) => warnings.push(msg)),
    null
  );
  assert.equal(warnings.length, 1);
  assert.equal(store.get(modules.isSettingsOpenAtom), false);
  assert.equal(store.get(modules.submenuOpenCountAtom), 0);
});

test('virtual audio mock applies all three explicit modes', async () => {
  installHandlers();

  for (const [mode, isMicEnabled, isSpeakerEnabled] of [
    ['off', false, false],
    ['mic', true, false],
    ['duplex', true, true]
  ]) {
    const response = await fetchJson(`${origin}/api/vm/device/virtual`, {
      method: 'POST',
      body: JSON.stringify({ device: 'audio', mode })
    });

    assert.equal(response.code, 0);
    assert.equal(response.data.isMicEnabled, isMicEnabled);
    assert.equal(response.data.isSpeakerEnabled, isSpeakerEnabled);
    assert.equal(response.data.audioMode, mode);
  }

  const invalid = await fetchJson(`${origin}/api/vm/device/virtual`, {
    method: 'POST',
    body: JSON.stringify({ device: 'audio', mode: 'invalid' })
  });
  assert.equal(invalid.code, -1);

  for (const device of ['mic', 'speaker']) {
    const legacy = await fetchJson(`${origin}/api/vm/device/virtual`, {
      method: 'POST',
      body: JSON.stringify({ device })
    });
    assert.equal(legacy.code, -1);
  }
});

test('onboarding mock persists the completed version with stateful read-back', async () => {
  installHandlers();

  const initial = await fetchJson(`${origin}/api/vm/onboarding`);
  assert.equal(initial.code, 0);
  assert.equal(initial.data.completedVersion, '');

  const completed = await fetchJson(`${origin}/api/vm/onboarding`, {
    method: 'POST',
    body: JSON.stringify({ completedVersion: 'v1' })
  });
  assert.equal(completed.code, 0);

  const after = await fetchJson(`${origin}/api/vm/onboarding`);
  assert.equal(after.data.completedVersion, 'v1');

  for (const completedVersion of ['', 'INVALID!!', 'a'.repeat(33)]) {
    const rejected = await fetchJson(`${origin}/api/vm/onboarding`, {
      method: 'POST',
      body: JSON.stringify({ completedVersion })
    });
    assert.equal(rejected.code, -1);
  }

  const unchanged = await fetchJson(`${origin}/api/vm/onboarding`);
  assert.equal(unchanged.data.completedVersion, 'v1');
});

test('onboarding mock keeps the mobile completion namespace independent', async () => {
  installHandlers();

  // GET always returns both fields; a first-run device reads back empty
  const initial = await fetchJson(`${origin}/api/vm/onboarding`);
  assert.equal(initial.code, 0);
  assert.equal(initial.data.completedVersion, '');
  assert.equal(initial.data.mobileCompletedVersion, '');

  // a mobile-only POST round-trips without touching the desktop namespace
  const completed = await fetchJson(`${origin}/api/vm/onboarding`, {
    method: 'POST',
    body: JSON.stringify({ mobileCompletedVersion: 'mobile-v1' })
  });
  assert.equal(completed.code, 0);

  const after = await fetchJson(`${origin}/api/vm/onboarding`);
  assert.equal(after.data.mobileCompletedVersion, 'mobile-v1');
  assert.equal(after.data.completedVersion, '');

  // both fields empty/missing is rejected and writes nothing
  const empty = await fetchJson(`${origin}/api/vm/onboarding`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(empty.code, -1);

  // an invalid mobile value is rejected and leaves both namespaces untouched
  for (const mobileCompletedVersion of ['', 'INVALID!!', 'a'.repeat(33)]) {
    const rejected = await fetchJson(`${origin}/api/vm/onboarding`, {
      method: 'POST',
      body: JSON.stringify({ mobileCompletedVersion })
    });
    assert.equal(rejected.code, -1);
  }

  const unchanged = await fetchJson(`${origin}/api/vm/onboarding`);
  assert.equal(unchanged.data.mobileCompletedVersion, 'mobile-v1');
  assert.equal(unchanged.data.completedVersion, '');
});

test('AP Wi-Fi provisioning mock scans deterministically and never retains passwords', async () => {
  const { state, hubs } = installHandlers();
  state.network.wifi.apMode = true;

  const rejectedVerification = await fetchJson(`${origin}/api/network/wifi/verify`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(rejectedVerification.code, -4);

  const verified = await fetchJson(`${origin}/api/network/wifi/verify`, {
    method: 'POST',
    headers: { 'X-AP-Key': 'mock' },
    body: JSON.stringify({})
  });
  assert.equal(verified.code, 0);

  const rejectedScan = await fetchJson(`${origin}/api/network/wifi/scan`);
  assert.equal(rejectedScan.code, -4);

  const scanned = await fetchJson(`${origin}/api/network/wifi/scan`, {
    headers: { 'X-AP-Key': 'mock' }
  });
  assert.equal(scanned.code, 0);
  assert.deepEqual(scanned.data.wifiList, state.network.wifi.available);

  const password = 'mock-ephemeral-password-only';
  const connected = await fetchJson(`${origin}/api/network/wifi`, {
    method: 'POST',
    headers: { 'X-AP-Key': 'mock' },
    body: JSON.stringify({ ssid: 'Guest Mock Network', password })
  });
  assert.equal(connected.code, 0);
  assert.equal(state.network.wifi.current?.ssid, 'Guest Mock Network');
  assert.equal(JSON.stringify(state).includes(password), false);

  hubs.dispose();
});

test('USB network mock preserves server configuration and clears it in client mode', async () => {
  const { hubs } = installHandlers();

  const initial = await fetchJson(`${origin}/api/vm/usb-network`);
  assert.deepEqual(initial, {
    code: 0,
    msg: 'ok',
    data: { mode: 'server', ipv4: '', forwarding: false }
  });

  const manualServer = await fetchJson(`${origin}/api/vm/usb-network`, {
    method: 'POST',
    body: JSON.stringify({
      mode: 'server',
      ipv4: '10.10.10.1/24',
      forwarding: true
    })
  });
  assert.deepEqual(manualServer.data, {
    mode: 'server',
    ipv4: '10.10.10.1/24',
    forwarding: true
  });

  const savedServer = await fetchJson(`${origin}/api/vm/usb-network`);
  assert.deepEqual(savedServer.data, manualServer.data);

  const client = await fetchJson(`${origin}/api/vm/usb-network`, {
    method: 'POST',
    body: JSON.stringify({
      mode: 'client',
      ipv4: '192.168.10.1/24',
      forwarding: true
    })
  });
  assert.deepEqual(client.data, { mode: 'client', ipv4: '', forwarding: false });

  const savedClient = await fetchJson(`${origin}/api/vm/usb-network`);
  assert.deepEqual(savedClient.data, client.data);

  hubs.dispose();
});

test('WoL MAC mock keeps the backend string-list contract across mutations', async () => {
  const { hubs } = installHandlers();

  const initial = await fetchJson(`${origin}/api/network/wol/mac`);
  assert.equal(initial.code, 0);
  assert.ok(Array.isArray(initial.data.macs));
  assert.ok(
    initial.data.macs.every((line) => typeof line === 'string'),
    'every wol mac entry must be a "mac" or "mac name" string line'
  );
  assert.deepEqual(initial.data.macs, ['02:00:00:00:00:10 Mock workstation']);

  const renamed = await fetchJson(`${origin}/api/network/wol/mac/name`, {
    method: 'POST',
    body: JSON.stringify({ mac: '02:00:00:00:00:10', name: 'Renamed workstation' })
  });
  assert.equal(renamed.code, 0);
  assert.deepEqual((await fetchJson(`${origin}/api/network/wol/mac`)).data.macs, [
    '02:00:00:00:00:10 Renamed workstation'
  ]);

  const missing = await fetchJson(`${origin}/api/network/wol/mac/name`, {
    method: 'POST',
    body: JSON.stringify({ mac: '02:00:00:00:00:99', name: 'Ghost' })
  });
  assert.equal(missing.code, -3);

  const removed = await fetchJson(`${origin}/api/network/wol/mac`, {
    method: 'DELETE',
    body: JSON.stringify({ mac: '02:00:00:00:00:10' })
  });
  assert.equal(removed.code, 0);
  const afterDelete = (await fetchJson(`${origin}/api/network/wol/mac`)).data.macs;
  assert.ok(!afterDelete.some((line) => line.split(' ')[0] === '02:00:00:00:00:10'));

  hubs.dispose();
});

test('FRP installs, retains secrets on edits and uninstall, and enforces action challenges', async () => {
  const { state, hubs } = installHandlers();
  const get = (endpoint) => fetchJson(`${origin}/api/extensions/frp/${endpoint}`);
  const post = (endpoint, body = {}) =>
    fetchJson(`${origin}/api/extensions/frp/${endpoint}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  assert.equal((await get('status')).data.state, 'notInstalled');
  assert.equal((await post('install')).code, 0);
  assert.equal((await get('status')).data.state, 'notConfigured');
  const config = { serverAddr: 'frp.example.com', serverPort: 7000, remotePort: 8443 };
  assert.equal((await post('config', { ...config, token: 'mock-private-token' })).code, 0);
  assert.equal((await post('start')).code, 0);
  assert.equal((await get('status')).data.state, 'connected');
  assert.equal((await post('config', { ...config, remotePort: 8444 })).code, 0);
  const saved = await get('config');
  assert.equal(saved.data.tokenSet, true);
  assert.equal(JSON.stringify(saved).includes('mock-private-token'), false);
  assert.equal(state.frp.token, 'mock-private-token');
  state.frp.error = 'proxyFailed';
  assert.equal((await get('status')).data.state, 'error');
  await post('stop');
  assert.equal((await get('status')).data.state, 'stopped');
  await post('uninstall');
  await post('install');
  assert.equal((await get('status')).data.state, 'stopped');
  assert.equal((await get('config')).data.tokenSet, true);
  state.auth.twoFactorEnabled = true;
  state.auth.twoFactorProtection.frp = true;
  assert.equal((await post('start')).code, -3);
  const verified = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'frp_start', code: '000000' })
  });
  const challenge = verified.data.challenge;
  assert.equal((await post('uninstall', { challenge })).code, -3);
  assert.equal((await post('start', { challenge })).code, 0);
  assert.equal((await post('start', { challenge })).code, -3);
  hubs.dispose();
});

test('Cloudflare installs, retains secrets on edits and uninstall, and enforces action challenges', async () => {
  const { state, hubs } = installHandlers();
  const get = (endpoint) => fetchJson(`${origin}/api/extensions/cloudflare/${endpoint}`);
  const post = (endpoint, body = {}) =>
    fetchJson(`${origin}/api/extensions/cloudflare/${endpoint}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  assert.equal((await get('status')).data.state, 'notInstalled');
  assert.equal((await post('install')).code, 0);
  assert.equal((await get('status')).data.state, 'notConfigured');
  const config = { accessUrl: 'https://kvm.example.com' };
  const token = btoa(
    JSON.stringify({ a: 'account', t: 'd1838763-3549-430f-a54e-692caf9f1653', s: btoa('secret') })
  );
  assert.equal((await post('config', { ...config, token })).code, 0);
  assert.equal((await post('start')).code, 0);
  assert.equal((await get('status')).data.state, 'connected');
  assert.equal(
    (await post('config', { ...config, accessUrl: 'https://other.example.com' })).code,
    0
  );
  const saved = await get('config');
  assert.equal(saved.data.tokenSet, true);
  assert.equal(JSON.stringify(saved).includes(token), false);
  assert.equal(state.cloudflare.token, token);
  state.cloudflare.error = 'connectionFailed';
  assert.equal((await get('status')).data.state, 'error');
  await post('stop');
  assert.equal((await get('status')).data.state, 'stopped');
  await post('uninstall');
  await post('install');
  assert.equal((await get('status')).data.state, 'stopped');
  assert.equal((await get('config')).data.tokenSet, true);
  state.auth.twoFactorEnabled = true;
  state.auth.twoFactorProtection.cloudflare = true;
  assert.equal((await post('start')).code, -3);
  const verified = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'cloudflare_start', code: '000000' })
  });
  const challenge = verified.data.challenge;
  assert.equal((await post('uninstall', { challenge })).code, -3);
  assert.equal((await post('start', { challenge })).code, 0);
  assert.equal((await post('start', { challenge })).code, -3);
  hubs.dispose();
});

test('Quick Tunnel switches modes without losing named credentials and renews its temporary address', async () => {
  const { state, hubs } = installHandlers();
  const post = (endpoint, body = {}) =>
    fetchJson(`${origin}/api/extensions/cloudflare/${endpoint}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  const status = async () => (await fetchJson(`${origin}/api/extensions/cloudflare/status`)).data;
  try {
    assert.equal((await post('mode', { mode: 'quick' })).code, 0);
    assert.equal((await status()).state, 'notInstalled');
    assert.equal((await status()).mode, 'quick');
    await post('mode', { mode: 'named' });
    await post('install');
    Object.assign(state.cloudflare, {
      token: 'saved-named-token',
      accessUrl: 'https://kvm.example.com',
      enabled: true
    });
    assert.equal((await post('mode', { mode: 'invalid' })).code, -1);
    state.auth.twoFactorEnabled = true;
    state.auth.twoFactorProtection.cloudflare = true;
    assert.equal((await post('mode', { mode: 'quick' })).code, -3);
    assert.equal(state.cloudflare.enabled, true);
    const verification = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
      method: 'POST',
      body: JSON.stringify({ action: 'cloudflare_configure', code: '000000' })
    });
    const challenge = verification.data.challenge;
    assert.equal((await post('mode', { mode: 'quick', challenge })).code, 0);
    assert.equal((await post('mode', { mode: 'named', challenge })).code, -3);
    state.auth.twoFactorEnabled = false;
    assert.equal(state.cloudflare.token, 'saved-named-token');
    assert.equal(state.cloudflare.accessUrl, 'https://kvm.example.com');
    assert.equal((await status()).enabled, false);
    assert.equal((await status()).accessUrl, '');
    state.cloudflare.token = '';
    assert.equal((await post('start')).code, 0);
    const firstURL = (await status()).accessUrl;
    assert.match(firstURL, /^https:\/\/.+\.trycloudflare\.com$/);
    assert.equal((await post('restart')).code, 0);
    assert.notEqual((await status()).accessUrl, firstURL);
    await post('stop');
    assert.equal((await status()).accessUrl, '');
    await post('mode', { mode: 'named' });
    assert.equal((await status()).accessUrl, 'https://kvm.example.com');
    assert.equal((await post('start')).code, -1);
  } finally {
    hubs.dispose();
  }
});

test('NetBird browser authorization, cancel, lifecycle and 2FA stay independent', async () => {
  const { state, hubs } = installHandlers();
  const post = (action, body = {}) =>
    fetchJson(`${origin}/api/extensions/netbird/${action}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  const status = async () => (await fetchJson(`${origin}/api/extensions/netbird/status`)).data;
  try {
    assert.equal((await status()).state, 'notInstalled');
    assert.equal((await post('install')).code, 0);
    assert.equal((await status()).version, '0.80.0');
    assert.equal((await post('login')).msg, 'startRequired');
    assert.equal((await status()).enabled, false);
    assert.equal((await status()).state, 'stopped');
    assert.equal((await post('start')).code, 0);
    state.auth.twoFactorEnabled = true;
    state.auth.twoFactorProtection.netbird = true;
    assert.equal((await post('login')).code, -3);
    const verify = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
      method: 'POST',
      body: JSON.stringify({ action: 'netbird_login', code: '000000' })
    });
    const challenge = verify.data.challenge;
    assert.equal((await post('logout', { challenge })).code, -3);
    assert.equal((await post('login', { challenge })).code, 0);
    assert.equal((await post('login', { challenge })).code, -3);
    state.auth.twoFactorProtection.netbird = false;
    state.netbird.authorizationPollsRemaining = -1;
    let current = await status();
    assert.equal(current.state, 'authorizing');
    assert.ok(current.authUrl.startsWith('https://'));
    assert.ok(current.userCode);
    assert.equal(current.busy, false);
    assert.equal(current.accessUrl, '');
    assert.equal((await post('login/cancel')).code, 0);
    current = await status();
    assert.equal(current.state, 'notLogin');
    assert.equal(current.authUrl, '');
    assert.equal((await post('login')).code, 0);
    await status();
    await status();
    current = await status();
    assert.equal(current.state, 'connected');
    assert.equal(current.accessUrl, 'https://100.91.0.25');
    assert.equal(current.authUrl, '');
    assert.equal((await post('stop')).code, 0);
    assert.equal((await status()).accessUrl, '');
    assert.equal((await post('login')).msg, 'startRequired');
    assert.equal((await status()).enabled, false);
    assert.equal((await post('uninstall')).code, 0);
    assert.equal(state.netbird.authorized, true);
    assert.equal((await post('install')).code, 0);
    assert.equal((await post('start')).code, 0);
    assert.equal((await status()).state, 'connected');
    assert.equal((await post('logout')).code, 0);
    assert.equal(state.netbird.authorized, false);
    assert.equal((await status()).accessUrl, '');
  } finally {
    hubs.dispose();
  }
});

test('ZeroTier authorization, lifecycle and membership clearing preserve node identity', async () => {
  const { state } = installHandlers();
  const post = (action, body = {}) =>
    fetchJson(`${origin}/api/extensions/zerotier/${action}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  const status = async () => (await fetchJson(`${origin}/api/extensions/zerotier/status`)).data;
  const id = '0123456789abcdef';
  const second = '1123456789abcdef';
  assert.equal((await status()).state, 'notInstalled');
  assert.equal((await post('install')).code, 0);
  assert.equal((await post('join', { networkId: id })).msg, 'startRequired');
  assert.equal((await post('start')).code, 0);
  const nodeId = (await status()).nodeId;
  assert.equal((await status()).state, 'noNetworks');
  state.auth.twoFactorEnabled = true;
  state.auth.twoFactorProtection.zerotier = true;
  assert.equal((await post('join', { networkId: id })).code, -3);
  const verification = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'zerotier_join', code: '000000' })
  });
  const challenge = verification.data.challenge;
  assert.equal((await post('join', { networkId: '../invalid', challenge })).code, -1);
  assert.equal((await post('leave', { networkId: id, challenge })).code, -3);
  assert.equal((await post('join', { networkId: id.toUpperCase(), challenge })).code, 0);
  assert.equal((await post('join', { networkId: second, challenge })).code, -3);
  let current = await status();
  assert.equal(current.state, 'connecting');
  assert.equal(current.networks[0].status, 'ACCESS_DENIED');
  assert.deepEqual(current.networks[0].accessUrls, []);
  state.zerotier.networks[id] = {
    name: 'Test network',
    status: 'OK',
    addresses: ['10.147.1.5', 'fd00::5']
  };
  current = await status();
  assert.equal(current.state, 'connected');
  assert.deepEqual(current.networks[0].accessUrls, ['https://10.147.1.5', 'https://[fd00::5]']);
  state.zerotier.online = false;
  assert.equal((await status()).state, 'offline');
  assert.deepEqual((await status()).networks[0].accessUrls, []);
  state.zerotier.online = true;
  state.auth.twoFactorProtection.zerotier = false;
  assert.equal((await post('join', { networkId: second })).code, 0);
  assert.equal((await post('stop')).code, 0);
  assert.equal((await status()).networks.length, 2);
  assert.deepEqual((await status()).networks[0].accessUrls, []);
  assert.equal((await post('uninstall')).code, 0);
  assert.equal((await status()).nodeId, nodeId);
  assert.equal((await post('leave', { networkId: id })).code, 0);
  assert.equal((await status()).networks.length, 1);
  assert.equal((await post('install')).code, 0);
  assert.equal((await post('start')).code, 0);
  assert.equal((await status()).nodeId, nodeId);
  assert.equal((await status()).networks[0].id, second);
});

for (const provider of ['frp', 'cloudflare']) {
  test(`${provider} clears saved credentials, stops the service, and retains other settings`, async () => {
    const { state, hubs } = installHandlers();
    const get = (endpoint) => fetchJson(`${origin}/api/extensions/${provider}/${endpoint}`);
    const post = (endpoint, body = {}) =>
      fetchJson(`${origin}/api/extensions/${provider}/${endpoint}`, {
        method: 'POST',
        body: JSON.stringify(body)
      });
    const config =
      provider === 'frp'
        ? { serverAddr: 'frp.example.com', serverPort: 7000, remotePort: 8443 }
        : { accessUrl: 'https://kvm.example.com' };
    const token =
      provider === 'frp'
        ? 'mock-private-token'
        : btoa(
            JSON.stringify({
              a: 'account',
              t: 'd1838763-3549-430f-a54e-692caf9f1653',
              s: btoa('secret')
            })
          );

    try {
      assert.equal((await post('install')).code, 0);
      assert.equal((await post('config', { ...config, token })).code, 0);
      assert.equal((await post('start')).code, 0);
      assert.equal((await post('clear-token')).code, 0);
      assert.equal((await get('status')).data.enabled, false);
      assert.equal(state[provider].token, '');
      assert.deepEqual((await get('config')).data, {
        ...config,
        tokenSet: false,
        ...(provider === 'cloudflare' ? { mode: 'named' } : {})
      });
      assert.equal((await post('clear-token')).code, 0);

      assert.equal((await post('config', { ...config, token })).code, 0);
      assert.equal((await post('uninstall')).code, 0);
      assert.equal((await get('config')).data.tokenSet, true);
      assert.equal((await post('clear-token')).code, 0);
      assert.equal((await get('status')).data.state, 'notInstalled');
      assert.deepEqual((await get('config')).data, {
        ...config,
        tokenSet: false,
        ...(provider === 'cloudflare' ? { mode: 'named' } : {})
      });
    } finally {
      hubs.dispose();
    }
  });

  test(`${provider} token clearing requires a single-use configure challenge`, async () => {
    const { state, hubs } = installHandlers();
    Object.assign(state[provider], { installed: true, enabled: true, token: 'saved-credential' });
    state.auth.twoFactorEnabled = true;
    state.auth.twoFactorProtection[provider] = true;
    const clear = (challenge) =>
      fetchJson(`${origin}/api/extensions/${provider}/clear-token`, {
        method: 'POST',
        body: JSON.stringify({ challenge })
      });
    const verify = async (action) =>
      (
        await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
          method: 'POST',
          body: JSON.stringify({ action: `${provider}_${action}`, code: '000000' })
        })
      ).data.challenge;

    try {
      assert.equal((await clear()).code, -3);
      assert.equal((await clear(await verify('stop'))).code, -3);
      assert.equal(state[provider].token, 'saved-credential');
      assert.equal(state[provider].enabled, true);
      const challenge = await verify('configure');
      assert.equal((await clear(challenge)).code, 0);
      assert.equal(state[provider].token, '');
      assert.equal(state[provider].enabled, false);
      assert.equal((await clear(challenge)).code, -3);
    } finally {
      hubs.dispose();
    }
  });
}

test('handlers are exact-origin, preserve envelopes, and retain stateful mutations', async () => {
  const { state, hubs } = installHandlers();

  assert.deepEqual(state.auth.twoFactorProtection, {
    frp: false,
    cloudflare: false,
    zerotier: false,
    netbird: false,
    login: false,
    webTerminal: false,
    ssh: false,
    tailscale: false,
    mcp: false,
    targetPower: false,
    nanoKVMRestart: false,
    scripts: true,
    screenTimelapse: false,
    deleteImage: false
  });

  const hostnameBefore = await fetchJson(`${origin}/api/vm/hostname`);
  assert.deepEqual(hostnameBefore, { code: 0, msg: 'ok', data: { hostname: 'mock-nanokvm' } });

  await fetchJson(`${origin}/api/vm/hostname`, {
    method: 'POST',
    body: JSON.stringify({ hostname: 'mock-renamed' })
  });
  const hostnameAfter = await fetchJson(`${origin}/api/vm/hostname`);
  assert.equal(hostnameAfter.data.hostname, 'mock-renamed');

  const fingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(fingerRobotDebug.data.enabled, true);
  assert.equal(fingerRobotDebug.data.zeroUs, 1500);
  assert.deepEqual(fingerRobotDebug.data.releaseUs, { min: 889, max: 2050, default: 2050 });
  assert.deepEqual(fingerRobotDebug.data.pressUs, { min: 889, max: 2000, default: 899 });
  assert.deepEqual(fingerRobotDebug.data.count, {
    min: 1,
    default: 50,
    unlimitedValue: 0
  });
  assert.deepEqual(fingerRobotDebug.data.pressSpeedUsPerSecond, {
    min: 200,
    max: 2000,
    default: 2000
  });
  assert.deepEqual(
    {
      releaseUs: fingerRobotDebug.data.releaseUs.default,
      pressUs: fingerRobotDebug.data.pressUs.default,
      count: fingerRobotDebug.data.count.default,
      pressMs: fingerRobotDebug.data.pressMs.default,
      pressSpeedUsPerSecond: fingerRobotDebug.data.pressSpeedUsPerSecond.default
    },
    { releaseUs: 2050, pressUs: 899, count: 50, pressMs: 100, pressSpeedUsPerSecond: 2000 }
  );
  const rejectedFingerRobotChannel = await fetchJson(`${origin}/api/vm/finger-robot/debug/test`, {
    method: 'POST',
    body: JSON.stringify({
      releaseUs: 2050,
      pressUs: 1900,
      count: 1,
      pressMs: 800,
      pressSpeedUsPerSecond: 1000,
      channel: 0
    })
  });
  assert.equal(rejectedFingerRobotChannel.code, -1);
  const rejectedFingerRobotSpeed = await fetchJson(`${origin}/api/vm/finger-robot/debug/test`, {
    method: 'POST',
    body: JSON.stringify({
      releaseUs: 2050,
      pressUs: 1900,
      count: 1,
      pressMs: 800,
      pressSpeedUsPerSecond: 199
    })
  });
  assert.equal(rejectedFingerRobotSpeed.code, -1);
  const startedFingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug/test`, {
    method: 'POST',
    body: JSON.stringify({
      releaseUs: 2050,
      pressUs: 1900,
      count: 501,
      pressMs: 800,
      pressSpeedUsPerSecond: 1000
    })
  });
  assert.equal(startedFingerRobotDebug.code, 0);
  assert.equal(startedFingerRobotDebug.data, null);
  const runningFingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(runningFingerRobotDebug.data.running, true);
  assert.equal(runningFingerRobotDebug.data.total, 501);
  const stoppedFingerRobotDebugRequest = await fetchJson(
    `${origin}/api/vm/finger-robot/debug/stop`,
    { method: 'POST' }
  );
  assert.equal(stoppedFingerRobotDebugRequest.code, 0);
  const stoppedFingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(stoppedFingerRobotDebug.data.running, false);
  assert.equal(stoppedFingerRobotDebug.data.state, 'idle');
  assert.equal(stoppedFingerRobotDebug.data.result, 'cancelled');

  const startedUnlimitedFingerRobotDebug = await fetchJson(
    `${origin}/api/vm/finger-robot/debug/test`,
    {
      method: 'POST',
      body: JSON.stringify({
        releaseUs: 2050,
        pressUs: 1900,
        count: 0,
        pressMs: 800,
        pressSpeedUsPerSecond: 1000
      })
    }
  );
  assert.equal(startedUnlimitedFingerRobotDebug.code, 0);
  const unlimitedFingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(unlimitedFingerRobotDebug.data.running, true);
  assert.equal(unlimitedFingerRobotDebug.data.total, 0);
  const continuingUnlimitedFingerRobotDebug = await fetchJson(
    `${origin}/api/vm/finger-robot/debug`
  );
  assert.equal(continuingUnlimitedFingerRobotDebug.data.running, true);
  assert.equal(continuingUnlimitedFingerRobotDebug.data.total, 0);

  const resetFingerRobotDebugRequest = await fetchJson(
    `${origin}/api/vm/finger-robot/debug/reset`,
    { method: 'POST' }
  );
  assert.equal(resetFingerRobotDebugRequest.code, 0);
  assert.equal(resetFingerRobotDebugRequest.data, null);
  const resetFingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(resetFingerRobotDebug.data.running, false);
  assert.equal(resetFingerRobotDebug.data.state, 'idle');
  assert.equal(resetFingerRobotDebug.data.result, 'reset');

  state.vm.fingerRobotDebug.available = false;
  const unavailableFingerRobotStop = await fetchJson(`${origin}/api/vm/finger-robot/debug/stop`, {
    method: 'POST'
  });
  assert.equal(unavailableFingerRobotStop.code, 0);
  assert.equal(state.vm.fingerRobotDebug.available, false);
  assert.equal(state.vm.fingerRobotDebug.state, 'idle');
  assert.equal(state.vm.fingerRobotDebug.result, 'reset');

  const recoveredFingerRobotReset = await fetchJson(`${origin}/api/vm/finger-robot/debug/reset`, {
    method: 'POST'
  });
  assert.equal(recoveredFingerRobotReset.code, 0);
  assert.equal(state.vm.fingerRobotDebug.available, true);

  await fetchJson(`${origin}/api/stream/quality`, {
    method: 'POST',
    body: JSON.stringify({ quality: 12345 })
  });
  const encoder = await fetchJson(`${origin}/api/stream/encoder`);
  assert.equal(encoder.data.bitrate, 12345);

  await fetchJson(`${origin}/api/network/wifi/disconnect`, { method: 'POST' });
  const disconnected = await fetchJson(`${origin}/api/network/wifi`);
  assert.equal(disconnected.data.connected, false);
  await fetchJson(`${origin}/api/network/wifi/connect`, {
    method: 'POST',
    body: JSON.stringify({ ssid: 'Guest Mock Network', password: '' })
  });
  const connected = await fetchJson(`${origin}/api/network/wifi`);
  assert.equal(connected.data.wifi.ssid, 'Guest Mock Network');

  const emptyPreset = await fetchJson(`${origin}/api/network/wifi/preset`);
  assert.equal(emptyPreset.data.ssid, '');
  const rejectedPreset = await fetchJson(`${origin}/api/network/wifi/preset`, {
    method: 'POST',
    body: JSON.stringify({ ssid: '', password: '' })
  });
  assert.equal(rejectedPreset.code, -1);
  const savedPreset = await fetchJson(`${origin}/api/network/wifi/preset`, {
    method: 'POST',
    body: JSON.stringify({ ssid: 'Target Site WiFi', password: 'mock-placeholder-only' })
  });
  assert.equal(savedPreset.code, 0);
  const readPreset = await fetchJson(`${origin}/api/network/wifi/preset`);
  assert.equal(readPreset.data.ssid, 'Target Site WiFi');
  assert.equal('password' in readPreset.data, false);
  await fetchJson(`${origin}/api/network/wifi/preset`, {
    method: 'POST',
    body: JSON.stringify({ ssid: 'Overwrite Site WiFi', password: '' })
  });
  const overwrittenPreset = await fetchJson(`${origin}/api/network/wifi/preset`);
  assert.equal(overwrittenPreset.data.ssid, 'Overwrite Site WiFi');
  const removedPreset = await fetchJson(`${origin}/api/network/wifi/preset/remove`, {
    method: 'POST'
  });
  assert.equal(removedPreset.code, 0);
  const clearedPreset = await fetchJson(`${origin}/api/network/wifi/preset`);
  assert.equal(clearedPreset.data.ssid, '');
  const removeMissingPreset = await fetchJson(`${origin}/api/network/wifi/preset/remove`, {
    method: 'POST'
  });
  assert.equal(removeMissingPreset.code, -1);

  await fetchJson(`${origin}/api/vm/device/virtual`, {
    method: 'POST',
    body: JSON.stringify({ device: 'network' })
  });
  const virtualDevice = await fetchJson(`${origin}/api/vm/device/virtual`);
  assert.equal(virtualDevice.data.isNetworkEnabled, false);

  await fetchJson(`${origin}/api/ai/config`, {
    method: 'POST',
    body: JSON.stringify({ mcpEnabled: false })
  });
  const aiConfig = await fetchJson(`${origin}/api/ai/config`);
  assert.equal(aiConfig.data.mcpEnabled, false);
  assert.equal('mcpApiKey' in aiConfig.data, false);
  assert.equal('mcpApiKeyMasked' in aiConfig.data, false);

  await fetchJson(`${origin}/api/ai/config`, {
    method: 'POST',
    body: JSON.stringify({ mcpEnabled: true })
  });
  const enabledAIConfig = await fetchJson(`${origin}/api/ai/config`);
  assert.equal(enabledAIConfig.data.mcpEnabled, true);
  assert.equal(enabledAIConfig.data.mcpApiKeyMasked, 'nag_mcp_mock...000000');
  assert.equal('mcpApiKey' in enabledAIConfig.data, false);

  const mcpKey = await fetchJson(`${origin}/api/ai/mcp/key`, { method: 'POST' });
  assert.equal(mcpKey.data.mcpApiKey, 'nag_mcp_mock-development-key-not-a-secret-000000000');

  await fetchJson(`${origin}/api/auth/2fa/enable`, { method: 'POST' });
  const authorization = await fetchJson(`${origin}/api/auth/2fa/protection/authorize`, {
    method: 'POST',
    body: JSON.stringify({ password: 'mock-password', code: '000000' })
  });
  assert.equal(authorization.data.expiresIn, 300);
  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: true,
      webTerminal: false,
      ssh: true,
      tailscale: true,
      mcp: true,
      targetPower: true,
      nanoKVMRestart: true,
      scripts: true,
      screenTimelapse: true,
      deleteImage: true
    })
  });
  const protection = await fetchJson(`${origin}/api/auth/2fa/protection`);
  assert.equal(protection.data.policy.login, true);
  assert.equal(protection.data.policy.webTerminal, false);
  assert.equal(protection.data.policy.deleteImage, true);

  const passwordOnlyTerminalAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'open_web_terminal', password: 'mock-password' })
    }
  );
  // webTerminal=false: the password alone mints the terminal challenge; any
  // submitted code is ignored.
  assert.equal(passwordOnlyTerminalAuthorization.code, 0);
  assert.equal(
    passwordOnlyTerminalAuthorization.data.challenge,
    'mock-open-web-terminal-challenge'
  );
  const terminalAuthorization = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({
      action: 'open_web_terminal',
      password: 'mock-password',
      code: '000000'
    })
  });
  assert.equal(terminalAuthorization.data.challenge, 'mock-open-web-terminal-challenge');
  const terminalGrant = await fetchJson(`${origin}/api/vm/terminal/auth`, {
    method: 'POST',
    body: JSON.stringify({ challenge: terminalAuthorization.data.challenge })
  });
  assert.equal(terminalGrant.code, 0);
  const reusedTerminalGrant = await fetchJson(`${origin}/api/vm/terminal/auth`, {
    method: 'POST',
    body: JSON.stringify({ challenge: terminalAuthorization.data.challenge })
  });
  assert.equal(reusedTerminalGrant.code, -1);

  // webTerminal=true: the second factor is required on top of the password.
  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: true,
      webTerminal: true,
      ssh: true,
      tailscale: true,
      mcp: true,
      targetPower: true,
      nanoKVMRestart: true,
      scripts: true,
      screenTimelapse: true,
      deleteImage: true
    })
  });
  const passwordOnlyTerminalDenied = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({
        action: 'open_web_terminal',
        password: '[REDACTED:Generic Password Field]'
      })
    }
  );
  assert.equal(passwordOnlyTerminalDenied.code, -2);
  const protectedTerminalAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({
        action: 'open_web_terminal',
        password: '[REDACTED:Generic Password Field]',
        code: '000000'
      })
    }
  );
  assert.equal(protectedTerminalAuthorization.data.challenge, 'mock-open-web-terminal-challenge');

  const disableScreenTimelapse = await fetchJson(`${origin}/api/ai/screen-timelapse/state`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false })
  });
  assert.equal(disableScreenTimelapse.data.enabled, false);
  const blockedScreenTimelapseEnable = await fetchJson(`${origin}/api/ai/screen-timelapse/state`, {
    method: 'POST',
    body: JSON.stringify({ enabled: true })
  });
  assert.equal(blockedScreenTimelapseEnable.code, -3);
  const screenTimelapseAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'screen_timelapse_enable', code: '000000' })
    }
  );
  assert.equal(
    screenTimelapseAuthorization.data.challenge,
    'mock-screen-timelapse-enable-challenge'
  );
  const authorizedScreenTimelapseEnable = await fetchJson(
    `${origin}/api/ai/screen-timelapse/state`,
    {
      method: 'POST',
      body: JSON.stringify({
        enabled: true,
        challenge: screenTimelapseAuthorization.data.challenge
      })
    }
  );
  assert.equal(authorizedScreenTimelapseEnable.data.enabled, true);
  await fetchJson(`${origin}/api/ai/screen-timelapse/state`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false })
  });
  const reusedScreenTimelapseAuthorization = await fetchJson(
    `${origin}/api/ai/screen-timelapse/state`,
    {
      method: 'POST',
      body: JSON.stringify({
        enabled: true,
        challenge: screenTimelapseAuthorization.data.challenge
      })
    }
  );
  assert.equal(reusedScreenTimelapseAuthorization.code, -3);

  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: true,
      webTerminal: false,
      ssh: true,
      tailscale: true,
      mcp: true,
      targetPower: true,
      nanoKVMRestart: true,
      scripts: true,
      screenTimelapse: false,
      deleteImage: true
    })
  });
  const unprotectedScreenTimelapseEnable = await fetchJson(
    `${origin}/api/ai/screen-timelapse/state`,
    {
      method: 'POST',
      body: JSON.stringify({ enabled: true })
    }
  );
  assert.equal(unprotectedScreenTimelapseEnable.data.enabled, true);

  // The servo actuations (press and replace-arm) require a target_power challenge
  // while the policy is enabled; release and config save succeed without one.
  const ungatedFingerRobotRelease = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(ungatedFingerRobotRelease.code, 0);
  const blockedFingerRobotPress = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(blockedFingerRobotPress.code, -3);
  const blockedFingerRobotReplaceArm = await fetchJson(
    `${origin}/api/vm/finger-robot/replace-arm`,
    {
      method: 'POST',
      body: JSON.stringify({})
    }
  );
  assert.equal(blockedFingerRobotReplaceArm.code, -3);
  const blockedFingerRobotDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug/test`, {
    method: 'POST',
    body: JSON.stringify({
      releaseUs: 2050,
      pressUs: 1900,
      count: 1,
      pressMs: 800,
      pressSpeedUsPerSecond: 1000
    })
  });
  assert.equal(blockedFingerRobotDebug.code, -3);
  const emergencyFingerRobotReset = await fetchJson(`${origin}/api/vm/finger-robot/debug/reset`, {
    method: 'POST'
  });
  assert.equal(emergencyFingerRobotReset.code, 0);

  const targetPowerAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'target_power', code: '000000' })
    }
  );
  assert.equal(targetPowerAuthorization.data.challenge, 'mock-target-power-challenge');
  // Ungated operations accept the challenge field but never consume it.
  const releaseWithIgnoredChallenge = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST',
    body: JSON.stringify({ challenge: targetPowerAuthorization.data.challenge })
  });
  assert.equal(releaseWithIgnoredChallenge.code, 0);
  const authorizedFingerRobotPress = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({ challenge: targetPowerAuthorization.data.challenge })
  });
  assert.equal(authorizedFingerRobotPress.code, 0);
  const reusedTargetPowerAuthorization = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({ challenge: targetPowerAuthorization.data.challenge })
  });
  assert.equal(reusedTargetPowerAuthorization.code, -3);

  // Replace-arm consumes its own one-time challenge from the same target_power slot.
  const replaceArmAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'target_power', code: '000000' })
    }
  );
  assert.equal(replaceArmAuthorization.code, 0);
  const authorizedFingerRobotReplaceArm = await fetchJson(
    `${origin}/api/vm/finger-robot/replace-arm`,
    {
      method: 'POST',
      body: JSON.stringify({ challenge: replaceArmAuthorization.data.challenge })
    }
  );
  assert.equal(authorizedFingerRobotReplaceArm.code, 0);
  const reusedReplaceArmAuthorization = await fetchJson(
    `${origin}/api/vm/finger-robot/replace-arm`,
    {
      method: 'POST',
      body: JSON.stringify({ challenge: replaceArmAuthorization.data.challenge })
    }
  );
  assert.equal(reusedReplaceArmAuthorization.code, -3);

  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: true,
      webTerminal: false,
      ssh: true,
      tailscale: true,
      mcp: true,
      targetPower: false,
      nanoKVMRestart: true,
      scripts: true,
      screenTimelapse: true,
      deleteImage: true
    })
  });
  const unprotectedFingerRobotRelease = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(unprotectedFingerRobotRelease.code, 0);
  const unprotectedFingerRobotReplaceArm = await fetchJson(
    `${origin}/api/vm/finger-robot/replace-arm`,
    {
      method: 'POST',
      body: JSON.stringify({})
    }
  );
  assert.equal(unprotectedFingerRobotReplaceArm.code, 0);

  const blockedNanoKVMRestart = await fetchJson(`${origin}/api/vm/system/reboot`, {
    method: 'POST'
  });
  assert.equal(blockedNanoKVMRestart.code, -3);

  const nanoKVMRestartAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'nanokvm_restart', code: '000000' })
    }
  );
  assert.equal(nanoKVMRestartAuthorization.data.challenge, 'mock-nanokvm-restart-challenge');
  const authorizedNanoKVMRestart = await fetchJson(`${origin}/api/vm/system/reboot`, {
    method: 'POST',
    body: JSON.stringify({ challenge: nanoKVMRestartAuthorization.data.challenge })
  });
  assert.equal(authorizedNanoKVMRestart.code, 0);
  const reusedNanoKVMRestartAuthorization = await fetchJson(`${origin}/api/vm/system/reboot`, {
    method: 'POST',
    body: JSON.stringify({ challenge: nanoKVMRestartAuthorization.data.challenge })
  });
  assert.equal(reusedNanoKVMRestartAuthorization.code, -3);

  const blockedFlashingModeRestart = await fetchJson(`${origin}/api/vm/system/reboot-to-flashing`, {
    method: 'POST'
  });
  assert.equal(blockedFlashingModeRestart.code, -3);
  const flashingModeRestartAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'nanokvm_restart', code: '000000' })
    }
  );
  const authorizedFlashingModeRestart = await fetchJson(
    `${origin}/api/vm/system/reboot-to-flashing`,
    {
      method: 'POST',
      body: JSON.stringify({ challenge: flashingModeRestartAuthorization.data.challenge })
    }
  );
  assert.equal(authorizedFlashingModeRestart.code, 0);
  assert.equal(authorizedFlashingModeRestart.data.mode, 'flashing');
  const reusedFlashingModeRestartAuthorization = await fetchJson(
    `${origin}/api/vm/system/reboot-to-flashing`,
    {
      method: 'POST',
      body: JSON.stringify({ challenge: flashingModeRestartAuthorization.data.challenge })
    }
  );
  assert.equal(reusedFlashingModeRestartAuthorization.code, -3);

  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: true,
      webTerminal: false,
      ssh: true,
      tailscale: true,
      mcp: true,
      targetPower: false,
      nanoKVMRestart: false,
      scripts: true,
      screenTimelapse: true,
      deleteImage: true
    })
  });
  const unprotectedNanoKVMRestart = await fetchJson(`${origin}/api/vm/system/reboot`, {
    method: 'POST'
  });
  assert.equal(unprotectedNanoKVMRestart.code, 0);
  const unprotectedFlashingModeRestart = await fetchJson(
    `${origin}/api/vm/system/reboot-to-flashing`,
    { method: 'POST' }
  );
  assert.equal(unprotectedFlashingModeRestart.code, 0);

  const blockedDeleteImage = await fetchJson(`${origin}/api/storage/image/delete`, {
    method: 'POST',
    body: JSON.stringify({ file: '/data/mock.iso' })
  });
  assert.equal(blockedDeleteImage.code, -3);

  const deleteImageAuthorization = await fetchJson(
    `${origin}/api/auth/2fa/verify-sensitive-action`,
    {
      method: 'POST',
      body: JSON.stringify({ action: 'delete_image', code: '000000' })
    }
  );
  assert.equal(deleteImageAuthorization.data.challenge, 'mock-delete-image-challenge');
  const authorizedDeleteImage = await fetchJson(`${origin}/api/storage/image/delete`, {
    method: 'POST',
    body: JSON.stringify({
      file: '/data/mock.iso',
      challenge: deleteImageAuthorization.data.challenge
    })
  });
  assert.equal(authorizedDeleteImage.code, 0);
  const reusedDeleteImageAuthorization = await fetchJson(`${origin}/api/storage/image/delete`, {
    method: 'POST',
    body: JSON.stringify({
      file: '/data/mock.iso',
      challenge: deleteImageAuthorization.data.challenge
    })
  });
  assert.equal(reusedDeleteImageAuthorization.code, -3);

  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: true,
      webTerminal: false,
      ssh: true,
      tailscale: true,
      mcp: true,
      targetPower: false,
      nanoKVMRestart: false,
      scripts: true,
      screenTimelapse: true,
      deleteImage: false
    })
  });
  const unprotectedDeleteImage = await fetchJson(`${origin}/api/storage/image/delete`, {
    method: 'POST',
    body: JSON.stringify({ file: '/data/mock.iso' })
  });
  assert.equal(unprotectedDeleteImage.code, 0);

  const directLoginStatus = await fetchJson(`${origin}/api/auth/login/2fa-direct`);
  assert.equal(directLoginStatus.data.enabled, true);
  const directLogin = await fetchJson(`${origin}/api/auth/login`, {
    method: 'POST',
    body: JSON.stringify({ code: '000000' })
  });
  assert.equal(directLogin.code, 0);

  const blockedMCPKeyCopy = await fetchJson(`${origin}/api/ai/mcp/key/copy`, {
    method: 'POST'
  });
  assert.equal(blockedMCPKeyCopy.code, -3);

  const mcpChallenge = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'mcp_show_key', code: '000000' })
  });
  assert.equal(mcpChallenge.data.challenge, 'mock-sensitive-action');

  const sharedMCPKeyCopy = await fetchJson(`${origin}/api/ai/mcp/key/copy`, {
    method: 'POST'
  });
  assert.equal(
    sharedMCPKeyCopy.data.mcpApiKey,
    'nag_mcp_mock-development-key-not-a-secret-000000000'
  );
  const protectionWithMCPGrant = await fetchJson(`${origin}/api/auth/2fa/protection`);
  assert.ok(protectionWithMCPGrant.data.mcpAuthorizationExpiresIn > 0);

  const sharedMCPDisable = await fetchJson(`${origin}/api/ai/config`, {
    method: 'POST',
    body: JSON.stringify({ mcpEnabled: false })
  });
  assert.equal(sharedMCPDisable.code, 0);

  await fetchJson(`${origin}/api/auth/2fa/disable`, { method: 'POST' });
  await fetchJson(`${origin}/api/auth/2fa/enable`, { method: 'POST' });
  const resetProtection = await fetchJson(`${origin}/api/auth/2fa/protection`);
  assert.equal(resetProtection.data.policy.login, false);
  assert.equal(resetProtection.data.policy.webTerminal, false);
  assert.equal(resetProtection.data.policy.scripts, true);
  const resetDirectLoginStatus = await fetchJson(`${origin}/api/auth/login/2fa-direct`);
  assert.equal(resetDirectLoginStatus.data.enabled, false);
  const blockedDirectLogin = await fetchJson(`${origin}/api/auth/login`, {
    method: 'POST',
    body: JSON.stringify({ code: '000000' })
  });
  assert.equal(blockedDirectLogin.code, -2);

  await fetchJson(`${origin}/api/extensions/tailscale/down`, { method: 'POST' });
  let tailscale = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(tailscale.data.state, 'stopped');
  assert.equal(tailscale.data.routingAvailable, true);
  assert.equal(tailscale.data.advertiseExitNode, false);
  assert.equal(tailscale.data.advertiseSubnetRoute, false);
  assert.equal(tailscale.data.lanSubnet, '192.168.0.0/22');

  await fetchJson(`${origin}/api/extensions/tailscale/exit-node`, {
    method: 'POST',
    body: JSON.stringify({ enabled: true })
  });
  await fetchJson(`${origin}/api/extensions/tailscale/subnet-route`, {
    method: 'POST',
    body: JSON.stringify({ enabled: true })
  });
  tailscale = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(tailscale.data.advertiseExitNode, true);
  assert.equal(tailscale.data.advertiseSubnetRoute, true);

  await fetchJson(`${origin}/api/auth/2fa/protection/authorize`, {
    method: 'POST',
    body: JSON.stringify({ code: '000000' })
  });
  await fetchJson(`${origin}/api/auth/2fa/protection`, {
    method: 'POST',
    body: JSON.stringify({
      login: false,
      webTerminal: false,
      ssh: false,
      tailscale: true,
      mcp: false,
      targetPower: false,
      nanoKVMRestart: false,
      scripts: true,
      screenTimelapse: false,
      deleteImage: false
    })
  });

  const blockedExitNode = await fetchJson(`${origin}/api/extensions/tailscale/exit-node`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false })
  });
  assert.equal(blockedExitNode.code, -3);

  const exitNodeChallenge = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'tailscale_set_exit_node', code: '000000' })
  });
  assert.equal(exitNodeChallenge.data.challenge, 'mock-tailscale-exit-node-challenge');
  const disabledExitNode = await fetchJson(`${origin}/api/extensions/tailscale/exit-node`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false, challenge: exitNodeChallenge.data.challenge })
  });
  assert.equal(disabledExitNode.code, 0);
  const reusedExitNodeChallenge = await fetchJson(`${origin}/api/extensions/tailscale/exit-node`, {
    method: 'POST',
    body: JSON.stringify({ enabled: true, challenge: exitNodeChallenge.data.challenge })
  });
  assert.equal(reusedExitNodeChallenge.code, -3);

  const blockedSubnetRoute = await fetchJson(`${origin}/api/extensions/tailscale/subnet-route`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false })
  });
  assert.equal(blockedSubnetRoute.code, -3);
  const subnetRouteChallenge = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'tailscale_set_subnet_route', code: '000000' })
  });
  assert.equal(subnetRouteChallenge.data.challenge, 'mock-tailscale-subnet-route-challenge');
  const disabledSubnetRoute = await fetchJson(`${origin}/api/extensions/tailscale/subnet-route`, {
    method: 'POST',
    body: JSON.stringify({
      enabled: false,
      challenge: subnetRouteChallenge.data.challenge
    })
  });
  assert.equal(disabledSubnetRoute.code, 0);

  tailscale = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(tailscale.data.advertiseExitNode, false);
  assert.equal(tailscale.data.advertiseSubnetRoute, false);

  hubs.dispose();
});

test('mocked Sipeed enable exposes managed states and reuses the login challenge', async () => {
  const { state, hubs } = installHandlers();
  state.tailscale.state = 'notLogin';
  state.tailscale.login = false;
  state.tailscale.activeMode = null;
  state.auth.twoFactorEnabled = true;
  state.auth.twoFactorProtection.tailscale = true;

  const initial = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.deepEqual(initial.data.availableModes, ['official', 'sipeed']);
  assert.equal(initial.data.activeMode, null);
  assert.equal(initial.data.managedState, 'disabled');

  const blocked = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/enable`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(blocked.code, -3);

  const verified = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'tailscale_login', code: '000000' })
  });
  assert.equal(verified.data.challenge, 'mock-tailscale-login-challenge');

  const accepted = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/enable`, {
    method: 'POST',
    body: JSON.stringify({ challenge: verified.data.challenge, acceptDisclaimer: true })
  });
  assert.equal(accepted.code, 0);
  assert.equal(accepted.data.activeMode, 'sipeed');
  assert.equal(accepted.data.managedState, 'enrolling');
  assert.equal(accepted.data.sipeedDisclaimerAccepted, true);
  assert.equal(state.tailscale.managedNetworkRegistered, true);
  assert.equal(state.tailscale.managedEnrollRequests, 1);
  assert.equal(state.tailscale.managedJoinKeyRequests, 0);

  const connecting = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(connecting.data.managedState, 'enabledConnecting');
  const connected = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(connected.data.managedState, 'enabledConnected');
  assert.equal(connected.data.state, 'running');
  assert.equal('deviceToken' in connected.data, false);
  assert.equal('networkId' in connected.data, false);
  assert.equal('profileId' in connected.data, false);

  state.tailscale.state = 'notLogin';
  state.tailscale.login = false;
  state.tailscale.activeMode = 'sipeed';
  state.tailscale.managedState = 'degraded';
  state.tailscale.managedError = 'connectionFailed';
  const retryChallenge = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'tailscale_login', code: '000000' })
  });
  const retried = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/enable`, {
    method: 'POST',
    body: JSON.stringify({ challenge: retryChallenge.data.challenge })
  });
  assert.equal(retried.code, 0);
  assert.equal(retried.data.managedState, 'enrolling');
  assert.equal(state.tailscale.managedEnrollRequests, 1);
  assert.equal(state.tailscale.managedJoinKeyRequests, 1);

  hubs.dispose();
});

test('mocked official restart retains login and exposes delayed startup', async () => {
  const { state, hubs } = installHandlers();
  const endpoint = `${origin}/api/extensions/tailscale`;
  try {
    assert.equal((await fetchJson(`${endpoint}/stop`, { method: 'POST' })).code, 0);
    assert.equal(state.tailscale.login, true);
    assert.equal((await fetchJson(`${endpoint}/start`, { method: 'POST' })).code, 0);
    assert.equal((await fetchJson(`${endpoint}/status`)).data.state, 'notRunning');
    const ready = await fetchJson(`${endpoint}/status`);
    assert.equal(ready.data.state, 'running');
    assert.equal(ready.data.activeMode, 'official');
    assert.equal(state.tailscale.login, true);
  } finally {
    hubs.dispose();
  }
});

test('mocked Sipeed enable drops the official login and does not restore it afterward', async () => {
  const { state, hubs } = installHandlers();
  const endpoint = `${origin}/api/extensions/tailscale`;
  try {
    await fetchJson(`${endpoint}/stop`, { method: 'POST' });
    assert.equal(state.tailscale.login, true);
    const enabled = await fetchJson(`${endpoint}/sipeed/enable`, {
      method: 'POST',
      body: JSON.stringify({ acceptDisclaimer: true })
    });
    assert.equal(enabled.code, 0);
    assert.equal(state.tailscale.login, false);
    await fetchJson(`${endpoint}/status`);
    const connected = await fetchJson(`${endpoint}/status`);
    assert.equal(connected.data.activeMode, 'sipeed');
    assert.equal(connected.data.managedState, 'enabledConnected');
    await fetchJson(`${endpoint}/stop`, { method: 'POST' });
    await fetchJson(`${endpoint}/start`, { method: 'POST' });
    await fetchJson(`${endpoint}/status`);
    const official = await fetchJson(`${endpoint}/status`);
    assert.equal(official.data.state, 'notLogin');
    assert.equal(official.data.activeMode, null);
  } finally {
    hubs.dispose();
  }
});

test('mocked consent survives logout and withdrawal requires agreement again', async () => {
  const { state, hubs } = installHandlers();
  state.tailscale.state = 'notLogin';
  state.tailscale.login = false;
  state.tailscale.activeMode = null;
  const post = (action, body = {}) =>
    fetchJson(`${origin}/api/extensions/tailscale/sipeed/${action}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  try {
    const blocked = await post('enable');
    assert.equal(blocked.code, -1);
    assert.equal(state.tailscale.managedEnrollRequests, 0);
    assert.equal(state.tailscale.sipeedDisclaimerAccepted, false);
    assert.equal((await post('enable', { acceptDisclaimer: true })).code, 0);
    await fetchJson(`${origin}/api/extensions/tailscale/status`);
    await fetchJson(`${origin}/api/extensions/tailscale/status`);
    assert.equal((await post('logout')).code, 0);
    assert.equal(state.tailscale.sipeedDisclaimerAccepted, true);
    assert.equal((await post('enable')).code, 0);
    await fetchJson(`${origin}/api/extensions/tailscale/status`);
    await fetchJson(`${origin}/api/extensions/tailscale/status`);
    assert.equal((await post('disclaimer/revoke')).code, 0);
    const refreshed = await fetchJson(`${origin}/api/extensions/tailscale/status`);
    assert.equal(refreshed.data.sipeedDisclaimerAccepted, false);
    assert.equal(refreshed.data.managedState, 'disabled');
    assert.equal(refreshed.data.state, 'notRunning');
    assert.equal(state.tailscale.managedNetworkRegistered, false);
    assert.deepEqual(state.tailscale.sipeedDevices, []);
    assert.equal((await post('enable')).code, -1);
  } finally {
    hubs.dispose();
  }
});

test('both regions support Sipeed and official modes while the manifest remains authoritative', async () => {
  for (const region of ['cn', 'global']) {
    const { state, hubs } = installHandlers();
    try {
      state.application.region = region;
      state.tailscale.state = 'notLogin';
      state.tailscale.login = false;
      state.tailscale.activeMode = null;
      const initial = await fetchJson(`${origin}/api/extensions/tailscale/status`);
      assert.deepEqual(initial.data.availableModes, ['official', 'sipeed']);
      const enabled = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/enable`, {
        method: 'POST',
        body: JSON.stringify({ acceptDisclaimer: true })
      });
      assert.equal(enabled.code, 0, region);
      await fetchJson(`${origin}/api/extensions/tailscale/status`);
      const connected = await fetchJson(`${origin}/api/extensions/tailscale/status`);
      assert.equal(connected.data.managedState, 'enabledConnected');
      assert.equal(connected.data.activeMode, 'sipeed');

      const changed = await fetchJson(`${origin}/api/application/region`, {
        method: 'POST',
        body: JSON.stringify({ region: region === 'cn' ? 'global' : 'cn' })
      });
      assert.equal(changed.code, 0);
      assert.equal(changed.data.cloudCleanupFailed, false);
      const afterChange = await fetchJson(`${origin}/api/extensions/tailscale/status`);
      assert.equal(afterChange.data.managedState, 'enabledConnected');
      assert.equal(afterChange.data.activeMode, 'sipeed');
      assert.equal(afterChange.data.sipeedDisclaimerAccepted, true);
      assert.equal(state.tailscale.managedNetworkRegistered, true);

      state.application.tailscaleCnEnabled = false;
      const disabled = await fetchJson(`${origin}/api/extensions/tailscale/status`);
      assert.deepEqual(disabled.data.availableModes, ['official']);
      const refused = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/enable`, {
        method: 'POST',
        body: JSON.stringify({ acceptDisclaimer: true })
      });
      assert.equal(refused.code, -1);
      state.application.tailscaleCnEnabled = true;
      const restored = await fetchJson(`${origin}/api/extensions/tailscale/status`);
      assert.equal(restored.data.activeMode, 'sipeed');
    } finally {
      hubs.dispose();
    }
  }
});

test('mocked manifest gate hides the Sipeed mode and refuses its endpoints', async () => {
  const { state, hubs } = installHandlers();
  // Default mock state is cn + manifest on, matching a real CN firmware.
  state.application.tailscaleCnEnabled = false;
  state.tailscale.managedState = 'enabledConnected';
  state.tailscale.activeMode = 'sipeed';
  state.tailscale.sipeedIdentityValid = true;

  const sync = await fetchJson(`${origin}/api/application/features/sync`, { method: 'POST' });
  assert.equal(sync.code, 0);
  assert.equal(state.application.tailscaleCnEnabled, false);

  const status = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(status.code, 0);
  assert.deepEqual(status.data.availableModes, ['official']);
  assert.equal(status.data.activeMode, null);
  assert.equal(status.data.managedState, 'disabled');
  assert.equal(status.data.managedError, '');
  assert.equal(status.data.retryAfterSeconds, 0);

  const endpoints = [
    ['POST', '/api/extensions/tailscale/sipeed/enable'],
    ['POST', '/api/extensions/tailscale/sipeed/logout'],
    ['POST', '/api/extensions/tailscale/sipeed/disclaimer/revoke'],
    ['POST', '/api/extensions/tailscale/sipeed/claim-code'],
    ['GET', '/api/extensions/tailscale/sipeed/pending'],
    ['POST', '/api/extensions/tailscale/sipeed/pending/pending-1/approve'],
    ['POST', '/api/extensions/tailscale/sipeed/pending/pending-1/reject'],
    ['POST', '/api/extensions/tailscale/sipeed/authorize'],
    ['GET', '/api/extensions/tailscale/sipeed/devices'],
    ['POST', '/api/extensions/tailscale/sipeed/devices/3/name'],
    ['DELETE', '/api/extensions/tailscale/sipeed/devices/3']
  ];
  for (const [method, path] of endpoints) {
    const init = { method };
    if (method !== 'GET') init.body = JSON.stringify({});
    const response = await fetchJson(`${origin}${path}`, init);
    assert.equal(response.code, -1, `${method} ${path}`);
    assert.equal(response.msg, 'managed tailscale mode is unavailable', `${method} ${path}`);
  }

  // The refused requests must not mutate the mock state.
  assert.equal(state.tailscale.activeMode, 'sipeed');
  assert.equal(state.tailscale.sipeedIdentityValid, true);

  hubs.dispose();
});

test('mocked Tailscale preserves rate limits and official/managed status independently', async () => {
  const { state, hubs } = installHandlers();

  const cleanOfficial = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(cleanOfficial.data.activeMode, 'official');
  assert.equal(cleanOfficial.data.managedState, 'disabled');
  assert.equal(cleanOfficial.data.managedError, '');

  state.tailscale.managedNetworkRegistered = true;
  state.tailscale.managedState = 'degraded';
  state.tailscale.managedError = 'pairingUnavailable';
  const officialWithManagedFailure = await fetchJson(`${origin}/api/extensions/tailscale/status`);
  assert.equal(officialWithManagedFailure.data.activeMode, 'official');
  assert.equal(officialWithManagedFailure.data.managedState, 'degraded');
  assert.equal(officialWithManagedFailure.data.managedError, 'pairingUnavailable');

  state.tailscale.state = 'notLogin';
  state.tailscale.login = false;
  state.tailscale.activeMode = 'sipeed';
  state.tailscale.managedError = 'rateLimited';
  state.tailscale.retryAfterSeconds = 41;
  state.tailscale.sipeedDisclaimerAccepted = true;
  const limited = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/enable`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(limited.code, -1);
  assert.equal(limited.data.managedState, 'degraded');
  assert.equal(limited.data.managedError, 'rateLimited');
  assert.equal(limited.data.retryAfterSeconds, 41);
  assert.equal(state.tailscale.managedEnrollRequests, 0);
  assert.equal(state.tailscale.managedJoinKeyRequests, 0);

  hubs.dispose();
});

test('mocked Sipeed codes renew explicitly and expire when consumed or approved', async () => {
  const { state, hubs } = installHandlers();
  state.tailscale.activeMode = 'sipeed';
  state.tailscale.managedState = 'enabledConnected';
  state.tailscale.sipeedIdentityValid = true;
  const post = (action, body = {}) =>
    fetchJson(`${origin}/api/extensions/tailscale/sipeed/${action}`, {
      method: 'POST',
      body: JSON.stringify(body)
    });
  try {
    const first = await post('claim-code');
    assert.equal(first.code, 0);
    assert.deepEqual((await post('claim-code')).data, first.data);
    const renewed = await post('claim-code', { refresh: true });
    assert.equal(renewed.code, 0);
    assert.notEqual(renewed.data.code, first.data.code);
    assert.deepEqual((await post('claim-code')).data, renewed.data);

    state.tailscale.sipeedPending = [
      { pendingId: 'pending-1', sas: ['1', '2', '3'], createdAt: new Date().toISOString() }
    ];
    assert.equal((await fetchJson(`${origin}/api/extensions/tailscale/sipeed/pending`)).code, 0);
    assert.equal(state.tailscale.sipeedClaimCode, null);
    const next = await post('claim-code');
    assert.notEqual(next.data.code, renewed.data.code);
    assert.equal((await post('pending/pending-1/approve')).code, 0);
    assert.equal(state.tailscale.sipeedClaimCode, null);
    assert.equal((await post('claim-code')).code, 0);
    assert.equal((await post('authorize', { code: '2345-678A' })).code, 0);
    assert.equal(state.tailscale.sipeedClaimCode, null);
  } finally {
    hubs.dispose();
  }
});

test('mocked Sipeed management uses HTTP 200 business errors and paces empty pending polls', async () => {
  const { state, hubs } = installHandlers();
  state.tailscale.activeMode = 'sipeed';
  state.tailscale.managedState = 'enabledConnected';
  state.tailscale.sipeedIdentityValid = false;

  let response = await fetch(`${origin}/api/extensions/tailscale/sipeed/devices`);
  let body = await response.json();
  assert.equal(response.status, 200);
  assert.equal(body.code, -2);
  assert.equal(body.data.managedState, 'disabled');

  state.tailscale.activeMode = 'sipeed';
  state.tailscale.managedState = 'enabledConnected';
  state.tailscale.sipeedIdentityValid = true;
  state.tailscale.sipeedManagementRateLimitUntil = Date.now() + 5000;
  response = await fetch(`${origin}/api/extensions/tailscale/sipeed/devices`);
  body = await response.json();
  assert.equal(response.status, 200);
  assert.equal(body.code, -2);
  assert.ok(body.data.retryAfterSeconds > 0);

  state.tailscale.sipeedManagementRateLimitUntil = 0;
  state.tailscale.sipeedPending = [];
  const startedAt = Date.now();
  body = await fetchJson(`${origin}/api/extensions/tailscale/sipeed/pending?wait=20`);
  assert.equal(body.code, 0);
  assert.deepEqual(body.data.pending, []);
  assert.ok(Date.now() - startedAt >= 200, 'empty pending response must not form a hot loop');

  hubs.dispose();
});

test('application update mocks expose status and progress online and offline operations', async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const { state, hubs } = installHandlers();

  const idle = await fetchJson(`${origin}/api/application/update/status`);
  assert.equal(idle.code, 0);
  assert.equal(idle.data.phase, 'idle');

  const file = new File(['mock signed bundle'], 'nanokvm-go_v2.1.0-mock.tar.gz', {
    type: 'application/gzip'
  });
  const form = new FormData();
  form.append('file', file);
  const started = await fetchJson(`${origin}/api/application/update/offline`, {
    method: 'POST',
    body: form
  });
  assert.equal(started.code, 0);
  assert.equal(started.data.source, 'offline');
  assert.equal(started.data.phase, 'uploading');

  const busy = await fetchJson(`${origin}/api/application/update`, { method: 'POST' });
  assert.equal(busy.code, -1);
  assert.equal(busy.data.operationId, started.data.operationId);

  t.mock.timers.runAll();
  const completed = await fetchJson(`${origin}/api/application/update/status`);
  assert.equal(completed.data.phase, 'succeeded');
  assert.equal(state.application.current, state.application.latest);

  const mismatched = await fetchJson(`${origin}/api/application/update/status/acknowledge`, {
    method: 'POST',
    body: JSON.stringify({ operationId: 'stale-operation' })
  });
  assert.equal(mismatched.code, -2);
  assert.equal(mismatched.data.operationId, completed.data.operationId);

  const acknowledged = await fetchJson(`${origin}/api/application/update/status/acknowledge`, {
    method: 'POST',
    body: JSON.stringify({ operationId: completed.data.operationId })
  });
  assert.deepEqual(acknowledged.data, {
    operationId: '',
    source: '',
    version: '',
    phase: 'idle',
    progress: 0,
    errorCode: '',
    startedAt: '',
    updatedAt: ''
  });

  const retried = await fetchJson(`${origin}/api/application/update/status/acknowledge`, {
    method: 'POST',
    body: JSON.stringify({ operationId: completed.data.operationId })
  });
  assert.equal(retried.code, 0);
  assert.equal(retried.data.phase, 'idle');
  hubs.dispose();
});

test('application signature verification mock requires and consumes sensitive-action challenge', async () => {
  const { state, hubs } = installHandlers();

  const initial = await fetchJson(`${origin}/api/application/signature-verification`);
  assert.deepEqual(initial, { code: 0, msg: 'ok', data: { enabled: true } });

  const unauthorized = await fetchJson(`${origin}/api/application/signature-verification`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false })
  });
  assert.equal(unauthorized.code, -4);
  assert.equal(state.application.signatureVerificationEnabled, true);

  const verified = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({
      action: 'update_signature_verification',
      password: 'mock-password',
      code: ''
    })
  });
  assert.equal(verified.data.challenge, 'mock-update-signature-verification-challenge');

  const changed = await fetchJson(`${origin}/api/application/signature-verification`, {
    method: 'POST',
    body: JSON.stringify({ enabled: false, challenge: verified.data.challenge })
  });
  assert.deepEqual(changed, { code: 0, msg: 'ok', data: { enabled: false } });
  assert.equal(state.application.signatureVerificationEnabled, false);

  const replay = await fetchJson(`${origin}/api/application/signature-verification`, {
    method: 'POST',
    body: JSON.stringify({ enabled: true, challenge: verified.data.challenge })
  });
  assert.equal(replay.code, -4);
  assert.equal(state.application.signatureVerificationEnabled, false);

  hubs.dispose();
});

test('finger robot product mock preserves configuration and synchronous action semantics', { timeout: 10_000 }, async (t) => {
  const { state, hubs } = installHandlers();

  const initial = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.equal(initial.code, 0);
  assert.deepEqual(initial.data.config, {
    version: 1,
    revision: 1,
    activeArm: 'short',
    strengthByArm: { long: 3, short: 3 },
    pressMs: 800,
    speed: 'slow'
  });
  assert.deepEqual(initial.data.profiles, [
    { id: 'long', displayName: '长摆臂', defaultStrength: 3 },
    { id: 'short', displayName: '短摆臂', defaultStrength: 3 }
  ]);
  assert.deepEqual(initial.data.strength, { min: 1, max: 5, default: 3 });
  assert.deepEqual(initial.data.pressMs, { min: 100, max: 30_000, default: 800 });
  assert.deepEqual(initial.data.speedOptions, ['slow', 'fast']);
  assert.equal(initial.data.defaultSpeed, 'slow');
  assert.deepEqual(initial.data.status, {
    running: false,
    state: 'idle',
    result: 'idle',
    error: ''
  });
  const publicContract = JSON.stringify(initial.data);
  for (const rawPWMField of ['"pressUs"', '"releaseUs"', '"usPerSecond"']) {
    assert.equal(publicContract.includes(rawPWMField), false);
  }
  assert.equal(publicContract.includes('operationRevision'), false);

  const missingPressBody = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST'
  });
  assert.equal(missingPressBody.code, -1);
  const missingReleaseBody = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST'
  });
  assert.equal(missingReleaseBody.code, -1);

  const incomplete = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify({ version: 1, revision: 1 })
  });
  assert.equal(incomplete.code, -1);
  const unknownField = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify({
      ...initial.data.config,
      rawPressUs: 1450
    })
  });
  assert.equal(unknownField.code, -1);
  const unknownProfile = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify({
      ...initial.data.config,
      strengthByArm: { long: 3, short: 3, medium: 3 }
    })
  });
  assert.equal(unknownProfile.code, -1);

  const updatedConfig = {
    version: 1,
    revision: 1,
    activeArm: 'long',
    strengthByArm: { long: 4, short: 2 },
    pressMs: 1200,
    speed: 'fast'
  };
  state.auth.twoFactorEnabled = true;
  state.auth.twoFactorProtection.targetPower = true;
  state.auth.targetPowerChallenge = 'preserved-target-power-challenge';

  const conflict = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify({
      ...updatedConfig,
      revision: 9,
      challenge: state.auth.targetPowerChallenge
    })
  });
  assert.equal(conflict.code, -6);
  assert.equal(conflict.data.changed, false);
  assert.deepEqual(conflict.data.config, initial.data.config);
  assert.equal(state.auth.targetPowerChallenge, 'preserved-target-power-challenge');

  // Config save is no longer gated: it succeeds without a challenge even while the
  // target_power policy is enabled, and it never consumes a minted challenge.
  const saved = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify(updatedConfig)
  });
  assert.equal(saved.code, 0);
  assert.equal(saved.data.changed, true);
  assert.deepEqual(saved.data.config, { ...updatedConfig, revision: 2 });
  assert.equal(state.auth.targetPowerChallenge, 'preserved-target-power-challenge');

  const noOp = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify(saved.data.config)
  });
  assert.equal(noOp.code, 0);
  assert.equal(noOp.data.changed, false);
  assert.equal(noOp.data.config.revision, 2);
  // A stale challenge field is tolerated and ignored on config save.
  const ignoredChallenge = await fetchJson(`${origin}/api/vm/finger-robot`, {
    method: 'PUT',
    body: JSON.stringify({
      ...saved.data.config,
      challenge: 'stale-target-power-challenge'
    })
  });
  assert.equal(ignoredChallenge.code, 0);
  assert.equal(ignoredChallenge.data.changed, false);
  assert.equal(state.auth.targetPowerChallenge, 'preserved-target-power-challenge');

  const blockedPress = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(blockedPress.code, -3);
  const pressAuthorization = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'target_power', code: '000000' })
  });
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const pressRequest = fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({ challenge: pressAuthorization.data.challenge })
  });
  await setImmediate();
  const running = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.equal(running.data.status.running, true);
  assert.equal(running.data.status.result, 'running');

  state.auth.twoFactorProtection.targetPower = false;
  const busy = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(busy.code, -2);
  const releaseRequest = fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  await setImmediate();
  t.mock.timers.runAll();
  assert.equal((await releaseRequest).code, 0);
  assert.equal((await pressRequest).code, -5);
  const afterRelease = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.deepEqual(afterRelease.data.status, {
    running: false,
    state: 'idle',
    result: 'success',
    error: ''
  });

  t.mock.timers.reset();
  const completed = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(completed.code, 0);
  const complete = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.equal(complete.data.status.running, false);
  assert.equal(complete.data.status.state, 'idle');
  assert.equal(complete.data.status.result, 'success');

  state.vm.fingerRobot.available = false;
  const unavailable = await fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(unavailable.code, -5);
  const unavailableRelease = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(unavailableRelease.code, 0);

  hubs.dispose();
});

test('finger robot arm replacement works outside debug mode and preserves saved settings', async () => {
  const { state, hubs } = installHandlers();
  try {
    state.vm.fingerRobotDebug.enabled = false;
    const savedConfig = structuredClone(state.vm.fingerRobot.config);
    state.auth.twoFactorEnabled = true;
    state.auth.twoFactorProtection.targetPower = true;

    const invalid = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({ pulseUs: 1500 })
    });
    assert.equal(invalid.code, -1);

    // Replace-arm is gated like press: with the target_power policy enabled it
    // refuses to move the arm without a one-time challenge.
    const blocked = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({})
    });
    assert.equal(blocked.code, -3);

    const authorization = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
      method: 'POST',
      body: JSON.stringify({ action: 'target_power', code: '000000' })
    });
    assert.equal(authorization.code, 0);

    // Availability is enforced before verification, so an unavailable servo
    // rejects without consuming the one-time challenge.
    state.vm.fingerRobot.available = false;
    const unavailable = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({ challenge: authorization.data.challenge })
    });
    assert.equal(unavailable.code, -5);
    assert.equal(state.vm.fingerRobot.available, false);
    assert.equal(state.auth.targetPowerChallenge, authorization.data.challenge);
    state.vm.fingerRobot.available = true;

    const positioned = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({ challenge: authorization.data.challenge })
    });
    assert.equal(positioned.code, 0);
    assert.equal(state.auth.targetPowerChallenge, null);
    const afterReplacement = await fetchJson(`${origin}/api/vm/finger-robot`);
    assert.equal(afterReplacement.data.available, true);
    assert.equal(afterReplacement.data.status.running, false);
    assert.equal(afterReplacement.data.status.result, 'reset');
    assert.deepEqual(afterReplacement.data.config, savedConfig);

    // The challenge is one-time: reuse is rejected without moving the arm again.
    const reused = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({ challenge: authorization.data.challenge })
    });
    assert.equal(reused.code, -3);
    assert.equal(state.vm.fingerRobot.status.result, 'reset');

    // Release is ungated, even with the target_power policy still enabled.
    const released = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
      method: 'POST',
      body: JSON.stringify({})
    });
    assert.equal(released.code, 0);
    assert.equal(state.vm.fingerRobot.status.result, 'success');
    assert.deepEqual(state.vm.fingerRobot.config, savedConfig);

    // A disabled servo also rejects before consuming a fresh challenge.
    const disabledAuthorization = await fetchJson(
      `${origin}/api/auth/2fa/verify-sensitive-action`,
      {
        method: 'POST',
        body: JSON.stringify({ action: 'target_power', code: '000000' })
      }
    );
    assert.equal(disabledAuthorization.code, 0);
    state.vm.fingerRobot.enabled = false;
    const disabled = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({ challenge: disabledAuthorization.data.challenge })
    });
    assert.equal(disabled.code, -5);
    assert.equal(state.auth.targetPowerChallenge, disabledAuthorization.data.challenge);
    assert.equal(state.vm.fingerRobot.status.result, 'success');
  } finally {
    hubs.dispose();
  }
});

test('finger robot arm replacement cancels a pending normal press and holds its position', { timeout: 10_000 }, async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const { state, hubs } = installHandlers();
  try {
    state.vm.fingerRobotDebug.enabled = false;
    state.vm.fingerRobot.config.pressMs = 30_000;
    const pressing = fetchJson(`${origin}/api/vm/finger-robot/press`, {
      method: 'POST',
      body: JSON.stringify({})
    });
    await setImmediate();
    assert.equal(state.vm.fingerRobot.status.running, true);
    const positioned = await fetchJson(`${origin}/api/vm/finger-robot/replace-arm`, {
      method: 'POST',
      body: JSON.stringify({})
    });
    assert.equal(positioned.code, 0);
    t.mock.timers.runAll();
    assert.equal((await pressing).code, -5);
    const status = await fetchJson(`${origin}/api/vm/finger-robot`);
    assert.deepEqual(status.data.status, {
      running: false,
      state: 'idle',
      result: 'reset',
      error: ''
    });
  } finally {
    hubs.dispose();
  }
});

test('finger robot product and debug mocks share one controller without leaking ownership', { timeout: 10_000 }, async (t) => {
  const { state, hubs } = installHandlers();

  const startedDebug = await fetchJson(`${origin}/api/vm/finger-robot/debug/test`, {
    method: 'POST',
    body: JSON.stringify({
      releaseUs: 2050,
      pressUs: 1900,
      count: 0,
      pressMs: 800,
      pressSpeedUsPerSecond: 1000
    })
  });
  assert.equal(startedDebug.code, 0);
  assert.equal(state.vm.fingerRobot.owner, 'debug');
  assert.equal(state.vm.fingerRobot.operationKind, 'test');

  const productDuringDebug = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.equal(productDuringDebug.data.status.running, true);
  assert.notEqual(productDuringDebug.data.status.state, 'idle');
  assert.equal(productDuringDebug.data.status.result, 'running');
  for (const privateField of ['owner', 'operationKind', 'operationRevision']) {
    assert.equal(
      Object.prototype.hasOwnProperty.call(productDuringDebug.data, privateField),
      false
    );
  }

  const releasedDebugFromProduct = await fetchJson(`${origin}/api/vm/finger-robot/release`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  assert.equal(releasedDebugFromProduct.code, 0);
  assert.equal(state.vm.fingerRobot.owner, 'normal');
  assert.equal(state.vm.fingerRobot.operationKind, 'release');

  const productAfterRelease = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.deepEqual(productAfterRelease.data.status, {
    running: false,
    state: 'idle',
    result: 'success',
    error: ''
  });
  const debugAfterProductRelease = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(debugAfterProductRelease.data.running, false);
  assert.equal(debugAfterProductRelease.data.current, 0);
  assert.equal(debugAfterProductRelease.data.total, 0);
  assert.equal(debugAfterProductRelease.data.state, 'idle');
  assert.equal(debugAfterProductRelease.data.result, 'idle');

  t.mock.timers.enable({ apis: ['setTimeout'] });
  const normalPress = fetchJson(`${origin}/api/vm/finger-robot/press`, {
    method: 'POST',
    body: JSON.stringify({})
  });
  await setImmediate();
  assert.equal(state.vm.fingerRobot.owner, 'normal');
  assert.equal(state.vm.fingerRobot.operationKind, 'press');

  const debugDuringNormalPress = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(debugDuringNormalPress.data.running, false);
  assert.equal(debugDuringNormalPress.data.current, 0);
  assert.equal(debugDuringNormalPress.data.total, 0);
  assert.equal(debugDuringNormalPress.data.state, 'idle');
  assert.equal(debugDuringNormalPress.data.result, 'idle');

  const rejectedDebugStop = await fetchJson(`${origin}/api/vm/finger-robot/debug/stop`, {
    method: 'POST'
  });
  assert.equal(rejectedDebugStop.code, -2);

  const resetDuringNormalPress = await fetchJson(`${origin}/api/vm/finger-robot/debug/reset`, {
    method: 'POST'
  });
  assert.equal(resetDuringNormalPress.code, 0);
  assert.equal(state.vm.fingerRobot.owner, 'debug');
  assert.equal(state.vm.fingerRobot.operationKind, 'reset');
  t.mock.timers.runAll();
  assert.equal((await normalPress).code, -5);

  const productAfterDebugReset = await fetchJson(`${origin}/api/vm/finger-robot`);
  assert.deepEqual(productAfterDebugReset.data.status, {
    running: false,
    state: 'idle',
    result: 'reset',
    error: ''
  });
  const debugAfterReset = await fetchJson(`${origin}/api/vm/finger-robot/debug`);
  assert.equal(debugAfterReset.data.running, false);
  assert.equal(debugAfterReset.data.current, 0);
  assert.equal(debugAfterReset.data.total, 0);
  assert.equal(debugAfterReset.data.state, 'idle');
  assert.equal(debugAfterReset.data.result, 'reset');

  hubs.dispose();
});

test('screen timelapse playback requires and reuses its scoped authorization', async () => {
  const { state, hubs } = installHandlers();

  await fetchJson(`${origin}/api/auth/2fa/enable`, { method: 'POST' });
  state.auth.twoFactorProtection.screenTimelapse = true;

  const blockedDays = await fetchJson(`${origin}/api/ai/screen-timelapse/days`);
  assert.equal(blockedDays.code, -3);
  const blockedVideo = await fetch(`${origin}/api/ai/screen-timelapse/video?date=2026-01-02`);
  assert.equal(blockedVideo.status, 403);

  const blockedAccess = await fetchJson(`${origin}/api/ai/screen-timelapse/access`, {
    method: 'POST'
  });
  assert.equal(blockedAccess.code, -3);

  const playbackAuthorization = await fetchJson(`${origin}/api/auth/2fa/verify-sensitive-action`, {
    method: 'POST',
    body: JSON.stringify({ action: 'screen_timelapse_playback', code: '000000' })
  });
  assert.equal(playbackAuthorization.data.challenge, 'mock-screen-timelapse-playback-challenge');

  const authorizedAccess = await fetchJson(`${origin}/api/ai/screen-timelapse/access`, {
    method: 'POST',
    body: JSON.stringify({ challenge: playbackAuthorization.data.challenge })
  });
  assert.equal(authorizedAccess.code, 0);

  const protectedStatus = await fetchJson(`${origin}/api/auth/2fa/protection`);
  assert.ok(protectedStatus.data.screenTimelapseAuthorizationExpiresIn > 0);

  const reusedAccess = await fetchJson(`${origin}/api/ai/screen-timelapse/access`, {
    method: 'POST'
  });
  assert.equal(reusedAccess.code, 0);
  const authorizedDays = await fetchJson(`${origin}/api/ai/screen-timelapse/days`);
  assert.equal(authorizedDays.data.days.length, 1);
  const authorizedFrames = await fetchJson(
    `${origin}/api/ai/screen-timelapse/frames?date=2026-01-02`
  );
  assert.equal(authorizedFrames.data.frames.length, 2);
  const authorizedSearch = await fetchJson(`${origin}/api/ai/screen-timelapse/search?query=mocked`);
  assert.equal(authorizedSearch.data.count, 2);
  const authorizedVideo = await fetch(`${origin}/api/ai/screen-timelapse/video?date=2026-01-02`);
  assert.equal(authorizedVideo.status, 200);
  const authorizedDelete = await fetchJson(
    `${origin}/api/ai/screen-timelapse/day?date=2026-01-02`,
    { method: 'DELETE' }
  );
  assert.equal(authorizedDelete.code, 0);
  assert.equal(authorizedDelete.data.date, '2026-01-02');
  assert.equal(authorizedDelete.data.deletedBytes, 1024);

  const daysAfterDelete = await fetchJson(`${origin}/api/ai/screen-timelapse/days`);
  assert.equal(daysAfterDelete.data.days.length, 0);

  state.auth.screenTimelapseAuthorizationUntil = 0;
  const expiredDays = await fetchJson(`${origin}/api/ai/screen-timelapse/days`);
  assert.equal(expiredDays.code, -3);
  const expiredVideo = await fetch(`${origin}/api/ai/screen-timelapse/video?date=2026-01-02`);
  assert.equal(expiredVideo.status, 403);
  const reusedChallenge = await fetchJson(`${origin}/api/ai/screen-timelapse/access`, {
    method: 'POST',
    body: JSON.stringify({ challenge: playbackAuthorization.data.challenge })
  });
  assert.equal(reusedChallenge.code, -3);

  state.auth.twoFactorProtection.screenTimelapse = false;
  const policyOffDays = await fetchJson(`${origin}/api/ai/screen-timelapse/days`);
  assert.equal(policyOffDays.data.days.length, 0);
  const policyOffVideo = await fetch(`${origin}/api/ai/screen-timelapse/video?date=2026-01-02`);
  assert.equal(policyOffVideo.status, 200);

  hubs.dispose();
});

test('screen timelapse retention mock preserves revision, preview, and trigger semantics', async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const { state, hubs } = installHandlers();
  const gib = 1024 * 1024 * 1024;

  const initial = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`);
  assert.equal(initial.code, 0);
  assert.equal(initial.data.config.revision, 0);
  assert.equal(initial.data.status.usageBytes, 6 * gib);

  const saved = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`, {
    method: 'PUT',
    body: JSON.stringify({
      revision: 0,
      cleanup: { enabled: true, retentionDays: 3650, maxStorageMiB: 0 }
    })
  });
  assert.equal(saved.code, 0);
  assert.equal(saved.data.changed, true);
  assert.equal(saved.data.cleanupTriggered, true);
  assert.equal(saved.data.config.revision, 1);

  const afterPut = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`);
  assert.deepEqual(afterPut.data.config, saved.data.config);
  assert.equal(afterPut.data.status.running, true);

  const conflict = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`, {
    method: 'PUT',
    body: JSON.stringify({
      revision: 0,
      cleanup: { enabled: true, retentionDays: 3650, maxStorageMiB: 0 }
    })
  });
  assert.equal(conflict.code, -3);
  assert.equal(conflict.data.config.revision, 1);
  assert.equal(conflict.data.changed, false);

  const preview = await fetchJson(`${origin}/api/ai/screen-timelapse/cleanup`, {
    method: 'POST',
    body: JSON.stringify({
      dryRun: true,
      config: { enabled: true, retentionDays: 1, maxStorageMiB: 0 }
    })
  });
  assert.equal(preview.code, 0);
  assert.equal(preview.data.dryRun, true);
  assert.equal(preview.data.wouldDeleteDays, 3);
  assert.equal(preview.data.wouldDeleteBytes, 4 * gib);
  assert.deepEqual(
    preview.data.candidates.map((candidate) => [candidate.date, candidate.reasons]),
    [
      ['2026-07-12', ['expired']],
      ['2026-07-17', ['expired']],
      ['2026-07-19', ['expired']]
    ]
  );
  const afterPreview = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`);
  assert.deepEqual(afterPreview.data.config, saved.data.config);
  assert.equal(state.ai.screenTimelapseRetention.days.length, 4);

  const triggerWithCandidate = await fetchJson(`${origin}/api/ai/screen-timelapse/cleanup`, {
    method: 'POST',
    body: JSON.stringify({
      dryRun: false,
      config: { enabled: true, retentionDays: 1, maxStorageMiB: 0 }
    })
  });
  assert.equal(triggerWithCandidate.code, -1);

  t.mock.timers.runAll();
  const afterSaveWorker = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`);
  assert.equal(afterSaveWorker.data.status.running, false);
  assert.equal(afterSaveWorker.data.status.lastDeletedDays, 0);
  assert.equal(afterSaveWorker.data.status.managedDays, 4);

  state.ai.screenTimelapseRetention.config.cleanup = {
    enabled: true,
    retentionDays: 1,
    maxStorageMiB: 0
  };
  state.auth.twoFactorProtection.screenTimelapse = true;
  await fetchJson(`${origin}/api/auth/2fa/enable`, { method: 'POST' });

  const challengeRequired = await fetchJson(`${origin}/api/ai/screen-timelapse/cleanup`, {
    method: 'POST',
    body: JSON.stringify({ dryRun: false })
  });
  assert.equal(challengeRequired.code, -4);

  const accepted = await fetchJson(`${origin}/api/ai/screen-timelapse/cleanup`, {
    method: 'POST',
    body: JSON.stringify({ dryRun: false, challenge: 'mock-sensitive-action' })
  });
  assert.deepEqual(accepted, { code: 0, msg: 'ok', data: { accepted: true } });

  const running = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`);
  assert.equal(running.data.status.running, true);
  assert.equal(running.data.status.lastTrigger, 'manual');

  t.mock.timers.runAll();
  const complete = await fetchJson(`${origin}/api/ai/screen-timelapse/retention`);
  assert.equal(complete.data.status.running, false);
  assert.equal(complete.data.status.lastDeletedDays, 3);
  assert.equal(complete.data.status.lastDeletedBytes, 4 * gib);
  assert.equal(complete.data.status.usageBytes, 2 * gib);
  assert.equal(complete.data.status.managedDays, 1);
  assert.equal(complete.data.status.lastTrigger, 'manual');

  hubs.dispose();
});

test('screen timelapse recording mock persists QP and reports restarts', async () => {
  const { state, hubs } = installHandlers();

  const initial = await fetchJson(`${origin}/api/ai/screen-timelapse/recording`);
  assert.deepEqual(initial.data, { qp: 42 });

  const updated = await fetchJson(`${origin}/api/ai/screen-timelapse/recording`, {
    method: 'PUT',
    body: JSON.stringify({ qp: 40 })
  });
  assert.deepEqual(updated.data, { qp: 40, changed: true, restarted: true });
  assert.equal(state.ai.screenTimelapseRecording.qp, 40);

  const unchanged = await fetchJson(`${origin}/api/ai/screen-timelapse/recording`, {
    method: 'PUT',
    body: JSON.stringify({ qp: 40 })
  });
  assert.deepEqual(unchanged.data, { qp: 40, changed: false, restarted: false });

  state.ai.screenTimelapseEnabled = false;
  const disabled = await fetchJson(`${origin}/api/ai/screen-timelapse/recording`, {
    method: 'PUT',
    body: JSON.stringify({ qp: 36 })
  });
  assert.deepEqual(disabled.data, { qp: 36, changed: true, restarted: false });

  const belowMinimum = await fetchJson(`${origin}/api/ai/screen-timelapse/recording`, {
    method: 'PUT',
    body: JSON.stringify({ qp: 35 })
  });
  assert.equal(belowMinimum.code, -2);

  const aboveMaximum = await fetchJson(`${origin}/api/ai/screen-timelapse/recording`, {
    method: 'PUT',
    body: JSON.stringify({ qp: 52 })
  });
  assert.equal(aboveMaximum.code, -2);

  hubs.dispose();
});

test('screen timelapse mock retains OCR search results whose video was cleaned', async () => {
  const { hubs } = installHandlers();

  const search = await fetchJson(`${origin}/api/ai/screen-timelapse/search?query=mock`);
  assert.equal(search.code, 0);
  assert.equal(search.data.count, 2);
  assert.equal(search.data.results.length, 2);
  assert.deepEqual(search.data.results[1], {
    capturedAt: '2025-12-28T04:15:00Z',
    date: '2025-12-28',
    snippet: 'Mocked OCR text remains searchable after cleanup',
    available: false,
    unavailableReason: 'missing_video'
  });

  hubs.dispose();
});

test('handlers work on a different Vite host/port and do not match third-party origins', async () => {
  const state = modules.createMockState();
  const hubs = modules.createMockHubs();
  let thirdPartyFallbackHit = false;
  server.resetHandlers(
    ...modules.createHttpHandlers(alternateOrigin, state, hubs),
    http.get('http://example.test/api/application/version', () => {
      thirdPartyFallbackHit = true;
      return HttpResponse.json({ source: 'third-party-fallback' });
    })
  );

  const response = await fetchJson(`${alternateOrigin}/api/application/version`);
  assert.equal(response.code, 0);
  assert.equal(response.data.current, 'v2.0.0-mock');

  const thirdParty = await fetchJson('http://example.test/api/application/version');
  assert.equal(thirdParty.source, 'third-party-fallback');
  assert.equal(thirdPartyFallbackHit, true);
  hubs.dispose();
});

test('transport hubs project deterministic AI and update events', (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] });
  const state = modules.createMockState();
  const hubs = modules.createMockHubs();
  const aiEvents = [];
  const updateEvents = [];

  hubs.addAIClient({ send: (event) => aiEvents.push(event) });
  hubs.addControlClient({ send: (event) => updateEvents.push(JSON.parse(event)) });
  hubs.publishAISnapshot(state);
  hubs.publishAIService(state, 'screenTimelapse', false);
  hubs.publishUpdate(state);

  t.mock.timers.runAll();

  assert.equal(aiEvents[0].event, 'snapshot');
  assert.equal(aiEvents[0].data.services.screenTimelapse.enabled, true);
  assert.deepEqual(aiEvents[1], {
    event: 'service_changed',
    data: {
      type: 'service_changed',
      revision: 2,
      service: 'screenTimelapse',
      enabled: false,
      source: 'mocked'
    }
  });
  assert.deepEqual(
    updateEvents.map((event) => JSON.parse(event.data)),
    [
      { step: 'download', progress: 25 },
      { step: 'download', progress: 75 },
      { step: 'install', progress: 100 },
      { step: 'install', progress: 100 },
      { step: 'install', progress: 100 },
      { step: 'install', progress: 100 },
      { step: 'install', progress: 100 }
    ]
  );

  hubs.dispose();
});
