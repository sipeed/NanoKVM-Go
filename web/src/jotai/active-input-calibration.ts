import { atom } from 'jotai';

import { AUTO_INPUT_PROFILE_ID, createAutomaticCropCalibration } from '@/lib/input-profile.ts';

import { inputCalibrationSessionAtom, selectedInputProfileAtom } from './input-profile.ts';
import { mouseModeAtom } from './mouse.ts';
import { effectiveScreenAreaAtom } from './screen.ts';

export const activeCalibrationAtom = atom((get) => {
  const mode = get(mouseModeAtom);
  if (mode === 'relative') return null;

  const session = get(inputCalibrationSessionAtom);
  if (session?.mode === mode) return null;

  const profile = get(selectedInputProfileAtom);
  if (profile?.id === AUTO_INPUT_PROFILE_ID) {
    return createAutomaticCropCalibration(mode, get(effectiveScreenAreaAtom));
  }
  return profile?.calibrations[mode] ?? null;
});
