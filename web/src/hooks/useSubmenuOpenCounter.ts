import { useCallback, useEffect, useRef } from 'react';
import { useSetAtom } from 'jotai';

import { submenuOpenCountAtom } from '@/jotai/settings.ts';

export interface SubmenuOpenCounter {
  setOpen: (open: boolean) => void;
}

/**
 * Keeps one shared visibility contribution for an independently controlled
 * menu popover. Ant Design may repeat open-change callbacks, so transitions
 * are tracked locally and cleanup balances any contribution left at unmount.
 */
export function useSubmenuOpenCounter(initiallyOpen = false): SubmenuOpenCounter {
  const setSubmenuOpenCount = useSetAtom(submenuOpenCountAtom);
  const initiallyOpenRef = useRef(initiallyOpen);
  const isOpenRef = useRef(initiallyOpen);
  const hasOpenCountRef = useRef(false);

  const setOpen = useCallback(
    (open: boolean) => {
      if (isOpenRef.current === open) {
        return;
      }

      isOpenRef.current = open;
      if (open) {
        if (hasOpenCountRef.current) {
          return;
        }

        hasOpenCountRef.current = true;
        setSubmenuOpenCount((count) => count + 1);
        return;
      }

      if (!hasOpenCountRef.current) {
        return;
      }

      hasOpenCountRef.current = false;
      setSubmenuOpenCount((count) => Math.max(0, count - 1));
    },
    [setSubmenuOpenCount]
  );

  useEffect(() => {
    if (initiallyOpenRef.current && isOpenRef.current && !hasOpenCountRef.current) {
      hasOpenCountRef.current = true;
      setSubmenuOpenCount((count) => count + 1);
    }

    return () => {
      if (!hasOpenCountRef.current) {
        return;
      }

      hasOpenCountRef.current = false;
      setSubmenuOpenCount((count) => Math.max(0, count - 1));
    };
  }, [setSubmenuOpenCount]);

  return { setOpen };
}
