import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const webRoot = path.resolve(import.meta.dirname, '..');

function source(relativePath) {
  return fs.readFileSync(path.join(webRoot, relativePath), 'utf8');
}

const advancedSource = source('src/pages/desktop/menu/settings/device/advanced/index.tsx');
const networkSource = source('src/pages/desktop/menu/settings/device/virtual-devices/network.tsx');
const usbNetworkSource = source(
  'src/pages/desktop/menu/settings/device/virtual-devices/usb-network.tsx'
);

test('USB network configuration moves out of Advanced behind an inline expand button', () => {
  assert.doesNotMatch(advancedSource, /UsbNetwork|usb-network/);
  assert.match(networkSource, /import \{ UsbNetworkConfig \} from '\.\/usb-network\.tsx';/);
  assert.match(
    networkSource,
    /import \{ ButtonBase \} from '@\/components\/ui\/button-base\.tsx';/
  );
  assert.match(
    networkSource,
    /import \{ ChevronRightIcon, CircleAlertIcon \} from 'lucide-react';/
  );
  assert.doesNotMatch(networkSource, /\bCollapse\b/);
  assert.match(networkSource, /<Switch[\s\S]*?<ButtonBase[\s\S]*?<ChevronRightIcon/);
  assert.match(networkSource, /isConfigExpanded && 'rotate-90'/);
  assert.doesNotMatch(networkSource, /rotate-180/);
  assert.match(networkSource, /aria-expanded=\{isConfigExpanded\}/);
  assert.match(
    networkSource,
    /aria-controls=\{isNetworkEnabled \? 'usb-network-config' : undefined\}/
  );
  assert.match(networkSource, /onClick=\{toggleConfig\}/);
});

test('enabling the device reveals a fixed-width button slot and moves the switch left', () => {
  assert.match(
    networkSource,
    /transition-\[width,margin,opacity\][\s\S]*?isNetworkEnabled[\s\S]*?'ml-2 w-8 opacity-100'[\s\S]*?'ml-0 w-0 opacity-0'/
  );
  assert.match(networkSource, /className="flex size-8 items-center justify-center/);
  assert.match(networkSource, /aria-hidden=\{!isNetworkEnabled\}/);
  assert.match(networkSource, /tabIndex=\{isNetworkEnabled \? 0 : -1\}/);
});

test('the master switch waits for the authoritative refresh and only locks during a save', () => {
  assert.match(networkSource, /getVirtualDevices: \(\) => Promise<void>/);
  assert.match(networkSource, /await getVirtualDevices\(\);/);
  assert.match(networkSource, /disabled=\{isDisabled \|\| isConfigSaving\}/);
  assert.doesNotMatch(networkSource, /loadError/);
  assert.match(networkSource, /disabled=\{isDisabled \|\| isToggleLoading\}/);
  assert.match(networkSource, /onSavingChange=\{setIsConfigSaving\}/);
});

test('the expand button lazily mounts one bordered panel and closing the device resets it', () => {
  assert.match(
    networkSource,
    /const \[isConfigExpanded, setIsConfigExpanded\] = useState\(false\)/
  );
  assert.match(networkSource, /const \[hasOpenedConfig, setHasOpenedConfig\] = useState\(false\)/);
  assert.match(
    networkSource,
    /function toggleConfig\(\)[\s\S]*?setHasOpenedConfig\(true\);[\s\S]*?setIsConfigExpanded\(\(expanded\) => !expanded\)/
  );
  assert.match(
    networkSource,
    /useEffect\(\(\) => \{[\s\S]*?if \(!isNetworkEnabled\) \{[\s\S]*?setIsConfigExpanded\(false\);[\s\S]*?setHasOpenedConfig\(false\)/
  );
  assert.match(networkSource, /\{isNetworkEnabled && hasOpenedConfig && \(/);
  assert.match(
    networkSource,
    /id="usb-network-config"[\s\S]*?rounded-lg border border-neutral-700\/60[\s\S]*?p-3 sm:p-4[\s\S]*?hidden=\{!isConfigExpanded\}[\s\S]*?<UsbNetworkConfig/
  );
  assert.equal((networkSource.match(/id="usb-network-config"/g) ?? []).length, 1);
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
  assert.match(
    networkSource,
    /settings\.device\.network[\s\S]*?<Tooltip[\s\S]*?usbNetwork\.forwardingTip[\s\S]*?<CircleAlertIcon[\s\S]*?settings\.device\.networkDesc/
  );
  assert.match(networkSource, /text-amber-400/);
  assert.doesNotMatch(networkSource, /border-amber-500\/30|bg-amber-500\/10/);
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
  assert.doesNotMatch(networkSource, /loadError/);
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
