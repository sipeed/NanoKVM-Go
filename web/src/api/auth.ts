import { http } from '@/lib/http';

export type LoginResponseData = {
  token?: string;
  requires2FA?: boolean;
  challenge?: string;
  recoveryCodeUsed?: boolean;
};

export type DirectTwoFactorLoginStatus = {
  enabled: boolean;
};

export type TwoFactorStatus = {
  enabled: boolean;
  recoveryEnabled: boolean;
  recoveryRemaining: number;
};

export type TwoFactorProtectionPolicy = {
  login: boolean;
  webTerminal: boolean;
  ssh: boolean;
  tailscale: boolean;
  frp: boolean;
  cloudflare: boolean;
  zerotier: boolean;
  netbird: boolean;
  mcp: boolean;
  targetPower: boolean;
  nanoKVMRestart: boolean;
  scripts: boolean;
  screenTimelapse: boolean;
  deleteImage: boolean;
};

export type TwoFactorProtectionStatus = {
  policy: TwoFactorProtectionPolicy;
  authorizationExpiresIn: number;
  mcpAuthorizationExpiresIn: number;
  screenTimelapseAuthorizationExpiresIn: number;
};

export type TwoFactorProtectionAuthorization = {
  expiresIn: number;
};

export type TwoFactorSetupData = {
  uri: string;
  secret: string;
};

export type RecoveryCodesData = {
  recoveryCodes: string[];
};

export type SensitiveActionData = {
  challenge: string;
  expiresIn: number;
};

export type AuthRateLimitData = {
  rateLimited?: boolean;
  retryAfter?: number;
  retryAt?: number;
};

export function getAuthRateLimitData(rsp: { data?: unknown }): AuthRateLimitData | null {
  const data = rsp.data as AuthRateLimitData | undefined;
  if (!data?.rateLimited) return null;
  if (typeof data.retryAfter !== 'number' || data.retryAfter <= 0) return null;
  return data;
}

export function login(username: string, password: string) {
  const data = {
    username,
    password
  };
  return http.post('/api/auth/login', data);
}

export function verifyLogin2FA(challenge: string, code: string) {
  return http.post('/api/auth/login', { challenge, code });
}

export function getDirectTwoFactorLoginStatus() {
  return http.get('/api/auth/login/2fa-direct');
}

export function loginWithDirectTwoFactor(code: string) {
  return http.post('/api/auth/login', { code });
}

export function logout() {
  return http.post('/api/auth/logout');
}

export function getAccount() {
  return http.get('/api/auth/account');
}

export function changePassword(username: string, password: string, challenge?: string) {
  const data = challenge ? { username, password, challenge } : { username, password };
  return http.post('/api/auth/password', data);
}

export function isPasswordUpdated() {
  return http.get('/api/auth/password');
}

export function getTwoFactorStatus() {
  return http.get('/api/auth/2fa/status');
}

export function setupTwoFactor(password: string) {
  return http.post('/api/auth/2fa/setup', { password });
}

export function cancelTwoFactorSetup() {
  return http.post('/api/auth/2fa/setup/cancel');
}

export function enableTwoFactor(code: string) {
  return http.post('/api/auth/2fa/enable', { code });
}

export function disableTwoFactor(password: string, code: string) {
  return http.post('/api/auth/2fa/disable', { password, code });
}

export function enableRecoveryCodes(password: string, code: string) {
  return http.post('/api/auth/2fa/recovery/enable', { password, code });
}

export function regenerateRecoveryCodes(password: string, code: string) {
  return http.post('/api/auth/2fa/recovery/regenerate', { password, code });
}

export function getTwoFactorProtection() {
  return http.get('/api/auth/2fa/protection');
}

export function authorizeTwoFactorProtection(code: string) {
  return http.post('/api/auth/2fa/protection/authorize', { code });
}

export function setTwoFactorProtection(policy: TwoFactorProtectionPolicy) {
  return http.post('/api/auth/2fa/protection', policy);
}

export type SensitiveActionName =
  | 'netbird_install'
  | 'netbird_uninstall'
  | 'netbird_start'
  | 'netbird_stop'
  | 'netbird_restart'
  | 'netbird_login'
  | 'netbird_login_cancel'
  | 'netbird_logout'
  | 'zerotier_install'
  | 'zerotier_uninstall'
  | 'zerotier_start'
  | 'zerotier_stop'
  | 'zerotier_restart'
  | 'zerotier_join'
  | 'zerotier_leave'
  | 'cloudflare_install'
  | 'cloudflare_uninstall'
  | 'cloudflare_configure'
  | 'cloudflare_start'
  | 'cloudflare_stop'
  | 'cloudflare_restart'
  | 'frp_install'
  | 'frp_uninstall'
  | 'frp_configure'
  | 'frp_start'
  | 'frp_stop'
  | 'frp_restart'
  | 'change_password'
  | 'run_script'
  | 'upload_script'
  | 'delete_script'
  | 'enable_ssh'
  | 'disable_ssh'
  | 'tailscale_install'
  | 'tailscale_login'
  | 'tailscale_up'
  | 'tailscale_down'
  | 'tailscale_start'
  | 'tailscale_stop'
  | 'tailscale_restart'
  | 'tailscale_logout'
  | 'tailscale_uninstall'
  | 'tailscale_set_exit_node'
  | 'tailscale_set_subnet_route'
  | 'mcp_show_key'
  | 'mcp_copy_key'
  | 'mcp_regenerate_key'
  | 'mcp_enable'
  | 'mcp_disable'
  | 'target_power'
  | 'nanokvm_restart'
  | 'delete_image'
  | 'open_web_terminal'
  | 'screen_timelapse_enable'
  | 'screen_timelapse_playback'
  | 'set_screen_timelapse_retention'
  | 'trigger_screen_timelapse_cleanup'
  | 'update_signature_verification';

export function verifySensitiveAction(
  action: SensitiveActionName,
  password: string | undefined,
  code: string
) {
  const data: { action: string; code: string; password?: string } = { action, code };
  if (password !== undefined && password !== '') {
    data.password = password;
  }
  return http.post('/api/auth/2fa/verify-sensitive-action', data);
}
