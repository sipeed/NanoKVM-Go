import assert from 'node:assert/strict';
import fs from 'node:fs';
import { createRequire } from 'node:module';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const sectionRoot = 'src/pages/desktop/menu/settings/remote-network';
const tailscaleRoot = `${sectionRoot}/components/tailscale`;
const require = createRequire(path.join(webRoot, 'package.json'));
const clsxModule = require('clsx');
const clsx = clsxModule.default ?? clsxModule.clsx ?? clsxModule;

function runModule(relativePath, modules) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      esModuleInterop: true,
      jsx: ts.JsxEmit.ReactJSX,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      window: {
        clearTimeout() {},
        setTimeout() {
          return 1;
        }
      },
      require(id) {
        if (id in modules) return modules[id];
        throw new Error(`unexpected module: ${id}`);
      }
    },
    { filename: file }
  );
  return module.exports;
}

function runModuleWithGlobals(relativePath, modules, globals) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      esModuleInterop: true,
      jsx: ts.JsxEmit.ReactJSX,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      require(id) {
        if (id in modules) return modules[id];
        throw new Error(`unexpected module: ${id}`);
      },
      ...globals
    },
    { filename: file }
  );
  return module.exports;
}

const jsxRuntime = {
  Fragment: Symbol('Fragment'),
  jsx: (type, props) => ({ type, props }),
  jsxs: (type, props) => ({ type, props })
};
const translate = (key, options) => {
  if (options?.seconds !== undefined) return `${key}:${options.seconds}`;
  if (options?.time !== undefined) return `${key}:${options.time}`;
  return key;
};
const messages = { useRemoteNetworkMessages: () => translate };

function depsChanged(previous, next) {
  return (
    previous === undefined ||
    next === undefined ||
    next.length !== previous.length ||
    next.some((value, index) => value !== previous[index])
  );
}

function createReactStub() {
  const slots = [];
  let cursor = 0;
  return {
    reset() {
      cursor = 0;
    },
    unmount() {
      for (const slot of slots) {
        slot?.cleanup?.();
        if (slot) slot.cleanup = undefined;
      }
    },
    replayEffects() {
      for (const slot of slots) {
        if (!slot?.effect) continue;
        slot.cleanup?.();
        slot.cleanup = slot.effect() ?? undefined;
      }
    },
    react: {
      useCallback(callback, deps) {
        const index = cursor++;
        const slot = (slots[index] ??= {});
        if (depsChanged(slot.deps, deps)) {
          slot.deps = deps;
          slot.value = callback;
        }
        return slot.value;
      },
      useMemo(factory, deps) {
        const index = cursor++;
        const slot = (slots[index] ??= {});
        if (depsChanged(slot.deps, deps)) {
          slot.deps = deps;
          slot.value = factory();
        }
        return slot.value;
      },
      useRef(initial) {
        const index = cursor++;
        const slot = (slots[index] ??= { value: { current: initial } });
        return slot.value;
      },
      useState(initial) {
        const index = cursor++;
        const slot = (slots[index] ??= {
          value: typeof initial === 'function' ? initial() : initial
        });
        return [
          slot.value,
          (next) => {
            slot.value = typeof next === 'function' ? next(slot.value) : next;
          }
        ];
      },
      useEffect(effect, deps) {
        const index = cursor++;
        const slot = (slots[index] ??= {});
        if (!depsChanged(slot.deps, deps)) return;
        slot.cleanup?.();
        slot.deps = deps;
        slot.effect = effect;
        slot.cleanup = effect() ?? undefined;
      }
    }
  };
}

function findAll(node, predicate, matches = []) {
  if (Array.isArray(node)) {
    for (const child of node) findAll(child, predicate, matches);
    return matches;
  }
  if (!node || typeof node !== 'object') return matches;
  if (predicate(node)) matches.push(node);
  for (const value of Object.values(node.props ?? {})) {
    for (const child of Array.isArray(value) ? value : [value]) findAll(child, predicate, matches);
  }
  return matches;
}

function findByType(tree, type) {
  return findAll(tree, (node) => node.type === type)[0];
}

test('managed claim code omits the renewal invalidation warning', () => {
  const Button = Symbol('Button');
  const Card = Symbol('Card');
  const module = runModule(`${tailscaleRoot}/sipeed-claim-code.tsx`, {
    '@ant-design/icons': {
      CheckOutlined: Symbol('CheckOutlined'),
      CopyOutlined: Symbol('CopyOutlined')
    },
    antd: { Button, Card },
    clsx,
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });

  const tree = module.SipeedClaimCode({
    code: '501530',
    secondsLeft: 60,
    isCopied: false,
    onCopy() {},
    onRenew() {}
  });
  assert.equal(
    findAll(
      tree,
      (node) => node.props?.children === 'settings.tailscale.sipeed.claimRenewInvalidates'
    ).length,
    0
  );
});

const icon = Symbol('Icon');
const managedMode = {
  id: 'sipeed',
  nameKey: 'settings.tailscale.mode.sipeed',
  descriptionKey: 'settings.tailscale.mode.sipeedDesc',
  badgeKey: 'settings.remoteNetwork.mode.experimental',
  Icon: icon,
  recommended: true
};
const officialMode = {
  id: 'official',
  nameKey: 'settings.tailscale.mode.official',
  descriptionKey: 'settings.tailscale.mode.officialDesc',
  Icon: icon
};

test('mode picker renders accessible radio cards and confirmation', () => {
  const { RemoteNetworkModePicker } = runModule(`${sectionRoot}/mode-picker.tsx`, {
    clsx,
    'react/jsx-runtime': jsxRuntime
  });
  const changes = [];
  const tree = RemoteNetworkModePicker({
    legend: 'network',
    name: 'mode',
    options: [
      { id: 'sipeed', label: 'Sipeed', Icon: icon },
      { id: 'official', label: 'Official', Icon: icon }
    ],
    value: 'official',
    onChange: (id) => changes.push(id)
  });
  assert.equal(tree.props.role, 'radiogroup');
  const inputs = findAll(tree, (node) => node.type === 'input');
  assert.equal(inputs.length, 2);
  assert.equal(inputs[1].props.checked, true);
  inputs[0].props.onChange();
  assert.deepEqual(changes, ['sipeed']);
});

function loadView({ modes, initialStatus }) {
  const react = createReactStub();
  const clock = createClock();
  let statusCalls = 0;
  const current = { status: { ...initialStatus } };
  const components = {
    Divider: Symbol('Divider'),
    Header: Symbol('Header'),
    Install: Symbol('Install'),
    Connection: Symbol('Connection'),
    Login: Symbol('Login'),
    Device: Symbol('Device'),
    ChooseMode: Symbol('ChooseMode'),
    SipeedAddDevice: Symbol('SipeedAddDevice'),
    SipeedOverview: Symbol('SipeedOverview')
  };
  const module = runModuleWithGlobals(
    `${tailscaleRoot}/view.tsx`,
    {
      react: react.react,
      antd: { Divider: components.Divider },
      'lucide-react': { LoaderCircleIcon: Symbol('LoaderCircleIcon') },
      '@/api/extensions/tailscale.ts': {
        getStatus: async () => {
          statusCalls++;
          return { code: 0, data: { ...current.status } };
        }
      },
      '../../messages.ts': messages,
      './choose-mode.tsx': { ChooseMode: components.ChooseMode },
      './device.tsx': { Device: components.Device },
      './header.tsx': { Header: components.Header },
      './install.tsx': { Install: components.Install },
      './login.tsx': { Login: components.Login },
      './modes.ts': {
        getDefaultTailscaleModeId: (list) => (list.find((mode) => mode.recommended) ?? list[0]).id,
        getTailscaleModes: () => modes
      },
      './connection.tsx': { Connection: components.Connection },
      './sipeed-add-device.tsx': { SipeedAddDevice: components.SipeedAddDevice },
      './sipeed-overview.tsx': { SipeedOverview: components.SipeedOverview },
      'react/jsx-runtime': jsxRuntime
    },
    { window: clock.window, Date: clock.Date }
  );
  return {
    components,
    clock,
    statusCalls: () => statusCalls,
    unmount: react.unmount,
    setStatus(next) {
      current.status = { ...current.status, ...next };
    },
    render(props) {
      react.reset();
      return module.TailscaleView(props);
    },
    settle: () => new Promise((resolve) => setTimeout(resolve, 0))
  };
}

const baseStatus = {
  state: 'notLogin',
  name: 'nano-kvm',
  ip: '',
  account: '',
  routingAvailable: false,
  advertiseExitNode: false,
  advertiseSubnetRoute: false,
  lanSubnet: '',
  availableModes: ['official', 'sipeed'],
  activeMode: null,
  managedState: 'disabled',
  managedError: '',
  retryAfterSeconds: 0
};
const viewProps = { onBack: () => {}, setLocked: () => {} };

test('official device details omit the duplicate logout action', () => {
  const react = createReactStub();
  const Switch = Symbol('Switch');
  const module = runModule(`${tailscaleRoot}/device.tsx`, {
    react: react.react,
    antd: { Switch },
    '@/api/extensions/tailscale.ts': {},
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate: () => ({ isGateBusy: false, modal: null })
    },
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });
  const tree = module.Device({
    status: { ...baseStatus, state: 'running', activeMode: 'official' },
    onStatusRefresh() {}
  });
  assert.equal(
    findAll(tree, (node) => node.props?.children === 'settings.tailscale.logout').length,
    0
  );
  assert.equal(findAll(tree, (node) => node.type === Switch).length, 3);
  react.unmount();
});

async function mountView(options) {
  const harness = loadView(options);
  harness.render(viewProps);
  await harness.settle();
  harness.render(viewProps);
  return harness;
}

test('CN identity-free status shows mode choice and official flow remains intact', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: baseStatus
  });
  const tree = harness.render(viewProps);
  const picker = findByType(tree, harness.components.ChooseMode);
  assert.ok(picker);
  assert.equal(picker.props.value, 'sipeed');
  picker.props.onChange('official');
  picker.props.onConfirm();
  const official = harness.render(viewProps);
  assert.ok(findByType(official, harness.components.Login));
  assert.equal(findByType(official, harness.components.ChooseMode), undefined);
});

test('failed managed identity returns to the Sipeed and official mode choice', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: {
      ...baseStatus,
      activeMode: 'sipeed',
      managedState: 'degraded',
      managedError: 'connectionFailed'
    }
  });
  const tree = harness.render(viewProps);
  const picker = findByType(tree, harness.components.ChooseMode);
  assert.ok(picker);
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);

  picker.props.onChange('official');
  picker.props.onConfirm();
  const official = harness.render(viewProps);
  assert.ok(findByType(official, harness.components.Login));
  assert.equal(findByType(official, harness.components.SipeedAddDevice), undefined);
});

test('managed enable transition returns to mode choice after failure', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: baseStatus
  });
  const picker = findByType(harness.render(viewProps), harness.components.ChooseMode);
  picker.props.onConfirm();

  let managed = harness.render(viewProps);
  let addDevice = findByType(managed, harness.components.SipeedAddDevice);
  addDevice.props.onStatus({
    ...baseStatus,
    activeMode: 'sipeed',
    managedState: 'enrolling'
  });
  harness.render(viewProps);
  harness.render(viewProps);

  managed = harness.render(viewProps);
  addDevice = findByType(managed, harness.components.SipeedAddDevice);
  addDevice.props.onStatus({
    ...baseStatus,
    activeMode: 'sipeed',
    managedState: 'degraded',
    managedError: 'pairingUnavailable'
  });
  harness.render(viewProps);
  const failed = harness.render(viewProps);
  const failedPicker = findByType(failed, harness.components.ChooseMode);
  assert.ok(failedPicker);
  assert.equal(findByType(failed, harness.components.SipeedAddDevice), undefined);

  failedPicker.props.onConfirm();
  const retry = findByType(harness.render(viewProps), harness.components.SipeedAddDevice);
  assert.equal(retry.props.status.managedError, 'pairingUnavailable');
});

test('active official identity never opens the Sipeed retry page', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: {
      ...baseStatus,
      state: 'running',
      activeMode: 'official',
      managedState: 'degraded',
      managedError: 'pairingUnavailable'
    }
  });
  const tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Device));
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
});

test('Tailscale installation precedes mode selection and managed enablement', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: { ...baseStatus, state: 'notInstall' }
  });
  let tree = harness.render(viewProps);
  let install = findByType(tree, harness.components.Install);
  assert.ok(install);
  assert.equal(install.props.isInstalling, false);
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
  assert.equal(findByType(tree, harness.components.Login), undefined);

  harness.setStatus({ state: 'installing' });
  await install.props.onSuccess();
  tree = harness.render(viewProps);
  install = findByType(tree, harness.components.Install);
  assert.ok(install);
  assert.equal(install.props.isInstalling, true);
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
  assert.equal(findByType(tree, harness.components.Header).props.isBackDisabled, true);

  harness.setStatus({ state: 'notLogin' });
  await install.props.onSuccess();
  tree = harness.render(viewProps);
  const picker = findByType(tree, harness.components.ChooseMode);
  assert.ok(picker);
  assert.equal(findByType(tree, harness.components.Install), undefined);
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);

  picker.props.onConfirm();
  tree = harness.render(viewProps);
  const addDevice = findByType(tree, harness.components.SipeedAddDevice);
  assert.ok(addDevice);
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);

  addDevice.props.onFailure();
  assert.ok(findByType(harness.render(viewProps), harness.components.ChooseMode));
});

test('a stopped managed daemon offers enable for its existing mode', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: {
      ...baseStatus,
      state: 'notRunning',
      activeMode: 'sipeed',
      managedState: 'degraded',
      managedError: 'localAPIUnavailable',
      sipeedDisclaimerAccepted: true
    }
  });
  const tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Connection));
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
});

test('refresh restores connected managed overview from activeMode and managedState', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: {
      ...baseStatus,
      state: 'running',
      ip: '100.64.0.3',
      activeMode: 'sipeed',
      managedState: 'enabledConnected'
    }
  });
  let tree = harness.render(viewProps);
  const overview = findByType(tree, harness.components.SipeedOverview);
  assert.ok(overview);
  assert.equal(overview.props.isAddPageOpen, false);
  assert.equal(findByType(tree, harness.components.Device), undefined);
  assert.equal(findByType(tree, harness.components.Divider).props.style, undefined);
  assert.equal(findAll(tree, (node) => node.props?.className === 'h-10').length, 1);

  overview.props.onAddPageOpenChange(true);
  tree = harness.render(viewProps);
  const addPageHeader = findByType(tree, harness.components.Header);
  assert.equal(addPageHeader.props.title, 'settings.tailscale.sipeed.addDevice');
  assert.equal(addPageHeader.props.status, undefined);
  assert.equal(findByType(tree, harness.components.SipeedOverview).props.isAddPageOpen, true);
  assert.equal(findByType(tree, harness.components.Divider).props.style, undefined);
  assert.equal(findAll(tree, (node) => node.props?.className === 'h-10').length, 1);

  addPageHeader.props.onBack();
  tree = harness.render(viewProps);
  assert.equal(findByType(tree, harness.components.Header).props.title, undefined);
  assert.equal(findByType(tree, harness.components.SipeedOverview).props.isAddPageOpen, false);
});

test('withdrawing consent returns to mode choice and then directly to the disclaimer', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: {
      ...baseStatus,
      state: 'running',
      activeMode: 'sipeed',
      managedState: 'enabledConnected',
      sipeedDisclaimerAccepted: true
    }
  });
  const header = findByType(harness.render(viewProps), harness.components.Header);
  harness.setStatus({
    state: 'notRunning',
    activeMode: null,
    managedState: 'disabled',
    sipeedDisclaimerAccepted: false
  });
  await header.props.onSipeedLoggedOut();
  harness.render(viewProps);
  const picker = findByType(harness.render(viewProps), harness.components.ChooseMode);
  assert.ok(picker);
  picker.props.onConfirm();
  const tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.SipeedAddDevice));
  assert.equal(findByType(tree, harness.components.Connection), undefined);
});

test('official identity removal restores mode picker while official-only builds never show it', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: { ...baseStatus, state: 'running', activeMode: 'official' }
  });
  const device = findByType(harness.render(viewProps), harness.components.Device);
  assert.ok(device);
  harness.setStatus({ state: 'notLogin', activeMode: null });
  await device.props.onStatusRefresh();
  harness.render(viewProps);
  assert.ok(findByType(harness.render(viewProps), harness.components.ChooseMode));

  const officialOnly = await mountView({ modes: [officialMode], initialStatus: baseStatus });
  const tree = officialOnly.render(viewProps);
  assert.ok(findByType(tree, officialOnly.components.Login));
  assert.equal(findByType(tree, officialOnly.components.ChooseMode), undefined);
});

test('managed exit returns to mode selection before offering enable again', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: {
      ...baseStatus,
      state: 'running',
      ip: '100.64.0.3',
      activeMode: 'sipeed',
      managedState: 'enabledConnected',
      sipeedDisclaimerAccepted: true
    }
  });
  let tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.SipeedOverview));
  harness.setStatus({
    state: 'notRunning',
    ip: '',
    activeMode: null,
    managedState: 'disabled',
    managedError: '',
    retryAfterSeconds: 0
  });
  await findByType(tree, harness.components.Connection).props.onSuccess();
  harness.render(viewProps);
  tree = harness.render(viewProps);
  const picker = findByType(tree, harness.components.ChooseMode);
  assert.ok(picker);
  assert.equal(findByType(tree, harness.components.Connection), undefined);
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
  picker.props.onConfirm();
  tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Connection));
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
});

for (const state of ['notRunning', 'notLogin']) {
  test(`Sipeed consent opens directly from ${state} and owns the complete enable transition`, async () => {
    const harness = await mountView({
      modes: [managedMode, officialMode],
      initialStatus: { ...baseStatus, state, sipeedDisclaimerAccepted: false }
    });
    findByType(harness.render(viewProps), harness.components.ChooseMode).props.onConfirm();
    let tree = harness.render(viewProps);
    const setup = findByType(tree, harness.components.SipeedAddDevice);
    assert.ok(setup);
    assert.equal(findByType(tree, harness.components.Connection), undefined);
    assert.equal(findByType(tree, harness.components.Login), undefined);

    const enrolling = {
      ...baseStatus,
      state: 'notRunning',
      activeMode: 'sipeed',
      managedState: 'enrolling',
      sipeedDisclaimerAccepted: true
    };
    setup.props.onStatus(enrolling);
    harness.render(viewProps);
    tree = harness.render(viewProps);
    assert.ok(findByType(tree, harness.components.SipeedAddDevice));
    assert.equal(findByType(tree, harness.components.Connection), undefined);

    setup.props.onStatus({ ...enrolling, state: 'running', managedState: 'enabledConnected' });
    tree = harness.render(viewProps);
    assert.ok(findByType(tree, harness.components.Connection));
    assert.ok(findByType(tree, harness.components.SipeedOverview));
    assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
  });
}

for (const mode of ['official', 'sipeed']) {
  test(`a stopped daemon waits for ${mode} selection and enable before login or enrollment`, async () => {
    const harness = await mountView({
      modes: [managedMode, officialMode],
      initialStatus: { ...baseStatus, state: 'notRunning', sipeedDisclaimerAccepted: true }
    });
    let tree = harness.render(viewProps);
    const picker = findByType(tree, harness.components.ChooseMode);
    assert.ok(picker);
    assert.equal(findByType(tree, harness.components.Connection), undefined);
    picker.props.onChange(mode);
    picker.props.onConfirm();
    tree = harness.render(viewProps);
    const connection = findByType(tree, harness.components.Connection);
    assert.ok(connection);
    assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
    assert.equal(findByType(tree, harness.components.Login), undefined);
    assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);
    harness.setStatus({ state: 'notLogin' });
    await connection.props.onSuccess();
    tree = harness.render(viewProps);
    assert.ok(findByType(tree, harness.components.Connection));
    assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
    assert.ok(
      findByType(tree, harness.components[mode === 'official' ? 'Login' : 'SipeedAddDevice'])
    );
  });
}

test('official-only builds show enable without a mode picker while stopped', async () => {
  const harness = await mountView({
    modes: [officialMode],
    initialStatus: { ...baseStatus, state: 'notRunning' }
  });
  const tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Connection));
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
  assert.equal(findByType(tree, harness.components.Login), undefined);
});

test('official startup keeps refreshing until the retained login is ready', async () => {
  const harness = await mountView({
    modes: [officialMode],
    initialStatus: { ...baseStatus, state: 'notRunning' }
  });
  const connection = findByType(harness.render(viewProps), harness.components.Connection);
  let completed = false;
  const refresh = connection.props.onSuccess('start').then(() => {
    completed = true;
  });
  await harness.settle();
  assert.equal(completed, false);
  harness.clock.advanceBy(1500);
  await harness.settle();
  assert.equal(completed, false);
  assert.equal(findByType(harness.render(viewProps), harness.components.Device), undefined);

  harness.setStatus({ state: 'running', activeMode: 'official', ip: '100.64.0.8' });
  harness.clock.advanceBy(1500);
  await refresh;
  assert.ok(findByType(harness.render(viewProps), harness.components.Device));
  const calls = harness.statusCalls();
  harness.clock.advanceBy(30_000);
  await harness.settle();
  assert.equal(harness.statusCalls(), calls, 'startup polling ends when ready');
  harness.unmount();
});

test('startup refresh times out with feedback and stops requesting after unmount', async () => {
  for (const unmount of [false, true]) {
    const harness = await mountView({
      modes: [officialMode],
      initialStatus: { ...baseStatus, state: 'notRunning' }
    });
    const refresh = findByType(
      harness.render(viewProps),
      harness.components.Connection
    ).props.onSuccess('start');
    await harness.settle();
    const calls = harness.statusCalls();
    if (unmount) harness.unmount();
    harness.clock.advanceBy(30_000);
    await refresh;
    if (unmount) assert.equal(harness.statusCalls(), calls);
    else
      assert.equal(
        findAll(harness.render(viewProps), (node) => node.props?.children === 'tailscale.error')
          .length,
        1
      );
    harness.unmount();
  }
});

test('confirmed picker selection survives a no-op refresh before enable persists', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: baseStatus
  });
  let tree = harness.render(viewProps);
  findByType(tree, harness.components.ChooseMode).props.onConfirm();
  tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.SipeedAddDevice));

  // The enable action has not persisted a managed record yet; a background
  // refresh returning the same activeMode:null status must not yank the user
  // back to the picker.
  await findByType(tree, harness.components.Header).props.onSuccess();
  tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.SipeedAddDevice));
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
});

test('transient official activeMode dropout never flashes the mode picker', async () => {
  const harness = await mountView({
    modes: [managedMode, officialMode],
    initialStatus: { ...baseStatus, state: 'running', activeMode: 'official' }
  });
  let tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Device));

  // A transient profile-read failure reports activeMode:null while running.
  harness.setStatus({ activeMode: null });
  await findByType(tree, harness.components.Header).props.onSuccess();
  tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Device));
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
  assert.equal(findByType(tree, harness.components.SipeedAddDevice), undefined);

  harness.setStatus({ activeMode: 'official' });
  await findByType(tree, harness.components.Header).props.onSuccess();
  tree = harness.render(viewProps);
  assert.ok(findByType(tree, harness.components.Device));
  assert.equal(findByType(tree, harness.components.ChooseMode), undefined);
});

test('managed enable reuses login challenge and synchronously blocks duplicate clicks', async () => {
  const react = createReactStub();
  const Button = Symbol('Button');
  const Card = Symbol('Card');
  let apiCalls = 0;
  const consentArguments = [];
  let resolveRequest;
  const gateCalls = [];
  const received = [];
  const module = runModule(`${tailscaleRoot}/sipeed-add-device.tsx`, {
    react: react.react,
    '@ant-design/icons': { CloudOutlined: Symbol('CloudOutlined') },
    antd: { Button, Card },
    '@/api/extensions/tailscale.ts': {
      enableSipeed(challenge, acceptDisclaimer) {
        apiCalls += 1;
        consentArguments.push([challenge, acceptDisclaimer]);
        return new Promise((resolve) => {
          resolveRequest = resolve;
        });
      }
    },
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate: () => ({
        isGateBusy: false,
        modal: null,
        async runWithChallenge(action, key, execute) {
          gateCalls.push([action, key]);
          return execute('approved');
        }
      })
    },
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });
  react.reset();
  const tree = module.SipeedAddDevice({
    status: baseStatus,
    onStatus: (status) => received.push(status)
  });
  const button = findByType(tree, Button);
  assert.ok(findByType(tree, Card), 'first enable shows the disclaimer');
  assert.equal(button.props.children, 'settings.tailscale.sipeed.acceptAndEnable');
  button.props.onClick();
  button.props.onClick();
  assert.equal(apiCalls, 1);
  assert.deepEqual(consentArguments, [['approved', true]]);
  assert.deepEqual(gateCalls, [['tailscale_login', 'tailscale']]);
  resolveRequest({
    code: 0,
    data: {
      ...baseStatus,
      activeMode: 'sipeed',
      managedState: 'enrolling',
      sipeedDisclaimerAccepted: true
    }
  });
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(received.length, 1);
  react.reset();
  const enabling = module.SipeedAddDevice({ status: received[0], onStatus() {} });
  assert.equal(
    findByType(enabling, Card),
    undefined,
    'saved consent hides the disclaimer immediately'
  );
  // Refreshing status after an attempt cannot trigger a second automatic
  // request. Explicit retries reuse consent; revocation asks for consent again.
  react.reset();
  const refreshed = module.SipeedAddDevice({
    status: { ...baseStatus, sipeedDisclaimerAccepted: true },
    onStatus() {}
  });
  const start = findByType(refreshed, Button);
  assert.equal(
    findByType(refreshed, Card),
    undefined,
    'saved consent hides the disclaimer after refresh'
  );
  assert.equal(apiCalls, 1, 'refresh does not repeat an attempted start');
  assert.equal(start.props.children, 'settings.tailscale.sipeed.retry');
  start.props.onClick();
  assert.deepEqual(consentArguments[1], ['approved', false]);
  resolveRequest({ code: 0 });
  await new Promise((resolve) => setTimeout(resolve, 0));
  react.reset();
  const revoked = module.SipeedAddDevice({
    status: { ...baseStatus, sipeedDisclaimerAccepted: false },
    onStatus() {}
  });
  assert.ok(findByType(revoked, Card), 'withdrawing consent shows the disclaimer again');
  assert.equal(
    findByType(revoked, Button).props.children,
    'settings.tailscale.sipeed.acceptAndEnable'
  );
});

function loadManagedEnable({
  initialStatus = { ...baseStatus, sipeedDisclaimerAccepted: true },
  enable,
  gate
} = {}) {
  const react = createReactStub();
  const Button = Symbol('Button');
  const Card = Symbol('Card');
  const calls = [];
  const gateCalls = [];
  let status = initialStatus;
  const protectionGate = gate ?? {
    isGateBusy: false,
    modal: null,
    async runWithChallenge(action, policy, execute) {
      gateCalls.push([action, policy]);
      return execute('approved');
    }
  };
  const module = runModule(`${tailscaleRoot}/sipeed-add-device.tsx`, {
    react: react.react,
    '@ant-design/icons': {
      CloudOutlined: Symbol('CloudOutlined'),
      LogoutOutlined: Symbol('LogoutOutlined')
    },
    antd: { Button, Card, Popconfirm: Symbol('Popconfirm') },
    '@/api/extensions/tailscale.ts': {
      async enableSipeed(...args) {
        calls.push(args);
        return (
          enable?.(...args) ?? {
            code: 0,
            data: { ...status, activeMode: 'sipeed', managedState: 'enrolling' }
          }
        );
      }
    },
    '@/hooks/useProtectionChallengeGate.tsx': { useProtectionChallengeGate: () => protectionGate },
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });
  const props = {
    onStatus(next) {
      status = next;
    },
    onLoggedOut() {}
  };
  return {
    Button,
    Card,
    calls,
    gateCalls,
    render() {
      react.reset();
      return module.SipeedAddDevice({ ...props, status });
    },
    replayEffects: () => react.replayEffects()
  };
}

test('saved consent starts immediately once without a second start screen', async () => {
  let finish;
  const harness = loadManagedEnable({
    enable: () =>
      new Promise((resolve) => {
        finish = resolve;
      })
  });
  let tree = harness.render();
  assert.deepEqual(harness.calls, [['approved', false]]);
  assert.deepEqual(harness.gateCalls, [['tailscale_login', 'tailscale']]);
  assert.equal(findByType(tree, harness.Card), undefined);
  assert.equal(findByType(tree, harness.Button), undefined);
  assert.equal(findAll(tree, (node) => node.props?.['aria-live'] === 'polite').length, 1);
  harness.replayEffects();
  harness.render();
  assert.equal(harness.calls.length, 1, 'effect replay and rerenders do not start twice');
  finish({
    code: 0,
    data: { ...baseStatus, sipeedDisclaimerAccepted: true, managedState: 'enrolling' }
  });
  await new Promise((resolve) => setTimeout(resolve, 0));
  tree = harness.render();
  assert.equal(harness.calls.length, 1);
  assert.equal(findByType(tree, harness.Button), undefined);
});

test('automatic start respects missing consent, in-flight work, failures and rate limits', () => {
  for (const overrides of [
    { sipeedDisclaimerAccepted: false },
    { activeMode: 'official' },
    { managedState: 'enrolling' },
    { managedState: 'enabledConnecting' },
    { managedState: 'degraded', managedError: 'connectionFailed' },
    { managedState: 'invalidIdentity', managedError: 'invalidToken' },
    { retryAfterSeconds: 30 }
  ]) {
    const harness = loadManagedEnable({
      initialStatus: { ...baseStatus, sipeedDisclaimerAccepted: true, ...overrides }
    });
    const tree = harness.render();
    harness.replayEffects();
    assert.equal(harness.calls.length, 0, JSON.stringify(overrides));
    if (overrides.sipeedDisclaimerAccepted === false) {
      assert.ok(findByType(tree, harness.Card));
      assert.equal(
        findByType(tree, harness.Button).props.children,
        'settings.tailscale.sipeed.acceptAndEnable'
      );
    }
  }
});

test('automatic start failures offer an explicit retry without a retry loop', async () => {
  const harness = loadManagedEnable({ enable: async () => ({ code: -1 }) });
  harness.render();
  await new Promise((resolve) => setTimeout(resolve, 0));
  const tree = harness.render();
  harness.replayEffects();
  assert.equal(harness.calls.length, 1);
  assert.equal(findAll(tree, (node) => node.props?.role === 'alert').length, 1);
  const retry = findByType(tree, harness.Button);
  assert.equal(retry.props.children, 'settings.tailscale.sipeed.retry');
  retry.props.onClick();
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.deepEqual(harness.calls, [
    ['approved', false],
    ['approved', false]
  ]);
});

test('automatic start waits for verification and does not reopen a canceled challenge', async () => {
  let executePending;
  let prompts = 0;
  const gate = {
    isGateBusy: false,
    modal: null,
    async runWithChallenge(action, policy, execute) {
      assert.equal(action, 'tailscale_login');
      assert.equal(policy, 'tailscale');
      prompts += 1;
      gate.isGateBusy = true;
      executePending = execute;
    }
  };
  const harness = loadManagedEnable({ gate });
  harness.render();
  await new Promise((resolve) => setTimeout(resolve, 0));
  let tree = harness.render();
  assert.equal(harness.calls.length, 0);
  assert.equal(findByType(tree, harness.Button), undefined);
  gate.isGateBusy = false;
  tree = harness.render();
  harness.replayEffects();
  assert.equal(prompts, 1);
  assert.equal(harness.calls.length, 0);
  findByType(tree, harness.Button).props.onClick();
  await executePending('verified');
  gate.isGateBusy = false;
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.deepEqual(harness.calls, [['verified', false]]);
  assert.equal(findByType(harness.render(), harness.Button), undefined);
});

test('disclaimer withdrawal confirms, uses logout protection, and permits retry after failure', async () => {
  const react = createReactStub();
  const ButtonBase = Symbol('ButtonBase');
  const Modal = Symbol('Modal');
  const calls = [];
  let resolveRequest;
  let completed = 0;
  const module = runModule(`${tailscaleRoot}/sipeed-revoke-disclaimer.tsx`, {
    react: react.react,
    antd: { Modal },
    '@/components/ui/button-base.tsx': { ButtonBase },
    '@/api/extensions/tailscale.ts': {
      revokeSipeedDisclaimer(challenge) {
        calls.push(challenge);
        return new Promise((resolve) => {
          resolveRequest = resolve;
        });
      }
    },
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate: () => ({
        isGateBusy: false,
        modal: null,
        runWithChallenge(action, key, execute) {
          assert.equal(action, 'tailscale_logout');
          assert.equal(key, 'tailscale');
          return execute('verified');
        }
      })
    },
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });
  const render = () => {
    react.reset();
    return module.SipeedRevokeDisclaimer({
      onSuccess: () => {
        completed += 1;
      }
    });
  };
  let tree = render();
  findByType(tree, ButtonBase).props.onClick();
  tree = render();
  assert.equal(findByType(tree, Modal).props.open, true);
  assert.equal(calls.length, 0);
  const confirm = findByType(tree, Modal).props.onOk;
  confirm();
  confirm();
  assert.deepEqual(calls, ['verified']);
  resolveRequest({ code: -1 });
  await new Promise((resolve) => setTimeout(resolve, 0));
  tree = render();
  assert.equal(completed, 0);
  assert.equal(findByType(tree, Modal).props.open, true);
  assert.equal(findAll(tree, (node) => node.props?.role === 'alert').length, 1);
  findByType(tree, Modal).props.onOk();
  resolveRequest({ code: 0, data: { cloudCleanupFailed: true } });
  await new Promise((resolve) => setTimeout(resolve, 0));
  tree = render();
  assert.equal(completed, 1);
  assert.equal(findByType(tree, Modal).props.open, false);
});

test('managed recovery omits the duplicate exit action', () => {
  const harness = loadManagedEnable({
    initialStatus: {
      ...baseStatus,
      sipeedDisclaimerAccepted: true,
      activeMode: 'sipeed',
      managedState: 'degraded',
      managedError: 'connectionLost'
    }
  });
  const tree = harness.render();
  assert.equal(
    findAll(tree, (node) => node.props?.children === 'settings.tailscale.sipeed.exit').length,
    0
  );
  assert.ok(
    findAll(tree, (node) => node.props?.children === 'settings.tailscale.sipeed.retry').length
  );
});

test('managed enable failure applies structured rate-limit state and countdown', async () => {
  const react = createReactStub();
  const Button = Symbol('Button');
  let received;
  const limitedStatus = {
    ...baseStatus,
    activeMode: 'sipeed',
    managedState: 'degraded',
    managedError: 'rateLimited',
    retryAfterSeconds: 23
  };
  const module = runModule(`${tailscaleRoot}/sipeed-add-device.tsx`, {
    react: react.react,
    '@ant-design/icons': { CloudOutlined: Symbol('CloudOutlined') },
    antd: { Button, Card: Symbol('Card') },
    '@/api/extensions/tailscale.ts': {
      enableSipeed: async () => ({
        code: -1,
        msg: 'managed tailscale enable is rate limited',
        data: limitedStatus
      })
    },
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate: () => ({
        isGateBusy: false,
        modal: null,
        runWithChallenge: (_action, _key, execute) => execute('approved')
      })
    },
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });

  react.reset();
  const initial = module.SipeedAddDevice({
    status: baseStatus,
    onStatus: (status) => {
      received = status;
    }
  });
  findByType(initial, Button).props.onClick();
  await new Promise((resolve) => setTimeout(resolve, 0));
  assert.equal(received.managedError, 'rateLimited');
  assert.equal(received.retryAfterSeconds, 23);

  react.reset();
  const limited = module.SipeedAddDevice({ status: received, onStatus: () => {} });
  const alert = findAll(limited, (node) => node.props?.role === 'alert')[0];
  assert.equal(alert.props.children, 'settings.tailscale.sipeed.rateLimited:23');
  assert.equal(findByType(limited, Button).props.disabled, true);

  react.reset();
  const expired = module.SipeedAddDevice({
    status: { ...received, retryAfterSeconds: 0 },
    onStatus: () => {}
  });
  const expiredAlert = findAll(expired, (node) => node.props?.role === 'alert')[0];
  assert.equal(expiredAlert.props.children, 'settings.tailscale.sipeed.rateLimited:0');
  assert.equal(findByType(expired, Button).props.disabled, false);
});

test('managed failure codes select specific identity, pairing, and local-state messages', () => {
  const react = createReactStub();
  const module = runModule(`${tailscaleRoot}/sipeed-add-device.tsx`, {
    react: react.react,
    '@ant-design/icons': { CloudOutlined: Symbol('CloudOutlined') },
    antd: { Button: Symbol('Button'), Card: Symbol('Card') },
    '@/api/extensions/tailscale.ts': { enableSipeed: async () => ({ code: 0 }) },
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate: () => ({
        isGateBusy: false,
        modal: null,
        runWithChallenge: (_action, _key, execute) => execute('approved')
      })
    },
    '../../messages.ts': messages,
    'react/jsx-runtime': jsxRuntime
  });

  for (const [managedError, messageKey] of [
    ['invalidToken', 'settings.tailscale.sipeed.invalidIdentity'],
    ['pairingUnavailable', 'settings.tailscale.sipeed.pairingUnavailable'],
    ['stateFileInvalid', 'settings.tailscale.sipeed.stateFileInvalid'],
    ['unknownFailure', 'settings.tailscale.sipeed.enableFailed']
  ]) {
    react.reset();
    const tree = module.SipeedAddDevice({
      status: { ...baseStatus, managedState: 'degraded', managedError },
      onStatus: () => {}
    });
    const alert = findAll(tree, (node) => node.props?.role === 'alert')[0];
    assert.equal(alert.props.children, messageKey, managedError);
  }
});

function createClock(initialNow = Date.parse('2026-09-19T08:00:00Z')) {
  let now = initialNow;
  let nextTimer = 1;
  const timers = new Map();

  class ControlledDate extends Date {
    constructor(...args) {
      super(...(args.length === 0 ? [now] : args));
    }

    static now() {
      return now;
    }
  }

  const schedule = (callback, delay, interval) => {
    const id = nextTimer++;
    timers.set(id, { at: now + Number(delay ?? 0), callback, interval });
    return id;
  };
  const fakeWindow = {
    clearInterval(id) {
      timers.delete(id);
    },
    clearTimeout(id) {
      timers.delete(id);
    },
    setInterval(callback, delay) {
      return schedule(callback, delay, Number(delay));
    },
    setTimeout(callback, delay) {
      return schedule(callback, delay, 0);
    }
  };

  return {
    Date: ControlledDate,
    window: fakeWindow,
    advanceBy(milliseconds) {
      const target = now + milliseconds;
      while (true) {
        let nextId;
        let next;
        for (const [id, timer] of timers) {
          if (timer.at <= target && (!next || timer.at < next.at)) {
            nextId = id;
            next = timer;
          }
        }
        if (!next) break;
        now = next.at;
        if (next.interval > 0) {
          next.at += next.interval;
        } else {
          timers.delete(nextId);
        }
        next.callback();
      }
      now = target;
    }
  };
}

function createOverviewHarness({ pendingResponses = [], devices = [] } = {}) {
  const react = createReactStub();
  const clock = createClock();
  const i18nStub = { language: 'en', resolvedLanguage: 'en' };
  const antdLocaleStubs = Object.fromEntries(
    [
      'cs_CZ',
      'da_DK',
      'de_DE',
      'en_US',
      'es_ES',
      'fr_FR',
      'hu_HU',
      'id_ID',
      'it_IT',
      'ja_JP',
      'ko_KR',
      'nb_NO',
      'nl_NL',
      'pl_PL',
      'ru_RU',
      'th_TH',
      'uk_UA',
      'vi_VN',
      'zh_CN',
      'zh_TW'
    ].map((name) => [`antd/locale/${name}`, {}])
  );
  // Load the real i18n locale module so tests exercise the genuine
  // normalizeLanguage/getIntlLocale mapping instead of a re-implemented stub.
  const localeModule = runModuleWithGlobals(
    'src/i18n/locale.ts',
    {
      ...antdLocaleStubs,
      '@/lib/localstorage.ts': { getLanguage: () => null, setLanguage: () => {} }
    },
    {}
  );
  const components = {
    Button: Symbol('Button'),
    Card: Symbol('Card'),
    Divider: Symbol('Divider'),
    Dropdown: Symbol('Dropdown'),
    Input: Symbol('Input'),
    Modal: Symbol('Modal'),
    Popconfirm: Symbol('Popconfirm'),
    SipeedClaimCode: Symbol('SipeedClaimCode')
  };
  const calls = {
    approve: [],
    authorize: [],
    claim: [],
    devices: 0,
    pendingSignals: [],
    reject: [],
    remove: [],
    rename: []
  };
  const pendingQueue = [...pendingResponses];
  const gate = {
    isGateBusy: false,
    modal: null,
    runWithChallenge: (_action, _key, execute) => execute('approved')
  };
  const api = {
    approveSipeedPending: async (...args) => {
      calls.approve.push(args);
      return { code: 0, data: {} };
    },
    authorizeSipeed: async (...args) => {
      calls.authorize.push(args);
      return { code: 0, data: {} };
    },
    getSipeedClaimCode: async (...args) => {
      calls.claim.push(args);
      return {
        code: 0,
        data: {
          code: String(501529 + calls.claim.length),
          expiresAt: new clock.Date(clock.Date.now() + 5000).toISOString(),
          singleUse: true
        }
      };
    },
    getSipeedDevices: async () => {
      calls.devices += 1;
      return { code: 0, data: { devices } };
    },
    getSipeedPending: (signal) => {
      calls.pendingSignals.push(signal);
      if (pendingQueue.length > 0) {
        const nextPending = pendingQueue.shift();
        if (nextPending instanceof Error) return Promise.reject(nextPending);
        return Promise.resolve({ code: 0, data: { pending: nextPending } });
      }
      return new Promise((resolve, reject) => {
        signal.addEventListener(
          'abort',
          () => {
            const error = new Error('cancelled');
            error.name = 'CanceledError';
            reject(error);
          },
          { once: true }
        );
      });
    },
    logoutSipeed: async () => ({ code: 0, data: { cloudCleanupFailed: false } }),
    rejectSipeedPending: async (...args) => {
      calls.reject.push(args);
      return { code: 0, data: {} };
    },
    removeSipeedDevice: async (...args) => {
      calls.remove.push(args);
      return { code: 0, data: {} };
    },
    renameSipeedDevice: async (...args) => {
      calls.rename.push(args);
      return { code: 0, data: {} };
    }
  };
  const module = runModuleWithGlobals(
    `${tailscaleRoot}/sipeed-overview.tsx`,
    {
      react: react.react,
      'react-i18next': { useTranslation: () => ({ i18n: i18nStub }) },
      '@ant-design/icons': {
        CheckOutlined: Symbol('CheckOutlined'),
        CopyOutlined: Symbol('CopyOutlined'),
        LogoutOutlined: Symbol('LogoutOutlined')
      },
      antd: components,
      'lucide-react': {
        CircleCheckIcon: Symbol('CircleCheckIcon'),
        EllipsisIcon: Symbol('EllipsisIcon'),
        LoaderCircleIcon: Symbol('LoaderCircleIcon'),
        PlusIcon: Symbol('PlusIcon')
      },
      '@/api/extensions/tailscale.ts': api,
      '@/hooks/useProtectionChallengeGate.tsx': {
        useProtectionChallengeGate: () => gate
      },
      '@/i18n/locale.ts': localeModule,
      '@/lib/feedback.tsx': { feedback: { error() {}, success() {} } },
      '../../messages.ts': messages,
      './sipeed-claim-code.tsx': { SipeedClaimCode: components.SipeedClaimCode },
      'react/jsx-runtime': jsxRuntime
    },
    {
      AbortController,
      Date: clock.Date,
      navigator: {
        clipboard: { writeText: async () => {} },
        language: 'en-US'
      },
      window: clock.window
    }
  );
  let isAddPageOpen = false;
  const props = {
    status: {
      ...baseStatus,
      state: 'running',
      activeMode: 'sipeed',
      managedState: 'enabledConnected',
      ip: '100.64.0.3'
    },
    onAddPageOpenChange: (isOpen) => {
      isAddPageOpen = isOpen;
    },
    onIdentityExpired: () => {},
    onLoggedOut: () => {}
  };

  return {
    api,
    calls,
    clock,
    components,
    i18n: i18nStub,
    setAddPageOpen(isOpen) {
      isAddPageOpen = isOpen;
    },
    render() {
      react.reset();
      return module.SipeedOverview({ ...props, isAddPageOpen });
    },
    settle: () => new Promise((resolve) => setTimeout(resolve, 0)),
    unmount: react.unmount
  };
}

function buttonsWithLabel(tree, Button, label) {
  return findAll(tree, (node) => node.type === Button && node.props.children === label);
}

test('managed overview balances its divider spacing and uses a compact add-device action', () => {
  const harness = createOverviewHarness();
  const tree = harness.render();
  assert.equal(
    buttonsWithLabel(tree, harness.components.Button, 'settings.tailscale.sipeed.exit').length,
    0
  );
  const balancedDivider = findAll(
    tree,
    (node) => node.type === harness.components.Divider && node.props.style?.margin === '8px 0 0'
  );

  assert.equal(balancedDivider.length, 1);
  assert.equal(
    buttonsWithLabel(tree, harness.components.Button, 'settings.tailscale.sipeed.addDevice')[0]
      .props.size,
    'small'
  );
  harness.unmount();
});

test('managed add-device page updates its countdown and cancels polling when left', async () => {
  const harness = createOverviewHarness();
  let tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();

  tree = harness.render();
  const claim = findByType(tree, harness.components.SipeedClaimCode);
  assert.equal(claim.props.secondsLeft, 5);
  assert.equal(
    findAll(tree, (node) => node.props?.children === 'settings.tailscale.sipeed.devices').length,
    0,
    'the overview is replaced by the add-device page'
  );
  const backupCodeInput = findAll(
    tree,
    (node) =>
      node.type === harness.components.Input &&
      node.props['aria-label'] === 'settings.tailscale.sipeed.backupCode'
  )[0];
  assert.equal(backupCodeInput.props.placeholder, 'settings.tailscale.sipeed.backupCode');
  assert.equal(harness.calls.pendingSignals.length, 1);
  assert.equal(harness.calls.pendingSignals[0].aborted, false);

  harness.clock.advanceBy(2000);
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.secondsLeft, 3);

  harness.clock.advanceBy(3000);
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.secondsLeft, 0);

  harness.setAddPageOpen(false);
  harness.render();
  assert.equal(harness.calls.pendingSignals[0].aborted, true);
  harness.unmount();
});

test('managed approval returns to the overview, refreshes devices and clears the consumed code', async () => {
  const harness = createOverviewHarness({
    pendingResponses: [[{ pendingId: 'pending-1', sas: ['1', '2', '3'], clientUa: 'Laptop' }]]
  });
  let tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  harness.render();
  await harness.settle();
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode), undefined);

  harness.api.approveSipeedPending = async () => ({ code: -1 });
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.pendingAllow'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  assert.equal(
    buttonsWithLabel(tree, harness.components.Button, 'settings.tailscale.sipeed.pendingAllow')
      .length,
    1,
    'a failed approval must keep the pending request available'
  );

  const refreshes = harness.calls.devices;
  harness.api.approveSipeedPending = async () => ({ code: 0 });
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.pendingAllow'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  assert.equal(harness.calls.devices, refreshes + 1);
  assert.equal(
    buttonsWithLabel(tree, harness.components.Button, 'settings.tailscale.sipeed.pendingAllow')
      .length,
    0
  );
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.code, '501531');
  assert.equal(harness.calls.pendingSignals[0].aborted, true);
  harness.unmount();
});

test('managed claim renewal preserves the card and replaces only its code', async () => {
  const harness = createOverviewHarness();
  let tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  assert.deepEqual(harness.calls.claim, [['approved', false]]);
  const backupInput = (currentTree) =>
    findAll(
      currentTree,
      (node) =>
        node.type === harness.components.Input &&
        node.props['aria-label'] === 'settings.tailscale.sipeed.backupCode'
    )[0];
  backupInput(tree).props.onChange({ target: { value: '2345-678A' } });
  tree = harness.render();
  const requestClaim = harness.api.getSipeedClaimCode;
  let finishRenewal;
  harness.api.getSipeedClaimCode = (...args) => {
    assert.deepEqual(args, ['approved', true]);
    return new Promise((resolve) => {
      finishRenewal = () => resolve(requestClaim(...args));
    });
  };
  const renew = findByType(tree, harness.components.SipeedClaimCode).props.onRenew;
  renew();
  const firstRenewal = finishRenewal;
  renew();
  assert.equal(finishRenewal, firstRenewal, 'repeated clicks must share the in-flight renewal');
  tree = harness.render();
  const refreshing = findByType(tree, harness.components.SipeedClaimCode);
  assert.ok(refreshing, 'keep the device-code card mounted while renewal is in flight');
  assert.equal(refreshing.props.code, '', 'hide the invalidated code while renewal is in flight');
  assert.equal(backupInput(tree).props.value, '2345-678A');
  assert.equal(harness.calls.pendingSignals[0].aborted, true);
  assert.equal(refreshing.props.isLoading, true);

  finishRenewal();
  await harness.settle();
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.code, '501531');
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.isLoading, false);
  assert.equal(backupInput(tree).props.value, '2345-678A');
  assert.equal(harness.calls.pendingSignals.length, 2);
  assert.deepEqual(harness.calls.claim, [
    ['approved', false],
    ['approved', true]
  ]);

  harness.api.getSipeedClaimCode = async () => ({ code: -1 });
  findByType(tree, harness.components.SipeedClaimCode).props.onRenew();
  await harness.settle();
  tree = harness.render();
  assert.equal(
    findByType(tree, harness.components.SipeedClaimCode).props.code,
    '',
    'a failed renewal must not restore the previous code'
  );
  const retry = findByType(tree, harness.components.SipeedClaimCode);
  assert.equal(retry.props.isRenewDisabled, false);
  assert.equal(retry.props.isLoading, false);
  harness.api.getSipeedClaimCode = requestClaim;
  retry.props.onRenew();
  await harness.settle();
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.code, '501532');
  assert.deepEqual(harness.calls.claim.at(-1), ['approved', true]);
  harness.unmount();
});

test('a late claim renewal cannot replace the code from a new add-device visit', async () => {
  const harness = createOverviewHarness();
  let tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  const requestClaim = harness.api.getSipeedClaimCode;
  let finishRenewal;
  harness.api.getSipeedClaimCode = () =>
    new Promise((resolve) => {
      finishRenewal = resolve;
    });
  findByType(tree, harness.components.SipeedClaimCode).props.onRenew();
  harness.render();
  harness.setAddPageOpen(false);
  tree = harness.render();
  harness.api.getSipeedClaimCode = requestClaim;
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.code, '501531');

  finishRenewal({
    code: 0,
    data: {
      code: '111111',
      expiresAt: new harness.clock.Date(harness.clock.Date.now() + 5000).toISOString(),
      singleUse: true
    }
  });
  await harness.settle();
  tree = harness.render();
  assert.equal(findByType(tree, harness.components.SipeedClaimCode).props.code, '501531');
  harness.unmount();
});

test('managed claim-code API distinguishes shared reads from explicit renewal', () => {
  const calls = [];
  const api = runModule('src/api/extensions/tailscale.ts', {
    '@/lib/http.ts': { http: { post: (...args) => calls.push(args) } }
  });
  api.getSipeedClaimCode();
  api.getSipeedClaimCode('verified', true);
  assert.equal(calls[0][0], '/api/extensions/tailscale/sipeed/claim-code');
  assert.equal(calls[0][1], undefined);
  assert.equal(calls[1][0], calls[0][0]);
  assert.deepEqual(JSON.parse(JSON.stringify(calls[1][1])), {
    refresh: true,
    challenge: 'verified'
  });
});

test('managed pending polling backs off and recovers after a transient request error', async () => {
  const harness = createOverviewHarness({
    pendingResponses: [new Error('temporary network failure'), []]
  });
  let tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  await harness.settle();

  assert.equal(harness.calls.pendingSignals.length, 1);
  harness.clock.advanceBy(2999);
  await harness.settle();
  assert.equal(harness.calls.pendingSignals.length, 1);
  harness.clock.advanceBy(1);
  await harness.settle();
  assert.equal(harness.calls.pendingSignals.length, 2);

  harness.setAddPageOpen(false);
  harness.render();
  harness.unmount();
});

test('managed pending, backup-code, and device actions execute their real component callbacks', async () => {
  const pending = [
    {
      pendingId: 'pending-1',
      sas: ['6', '6', '5'],
      createdAt: '2026-09-19T08:00:00Z',
      clientUa: 'Laptop'
    },
    {
      pendingId: 'pending-2',
      sas: ['1', '2', '3'],
      createdAt: '2026-09-19T08:00:01Z',
      clientUa: 'Phone'
    }
  ];
  const devices = [
    {
      nodeId: 'self',
      name: 'NanoKVM',
      ips: ['100.64.0.3'],
      online: true,
      isSelf: true
    },
    {
      nodeId: 'peer',
      name: 'Laptop',
      ips: ['100.64.0.4'],
      online: false,
      lastSeen: '2026-09-19T07:48:00Z',
      isSelf: false
    }
  ];
  const harness = createOverviewHarness({ pendingResponses: [pending], devices });
  let tree = harness.render();
  await harness.settle();
  tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();
  await harness.settle();
  tree = harness.render();

  const liveCards = findAll(tree, (node) => node.props?.['aria-live'] === 'polite');
  assert.equal(liveCards.length, 2);
  for (const card of liveCards) {
    assert.equal(findAll(card, (node) => node.props?.['aria-hidden'] === 'true').length, 3);
    assert.ok(
      findAll(card, (node) => node.props?.children === 'settings.tailscale.sipeed.pendingMismatch')
        .length > 0
    );
  }
  // The claim is single-use. Once its pending batch arrives, the code panel
  // closes and the long poll must be cancelled; otherwise the upstream's
  // non-empty response returns immediately and degrades into a short poll.
  assert.equal(
    harness.calls.pendingSignals.length,
    1,
    'polling must stop after the claim code is consumed'
  );
  assert.equal(harness.calls.pendingSignals[0].aborted, true);

  let nameInputs = findAll(
    tree,
    (node) =>
      node.type === harness.components.Input &&
      node.props['aria-label'] === 'settings.tailscale.sipeed.pendingName'
  );
  nameInputs[0].props.onChange({ target: { value: 'work-laptop' } });
  nameInputs[1].props.onChange({ target: { value: 'phone' } });
  tree = harness.render();
  nameInputs = findAll(
    tree,
    (node) =>
      node.type === harness.components.Input &&
      node.props['aria-label'] === 'settings.tailscale.sipeed.pendingName'
  );
  assert.equal(nameInputs[0].props.value, 'work-laptop');
  assert.equal(nameInputs[1].props.value, 'phone');

  const rejectButton = buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.pendingReject'
  )[1];
  assert.equal(rejectButton.props.danger, true);
  rejectButton.props.onClick();
  await harness.settle();
  assert.deepEqual(harness.calls.reject[0], ['pending-2', 'approved']);
  tree = harness.render();
  assert.equal(findAll(tree, (node) => node.props?.['aria-live'] === 'polite').length, 1);
  assert.equal(
    findAll(
      tree,
      (node) =>
        node.type === harness.components.Input &&
        node.props['aria-label'] === 'settings.tailscale.sipeed.pendingName'
    )[0].props.value,
    'work-laptop'
  );

  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.pendingAllow'
  )[0].props.onClick();
  await harness.settle();
  assert.deepEqual(harness.calls.approve[0], ['pending-1', 'work-laptop', 'approved']);
  tree = harness.render();
  assert.equal(findAll(tree, (node) => node.props?.['aria-live'] === 'polite').length, 0);
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.addDevice'
  )[0].props.onClick();
  await harness.settle();
  tree = harness.render();

  let backupInput = findAll(
    tree,
    (node) =>
      node.type === harness.components.Input &&
      node.props['aria-label'] === 'settings.tailscale.sipeed.backupCode'
  )[0];
  backupInput.props.onChange({ target: { value: '1234-5678' } });
  tree = harness.render();
  assert.equal(
    buttonsWithLabel(tree, harness.components.Button, 'settings.tailscale.sipeed.authorize')[0]
      .props.disabled,
    true
  );
  backupInput = findAll(
    tree,
    (node) =>
      node.type === harness.components.Input &&
      node.props['aria-label'] === 'settings.tailscale.sipeed.backupCode'
  )[0];
  backupInput.props.onChange({ target: { value: '2345-678A' } });
  findAll(
    tree,
    (node) =>
      node.type === harness.components.Input &&
      node.props['aria-label'] === 'settings.tailscale.sipeed.pendingName'
  )[0].props.onChange({ target: { value: 'tablet' } });
  tree = harness.render();
  const authorize = buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.authorize'
  )[0];
  assert.equal(authorize.props.disabled, false);
  authorize.props.onClick();
  await harness.settle();
  assert.deepEqual(harness.calls.authorize[0], ['2345-678A', 'tablet', 'approved']);
  tree = harness.render();

  let deviceMenus = findAll(tree, (node) => node.type === harness.components.Dropdown);
  assert.equal(deviceMenus.length, 2);
  assert.equal(deviceMenus[0].props.menu.items[1].disabled, true);
  assert.equal(deviceMenus[1].props.menu.items[1].disabled, false);
  assert.equal(
    findAll(
      tree,
      (node) => node.props?.children === 'settings.tailscale.sipeed.lastSeen:12 minutes ago'
    ).length,
    1
  );

  deviceMenus[1].props.menu.onClick({ key: 'rename' });
  tree = harness.render();
  findAll(
    tree,
    (node) => node.type === harness.components.Input && node.props.autoFocus
  )[0].props.onChange({
    target: { value: 'renamed-laptop' }
  });
  tree = harness.render();
  buttonsWithLabel(
    tree,
    harness.components.Button,
    'settings.tailscale.sipeed.save'
  )[0].props.onClick();
  await harness.settle();
  assert.deepEqual(harness.calls.rename[0], ['peer', 'renamed-laptop', 'approved']);

  tree = harness.render();
  deviceMenus = findAll(tree, (node) => node.type === harness.components.Dropdown);
  deviceMenus[1].props.menu.onClick({ key: 'remove' });
  tree = harness.render();
  const removeModal = findByType(tree, harness.components.Modal);
  assert.equal(removeModal.props.open, true);
  assert.equal(
    removeModal.props.children.props.children,
    'settings.tailscale.sipeed.removeConfirm'
  );
  removeModal.props.onOk();
  await harness.settle();
  assert.deepEqual(harness.calls.remove[0], ['peer', 'approved']);
  harness.unmount();
});

test('last-seen relative time follows the app language, not the browser language', async () => {
  const devices = [
    {
      nodeId: 'self',
      name: 'NanoKVM',
      ips: ['100.64.0.3'],
      online: true,
      isSelf: true
    },
    {
      nodeId: 'peer',
      name: 'Laptop',
      ips: ['100.64.0.4'],
      online: false,
      lastSeen: '2026-09-19T07:48:00Z',
      isSelf: false
    }
  ];
  const harness = createOverviewHarness({ devices });
  const lastSeenNode = (tree, text) =>
    findAll(tree, (node) => node.props?.children === `settings.tailscale.sipeed.lastSeen:${text}`);

  let tree = harness.render();
  await harness.settle();
  tree = harness.render();
  // Baseline: app language en with the browser stub pinned to en-US.
  assert.equal(lastSeenNode(tree, '12 minutes ago').length, 1);

  // Switch the app language while the browser language stays en-US: the
  // relative time must re-render in the app-selected language.
  harness.i18n.language = 'zh';
  harness.i18n.resolvedLanguage = 'zh';
  tree = harness.render();
  assert.equal(lastSeenNode(tree, '12分钟前').length, 1);
  assert.equal(lastSeenNode(tree, '12 minutes ago').length, 0);
  harness.unmount();
});

test('Tailscale header keeps back and management actions without duplicate connection controls', () => {
  const components = {
    ButtonBase: Symbol('ButtonBase'),
    Popover: Symbol('Popover'),
    Uninstall: Symbol('Uninstall'),
    SipeedRevokeDisclaimer: Symbol('SipeedRevokeDisclaimer')
  };
  const module = runModule(`${tailscaleRoot}/header.tsx`, {
    antd: { Popover: components.Popover },
    'lucide-react': {
      ArrowUpRightIcon: Symbol('ArrowUpRightIcon'),
      BookOpenIcon: Symbol('BookOpenIcon'),
      ChevronLeftIcon: Symbol('ChevronLeftIcon'),
      EllipsisIcon: Symbol('EllipsisIcon')
    },
    '@/components/icons/tailscale': { Tailscale: Symbol('TailscaleIcon') },
    '@/components/ui/button-base.tsx': { ButtonBase: components.ButtonBase },
    '../../messages.ts': messages,
    './uninstall.tsx': { Uninstall: components.Uninstall },
    './sipeed-revoke-disclaimer.tsx': { SipeedRevokeDisclaimer: components.SipeedRevokeDisclaimer },
    'react/jsx-runtime': jsxRuntime
  });
  let backCalls = 0;
  const props = {
    status: { ...baseStatus, state: 'running' },
    isBackDisabled: false,
    onBack: () => {
      backCalls += 1;
    },
    onSuccess() {}
  };
  let tree = module.Header(props);
  findByType(tree, components.ButtonBase).props.onClick();
  assert.equal(backCalls, 1);
  assert.ok(findByType(tree, components.Uninstall));
  assert.equal(findByType(tree, components.SipeedRevokeDisclaimer), undefined);
  props.status = { ...props.status, activeMode: 'sipeed', managedState: 'enabledConnected' };
  tree = module.Header(props);
  const menu = findByType(tree, components.Popover).props.content;
  assert.equal(menu.props.children[0].type, components.Uninstall);
  assert.equal(menu.props.children[1].type, components.SipeedRevokeDisclaimer);
  props.isBackDisabled = true;
  tree = module.Header(props);
  assert.equal(findByType(tree, components.ButtonBase).props.disabled, true);
  assert.equal(findByType(tree, components.Popover).props.trigger.length, 0);
});

function loadConnection({
  mode = 'official',
  status = { ...baseStatus, state: 'notRunning' },
  deferredGate = false,
  response = { code: 0 }
} = {}) {
  const react = createReactStub();
  const TunnelConnection = Symbol('TunnelConnection');
  const calls = [];
  const challenges = [];
  let refreshed = 0;
  let pending;
  const model = runModule(`${tailscaleRoot}/status-model.ts`, {});
  const module = runModule(`${tailscaleRoot}/connection.tsx`, {
    react: react.react,
    '@/api/extensions/tailscale.ts': Object.fromEntries(
      ['start', 'stop', 'restart', 'enableSipeed'].map((action) => [
        action,
        async (...args) => {
          calls.push([action, ...args]);
          if (response instanceof Error) throw response;
          return response;
        }
      ])
    ),
    '@/hooks/useProtectionChallengeGate.tsx': {
      useProtectionChallengeGate: () => ({
        isGateBusy: false,
        modal: null,
        async runWithChallenge(action, key, execute) {
          challenges.push([action, key]);
          if (deferredGate) pending = execute;
          else return execute('approved');
        }
      })
    },
    '../../messages.ts': messages,
    '../../tunnel-controls.tsx': { TunnelConnection },
    './status-model.ts': model,
    'react/jsx-runtime': jsxRuntime
  });
  const props = {
    mode,
    status,
    onStatus() {},
    onBusyChange() {},
    onSuccess: async () => {
      refreshed++;
    }
  };
  return {
    calls,
    challenges,
    TunnelConnection,
    refreshed: () => refreshed,
    confirm: () => pending('approved'),
    render(nextStatus = status) {
      react.reset();
      return module.Connection({ ...props, status: nextStatus });
    },
    settle: () => new Promise((resolve) => setTimeout(resolve, 0))
  };
}

test('shared Tailscale enable, stop, and restart use their protected lifecycle endpoints', async () => {
  const harness = loadConnection();
  for (const action of ['start', 'stop', 'restart']) {
    const tree = harness.render();
    const connection = findByType(tree, harness.TunnelConnection);
    connection.props.onAction(action);
    connection.props.onAction(action);
    await harness.settle();
    assert.equal(findByType(harness.render(), harness.TunnelConnection).props.busy, false);
  }
  assert.deepEqual(harness.calls, [
    ['start', 'approved'],
    ['stop', 'approved'],
    ['restart', 'approved']
  ]);
  assert.deepEqual(harness.challenges, [
    ['tailscale_start', 'tailscale'],
    ['tailscale_stop', 'tailscale'],
    ['tailscale_restart', 'tailscale']
  ]);
  assert.equal(harness.refreshed(), 3);
});

test('enable completes after deferred two-factor verification and does not get stuck busy', async () => {
  const harness = loadConnection({ deferredGate: true });
  findByType(harness.render(), harness.TunnelConnection).props.onAction('start');
  await harness.settle();
  assert.equal(harness.calls.length, 0);
  await harness.confirm();
  assert.deepEqual(harness.calls, [['start', 'approved']]);
  assert.equal(harness.refreshed(), 1);
  assert.equal(findByType(harness.render(), harness.TunnelConnection).props.busy, false);
});

test('selected Sipeed starts its managed network instead of resuming the official daemon', async () => {
  const harness = loadConnection({ mode: 'sipeed' });
  findByType(harness.render(), harness.TunnelConnection).props.onAction('start');
  await harness.settle();
  assert.deepEqual(harness.calls, [['enableSipeed', 'approved']]);
  assert.deepEqual(harness.challenges, [['tailscale_login', 'tailscale']]);
  assert.equal(harness.refreshed(), 1);
});

test('Sipeed enable only sends consent and the protection challenge', async () => {
  const calls = [];
  const api = runModule('src/api/extensions/tailscale.ts', {
    '@/lib/http.ts': {
      http: {
        post: (...args) => {
          calls.push(args);
          return Promise.resolve({ code: 0 });
        }
      }
    }
  });
  await api.enableSipeed('approved', true);
  assert.equal(calls[0][0], '/api/extensions/tailscale/sipeed/enable');
  assert.deepEqual(JSON.parse(JSON.stringify(calls[0][1])), {
    acceptDisclaimer: true,
    challenge: 'approved'
  });
});

test('failed enable remains available and displays an error without refreshing', async () => {
  for (const response of [{ code: -1 }, new Error('offline')]) {
    const harness = loadConnection({ response });
    findByType(harness.render(), harness.TunnelConnection).props.onAction('start');
    await harness.settle();
    const tree = harness.render();
    assert.equal(findByType(tree, harness.TunnelConnection).props.busy, false);
    assert.equal(harness.refreshed(), 0);
    assert.ok(findAll(tree, (node) => node.props?.role === 'alert').length);
  }
});

test('connection control reflects stopped, logged-out, connected, and managed transition states', () => {
  const harness = loadConnection();
  for (const [next, active, state, tone] of [
    [{ state: 'notRunning' }, false, 'stopped', 'off'],
    [{ state: 'notLogin' }, true, 'notLogin', 'off'],
    [{ state: 'stopped' }, true, 'offline', 'off'],
    [{ state: 'running' }, true, 'connected', 'on'],
    [{ state: 'notLogin', managedState: 'enrolling' }, true, 'connecting', 'busy'],
    [
      { state: 'notLogin', managedState: 'degraded', managedError: 'connectionLost' },
      true,
      'error',
      'bad'
    ]
  ]) {
    const control = findByType(
      harness.render({ ...baseStatus, ...next }),
      harness.TunnelConnection
    );
    assert.equal(control.props.active, active);
    assert.equal(control.props.state, state);
    assert.equal(control.props.tone, tone);
  }
});
