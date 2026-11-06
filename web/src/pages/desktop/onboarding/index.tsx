import { useEffect, useRef, useState } from 'react';
import { Button } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { getOnboarding, setOnboardingCompleted } from '@/api/vm.ts';
import { LAYER_DESKTOP_MENU_POPOVER } from '@/lib/layers.ts';
import {
  onboardingActiveAtom,
  onboardingLaterDismissedAtom,
  onboardingPromptOpenAtom,
  startOnboardingAtom
} from '@/jotai/onboarding.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import { menuConfigInitializedAtom } from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';
import { LanguagePicker } from '@/components/language-picker.tsx';

import { shouldOfferOnboarding } from './offer.ts';
import { OnboardingShield } from './shield.tsx';
import { ONBOARDING_VERSION } from './steps.ts';
import { OnboardingTour } from './tour.tsx';

// First-run trigger + persistence wiring for the onboarding tour. Owns the
// completion-version fetch, the soft prompt card, and the completion POST.
// Nothing here renders or fetches on mobile devices (owner rule) or in mobile
// portrait — that form factor is owned by the mobile tour (mutual exclusion).
export const Onboarding = () => {
  const { t } = useTranslation();
  const { isMobileDevice, isMobilePortrait } = useResponsiveDevice();

  const videoMode = useAtomValue(videoModeAtom);
  const menuConfigInitialized = useAtomValue(menuConfigInitializedAtom);
  const [promptOpen, setPromptOpen] = useAtom(onboardingPromptOpenAtom);
  const [laterDismissed, setLaterDismissed] = useAtom(onboardingLaterDismissedAtom);
  const tourActive = useAtomValue(onboardingActiveAtom);
  const startOnboarding = useSetAtom(startOnboardingAtom);

  // null = unknown (not fetched yet, or the fetch failed); fail-closed below.
  const [completedVersion, setCompletedVersion] = useState<string | null>(null);
  const fetchedRef = useRef(false);

  useEffect(() => {
    if (isMobileDevice || isMobilePortrait || fetchedRef.current) return;
    fetchedRef.current = true;

    getOnboarding()
      .then((rsp) => {
        if (rsp.code === 0 && typeof rsp.data?.completedVersion === 'string') {
          setCompletedVersion(rsp.data.completedVersion);
        } else {
          console.error('[onboarding] unexpected completion-state response:', rsp.code);
        }
      })
      .catch((error) => {
        console.error('[onboarding] failed to fetch completion state', error);
      });
  }, [isMobileDevice, isMobilePortrait]);

  const offered = shouldOfferOnboarding({
    isMobileDevice,
    isMobilePortrait,
    videoMode,
    menuConfigInitialized,
    completedVersion,
    laterDismissed,
    tourActive,
    currentVersion: ONBOARDING_VERSION
  });

  // Mirror the offer gate in both directions: opening is the common path,
  // and closing clears any residue (e.g. the atom survives a Desktop remount).
  useEffect(() => {
    setPromptOpen(offered);
  }, [offered, setPromptOpen]);

  // Completion is best-effort: a failed POST only means the prompt may
  // reappear on the next load, which beats an error toast for a tour.
  function persistCompleted() {
    // Optimistic: suppress the prompt for this session even while the POST is
    // in flight. Without this, endTour flipping tourActive off would re-open
    // the prompt until the POST resolves (offered false→true transition).
    setCompletedVersion(ONBOARDING_VERSION);
    setOnboardingCompleted(ONBOARDING_VERSION)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.error('[onboarding] completion rejected:', rsp.code);
        }
      })
      .catch((error) => {
        console.error('[onboarding] failed to persist completion', error);
      });
  }

  function handleStart() {
    // The atom closes the prompt itself and refuses to start over an
    // exclusive input session (calibration / portal selection). A refusal
    // leaves the prompt open so the user can retry once the session ends;
    // the boolean return is intentionally unused.
    startOnboarding();
  }

  function handleLater() {
    // Session-only dismissal: the next page load offers the tour again.
    setLaterDismissed(true);
    setPromptOpen(false);
  }

  function handleNever() {
    setPromptOpen(false);
    persistCompleted();
  }

  function handleTourEnd() {
    // Finish and skip both count as "seen": the tour either taught the rail
    // layout or the user opted out of it.
    persistCompleted();
  }

  return (
    <>
      <OnboardingShield />
      <OnboardingTour onEnd={handleTourEnd} />
      {promptOpen && !tourActive && !isMobileDevice && !isMobilePortrait && (
        <div
          className="fixed left-1/2 top-14 w-[320px] -translate-x-1/2 rounded-lg border border-neutral-700 bg-neutral-900/95 p-4 shadow-xl"
          style={{ zIndex: LAYER_DESKTOP_MENU_POPOVER }}
        >
          <div className="text-sm font-semibold text-neutral-200">
            {t('onboarding.prompt.title')}
          </div>
          <div className="mt-1 text-xs text-neutral-500">{t('onboarding.prompt.description')}</div>
          <div className="mt-3 flex items-center justify-between">
            <span className="text-xs text-neutral-500">{t('onboarding.prompt.language')}</span>
            <LanguagePicker />
          </div>
          <div className="mt-3 flex items-center justify-end gap-2">
            <Button size="small" type="text" onClick={handleNever}>
              {t('onboarding.prompt.never')}
            </Button>
            <Button size="small" onClick={handleLater}>
              {t('onboarding.prompt.later')}
            </Button>
            <Button size="small" type="primary" onClick={handleStart}>
              {t('onboarding.prompt.start')}
            </Button>
          </div>
        </div>
      )}
    </>
  );
};
