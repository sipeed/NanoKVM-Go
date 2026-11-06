import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const platformFile = path.join(webRoot, 'src/pages/desktop/menu/fullscreen/platform.ts');
const componentFile = path.join(webRoot, 'src/pages/desktop/menu/fullscreen/index.tsx');

function loadTypeScriptModule(file) {
  const source = fs.readFileSync(file, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: file
  }).outputText;
  const module = { exports: {} };

  vm.runInNewContext(output, { exports: module.exports, module }, { filename: file });
  return module.exports;
}

const platform = loadTypeScriptModule(platformFile);

const userAgents = {
  iphoneSafari:
    'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1',
  iphoneChrome:
    'Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 CriOS/126.0.6478.153 Mobile/15E148 Safari/604.1',
  ipod: 'Mozilla/5.0 (iPod touch; CPU iPhone OS 15_7 like Mac OS X) AppleWebKit/605.1.15 Version/15.0 Mobile/15E148 Safari/604.1',
  ipad: 'Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Version/18.0 Mobile/15E148 Safari/604.1',
  android:
    'Mozilla/5.0 (Linux; Android 15; Pixel 9) AppleWebKit/537.36 Chrome/126.0.0.0 Mobile Safari/537.36',
  desktop:
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 14_6) AppleWebKit/537.36 Chrome/126.0.0.0 Safari/537.36'
};

function environment({
  userAgent = userAgents.desktop,
  platformName = '',
  maxTouchPoints = 0,
  screen = { width: 1440, height: 900 },
  fullscreen = false,
  navigatorStandalone = false,
  displayModeStandalone = false,
  requestFullscreen = async () => {},
  exitFullscreen = async () => {},
  keyboard
} = {}) {
  return {
    document: {
      fullscreenEnabled: fullscreen,
      documentElement: fullscreen ? { requestFullscreen } : {},
      exitFullscreen
    },
    navigator: {
      userAgent,
      platform: platformName,
      maxTouchPoints,
      standalone: navigatorStandalone,
      keyboard
    },
    window: {
      screen,
      matchMedia(query) {
        assert.equal(query, '(display-mode: standalone)');
        return { matches: displayModeStandalone };
      }
    }
  };
}

test('detects iPhone browsers and desktop-site iPhone', () => {
  const safari = environment({ userAgent: userAgents.iphoneSafari });
  const chrome = environment({ userAgent: userAgents.iphoneChrome });
  const desktopSite = environment({
    userAgent: userAgents.desktop,
    platformName: 'MacIntel',
    maxTouchPoints: 5,
    screen: { width: 430, height: 932 }
  });

  assert.equal(platform.isIPhone(safari), true);
  assert.equal(platform.isIPhone(chrome), true);
  assert.equal(platform.isIPhone(desktopSite), true);
  assert.equal(platform.isIPad(desktopSite), false);
  assert.equal(platform.isIPhone(environment({ userAgent: userAgents.ipod })), true);
});

test('does not classify iPad, Android, or desktop browsers as iPhone', () => {
  const ipad = environment({ userAgent: userAgents.ipad });
  const ipadDesktopSite = environment({
    userAgent: userAgents.desktop,
    platformName: 'MacIntel',
    maxTouchPoints: 5,
    screen: { width: 744, height: 1133 }
  });
  const android = environment({ userAgent: userAgents.android });
  const desktop = environment({ userAgent: userAgents.desktop });

  assert.equal(platform.isIPad(ipad), true);
  assert.equal(platform.isIPhone(ipad), false);
  assert.equal(platform.isIPad(ipadDesktopSite), true);
  assert.equal(platform.isIPhone(ipadDesktopSite), false);
  assert.equal(platform.isIPhone(android), false);
  assert.equal(platform.isIPhone(desktop), false);
});

test('detects both iOS and display-mode standalone states', () => {
  assert.equal(platform.isStandalone(environment()), false);
  assert.equal(platform.isStandalone(environment({ navigatorStandalone: true })), true);
  assert.equal(platform.isStandalone(environment({ displayModeStandalone: true })), true);
});

test('selects native fullscreen, iPhone guide, and hidden modes', () => {
  assert.equal(platform.getFullscreenMode(environment({ fullscreen: true })), 'native');
  assert.equal(
    platform.getFullscreenMode(environment({ userAgent: userAgents.ipad, fullscreen: true })),
    'native'
  );
  assert.equal(
    platform.getFullscreenMode(environment({ userAgent: userAgents.iphoneSafari })),
    'guide'
  );
  assert.equal(
    platform.getFullscreenMode(
      environment({
        userAgent: userAgents.iphoneSafari,
        navigatorStandalone: true,
        fullscreen: true
      })
    ),
    'hidden'
  );
  assert.equal(
    platform.getFullscreenMode(
      environment({
        userAgent: userAgents.iphoneSafari,
        displayModeStandalone: true
      })
    ),
    'hidden'
  );
  assert.equal(
    platform.getFullscreenMode(environment({ userAgent: userAgents.android })),
    'hidden'
  );
});

test('requires fullscreenEnabled and requestFullscreen together', () => {
  const enabledWithoutRequest = environment({ fullscreen: true });
  enabledWithoutRequest.document.documentElement = {};
  const requestWithoutEnabled = environment({ fullscreen: false });
  requestWithoutEnabled.document.documentElement.requestFullscreen = async () => {};

  assert.equal(platform.supportsNativeFullscreen(enabledWithoutRequest), false);
  assert.equal(platform.supportsNativeFullscreen(requestWithoutEnabled), false);
  assert.equal(platform.supportsNativeFullscreen(environment({ fullscreen: true })), true);
});

test('requests Keyboard Lock only after fullscreen succeeds', async () => {
  const calls = [];
  const successful = environment({
    fullscreen: true,
    requestFullscreen: async () => calls.push('requestFullscreen'),
    keyboard: {
      lock: async () => calls.push('keyboard.lock')
    }
  });

  await platform.enterNativeFullscreen(successful);
  assert.deepEqual(calls, ['requestFullscreen', 'keyboard.lock']);

  calls.length = 0;
  const rejected = environment({
    fullscreen: true,
    requestFullscreen: async () => {
      calls.push('requestFullscreen');
      throw new Error('denied');
    },
    keyboard: {
      lock: async () => calls.push('keyboard.lock')
    }
  });

  await assert.rejects(platform.enterNativeFullscreen(rejected), /denied/);
  assert.deepEqual(calls, ['requestFullscreen']);
});

test('does not let a pending Keyboard Lock request block fullscreen entry', async () => {
  const calls = [];
  const browser = environment({
    fullscreen: true,
    requestFullscreen: async () => calls.push('requestFullscreen'),
    keyboard: {
      lock: () => {
        calls.push('keyboard.lock');
        return new Promise(() => {});
      }
    }
  });

  await platform.enterNativeFullscreen(browser);
  assert.deepEqual(calls, ['requestFullscreen', 'keyboard.lock']);
});

test('exits fullscreen before releasing Keyboard Lock', async () => {
  const calls = [];
  const browser = environment({
    fullscreen: true,
    exitFullscreen: async () => calls.push('exitFullscreen'),
    keyboard: {
      unlock: async () => calls.push('keyboard.unlock')
    }
  });

  await platform.exitNativeFullscreen(browser);
  assert.deepEqual(calls, ['exitFullscreen', 'keyboard.unlock']);

  calls.length = 0;
  const rejected = environment({
    fullscreen: true,
    exitFullscreen: async () => {
      calls.push('exitFullscreen');
      throw new Error('denied');
    },
    keyboard: {
      unlock: async () => calls.push('keyboard.unlock')
    }
  });

  await assert.rejects(platform.exitNativeFullscreen(rejected), /denied/);
  assert.deepEqual(calls, ['exitFullscreen']);
});

test('fullscreen control remains a landscape-only accessible button', () => {
  const source = fs.readFileSync(componentFile, 'utf8');

  assert.match(source, /<ButtonBase/);
  assert.match(source, /className="hidden[^"\n]*sm:flex"/);
  assert.match(source, /aria-label=\{t\('fullscreen\.toggle'\)\}/);
  assert.match(source, /aria-haspopup=\{capability\.mode === 'guide' \? 'dialog' : undefined\}/);
  assert.match(source, /feedback\.error\(t\('fullscreen\.error'\)\)/);
});

test('closes the fullscreen tooltip while the iPhone guide is open', () => {
  const source = fs.readFileSync(componentFile, 'utf8');

  assert.match(source, /setIsTooltipOpen\(false\);[\s\S]*setIsGuideOpen\(true\);/);
  assert.match(source, /open=\{isTooltipOpen && !isGuideOpen\}/);
  assert.match(source, /onOpenChange=\{\(open\) => setIsTooltipOpen\(open && !isGuideOpen\)\}/);
});

test('uses one browser-independent iPhone sharing guide', () => {
  const componentSource = fs.readFileSync(componentFile, 'utf8');
  const platformSource = fs.readFileSync(platformFile, 'utf8');

  assert.match(componentSource, /t\('fullscreen\.guide\.shareMenu'\)/);
  assert.doesNotMatch(componentSource, /safariShare|chromeShare|isChrome|Chrome iOS|Safari/);
  assert.doesNotMatch(platformSource, /IPhoneBrowser|isChromeIOS|getIPhoneBrowser/);
});

test('ships standalone manifest and keeps the existing favicon without adding a home-screen icon', () => {
  const manifest = JSON.parse(
    fs.readFileSync(path.join(webRoot, 'public/manifest.webmanifest'), 'utf8')
  );
  const html = fs.readFileSync(path.join(webRoot, 'index.html'), 'utf8');

  assert.equal(manifest.name, 'NanoKVM');
  assert.equal(manifest.short_name, 'NanoKVM');
  assert.equal(manifest.start_url, '/');
  assert.equal(manifest.scope, '/');
  assert.equal(manifest.display, 'standalone');
  assert.equal(manifest.background_color, '#000000');
  assert.equal(manifest.theme_color, '#000000');
  assert.equal('icons' in manifest, false);

  assert.match(html, /<link rel="manifest" href="\/manifest\.webmanifest" \/>/);
  assert.doesNotMatch(html, /apple-touch-icon/);
  assert.match(html, /<link rel="icon" type="image\/x-icon" href="\/sipeed\.ico" \/>/);
  assert.match(html, /<meta name="apple-mobile-web-app-capable" content="yes" \/>/);
  assert.match(html, /<meta name="apple-mobile-web-app-status-bar-style" content="black" \/>/);
  assert.match(html, /<meta name="theme-color" content="#000000" \/>/);
  assert.equal(fs.existsSync(path.join(webRoot, 'public/sipeed.ico')), true);
});
