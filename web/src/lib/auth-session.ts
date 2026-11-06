import { removeToken } from '@/lib/cookie.ts';

export const SESSION_EXPIRED_CLOSE_CODE = 4001;
export const TWO_FACTOR_ENABLED_CLOSE_CODE = 4002;
export const SYSTEM_TIME_UPDATED_CLOSE_CODE = 4003;

const AUTH_SESSION_NOTICE_KEY = 'nano-kvm-auth-session-notice';

export type AuthSessionNotice = 'twoFactorEnabled' | 'systemTimeUpdated';

let authSessionExpiring = false;

function saveAuthSessionNotice(notice?: AuthSessionNotice) {
  if (!notice) return;

  window.sessionStorage.setItem(AUTH_SESSION_NOTICE_KEY, notice);
}

export function consumeAuthSessionNotice(): AuthSessionNotice | null {
  const notice = window.sessionStorage.getItem(AUTH_SESSION_NOTICE_KEY);
  window.sessionStorage.removeItem(AUTH_SESSION_NOTICE_KEY);

  return notice === 'twoFactorEnabled' || notice === 'systemTimeUpdated' ? notice : null;
}

export function expireAuthSession(notice?: AuthSessionNotice) {
  if (authSessionExpiring) return;

  authSessionExpiring = true;
  saveAuthSessionNotice(notice);
  removeToken();
  window.location.reload();
}

export function handleSessionExpiredClose(event: { code?: number }) {
  if (event.code !== SESSION_EXPIRED_CLOSE_CODE) return false;

  expireAuthSession();
  return true;
}

export function handleTwoFactorEnabledClose(event: { code?: number }) {
  if (event.code !== TWO_FACTOR_ENABLED_CLOSE_CODE) return false;

  expireAuthSession('twoFactorEnabled');
  return true;
}

export function handleSystemTimeUpdatedClose(event: { code?: number }) {
  if (event.code !== SYSTEM_TIME_UPDATED_CLOSE_CODE) return false;

  expireAuthSession('systemTimeUpdated');
  return true;
}

export function handleAuthSessionClose(event: { code?: number }) {
  return (
    handleTwoFactorEnabledClose(event) ||
    handleSystemTimeUpdatedClose(event) ||
    handleSessionExpiredClose(event)
  );
}
