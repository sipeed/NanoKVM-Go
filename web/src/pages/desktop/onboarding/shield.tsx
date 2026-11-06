import { useEffect, useRef, type SyntheticEvent } from 'react';
import { useAtomValue } from 'jotai';

import { LAYER_ONBOARDING_BACKDROP } from '@/lib/layers.ts';
import { onboardingActiveAtom } from '@/jotai/onboarding.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

function blockEvent(event: SyntheticEvent) {
  event.preventDefault();
  event.stopPropagation();
}

// Fully transparent full-screen shield rendered while the onboarding tour
// runs. The antd Tour mask owns the dimming; this layer only guarantees that
// no pointer, wheel, or context-menu event can reach the desktop surfaces
// below (screen input region, menu rail, popovers) — including any moment the
// tour mask itself is between mounts. The tour popover renders above at
// LAYER_ONBOARDING and stays interactive.
export const OnboardingShield = () => {
  // Covers whichever tour can run on this form factor: the mobile-portrait
  // guide on touch portrait, or the desktop tour on non-mobile devices.
  const { isMobileDevice, isMobilePortrait } = useResponsiveDevice();
  const active = useAtomValue(onboardingActiveAtom) && (isMobilePortrait || !isMobileDevice);
  const shieldRef = useRef<HTMLDivElement>(null);

  // React attaches onWheel as a passive root listener, where preventDefault
  // is a no-op; block native scroll with a non-passive listener instead
  // (pointer-events-cap.tsx precedent).
  useEffect(() => {
    const node = shieldRef.current;
    if (!node) return;

    const blockWheel = (event: WheelEvent) => {
      event.preventDefault();
      event.stopPropagation();
    };
    node.addEventListener('wheel', blockWheel, { passive: false });
    return () => node.removeEventListener('wheel', blockWheel);
  }, [active]);

  if (!active) return null;

  return (
    <div
      ref={shieldRef}
      aria-hidden="true"
      className="fixed inset-0 touch-none bg-transparent"
      style={{ zIndex: LAYER_ONBOARDING_BACKDROP }}
      onPointerDown={blockEvent}
      onPointerMove={blockEvent}
      onPointerUp={blockEvent}
      onPointerCancel={blockEvent}
      onContextMenu={blockEvent}
    />
  );
};
