import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { after, before, test } from 'node:test';
import { createServer } from 'vite';

const webRoot = path.resolve(import.meta.dirname, '..');
let vite;
let model;
let locale;
let images;
let panelSource;
let imagesSource;
let apiSource;
let menuSource;

const defaultConfig = {
  version: 1,
  revision: 1,
  activeArm: 'short',
  strengthByArm: { long: 3, short: 3 },
  pressMs: 800,
  speed: 'slow'
};

const normalResponse = {
  enabled: true,
  available: true,
  config: defaultConfig,
  profiles: [
    { id: 'long', displayName: '长摆臂', defaultStrength: 3 },
    { id: 'short', displayName: '短摆臂', defaultStrength: 3 }
  ],
  strength: { min: 1, max: 5, default: 3 },
  pressMs: { min: 100, max: 30_000, default: 800 },
  speedOptions: ['slow', 'fast'],
  defaultSpeed: 'slow',
  status: { running: false, state: 'idle', result: 'idle', error: '' }
};

function clone(value) {
  return structuredClone(value);
}

function responseWith(change) {
  return Object.assign(clone(normalResponse), change);
}

before(async () => {
  vite = await createServer({
    root: webRoot,
    appType: 'custom',
    logLevel: 'error',
    server: { middlewareMode: true, hmr: false, ws: false }
  });

  [model, locale, images, panelSource, imagesSource, apiSource, menuSource] = await Promise.all([
    vite.ssrLoadModule('/src/pages/desktop/menu/finger-robot/finger-robot-model.ts'),
    vite.ssrLoadModule('/src/pages/desktop/menu/finger-robot/finger-robot-text.ts'),
    vite.ssrLoadModule('/src/pages/desktop/menu/finger-robot/finger-robot-images.ts'),
    readFile(path.join(webRoot, 'src/pages/desktop/menu/finger-robot/index.tsx'), 'utf8'),
    readFile(
      path.join(webRoot, 'src/pages/desktop/menu/finger-robot/finger-robot-images.ts'),
      'utf8'
    ),
    readFile(path.join(webRoot, 'src/api/vm.ts'), 'utf8'),
    readFile(path.join(webRoot, 'src/pages/desktop/menu/index.tsx'), 'utf8')
  ]);
});

after(async () => {
  await vite.close();
});

test('parses the product contract and keeps PWM implementation fields private', () => {
  const response = responseWith({
    releaseUs: 2050,
    pressUs: 1450,
    pressSpeedUsPerSecond: 500,
    config: {
      ...defaultConfig,
      releaseUs: 2050,
      pressUs: 1450,
      pressSpeedUsPerSecond: 500
    },
    profiles: normalResponse.profiles.map((profile) => ({
      ...profile,
      minSafePressUs: 889,
      pressUsByStrength: [1550, 1500, 1450, 1400, 1350]
    }))
  });

  const parsed = model.parseFingerRobotData(response);
  assert.ok(parsed);
  assert.deepEqual(parsed.config, defaultConfig);
  assert.deepEqual(parsed.profiles, normalResponse.profiles);
  assert.deepEqual(parsed.strength, { min: 1, max: 5, default: 3 });
  assert.deepEqual(parsed.pressMs, { min: 100, max: 30_000, default: 800 });
  assert.deepEqual(parsed.speedOptions, ['slow', 'fast']);
  assert.equal(parsed.defaultSpeed, 'slow');

  const serialized = JSON.stringify(parsed);
  for (const privateField of [
    'releaseUs',
    'pressUs',
    'pressSpeedUsPerSecond',
    'minSafePressUs',
    'pressUsByStrength'
  ]) {
    assert.doesNotMatch(serialized, new RegExp(`"${privateField}"`));
  }
});

test('rejects incomplete, duplicate, and malformed product responses', () => {
  assert.equal(model.parseFingerRobotData(null), null);
  assert.equal(model.parseFingerRobotData({}), null);
  assert.equal(model.parseFingerRobotData(responseWith({ enabled: 'true' })), null);
  assert.equal(
    model.parseFingerRobotData(responseWith({ strength: { min: 1, max: 5, default: 6 } })),
    null
  );
  assert.equal(
    model.parseFingerRobotData(responseWith({ strength: { min: 1, max: 6, default: 3 } })),
    null
  );
  assert.equal(
    model.parseFingerRobotData(responseWith({ profiles: [normalResponse.profiles[0]] })),
    null
  );
  assert.equal(
    model.parseFingerRobotData(
      responseWith({ profiles: [normalResponse.profiles[0], normalResponse.profiles[0]] })
    ),
    null
  );
  assert.equal(
    model.parseFingerRobotData(
      responseWith({
        profiles: [
          normalResponse.profiles[0],
          { id: 'medium', displayName: '中摆臂', defaultStrength: 3 }
        ]
      })
    ),
    null
  );
  assert.equal(model.parseFingerRobotData(responseWith({ speedOptions: ['slow', 'slow'] })), null);
  assert.equal(
    model.parseFingerRobotData(
      responseWith({ status: { running: false, state: 'unknown', result: 'idle', error: '' } })
    ),
    null
  );
  assert.equal(
    model.parseFingerRobotData(
      responseWith({ status: { running: false, state: 'idle', result: 'unknown', error: '' } })
    ),
    null
  );

  const capabilityOnly = model.parseFingerRobotData(responseWith({ config: null }));
  assert.ok(capabilityOnly);
  assert.equal(capabilityOnly.config, undefined);
});

test('validates complete versioned configurations and advertised ranges', () => {
  assert.deepEqual(model.parseFingerRobotConfig(defaultConfig), defaultConfig);

  for (const invalid of [
    { ...defaultConfig, version: 2 },
    { ...defaultConfig, revision: 0 },
    { ...defaultConfig, revision: 1.5 },
    { ...defaultConfig, activeArm: 'medium' },
    { ...defaultConfig, strengthByArm: { long: 0, short: 3 } },
    { ...defaultConfig, strengthByArm: { long: 3, short: 6 } },
    { ...defaultConfig, strengthByArm: { long: 3 } },
    { ...defaultConfig, pressMs: 99 },
    { ...defaultConfig, pressMs: 30_001 },
    { ...defaultConfig, pressMs: 800.5 },
    { ...defaultConfig, speed: 'medium' }
  ]) {
    assert.equal(model.parseFingerRobotConfig(invalid), null);
  }

  assert.ok(model.parseFingerRobotConfig({ ...defaultConfig, pressMs: 100 }));
  assert.ok(model.parseFingerRobotConfig({ ...defaultConfig, pressMs: 30_000 }));
  assert.equal(
    model.parseFingerRobotConfig(
      { ...defaultConfig, speed: 'fast' },
      normalResponse.strength,
      normalResponse.pressMs,
      ['slow']
    ),
    null
  );
});

test('arm selection preserves each arm strength and every config field', () => {
  const source = { ...defaultConfig, revision: 9, pressMs: 1200, speed: 'fast' };
  const longArm = model.selectFingerRobotArm(source, 'long');
  const longLevelFour = model.selectFingerRobotStrength(longArm, 4, normalResponse.strength);
  assert.ok(longLevelFour);

  const shortArm = model.selectFingerRobotArm(longLevelFour, 'short');
  const shortLevelTwo = model.selectFingerRobotStrength(shortArm, 2, normalResponse.strength);
  assert.ok(shortLevelTwo);

  const longAgain = model.selectFingerRobotArm(shortLevelTwo, 'long');
  assert.equal(model.getActiveFingerRobotStrength(longAgain), 4);
  assert.deepEqual(longAgain, {
    version: 1,
    revision: 9,
    activeArm: 'long',
    strengthByArm: { long: 4, short: 2 },
    pressMs: 1200,
    speed: 'fast'
  });

  const shortAgain = model.selectFingerRobotArm(longAgain, 'short');
  assert.equal(model.getActiveFingerRobotStrength(shortAgain), 2);
  assert.deepEqual(
    source.strengthByArm,
    { long: 3, short: 3 },
    'selectors must not mutate saved data'
  );

  const copy = model.cloneFingerRobotConfig(longAgain);
  assert.deepEqual(copy, longAgain);
  assert.notEqual(copy.strengthByArm, longAgain.strengthByArm);
});

test('strength, duration, and speed selectors enforce product choices', () => {
  assert.ok(model.selectFingerRobotStrength(defaultConfig, 1, normalResponse.strength));
  assert.ok(model.selectFingerRobotStrength(defaultConfig, 5, normalResponse.strength));
  assert.equal(model.selectFingerRobotStrength(defaultConfig, 0, normalResponse.strength), null);
  assert.equal(model.selectFingerRobotStrength(defaultConfig, 6, normalResponse.strength), null);
  assert.equal(model.selectFingerRobotStrength(defaultConfig, 2.5, normalResponse.strength), null);

  assert.equal(
    model.selectFingerRobotPressMS(defaultConfig, 100, normalResponse.pressMs)?.pressMs,
    100
  );
  assert.equal(
    model.selectFingerRobotPressMS(defaultConfig, 30_000, normalResponse.pressMs)?.pressMs,
    30_000
  );
  assert.equal(model.selectFingerRobotPressMS(defaultConfig, 99, normalResponse.pressMs), null);
  assert.equal(model.selectFingerRobotPressMS(defaultConfig, 30_001, normalResponse.pressMs), null);
  assert.equal(model.selectFingerRobotPressMS(defaultConfig, 800.5, normalResponse.pressMs), null);

  assert.equal(
    model.selectFingerRobotSpeed(defaultConfig, 'fast', normalResponse.speedOptions)?.speed,
    'fast'
  );
  assert.equal(model.selectFingerRobotSpeed(defaultConfig, 'fast', ['slow']), null);
});

test('maps controller and local operation states to product states', () => {
  const status = (state, result = 'running', running = true, error = '') => ({
    running,
    state,
    result,
    error
  });

  assert.equal(model.resolveFingerRobotDisplayState(status('idle', 'idle', false)), 'idle');
  assert.equal(model.resolveFingerRobotDisplayState(status('idle')), 'moving');
  assert.equal(model.resolveFingerRobotDisplayState(status('pressing')), 'moving');
  assert.equal(model.resolveFingerRobotDisplayState(status('resetting')), 'moving');
  assert.equal(model.resolveFingerRobotDisplayState(status('holding')), 'holding');
  assert.equal(model.resolveFingerRobotDisplayState(status('releasing')), 'releasing');
  assert.equal(model.resolveFingerRobotDisplayState(status('idle', 'success', false)), 'completed');
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'cancelled', false)),
    'completed'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'reset', false)),
    'armReplacement'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'reset', false), {
      replaceArmPending: true
    }),
    'moving'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'reset', false), {
      releasePending: true
    }),
    'releasing'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('releasing', 'idle', false)),
    'releasing'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'idle', false), { busy: true }),
    'busy'
  );
  assert.equal(model.resolveFingerRobotDisplayState(status('idle', 'failed', false)), 'failed');
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'idle', false, 'operation failed')),
    'failed'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'idle', false), { failed: true }),
    'failed'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'idle', false), {
      releasePending: true
    }),
    'releasing'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('idle', 'success', false), {
      pressPending: true
    }),
    'moving'
  );
  assert.equal(
    model.resolveFingerRobotDisplayState(status('holding'), { pressPending: true }),
    'holding',
    'polled controller phases must take priority over the pending HTTP request'
  );
});

test('provides explicit English and Chinese product copy', () => {
  const english = locale.getFingerRobotText('en-US');
  const chinese = locale.getFingerRobotText('zh-CN');
  assert.equal(english.title, 'Finger Robot');
  assert.equal(english.longArm, 'Long arm');
  assert.equal(english.shortArm, 'Short arm');
  assert.equal(english.loading, 'Loading');
  assert.equal(english.pressOnce, 'Press once');
  assert.equal(english.releaseNow, 'Release now');
  assert.equal(english.replaceArm, 'Replace arm');
  assert.equal(english.replaceArmConfirm, 'Move the arm to the replacement position?');
  assert.equal(english.unavailable, 'Finger Robot is not connected.');
  assert.equal(english.level(5), 'Level 5');
  assert.equal(chinese.title, '手指机器人');
  assert.equal(chinese.longArm, '长摆臂');
  assert.equal(chinese.shortArm, '短摆臂');
  assert.equal(chinese.loading, '加载中');
  assert.equal(chinese.pressOnce, '按压一次');
  assert.equal(chinese.releaseNow, '立即松开');
  assert.equal(chinese.replaceArm, '更换摆臂');
  assert.equal(chinese.replaceArmConfirm, '确认将摆臂移动到更换位置？');
  assert.equal(chinese.unavailable, '手指机器人未连接。');
  assert.equal(chinese.level(5), '第 5 档');
  assert.equal(locale.getFingerRobotText('zh-TW').holding, '保持中');
  assert.equal(locale.getFingerRobotText('de').title, 'Fingerroboter');
  // 未支持的语言仍须回退英文
  assert.equal(locale.getFingerRobotText('xx-YY').title, 'Finger Robot');
  assert.equal(locale.formatFingerRobotDisplayState('moving', chinese), '移动中');
  assert.equal(locale.formatFingerRobotDisplayState('busy', english), 'Busy');
  assert.equal(locale.formatFingerRobotDisplayState('armReplacement', chinese), '可更换摆臂');

  for (const key of Object.keys(locale.fingerRobotText.en)) {
    const englishValue = locale.fingerRobotText.en[key];
    const chineseValue = locale.fingerRobotText.zh[key];
    if (typeof englishValue !== 'string' || typeof chineseValue !== 'string') {
      continue;
    }

    assert.equal(
      /[.!?]$/.test(englishValue),
      /[。！？]$/.test(chineseValue),
      `${key} must use matching terminal punctuation in English and Chinese`
    );
  }
});

test('arm replacement copy is available in every supported language without PWM details', () => {
  for (const [language, text] of Object.entries(locale.fingerRobotText)) {
    for (const key of [
      'replaceArm',
      'replaceArmConfirm',
      'replaceArmHint',
      'armReplacement',
      'replaceArmFailed'
    ]) {
      assert.equal(typeof text[key], 'string', `${language}.${key}`);
      assert.ok(text[key].trim(), `${language}.${key} must not be empty`);
      assert.doesNotMatch(text[key], /1500|2050|PWM|\b(?:us|ms)\b/);
    }
  }
});

test('maps every arm and strength level to its bundled image', () => {
  assert.deepEqual(images.fingerRobotImageLevels, [1, 2, 3, 4, 5]);

  for (const arm of ['short', 'long']) {
    for (const level of images.fingerRobotImageLevels) {
      assert.ok(Object.hasOwn(images.armImageMap[arm], level));
      const source = images.armImageMap[arm][level];
      assert.equal(typeof source, 'string');
      const isInline = /^data:image\/webp[,;]/.test(source);
      assert.ok(
        isInline || new RegExp(`/${arm}-l${level}\\.webp(?:\\?|$)`).test(source),
        `${arm} L${level} must resolve to its bundled WebP`
      );
      assert.match(imagesSource, new RegExp(`${level}: resolveArmImage\\('${arm}', ${level}\\)`));
    }
  }

  assert.match(imagesSource, /import\.meta\.glob<string>\(/);
  assert.match(imagesSource, /finger-robot\/short-l\{1,2,3,4,5\}\.webp/);
  assert.match(imagesSource, /finger-robot\/long-l\{1,2,3,4,5\}\.webp/);
});

test('polling remains active for an open panel, pending action, or running controller', () => {
  assert.equal(model.shouldPollFingerRobotStatus(false, false, false, false), false);
  assert.equal(model.shouldPollFingerRobotStatus(true, false, false, false), true);
  assert.equal(model.shouldPollFingerRobotStatus(false, true, false, false), true);
  assert.equal(model.shouldPollFingerRobotStatus(false, false, true, false), true);
  assert.equal(model.shouldPollFingerRobotStatus(false, false, false, true), true);
  assert.equal(model.shouldPollFingerRobotStatus(true, true, true, true), true);
});

test('an unavailable robot only triggers connection rechecks while its panel is open', () => {
  const unavailable = model.parseFingerRobotData(responseWith({ available: false }));
  assert.ok(unavailable);
  assert.equal(unavailable.enabled, true);
  assert.equal(unavailable.available, false);
  assert.equal(
    model.shouldPollFingerRobotStatus(false, false, false, unavailable.status.running),
    false,
    'enabling the menu without a connected robot must not start background polling'
  );
  assert.equal(
    model.shouldPollFingerRobotStatus(true, false, false, unavailable.status.running),
    true,
    'opening the panel must still detect a newly connected robot'
  );

  const english = locale.getFingerRobotText('en');
  const chinese = locale.getFingerRobotText('zh');
  assert.equal(english.reconnecting, 'Rechecking the connection');
  assert.equal(chinese.reconnecting, '正在重新检测连接');
  assert.match(panelSource, /Boolean\(data\?\.enabled && !data\.available\)/);
  assert.doesNotMatch(panelSource, /description=\{text\.reconnecting\}/);
});

test('only sensitive-action codes are returned to the challenge gate', () => {
  assert.equal(model.isFingerRobotChallengeResponse(-3), true);
  assert.equal(model.isFingerRobotChallengeResponse(-4), true);
  for (const code of [0, -1, -2, -5, -6]) {
    assert.equal(model.isFingerRobotChallengeResponse(code), false);
  }
});

test('top menu uses the new APIs, one shared gate, and a five-stop commit action', () => {
  assert.doesNotMatch(panelSource, /\bsetPower\b/);
  assert.doesNotMatch(panelSource, /['"]\/api\/vm\/power['"]/);
  assert.doesNotMatch(panelSource, /\bPowerShort\b|\bPowerLong\b/);
  assert.match(panelSource, /api\.setFingerRobot\(/);
  assert.match(panelSource, /api\.pressFingerRobot\(/);
  assert.match(panelSource, /api\.releaseFingerRobot\(/);
  assert.match(panelSource, /api\.replaceFingerRobotArm\(/);

  assert.equal(
    [...panelSource.matchAll(/useProtectionChallengeGate\(\)/g)].length,
    1,
    'press and arm replacement must share one long-lived challenge gate'
  );
  assert.match(panelSource, /const challengeGate = useProtectionChallengeGate\(\);/);
  assert.equal(
    [...panelSource.matchAll(/challengeGate\s*\.runWithChallenge\(/g)].length,
    2,
    'both gated actuations (press and arm replacement) must run through the shared gate'
  );

  const releaseDisabled = panelSource.match(/const releaseDisabled =\s*([^;]+);/s)?.[1];
  assert.ok(releaseDisabled, 'release disabled policy must remain explicit');
  assert.match(
    releaseDisabled,
    /!data\?\.enabled\s*\|\|\s*!data\.available/,
    'release must be disabled while the controller is disabled or unavailable'
  );
  assert.doesNotMatch(
    releaseDisabled,
    /pressPending|challengeGate|data\.status\.running/,
    'a pending or busy press must never block the independent release action'
  );
  assert.match(
    panelSource,
    /const release = useCallback\(\(\) => \{\s+if \(\s*!data\?\.enabled \|\|\s*!data\.available/,
    'the release handler must reject stale clicks after availability is lost'
  );

  assert.match(panelSource, /step=\{null\}/);
  assert.match(panelSource, /step=\{100\}/);
  assert.match(panelSource, /onChangeComplete=\{updateStrength\}/);
  assert.match(
    panelSource,
    /Array\.from\(\{ length: data\.strength\.max - data\.strength\.min \+ 1 \}/
  );
  assert.doesNotMatch(
    panelSource,
    /\b(?:releaseUs|pressUs|pressSpeedUsPerSecond|minSafePressUs|pressUsByStrength)\b/
  );
});

test('arm replacement requires confirmation and a detected servo', () => {
  const confirmation = panelSource.match(
    /<Popconfirm\s+title=\{text\.replaceArmConfirm\}[\s\S]*?<\/Popconfirm>/
  )?.[0];
  assert.ok(confirmation, 'the arm replacement button must be inside its confirmation');
  assert.match(confirmation, /onConfirm=\{replaceArm\}/);
  assert.doesNotMatch(confirmation, /onClick=\{replaceArm\}/);
  assert.match(confirmation, /okButtonProps=\{\{ disabled: replaceArmDisabled \}\}/);
  assert.match(confirmation, /disabled=\{replaceArmDisabled\}/);
  assert.match(confirmation, /\{text\.replaceArm\}/);

  const disabled = panelSource.match(/const replaceArmDisabled =\s*([^;]+);/s)?.[1];
  assert.ok(disabled);
  assert.match(disabled, /!data\?\.enabled\s*\|\|\s*!data\.available/);
  assert.match(
    panelSource,
    /const replaceArm = useCallback\(\(\) => \{\s+if \(\s*!data\?\.enabled \|\|\s*!data\.available/,
    'confirmation must recheck availability when it is accepted'
  );
});

test('arm cards follow preview strength, preload five states, and fall back to text', () => {
  assert.match(panelSource, /armImageMap\[profile\.id\]\[strength\]/);
  assert.match(panelSource, /fingerRobotImageLevels\.flatMap\(/);
  assert.match(panelSource, /new Image\(\)/);
  assert.match(panelSource, /image\.src = source/);
  assert.match(
    panelSource,
    /const showImage = imageSource && !failedImageSources\.has\(imageSource\)/
  );
  assert.match(panelSource, /onError=\{\(\) => \{/);
  assert.match(
    panelSource,
    /const armName = profile\.id === 'long' \? text\.longArm : text\.shortArm/
  );
  assert.match(panelSource, /alt=\{`\$\{armName\} - \$\{text\.level\(strength\)\}`\}/);
  assert.doesNotMatch(panelSource, /\{profile\.displayName\}/);
  assert.match(
    panelSource,
    /<fieldset\s+disabled=\{formDisabled\}/
  );
});

test('normal API wiring uses only the finger robot endpoints', () => {
  assert.match(
    apiSource,
    /http\.request\(\{ method: 'put', url: '\/api\/vm\/finger-robot', data \}\)/
  );
  assert.match(
    apiSource,
    /http\.post\('\/api\/vm\/finger-robot\/press', challenge \? \{ challenge \} : \{\}\)/
  );
  assert.match(
    apiSource,
    /http\.post\('\/api\/vm\/finger-robot\/release', challenge \? \{ challenge \} : \{\}\)/
  );
  assert.match(
    apiSource,
    /http\.post\('\/api\/vm\/finger-robot\/replace-arm', challenge \? \{ challenge \} : \{\}\)/
  );
  assert.doesNotMatch(apiSource, /\/api\/vm\/power|\bsetPower\b/);

  const renderedMenuSource = `${menuSource}\n${panelSource}`;
  assert.match(menuSource, /import \{ FingerRobot \} from '\.\/finger-robot';/);
  assert.match(menuSource, /if \(isEnabled\('power'\)\)/);
  assert.match(menuSource, /<FingerRobot key="power" \/>/);
  assert.doesNotMatch(renderedMenuSource, /import \{ Power \}|<Power\s/);
  assert.doesNotMatch(renderedMenuSource, /\bPowerShort\b|\bPowerLong\b/);
});
