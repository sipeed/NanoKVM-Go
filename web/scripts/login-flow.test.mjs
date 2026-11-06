import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import vm from 'node:vm';
import ts from 'typescript';

const webRoot = path.resolve(import.meta.dirname, '..');
const loginSource = fs.readFileSync(path.join(webRoot, 'src/pages/auth/login/index.tsx'), 'utf8');
const passwordSource = fs.readFileSync(
  path.join(webRoot, 'src/pages/auth/password/index.tsx'),
  'utf8'
);
const tipsSource = fs.readFileSync(path.join(webRoot, 'src/pages/auth/login/tips.tsx'), 'utf8');
const enLocaleSource = fs.readFileSync(path.join(webRoot, 'src/i18n/locales/en.ts'), 'utf8');
const authApiSource = fs.readFileSync(path.join(webRoot, 'src/api/auth.ts'), 'utf8');
const terminalSource = fs.readFileSync(path.join(webRoot, 'src/pages/terminal/index.tsx'), 'utf8');
const mockStateSource = fs.readFileSync(path.join(webRoot, 'src/mocks/state.ts'), 'utf8');
const mockAuthSource = fs.readFileSync(path.join(webRoot, 'src/mocks/handlers/auth.ts'), 'utf8');
const tailscaleLoginSource = fs.readFileSync(
  path.join(
    webRoot,
    'src/pages/desktop/menu/settings/remote-network/components/tailscale/login.tsx'
  ),
  'utf8'
);
const safeRedirectFile = path.join(webRoot, 'src/lib/safe-redirect.ts');

function loadSafeRedirectHelper() {
  const source = fs.readFileSync(safeRedirectFile, 'utf8');
  const output = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022
    },
    fileName: safeRedirectFile
  }).outputText;
  const context = { exports: {}, module: { exports: {} } };
  vm.runInNewContext(output, context, { filename: safeRedirectFile });
  return context.exports;
}

function typeBody(source, name) {
  const match = source.match(new RegExp(`type ${name} = \\{([\\s\\S]*?)\\n\\};`));
  assert.ok(match, `${name} type must exist`);
  return match[1];
}

function sourceBetween(source, start, end) {
  const startIndex = source.indexOf(start);
  const endIndex = source.indexOf(end, startIndex);
  assert.notEqual(startIndex, -1, `missing ${start}`);
  assert.notEqual(endIndex, -1, `missing ${end}`);
  return source.slice(startIndex, endIndex);
}

function sourceContaining(source, marker, start, end) {
  const markerIndex = source.indexOf(marker);
  assert.notEqual(markerIndex, -1, `missing ${marker}`);

  const startIndex = source.lastIndexOf(start, markerIndex);
  const endIndex = source.indexOf(end, markerIndex);
  assert.notEqual(startIndex, -1, `missing ${start} before ${marker}`);
  assert.notEqual(endIndex, -1, `missing ${end} after ${marker}`);
  return source.slice(startIndex, endIndex);
}

test('remember-password state remains local and outside the login API contract', () => {
  const loginValues = typeBody(loginSource, 'LoginValues');
  const pending2FA = typeBody(loginSource, 'Pending2FA');

  assert.doesNotMatch(loginValues, /rememberPassword/);
  assert.doesNotMatch(pending2FA, /password|rememberPassword/);
  assert.match(loginSource, /const \[rememberPassword, setRememberPassword\] = useState\(false\);/);
  assert.doesNotMatch(loginSource, /initialValuesRef\.current\.password/);
  assert.match(loginSource, /api\s*\.login\(username, password\)/);
  assert.match(authApiSource, /const data = \{\s+username,\s+password\s+\};/);
  assert.doesNotMatch(authApiSource, /rememberPassword/);
});

test('only accepted password-factor results can persist a remembered password', () => {
  const saveFunction = sourceBetween(
    loginSource,
    'function saveRememberedPassword',
    'function changeRememberPassword'
  );
  const passwordLogin = sourceBetween(
    loginSource,
    'function login(values: LoginValues)',
    'function finishLogin'
  );
  const challengeBranch = sourceBetween(
    passwordLogin,
    'if (data.requires2FA && data.challenge)',
    "setMsg(t('auth.error'));"
  );

  assert.match(saveFunction, /if \(!rememberPassword\) return;/);
  assert.match(saveFunction, /storage\.setLoginPassword\(password\)/);
  assert.match(saveFunction, /storage\.setLoginUsername\(username\)/);
  assert.match(saveFunction, /storage\.removeLoginPassword\(\);/);
  assert.equal(
    passwordLogin.match(/saveRememberedPassword\(username, password\);/g)?.length,
    2,
    'normal success and a valid 2FA challenge are the only save sites'
  );
  assert.match(challengeBranch, /form\.setFieldValue\('password', ''\);/);
  assert.match(challengeBranch, /setPending2FA\(\{ challenge: data\.challenge, username \}\);/);
});

test('direct 2FA and mocked fixtures do not receive remembered passwords', () => {
  const directTwoFactorLogin = sourceBetween(
    loginSource,
    'function loginWithDirectTwoFactor',
    'function submit'
  );

  assert.doesNotMatch(directTwoFactorLogin, /LoginPassword/);
  assert.doesNotMatch(mockStateSource, /nano-kvm-login-password/);
  assert.doesNotMatch(mockAuthSource, /nano-kvm-login-password|rememberPassword/);
});

test('a successful web-password change clears the remembered password before logout navigation', () => {
  const removalIndex = passwordSource.indexOf('storage.removeLoginPassword();');
  const tokenRemovalIndex = passwordSource.indexOf('removeToken();');
  const navigateIndex = passwordSource.indexOf("navigate('/auth/login', { replace: true });");

  assert.notEqual(removalIndex, -1);
  assert.ok(removalIndex < tokenRemovalIndex);
  assert.ok(tokenRemovalIndex < navigateIndex);
});

test('auth reset wiki links label themselves through the shared i18n key', () => {
  const consumers = [
    ['login tips modal', tipsSource],
    ['change-password page', passwordSource]
  ];

  for (const [name, source] of consumers) {
    assert.ok(
      source.includes("t('auth.tips.resetLink')"),
      `${name} must label the reset wiki link via t('auth.tips.resetLink')`
    );
    assert.doesNotMatch(
      source,
      />\s*Reset\s*</,
      `${name} must not hardcode the Reset link label as a JSX text node`
    );
  }

  assert.match(enLocaleSource, /resetLink: 'Reset'/);
});

test('login redirects accept only normalized internal application routes', () => {
  const { safeLoginRedirect } = loadSafeRedirectHelper();

  for (const route of ['/', '/desktop', '/desktop?tab=ai#memory', '/auth/password']) {
    assert.equal(safeLoginRedirect(route), route);
  }

  for (const route of [
    null,
    '',
    'desktop',
    'https://example.com/',
    '//example.com/',
    '/\\example.com',
    '/safe\\..\\outside',
    '/javascript:alert(1)',
    '/data:text/html,test',
    '/safe/../outside',
    '/safe/%2e%2e/outside',
    '/%2f%2fevil.example',
    '/%252f%252fevil.example',
    '/%5cevil.example',
    '/safe%0apath',
    '/safe%250apath',
    '/safe\u0085path',
    '/broken%'
  ]) {
    assert.equal(safeLoginRedirect(route), '/', `expected ${String(route)} to be rejected`);
  }
});

test('both login navigation paths use the shared safe redirect boundary', () => {
  assert.match(loginSource, /import \{ safeLoginRedirect \} from '@\/lib\/safe-redirect\.ts';/);
  assert.equal(
    loginSource.match(/navigate\(safeLoginRedirect\(searchParams\.get\('redirect'\)\)/g)?.length,
    2
  );
  assert.doesNotMatch(loginSource, /navigate\(redirect \|\| '\/'/);
});

test('tailscale login links isolate the new browsing context', () => {
  assert.match(tailscaleLoginSource, /window\.open\(url, '_blank', 'noopener,noreferrer'\);/);
  assert.match(
    tailscaleLoginSource,
    /href=\{loginUrl\}[\s\S]*?target="_blank"[\s\S]*?rel="noopener noreferrer"/
  );
});

test('web terminal authorization remains open until authorization succeeds', () => {
  const authorizationModal = sourceBetween(terminalSource, '<Modal', 'footer={');

  assert.match(authorizationModal, /closable=\{false\}/);
  assert.match(authorizationModal, /maskClosable=\{false\}/);
  assert.match(authorizationModal, /keyboard=\{false\}/);
  assert.doesNotMatch(authorizationModal, /onCancel=/);
  assert.doesNotMatch(terminalSource, /handleCancel/);
});

test('web terminal focuses the password input after authorization options are ready', () => {
  const passwordFocusEffect = sourceContaining(
    terminalSource,
    'const input = passwordInputRef.current;',
    '  useEffect(() => {',
    ']);'
  );
  const passwordInput = sourceBetween(terminalSource, '<Input.Password', '</Form.Item>');
  const passwordFocusGuard = sourceBetween(passwordFocusEffect, '    if (', '    const focusFrame');

  assert.match(terminalSource, /const passwordInputRef = useRef<InputRef>\(null\);/);
  for (const readinessCondition of [
    '!modal',
    'authed',
    'isResolvingOptions',
    'isSubmitting',
    'authRateLimit.isLocked',
    'authRequirements === null'
  ]) {
    assert.ok(passwordFocusGuard.includes(readinessCondition), `missing ${readinessCondition}`);
  }
  assert.match(passwordFocusEffect, /const focusFrame = requestAnimationFrame\(\(\) => \{/);
  assert.match(passwordFocusEffect, /if \(!input\?\.input \|\| input\.input\.disabled\) return;/);
  assert.match(passwordFocusEffect, /input\.focus\(\);/);
  assert.match(passwordFocusEffect, /return \(\) => cancelAnimationFrame\(focusFrame\);/);
  assert.match(passwordInput, /ref=\{passwordInputRef\}/);
  assert.doesNotMatch(passwordInput, /\bautoFocus\b/);
});

test('web terminal waits for both socket attachment and modal close before focusing xterm', () => {
  const focusHelper = sourceBetween(
    terminalSource,
    'const focusTerminalWhenReady',
    'const refreshAuthOptions'
  );
  const socketOpen = sourceBetween(terminalSource, 'ws.onopen = () => {', 'ws.onclose =');
  const socketClose = sourceBetween(terminalSource, 'ws.onclose =', 'const sendSize');
  const terminalSetup = sourceBetween(
    terminalSource,
    'const terminal = new XtermTerminal',
    'const fitAddon = new FitAddon()'
  );
  const terminalCleanup = sourceContaining(
    terminalSource,
    'disposed = true;',
    'return () => {',
    'terminal.dispose();'
  );
  const authSuccess = sourceBetween(
    terminalSource,
    'function applyAuthSuccess',
    'function applyAuthFailure'
  );
  const modalHandler = sourceBetween(
    terminalSource,
    'function handleModalOpenChange',
    '\n\n  return ('
  );
  const authorizationModal = sourceBetween(terminalSource, '<Modal', 'footer={');

  assert.equal(focusHelper.match(/!terminalAttachedRef\.current/g)?.length, 2);
  assert.equal(focusHelper.match(/!terminalModalClosedRef\.current/g)?.length, 2);
  assert.match(focusHelper, /terminalFocusFrameRef\.current !== null/);
  assert.match(focusHelper, /terminalRef\.current !== terminal/);
  assert.match(focusHelper, /terminalHasAutoFocusedRef\.current = true;/);
  assert.match(focusHelper, /terminal\.focus\(\);/);

  const attachIndex = socketOpen.indexOf('terminal.loadAddon(attachAddon);');
  const readyIndex = socketOpen.indexOf('terminalAttachedRef.current = true;');
  const focusIndex = socketOpen.indexOf('focusTerminalWhenReady();');
  assert.ok(attachIndex !== -1 && attachIndex < readyIndex && readyIndex < focusIndex);
  assert.match(socketOpen, /disposed \|\| terminalRef\.current !== terminal/);

  assert.match(socketClose, /terminalModalClosedRef\.current = false;/);
  assert.match(socketClose, /cancelPendingTerminalFocus\(\);/);
  assert.match(socketClose, /authedRef\.current = false;/);
  assert.match(socketClose, /modalRef\.current = true;/);
  assert.doesNotMatch(terminalSetup, /terminalModalClosedRef\.current/);
  assert.match(terminalCleanup, /cancelPendingTerminalFocus\(\);/);
  assert.match(terminalCleanup, /terminalRef\.current = null;/);
  assert.match(terminalCleanup, /terminalAttachedRef\.current = false;/);
  assert.match(terminalCleanup, /terminalHasAutoFocusedRef\.current = false;/);
  assert.doesNotMatch(terminalCleanup, /terminalModalClosedRef\.current/);

  assert.ok(
    authSuccess.indexOf('terminalModalClosedRef.current = false;') <
      authSuccess.indexOf('setAuthState(true);')
  );
  assert.match(authSuccess, /authedRef\.current = true;/);
  assert.match(authSuccess, /modalRef\.current = false;/);

  assert.match(modalHandler, /if \(open \|\| modalRef\.current \|\| !authedRef\.current\)/);
  assert.match(modalHandler, /terminalModalClosedRef\.current = false;/);
  assert.match(modalHandler, /cancelPendingTerminalFocus\(\);/);
  assert.ok(
    modalHandler.indexOf('terminalModalClosedRef.current = true;') <
      modalHandler.indexOf('focusTerminalWhenReady();')
  );
  assert.match(authorizationModal, /afterOpenChange=\{handleModalOpenChange\}/);
});
