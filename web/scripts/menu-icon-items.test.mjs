import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

function loadTypeScriptModule(file) {
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
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

const menuIconItems = loadTypeScriptModule(
  path.resolve(
    import.meta.dirname,
    '../src/pages/desktop/menu/settings/appearance/menu-icon-items.ts'
  )
);

test('accepts only explicit boolean true for supported services', () => {
  assert.equal(typeof menuIconItems.getSupportedServiceIDs, 'function');
  assert.deepEqual(
    Array.from(
      menuIconItems.getSupportedServiceIDs([
        { id: 'supported', supported: true },
        { id: 'boolean-false', supported: false },
        { id: 'string-false', supported: 'false' },
        { id: 'missing-supported' },
        { id: 123, supported: true },
        null
      ])
    ),
    ['supported']
  );
  assert.deepEqual(Array.from(menuIconItems.getSupportedServiceIDs({})), []);
});

test('hides screen timelapse while device capabilities are unknown', () => {
  const items = [{ key: 'volume' }, { key: 'screenTimelapse' }];

  assert.equal(typeof menuIconItems.filterMenuIconItems, 'function');
  assert.deepEqual(
    menuIconItems.filterMenuIconItems(items, {
      isMobilePortrait: false,
      supportedServiceIDs: new Set()
    }),
    [{ key: 'volume' }]
  );
});

test('hides screen timelapse when the device reports other supported services', () => {
  const items = [{ key: 'volume' }, { key: 'screenTimelapse' }];

  assert.deepEqual(
    menuIconItems.filterMenuIconItems(items, {
      isMobilePortrait: false,
      supportedServiceIDs: new Set(['mcp'])
    }),
    [{ key: 'volume' }]
  );
});

test('shows screen timelapse when device support is explicitly available', () => {
  const items = [{ key: 'volume' }, { key: 'screenTimelapse' }];

  assert.deepEqual(
    menuIconItems.filterMenuIconItems(items, {
      isMobilePortrait: false,
      supportedServiceIDs: new Set(['screenTimelapse'])
    }),
    items
  );
});

test('keeps fullscreen and collapse hidden in mobile portrait', () => {
  const items = [{ key: 'volume' }, { key: 'fullscreen' }, { key: 'collapse' }];

  assert.deepEqual(
    menuIconItems.filterMenuIconItems(items, {
      isMobilePortrait: true,
      supportedServiceIDs: new Set()
    }),
    [{ key: 'volume' }]
  );
});
