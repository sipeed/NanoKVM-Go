import { useAtomValue } from 'jotai';

import { mouseModeAtom, pointerModeDeviceStatusAtom, pointerModeInputEnabledAtom } from '@/jotai/mouse.ts';
import { onboardingActiveAtom } from '@/jotai/onboarding.ts';

import { Absolute } from './absolute.tsx';
import { CalibrationOverlay } from './calibration-overlay.tsx';
import { InputProfileController } from './input-profile-controller.tsx';
import { InputRegion } from './input-region.tsx';
import { PointerEventsCap } from './pointer-events-cap.tsx';
import { Relative } from './relative.tsx';
import { TouchScreen } from './touch-screen.tsx';

export const Mouse = () => {
  const mouseMode = useAtomValue(mouseModeAtom);
  const { singleEndpoint = false } = useAtomValue(pointerModeDeviceStatusAtom);
  const pointerInputEnabled = useAtomValue(pointerModeInputEnabledAtom);
  const onboardingActive = useAtomValue(onboardingActiveAtom);
  // The onboarding tour shields host-bound pointer input by unmounting the
  // producers; their effect cleanups emit release reports on the way out.
  const inputEnabled = pointerInputEnabled && !onboardingActive;

  return (
    <>
      <InputProfileController />
      <InputRegion />
      <CalibrationOverlay />
      {inputEnabled && (
        <>
          <PointerEventsCap />
          {mouseMode === 'relative' ? (
            <Relative />
          ) : mouseMode === 'touch-screen' ? (
            <TouchScreen />
          ) : (
            <Absolute key={singleEndpoint ? 'single' : 'dual'} singleEndpoint={singleEndpoint} />
          )}
        </>
      )}
    </>
  );
};
