import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

function loadLocalStorageModule(initialEntries = []) {
  const values = new Map(initialEntries);
  const localStorage = {
    getItem(key) {
      return values.has(key) ? values.get(key) : null;
    },
    removeItem(key) {
      values.delete(key);
    },
    setItem(key, value) {
      values.set(key, String(value));
    }
  };
  const file = path.resolve(import.meta.dirname, '../src/lib/localstorage.ts');
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, localStorage, module }, { filename: file });

  return { localStorage: module.exports, values };
}

test('reads a saved volume regardless of the removed remember-volume preference', () => {
  const { localStorage } = loadLocalStorageModule([
    ['nano-kvm-volume-panel-remember-state', 'false'],
    ['nano-kvm-video-volume', '37']
  ]);

  assert.equal(localStorage.getSavedVideoVolume(), 37);
});

test('saves every valid user-selected volume', () => {
  const { localStorage, values } = loadLocalStorageModule();

  for (const volume of [0, 37, 100]) {
    localStorage.setSavedVideoVolume(volume);
    assert.equal(values.get('nano-kvm-video-volume'), String(volume));
    assert.equal(localStorage.getSavedVideoVolume(), volume);
  }
});

test('ignores invalid saved and requested volume values', () => {
  for (const stored of ['', ' ', 'NaN', '-1', '101']) {
    const { localStorage } = loadLocalStorageModule([['nano-kvm-video-volume', stored]]);
    assert.equal(localStorage.getSavedVideoVolume(), null);
  }

  const { localStorage, values } = loadLocalStorageModule([['nano-kvm-video-volume', '37']]);
  for (const volume of [Number.NaN, -1, 101]) {
    localStorage.setSavedVideoVolume(volume);
    assert.equal(values.get('nano-kvm-video-volume'), '37');
  }
});

test('mute restores the last audible level, including a level saved before the toolbar update', () => {
  const { localStorage } = loadLocalStorageModule([['nano-kvm-video-volume', '37']]);
  localStorage.setSavedVideoVolume(0);
  assert.equal(localStorage.getUnmuteVolume(), 37);
  localStorage.setSavedVideoVolume(72);
  localStorage.setSavedVideoVolume(0);
  localStorage.setSavedVideoVolume(0);
  assert.equal(localStorage.getUnmuteVolume(), 72);
  assert.equal(loadLocalStorageModule().localStorage.getUnmuteVolume(), 50);
});

test('connect preferences snapshot the old volume and remain independent of toolbar changes', () => {
  const { localStorage } = loadLocalStorageModule([['nano-kvm-video-volume', '37']]);
  const initial = localStorage.getConnectPreferences();
  assert.deepEqual({ ...initial }, {
    volume: true, microphone: false, webcam: false, mouseJiggler: false
  });
  localStorage.setSavedVideoVolume(0);
  assert.equal(localStorage.getConnectPreferences().volume, true);
  localStorage.setConnectPreferences({ ...initial, volume: false, microphone: true });
  localStorage.setSavedVideoVolume(80);
  const next = localStorage.getConnectPreferences();
  assert.equal(next.volume, false);
  assert.equal(next.microphone, true);
});

test('new browsers start with all four preferences off and invalid capture flags never opt in', () => {
  const { localStorage, values } = loadLocalStorageModule();
  assert.ok(Object.values(localStorage.getConnectPreferences()).every((value) => value === false));
  values.set('nano-kvm-activate-on-connect', '{"webcam":"true","microphone":1}');
  assert.equal(localStorage.getConnectPreferences().webcam, false);
  assert.equal(localStorage.getConnectPreferences().microphone, false);
});
