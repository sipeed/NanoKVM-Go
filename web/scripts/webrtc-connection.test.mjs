import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const jsx = (type, props) => ({ type, props });

function loadModule(relativePath, modules = {}, globals = {}) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      jsx: ts.JsxEmit.ReactJSX
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      console,
      ...globals,
      exports: module.exports,
      module,
      require(specifier) {
        if (specifier === 'react/jsx-runtime') return { jsx, jsxs: jsx };
        assert.ok(specifier in modules, `Unexpected import: ${specifier}`);
        return modules[specifier];
      }
    },
    { filename: file }
  );
  return module.exports;
}

const types = loadModule('src/types/index.ts');
const videoStatus = loadModule('src/pages/desktop/screen/video-status.ts', { '@/types': types });
const atomNames = [
  'videoParametersAtom',
  'videoStatusAtom',
  'videoBandwidthWarningAtom',
  'videoVolumeAtom',
  'remoteScreenAspectRatioAtom',
  'webrtcConnectionFailedAtom'
];
const atoms = Object.fromEntries(atomNames.map((name) => [name, name]));

function createHooks(values) {
  const effects = [];
  const states = [];
  return {
    effects,
    states,
    react: {
      useEffect: (effect) => effects.push(effect),
      useLayoutEffect() {},
      useRef: (current) => ({ current }),
      useState(initial) {
        const index = states.push(initial) - 1;
        return [initial, (value) => (states[index] = value)];
      }
    },
    jotai: {
      useAtomValue: (atom) => values.get(atom),
      useSetAtom: (atom) => (value) => values.set(atom, value),
      useAtom: (atom) => [values.get(atom), (value) => values.set(atom, value)]
    }
  };
}

function mountWebrtc(codec) {
  const values = new Map([
    [atoms.videoParametersAtom, { scale: 1 }],
    [atoms.videoVolumeAtom, 0]
  ]);
  const hooks = createHooks(values);
  const peers = [];
  const timers = new Map();
  let now = 0;
  let timerId = 0;
  const ws = { readyState: 0, close() {} };
  const component = loadModule(
    `src/pages/desktop/screen/${codec}-webrtc.tsx`,
    {
      react: hooks.react,
      jotai: hooks.jotai,
      antd: { Spin: 'Spin' },
      clsx: { __esModule: true, default: () => '' },
      'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
      '@/api/stream.ts': { [codec === 'h264' ? 'webrtcH264' : 'webrtcH265']: () => ws },
      '@/types': types,
      '@/lib/auth-session.ts': { handleAuthSessionClose() {} },
      '@/jotai/screen.ts': atoms,
      '../microphone-context.tsx': { useMicrophone: () => ({ registerAudioTransport() {} }) },
      './audio-volume.ts': { disposeAudioVolumeSynchronization() {} },
      './screen-viewport.tsx': { ScreenViewport: 'ScreenViewport' },
      './use-screen-layout.ts': {
        useScreenLayout: () => ({}),
        usePortalDetectionFromElement() {},
        useScreenSourceSize() {}
      },
      './video-status.ts': videoStatus
    },
    {
      RTCPeerConnection: class {
        connectionState = 'new';
        constructor() {
          peers.push(this);
        }
        close() {
          this.connectionState = 'closed';
          this.onconnectionstatechange?.();
        }
      },
      WebSocket: { CONNECTING: 0, OPEN: 1 },
      setTimeout(callback, delay) {
        timers.set(++timerId, { callback, at: now + delay });
        return timerId;
      },
      clearTimeout: (id) => timers.delete(id),
      setInterval: () => 0,
      clearInterval() {}
    }
  );
  component[codec === 'h264' ? 'H264Webrtc' : 'H265Webrtc']();
  const cleanups = hooks.effects.map((effect) => effect());
  return {
    failed: () => values.get(atoms.webrtcConnectionFailedAtom),
    loading: () => hooks.states[1],
    advance(milliseconds) {
      now += milliseconds;
      for (const [id, timer] of timers) {
        if (timer.at > now) continue;
        timers.delete(id);
        timer.callback();
      }
    },
    connect(state) {
      peers[0].connectionState = state;
      peers[0].onconnectionstatechange();
    },
    receiveStatus(status) {
      ws.onmessage({ data: JSON.stringify({ event: 'video-status', data: String(status) }) });
    },
    cleanup: () => cleanups.forEach((cleanup) => cleanup?.())
  };
}

for (const codec of ['h264', 'h265']) {
  test(`${codec}: capture status does not suppress the connection timeout`, () => {
    const screen = mountWebrtc(codec);
    screen.receiveStatus(types.VideoStatus.Normal);
    screen.advance(9999);
    assert.equal(screen.failed(), false);
    screen.advance(1);
    assert.equal(screen.failed(), true);
    assert.equal(screen.loading(), false);
    screen.receiveStatus(types.VideoStatus.Normal);
    assert.equal(screen.failed(), true);
    screen.cleanup();
  });

  test(`${codec}: an established connection cancels the timeout`, () => {
    const screen = mountWebrtc(codec);
    screen.connect('connected');
    screen.advance(10000);
    assert.equal(screen.failed(), false);
    assert.equal(screen.loading(), false);
    screen.cleanup();
  });

  test(`${codec}: a failed connection immediately stops loading and reports failure`, () => {
    const screen = mountWebrtc(codec);
    screen.connect('failed');
    assert.equal(screen.failed(), true);
    assert.equal(screen.loading(), false);
    screen.cleanup();
  });

  test(`${codec}: a late connection clears the timeout warning`, () => {
    const screen = mountWebrtc(codec);
    screen.advance(10000);
    assert.equal(screen.failed(), true);
    screen.connect('connected');
    assert.equal(screen.failed(), false);
    screen.cleanup();
  });

  test(`${codec}: switching modes clears failure and ignores the old connection`, () => {
    for (const timedOut of [false, true]) {
      const screen = mountWebrtc(codec);
      if (timedOut) screen.advance(10000);
      screen.cleanup();
      screen.connect('failed');
      screen.advance(10000);
      assert.equal(screen.failed(), false);
    }
  });
}

function renderNotice({ failed = false, bandwidth = false, status = types.VideoStatus.Normal }) {
  const hooks = createHooks(
    new Map([
      [atoms.webrtcConnectionFailedAtom, failed],
      [atoms.videoBandwidthWarningAtom, bandwidth],
      [atoms.videoStatusAtom, status]
    ])
  );
  let notice;
  let destroyed = false;
  const { Message } = loadModule(
    'src/pages/desktop/message.tsx',
    {
      react: hooks.react,
      jotai: hooks.jotai,
      antd: {
        message: {
          useMessage: () => [
            { open: (value) => (notice = value), destroy: () => (destroyed = true) },
            null
          ]
        }
      },
      'lucide-react': {
        MonitorXIcon: 'MonitorXIcon',
        MonitorPauseIcon: 'MonitorPauseIcon',
        VideoOffIcon: 'VideoOffIcon'
      },
      'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
      '@/types': types,
      '@/lib/layers.ts': { LAYER_VIDEO_STATUS_MESSAGE: 100 },
      '@/jotai/screen.ts': atoms
    },
    {
      window: { addEventListener() {}, requestAnimationFrame: () => 1 }
    }
  );
  Message();
  hooks.effects.forEach((effect) => effect());
  return { notice, destroyed, text: notice?.content.props.children.props.children };
}

test('the Direct-mode hint persists even when a bandwidth warning is active', () => {
  const result = renderNotice({ failed: true, bandwidth: true });
  assert.equal(result.text, 'screen.webrtcConnectionFailed');
  assert.equal(result.notice.duration, 0);
  assert.equal(result.destroyed, false);
});

test('connection failure takes precedence over capture status', () => {
  const result = renderNotice({ failed: true, status: types.VideoStatus.NoImage });
  assert.equal(result.text, 'screen.webrtcConnectionFailed');
});

test('successful connections preserve the existing video-status messages', () => {
  assert.equal(renderNotice({}).destroyed, true);
  assert.equal(renderNotice({ status: types.VideoStatus.NoImage }).text, 'screen.noSignal');
  assert.equal(renderNotice({ bandwidth: true }).notice.duration, 2);
});
