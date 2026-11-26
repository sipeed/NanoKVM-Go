import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';
import vm from 'node:vm';
import ts from 'typescript';

const repoRoot = path.resolve(import.meta.dirname, '..');
const srcRoot = path.join(repoRoot, 'src');
const i18nRoot = path.join(srcRoot, 'i18n');
const localeRoot = path.join(i18nRoot, 'locales');

const allowedSameAsEnglish = new Set([
  'head.terminal',
  'head.wifi',
  'wifi.title',
  'wifi.steps.wifi',
  'wifi.ssidPlaceholder',
  'wifi.passwordPlaceholder',
  'keyboard.leaderKey.recorder.rec',
  'mouse.hidOnly.title',
  'image.cdRom',
  'storage.mountModeCdrom',
  'terminal.title',
  'wol.title',
  'settings.tabs.tailscale',
  'settings.about.mdns',
  'settings.screen.bitrate.title',
  'settings.tailscale.title',
  'settings.tailscale.tailscale',
  'settings.tailscale.url',
  'settings.device.ssh',
  'settings.account.twoFactorProtection.ssh',
  'settings.account.twoFactorProtection.tailscale',
  'settings.account.twoFactorProtection.frp',
  'settings.account.twoFactorProtection.cloudflare',
  'settings.account.twoFactorProtection.zerotier',
  'settings.account.twoFactorProtection.netbird',
  'settings.ai.apiKey',
  'settings.ai.mcpService',
  'settings.ai.mcp',
  'settings.ai.memoryFabricService',
  'settings.ai.memoryFabric',
  'settings.ai.screenTimelapsePlayerUnsupported',
  'settings.device.usbId.title',
  'settings.device.usbId.vid',
  'settings.device.usbId.pid'
]);

const allowedSameAsEnglishByLanguage = {
  // These storage labels are also native words or established loanwords in
  // the listed languages; identical spelling does not indicate missing copy.
  da: new Set([
    'screen.upload',
    'mouse.inputAdapter.touchpad',
    'script.upload',
    'settings.about.information',
    'settings.device.dialBackground.upload',
    'settings.tailscale.run',
    'settings.update.download',
    'settings.update.upload',
    'settings.account.downloadRecoveryCodes',
    'storage.files.sort.type',
    'storage.files.download'
  ]),
  de: new Set(['mouse.inputAdapter.touchpad', 'storage.files.sort.name', 'download.sha256']),
  es: new Set([
    'mouse.scrollNatural',
    'portal.manual',
    'settings.remoteNetwork.status.bad',
    'settings.remoteNetwork.mode.experimental'
  ]),
  fr: new Set([
    'volume.title',
    'mouse.calibration.mode',
    'mouse.calibration.corner',
    'storage.groups.images',
    'storage.files.sort.type'
  ]),
  id: new Set([
    'volume.title',
    'mouse.title',
    'mouse.inputAdapter.touchpad',
    'portal.manual',
    'mouse.calibration.mode',
    'storage.files.folder',
    'storage.files.itemCount_one'
  ]),
  it: new Set([
    // "Audio" is the correct Italian loanword (l'audio), not untranslated English.
    'onboarding.steps.media.title',
    'volume.title',
    'mouse.title',
    'mouse.inputAdapter.touchpad',
    'terminal.password',
    'settings.device.wifi.password',
    'settings.tailscale.account',
    'settings.account.title',
    'settings.account.password'
  ]),
  nb: new Set(['settings.tailscale.run', 'storage.files.sort.type']),
  nl: new Set([
    'onboarding.prompt.later',
    'volume.title',
    'mouse.inputAdapter.touchpad',
    'settings.tailscale.account',
    'settings.account.title',
    'storage.files.sort.type',
    'storage.files.itemCount_one',
    'storage.files.itemCount_other'
  ]),
  pl: new Set(['mouse.inputAdapter.touchpad', 'storage.files.folder'])
};

function readText(file) {
  return fs.readFileSync(file, 'utf8');
}

function listFiles(dir, predicate) {
  const files = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const file = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...listFiles(file, predicate));
    } else if (predicate(file)) {
      files.push(file);
    }
  }
  return files;
}

function flatten(value, prefix = '', output = new Map()) {
  if (typeof value === 'string') {
    output.set(prefix, value);
    return output;
  }

  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    output.set(prefix, value);
    return output;
  }

  for (const [key, child] of Object.entries(value)) {
    flatten(child, prefix ? `${prefix}.${key}` : key, output);
  }
  return output;
}

function loadTsModule(file) {
  const source = readText(file);
  const js = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      esModuleInterop: true
    },
    fileName: file
  }).outputText;

  const context = {
    exports: {},
    module: { exports: {} },
    require(specifier) {
      throw new Error(`Unexpected require(${specifier}) while loading ${file}`);
    }
  };

  vm.runInNewContext(js, context, { filename: file });
  return context.module.exports.default ?? context.exports.default;
}

function loadTsModuleExports(file) {
  const source = readText(file);
  const js = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      esModuleInterop: true
    },
    fileName: file
  }).outputText;

  const context = {
    exports: {},
    module: { exports: {} },
    require(specifier) {
      throw new Error(`Unexpected require(${specifier}) while loading ${file}`);
    }
  };

  vm.runInNewContext(js, context, { filename: file });
  return { ...context.module.exports, ...context.exports };
}

const componentMessagesRoot = path.join(
  srcRoot,
  'pages/desktop/menu/settings/remote-network/components'
);

/**
 * Remote network components ship their own `messages.ts` bundle, so their keys
 * never have to be added to the global locale files. Validate those bundles here
 * and accept their keys as known translations.
 */
function loadComponentMessageBundles(supportedLanguages) {
  const keys = new Set();
  const issues = [];

  if (!fs.existsSync(componentMessagesRoot)) {
    return { keys, issues };
  }

  const bundleFiles = listFiles(componentMessagesRoot, (file) => file.endsWith('messages.ts'));

  for (const file of bundleFiles) {
    const relative = path.relative(repoRoot, file);
    const exports = loadTsModuleExports(file);
    const bundle = Object.values(exports).find(
      (value) => value && typeof value === 'object' && !Array.isArray(value)
    );

    if (!bundle) {
      issues.push(`${relative}: no message bundle export`);
      continue;
    }

    for (const language of supportedLanguages) {
      if (!(language in bundle)) {
        issues.push(`${relative}: missing language ${language}`);
      }
    }
    for (const language of Object.keys(bundle)) {
      if (!supportedLanguages.includes(language)) {
        issues.push(`${relative}: unsupported language ${language}`);
      }
    }

    const referenceKeys = Object.keys(bundle.en ?? {});
    for (const [language, messages] of Object.entries(bundle)) {
      if (!messages || typeof messages !== 'object' || Array.isArray(messages)) {
        issues.push(`${relative} [${language}]: bundle is not an object`);
        continue;
      }

      for (const key of referenceKeys) {
        if (!(key in messages)) issues.push(`${relative} [${language}]: missing key ${key}`);
      }
      for (const key of Object.keys(messages)) {
        if (!referenceKeys.includes(key))
          issues.push(`${relative} [${language}]: extra key ${key}`);
        if (typeof messages[key] !== 'string') {
          issues.push(`${relative} [${language}]: ${key} is not a string`);
        }
      }
    }

    for (const key of referenceKeys) keys.add(key);
  }

  return { keys, issues };
}

function loadLocaleFiles() {
  const retentionMessages = loadTsModule(path.join(i18nRoot, 'retention.ts'));
  const appMessages = loadTsModule(path.join(i18nRoot, 'apps.ts'));
  const fullscreenMessages = loadTsModule(path.join(i18nRoot, 'fullscreen.ts'));
  const updateMessages = loadTsModule(path.join(i18nRoot, 'update.ts'));
  const portalMessages = loadTsModule(path.join(i18nRoot, 'portal.ts'));
  const inputProfileMessages = loadTsModule(path.join(i18nRoot, 'input-profile.ts'));
  const onboardingMessages = loadTsModule(path.join(i18nRoot, 'onboarding.ts'));
  const remoteNetworkMessages = loadTsModule(path.join(i18nRoot, 'remote-network.ts'));
  const localeFiles = fs
    .readdirSync(localeRoot)
    .filter((name) => name.endsWith('.ts'))
    .sort();

  return new Map(
    localeFiles.map((name) => {
      const language = name.replace(/\.ts$/, '');
      const data = loadTsModule(path.join(localeRoot, name));
      const translation = data.translation ?? data;
      const retention = retentionMessages[language];
      const hydrated =
        retention && translation?.settings?.ai
          ? {
              ...translation,
              settings: {
                ...translation.settings,
                ai: {
                  ...translation.settings.ai,
                  retention: retention.retention,
                  screenTimelapseRecordingCleaned: retention.recordingCleaned
                }
              }
            }
          : translation;
      const withApps = {
        ...hydrated,
        settings: {
          ...hydrated.settings,
          apps: appMessages(language)
        }
      };
      const withFullscreen = {
        ...withApps,
        fullscreen: {
          ...withApps.fullscreen,
          ...fullscreenMessages(language)
        }
      };
      const withUpdate = {
        ...withFullscreen,
        settings: {
          ...withFullscreen.settings,
          update: {
            ...withFullscreen.settings.update,
            ...updateMessages(language)
          }
        }
      };
      return [
        language,
        flatten({
          ...withUpdate,
          onboarding: onboardingMessages(language),
          settings: {
            ...withUpdate.settings,
            remoteNetwork: remoteNetworkMessages(language)
          },
          mouse: {
            ...withUpdate.mouse,
            calibration: inputProfileMessages(language)
          },
          portal: portalMessages(language)
        })
      ];
    })
  );
}

function collectLiteralKeysFromSource() {
  const sourceFiles = listFiles(
    srcRoot,
    (file) =>
      /\.(ts|tsx)$/.test(file) &&
      !file.startsWith(localeRoot) &&
      !file.endsWith(path.join('i18n', 'index.ts'))
  );
  const keys = new Map();

  for (const file of sourceFiles) {
    const source = readText(file);
    const ast = ts.createSourceFile(file, source, ts.ScriptTarget.Latest, true);

    function remember(key, node) {
      if (!keys.has(key)) keys.set(key, []);
      const position = ast.getLineAndCharacterOfPosition(node.getStart(ast));
      keys.get(key).push(`${path.relative(repoRoot, file)}:${position.line + 1}`);
    }

    function visit(node) {
      if (ts.isCallExpression(node)) {
        const expression = node.expression;
        const isTranslationCall =
          (ts.isIdentifier(expression) && expression.text === 't') ||
          (ts.isPropertyAccessExpression(expression) && expression.name.text === 't');

        if (isTranslationCall) {
          const [firstArg] = node.arguments;
          if (firstArg && ts.isStringLiteralLike(firstArg)) {
            remember(firstArg.text, firstArg);
          }
        }
      }

      ts.forEachChild(node, visit);
    }

    visit(ast);
  }

  return keys;
}

function getSupportedLanguages() {
  const locale = readText(path.join(i18nRoot, 'locale.ts'));
  const match = locale.match(/supportedLanguageKeys\s*=\s*\[([\s\S]*?)\]\s*as const/);
  if (!match) throw new Error('Unable to find supportedLanguageKeys in locale.ts');
  return [...match[1].matchAll(/['"]([^'"]+)['"]/g)].map((item) => item[1]);
}

function getLanguageOptions() {
  const languages = readText(path.join(i18nRoot, 'languages.ts'));
  return [...languages.matchAll(/\{\s*key:\s*['"]([^'"]+)['"]/g)].map((item) => item[1]);
}

function isSuspiciousSameAsEnglish(language, key, value) {
  if (allowedSameAsEnglish.has(key)) return false;
  if (allowedSameAsEnglishByLanguage[language]?.has(key)) return false;
  if (!/[A-Za-z]{3,}/.test(value)) return false;
  if (/^[-+]?\d+(?:\.\d+)?\s*(?:fps|Hz|Mbps|Kbps)?$/i.test(value)) return false;
  if (/^(ok|on|off|yes|no)$/i.test(value)) return true;
  if (value.length <= 2) return false;
  return true;
}

function main() {
  const errors = [];
  const localeMaps = loadLocaleFiles();
  const english = localeMaps.get('en');
  if (!english) throw new Error('Missing en locale');

  const supportedLanguages = getSupportedLanguages();
  const languageOptions = getLanguageOptions();
  const localeLanguages = [...localeMaps.keys()].sort();

  for (const language of supportedLanguages) {
    if (!localeMaps.has(language))
      errors.push(`Missing locale file for supported language: ${language}`);
    if (!languageOptions.includes(language))
      errors.push(`Missing language picker option: ${language}`);
  }

  for (const language of localeLanguages) {
    if (!supportedLanguages.includes(language))
      errors.push(`Locale file is not supported: ${language}`);
  }

  for (const language of languageOptions) {
    if (!supportedLanguages.includes(language))
      errors.push(`Language picker option is not supported: ${language}`);
  }

  const englishKeys = [...english.keys()].sort();
  for (const [language, translations] of localeMaps) {
    const translationKeys = [...translations.keys()].sort();
    const missing = englishKeys.filter((key) => !translations.has(key));
    const extra = translationKeys.filter((key) => !english.has(key));

    for (const key of missing) errors.push(`${language}: missing key ${key}`);
    for (const key of extra) errors.push(`${language}: extra key ${key}`);
  }

  const componentBundles = loadComponentMessageBundles(supportedLanguages);
  for (const issue of componentBundles.issues) errors.push(`component messages: ${issue}`);

  const usedKeys = collectLiteralKeysFromSource();
  for (const [key, locations] of usedKeys) {
    if (!english.has(key) && !componentBundles.keys.has(key)) {
      errors.push(`Missing English key used at ${locations.join(', ')}: ${key}`);
    }
  }

  for (const [language, translations] of localeMaps) {
    if (language === 'en') continue;
    for (const [key, englishValue] of english) {
      const value = translations.get(key);
      if (typeof englishValue !== 'string' || typeof value !== 'string') continue;
      if (value === englishValue && isSuspiciousSameAsEnglish(language, key, value)) {
        errors.push(`${language}: suspicious untranslated English at ${key}: ${value}`);
      }
    }
  }

  if (errors.length > 0) {
    console.error(`i18n audit failed with ${errors.length} issue(s):`);
    for (const error of errors) console.error(`- ${error}`);
    process.exit(1);
  }

  console.log(
    `i18n audit passed for ${localeMaps.size} locale(s) and ${englishKeys.length} key(s).`
  );
}

main();
