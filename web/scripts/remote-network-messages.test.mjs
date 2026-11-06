import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const sectionRoot = 'src/pages/desktop/menu/settings/remote-network';

function runModule(relativePath, modules) {
  const file = path.join(webRoot, relativePath);
  const output = ts.transpileModule(fs.readFileSync(file, 'utf8'), {
    compilerOptions: {
      esModuleInterop: true,
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
      require(id) {
        if (!(id in modules)) throw new Error(`unexpected module: ${id}`);
        return modules[id];
      }
    },
    { filename: file }
  );

  return module.exports;
}

const bundle = {
  en: {
    'settings.tailscale.title': 'Tailscale',
    'settings.tailscale.subnetRouteDesc': 'Advertise {{subnet}} to your tailnet.'
  },
  zh: {
    'settings.tailscale.title': 'Tailscale 中文',
    'settings.tailscale.subnetRouteDesc': '向 tailnet 发布 {{subnet}}。'
  }
};

/** Loads the hook with a stubbed global i18n bundle. */
function loadMessages({ language = 'zh', contextMessages = bundle } = {}) {
  const globalCalls = [];
  const interpolations = [];
  const i18n = {
    language,
    services: {
      interpolator: {
        interpolate(message, data, lng, options) {
          interpolations.push({ message, data, lng, options });
          return `${message} [${Object.values(data).join(',')}]`;
        }
      }
    }
  };

  const module = runModule(`${sectionRoot}/messages.ts`, {
    react: {
      createContext: (initial) => ({ initial }),
      useCallback: (callback) => callback,
      useContext: () => contextMessages
    },
    i18next: {},
    'react-i18next': {
      useTranslation: () => ({
        t: (key, options) => {
          globalCalls.push({ key, options });
          return `global:${key}`;
        },
        i18n
      })
    },
    '@/i18n/locale.ts': {}
  });

  return { module, globalCalls, interpolations };
}

test('component bundles win over the global i18n bundle', () => {
  const { module } = loadMessages({ language: 'zh' });
  const t = module.useRemoteNetworkMessages();

  assert.equal(t('settings.tailscale.title'), 'Tailscale 中文');
});

test('a language without the key falls back to the English component bundle', () => {
  const { module, globalCalls } = loadMessages({
    language: 'th',
    contextMessages: { en: { 'settings.tailscale.title': 'Tailscale' } }
  });
  const t = module.useRemoteNetworkMessages();

  assert.equal(t('settings.tailscale.title'), 'Tailscale');
  assert.equal(globalCalls.length, 0);
});

test('keys outside the component bundle fall back to the global i18n bundle', () => {
  const { module, globalCalls } = loadMessages({ language: 'zh' });
  const t = module.useRemoteNetworkMessages();

  assert.equal(t('settings.back'), 'global:settings.back');
  assert.deepEqual(globalCalls, [{ key: 'settings.back', options: undefined }]);
});

test('component messages keep i18next interpolation', () => {
  const { module, interpolations } = loadMessages({ language: 'zh' });
  const t = module.useRemoteNetworkMessages();

  const text = t('settings.tailscale.subnetRouteDesc', { subnet: '192.168.0.0/22' });

  assert.equal(text, '向 tailnet 发布 {{subnet}}。 [192.168.0.0/22]');
  assert.deepEqual(interpolations[0].data, { subnet: '192.168.0.0/22' });
  assert.equal(interpolations[0].lng, 'zh');
});

test('without a component bundle every key uses the global i18n bundle', () => {
  const { module, globalCalls } = loadMessages({ language: 'zh', contextMessages: null });
  const t = module.useRemoteNetworkMessages();

  assert.equal(t('settings.tailscale.title'), 'global:settings.tailscale.title');
  assert.equal(globalCalls.length, 1);
});

test('every component ships a message bundle for every supported language', () => {
  const messagesSource = fs.readFileSync(
    path.join(webRoot, `${sectionRoot}/components/tailscale/messages.ts`),
    'utf8'
  );

  assert.match(messagesSource, /export const tailscaleMessages: RemoteNetworkComponentMessages/);
  for (const language of [
    'en',
    'nl',
    'da',
    'de',
    'es',
    'fr',
    'id',
    'it',
    'pl',
    'ru',
    'ko',
    'zh',
    'zh_tw',
    'hu',
    'vi',
    'ja',
    'cz',
    'uk',
    'nb',
    'th'
  ]) {
    assert.match(messagesSource, new RegExp(`^  ${language}: \\{`, 'm'), `missing ${language}`);
  }
});

test('each component language bundle ships the connection mode strings', () => {
  const { tailscaleMessages } = runModule(`${sectionRoot}/components/tailscale/messages.ts`, {});
  const languageKeys = Object.keys(tailscaleMessages);

  assert.equal(languageKeys.length, 20);

  const modeKeys = [
    'settings.tailscale.mode.official',
    'settings.tailscale.mode.officialDesc',
    'settings.tailscale.mode.sipeed',
    'settings.tailscale.mode.sipeedDesc',
    'settings.tailscale.sipeed.addDevice',
    'settings.tailscale.sipeed.addDeviceDesc'
  ];

  for (const language of languageKeys) {
    for (const key of modeKeys) {
      const value = tailscaleMessages[language][key];
      assert.equal(typeof value, 'string', `${language} is missing ${key}`);
      assert.notEqual(value, '', `${language} has an empty ${key}`);
    }
  }

  for (const language of ['en', 'zh']) {
    for (const key of [
      'settings.tailscale.sipeed.enableDesc',
      'settings.tailscale.sipeed.enabling',
      'settings.tailscale.sipeed.enableFailed',
      'settings.tailscale.sipeed.retry',
      'settings.tailscale.sipeed.connected',
      'settings.tailscale.sipeed.emptyDevices'
    ]) {
      assert.equal(
        typeof tailscaleMessages[language][key],
        'string',
        `${language} is missing ${key}`
      );
    }
  }
});

test('every non-English bundle localizes the Sipeed enrollment runtime', () => {
  const { tailscaleMessages } = runModule(`${sectionRoot}/components/tailscale/messages.ts`, {});
  const keys = [
    'settings.tailscale.sipeed.enableDesc',
    'settings.tailscale.sipeed.enabling',
    'settings.tailscale.sipeed.enableFailed',
    'settings.tailscale.sipeed.officialActive',
    'settings.tailscale.sipeed.invalidIdentity',
    'settings.tailscale.sipeed.identityMissing',
    'settings.tailscale.sipeed.rateLimited',
    'settings.tailscale.sipeed.pairingUnavailable',
    'settings.tailscale.sipeed.pairingUpstreamError',
    'settings.tailscale.sipeed.pairingRejected',
    'settings.tailscale.sipeed.nodeNotFound',
    'settings.tailscale.sipeed.protocolError',
    'settings.tailscale.sipeed.localAPIUnavailable',
    'settings.tailscale.sipeed.stateFileInvalid',
    'settings.tailscale.sipeed.stateWriteFailed',
    'settings.tailscale.sipeed.installFailed',
    'settings.tailscale.sipeed.profileReadFailed',
    'settings.tailscale.sipeed.hostnameUnavailable',
    'settings.tailscale.sipeed.profileSetupFailed',
    'settings.tailscale.sipeed.profileStartFailed',
    'settings.tailscale.sipeed.connectionFailed',
    'settings.tailscale.sipeed.profileVerificationFailed',
    'settings.tailscale.sipeed.rollbackFailed',
    'settings.tailscale.sipeed.interrupted',
    'settings.tailscale.sipeed.identityGenerationFailed',
    'settings.tailscale.sipeed.retry',
    'settings.tailscale.sipeed.localStatus',
    'settings.tailscale.sipeed.connected',
    'settings.tailscale.sipeed.copyIP',
    'settings.tailscale.sipeed.ipCopied',
    'settings.tailscale.sipeed.devices',
    'settings.tailscale.sipeed.emptyDevices'
  ];

  for (const [language, bundle] of Object.entries(tailscaleMessages)) {
    if (language === 'en') continue;

    for (const key of keys) {
      assert.equal(typeof bundle[key], 'string', `${language} is missing ${key}`);
      assert.notEqual(
        bundle[key],
        tailscaleMessages.en[key],
        `${language} fell back to English for ${key}`
      );
    }
  }
});

test('the English bundle covers every Tailscale key referenced by the component', () => {
  const { tailscaleMessages } = runModule(`${sectionRoot}/components/tailscale/messages.ts`, {});
  const componentDirectory = path.join(webRoot, `${sectionRoot}/components/tailscale`);
  const referencedKeys = new Set();

  for (const entry of fs.readdirSync(componentDirectory, { withFileTypes: true })) {
    if (!entry.isFile() || !/\.tsx?$/.test(entry.name) || entry.name === 'messages.ts') continue;

    const componentSource = fs.readFileSync(path.join(componentDirectory, entry.name), 'utf8');
    for (const match of componentSource.matchAll(
      /settings\.tailscale(?:\.[A-Za-z][A-Za-z0-9]*)+/g
    )) {
      referencedKeys.add(match[0]);
    }
  }

  assert.ok(referencedKeys.size > 0, 'no Tailscale translation keys were discovered');
  for (const key of referencedKeys) {
    assert.equal(typeof tailscaleMessages.en[key], 'string', `English bundle is missing ${key}`);
  }
});

test('non-English bundles only retain language-neutral English text', () => {
  const { tailscaleMessages } = runModule(`${sectionRoot}/components/tailscale/messages.ts`, {});
  const languageNeutralKeys = new Set([
    'tailscale.offline',
    'settings.tailscale.title',
    'settings.tailscale.account',
    'settings.tailscale.run',
    'settings.tailscale.sipeed.pendingClient',
    'settings.tailscale.sipeed.deviceOnline',
    'settings.tailscale.sipeed.deviceOffline'
  ]);

  for (const [language, bundle] of Object.entries(tailscaleMessages)) {
    if (language === 'en') continue;

    const fallbackKeys = Object.entries(bundle)
      .filter(([key, value]) => value === tailscaleMessages.en[key])
      .map(([key]) => key)
      .filter((key) => !languageNeutralKeys.has(key));

    assert.deepEqual(fallbackKeys, [], `${language} contains English fallback text`);
  }
});

test('every Tailscale locale has the same message keys as English', () => {
  const { tailscaleMessages } = runModule(`${sectionRoot}/components/tailscale/messages.ts`, {});
  const englishKeys = Object.keys(tailscaleMessages.en).sort();

  for (const [language, bundle] of Object.entries(tailscaleMessages)) {
    assert.deepEqual(Object.keys(bundle).sort(), englishKeys, `${language} message keys differ`);
  }
});

test('every non-English bundle localizes the Sipeed management workflow', () => {
  const { tailscaleMessages } = runModule(`${sectionRoot}/components/tailscale/messages.ts`, {});
  const keys = [
    'settings.tailscale.sipeed.exitDesc',
    'settings.tailscale.sipeed.claimHint',
    'settings.tailscale.sipeed.pendingMismatch',
    'settings.tailscale.sipeed.backupCodeDesc',
    'settings.tailscale.sipeed.managementFailed',
    'settings.tailscale.sipeed.removeConfirm'
  ];

  for (const [language, bundle] of Object.entries(tailscaleMessages)) {
    if (language === 'en') continue;
    for (const key of keys) {
      assert.equal(typeof bundle[key], 'string', `${language} is missing ${key}`);
      assert.notEqual(
        bundle[key],
        tailscaleMessages.en[key],
        `${language} did not localize ${key}`
      );
    }
  }
});
