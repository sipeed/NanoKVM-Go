import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';

const webRoot = path.resolve(import.meta.dirname, '..');
const guideSource = fs.readFileSync(
  path.join(webRoot, 'src/pages/desktop/menu/mouse/touchpad-guide.tsx'),
  'utf8'
);
const localesDir = path.join(webRoot, 'src/i18n/locales');

function sourceBetween(source, start, end) {
  const startIndex = source.indexOf(start);
  const endIndex = source.indexOf(end, startIndex);
  assert.notEqual(startIndex, -1, `missing ${start}`);
  assert.notEqual(endIndex, -1, `missing ${end}`);
  return source.slice(startIndex, endIndex);
}

test('touchpad guide diagram labels the screen area through the shared i18n key', () => {
  const touchpadArea = sourceBetween(guideSource, 'const TouchpadArea', 'const SwipeDiagram');

  assert.ok(
    touchpadArea.includes("t('screen.title')"),
    "TouchpadArea must label the screen-area annotation via t('screen.title')"
  );
  assert.doesNotMatch(
    guideSource,
    />\s*screen\s*</,
    'touchpad-guide.tsx must not hardcode the screen label as a JSX text node'
  );
});

test('screen.title stays defined in every locale file', () => {
  const localeFiles = fs.readdirSync(localesDir).filter((name) => name.endsWith('.ts'));
  assert.ok(localeFiles.length > 0, 'no locale files found');

  for (const name of localeFiles) {
    const source = fs.readFileSync(path.join(localesDir, name), 'utf8');
    const screenBlock = source.match(/^(\s*)screen:\s*\{([\s\S]*?)\n\1\}/m);
    assert.ok(screenBlock, `${name} must define a screen section`);
    assert.match(
      screenBlock[2],
      /title:\s*'[^']+'/,
      `${name} must define screen.title (consumed by the touchpad guide diagram)`
    );
  }
});

test('no hardcoded JSX text nodes outside the language-neutral whitelist', () => {
  // Whitelist rationale: '1s' is a time unit and 'L' is a mouse-button symbol;
  // both are intentionally language-neutral technical literals per the frontend
  // internationalization spec. Any other bare JSX text node is user-visible copy
  // that pnpm i18n:audit cannot detect and must go through t() instead.
  const whitelist = new Set(['1s', 'L']);
  const jsxTextNodes = [...guideSource.matchAll(/>\s*([^\s<>{}(][^<>{}]*?)\s*</g)]
    .map((match) => match[1].trim())
    .filter(Boolean);

  for (const node of jsxTextNodes) {
    assert.ok(
      whitelist.has(node),
      `hardcoded JSX text node "${node}" must be translated via t() or whitelisted with a rationale`
    );
  }

  // The whitelist must stay attached to reality: if the diagrams change, revisit it.
  assert.match(guideSource, />\s*1s\s*</, 'hold-duration badge must stay the language-neutral 1s');
  assert.match(guideSource, />\s*L\s*</, 'left-button badge must stay the language-neutral L');
});

test('touchpad guide descriptions keep the spec secondary-text style', () => {
  const classTokensOf = (regex, label) => {
    const match = guideSource.match(regex);
    assert.ok(match, `missing ${label}`);
    return match[1].split(/\s+/);
  };
  const banner = classTokensOf(/className="([^"]*bg-neutral-950\/70[^"]*)"/, 'scope banner');
  const desc = classTokensOf(/className="([^"]+)"[^>]*>\{item\.desc\}/, 'card description');
  const title = classTokensOf(/className="([^"]+)"[^>]*>\{item\.title\}/, 'card title');

  for (const [label, tokens] of [
    ['scope banner', banner],
    ['card description', desc]
  ]) {
    assert.ok(tokens.includes('text-xs'), `${label} must use text-xs`);
    assert.ok(tokens.includes('text-neutral-500'), `${label} must use text-neutral-500`);
    assert.ok(!tokens.includes('text-sm'), `${label} must not reintroduce text-sm`);
    assert.ok(
      !tokens.some((token) => token.startsWith('leading-')),
      `${label} must not keep an explicit leading`
    );
  }

  assert.doesNotMatch(
    guideSource,
    /text-neutral-400/,
    'description roles must not reintroduce text-neutral-400'
  );
  // Deliberate boundary: card titles are primary content, not description text.
  assert.ok(title.includes('text-sm') && title.includes('font-medium'));
});
