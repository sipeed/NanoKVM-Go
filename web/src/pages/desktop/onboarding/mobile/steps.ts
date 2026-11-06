import {
  deriveOnboardingSteps,
  type OnboardingGroup,
  type OnboardingStepInput
} from '../steps.ts';

// Pure derivation for the mobile-portrait tour (option E). Reuses the desktop
// group derivation with isMobilePortrait=true (which already strips the
// window-only items), then inserts the whole-rail step after welcome: the
// floating rail itself is the novel element a first-run mobile user must find.

export const MOBILE_ONBOARDING_VERSION = 'mobile-v1';

export type MobileOnboardingStepDef = {
  key: 'welcome' | 'rail' | OnboardingGroup | 'finish';
  /** 'rail' spotlights the whole rail box; a group spotlights its union rect;
   * absent for the centered welcome/finish cards. */
  anchor?: 'rail' | OnboardingGroup;
  /** Item line keys under onboarding.steps.<group>.items (group steps only). */
  items: string[];
};

export type MobileOnboardingStepInput = Omit<OnboardingStepInput, 'isMobilePortrait'>;

export function deriveMobileOnboardingSteps(
  input: MobileOnboardingStepInput
): MobileOnboardingStepDef[] {
  const desktopDefs = deriveOnboardingSteps({ ...input, isMobilePortrait: true });

  const steps: MobileOnboardingStepDef[] = [
    { key: 'welcome', items: [] },
    { key: 'rail', anchor: 'rail', items: [] }
  ];
  for (const def of desktopDefs.slice(1, -1)) {
    steps.push({ key: def.key, anchor: def.group, items: def.items });
  }
  steps.push({ key: 'finish', items: [] });
  return steps;
}

export type MobileOnboardingOfferInput = {
  // the mobile tour owns the touch-portrait form factor exclusively
  isMobilePortrait: boolean;
  // the rail only exists once a video mode is selected
  videoMode: string | null;
  // menu enablement must be loaded so derived steps match the visible rail
  menuConfigInitialized: boolean;
  // device-level mobile completion flag; null = not fetched or fetch failed
  mobileCompletedVersion: string | null;
  // session-only "Later" dismissal
  laterDismissed: boolean;
  tourActive: boolean;
};

// Whether the mobile first-run soft prompt may open. Fail-closed on an unknown
// completion version, same contract as the desktop offer gate.
export function shouldOfferMobileOnboarding(input: MobileOnboardingOfferInput): boolean {
  if (!input.isMobilePortrait) return false;
  if (!input.videoMode || !input.menuConfigInitialized) return false;
  if (input.mobileCompletedVersion === null) return false;
  if (input.mobileCompletedVersion === MOBILE_ONBOARDING_VERSION) return false;
  if (input.laterDismissed || input.tourActive) return false;
  return true;
}
