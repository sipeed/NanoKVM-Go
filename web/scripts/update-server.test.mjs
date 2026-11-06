import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const file = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/menu/settings/update/server.tsx'
);
const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
  compilerOptions: {
    jsx: ts.JsxEmit.ReactJSX,
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2022
  },
  fileName: file
}).outputText;

function harness(api) {
  const slots = [];
  const effects = [];
  const busy = [];
  let cursor = 0;
  let checks = 0;
  let pending = false;
  const sameDependencies = (left, right) =>
    left?.length === right.length && right.every((value, index) => Object.is(value, left[index]));
  const jsx = (type, props) => ({ type, props });
  const modules = {
    react: {
      useState(initial) {
        const slot = cursor++;
        if (!(slot in slots)) slots[slot] = initial;
        return [
          slots[slot],
          (value) => {
            slots[slot] = value;
          }
        ];
      },
      useRef(initial) {
        const slot = cursor++;
        if (!(slot in slots)) slots[slot] = { current: initial };
        return slots[slot];
      },
      useCallback(callback, dependencies) {
        const slot = cursor++;
        if (!sameDependencies(slots[slot]?.dependencies, dependencies)) {
          slots[slot] = { callback, dependencies };
        }
        return slots[slot].callback;
      },
      useEffect(effect, dependencies) {
        const slot = cursor++;
        const previous = slots[slot];
        if (sameDependencies(previous?.dependencies, dependencies)) return;
        const entry = { dependencies };
        slots[slot] = entry;
        effects.push(() => {
          previous?.cleanup?.();
          entry.cleanup = effect();
        });
      }
    },
    'react/jsx-runtime': { jsx, jsxs: jsx },
    antd: { Button: 'Button', Input: 'Input', Switch: 'Switch' },
    'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
    '@/api/application.ts': api
  };
  const module = { exports: {} };
  vm.runInNewContext(
    output,
    {
      module,
      exports: module.exports,
      require(id) {
        if (!(id in modules)) throw new Error(`Unexpected module ${id}`);
        return modules[id];
      }
    },
    { filename: file }
  );
  const props = {
    disabled: false,
    onBusyChange: (value) => busy.push(value),
    onPendingChange: (value) => {
      pending = value;
    },
    checkForUpdates: () => {
      checks += 1;
    }
  };
  return {
    busy,
    props,
    get checks() {
      return checks;
    },
    get pending() {
      return pending;
    },
    cleanup() {
      slots.forEach((slot) => slot?.cleanup?.());
    },
    render() {
      cursor = 0;
      const tree = module.exports.UpdateServer(props);
      while (effects.length) effects.shift()();
      const nodes = [];
      function visit(node) {
        if (Array.isArray(node)) return node.forEach(visit);
        if (!node || typeof node !== 'object') return;
        nodes.push(node);
        visit(node.props?.children);
      }
      visit(tree);
      return {
        input: nodes.find((node) => node.type === 'Input')?.props,
        toggle: nodes.find((node) => node.type === 'Switch').props,
        button: (key) =>
          nodes.find(
            (node) => node.type === 'Button' && node.props.children === `settings.update.${key}`
          )?.props,
        error: nodes.find((node) => node.props.role === 'alert')?.props.children
      };
    }
  };
}

async function flush() {
  await Promise.resolve();
  await Promise.resolve();
}

test('saving and restoring the server refresh versions only after persistence succeeds', async () => {
  const writes = [];
  const app = harness({
    getUpdateServer: async () => ({ code: 0, data: { server: '' } }),
    setUpdateServer: async (server) => {
      writes.push(server);
      return { code: 0, data: { server: server.replace(/\/+$/, '') } };
    }
  });
  assert.equal(app.render().toggle.disabled, true);
  assert.equal(app.render().input, undefined);
  await flush();
  app.render().toggle.onChange(true);
  assert.equal(app.render().input.value, 'https://cdn.sipeed.com/nanokvm-go/v2');
  assert.equal(app.pending, true);
  app.render().input.onChange({ target: { value: ' http://192.168.1.20:8080/releases/ ' } });
  app.render().button('server.save').onClick();
  assert.equal(app.checks, 0);
  assert.equal(app.render().input.disabled, true);
  await flush();
  assert.equal(app.render().input.value, 'http://192.168.1.20:8080/releases');
  assert.equal(app.pending, false);
  assert.equal(app.checks, 1);
  app.render().toggle.onChange(false);
  await flush();
  assert.deepEqual(writes, ['http://192.168.1.20:8080/releases/', '']);
  assert.equal(app.render().input, undefined);
  assert.equal(app.render().toggle.checked, false);
  assert.equal(app.pending, false);
  assert.equal(app.checks, 2);
  assert.deepEqual(app.busy, [true, false, true, false, true, false]);
});

test('failed saves preserve the draft and show errors without refreshing versions', async () => {
  let reject = false;
  const app = harness({
    getUpdateServer: async () => ({ code: 0, data: { server: '' } }),
    setUpdateServer: async () => {
      if (reject) throw new Error('connection failed');
      return { code: -1, data: null };
    }
  });
  app.render();
  await flush();
  app.render().toggle.onChange(true);
  app.render().input.onChange({ target: { value: 'invalid-address' } });
  app.render().button('server.save').onClick();
  await flush();
  assert.equal(app.render().error, 'settings.update.server.invalidURL');
  assert.equal(app.render().input.value, 'invalid-address');
  reject = true;
  app.render().button('server.save').onClick();
  await flush();
  assert.equal(app.render().error, 'settings.update.server.saveFailed');
  assert.equal(app.render().input.disabled, false);
  assert.equal(app.pending, true);
  assert.equal(app.checks, 0);
});

test('failed initial reads allow retry without permitting an unknown setting to be overwritten', async () => {
  let reads = 0;
  const app = harness({
    getUpdateServer: async () => {
      if (++reads === 1) throw new Error('connection failed');
      return { code: 0, data: { server: 'https://updates.example.com' } };
    },
    setUpdateServer: () => assert.fail('must load settings before saving')
  });
  app.render();
  await flush();
  assert.equal(app.render().toggle.disabled, true);
  assert.equal(app.render().error, 'settings.update.server.loadFailed');
  app.render().toggle.onChange(true);
  assert.equal(app.render().input, undefined);
  app.render().button('retry').onClick();
  await flush();
  assert.equal(app.render().input.value, 'https://updates.example.com');
  assert.equal(app.render().input.disabled, false);
});

test('unmount discards a pending save response and releases its busy state', async () => {
  let finish;
  const app = harness({
    getUpdateServer: async () => ({ code: 0, data: { server: '' } }),
    setUpdateServer: () =>
      new Promise((resolve) => {
        finish = resolve;
      })
  });
  app.render();
  await flush();
  app.render().toggle.onChange(true);
  app.render().input.onChange({ target: { value: 'https://updates.example.com/' } });
  app.render().button('server.save').onClick();
  app.cleanup();
  finish({ code: 0, data: { server: 'https://updates.example.com' } });
  await flush();
  assert.equal(app.checks, 0);
  assert.equal(app.pending, false);
  assert.deepEqual(app.busy, [true, false, true, false]);
});
