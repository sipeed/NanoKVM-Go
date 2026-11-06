import { useEffect, useMemo, useRef, useState } from 'react';
import { Button, Tour } from 'antd';
import type { TourProps } from 'antd';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import { suspendInput } from '@/lib/input-suspension.ts';
import { LAYER_ONBOARDING } from '@/lib/layers.ts';
import { onboardingActiveAtom, onboardingStepAtom } from '@/jotai/onboarding.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import { menuDisabledItemsAtom, screenTimelapseEnabledAtom } from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

import { measureMenuGroupRect } from './anchor.ts';
import { deriveOnboardingSteps, type OnboardingStepDef } from './steps.ts';

export type OnboardingEndReason = 'finish' | 'skip';

type OnboardingTourProps = {
  onEnd?: (reason: OnboardingEndReason) => void;
};

// The first-run guided tour of the desktop menu rail. Desktop-only: on mobile
// devices the controller renders nothing and never engages the input shield.
// While active it (1) suspends every HID path at the input router, (2) pins
// the rail (see useMenuVisibility), and (3) spotlights one rail group per
// step through a measured proxy element under the antd Tour mask.
export const OnboardingTour = ({ onEnd }: OnboardingTourProps) => {
  const { t } = useTranslation();
  const { isMobileDevice, isMobilePortrait } = useResponsiveDevice();

  const active = useAtomValue(onboardingActiveAtom);
  const [step, setStep] = useAtom(onboardingStepAtom);
  const setActive = useSetAtom(onboardingActiveAtom);
  const disabledItems = useAtomValue(menuDisabledItemsAtom);
  const videoMode = useAtomValue(videoModeAtom);
  const screenTimelapseEnabled = useAtomValue(screenTimelapseEnabledAtom);

  // Two-phase open: activation re-renders the rail in the same commit, but
  // the pin (force expanded/visible) lands through passive-effect state. Wait
  // two frames so measurement below reads the settled layout.
  const [pinned, setPinned] = useState(false);

  const proxyRef = useRef<HTMLDivElement>(null);
  const restoreFocusRef = useRef<Element | null>(null);
  const onEndRef = useRef(onEnd);
  onEndRef.current = onEnd;
  const stepRef = useRef(step);
  stepRef.current = step;
  const stepDefsRef = useRef<OnboardingStepDef[]>([]);
  const activeRef = useRef(active);
  activeRef.current = active;

  const enabled = active && !isMobileDevice && !isMobilePortrait;

  const endTour = (reason: OnboardingEndReason) => {
    if (!activeRef.current) return;
    activeRef.current = false;
    setStep(0);
    setActive(false);
    onEndRef.current?.(reason);
  };
  const endTourRef = useRef(endTour);
  endTourRef.current = endTour;

  // --- Shield: suspend every HID path at the router. Passive effect, so the
  // keyboard gate flip's releaseKeys() cleanup flushes held keys to the host
  // before the router closes (all destroys run before all creates).
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

  // --- Focus: capture the previous focus as soon as the tour activates (the
  // trigger control unmounts during the pin settle window), move it to the
  // tour's primary action once pinned, and restore it when the tour ends.
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
      const primary = document.querySelector('.ant-tour .ant-tour-next-btn');
      if (primary instanceof HTMLElement) primary.focus();
    });
    return () => cancelAnimationFrame(raf);
  }, [pinned]);

  const stepDefs = useMemo(
    () =>
      deriveOnboardingSteps({ disabledItems, videoMode, screenTimelapseEnabled, isMobilePortrait }),
    [disabledItems, videoMode, screenTimelapseEnabled, isMobilePortrait]
  );
  stepDefsRef.current = stepDefs;
  const clampedStep = Math.min(step, stepDefs.length - 1);
  const stepDef = stepDefs[clampedStep];

  // --- Keyboard: capture-phase tour navigation. The desktop keyboard handler
  // is already gated by keyboardInputEnabledAtom; stopImmediatePropagation is
  // belt and braces for any other document listener.
  useEffect(() => {
    if (!enabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      const key = event.key;
      if (key !== 'Escape' && key !== 'ArrowRight' && key !== 'ArrowLeft' && key !== 'Enter') {
        return;
      }
      // Panel buttons activate themselves natively on Enter/Space.
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

  // Re-measure when the viewport changes while pinned (the pinned rail is
  // otherwise static: dragging is disabled and auto-hide is suspended).
  const [measureTick, setMeasureTick] = useState(0);
  useEffect(() => {
    if (!pinned) return;
    const handleResize = () => setMeasureTick((tick) => tick + 1);
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [pinned]);

  if (!enabled) return null;

  // Measured during render: the rail is pinned and stable once pinned=true,
  // and rc-tour (a child) resolves and measures its target in layout effects,
  // which run after this commit's mutation phase has positioned the proxy.
  const groupRect = pinned && stepDef?.group ? measureMenuGroupRect(stepDef.group) : null;

  const tourSteps: TourProps['steps'] = stepDefs.map(
    (def): NonNullable<TourProps['steps']>[number] => {
      const navigation = {
        prevButtonProps: { children: t('onboarding.tour.prev') },
        nextButtonProps: {
          children: def.key === 'finish' ? t('onboarding.tour.end') : t('onboarding.tour.next')
        }
      };
      if (!def.group) {
        return {
          ...navigation,
          target: null,
          title:
            def.key === 'welcome' ? t('onboarding.welcome.title') : t('onboarding.finish.title'),
          description:
            def.key === 'welcome'
              ? t('onboarding.welcome.description')
              : t('onboarding.finish.description')
        };
      }
      const group = def.group;
      return {
        ...navigation,
        // Invariant: group steps render only while the proxy is positioned
        // (groupRect non-null); rc-tour normalizes a falsy element to null.
        target: () => proxyRef.current as HTMLElement,
        placement: 'bottom' as const,
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
      {pinned && groupRect && (
        <div
          // A fresh node per group (and per resize tick) forces rc-tour's
          // targetElement identity to change, so the mask hole and popover
          // re-measure on every step and after every viewport resize.
          key={`${stepDef?.group}:${measureTick}`}
          ref={proxyRef}
          aria-hidden="true"
          className="pointer-events-none fixed"
          style={{
            left: groupRect.left,
            top: groupRect.top,
            width: groupRect.width,
            height: groupRect.height,
            visibility: 'hidden'
          }}
        />
      )}
      <Tour
        open={pinned}
        current={clampedStep}
        onChange={(current) => setStep(current)}
        // rc-tour fires onClose inside onFinish; deferring the skip report to a
        // microtask lets the synchronous finish report win (endTour is
        // idempotent), so a Finish click never arrives as 'skip'.
        onClose={() => queueMicrotask(() => endTourRef.current('skip'))}
        onFinish={() => endTourRef.current('finish')}
        steps={tourSteps}
        zIndex={LAYER_ONBOARDING}
        mask={{ color: 'rgba(8,8,12,.62)' }}
        gap={{ offset: 6, radius: 6 }}
        disabledInteraction
        actionsRender={(originNode, { current, total }) => (
          <>
            {current < total - 1 && (
              <Button size="small" type="text" onClick={() => endTourRef.current('skip')}>
                {t('onboarding.skip')}
              </Button>
            )}
            {originNode}
          </>
        )}
      />
    </>
  );
};
