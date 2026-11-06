import { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Tour } from 'antd';
import type { TourProps } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { suspendInput } from '@/lib/input-suspension.ts';
import { LAYER_ONBOARDING, LAYER_ONBOARDING_RING } from '@/lib/layers.ts';
import { onboardingActiveAtom, onboardingStepAtom } from '@/jotai/onboarding.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import { menuDisabledItemsAtom, screenTimelapseEnabledAtom } from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

import { measureMenuGroupRect, measureMenuRailRect, type UnionRect } from '../anchor.ts';
import { deriveMobileOnboardingSteps, type MobileOnboardingStepDef } from './steps.ts';

// 'abort' covers the portrait→landscape rotation mid-tour: the tour ends
// without persisting completion, so the prompt may re-offer back in portrait.
export type MobileOnboardingEndReason = 'finish' | 'skip' | 'abort';

type OnboardingMobileTourProps = {
  onEnd?: (reason: MobileOnboardingEndReason) => void;
};

function measureAnchorRect(anchor: NonNullable<MobileOnboardingStepDef['anchor']>): UnionRect | null {
  return anchor === 'rail' ? measureMenuRailRect() : measureMenuGroupRect(anchor);
}

// The mobile-portrait guided tour (option E): the same antd Tour engine and
// interaction contract as the desktop tour — mask, spotlight hole, Prev/Next,
// X/Escape skip — with portrait adaptations: placement follows the rail's
// measured side, buttons meet the 44px touch target, and a self-rendered ring
// makes the anchored control readable against the dark rail.
export const OnboardingMobileTour = ({ onEnd }: OnboardingMobileTourProps) => {
  const { t } = useTranslation();
  const { isMobilePortrait } = useResponsiveDevice();

  const active = useAtomValue(onboardingActiveAtom);
  const [step, setStep] = useAtom(onboardingStepAtom);
  const setActive = useSetAtom(onboardingActiveAtom);
  const disabledItems = useAtomValue(menuDisabledItemsAtom);
  const videoMode = useAtomValue(videoModeAtom);
  const screenTimelapseEnabled = useAtomValue(screenTimelapseEnabledAtom);

  // Two-phase open mirrors the desktop tour: activation re-renders the rail in
  // the same commit, the pin lands through passive-effect state.
  const [pinned, setPinned] = useState(false);

  const proxyRef = useRef<HTMLDivElement>(null);
  const restoreFocusRef = useRef<Element | null>(null);
  const onEndRef = useRef(onEnd);
  onEndRef.current = onEnd;
  const stepRef = useRef(step);
  stepRef.current = step;
  const stepDefsRef = useRef<MobileOnboardingStepDef[]>([]);
  const activeRef = useRef(active);
  activeRef.current = active;

  const enabled = active && isMobilePortrait;

  const endTour = (reason: MobileOnboardingEndReason) => {
    if (!activeRef.current) return;
    activeRef.current = false;
    setStep(0);
    setActive(false);
    onEndRef.current?.(reason);
  };
  const endTourRef = useRef(endTour);
  endTourRef.current = endTour;

  // Rotation abort: leaving portrait mid-tour ends it without persisting
  // completion (the offer gate re-prompts when the device returns to portrait).
  useEffect(() => {
    if (activeRef.current && !isMobilePortrait) {
      endTourRef.current('abort');
    }
  }, [isMobilePortrait]);

  // The reverse rotation (a desktop tour in flight on a non-mobile-UA
  // touchscreen rotating into portrait) enables this tour with a stale desktop
  // step index; restart from the first mobile step on the rising edge.
  const wasEnabledRef = useRef(false);
  useEffect(() => {
    if (enabled && !wasEnabledRef.current) setStep(0);
    wasEnabledRef.current = enabled;
  }, [enabled, setStep]);

  // --- Shield: suspend every HID path at the router (same as desktop).
  useEffect(() => {
    if (!enabled) return;
    return suspendInput('onboarding');
  }, [enabled]);

  // --- Pin settle window.
  useEffect(() => {
    if (!enabled) {
      setPinned(false);
      return;
    }
    let cancelled = false;
    let second = 0;
    const first = requestAnimationFrame(() => {
      second = requestAnimationFrame(() => {
        if (!cancelled) setPinned(true);
      });
    });
    return () => {
      cancelled = true;
      cancelAnimationFrame(first);
      cancelAnimationFrame(second);
    };
  }, [enabled]);

  // --- Focus: capture previous focus on activation, move it to the tour's
  // primary action once pinned, restore on end (desktop contract).
  useEffect(() => {
    if (enabled) {
      restoreFocusRef.current = document.activeElement;
      return;
    }
    const previous = restoreFocusRef.current;
    restoreFocusRef.current = null;
    if (previous instanceof HTMLElement && previous.isConnected) {
      previous.focus();
    }
  }, [enabled]);

  useEffect(() => {
    if (!pinned) return;
    const raf = requestAnimationFrame(() => {
      // rc-tour ships no dialog semantics; add them to the popover element
      // (rootClassName also lands on the mask/placeholder, so scope by ant-tour).
      const popup = document.querySelector('.nanokvm-mobile-tour.ant-tour');
      if (popup instanceof HTMLElement) {
        popup.setAttribute('role', 'dialog');
        popup.setAttribute('aria-modal', 'true');
      }
      const primary = document.querySelector('.nanokvm-mobile-tour .ant-tour-next-btn');
      if (primary instanceof HTMLElement) primary.focus();
    });
    return () => cancelAnimationFrame(raf);
  }, [pinned]);

  const stepDefs = useMemo(
    () => deriveMobileOnboardingSteps({ disabledItems, videoMode, screenTimelapseEnabled }),
    [disabledItems, videoMode, screenTimelapseEnabled]
  );
  stepDefsRef.current = stepDefs;
  const clampedStep = Math.min(step, stepDefs.length - 1);
  const stepDef = stepDefs[clampedStep];

  // --- Keyboard: capture-phase tour navigation (external keyboards on
  // phones/tablets get the desktop contract: Escape skips, arrows navigate).
  useEffect(() => {
    if (!enabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const key = event.key;
      if (key !== 'Escape' && key !== 'ArrowRight' && key !== 'ArrowLeft' && key !== 'Enter') {
        return;
      }
      const target = event.target;
      if (target instanceof HTMLElement && target.closest('.ant-tour') && key === 'Enter') {
        return;
      }

      event.preventDefault();
      event.stopImmediatePropagation();

      if (key === 'Escape') {
        endTourRef.current('skip');
      } else if (key === 'ArrowLeft') {
        setStep(Math.max(0, stepRef.current - 1));
      } else {
        if (stepRef.current >= stepDefsRef.current.length - 1) {
          endTourRef.current('finish');
        } else {
          setStep(stepRef.current + 1);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown, true);
    return () => window.removeEventListener('keydown', handleKeyDown, true);
  }, [enabled, setStep]);

  // Re-measure when the viewport changes while pinned (mobile URL bar
  // collapse/expansion changes the layout viewport without a window resize).
  const [measureTick, setMeasureTick] = useState(0);
  useEffect(() => {
    if (!pinned) return;
    const handleResize = () => setMeasureTick((tick) => tick + 1);
    window.addEventListener('resize', handleResize);
    window.visualViewport?.addEventListener('resize', handleResize);
    return () => {
      window.removeEventListener('resize', handleResize);
      window.visualViewport?.removeEventListener('resize', handleResize);
    };
  }, [pinned]);

  if (!enabled) return null;

  // Measured during render, same contract as the desktop tour: the rail is
  // pinned (dragging and collapse are disabled through onboardingActiveAtom).
  const anchorRect = pinned && stepDef?.anchor ? measureAnchorRect(stepDef.anchor) : null;

  // Popover parks on the side opposite the rail so it never covers the anchor.
  const viewportWidth = window.visualViewport?.width ?? window.innerWidth;
  const placement: 'left' | 'right' =
    anchorRect && anchorRect.left < viewportWidth / 2 ? 'right' : 'left';

  const tourSteps: TourProps['steps'] = stepDefs.map(
    (def): NonNullable<TourProps['steps']>[number] => {
      // rc-tour's top-level scrollIntoViewOptions loses to the per-step
      // default; the fixed proxy is always in-viewport, so opt out per step.
      const base = { scrollIntoViewOptions: false };
      // Owned short labels (i18n/onboarding.ts mobile.tour): antd's locale
      // Prev/Next/Finish (e.g. fr "Étape précédente") and the root
      // sentence-style skip cannot fit the 268px footer row.
      const stepNavProps = {
        prevButtonProps: { children: t('onboarding.mobile.tour.prev') },
        nextButtonProps: { children: t('onboarding.mobile.tour.next') }
      };
      if (!def.anchor) {
        // welcome is an ordinary first step (Next only — antd never renders
        // Prev at index 0); finish ends the tour and also shows the short
        // prev label beside the end button.
        const boundaryNavProps =
          def.key === 'welcome'
            ? { nextButtonProps: { children: t('onboarding.mobile.tour.next') } }
            : {
                ...stepNavProps,
                nextButtonProps: { children: t('onboarding.mobile.tour.end') }
              };
        return {
          ...base,
          ...boundaryNavProps,
          target: null,
          title:
            def.key === 'welcome'
              ? t('onboarding.mobile.welcome.title')
              : t('onboarding.finish.title'),
          description:
            def.key === 'welcome'
              ? t('onboarding.mobile.welcome.description')
              : t('onboarding.finish.description')
        };
      }
      if (def.anchor === 'rail') {
        return {
          ...base,
          ...stepNavProps,
          target: () => proxyRef.current as HTMLElement,
          placement,
          title: t('onboarding.mobile.steps.rail.title'),
          description: t('onboarding.mobile.steps.rail.description')
        };
      }
      const group = def.anchor;
      return {
        ...base,
        ...stepNavProps,
        target: () => proxyRef.current as HTMLElement,
        placement,
        title: t(`onboarding.steps.${group}.title`),
        description: (
          <ul className="m-0 list-none space-y-1 p-0 text-xs leading-relaxed text-neutral-300">
            {def.items.map((item) => (
              <li key={item}>{t(`onboarding.steps.${group}.items.${item}`)}</li>
            ))}
          </ul>
        )
      };
    }
  );

  return (
    <>
      {pinned && anchorRect && (
        <>
          <div
            // Fresh node per anchor step/resize, forcing rc-tour's
            // targetElement identity to change (desktop proxy contract).
            key={`${stepDef?.anchor}:${measureTick}`}
            ref={proxyRef}
            aria-hidden="true"
            className="pointer-events-none fixed"
            style={{
              left: anchorRect.left,
              top: anchorRect.top,
              width: anchorRect.width,
              height: anchorRect.height,
              visibility: 'hidden'
            }}
          />
          {/* self-rendered spotlight ring: rc-tour's hole has no outline, and
              the dark rail needs the extra contrast (option E review) */}
          <div
            aria-hidden="true"
            className="nanokvm-mobile-tour-ring pointer-events-none fixed"
            style={{
              left: anchorRect.left - 6,
              top: anchorRect.top - 6,
              width: anchorRect.width + 12,
              height: anchorRect.height + 12,
              zIndex: LAYER_ONBOARDING_RING
            }}
          />
        </>
      )}
      <Tour
        open={pinned}
        current={clampedStep}
        onChange={(current) => setStep(current)}
        // Same deferral as the desktop tour: onClose fires inside onFinish, and
        // the synchronous finish report must win (endTour is idempotent).
        onClose={() => queueMicrotask(() => endTourRef.current('skip'))}
        onFinish={() => endTourRef.current('finish')}
        steps={tourSteps}
        zIndex={LAYER_ONBOARDING}
        rootClassName="nanokvm-mobile-tour"
        arrow={{ pointAtCenter: true }}
        mask={{ color: 'rgba(8,8,12,.62)' }}
        gap={{ offset: 6, radius: 6 }}
        disabledInteraction
        actionsRender={(originNode, { current, total }) => (
          <>
            {current < total - 1 && (
              <Button size="small" type="text" onClick={() => endTourRef.current('skip')}>
                {t('onboarding.mobile.tour.skip')}
              </Button>
            )}
            {originNode}
          </>
        )}
      />
    </>
  );
};
