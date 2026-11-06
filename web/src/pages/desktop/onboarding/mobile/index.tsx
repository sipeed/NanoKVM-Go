import { useEffect, useRef, useState } from 'react';
import { Button } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { getOnboarding, setMobileOnboardingCompleted } from '@/api/vm.ts';
import { LAYER_ONBOARDING_MOBILE_PROMPT } from '@/lib/layers.ts';
import {
  onboardingActiveAtom,
  onboardingMobileLaterDismissedAtom,
  onboardingMobilePromptOpenAtom,
  startOnboardingAtom
} from '@/jotai/onboarding.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import { isSettingsOpenAtom, menuConfigInitializedAtom } from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';
import { LanguagePicker } from '@/components/language-picker.tsx';

import { OnboardingMobileTour, type MobileOnboardingEndReason } from './tour.tsx';
import { MOBILE_ONBOARDING_VERSION, shouldOfferMobileOnboarding } from './steps.ts';

// First-run trigger + persistence wiring for the mobile-portrait tour (option
// E). Mirrors the desktop controller's contract: fetch the device-level
// completion version once, fail-closed soft prompt, optimistic completion POST.
// Renders and fetches only in mobile portrait; the desktop tour owns every
// other form factor (mutual exclusion, see design.md §3).
export const OnboardingMobile = () => {
  const { t } = useTranslation();
  const { isMobilePortrait } = useResponsiveDevice();

  const videoMode = useAtomValue(videoModeAtom);
  const menuConfigInitialized = useAtomValue(menuConfigInitializedAtom);
  const [promptOpen, setPromptOpen] = useAtom(onboardingMobilePromptOpenAtom);
  const [laterDismissed, setLaterDismissed] = useAtom(onboardingMobileLaterDismissedAtom);
  const tourActive = useAtomValue(onboardingActiveAtom);
  const settingsOpen = useAtomValue(isSettingsOpenAtom);
  const startOnboarding = useSetAtom(startOnboardingAtom);

  // null = unknown (not fetched yet, or the fetch failed); fail-closed below.
  const [completedVersion, setCompletedVersion] = useState<string | null>(null);
  const fetchedRef = useRef(false);

  useEffect(() => {
    if (!isMobilePortrait || fetchedRef.current) return;
    fetchedRef.current = true;

    getOnboarding()
      .then((rsp) => {
        if (rsp.code === 0 && typeof rsp.data?.mobileCompletedVersion === 'string') {
          setCompletedVersion(rsp.data.mobileCompletedVersion);
        } else {
          console.error('[onboarding-mobile] unexpected completion-state response:', rsp.code);
        }
      })
      .catch((error) => {
        console.error('[onboarding-mobile] failed to fetch completion state', error);
      });
  }, [isMobilePortrait]);

  const offered = shouldOfferMobileOnboarding({
    isMobilePortrait,
    videoMode,
    menuConfigInitialized,
    mobileCompletedVersion: completedVersion,
    laterDismissed,
    tourActive
  });

  // Mirror the offer gate in both directions (desktop prompt contract).
  useEffect(() => {
    setPromptOpen(offered);
  }, [offered, setPromptOpen]);

  // Best-effort persistence, same policy as the desktop controller: a failed
  // POST only means the prompt may reappear on the next load.
  function persistCompleted() {
    // Optimistic: suppress the prompt for this session even while the POST is
    // in flight (endTour flipping tourActive off would otherwise re-open it).
    setCompletedVersion(MOBILE_ONBOARDING_VERSION);
    setMobileOnboardingCompleted(MOBILE_ONBOARDING_VERSION)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.error('[onboarding-mobile] completion rejected:', rsp.code);
        }
      })
      .catch((error) => {
        console.error('[onboarding-mobile] failed to persist completion', error);
      });
  }

  function handleStart() {
    // The guarded atom refuses to start over an exclusive input session; only
    // close the prompt on success so the user can retry once the session ends.
    if (startOnboarding()) {
      setPromptOpen(false);
    }
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

  function handleTourEnd(reason: MobileOnboardingEndReason) {
    // Finish and skip count as "seen"; a rotation abort must not persist —
    // the tour never ran to completion.
    if (reason === 'abort') return;
    persistCompleted();
  }

  return (
    <>
      <OnboardingMobileTour onEnd={handleTourEnd} />
      {/* The card must clear the rail (1000) yet stay under the LanguagePicker's
          Dropdown popup (1050) — LAYER_ONBOARDING_MOBILE_PROMPT sits between.
          It also yields to the settings Modal (1000): hide while settings is
          open instead of painting over it. */}
      {promptOpen && !tourActive && !settingsOpen && isMobilePortrait && (
        <div
          className="fixed left-1/2 top-14 w-[min(320px,calc(100vw-32px))] -translate-x-1/2 rounded-lg border border-neutral-700 bg-neutral-900/95 p-4 shadow-xl"
          style={{ zIndex: LAYER_ONBOARDING_MOBILE_PROMPT }}
        >
          <div className="text-sm font-semibold text-neutral-200">
            {t('onboarding.mobile.prompt.title')}
          </div>
          <div className="mt-1 text-xs text-neutral-500">
            {t('onboarding.mobile.prompt.description')}
          </div>
          <div className="mt-3 flex items-center justify-between">
            <span className="text-xs text-neutral-500">{t('onboarding.prompt.language')}</span>
            <LanguagePicker className="min-h-[44px]" />
          </div>
          <div className="mt-3 flex items-center justify-end gap-2">
            <Button size="small" type="text" className="min-h-[44px]" onClick={handleNever}>
              {t('onboarding.prompt.never')}
            </Button>
            <Button size="small" className="min-h-[44px]" onClick={handleLater}>
              {t('onboarding.prompt.later')}
            </Button>
            <Button size="small" type="primary" className="min-h-[44px]" onClick={handleStart}>
              {t('onboarding.prompt.start')}
            </Button>
          </div>
        </div>
      )}
    </>
  );
};
