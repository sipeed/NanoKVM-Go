import { RefObject, useCallback, useEffect, useLayoutEffect, useState } from 'react';
import { useAtomValue } from 'jotai';

import { menuDisabledItemsAtom } from '@/jotai/settings.ts';

export interface MenuBounds {
  left: number;
  right: number;
  top: number;
  bottom: number;
  viewportWidth: number;
  isMeasured: boolean;
}

export function useMenuBounds(
  nodeRef: RefObject<HTMLDivElement | null>,
  isMenuExpanded: boolean,
  isActive = true
): MenuBounds {
  const menuDisabledItems = useAtomValue(menuDisabledItemsAtom);
  const [menuBounds, setMenuBounds] = useState<MenuBounds>({
    left: 0,
    right: 0,
    top: 0,
    bottom: 0,
    viewportWidth: 0,
    isMeasured: false
  });

  const handleResize = useCallback(() => {
    if (!nodeRef.current) return;

    const elementRect = nodeRef.current.getBoundingClientRect();
    const viewport = window.visualViewport;
    const viewportLeft = viewport?.offsetLeft ?? 0;
    const viewportTop = viewport?.offsetTop ?? 0;
    const viewportWidth = viewport?.width ?? window.innerWidth;
    const viewportHeight = viewport?.height ?? window.innerHeight;
    const elementLeft = (window.innerWidth - elementRect.width) / 2;
    const elementTop = 10;

    setMenuBounds({
      left: viewportLeft - elementLeft,
      top: viewportTop - elementTop,
      right: viewportLeft + viewportWidth - elementLeft - elementRect.width,
      bottom: viewportTop + viewportHeight - elementTop - elementRect.height,
      viewportWidth,
      isMeasured: true
    });
  }, [nodeRef]);

  useLayoutEffect(() => {
    if (!isActive) {
      setMenuBounds((bounds) => (bounds.isMeasured ? { ...bounds, isMeasured: false } : bounds));
      return;
    }

    handleResize();
    window.addEventListener('resize', handleResize);
    window.visualViewport?.addEventListener('resize', handleResize);
    window.visualViewport?.addEventListener('scroll', handleResize);
    const resizeObserver = new ResizeObserver(handleResize);
    if (nodeRef.current) resizeObserver.observe(nodeRef.current);

    return () => {
      window.removeEventListener('resize', handleResize);
      window.visualViewport?.removeEventListener('resize', handleResize);
      window.visualViewport?.removeEventListener('scroll', handleResize);
      resizeObserver.disconnect();
    };
  }, [handleResize, isActive, nodeRef]);

  // Update bounds when menu items or expansion state change
  useEffect(() => {
    if (!isActive) return;
    handleResize();
  }, [menuDisabledItems, isMenuExpanded, handleResize, isActive]);

  return menuBounds;
}
