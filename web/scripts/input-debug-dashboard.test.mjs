import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const htmlFile = path.resolve(import.meta.dirname, '../public/input-debug.html');
const cssFile = path.resolve(import.meta.dirname, '../public/input-debug.css');
const scriptFile = path.resolve(import.meta.dirname, '../public/input-debug.js');
const html = fs.readFileSync(htmlFile, 'utf8');
const css = fs.readFileSync(cssFile, 'utf8');
const source = fs.readFileSync(scriptFile, 'utf8');

test('input debug dashboard exposes every pipeline stage', () => {
  for (const stage of ['capture', 'mapping', 'route', 'encode', 'dispatch']) {
    assert.match(html, new RegExp(`data-stage="${stage}"`));
  }
  assert.match(html, /Browser Event → Router → HID Gadget/);
  assert.match(html, /浏览器输入/);
  assert.match(html, /HID 输出/);
  assert.match(html, /id="browser-event-json"/);
  assert.match(html, /id="router-flow-graph"/);
  assert.match(html, /id="point-canvas"/);
  assert.match(html, /id="routes"/);
  assert.match(html, /id="sessions"/);
  assert.match(css, /\.pipeline-node/);
});

test('selected flow keeps a stable viewport and trace rows are reconciled in place', () => {
  assert.match(css, /--pipeline-node-height:\s*320px/);
  assert.match(css, /height:\s*var\(--pipeline-node-height\)/);
  assert.match(css, /\.pipeline-panel > \.panel-heading \{ height:\s*77px/);
  assert.match(css, /scrollbar-gutter:\s*stable/);
  assert.match(source, /function createTransactionRow\(\)/);
  assert.match(source, /function updateTransactionRow\(row, transaction\)/);
  assert.match(source, /insertBefore\(rows\[index\], current \|\| null\)/);
  assert.doesNotMatch(source, /elements\['trace-body'\]\.replaceChildren\(\.\.\.rows\)/);
  assert.match(source, /canvas\.width !== pixelWidth/);
  assert.match(source, /context\.setTransform\(scale, 0, 0, scale, 0, 0\)/);
});

test('input debug dashboard streams traces only while the page is open', () => {
  assert.match(source, /\/api\/hid\/input-router\/status/);
  assert.match(source, /\/api\/hid\/input-router\/trace-stream/);
  assert.match(source, /Accept: 'application\/x-ndjson'/);
  assert.match(source, /credentials: 'same-origin'/);
  assert.match(source, /window\.addEventListener\('pagehide'/);
  assert.match(source, /state\.streamAbort\?\.abort\(\)/);
  assert.doesNotMatch(source, /input-router\/traces\?limit=|localStorage|sessionStorage/);
});

test('input debug dashboard bounds, filters, pauses and exports local evidence', () => {
  assert.match(source, /const LOCAL_LIMIT = 1200/);
  assert.match(source, /state\.traces\.splice\(0, state\.traces\.length - LOCAL_LIMIT\)/);
  assert.match(source, /state\.paused = !state\.paused/);
  assert.match(source, /function filteredTransactions\(\)/);
  assert.match(source, /nanokvm-input-debug-dashboard\/v2/);
  assert.match(source, /browser_event/);
  assert.match(source, /capture_sequence/);
  assert.match(source, /client_delay_us/);
  assert.match(source, /router_duration_us/);
  assert.match(source, /hid_write_us/);
  assert.match(source, /function renderRouterFlowGraph\(transaction\)/);
  assert.match(source, /window\.inputDebugDashboard = \{ snapshot: dashboardSnapshot, clear: clearLocalTraces \}/);
});

test('input debug dashboard complies with self-only script CSP', () => {
  assert.match(html, /<link rel="stylesheet" href="\/input-debug\.css" \/>/);
  assert.match(html, /<script src="\/input-debug\.js" defer><\/script>/);
  assert.doesNotMatch(html, /<script>(?:.|\n)*<\/script>/);
  assert.doesNotMatch(html, /\son\w+=/);
});
