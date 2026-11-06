import { Button } from 'antd';
import { useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { startOnboardingAtom } from '@/jotai/onboarding.ts';
import {
  isSettingsDetailOpenAtom,
  isSettingsOpenAtom,
  settingTabAtom,
  submenuOpenCountAtom
} from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

// Replay entry for the first-run menu tour, on Settings → Appearance →
// Menu Bar. Visible on desktop and in mobile portrait (each form factor owns
// a tour); hidden on mobile devices in landscape, where neither tour runs.
export const MenuTour = () => {
  const { t } = useTranslation();
  const { isMobileDevice, isMobilePortrait } = useResponsiveDevice();

  const startOnboarding = useSetAtom(startOnboardingAtom);
  const setIsKeyboardEnable = useSetAtom(isKeyboardEnableAtom);
  const setIsSettingsOpen = useSetAtom(isSettingsOpenAtom);
  const setIsMobileDetailOpen = useSetAtom(isSettingsDetailOpenAtom);
  const setSettingTab = useSetAtom(settingTabAtom);
  const setSubmenuOpenCount = useSetAtom(submenuOpenCountAtom);

  // Mobile landscape has no tour at all; every other form factor does.
  if (isMobileDevice && !isMobilePortrait) return null;

  function replay() {
    // Mirror closeModal() in settings/index.tsx: the modal cannot stay open
    // under the tour (it would cover the rail the tour spotlights), so close
    // it first. This bypasses the update tab's local isLocked guard — a
    // replay click during a firmware update drops the progress view while the
    // server-side update continues; accepted edge case.
    setIsKeyboardEnable(true);
    setIsSettingsOpen(false);
    setIsMobileDetailOpen(false);
    setSettingTab('about');
    setSubmenuOpenCount((count) => Math.max(0, count - 1));
    startOnboarding();
  }

  return (
    <div className="mt-5 flex w-full items-center justify-between">
      <div className="flex flex-col">
        <span>{t('onboarding.replay.label')}</span>
        <span className="text-xs text-neutral-500">{t('onboarding.replay.description')}</span>
      </div>

      <Button
        size="small"
        onClick={replay}
        className={isMobilePortrait ? 'min-h-[44px]' : undefined}
      >
        {t('onboarding.replay.action')}
      </Button>
    </div>
  );
};
