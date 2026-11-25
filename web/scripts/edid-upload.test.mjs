import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

function loadTypeScriptModule(relativePath) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(source(relativePath), {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, module }, { filename: file });
  return module.exports;
}

const model = loadTypeScriptModule('src/pages/desktop/menu/screen/edid-model.ts');
const menuSource = source('src/pages/desktop/menu/screen/edid.tsx');
const settingsSource = source('src/pages/desktop/menu/settings/screen/edid.tsx');
const apiSource = source('src/api/vm.ts');

const reasonKeys = {
  bad_request: 'settings.screen.edid.errors.badRequest',
  invalid_file_name: 'settings.screen.edid.errors.invalidFileName',
  edid_too_large: 'settings.screen.edid.errors.fileTooLarge',
  invalid_edid_length: 'settings.screen.edid.errors.invalidLength',
  invalid_edid_header: 'settings.screen.edid.errors.invalidHeader',
  unsupported_edid_version: 'settings.screen.edid.errors.unsupportedVersion',
  edid_extension_count_mismatch: 'settings.screen.edid.errors.extensionCountMismatch',
  invalid_edid_checksum: 'settings.screen.edid.errors.invalidChecksum',
  invalid_cta_extension: 'settings.screen.edid.errors.invalidCta',
  current_edid_cannot_be_overwritten: 'settings.screen.edid.errors.currentCannotOverwrite',
  save_failed: 'settings.screen.edid.errors.saveFailed'
};

test('preflight accepts only .bin files with supported EDID block lengths', () => {
  for (const size of [128, 256, 384, 512]) {
    assert.equal(model.getEdidUploadPreflightError({ name: 'display.bin', size }), undefined);
  }
  assert.equal(model.getEdidUploadPreflightError({ name: 'DISPLAY.BIN', size: 128 }), undefined);

  for (const size of [0, 127, 129]) {
    assert.equal(
      model.getEdidUploadPreflightError({ name: 'display.bin', size }),
      'invalid_edid_length'
    );
  }
  assert.equal(
    model.getEdidUploadPreflightError({ name: 'display.bin', size: 513 }),
    'edid_too_large'
  );
  assert.equal(
    model.getEdidUploadPreflightError({ name: 'display.edid', size: 128 }),
    'invalid_file_name'
  );
});

test('preflight failures do not issue an upload request', async () => {
  const invalidFiles = [
    {
      file: { name: 'invalid.txt', size: 128 },
      key: 'settings.screen.edid.errors.invalidFileName'
    },
    {
      file: { name: 'truncated.bin', size: 127 },
      key: 'settings.screen.edid.errors.invalidLength'
    },
    {
      file: { name: 'oversized.bin', size: 513 },
      key: 'settings.screen.edid.errors.fileTooLarge'
    }
  ];

  for (const { file, key } of invalidFiles) {
    let requests = 0;
    const loading = [];
    const errors = [];
    const succeeded = await model.runEdidUpload({
      file,
      request: async () => {
        requests += 1;
        return { code: 0, msg: 'success' };
      },
      refreshCustomList: () => assert.fail('invalid upload refreshed the list'),
      setLoading: (value) => loading.push(value),
      onError: (errorKey) => errors.push(errorKey)
    });

    assert.equal(succeeded, false, file.name);
    assert.equal(requests, 0, file.name);
    assert.deepEqual(loading, [], file.name);
    assert.deepEqual(errors, [key], file.name);
  }
});

test('every backend EDID reason maps to its localized message', async () => {
  for (const [reason, expectedKey] of Object.entries(reasonKeys)) {
    const errors = [];
    const succeeded = await model.runEdidUpload({
      file: { name: 'valid.bin', size: 128 },
      request: async () => ({ code: -1, msg: 'rejected', data: { reason } }),
      refreshCustomList: () => assert.fail(`${reason} refreshed the list`),
      setLoading: () => {},
      onError: (key) => errors.push(key)
    });

    assert.equal(succeeded, false, reason);
    assert.deepEqual(errors, [expectedKey], reason);
  }
});

test('successful uploads refresh the authoritative custom list and release loading', async () => {
  const events = [];
  const succeeded = await model.runEdidUpload({
    file: { name: 'valid.bin', size: 256 },
    request: async () => {
      events.push('request');
      return { code: 0, msg: 'success', data: { file: 'valid.bin' } };
    },
    refreshCustomList: () => events.push('refresh'),
    setLoading: (value) => events.push(`loading:${value}`),
    onError: (key) => assert.fail(`unexpected error: ${key}`)
  });

  assert.equal(succeeded, true);
  assert.deepEqual(events, ['loading:true', 'request', 'refresh', 'loading:false']);
});

test('upload and switch network failures always release loading', async () => {
  const uploadLoading = [];
  const uploadErrors = [];
  await model.runEdidUpload({
    file: { name: 'valid.bin', size: 128 },
    request: async () => {
      throw new Error('offline');
    },
    refreshCustomList: () => assert.fail('failed upload refreshed the list'),
    setLoading: (value) => uploadLoading.push(value),
    onError: (key) => uploadErrors.push(key)
  });
  assert.deepEqual(uploadLoading, [true, false]);
  assert.deepEqual(uploadErrors, ['settings.screen.edid.errors.networkFailed']);

  const switchLoading = [];
  const switchErrors = [];
  let refreshes = 0;
  await model.runEdidSwitch({
    request: async () => {
      throw new Error('offline');
    },
    refreshCurrent: () => {
      refreshes += 1;
    },
    setLoading: (value) => switchLoading.push(value),
    onError: (key) => switchErrors.push(key)
  });
  assert.equal(refreshes, 1);
  assert.deepEqual(switchLoading, [true, false]);
  assert.deepEqual(switchErrors, ['settings.screen.edid.errors.networkFailed']);
});

test('failed switches refresh current state after a possible backend rollback', async () => {
  const events = [];
  const succeeded = await model.runEdidSwitch({
    request: async () => {
      events.push('request');
      return {
        code: -1,
        msg: 'failed to switch EDID',
        data: { reason: 'invalid_edid_checksum' }
      };
    },
    refreshCurrent: () => events.push('refresh-current'),
    setLoading: (value) => events.push(`loading:${value}`),
    onError: (key) => events.push(`error:${key}`)
  });

  assert.equal(succeeded, false);
  assert.deepEqual(events, [
    'loading:true',
    'request',
    'error:settings.screen.edid.errors.invalidChecksum',
    'refresh-current',
    'loading:false'
  ]);
});

test('both EDID surfaces use the shared tested workflows and visible error messages', () => {
  assert.match(menuSource, /runEdidUpload\(\{/);
  assert.match(menuSource, /refreshCustomList: getCustomEdidList/);
  assert.match(menuSource, /runEdidSwitch\(\{/);
  assert.match(menuSource, /refreshCurrent: getCurrentEdid/);
  assert.match(menuSource, /feedback\.error\(t\(key\)\)/);
  assert.match(
    menuSource.slice(
      menuSource.indexOf('async function deleteEdid'),
      menuSource.indexOf('const content')
    ),
    /feedback\.error\(t\('settings\.screen\.edid\.errors\.deleteFailed'\)\)[\s\S]*?catch[\s\S]*?feedback\.error\(t\('settings\.screen\.edid\.errors\.networkFailed'\)\)/
  );

  assert.match(settingsSource, /runEdidSwitch\(\{/);
  assert.match(settingsSource, /refreshCurrent: getCurrentEdid/);
  assert.match(settingsSource, /feedback\.error\(t\(key\)\)/);
  assert.match(apiSource, /export type EdidErrorReason =/);
  for (const reason of Object.keys(reasonKeys)) {
    assert.match(apiSource, new RegExp(`'${reason}'`));
  }
});
