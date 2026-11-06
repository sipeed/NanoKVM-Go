import { http } from 'msw';

import { fail, ok, readBoolean, readJsonObject, readString } from '../response.ts';
import { createMockTwoFactorProtection, type MockState } from '../state.ts';

const mcpSensitiveActions = new Set([
  'mcp_enable',
  'mcp_disable',
  'mcp_show_key',
  'mcp_copy_key',
  'mcp_regenerate_key'
]);

function protectionResponse(state: MockState) {
  return {
    policy: state.auth.twoFactorProtection,
    authorizationExpiresIn: Math.max(
      0,
      Math.ceil((state.auth.twoFactorProtectionAuthorizedUntil - Date.now()) / 1000)
    ),
    mcpAuthorizationExpiresIn: Math.max(
      0,
      Math.ceil((state.auth.mcpAuthorizationUntil - Date.now()) / 1000)
    ),
    screenTimelapseAuthorizationExpiresIn: Math.max(
      0,
      Math.ceil((state.auth.screenTimelapseAuthorizationUntil - Date.now()) / 1000)
    )
  };
}

export function createAuthHandlers(origin: string, state: MockState) {
  return [
    http.post(`${origin}/api/auth/login`, async ({ request }) => {
      const body = await readJsonObject(request);
      const code = readString(body, 'code');
      const challenge = readString(body, 'challenge');
      const username = readString(body, 'username');
      const password = readString(body, 'password');
      const directLogin =
        code !== null && challenge === null && username === null && password === null;
      if (directLogin) {
        if (!state.auth.twoFactorEnabled || !state.auth.twoFactorProtection.login) {
          return fail(-2, 'invalid authentication credentials', {});
        }
        if (!code.trim()) {
          return fail(-1, 'invalid parameters', {});
        }
      }
      return ok({});
    }),
    http.get(`${origin}/api/auth/login/2fa-direct`, () =>
      ok({ enabled: state.auth.twoFactorEnabled && state.auth.twoFactorProtection.login })
    ),
    http.post(`${origin}/api/auth/logout`, () => ok({})),
    http.get(`${origin}/api/auth/account`, () => ok({ username: state.auth.username })),
    http.post(`${origin}/api/auth/password`, () => {
      state.auth.passwordUpdated = true;
      return ok({});
    }),
    http.get(`${origin}/api/auth/password`, () => ok({ isUpdated: state.auth.passwordUpdated })),
    http.get(`${origin}/api/auth/2fa/status`, () =>
      ok({
        enabled: state.auth.twoFactorEnabled,
        recoveryEnabled: state.auth.recoveryEnabled,
        recoveryRemaining: state.auth.recoveryRemaining
      })
    ),
    http.post(`${origin}/api/auth/2fa/setup`, () =>
      ok({
        uri: 'otpauth://totp/NanoKVM%20Mock?secret=MOCKDEVELOPMENTKEY&issuer=NanoKVM',
        secret: 'MOCKDEVELOPMENTKEY'
      })
    ),
    http.post(`${origin}/api/auth/2fa/setup/cancel`, () => ok({})),
    http.post(`${origin}/api/auth/2fa/enable`, () => {
      state.auth.twoFactorEnabled = true;
      return ok({});
    }),
    http.post(`${origin}/api/auth/2fa/disable`, () => {
      state.auth.twoFactorEnabled = false;
      state.auth.recoveryEnabled = false;
      state.auth.recoveryRemaining = 0;
      state.auth.twoFactorProtection = createMockTwoFactorProtection();
      state.auth.twoFactorProtectionAuthorizedUntil = 0;
      state.auth.mcpAuthorizationUntil = 0;
      state.auth.screenTimelapseAuthorizationUntil = 0;
      state.auth.targetPowerChallenge = null;
      state.auth.nanoKVMRestartChallenge = null;
      state.auth.deleteImageChallenge = null;
      state.auth.screenTimelapseChallenge = null;
      state.auth.screenTimelapsePlaybackChallenge = null;
      state.auth.webTerminalChallenge = null;
      state.auth.tailscaleExitNodeChallenge = null;
      state.auth.tailscaleSubnetRouteChallenge = null;
      state.auth.tailscaleLoginChallenge = null;
      state.auth.frpChallenge = null;
      state.auth.cloudflareChallenge = null;
      state.auth.zerotierChallenge = null;
      state.auth.netbirdChallenge = null;
      state.auth.updateSignatureVerificationChallenge = null;
      return ok({});
    }),
    http.post(`${origin}/api/auth/2fa/recovery/enable`, () => {
      state.auth.recoveryEnabled = true;
      state.auth.recoveryRemaining = state.auth.recoveryCodes.length;
      return ok({ recoveryCodes: state.auth.recoveryCodes });
    }),
    http.post(`${origin}/api/auth/2fa/recovery/regenerate`, () => {
      state.auth.recoveryCodes = ['MOCK-DELTA-0004', 'MOCK-ECHO-0005', 'MOCK-FOXTROT-0006'];
      state.auth.recoveryEnabled = true;
      state.auth.recoveryRemaining = state.auth.recoveryCodes.length;
      return ok({ recoveryCodes: state.auth.recoveryCodes });
    }),
    http.post(`${origin}/api/auth/2fa/verify-sensitive-action`, async ({ request }) => {
      const body = await readJsonObject(request);
      const action = readString(body, 'action');
      if (
        action &&
        [
          'frp_install',
          'frp_uninstall',
          'frp_configure',
          'frp_start',
          'frp_stop',
          'frp_restart'
        ].includes(action)
      ) {
        if (state.auth.twoFactorEnabled && !readString(body, 'code')?.trim()) {
          return fail(-2, 'invalid authentication credentials', {});
        }
        state.auth.frpChallenge = { action, token: `mock-${action}-challenge` };
        return ok({ challenge: state.auth.frpChallenge.token, expiresIn: 300 });
      }
      if (
        action &&
        [
          'cloudflare_install',
          'cloudflare_uninstall',
          'cloudflare_configure',
          'cloudflare_start',
          'cloudflare_stop',
          'cloudflare_restart'
        ].includes(action)
      ) {
        if (state.auth.twoFactorEnabled && !readString(body, 'code')?.trim()) {
          return fail(-2, 'invalid authentication credentials', {});
        }
        state.auth.cloudflareChallenge = { action, token: `mock-${action}-challenge` };
        return ok({ challenge: state.auth.cloudflareChallenge.token, expiresIn: 300 });
      }
      if (
        action &&
        [
          'zerotier_install',
          'zerotier_uninstall',
          'zerotier_join',
          'zerotier_leave',
          'zerotier_start',
          'zerotier_stop',
          'zerotier_restart'
        ].includes(action)
      ) {
        if (state.auth.twoFactorEnabled && !readString(body, 'code')?.trim()) {
          return fail(-2, 'invalid authentication credentials', {});
        }
        state.auth.zerotierChallenge = { action, token: `mock-${action}-challenge` };
        return ok({ challenge: state.auth.zerotierChallenge.token, expiresIn: 300 });
      }
      if (
        action &&
        [
          'netbird_install',
          'netbird_uninstall',
          'netbird_login',
          'netbird_login_cancel',
          'netbird_logout',
          'netbird_start',
          'netbird_stop',
          'netbird_restart'
        ].includes(action)
      ) {
        if (state.auth.twoFactorEnabled && !readString(body, 'code')?.trim()) {
          return fail(-2, 'invalid authentication credentials', {});
        }
        state.auth.netbirdChallenge = { action, token: `mock-${action}-challenge` };
        return ok({ challenge: state.auth.netbirdChallenge.token, expiresIn: 300 });
      }
      if (action === 'open_web_terminal') {
        const password = readString(body, 'password');
        const code = readString(body, 'code');
        const secondFactorRequired =
          state.auth.twoFactorEnabled && state.auth.twoFactorProtection.webTerminal;
        if (!password?.trim() || (secondFactorRequired && !code?.trim())) {
          return fail(-2, 'invalid authentication credentials', {});
        }
        state.auth.webTerminalChallenge = 'mock-open-web-terminal-challenge';
        return ok({ challenge: state.auth.webTerminalChallenge, expiresIn: 300 });
      }
      if (
        action !== null &&
        mcpSensitiveActions.has(action) &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.mcp
      ) {
        state.auth.mcpAuthorizationUntil = Date.now() + 5 * 60 * 1000;
      }
      if (
        action === 'target_power' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.targetPower
      ) {
        state.auth.targetPowerChallenge = 'mock-target-power-challenge';
        return ok({ challenge: state.auth.targetPowerChallenge, expiresIn: 300 });
      }
      if (
        action === 'nanokvm_restart' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.nanoKVMRestart
      ) {
        state.auth.nanoKVMRestartChallenge = 'mock-nanokvm-restart-challenge';
        return ok({ challenge: state.auth.nanoKVMRestartChallenge, expiresIn: 300 });
      }
      if (
        action === 'delete_image' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.deleteImage
      ) {
        state.auth.deleteImageChallenge = 'mock-delete-image-challenge';
        return ok({ challenge: state.auth.deleteImageChallenge, expiresIn: 300 });
      }
      if (
        action === 'screen_timelapse_enable' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.screenTimelapse
      ) {
        state.auth.screenTimelapseChallenge = 'mock-screen-timelapse-enable-challenge';
        return ok({ challenge: state.auth.screenTimelapseChallenge, expiresIn: 300 });
      }
      if (
        action === 'screen_timelapse_playback' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.screenTimelapse
      ) {
        state.auth.screenTimelapsePlaybackChallenge = 'mock-screen-timelapse-playback-challenge';
        return ok({ challenge: state.auth.screenTimelapsePlaybackChallenge, expiresIn: 300 });
      }
      if (
        action === 'tailscale_login' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.tailscale
      ) {
        state.auth.tailscaleLoginChallenge = 'mock-tailscale-login-challenge';
        return ok({ challenge: state.auth.tailscaleLoginChallenge, expiresIn: 300 });
      }
      if (
        action === 'tailscale_set_exit_node' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.tailscale
      ) {
        state.auth.tailscaleExitNodeChallenge = 'mock-tailscale-exit-node-challenge';
        return ok({ challenge: state.auth.tailscaleExitNodeChallenge, expiresIn: 300 });
      }
      if (
        action === 'tailscale_set_subnet_route' &&
        state.auth.twoFactorEnabled &&
        state.auth.twoFactorProtection.tailscale
      ) {
        state.auth.tailscaleSubnetRouteChallenge = 'mock-tailscale-subnet-route-challenge';
        return ok({ challenge: state.auth.tailscaleSubnetRouteChallenge, expiresIn: 300 });
      }
      if (action === 'update_signature_verification') {
        state.auth.updateSignatureVerificationChallenge =
          'mock-update-signature-verification-challenge';
        return ok({
          challenge: state.auth.updateSignatureVerificationChallenge,
          expiresIn: 300
        });
      }
      return ok({ challenge: 'mock-sensitive-action', expiresIn: 300 });
    }),
    http.get(`${origin}/api/auth/2fa/protection`, () => {
      if (!state.auth.twoFactorEnabled) {
        return fail(-2, 'two factor authentication is disabled', protectionResponse(state));
      }
      return ok(protectionResponse(state));
    }),
    http.post(`${origin}/api/auth/2fa/protection/authorize`, async ({ request }) => {
      await readJsonObject(request);
      if (!state.auth.twoFactorEnabled) {
        return fail(-2, 'invalid authentication credentials', { expiresIn: 0 });
      }
      state.auth.twoFactorProtectionAuthorizedUntil = Date.now() + 5 * 60 * 1000;
      return ok({ expiresIn: 300 });
    }),
    http.post(`${origin}/api/auth/2fa/protection`, async ({ request }) => {
      const body = await readJsonObject(request);
      const login = readBoolean(body, 'login');
      const webTerminal = readBoolean(body, 'webTerminal');
      const ssh = readBoolean(body, 'ssh');
      const tailscale = readBoolean(body, 'tailscale');
      const mcp = readBoolean(body, 'mcp');
      const targetPower = readBoolean(body, 'targetPower');
      const nanoKVMRestart = readBoolean(body, 'nanoKVMRestart');
      const scripts = readBoolean(body, 'scripts');
      const screenTimelapse = readBoolean(body, 'screenTimelapse');
      const deleteImage = readBoolean(body, 'deleteImage');
      if (
        login === null ||
        webTerminal === null ||
        ssh === null ||
        tailscale === null ||
        mcp === null ||
        targetPower === null ||
        nanoKVMRestart === null ||
        scripts === null ||
        screenTimelapse === null ||
        deleteImage === null
      ) {
        return fail(-1, 'invalid parameters', protectionResponse(state));
      }
      if (!state.auth.twoFactorEnabled) {
        return fail(-2, 'two factor authentication is disabled', protectionResponse(state));
      }
      if (state.auth.twoFactorProtectionAuthorizedUntil <= Date.now()) {
        return fail(-4, 'two factor protection authorization required', protectionResponse(state));
      }

      state.auth.twoFactorProtection = {
        frp: readBoolean(body, 'frp') ?? state.auth.twoFactorProtection.frp,
        cloudflare: readBoolean(body, 'cloudflare') ?? state.auth.twoFactorProtection.cloudflare,
        zerotier: readBoolean(body, 'zerotier') ?? state.auth.twoFactorProtection.zerotier,
        netbird: readBoolean(body, 'netbird') ?? state.auth.twoFactorProtection.netbird,
        login,
        webTerminal,
        ssh,
        tailscale,
        mcp,
        targetPower,
        nanoKVMRestart,
        scripts,
        screenTimelapse,
        deleteImage
      };
      return ok(protectionResponse(state));
    }),
    http.post(`${origin}/api/vm/terminal/auth`, async ({ request }) => {
      const body = await readJsonObject(request);
      const challenge = readString(body, 'challenge');
      if (!challenge || challenge !== state.auth.webTerminalChallenge) {
        return fail(-1, 'authentication failed');
      }
      state.auth.webTerminalChallenge = null;
      return ok(null);
    })
  ];
}
