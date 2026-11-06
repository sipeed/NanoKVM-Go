import { atom } from 'jotai';

import { inputCalibrationSessionAtom } from './input-profile.ts';
import { isKeyboardEnableAtom } from './keyboard.ts';
import { portalSelectingAtom } from './screen.ts';

// is the first-run onboarding tour running
export const onboardingActiveAtom = atom(false);

// current tour step index, driven by the tour engine
export const onboardingStepAtom = atom(0);

// is the first-run soft prompt card visible
export const onboardingPromptOpenAtom = atom(false);

// session-only dismissal recorded by the prompt's "Later" action;
// intentionally not persisted so the next page load re-prompts
export const onboardingLaterDismissedAtom = atom(false);

// mobile-portrait guide: soft prompt visibility, separate from the desktop
// prompt atom so the two form factors never cross-talk through their mirror
// effects (the desktop controller forces its prompt closed in portrait)
export const onboardingMobilePromptOpenAtom = atom(false);

// mobile-portrait session-only "Later" dismissal
export const onboardingMobileLaterDismissedAtom = atom(false);

// host keyboard input gate: the shared enable flag minus an active tour.
// Read-only by construction; the many writers of isKeyboardEnableAtom keep
// their semantics and can never break the tour shield.
// (menuConfigInitializedAtom, the tour trigger's menu-load signal, lives in
// jotai/settings.ts beside menuDisabledItemsAtom — the menu domain's owner.)
export const keyboardInputEnabledAtom = atom(
  (get) => get(isKeyboardEnableAtom) && !get(onboardingActiveAtom)
);

// exclusive screen-wide input sessions that must not overlap with the tour
export const onboardingSessionConflictAtom = atom(
  (get) => get(inputCalibrationSessionAtom) !== null || get(portalSelectingAtom)
);

// Guarded tour entry. Refuses to start over a calibration or portal-selection
// session (no-op + warn); the reverse direction is structurally blocked
// because those sessions are opened from the menu rail, which the tour pins
// and shields. Returns whether the tour is running after the call.
export const startOnboardingAtom = atom(null, (get, set) => {
  if (get(onboardingActiveAtom)) return true;
  if (get(onboardingSessionConflictAtom)) {
    console.warn('[onboarding] tour start rejected: exclusive input session active');
    return false;
  }
  set(onboardingStepAtom, 0);
  set(onboardingPromptOpenAtom, false);
  set(onboardingActiveAtom, true);
  return true;
});
