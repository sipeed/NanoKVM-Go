import { atom } from 'jotai';

import {
  getConnectPreferences,
  getDefaultMenuDisabledItems,
  setConnectPreferences,
  type ConnectFeature
} from '@/lib/localstorage.ts';

export const settingTabIDs = [
  'about',
  'appearance',
  'screen',
  'device',
  'ai',
  'apps',
  'remoteNetwork',
  'update',
  'account'
] as const;

export type SettingTab = (typeof settingTabIDs)[number];

// is the settings modal opened
export const isSettingsOpenAtom = atom(false);

// current setting tab
export const settingTabAtom = atom<SettingTab>('about');

// show the active settings tab content on mobile
export const isSettingsDetailOpenAtom = atom(false);

// web title
export const webTitleAtom = atom('');

// menu bar disabled items
export const menuDisabledItemsAtom = atom<string[]>(getDefaultMenuDisabledItems());

// the desktop menu bar finished its initial config fetch. Lifted from the
// menu's former local state; menuDisabledItemsAtom cannot signal this because
// it has a synchronous storage default.
export const menuConfigInitializedAtom = atom(false);

// track how many submenus are currently open
export const submenuOpenCountAtom = atom(0);

// screen timelapse service state from the AI service event stream; gates the
// menu's timelapse item and the onboarding tour's automation step
export const screenTimelapseEnabledAtom = atom(false);

// menu display mode: 'off' | 'auto' | 'always'
export const menuDisplayModeAtom = atom<string>('auto');

const savedConnectPreferencesAtom = atom(getConnectPreferences());
export const connectPreferencesAtom = atom(
  (get) => get(savedConnectPreferencesAtom),
  (get, set, feature: ConnectFeature, checked: boolean) => {
    const next = { ...get(savedConnectPreferencesAtom), [feature]: checked };
    setConnectPreferences(next);
    set(savedConnectPreferencesAtom, next);
  }
);
