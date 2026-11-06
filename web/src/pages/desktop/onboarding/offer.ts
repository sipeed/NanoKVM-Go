export type OnboardingOfferInput = {
  // owner rule: the onboarding tour never runs on mobile devices
  isMobileDevice: boolean;
  // mutual exclusion: the mobile-portrait guide owns the touch-portrait form
  // factor — a large touchscreen in portrait renders the mobile rail, so the
  // desktop tour's anchors would measure the wrong layout there
  isMobilePortrait: boolean;
  // the rail only exists once a video mode is selected
  videoMode: string | null;
  // menu enablement must be loaded so derived steps match the visible rail
  menuConfigInitialized: boolean;
  // device-level completion flag; null = not fetched or fetch failed
  completedVersion: string | null;
  // session-only "Later" dismissal
  laterDismissed: boolean;
  tourActive: boolean;
  currentVersion: string;
};

// Whether the first-run soft prompt may open. Fail-closed on an unknown
// completion version: a GET error must never turn into a nagging prompt.
export function shouldOfferOnboarding(input: OnboardingOfferInput): boolean {
  if (input.isMobileDevice || input.isMobilePortrait) return false;
  if (!input.videoMode || !input.menuConfigInitialized) return false;
  if (input.completedVersion === null) return false;
  if (input.completedVersion === input.currentVersion) return false;
  if (input.laterDismissed || input.tourActive) return false;
  return true;
}
