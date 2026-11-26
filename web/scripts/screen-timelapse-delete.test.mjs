// Module-level evidence for the screen timelapse day-deletion model: the
// response-code mapping and the adjacent-day selection. Control wiring
// (confirm dialog, authorization gate continuation, player lease release) is
// exercised through the mocked runtime and manual verification, not here.
import assert from 'node:assert/strict';
import path from 'node:path';
import { after, before, test } from 'node:test';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');

let vite;
let model;

before(async () => {
  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false, ws: false }
  });
  model = await vite.ssrLoadModule(
    '/src/pages/desktop/menu/settings/ai/screen-timelapse-delete-model.ts'
  );
});

after(async () => {
  await vite.close();
});

function day(date) {
  return {
    date,
    videoUrl: `/api/ai/screen-timelapse/video?date=${date}`,
    startTime: `${date}T00:00:00Z`,
    endTime: `${date}T01:00:00Z`,
    durationMs: 3_600_000,
    frameCount: 10,
    filesize: 2048,
    deletable: true
  };
}

function days(...dates) {
  return dates.map(day);
}

test('delete outcome maps only -3 to the authorization flow', () => {
  assert.equal(model.resolveScreenTimelapseDeleteOutcome(0), 'success');
  assert.equal(model.resolveScreenTimelapseDeleteOutcome(-3), 'authorization_required');
  assert.equal(model.resolveScreenTimelapseDeleteOutcome(-5), 'busy');
});

test('delete outcome keeps generic failures out of the authorization semantics', () => {
  // -6 is the dedicated generic deletion failure; it must never be treated
  // as an authorization prompt again.
  for (const code of [-1, -2, -6, -99]) {
    assert.equal(model.resolveScreenTimelapseDeleteOutcome(code), 'failed');
  }
});

test('adjacent selection after deleting the selected head day', () => {
  const list = days('2026-01-03', '2026-01-02', '2026-01-01');
  assert.equal(
    model.selectAdjacentDayAfterDeletion(list, '2026-01-03', '2026-01-03'),
    '2026-01-02'
  );
});

test('adjacent selection after deleting the selected middle day', () => {
  const list = days('2026-01-03', '2026-01-02', '2026-01-01');
  assert.equal(
    model.selectAdjacentDayAfterDeletion(list, '2026-01-02', '2026-01-02'),
    '2026-01-01'
  );
});

test('adjacent selection after deleting the selected tail day', () => {
  const list = days('2026-01-03', '2026-01-02', '2026-01-01');
  assert.equal(
    model.selectAdjacentDayAfterDeletion(list, '2026-01-01', '2026-01-01'),
    '2026-01-02'
  );
});

test('adjacent selection after deleting the only day clears the selection', () => {
  const list = days('2026-01-02');
  assert.equal(model.selectAdjacentDayAfterDeletion(list, '2026-01-02', '2026-01-02'), '');
});

test('deleting a non-selected day keeps the current selection', () => {
  const list = days('2026-01-03', '2026-01-02', '2026-01-01');
  assert.equal(
    model.selectAdjacentDayAfterDeletion(list, '2026-01-02', '2026-01-03'),
    '2026-01-03'
  );
});

test('adjacent selection on an empty list yields an empty selection', () => {
  assert.equal(model.selectAdjacentDayAfterDeletion([], '2026-01-02', '2026-01-02'), '');
});

test('deleting a date outside the list falls back to the first remaining day', () => {
  const list = days('2026-01-03', '2026-01-02');
  assert.equal(
    model.selectAdjacentDayAfterDeletion(list, '2026-01-09', '2026-01-09'),
    '2026-01-03'
  );
});
