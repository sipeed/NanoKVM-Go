import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const keyboardFile = path.resolve(
  import.meta.dirname,
  '../src/pages/desktop/virtual-keyboard/index.tsx'
);
const keyboard = fs.readFileSync(keyboardFile, 'utf8');

test('virtual keyboard sends a neutral report on close and unmount', () => {
  assert.match(keyboard, /else if \(!isKeyboardOpen && wasKeyboardOpen\.current\)/);
  assert.match(keyboard, /function releaseKeyboardState\(\)/);
  assert.ok(
    keyboard.match(
      /inputRouter\.sendKeyboard\(InputSource\.VirtualKeyboard, new Uint8Array\(8\)\)/g
    )?.length >= 2
  );
});
