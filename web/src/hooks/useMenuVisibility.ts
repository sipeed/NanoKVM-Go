import { useCallback, useEffect, useRef, useState } from 'react';
import { useAtom, useAtomValue } from 'jotai';

import { getMenuDisplayMode } from '@/lib/localstorage.ts';
import { onboardingActiveAtom } from '@/jotai/onboarding.ts';
import { menuDisplayModeAtom, submenuOpenCountAtom } from '@/jotai/settings.ts';

const HIDE_TIMEOUT = 5000;

export interface MenuVisibilityState {
  isMenuExpanded: boolean;
  isMenuHidden: boolean;
  isMenuMoved: boolean;
  handleHovered: (hovered: boolean) => void;
  handleMoved: () => void;
  setIsMenuExpanded: (expanded: boolean) => void;
}

export function useMenuVisibility(): MenuVisibilityState {
  const [menuDisplayMode, setMenuDisplayMode] = useAtom(menuDisplayModeAtom);
  const submenuOpenCount = useAtomValue(submenuOpenCountAtom);
  const onboardingActive = useAtomValue(onboardingActiveAtom);

  const [isMenuExpanded, setIsMenuExpanded] = useState(true);
  const [isMenuMoved, setIsMenuMoved] = useState(false);
  const [isMenuHovered, setIsMenuHovered] = useState(false);
  const [isMenuHidden, setIsMenuHidden] = useState(false);

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  // Remembers the pre-tour expanded state so deactivation restores it exactly.
  const preTourExpandedRef = useRef<boolean | null>(null);
  const latestStateRef = useRef({
    menuDisplayMode,
    submenuOpenCount,
    isMenuExpanded,
    isMenuMoved,
    onboardingActive
  });
  latestStateRef.current = {
    menuDisplayMode,
    submenuOpenCount,
    isMenuExpanded,
    isMenuMoved,
    onboardingActive
  };

  const startCountdown = useCallback(() => {
    if (
      menuDisplayMode !== 'auto' ||
      submenuOpenCount > 0 ||
      !isMenuExpanded ||
      isMenuMoved ||
      onboardingActive
    ) {
      return;
    }

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      const state = latestStateRef.current;
      if (
        state.menuDisplayMode === 'auto' &&
        state.submenuOpenCount === 0 &&
        state.isMenuExpanded &&
        !state.isMenuMoved &&
        !state.onboardingActive
      ) {
        setIsMenuHidden(true);
      }
    }, HIDE_TIMEOUT);
  }, [menuDisplayMode, submenuOpenCount, isMenuExpanded, isMenuMoved, onboardingActive]);

  const stopCountdown = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
  }, []);

  // Initialize menu settings
  useEffect(() => {
    const displayMode = getMenuDisplayMode();
    setMenuDisplayMode(displayMode);

    if (displayMode === 'off') {
      setIsMenuExpanded(false);
    }

    return () => {
      stopCountdown();
    };
  }, []);

  // Handle display mode changes
  useEffect(() => {
    setIsMenuHidden(false);

    if (menuDisplayMode === 'auto') {
      startCountdown();
    } else {
      stopCountdown();
    }
  }, [menuDisplayMode, startCountdown, stopCountdown]);

  // Pin the rail while the onboarding tour runs: force expanded and visible,
  // suspend the auto-hide countdown; on deactivation restore the exact
  // pre-tour expanded state. The countdown itself resumes through the
  // startCountdown identity change in the display-mode effect above.
  useEffect(() => {
    if (onboardingActive) {
      if (preTourExpandedRef.current === null) {
        preTourExpandedRef.current = latestStateRef.current.isMenuExpanded;
      }
      stopCountdown();
      setIsMenuHidden(false);
      setIsMenuExpanded(true);
      return;
    }

    if (preTourExpandedRef.current !== null) {
      const restore = preTourExpandedRef.current;
      preTourExpandedRef.current = null;
      setIsMenuExpanded(restore);
    }
  }, [onboardingActive, stopCountdown]);

  // Handle hover and submenu state
  useEffect(() => {
    if (submenuOpenCount > 0) {
      stopCountdown();
      setIsMenuHidden(false);
    } else if (!isMenuHovered) {
      startCountdown();
    } else {
      stopCountdown();
    }
  }, [submenuOpenCount, isMenuHovered, startCountdown, stopCountdown]);

  // Handle hover state
  const handleHovered = useCallback((hovered: boolean) => {
    setIsMenuHovered(hovered);
    if (hovered) {
      setIsMenuHidden(false);
    }
  }, []);

  const handleMoved = useCallback(() => {
    if (!isMenuMoved) {
      setIsMenuMoved(true);
    }
  }, [isMenuMoved]);

  return {
    isMenuExpanded,
    isMenuHidden,
    isMenuMoved,
    handleHovered,
    handleMoved,
    setIsMenuExpanded
  };
}
