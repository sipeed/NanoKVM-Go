import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';
import { build } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
const keyboardFile = path.join(webRoot, 'src/pages/desktop/virtual-keyboard/index.tsx');

test('production bundle imports a renderable virtual keyboard component', async () => {
  const source = ts.createSourceFile(
    keyboardFile,
    fs.readFileSync(keyboardFile, 'utf8'),
    ts.ScriptTarget.Latest,
    true,
    ts.ScriptKind.TSX
  );
  const keyboardImport = source.statements.find(
    (statement) =>
      ts.isImportDeclaration(statement) &&
      statement.moduleSpecifier.text === 'react-simple-keyboard'
  );
  assert.ok(keyboardImport);

  // Keep the actual import and its package's ESM context: Vite 8 can resolve a
  // CommonJS default export differently in production than the dev server.
  const entry = path.join(path.dirname(keyboardFile), '__keyboard-build-test.mts');
  const result = await build({
    root: webRoot,
    configFile: false,
    logLevel: 'silent',
    define: { 'process.env.NODE_ENV': JSON.stringify('production') },
    plugins: [
      {
        name: 'virtual-keyboard-build-test',
        resolveId(id) {
          if (id === entry) return entry;
        },
        load(id) {
          if (id !== entry) return;
          return `
            ${keyboardImport.getText(source)}
            import { createElement } from 'react';
            import { renderToString } from 'react-dom/server.browser';
            export function render() {
              return renderToString(createElement(Keyboard));
            }
          `;
        }
      }
    ],
    build: {
      write: false,
      minify: true,
      lib: { entry, name: 'KeyboardBuildTest', formats: ['iife'] }
    }
  });

  const outputs = (Array.isArray(result) ? result : [result]).flatMap((bundle) => bundle.output);
  const chunk = outputs.find((output) => output.type === 'chunk' && output.isEntry);
  assert.ok(chunk);
  const context = vm.createContext({ TextEncoder });
  vm.runInContext(chunk.code, context);
  assert.match(context.KeyboardBuildTest.render(), /class="react-simple-keyboard"/);
});
