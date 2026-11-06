import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const sourceFile = path.resolve(import.meta.dirname, '../src/lib/webhid-input.ts');

function loadModule() {
  const output = ts.transpileModule(fs.readFileSync(sourceFile, 'utf8'), {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    fileName: sourceFile
  }).outputText;
  const module = { exports: {} };
  const InputKind = {
    KeyboardState: 1,
    RelativePointer: 2,
    AbsolutePointer: 3,
    Stylus: 4,
    Touchscreen: 5
  };

  vm.runInNewContext(
    output,
    {
      exports: module.exports,
      module,
      performance: { now: () => 123 },
      require(specifier) {
        if (specifier === '@/lib/input-router.ts') {
          return { InputKind, InputSource: { WebHID: 32 }, inputRouter: {} };
        }
        throw new Error(`Unexpected require(${specifier})`);
      }
    },
    { filename: sourceFile }
  );
  return { ...module.exports, InputKind };
}

class FakeDevice {
  opened = false;
  productId = 2;
  productName = 'Test Pad';
  vendorId = 1;
  listeners = new Set();

  async open() {
    this.opened = true;
  }

  async close() {
    this.opened = false;
  }

  addEventListener(_type, listener) {
    this.listeners.add(listener);
  }

  removeEventListener(_type, listener) {
    this.listeners.delete(listener);
  }

  emit(reportId, bytes) {
    const array = Uint8Array.from(bytes);
    const report = {
      device: this,
      reportId,
      data: new DataView(array.buffer)
    };
    for (const listener of this.listeners) listener(report);
  }
}

test('WebHID profile maps reports and releases through semantic router', async () => {
  const { WebHIDInputManager, InputKind } = loadModule();
  const device = new FakeDevice();
  const provider = {
    async requestDevice() {
      return [device];
    },
    async getDevices() {
      return [device];
    }
  };
  const calls = [];
  const router = {
    sendKeyboard(source, report, capturedAt) {
      calls.push({ source, report: Array.from(report), capturedAt });
      return true;
    }
  };
  const manager = new WebHIDInputManager(provider, router, () => 55);
  manager.register({
    id: 'test-keyboard',
    filters: [{ vendorId: 1, productId: 2 }],
    matches(candidate) {
      return candidate.vendorId === 1 && candidate.productId === 2;
    },
    decode(report) {
      return [{ kind: InputKind.KeyboardState, report: new Uint8Array(report.data.buffer) }];
    },
    release() {
      return [{ kind: InputKind.KeyboardState, report: new Uint8Array(8) }];
    }
  });

  const connected = await manager.requestProfile('test-keyboard');
  assert.equal(connected.length, 1);
  assert.equal(device.opened, true);
  assert.deepEqual(JSON.parse(JSON.stringify(manager.activeDevices)), [
    { profile: 'test-keyboard', productId: 2, productName: 'Test Pad', vendorId: 1 }
  ]);

  device.emit(7, [1, 2, 3, 4, 5, 6, 7, 8]);
  assert.deepEqual(calls[0], {
    source: 32,
    report: [1, 2, 3, 4, 5, 6, 7, 8],
    capturedAt: 55
  });

  await manager.disconnect(device);
  assert.equal(device.opened, false);
  assert.deepEqual(calls[1], { source: 32, report: Array(8).fill(0), capturedAt: 55 });
});

test('WebHID manager rejects duplicate and unknown profiles', async () => {
  const { WebHIDInputManager } = loadModule();
  const provider = { requestDevice: async () => [], getDevices: async () => [] };
  const manager = new WebHIDInputManager(provider, {});
  const profile = {
    id: 'pad',
    filters: [{ vendorId: 1 }],
    matches: () => true,
    decode: () => []
  };

  manager.register(profile);
  assert.throws(() => manager.register(profile), /already registered/);
  await assert.rejects(manager.requestProfile('missing'), /unknown WebHID profile/);
});

test('WebHID profile registry publishes safe summaries and unregister releases devices', async () => {
  const { WebHIDInputManager, InputKind } = loadModule();
  const device = new FakeDevice();
  const calls = [];
  const manager = new WebHIDInputManager(
    {
      requestDevice: async () => [device],
      getDevices: async () => [device]
    },
    {
      sendKeyboard(_source, report) {
        calls.push(Array.from(report));
      }
    },
    () => 77
  );
  let notifications = 0;
  const unsubscribe = manager.subscribe(() => {
    notifications += 1;
  });
  const unregister = manager.register({
    id: 'known-keyboard',
    label: 'Known keyboard',
    filters: [{ vendorId: 0x1234, productId: 0x5678 }],
    matches: () => true,
    decode: () => [],
    release: () => [{ kind: InputKind.KeyboardState, report: new Uint8Array(8) }]
  });

  const summaries = manager.registeredProfiles;
  assert.deepEqual(JSON.parse(JSON.stringify(summaries)), [
    {
      id: 'known-keyboard',
      label: 'Known keyboard',
      filters: [{ vendorId: 0x1234, productId: 0x5678 }]
    }
  ]);
  summaries[0].filters[0].vendorId = 1;
  assert.equal(manager.registeredProfiles[0].filters[0].vendorId, 0x1234);

  await manager.requestProfile('known-keyboard');
  await unregister();
  assert.equal(device.opened, false);
  assert.deepEqual(calls, [Array(8).fill(0)]);
  assert.deepEqual(JSON.parse(JSON.stringify(manager.registeredProfiles)), []);
  assert.ok(notifications >= 3);
  unsubscribe();
});

test('WebHID disconnect closes the device even when profile release fails', async () => {
  const { WebHIDInputManager } = loadModule();
  const device = new FakeDevice();
  const manager = new WebHIDInputManager(
    { requestDevice: async () => [device], getDevices: async () => [device] },
    {}
  );
  manager.register({
    id: 'broken-release',
    filters: [{ vendorId: 1 }],
    matches: () => true,
    decode: () => [],
    release() {
      throw new Error('release failed');
    }
  });

  await manager.requestProfile('broken-release');
  await assert.rejects(manager.disconnect(device), /failed to release WebHID input/);
  assert.equal(device.opened, false);
  assert.deepEqual(JSON.parse(JSON.stringify(manager.activeDevices)), []);
});
