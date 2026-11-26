import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { setImmediate } from 'node:timers/promises';
import clsx from 'clsx';

import { loadComponent } from './hook-harness.mjs';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

const advancedSource = source('src/pages/desktop/menu/settings/device/advanced/index.tsx');
const networkFile = path.join(
  webRoot,
  'src/pages/desktop/menu/settings/device/virtual-devices/network.tsx'
);
const usbNetworkSource = source(
  'src/pages/desktop/menu/settings/device/virtual-devices/usb-network.tsx'
);

function networkHarness(api = {}) {
  const component = loadComponent(networkFile, 'Network', {
    antd: { Switch: 'Switch', Tooltip: 'Tooltip' },
    clsx,
    'lucide-react': { ChevronRightIcon: 'Chevron', CircleAlertIcon: 'Alert' },
    '@/api/virtual-device.ts': api,
    '@/components/ui/button-base.tsx': { ButtonBase: 'Button' },
    './usb-network.tsx': { UsbNetworkConfig: 'Config' }
  });
  const props = { isNetworkEnabled: false, isDisabled: false, getVirtualDevices: async () => {} };
  return {
    props,
    render() {
      const nodes = component.render(props);
      const byType = (type) => nodes.find((node) => node.type === type)?.props;
      const button = byType('Button');
      return {
        button,
        toggle: byType('Switch'),
        config: byType('Config'),
        panel: nodes.find((node) => node.props.id && node.props.id === button['aria-controls'])
          ?.props,
        tip: byType('Tooltip').title
      };
    }
  };
}

test('USB network settings open on demand, preserve the form on collapse and reset when disabled', () => {
  assert.doesNotMatch(advancedSource, /UsbNetwork|usb-network/);
  const { props, render } = networkHarness();
  let view = render();
  assert.equal(view.button.disabled, true);
  assert.equal(view.button.tabIndex, -1);
  assert.equal(view.config, undefined);
  props.isNetworkEnabled = true;
  view = render();
  assert.equal(view.button.tabIndex, 0);
  assert.equal(view.button['aria-expanded'], false);
  assert.equal(view.config, undefined);
  assert.equal(view.tip, 'settings.device.usbNetwork.forwardingTip');
  view.button.onClick();
  view = render();
  assert.equal(view.button['aria-expanded'], true);
  assert.equal(view.panel.hidden, false);
  assert.ok(view.config);
  view.button.onClick();
  view = render();
  assert.equal(view.panel.hidden, true);
  assert.ok(view.config);
  props.isNetworkEnabled = false;
  assert.equal(render().config, undefined);
  props.isNetworkEnabled = true;
  assert.equal(render().config, undefined);
});

test('saving locks the master switch and toggling waits for the authoritative refresh', async () => {
  let calls = 0;
  const { props, render } = networkHarness({
    updateVirtualDevice: async (type) => {
      assert.equal(type, 'network');
      calls++;
      return { code: 0 };
    }
  });
  props.isNetworkEnabled = true;
  let finishRefresh;
  props.getVirtualDevices = () =>
    new Promise((resolve) => {
      finishRefresh = resolve;
    });
  render().button.onClick();
  render().config.onSavingChange(true);
  assert.equal(render().toggle.disabled, true);
  render().config.onSavingChange(false);
  render().toggle.onChange();
  await setImmediate();
  assert.equal(calls, 1);
  assert.equal(render().toggle.loading, true);
  assert.equal(render().config.disabled, true);
  finishRefresh();
  await setImmediate();
  assert.equal(render().toggle.loading, false);
  assert.equal(render().config.disabled, false);
});

test('inline settings load the saved baseline and expose server and client modes', () => {
  assert.doesNotMatch(usbNetworkSource, /\bModal\b|isModalOpen|openModal/);
  assert.match(usbNetworkSource, /await api\.getUsbNetwork\(\)/);
  assert.match(usbNetworkSource, /setConfig\(next\);[\s\S]*?setDraft\(next\);/);
  assert.match(usbNetworkSource, /<Select<UsbNetworkConfigValue\['mode'\]>/);
  assert.match(usbNetworkSource, /value: 'server'/);
  assert.match(usbNetworkSource, /value: 'client'/);
  assert.match(usbNetworkSource, /\{mode === 'server' && \(/);
  assert.match(usbNetworkSource, /manualIpEnabled/);
  assert.match(usbNetworkSource, /forwardingEnabled/);
  assert.doesNotMatch(usbNetworkSource, /usbNetwork\.forwardingTip/);
});

test('cancel restores the saved baseline and apply posts once before refreshing status', () => {
  assert.match(usbNetworkSource, /function cancel\(\)[\s\S]*?setDraft\(config\)/);
  assert.match(usbNetworkSource, /const hasChanges = !isSameConfig\(config, draft\)/);
  assert.match(usbNetworkSource, /await api\.setUsbNetwork\(draft\)/);
  assert.match(usbNetworkSource, /setConfig\(saved\);[\s\S]*?setDraft\(saved\);/);
  assert.match(usbNetworkSource, /await getVirtualDevices\(\);/);
  assert.match(usbNetworkSource, /disabled=\{[^}]*!hasChanges[^}]*\}/);
});

test('load failures disable only the inline controls and saving is reported to the parent', () => {
  assert.match(usbNetworkSource, /setLoadError\(true\)/);
  assert.match(usbNetworkSource, /\) : loadError \? \([\s\S]*?usbNetwork\.loadFailed/);
  assert.match(usbNetworkSource, /onSavingChange\(true\)/);
  assert.match(usbNetworkSource, /finally \{[\s\S]*?onSavingChange\(false\)/);
  assert.match(usbNetworkSource, /if \(disabled \|\| isSaving \|\| !hasChanges/);
  assert.match(usbNetworkSource, /disabled=\{disabled \|\| isSaving\}/);
});

test('configuration rows stack on narrow screens and the mode selector becomes full width', () => {
  assert.match(usbNetworkSource, /flex-col[^"']*md:flex-row/);
  assert.match(usbNetworkSource, /w-full[^"']*md:w-\d+/);
  assert.match(usbNetworkSource, /grid w-full grid-cols-2[^"']*md:flex/);
  assert.doesNotMatch(usbNetworkSource, /rounded-lg bg-neutral-800\/60 p-4/);
  assert.doesNotMatch(usbNetworkSource, /className="min-h-5"/);
  assert.doesNotMatch(usbNetworkSource, /usbNetwork\.tip/);
});

test('every locale keeps the inline apply action and drops the removed modal actions', () => {
  const localesDir = path.join(webRoot, 'src/i18n/locales');
  const localeFiles = fs.readdirSync(localesDir).filter((file) => file.endsWith('.ts'));

  for (const localeFile of localeFiles) {
    const localeSource = fs.readFileSync(path.join(localesDir, localeFile), 'utf8');
    const start = localeSource.indexOf('usbNetwork: {');
    const end = localeSource.indexOf('\n        usbId:', start);
    assert.notEqual(start, -1, `${localeFile} is missing usbNetwork`);
    assert.notEqual(end, -1, `${localeFile} has an invalid usbNetwork block`);

    const usbNetworkBlock = localeSource.slice(start, end);
    assert.match(usbNetworkBlock, /^\s+applyBtn:/m, `${localeFile} is missing applyBtn`);
    assert.match(usbNetworkBlock, /^\s+forwardingTip:/m, `${localeFile} is missing forwardingTip`);
    assert.doesNotMatch(usbNetworkBlock, /^\s+tip:/m, `${localeFile} retains tip`);
    assert.doesNotMatch(usbNetworkBlock, /^\s+config:/m, `${localeFile} retains config`);
    assert.doesNotMatch(usbNetworkBlock, /^\s+okBtn:/m, `${localeFile} retains okBtn`);
  }
});
