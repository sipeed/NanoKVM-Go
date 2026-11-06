import { useCallback, useEffect, useState } from 'react';

import {
  getBrowserLayoutViewportSize,
  getResponsiveDeviceState,
  ResponsiveDeviceState
} from '@/lib/mobile-layout.ts';

const initialState: ResponsiveDeviceState = {
  isTouchLikeDevice: false,
  isMobileDevice: false,
  isPortraitViewport: false,
  isLandscapeViewport: true,
  isMobilePortrait: false
};

function getState(): ResponsiveDeviceState {
  if (typeof window === 'undefined') {
    return initialState;
  }

  const viewport = getBrowserLayoutViewportSize();
  return getResponsiveDeviceState({
    width: viewport.width,
    height: viewport.height,
    hasTouch: navigator.maxTouchPoints > 0,
    coarsePointer: window.matchMedia('(pointer: coarse)').matches,
    userAgent: navigator.userAgent
  });
}

export function useResponsiveDevice() {
  const [state, setState] = useState(getState);

  const updateState = useCallback(() => {
    setState(getState());
  }, []);

  useEffect(() => {
    updateState();
    window.addEventListener('resize', updateState);
    window.addEventListener('orientationchange', updateState);

    return () => {
      window.removeEventListener('resize', updateState);
      window.removeEventListener('orientationchange', updateState);
    };
  }, [updateState]);

  return state;
}
