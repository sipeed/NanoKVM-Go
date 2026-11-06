import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const runtimeFile = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/webhid-input-runtime.tsx'
);
const desktopFile = path.resolve(import.meta.dirname, '../src/pages/desktop/index.tsx');
const runtime = fs.readFileSync(runtimeFile, 'utf8');
const desktop = fs.readFileSync(desktopFile, 'utf8');

test('WebHID runtime is mounted and reconnects only registered granted devices', () => {
  assert.match(desktop, /<WebHIDInputRuntime\s*\/>/);
  assert.match(runtime, /webHIDInput\.connectGrantedDevices\(\)/);
  assert.match(runtime, /webHIDInput\.subscribe/);
  assert.match(runtime, /nextProfileKey === profileKey/);
});

test('WebHID runtime releases devices across page and component lifecycle', () => {
  assert.match(runtime, /window\.addEventListener\('pagehide', handlePageHide\)/);
  assert.match(runtime, /window\.addEventListener\('pageshow', handlePageShow\)/);
  assert.match(runtime, /suspended = true/);
  assert.match(runtime, /webHIDInput\.disconnectAll\(\)/);
});
