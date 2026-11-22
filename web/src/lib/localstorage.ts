import type { InputProfileStore } from './input-profile.ts';

const LANGUAGE_KEY = 'nano-kvm-language';
const VIDEO_MODE_KEY = 'nano-kvm-vide-mode';
const VIDEO_PARAMETERS_KEY = 'nano-kvm-video-parameters';
export const MOUSE_MODE_KEY = 'nano-kvm-mouse-mode';
const MOUSE_SCROLL_DIRECTION_KEY = 'nanokvm-kvm-mouse-scroll-direction';
const MOUSE_SCROLL_INTERVAL_KEY = 'nanokvm-kvm-mouse-scroll-interval';
const SKIP_UPDATE_KEY = 'nano-kvm-check-update';
const KEYBOARD_SYSTEM_KEY = 'nano-kvm-keyboard-system';
const KEYBOARD_LANGUAGE_KEY = 'nano-kvm-keyboard-language';
const KEYBOARD_SWAP_COMMAND_CTRL_KEY = 'nano-kvm-keyboard-swap-command-ctrl';
const SKIP_MODIFY_PASSWORD_KEY = 'nano-kvm-skip-modify-password';
const SKIP_BROWSER_NOTIFICATION_KEY = 'nano-kvm-skip-browser-notification';
const MENU_DISABLED_ITEMS_KEY = 'nano-kvm-menu-disabled-items';
const MENU_AUTO_HIDE_KEY = 'nano-kvm-menu-auto-hide';
const POWER_CONFIRM_KEY = 'nano-kvm-power-confirm';
const VIDEO_VOLUME_KEY = 'nano-kvm-video-volume';
const PORTAL_MODE_KEY = 'nano-kvm-portal-mode';
const WIFI_PORTAL_VIEW_MODE_KEY = 'nanokvm-wifi-view-mode';
const PORTAL_BORDER_ENABLED_KEY = 'nano-kvm-portal-border-enabled';
const PORTAL_BORDER_COLOR_KEY = 'nano-kvm-portal-border-color';
export const INPUT_PROFILES_KEY = 'nano-kvm-input-profiles';
const LOGIN_USERNAME_KEY = 'nano-kvm-login-username';
const LOGIN_PASSWORD_KEY = 'nano-kvm-login-password';
const REMEMBERED_LOGIN_PASSWORD_TTL_MS = 30 * 24 * 60 * 60 * 1000;

const DEFAULT_MENU_DISABLED_ITEMS = ['power', 'mouseJiggler'];

type ItemWithExpiry = {
  value: string;
  expiry: number;
};

type RememberedLoginPassword = {
  value: string;
  expiry: number;
};

type PortalMode = 'auto' | 'manual' | 'off';

export type WifiPortalViewMode = 'simple' | 'detailed';

// set the value with expiration time (unit: milliseconds)
function setWithExpiry(key: string, value: string, ttl: number) {
  const now = new Date();

  const item: ItemWithExpiry = {
    value: value,
    expiry: now.getTime() + ttl
  };

  localStorage.setItem(key, JSON.stringify(item));
}

// get the value with expiration time
function getWithExpiry(key: string) {
  const itemStr = localStorage.getItem(key);
  if (!itemStr) return null;

  const item: ItemWithExpiry = JSON.parse(itemStr);
  const now = new Date();
  if (now.getTime() > item.expiry) {
    localStorage.removeItem(key);
    return null;
  }

  return item.value;
}

export function getLanguage() {
  return localStorage.getItem(LANGUAGE_KEY);
}

export function setLanguage(language: string) {
  localStorage.setItem(LANGUAGE_KEY, language);
}

export function getVideoMode() {
  return localStorage.getItem(VIDEO_MODE_KEY);
}

export function setVideoMode(mode: string) {
  localStorage.setItem(VIDEO_MODE_KEY, mode);
}

export function getVideoParameters() {
  try {
    return localStorage.getItem(VIDEO_PARAMETERS_KEY);
  } catch (err) {
    console.log(err);
    return null;
  }
}

export function setVideoParameters(parameters: string) {
  try {
    localStorage.setItem(VIDEO_PARAMETERS_KEY, parameters);
  } catch (err) {
    console.log(err);
  }
}

export function getMouseMode() {
  return localStorage.getItem(MOUSE_MODE_KEY);
}

export function setMouseMode(mouse: string) {
  localStorage.setItem(MOUSE_MODE_KEY, mouse);
}

export function getInputProfileStore(): unknown {
  try {
    const value = localStorage.getItem(INPUT_PROFILES_KEY);
    return value ? JSON.parse(value) : null;
  } catch {
    return null;
  }
}

export function setInputProfileStore(store: InputProfileStore): void {
  try {
    localStorage.setItem(INPUT_PROFILES_KEY, JSON.stringify(store));
  } catch (err) {
    console.log(err);
  }
}

export function getMouseScrollDirection(): number | null {
  const direction = localStorage.getItem(MOUSE_SCROLL_DIRECTION_KEY);
  if (direction && Number(direction)) {
    return Number(direction);
  }
  return null;
}

export function setMouseScrollDirection(direction: number): void {
  localStorage.setItem(MOUSE_SCROLL_DIRECTION_KEY, String(direction));
}

export function getMouseScrollInterval() {
  const interval = localStorage.getItem(MOUSE_SCROLL_INTERVAL_KEY);
  return interval ? Number(interval) : null;
}

export function setMouseScrollInterval(interval: number): void {
  localStorage.setItem(MOUSE_SCROLL_INTERVAL_KEY, String(interval));
}

export function getSkipUpdate() {
  const skip = getWithExpiry(SKIP_UPDATE_KEY);
  return skip === 'true';
}

export function setSkipUpdate(skip: boolean) {
  const expiry = 3 * 24 * 60 * 60 * 1000; // 3 days
  setWithExpiry(SKIP_UPDATE_KEY, String(skip), expiry);
}

export function setKeyboardSystem(system: string) {
  localStorage.setItem(KEYBOARD_SYSTEM_KEY, system);
}

export function getKeyboardSystem() {
  return localStorage.getItem(KEYBOARD_SYSTEM_KEY);
}

export function setKeyboardLanguage(language: string) {
  localStorage.setItem(KEYBOARD_LANGUAGE_KEY, language);
}

export function getKeyboardLanguage() {
  return localStorage.getItem(KEYBOARD_LANGUAGE_KEY);
}

export function setKeyboardSwapCommandCtrl(enabled: boolean) {
  localStorage.setItem(KEYBOARD_SWAP_COMMAND_CTRL_KEY, String(enabled));
}

export function getKeyboardSwapCommandCtrl() {
  try {
    return localStorage.getItem(KEYBOARD_SWAP_COMMAND_CTRL_KEY) === 'true';
  } catch {
    return false;
  }
}

export function setSkipModifyPassword(skip: boolean) {
  const expiry = 3 * 24 * 60 * 60 * 1000; // 3 days
  setWithExpiry(SKIP_MODIFY_PASSWORD_KEY, String(skip), expiry);
}

export function getSkipModifyPassword() {
  const skip = getWithExpiry(SKIP_MODIFY_PASSWORD_KEY);
  return skip === 'true';
}

export function setSkipBrowserNotification(skip: boolean) {
  localStorage.setItem(SKIP_BROWSER_NOTIFICATION_KEY, String(skip));
}

export function getSkipBrowserNotification() {
  const skip = localStorage.getItem(SKIP_BROWSER_NOTIFICATION_KEY);
  return skip === 'true';
}

export function getMenuDisabledItems(): string[] {
  const value = localStorage.getItem(MENU_DISABLED_ITEMS_KEY);
  return value ? JSON.parse(value) : getDefaultMenuDisabledItems();
}

export function getDefaultMenuDisabledItems(): string[] {
  return [...DEFAULT_MENU_DISABLED_ITEMS];
}

export function normalizeMenuDisabledItems(items?: string[] | null): string[] {
  if (Array.isArray(items)) {
    return items;
  }

  return getDefaultMenuDisabledItems();
}

export function getMenuDisplayMode(): string {
  const value = localStorage.getItem(MENU_AUTO_HIDE_KEY);
  return value || 'auto';
}

export function setMenuDisplayMode(mode: string) {
  localStorage.setItem(MENU_AUTO_HIDE_KEY, mode);
}

export function getPowerConfirm() {
  const enabled = localStorage.getItem(POWER_CONFIRM_KEY);
  return enabled === 'true';
}

export function setPowerConfirm(enabled: boolean) {
  localStorage.setItem(POWER_CONFIRM_KEY, String(enabled));
}

export function getSavedVideoVolume(): number | null {
  const stored = localStorage.getItem(VIDEO_VOLUME_KEY);
  if (!stored?.trim()) return null;

  const volume = Number(stored);
  return Number.isFinite(volume) && volume >= 0 && volume <= 100 ? volume : null;
}

export function setSavedVideoVolume(volume: number) {
  if (!Number.isFinite(volume) || volume < 0 || volume > 100) return;

  localStorage.setItem(VIDEO_VOLUME_KEY, String(volume));
}

export function getPortalMode(): PortalMode {
  try {
    const mode = localStorage.getItem(PORTAL_MODE_KEY);
    if (mode === 'auto' || mode === 'manual' || mode === 'off') return mode;
    if (mode === 'on') return 'auto';

    return 'auto';
  } catch {
    return 'auto';
  }
}

export function getStoredPortalMode(): PortalMode | null {
  try {
    const mode = localStorage.getItem(PORTAL_MODE_KEY);
    if (mode === 'auto' || mode === 'manual' || mode === 'off') return mode;
    if (mode === 'on') return 'auto';
    return null;
  } catch {
    return null;
  }
}

export function setPortalMode(mode: PortalMode) {
  try {
    localStorage.setItem(PORTAL_MODE_KEY, mode);
  } catch {
    // The current page remains usable when browser storage is unavailable.
  }
}

export function getWifiPortalViewMode(): WifiPortalViewMode {
  try {
    return localStorage.getItem(WIFI_PORTAL_VIEW_MODE_KEY) === 'detailed' ? 'detailed' : 'simple';
  } catch {
    return 'simple';
  }
}

export function setWifiPortalViewMode(mode: WifiPortalViewMode) {
  try {
    localStorage.setItem(WIFI_PORTAL_VIEW_MODE_KEY, mode);
  } catch {
    // The provisioning page remains usable when browser storage is unavailable.
  }
}

export function getPortalBorderEnabled() {
  try {
    return localStorage.getItem(PORTAL_BORDER_ENABLED_KEY) === 'true';
  } catch {
    return false;
  }
}

export function setPortalBorderEnabled(enabled: boolean) {
  try {
    localStorage.setItem(PORTAL_BORDER_ENABLED_KEY, String(enabled));
  } catch {
    // The current page remains usable when browser storage is unavailable.
  }
}

export function getPortalBorderColor() {
  try {
    const value = localStorage.getItem(PORTAL_BORDER_COLOR_KEY);
    return /^#[0-9a-f]{6}$/i.test(value ?? '') ? value! : '#00e5ff';
  } catch {
    return '#00e5ff';
  }
}

export function setPortalBorderColor(color: string) {
  if (!/^#[0-9a-f]{6}$/i.test(color)) return;
  try {
    localStorage.setItem(PORTAL_BORDER_COLOR_KEY, color);
  } catch {
    // The current page remains usable when browser storage is unavailable.
  }
}

export function getLoginUsername() {
  try {
    return localStorage.getItem(LOGIN_USERNAME_KEY) || '';
  } catch {
    return '';
  }
}

export function setLoginUsername(username: string): boolean {
  try {
    localStorage.setItem(LOGIN_USERNAME_KEY, username);
    return true;
  } catch {
    // Login remains usable when browser storage is unavailable.
    return false;
  }
}

function parseRememberedLoginPassword(value: string): RememberedLoginPassword | null {
  try {
    const parsed: unknown = JSON.parse(value);
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) return null;

    const item = parsed as Record<string, unknown>;
    if (
      typeof item.value !== 'string' ||
      !item.value ||
      typeof item.expiry !== 'number' ||
      !Number.isFinite(item.expiry)
    ) {
      return null;
    }

    return { value: item.value, expiry: item.expiry };
  } catch {
    return null;
  }
}

export function getLoginPassword(): string {
  let stored: string | null;
  try {
    stored = localStorage.getItem(LOGIN_PASSWORD_KEY);
  } catch {
    return '';
  }

  if (stored === null) return '';

  const password = parseRememberedLoginPassword(stored);
  if (!password || Date.now() >= password.expiry) {
    removeLoginPassword();
    return '';
  }

  return password.value;
}

export function setLoginPassword(password: string): boolean {
  if (!password) return removeLoginPassword();

  const rememberedPassword: RememberedLoginPassword = {
    value: password,
    expiry: Date.now() + REMEMBERED_LOGIN_PASSWORD_TTL_MS
  };

  try {
    localStorage.setItem(LOGIN_PASSWORD_KEY, JSON.stringify(rememberedPassword));
    return true;
  } catch {
    return false;
  }
}

export function removeLoginPassword(): boolean {
  try {
    localStorage.removeItem(LOGIN_PASSWORD_KEY);
    return true;
  } catch {
    return false;
  }
}
