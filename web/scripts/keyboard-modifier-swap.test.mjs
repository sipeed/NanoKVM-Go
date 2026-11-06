import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

function loadTypeScriptModule(file, context = {}) {
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
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
      ...context
    },
    { filename: file }
  );

  return module.exports;
}

const keymapFile = path.resolve(import.meta.dirname, '../src/lib/keymap.ts');
const keyboardFile = path.resolve(import.meta.dirname, '../src/lib/keyboard.ts');
const localStorageFile = path.resolve(import.meta.dirname, '../src/lib/localstorage.ts');

const keymap = loadTypeScriptModule(keymapFile);
const keyboardModule = loadTypeScriptModule(keyboardFile, {
  require(specifier) {
    if (specifier === './keymap') return keymap;
    throw new Error(`Unexpected require(${specifier})`);
  }
});

test('swaps only matching left and right Command/Ctrl codes', () => {
  const expected = new Map([
    ['ControlLeft', 'MetaLeft'],
    ['MetaLeft', 'ControlLeft'],
    ['ControlRight', 'MetaRight'],
    ['MetaRight', 'ControlRight']
  ]);

  for (const [code, mapped] of expected) {
    assert.equal(keymap.mapCommandCtrlCode(code, false), code);
    assert.equal(keymap.mapCommandCtrlCode(code, true), mapped);
  }

  for (const code of ['ShiftLeft', 'AltRight', 'KeyA']) {
    assert.equal(keymap.mapCommandCtrlCode(code, true), code);
  }
});

test('builds swapped HID modifier reports and releases the same mapped bit', () => {
  const { KeyboardReport } = keyboardModule;
  const cases = [
    ['ControlLeft', keymap.ModifierBits.LeftMeta],
    ['MetaLeft', keymap.ModifierBits.LeftCtrl],
    ['ControlRight', keymap.ModifierBits.RightMeta],
    ['MetaRight', keymap.ModifierBits.RightCtrl]
  ];

  for (const [code, expectedBit] of cases) {
    const keyboard = new KeyboardReport();
    assert.equal(keyboard.keyDown(code, true)[0], expectedBit);
    assert.equal(keyboard.keyUp(code, true)[0], 0);
  }

  const keyboard = new KeyboardReport();
  assert.equal(keyboard.keyDown('ControlLeft')[0], keymap.ModifierBits.LeftCtrl);
});

test('persists the browser-local swap preference', () => {
  const values = new Map();
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
  const storage = loadTypeScriptModule(localStorageFile, { localStorage });

  assert.equal(storage.getKeyboardSwapCommandCtrl(), false);
  storage.setKeyboardSwapCommandCtrl(true);
  assert.equal(storage.getKeyboardSwapCommandCtrl(), true);
  storage.setKeyboardSwapCommandCtrl(false);
  assert.equal(storage.getKeyboardSwapCommandCtrl(), false);
});

test('defaults to disabled when browser storage is unavailable', () => {
  const storage = loadTypeScriptModule(localStorageFile);
  assert.equal(storage.getKeyboardSwapCommandCtrl(), false);
});
