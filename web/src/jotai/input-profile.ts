import { atom } from 'jotai';

import { setInputProfileConfig } from '@/api/vm.ts';
import { normalizeInputProfileStore } from '@/lib/input-profile.ts';
import type {
  CalibratableMouseMode,
  CalibrationCorner,
  CalibrationTargetRect,
  FourPointCalibration,
  InputProfileStore,
  NormalizedPoint
} from '@/lib/input-profile.ts';
import * as ls from '@/lib/localstorage.ts';
import type { ScreenCropState } from '@/lib/screen-crop.ts';

const inputProfileStoreStateAtom = atom<InputProfileStore>(
  normalizeInputProfileStore(ls.getInputProfileStore())
);

export const inputProfileStoreAtom = atom(
  (get) => get(inputProfileStoreStateAtom),
  (_get, set, value: InputProfileStore) => {
    const store = normalizeInputProfileStore(value);
    set(inputProfileStoreStateAtom, store);
    ls.setInputProfileStore(store);
    void setInputProfileConfig(store).catch(() => undefined);
  }
);

export const hydrateInputProfileStoreAtom = atom(null, (_get, set, value: InputProfileStore) => {
  const store = normalizeInputProfileStore(value);
  set(inputProfileStoreStateAtom, store);
  ls.setInputProfileStore(store);
});

export type InputCalibrationSession = {
  phase: 'position' | 'capture';
  profileId: string;
  mode: CalibratableMouseMode;
  targetRect: CalibrationTargetRect;
  cornerIndex: number;
  controlPoints?: Record<CalibrationCorner, NormalizedPoint>;
  hidSamples: Partial<Record<CalibrationCorner, NormalizedPoint>>;
};

export const inputCalibrationSessionAtom = atom<InputCalibrationSession | null>(null);
export const calibrationRawPointAtom = atom<NormalizedPoint | null>(null);
export const inputCalibrationScreenAreaAtom = atom<ScreenCropState | null>(null);

export const selectedInputProfileAtom = atom(
  (get) => {
    const store = get(inputProfileStoreAtom);
    return store.profiles.find((profile) => profile.id === store.selectedProfileId) ?? null;
  },
  (get, set, profileId: string) => {
    const store = get(inputProfileStoreAtom);
    const nextStore = { ...store, selectedProfileId: profileId };
    set(inputProfileStoreAtom, nextStore);
  }
);

export function setProfileCalibration(
  store: InputProfileStore,
  profileId: string,
  mode: CalibratableMouseMode,
  calibration: FourPointCalibration | null
): InputProfileStore {
  const now = Date.now();
  const profiles = store.profiles.map((profile) => {
    if (profile.id !== profileId || profile.origin !== 'user') return profile;
    const calibrations = { ...profile.calibrations };
    if (calibration) calibrations[mode] = calibration;
    else delete calibrations[mode];
    return { ...profile, calibrations, updatedAt: now };
  });
  return { ...store, profiles };
}
