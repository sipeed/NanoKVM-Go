import type { Store } from 'jotai/vanilla/store';

import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import {
  isSettingsDetailOpenAtom,
  isSettingsOpenAtom,
  settingTabAtom,
  settingTabIDs,
  submenuOpenCountAtom,
  type SettingTab
} from '@/jotai/settings.ts';

export type MockTarget = {
  type: 'settings';
  tab: SettingTab;
};

/** Older mocked deep links keep working after the settings tab was renamed. */
const legacySettingTabAliases: Record<string, SettingTab> = {
  tailscale: 'remoteNetwork'
};

let didWarnForInvalidTarget = false;

function getMockTargetValue(hash: string): string | null {
  const queryStart = hash.indexOf('?');
  if (queryStart === -1) return null;

  return new URLSearchParams(hash.slice(queryStart + 1)).get('mockTarget');
}

/**
 * Device-level feature switches a mocked session can flip for direct debugging.
 * The only supported selector is `mockFeature=tailscale_cn:on|off`; unknown
 * keys and values are ignored so a stale deep link cannot weaken a mock.
 */
export type MockFeatureOverrides = {
  tailscaleCnEnabled?: boolean;
};

function getMockFeatureValue(hash: string): string | null {
  const queryStart = hash.indexOf('?');
  if (queryStart === -1) return null;

  return new URLSearchParams(hash.slice(queryStart + 1)).get('mockFeature');
}

/**
 * Parses the dev-only `mockFeature` hash selector. It is a pure function: the
 * caller (the mocked bootstrap) applies the result to the mock state.
 */
export function parseMockFeatures(hash: string): MockFeatureOverrides {
  const raw = getMockFeatureValue(hash);
  if (!raw) return {};

  const overrides: MockFeatureOverrides = {};
  for (const entry of raw.split(',')) {
    const separator = entry.indexOf(':');
    if (separator === -1) continue;
    const key = entry.slice(0, separator).trim();
    const value = entry.slice(separator + 1).trim().toLowerCase();
    if (key !== 'tailscale_cn') continue;
    if (value === 'off') overrides.tailscaleCnEnabled = false;
    else if (value === 'on') overrides.tailscaleCnEnabled = true;
  }
  return overrides;
}

export function parseMockTarget(hash: string): MockTarget | null {
  const value = getMockTargetValue(hash);
  if (!value || !value.startsWith('settings:')) return null;

  const tab = value.slice('settings:'.length);
  const resolvedTab = settingTabIDs.includes(tab as SettingTab)
    ? (tab as SettingTab)
    : legacySettingTabAliases[tab];
  if (!resolvedTab) return null;

  return { type: 'settings', tab: resolvedTab };
}

export function applyMockTarget(
  hash: string,
  store: Store,
  warn: (message: string) => void = console.warn
): MockTarget | null {
  const rawTarget = getMockTargetValue(hash);
  if (rawTarget === null) return null;

  const target = parseMockTarget(hash);
  if (!target) {
    if (!didWarnForInvalidTarget) {
      didWarnForInvalidTarget = true;
      warn(`[mocked] Ignored unsupported mockTarget: ${rawTarget}`);
    }
    return null;
  }

  store.set(settingTabAtom, target.tab);
  store.set(isSettingsOpenAtom, true);
  store.set(isSettingsDetailOpenAtom, true);
  store.set(isKeyboardEnableAtom, false);
  store.set(submenuOpenCountAtom, 1);

  return target;
}

export function resetMockTargetWarningForTests(): void {
  didWarnForInvalidTarget = false;
}
