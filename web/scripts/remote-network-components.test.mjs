import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const sectionRoot = 'src/pages/desktop/menu/settings/remote-network';

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

/** Drop block and line comments so guards only inspect executable code. */
function withoutComments(text) {
  return text.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^[ \t]*\/\/.*$/gm, '');
}

function runModule(relativePath, modules) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(source(relativePath), {
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
        if (!(id in modules)) throw new Error(`unexpected module: ${id}`);
        return modules[id];
      }
    },
    { filename: file }
  );

  return module.exports;
}

const jsxRuntime = {
  Fragment: Symbol('Fragment'),
  jsx(type, props) {
    return { type, props };
  },
  jsxs(type, props) {
    return { type, props };
  }
};

const MessagesProvider = Symbol('RemoteNetworkMessagesProvider');

/**
 * Minimal React stub: ordered state slots, effects that respect dependencies and
 * run their cleanup on unmount, plus pass-through callbacks.
 */
function createReactStub() {
  const stateSlots = [];
  const effectSlots = [];
  let cursor = 0;

  return {
    reset() {
      cursor = 0;
    },
    unmount() {
      for (const slot of effectSlots) slot?.cleanup?.();
    },
    replayEffects() {
      for (const slot of effectSlots) {
        if (!slot) continue;
        slot.cleanup?.();
        slot.cleanup = slot.effect() ?? undefined;
      }
    },
    react: {
      useCallback(callback) {
        cursor += 1;
        return callback;
      },
      useRef(initial) {
        const index = cursor;
        cursor += 1;
        stateSlots[index] ??= { current: initial };
        return stateSlots[index];
      },
      useState(initial) {
        const index = cursor;
        cursor += 1;
        stateSlots[index] ??= { value: initial };
        const slot = stateSlots[index];

        return [
          slot.value,
          (next) => {
            slot.value = typeof next === 'function' ? next(slot.value) : next;
          }
        ];
      },
      useEffect(effect, deps) {
        const index = cursor;
        cursor += 1;
        effectSlots[index] ??= {};
        const slot = effectSlots[index];
        const previous = slot.deps;
        const changed =
          previous === undefined ||
          deps === undefined ||
          deps.some((dep, i) => dep !== previous[i]);

        if (!changed) return;

        slot.cleanup?.();
        slot.deps = deps;
        slot.effect = effect;
        slot.cleanup = effect() ?? undefined;
      }
    }
  };
}

function loadRegistry(tailscaleComponent, frpComponent = fakeComponent('frp')) {
  return runModule(`${sectionRoot}/registry.ts`, {
    './components/tailscale': { tailscaleComponent },
    './components/frp': { frpComponent },
    './components/cloudflare': { cloudflareComponent: fakeComponent('cloudflare') },
    './components/netbird': { netbirdComponent: fakeComponent('netbird') },
    './components/zerotier': { zerotierComponent: fakeComponent('zerotier') }
  });
}

function fakeComponent(id, extra = {}) {
  return {
    id,
    name: id === 'tailscale' ? 'Tailscale' : id,
    Icon: () => null,
    messages: { en: {} },
    View: Symbol(`${id}View`),
    ...extra
  };
}

/** Registry arrays are created inside the vm realm, so compare plain values. */
function componentIds(registry) {
  return registry.getRemoteNetworkComponents().map((component) => component.id);
}

function loadShell(components, syncServerFeatures = async () => ({ code: 0, data: null })) {
  const Home = Symbol('RemoteNetworkHome');
  const stub = createReactStub();
  const shell = runModule(`${sectionRoot}/index.tsx`, {
    react: stub.react,
    antd: { Spin: Symbol('Spin') },
    '@/api/application.ts': { syncServerFeatures },
    './home.tsx': { RemoteNetworkHome: Home },
    './messages.ts': { RemoteNetworkMessagesContext: { Provider: MessagesProvider } },
    './registry.ts': {
      getRemoteNetworkComponents: () => components,
      getRemoteNetworkComponent: (id) => components.find((component) => component.id === id)
    },
    'react/jsx-runtime': jsxRuntime
  });

  return {
    Home,
    unmount: stub.unmount,
    replayEffects: stub.replayEffects,
    settle: () => new Promise((resolve) => setTimeout(resolve, 0)),
    render(props) {
      stub.reset();
      return shell.RemoteNetwork(props);
    }
  };
}

async function mountShell(components, props) {
  const harness = loadShell(components);
  harness.render(props);
  await harness.settle();
  return harness;
}

test('Tailscale, NetBird, ZeroTier, Cloudflare and FRP are registered as remote network components', () => {
  const registrySource = withoutComments(source(`${sectionRoot}/registry.ts`));
  const registrations = [
    ...registrySource.matchAll(/registerRemoteNetworkComponent\((\w+)\);/g)
  ].map((match) => match[1]);

  assert.deepEqual(registrations, [
    'tailscaleComponent',
    'netbirdComponent',
    'zerotierComponent',
    'cloudflareComponent',
    'frpComponent'
  ]);
  assert.match(registrySource, /import \{ tailscaleComponent \} from '\.\/components\/tailscale';/);
  assert.match(registrySource, /import \{ frpComponent \} from '\.\/components\/frp';/);
});

test('the home page lists exactly the components the registry provides', () => {
  const homeSource = source(`${sectionRoot}/home.tsx`);

  assert.match(homeSource, /components\.map\(\(component\) =>/);
  assert.doesNotMatch(homeSource, /tailscale|frp/i);
});

test('the registry resolves the registered component and rejects unknown ids', () => {
  const tailscale = fakeComponent('tailscale');
  const registry = loadRegistry(tailscale);

  assert.equal(componentIds(registry).join(','), 'tailscale,netbird,zerotier,cloudflare,frp');
  assert.equal(registry.getRemoteNetworkComponent('tailscale'), tailscale);
  assert.equal(registry.getRemoteNetworkComponent('unknown'), undefined);
});

test('a new component joins the hub by registering alone', () => {
  const registry = loadRegistry(fakeComponent('tailscale'));
  const frp = fakeComponent('sample');

  registry.registerRemoteNetworkComponent(frp);
  assert.equal(
    componentIds(registry).join(','),
    'tailscale,netbird,zerotier,cloudflare,frp,sample'
  );

  // Re-registering the same id is ignored instead of duplicating the card.
  registry.registerRemoteNetworkComponent({ ...frp });
  assert.equal(
    componentIds(registry).join(','),
    'tailscale,netbird,zerotier,cloudflare,frp,sample'
  );
  assert.equal(registry.getRemoteNetworkComponent('sample'), frp);
});

test('the home page shows every registered component', async () => {
  const tailscale = fakeComponent('tailscale');
  const frp = fakeComponent('frp');
  const props = { setIsLocked: () => {} };
  const { render, Home } = await mountShell([tailscale, frp], props);

  const home = render(props);

  assert.equal(home.type, Home);
  assert.deepEqual(
    home.props.components.map((component) => component.id),
    ['tailscale', 'frp']
  );
});

test('the shell opens the active component page and returns home on back', async () => {
  const tailscale = fakeComponent('tailscale');
  const setIsLocked = () => {};
  const { render, Home } = await mountShell([tailscale], { setIsLocked });

  const home = render({ setIsLocked });
  assert.equal(home.type, Home);
  assert.equal(typeof home.props.onOpen, 'function');

  home.props.onOpen('tailscale');
  const detail = render({ setIsLocked });
  assert.equal(detail.type, MessagesProvider);
  assert.equal(detail.props.value, tailscale.messages);

  const view = detail.props.children;
  assert.equal(view.type, tailscale.View);
  assert.equal(typeof view.props.setLocked, 'function');
  assert.equal(typeof view.props.onBack, 'function');

  view.props.onBack();
  assert.equal(render({ setIsLocked }).type, Home);
});

test('a locked component cannot be left through the in-page back button', async () => {
  const modalLock = [];
  const setIsLocked = (isLocked) => modalLock.push(isLocked);
  const tailscale = fakeComponent('tailscale');
  const { render, Home } = await mountShell([tailscale], { setIsLocked });

  render({ setIsLocked }).props.onOpen('tailscale');
  render({ setIsLocked }).props.children.props.setLocked(true);

  const locked = render({ setIsLocked });
  assert.equal(modalLock.at(-1), true);
  assert.equal(locked.props.children.type, tailscale.View);

  locked.props.children.props.onBack();
  assert.equal(render({ setIsLocked }).type, MessagesProvider);

  // Unlocking restores the only way home.
  locked.props.children.props.setLocked(false);
  const unlocked = render({ setIsLocked });
  assert.equal(modalLock.at(-1), false);

  unlocked.props.children.props.onBack();
  assert.equal(render({ setIsLocked }).type, Home);
});

test('the shell clears the lock when it unmounts', async () => {
  const modalLock = [];
  const setIsLocked = (isLocked) => modalLock.push(isLocked);
  const { render, unmount } = await mountShell([fakeComponent('tailscale')], { setIsLocked });

  render({ setIsLocked }).props.onOpen('tailscale');
  render({ setIsLocked }).props.children.props.setLocked(true);
  render({ setIsLocked });
  assert.equal(modalLock.at(-1), true);

  unmount();
  assert.equal(modalLock.at(-1), false);
});

test('the shell never imports a component implementation', () => {
  const shellSource = source(`${sectionRoot}/index.tsx`);
  const imports = [...shellSource.matchAll(/from '([^']+)'/g)].map((match) => match[1]);

  assert.deepEqual(
    imports.sort(),
    ['./home.tsx', './messages.ts', './registry.ts', 'react', 'antd', '@/api/application.ts'].sort()
  );
});

test('feature sync finishes before network status mounts and runs once per visit', async () => {
  let finishSync;
  let syncCalls = 0;
  const pending = new Promise((resolve) => {
    finishSync = resolve;
  });
  const syncServerFeatures = () => {
    syncCalls += 1;
    return pending;
  };
  const components = [fakeComponent('tailscale')];
  const props = { setIsLocked: () => {} };
  const harness = loadShell(components, syncServerFeatures);

  assert.notEqual(harness.render(props).type, harness.Home);
  harness.replayEffects();
  assert.notEqual(harness.render(props).type, harness.Home);
  assert.equal(syncCalls, 1, 'effect replay must share the pending request');

  finishSync({ code: 0, data: null });
  await harness.settle();
  const home = harness.render(props);
  assert.equal(home.type, harness.Home);
  home.props.onOpen('tailscale');
  harness.render(props).props.children.props.onBack();
  assert.equal(harness.render(props).type, harness.Home);
  assert.equal(syncCalls, 1, 'navigation within the section must not refetch the CDN');
  harness.unmount();

  const reopened = loadShell(components, syncServerFeatures);
  reopened.render(props);
  await reopened.settle();
  assert.equal(reopened.render(props).type, reopened.Home);
  assert.equal(syncCalls, 2, 'a new visit must refresh the feature flags again');
  reopened.unmount();
});

test('failed feature sync still opens network settings with the local feature flags', async () => {
  for (const sync of [
    async () => ({ code: -1, data: null }),
    async () => {
      throw new Error('device unavailable');
    }
  ]) {
    const harness = loadShell([fakeComponent('tailscale')], sync);
    const props = { setIsLocked: () => {} };
    harness.render(props);
    await harness.settle();
    assert.equal(harness.render(props).type, harness.Home);
    harness.unmount();
  }
});

test('feature sync uses the authenticated device application API', async () => {
  const calls = [];
  const api = runModule('src/api/application.ts', {
    '@/lib/http.ts': {
      http: {
        post: async (...args) => {
          calls.push(args);
          return { code: 0, data: null };
        }
      }
    }
  });
  await api.syncServerFeatures();
  assert.deepEqual(calls, [['/api/application/features/sync']]);
});

test('settings mounts the remote network section under its own tab id', () => {
  const settingsSource = source('src/pages/desktop/menu/settings/index.tsx');

  assert.match(settingsSource, /import \{ RemoteNetwork \} from '\.\/remote-network';/);
  assert.match(
    settingsSource,
    /id: 'remoteNetwork',\s*icon: <NetworkIcon size=\{16\} \/>,\s*component: <RemoteNetwork setIsLocked=\{setIsLocked\} \/>/
  );
  assert.doesNotMatch(settingsSource, /icons\/tailscale/);
});
