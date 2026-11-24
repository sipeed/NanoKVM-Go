import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const pointerModeFile = path.resolve(import.meta.dirname, '../src/lib/pointer-mode.ts');
const controllerFile = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/pointer-mode-controller.tsx'
);
const selectionFile = path.resolve(import.meta.dirname, '../src/hooks/usePointerModeSelection.ts');
const modeOptionsFile = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/menu/mouse/mouse-mode-options.tsx'
);
const apiFile = path.resolve(import.meta.dirname, '../src/api/hid.ts');
const websocketFile = path.resolve(import.meta.dirname, '../src/lib/websocket.ts');

function loadPointerMode(getPointerMode = async () => ({ code: 0, data: {} })) {
  const source = fs.readFileSync(pointerModeFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: pointerModeFile
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      Event,
      setTimeout,
      require(specifier) {
        if (specifier === '@/api/hid.ts') {
          return {
            POINTER_MODES: ['absolute', 'digitizer-stylus', 'touch-screen'],
            getPointerMode
          };
        }
        throw new Error(`Unexpected require(${specifier})`);
      }
    },
    { filename: pointerModeFile }
  );

  return module.exports;
}

function pointerStatus(mode, overrides = {}) {
  return {
    mode,
    switching: false,
    degraded: false,
    ready: true,
    error: '',
    ...overrides
  };
}

function pointerSelectionActions(options = {}) {
  const state = {
    events: [],
    appliedModes: [],
    statuses: [],
    inputEnabled: [],
    postModes: [],
    errors: [],
    syncRequests: 0,
    ensureConnected: 0
  };
  const actions = {
    applyMouseMode(mode) {
      state.events.push(`mode:${mode}`);
      state.appliedModes.push(mode);
    },
    publishStatus(status) {
      state.events.push(`status:${status.mode}:${status.switching}:${status.ready}`);
      state.statuses.push(JSON.parse(JSON.stringify(status)));
    },
    beginOperation() {
      state.events.push('begin');
    },
    endOperation() {
      state.events.push('end');
    },
    setInputEnabled(enabled) {
      state.events.push(`input:${enabled}`);
      state.inputEnabled.push(enabled);
    },
    ensureConnected() {
      state.events.push('ensure-connect');
      state.ensureConnected += 1;
    },
    reconnectRelative() {
      state.events.push('reconnect-relative');
    },
    async postPointerMode(mode) {
      state.events.push(`post:${mode}`);
      state.postModes.push(mode);
      if (options.postError) throw options.postError;
      return options.postResponse ?? { code: 0, msg: 'success' };
    },
    async pollStatus(publish) {
      state.events.push('poll');
      for (const status of options.pollUpdates ?? []) publish(status);
      return options.finalStatus ?? null;
    },
    requestSync() {
      state.events.push('sync');
      state.syncRequests += 1;
    },
    reportError(error) {
      state.events.push(`error:${error}`);
      state.errors.push(error);
    },
    fallbackError: 'generic error'
  };
  return { actions, state };
}

test('accepts only canonical device pointer modes', () => {
  const { isPointerMode } = loadPointerMode();
  for (const mode of ['absolute', 'digitizer-stylus', 'touch-screen']) {
    assert.equal(isPointerMode(mode), true);
  }
  for (const mode of ['relative', 'absolute-android', 'unknown', '', null]) {
    assert.equal(isPointerMode(mode), false);
  }
});

test('normalizes healthy, switching, and degraded backend states', () => {
  const { normalizePointerModeStatus } = loadPointerMode();
  const plain = (value) => JSON.parse(JSON.stringify(value));
  assert.deepEqual(
    plain(normalizePointerModeStatus({ mode: 'digitizer-stylus', switching: false })),
    {
      mode: 'digitizer-stylus',
      switching: false,
      degraded: false,
      ready: true,
      error: ''
    }
  );
  assert.equal(normalizePointerModeStatus({ mode: 'absolute', switching: true }).ready, false);
  assert.deepEqual(
    plain(
      normalizePointerModeStatus({
        mode: 'unknown',
        switching: false,
        degraded: true,
        error: 'rollback failed'
      })
    ),
    {
      mode: 'unknown',
      switching: false,
      degraded: true,
      ready: false,
      error: 'rollback failed'
    }
  );
});

test('initial reconciliation follows GET without turning a local preference into a mutation', () => {
  const { resolveAuthoritativeMouseMode } = loadPointerMode();
  assert.equal(
    resolveAuthoritativeMouseMode('digitizer-stylus', pointerStatus('absolute')),
    'absolute'
  );
  assert.equal(
    resolveAuthoritativeMouseMode('relative', pointerStatus('touch-screen')),
    'relative'
  );
  assert.equal(
    resolveAuthoritativeMouseMode(
      'digitizer-stylus',
      pointerStatus('absolute', { ready: false, switching: true })
    ),
    'digitizer-stylus'
  );
});

test('polling tolerates a disconnect and waits for a terminal status', async () => {
  const { pollPointerModeStatus } = loadPointerMode();
  const results = [
    new Error('disconnected'),
    { mode: 'absolute', switching: true, degraded: false, ready: false, error: '' },
    {
      mode: 'touch-screen',
      switching: false,
      degraded: false,
      ready: true,
      error: ''
    }
  ];
  const published = [];
  const sleeps = [];
  const finalStatus = await pollPointerModeStatus((status) => published.push(status), {
    attempts: 3,
    intervalMs: 7,
    getStatus: async () => {
      const value = results.shift();
      if (value instanceof Error) throw value;
      return value;
    },
    sleep: async (milliseconds) => sleeps.push(milliseconds)
  });

  assert.equal(finalStatus.mode, 'touch-screen');
  assert.deepEqual(sleeps, [7, 7]);
  assert.deepEqual(
    published.map((status) => [status.mode, status.switching, status.ready]),
    [
      ['absolute', true, false],
      ['touch-screen', false, false]
    ]
  );
});

test('same HID2 mode and relative mode never POST a pointer profile', async () => {
  const { performPointerModeSelection } = loadPointerMode();

  const same = pointerSelectionActions();
  assert.equal(
    await performPointerModeSelection(
      'absolute',
      pointerStatus('absolute'),
      'absolute',
      same.actions
    ),
    true
  );
  assert.deepEqual(same.state.postModes, []);
  assert.deepEqual(same.state.appliedModes, ['absolute']);
  assert.doesNotMatch(same.state.events.join(','), /input|poll|post/);

  const relative = pointerSelectionActions();
  assert.equal(
    await performPointerModeSelection(
      'touch-screen',
      pointerStatus('touch-screen'),
      'relative',
      relative.actions
    ),
    true
  );
  assert.deepEqual(relative.state.postModes, []);
  assert.deepEqual(relative.state.appliedModes, ['relative']);
  assert.deepEqual(relative.state.events, ['mode:relative', 'reconnect-relative']);
});

test('all three HID2 selections gate input, POST, and confirm by GET', async () => {
  const { performPointerModeSelection } = loadPointerMode();
  for (const [current, target] of [
    ['digitizer-stylus', 'absolute'],
    ['absolute', 'digitizer-stylus'],
    ['absolute', 'touch-screen']
  ]) {
    const finalStatus = pointerStatus(target);
    const { actions, state } = pointerSelectionActions({
      pollUpdates: [pointerStatus(current, { switching: true, ready: false })],
      finalStatus
    });
    assert.equal(
      await performPointerModeSelection(current, pointerStatus(current), target, actions),
      true
    );
    assert.deepEqual(state.postModes, [target]);
    assert.deepEqual(state.appliedModes, [target]);
    assert.deepEqual(state.inputEnabled, [false]);
    assert.equal(state.ensureConnected, 1);
    assert.deepEqual(state.errors, []);
    assert.equal(state.events[0], 'begin');
    assert.equal(state.events.at(-1), 'end');
    assert.equal(state.events.at(-2), 'ensure-connect');
    assert.ok(state.events.indexOf('input:false') < state.events.indexOf(`post:${target}`));
    assert.ok(state.events.indexOf(`post:${target}`) < state.events.indexOf(`mode:${target}`));
    assert.doesNotMatch(state.events.join(','), /release|disconnect/);
  }
});

test('lost POST response still succeeds when GET confirms the target', async () => {
  const { performPointerModeSelection } = loadPointerMode();
  const { actions, state } = pointerSelectionActions({
    postError: new Error('connection reset'),
    finalStatus: pointerStatus('touch-screen')
  });
  assert.equal(
    await performPointerModeSelection(
      'absolute',
      pointerStatus('absolute'),
      'touch-screen',
      actions
    ),
    true
  );
  assert.deepEqual(state.appliedModes, ['touch-screen']);
  assert.deepEqual(state.inputEnabled, [false]);
  assert.equal(state.ensureConnected, 1);
  assert.deepEqual(state.errors, []);
});

test('verified rollback restores the previous mode and reports the backend failure', async () => {
  const { performPointerModeSelection } = loadPointerMode();
  const { actions, state } = pointerSelectionActions({
    postResponse: { code: -5, msg: 'USB rebuild failed' },
    finalStatus: pointerStatus('absolute')
  });
  assert.equal(
    await performPointerModeSelection(
      'absolute',
      pointerStatus('absolute'),
      'touch-screen',
      actions
    ),
    false
  );
  assert.deepEqual(state.appliedModes, ['absolute']);
  assert.deepEqual(state.inputEnabled, [false]);
  assert.equal(state.ensureConnected, 1);
  assert.deepEqual(state.errors, ['USB rebuild failed']);
  assert.equal(state.statuses.at(-1).mode, 'absolute');
});

test('degraded result remains input-gated and exposes an error', async () => {
  const { performPointerModeSelection } = loadPointerMode();
  const degraded = pointerStatus('unknown', {
    degraded: true,
    ready: false,
    error: 'rollback failed'
  });
  const { actions, state } = pointerSelectionActions({ finalStatus: degraded });
  assert.equal(
    await performPointerModeSelection(
      'absolute',
      pointerStatus('absolute'),
      'digitizer-stylus',
      actions
    ),
    false
  );
  assert.deepEqual(state.appliedModes, []);
  assert.deepEqual(state.inputEnabled, [false]);
  assert.equal(state.ensureConnected, 1);
  assert.deepEqual(state.errors, ['rollback failed']);
  assert.equal(state.statuses.at(-1).mode, 'unknown');
  assert.equal(state.statuses.at(-1).degraded, true);
});

test('uninitialized state requests a fresh GET without posting', async () => {
  const { performPointerModeSelection } = loadPointerMode();
  const { actions, state } = pointerSelectionActions();
  assert.equal(
    await performPointerModeSelection(
      'absolute',
      pointerStatus(null, { ready: false }),
      'touch-screen',
      actions
    ),
    false
  );
  assert.deepEqual(state.postModes, []);
  assert.equal(state.syncRequests, 1);
  assert.deepEqual(state.errors, ['generic error']);
});

test('iPhone repair is available only for enabled absolute input', () => {
  const { canRepairIphoneDrag } = loadPointerMode();
  assert.equal(canRepairIphoneDrag('absolute', true), true);
  for (const mode of ['absolute', 'digitizer-stylus', 'touch-screen', 'relative']) {
    assert.equal(canRepairIphoneDrag(mode, false), false);
  }
  for (const mode of ['digitizer-stylus', 'touch-screen', 'relative']) {
    assert.equal(canRepairIphoneDrag(mode, true), false);
  }
});

test('browser mode selection keeps relative local and HID2 modes device-authoritative', () => {
  const pointerModeSource = fs.readFileSync(pointerModeFile, 'utf8');
  const apiSource = fs.readFileSync(apiFile, 'utf8');
  const websocketSource = fs.readFileSync(websocketFile, 'utf8');
  const controller = fs.readFileSync(controllerFile, 'utf8');
  const selection = fs.readFileSync(selectionFile, 'utf8');
  const options = fs.readFileSync(modeOptionsFile, 'utf8');

  assert.match(controller, /fetchPointerModeStatus\(\)/);
  assert.match(apiSource, /getPointerMode\(\)/);
  assert.match(apiSource, /setPointerMode\(mode: MouseMode, singleEndpoint/);
  assert.match(apiSource, /\/api\/hid\/pointer-mode/);
  assert.match(controller, /resolveAuthoritativeMouseMode/);
  assert.match(controller, /client\.onConnectionState/);
  assert.match(websocketSource, /onConnectionState/);
  assert.match(websocketSource, /notifyConnectionState\('open'\)/);
  assert.match(controller, /document\.addEventListener\('visibilitychange'/);
  assert.match(selection, /postPointerMode: api\.setPointerMode/);
  assert.match(selection, /performPointerModeSelection/);
  assert.match(selection, /pollPointerModeStatus/);
  assert.match(selection, /setInputEnabled: setBaseInputEnabled/);
  assert.doesNotMatch(pointerModeSource, /releaseBrowserInput|actions\.(?:disconnect|connect)/);
  assert.doesNotMatch(selection, /\b(?:disconnect|connect)\s*:/);
  assert.equal(selection.match(/client\.close\(\)/g)?.length, 1);
  assert.equal(selection.match(/client\.connect\(\)/g)?.length, 2);
  assert.match(selection, /ensureConnected\(\)[\s\S]*!client\.isConnected/);
  assert.match(controller, /!localOperation/);
  assert.match(options, /requestPointerModeSync\(\)/);
  assert.doesNotMatch(options, /pointerMode\./);
  assert.doesNotMatch(controller, /Modal\.confirm/);

  const localeRoot = path.resolve(import.meta.dirname, '../src/i18n/locales');
  for (const locale of fs.readdirSync(localeRoot).filter((file) => file.endsWith('.ts'))) {
    assert.doesNotMatch(
      fs.readFileSync(path.join(localeRoot, locale), 'utf8'),
      /pointerMode\s*:/,
      `${locale} unexpectedly adds a pointerMode translation block`
    );
  }
});

test('single-pointer selection switches the device and overrides browser-local relative mode', async () => {
  const { performPointerModeSelection, resolveAuthoritativeMouseMode } = loadPointerMode();
  const absolute = { ...pointerStatus('absolute'), singleEndpoint: true };
  const relative = { ...pointerStatus('relative'), singleEndpoint: true };
  assert.equal(resolveAuthoritativeMouseMode('relative', absolute), 'absolute');
  const { actions, state } = pointerSelectionActions({ finalStatus: relative });
  assert.equal(await performPointerModeSelection('absolute', absolute, 'relative', actions), true);
  assert.deepEqual(state.postModes, ['relative']);
  assert.deepEqual(state.appliedModes, ['relative']);
});
