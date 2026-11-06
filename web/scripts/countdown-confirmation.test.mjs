import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const confirmationFile = path.resolve(
  import.meta.dirname,
  '../src/hooks/useCountdownConfirmation.tsx'
);

function createHarness() {
  const source = fs.readFileSync(confirmationFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      esModuleInterop: true,
      jsx: ts.JsxEmit.ReactJSX,
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: confirmationFile
  }).outputText;

  let now = 0;
  let nextTimer = 1;
  let cleanup;
  const timers = new Map();
  const dialogs = [];
  const contextHolder = { type: 'context-holder' };

  const modal = {
    confirm(config) {
      const dialog = {
        config,
        destroyed: false,
        destroy() {
          this.destroyed = true;
        },
        update(updater) {
          this.config = updater(this.config);
        }
      };
      dialogs.push(dialog);
      return dialog;
    }
  };

  const modules = {
    react: {
      useEffect(effect) {
        cleanup = effect();
      },
      useRef(initialValue) {
        return { current: initialValue };
      }
    },
    antd: {
      Modal: {
        useModal() {
          return [modal, contextHolder];
        }
      }
    },
    'react/jsx-runtime': {
      jsx(type, props) {
        return { type, props };
      }
    }
  };
  const module = { exports: {} };
  const fakeWindow = {
    clearInterval(timer) {
      timers.delete(timer);
    },
    setInterval(callback) {
      const timer = nextTimer;
      nextTimer += 1;
      timers.set(timer, callback);
      return timer;
    }
  };

  vm.runInNewContext(
    output,
    {
      Date: { now: () => now },
      exports: module.exports,
      module,
      require(id) {
        if (!(id in modules)) throw new Error(`unexpected module: ${id}`);
        return modules[id];
      },
      window: fakeWindow
    },
    { filename: confirmationFile }
  );

  const confirmation = module.exports.useCountdownConfirmation({
    title: 'confirm title',
    description: 'confirm description',
    ok: 'continue',
    okCountdown: (seconds) => `continue (${seconds})`,
    cancel: 'cancel'
  });

  return {
    cleanup: () => cleanup?.(),
    confirmation,
    contextHolder,
    dialogs,
    advanceTo(milliseconds) {
      now = milliseconds;
      for (const callback of [...timers.values()]) callback();
    }
  };
}

test('countdown confirmation gates its action for five seconds', async () => {
  const harness = createHarness();
  let actionCalls = 0;
  const actionResult = Promise.resolve('updated');

  harness.confirmation.confirm(() => {
    actionCalls += 1;
    return actionResult;
  });
  harness.confirmation.confirm(() => assert.fail('duplicate dialog action must not be installed'));

  assert.equal(harness.confirmation.contextHolder, harness.contextHolder);
  assert.equal(harness.dialogs.length, 1);
  const dialog = harness.dialogs[0];
  assert.equal(dialog.config.title, 'confirm title');
  assert.equal(dialog.config.content.props.children, 'confirm description');
  assert.equal(dialog.config.okText, 'continue (5)');
  assert.equal(dialog.config.okButtonProps.disabled, true);
  assert.equal(dialog.config.cancelText, 'cancel');
  assert.equal(dialog.config.autoFocusButton, 'cancel');
  assert.equal(actionCalls, 0);

  harness.advanceTo(4999);
  assert.equal(dialog.config.okText, 'continue (1)');
  assert.equal(dialog.config.okButtonProps.disabled, true);
  assert.equal(actionCalls, 0);

  harness.advanceTo(5000);
  assert.equal(dialog.config.okText, 'continue');
  assert.equal(dialog.config.okButtonProps.disabled, false);
  assert.equal(actionCalls, 0);

  assert.equal(dialog.config.onOk(), actionResult);
  assert.equal(actionCalls, 1);
  assert.equal(await actionResult, 'updated');
  dialog.config.afterClose();
  harness.cleanup();
});

test('cancelling the countdown does not run the protected action', () => {
  const harness = createHarness();
  let actionCalls = 0;

  harness.confirmation.confirm(() => {
    actionCalls += 1;
  });
  const dialog = harness.dialogs[0];
  dialog.config.onCancel();
  dialog.config.afterClose();

  assert.equal(actionCalls, 0);
  harness.cleanup();
});
