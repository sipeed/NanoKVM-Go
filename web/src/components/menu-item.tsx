import {
  cloneElement,
  useCallback,
  useContext,
  useEffect,
  useId,
  useMemo,
  useRef,
  useState,
  type ReactElement,
  type KeyboardEvent as ReactKeyboardEvent,
  type MouseEvent as ReactMouseEvent,
  type ReactNode
} from 'react';
import { Popover, Tooltip, type PopoverProps } from 'antd';
import clsx from 'clsx';
import { useSetAtom } from 'jotai';
import { ChevronLeftIcon, XIcon } from 'lucide-react';
import { createPortal } from 'react-dom';
import { useTranslation } from 'react-i18next';
import { useMediaQuery } from 'react-responsive';

import {
  LAYER_DESKTOP_MENU_POPOVER,
  LAYER_MOBILE_MENU_POPOVER
} from '@/lib/layers.ts';
import { submenuOpenCountAtom } from '@/jotai/settings.ts';
import { useSubmenuOpenCounter } from '@/hooks/useSubmenuOpenCounter.ts';

import {
  MobileMenuItemContext,
  MobileMenuNavigationContext,
  type MobileMenuItemContextValue,
  type MobileMenuNavigationContextValue,
  type MobileMenuNavigationEntry
} from './mobile-menu-context.ts';
import { ButtonBase } from './ui/button-base.tsx';

// Canonical values live in lib/layers.ts; re-exported here for the menu popover
// contract (desktop: above the video-status host 998, below the menu rail 1000;
// mobile: above the rail itself so panels stay interactive).
export const desktopMenuPopoverZIndex = LAYER_DESKTOP_MENU_POPOVER;
export const mobileMenuPopoverZIndex = LAYER_MOBILE_MENU_POPOVER;

export const MobileMenuItemProvider = ({
  children,
  mobilePlacement,
  mobilePopoverClassName,
  dismissMenuKey,
  onRequestMobileMenuDismiss
}: MobileMenuItemContextValue & { children: ReactNode }) => {
  return (
    <MobileMenuItemContext.Provider
      value={{
        mobilePlacement,
        mobilePopoverClassName,
        dismissMenuKey,
        onRequestMobileMenuDismiss
      }}
    >
      {children}
    </MobileMenuItemContext.Provider>
  );
};

type MenuItemProps = {
  title: string;
  icon: ReactNode;
  content: ReactNode;
  className?: string;
  disabled?: boolean;
  fresh?: boolean;
  mobilePlacement?: 'left' | 'right';
  mobilePopoverClassName?: string;
  onOpenChange?: (open: boolean) => void;
};

type MobileMenuPanelProps = {
  content: ReactNode;
  open: boolean;
  onClose: () => void;
};

export const MobileMenuPanel = ({ content, open, onClose }: MobileMenuPanelProps) => {
  const { t } = useTranslation();
  const [activePath, setActivePath] = useState<MobileMenuNavigationEntry[]>([]);
  const activePathRef = useRef<MobileMenuNavigationEntry[]>([]);
  const wasOpenRef = useRef(open);
  const [layerHost, setLayerHost] = useState<HTMLDivElement | null>(null);

  const resetPath = useCallback((notify = false) => {
    const previousPath = activePathRef.current;
    if (notify) {
      [...previousPath].reverse().forEach((entry) => entry.onDismiss?.());
    }

    activePathRef.current = [];
    setActivePath([]);
  }, []);

  useEffect(() => {
    if (open && !wasOpenRef.current) {
      resetPath();
    } else if (!open && wasOpenRef.current) {
      resetPath(true);
    }

    wasOpenRef.current = open;
  }, [open, resetPath]);

  const push = useCallback((entry: MobileMenuNavigationEntry) => {
    const currentPath = activePathRef.current;
    const existingIndex = currentPath.findIndex((item) => item.id === entry.id);
    const removedEntries =
      existingIndex === -1 ? [] : currentPath.slice(existingIndex + 1).reverse();
    removedEntries.forEach((item) => item.onDismiss?.());

    const nextPath =
      existingIndex === -1 ? [...currentPath, entry] : currentPath.slice(0, existingIndex + 1);
    activePathRef.current = nextPath;
    setActivePath(nextPath);
  }, []);

  const pop = useCallback(() => {
    const currentPath = activePathRef.current;
    const activeEntry = currentPath[currentPath.length - 1];
    if (!activeEntry || activeEntry.canLeave?.() === false) {
      return;
    }

    activeEntry.onDismiss?.();
    const nextPath = currentPath.slice(0, -1);
    activePathRef.current = nextPath;
    setActivePath(nextPath);
  }, []);

  const close = useCallback(() => {
    const activeEntry = activePathRef.current[activePathRef.current.length - 1];
    if (activeEntry?.canLeave?.() === false) {
      return;
    }

    resetPath(true);
    onClose();
  }, [onClose, resetPath]);

  useEffect(() => {
    if (!open) {
      return;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') {
        return;
      }

      event.preventDefault();
      close();
    };

    window.addEventListener('keydown', handleKeyDown, true);
    return () => window.removeEventListener('keydown', handleKeyDown, true);
  }, [close, open]);

  const setLayerHostRef = useCallback((node: HTMLDivElement | null) => {
    setLayerHost(node);
  }, []);

  const navigation = useMemo<MobileMenuNavigationContextValue>(
    () => ({ activePath, layerHost, push, pop, close }),
    [activePath, close, layerHost, pop, push]
  );
  const activeEntry = activePath[activePath.length - 1];

  return (
    <MobileMenuNavigationContext.Provider value={navigation}>
      <div className="nanokvm-mobile-menu-panel">
        {activeEntry && (
          <div className="flex h-[32px] min-w-0 items-center border-b border-neutral-700/80 px-1">
            <ButtonBase
              className="flex size-[30px] shrink-0 items-center justify-center rounded text-neutral-300 transition-colors hover:bg-neutral-700/80 hover:text-white active:bg-neutral-600/70"
              onClick={pop}
              aria-label={t('settings.back')}
            >
              <ChevronLeftIcon size={18} aria-hidden="true" />
            </ButtonBase>
            <span className="min-w-0 flex-1 truncate px-2 text-sm text-neutral-100">
              {activeEntry.title}
            </span>
            <ButtonBase
              className="flex size-[30px] shrink-0 items-center justify-center rounded text-neutral-300 transition-colors hover:bg-neutral-700/80 hover:text-white active:bg-neutral-600/70"
              onClick={close}
              aria-label={t('settings.close')}
            >
              <XIcon size={17} aria-hidden="true" />
            </ButtonBase>
          </div>
        )}
        <div className="nanokvm-mobile-menu-panel__body">
          <div
            className={clsx(
              'nanokvm-mobile-menu-panel__root',
              activeEntry && 'pointer-events-none invisible'
            )}
            aria-hidden={activeEntry ? true : undefined}
          >
            {content}
          </div>
          <div ref={setLayerHostRef} className="contents" />
        </div>
      </div>
    </MobileMenuNavigationContext.Provider>
  );
};

type MenuSubmenuTriggerProps = {
  onClick?: (event: ReactMouseEvent<HTMLElement>) => void;
  onKeyDown?: (event: ReactKeyboardEvent<HTMLElement>) => void;
  role?: string;
  tabIndex?: number;
  'aria-expanded'?: boolean;
  'aria-haspopup'?: boolean | 'dialog';
};

type MenuSubmenuProps = {
  title: string;
  content: ReactNode;
  children: ReactElement<MenuSubmenuTriggerProps>;
  popoverProps?: Omit<PopoverProps, 'children' | 'content'>;
  onBeforeLeave?: () => boolean;
};

export const MenuSubmenu = ({
  title,
  content,
  children,
  popoverProps,
  onBeforeLeave
}: MenuSubmenuProps) => {
  const navigation = useContext(MobileMenuNavigationContext);
  const desktopPopoverOpen = popoverProps?.open ?? popoverProps?.visible;
  const desktopPopoverDefaultOpen = popoverProps?.defaultOpen ?? popoverProps?.defaultVisible;
  const { setOpen: setDesktopPopoverOpen } = useSubmenuOpenCounter(
    !navigation && desktopPopoverOpen === undefined && Boolean(desktopPopoverDefaultOpen)
  );
  const submenuID = useId();
  const onOpenChangeRef = useRef(popoverProps?.onOpenChange);
  const onBeforeLeaveRef = useRef(onBeforeLeave);

  useEffect(() => {
    onOpenChangeRef.current = popoverProps?.onOpenChange;
  }, [popoverProps?.onOpenChange]);

  useEffect(() => {
    onBeforeLeaveRef.current = onBeforeLeave;
  }, [onBeforeLeave]);

  useEffect(() => {
    if (navigation) {
      setDesktopPopoverOpen(false);
      return;
    }

    if (typeof desktopPopoverOpen === 'boolean') {
      setDesktopPopoverOpen(desktopPopoverOpen);
    }
  }, [desktopPopoverOpen, navigation, setDesktopPopoverOpen]);

  const canLeave = useCallback(() => onBeforeLeaveRef.current?.() !== false, []);
  const dismiss = useCallback(() => {
    onOpenChangeRef.current?.(false);
  }, []);

  const handleDesktopOpenChange = useCallback(
    (open: boolean) => {
      if (desktopPopoverOpen === undefined) {
        setDesktopPopoverOpen(open);
      }

      onOpenChangeRef.current?.(open);
    },
    [desktopPopoverOpen, setDesktopPopoverOpen]
  );

  const openSubmenu = useCallback(() => {
    onOpenChangeRef.current?.(true);
    navigation?.push({ id: submenuID, title, canLeave, onDismiss: dismiss });
  }, [canLeave, dismiss, navigation, submenuID, title]);

  if (!navigation) {
    return (
      <Popover content={content} {...popoverProps} onOpenChange={handleDesktopOpenChange}>
        {children}
      </Popover>
    );
  }

  const activeEntry = navigation.activePath[navigation.activePath.length - 1];
  const isCurrent = activeEntry?.id === submenuID;
  const isInPath = navigation.activePath.some((entry) => entry.id === submenuID);

  const trigger = cloneElement(children, {
    role: children.props.role ?? 'button',
    tabIndex: children.props.tabIndex ?? 0,
    'aria-haspopup': 'dialog',
    'aria-expanded': isCurrent,
    onClick: (event: ReactMouseEvent<HTMLElement>) => {
      children.props.onClick?.(event);
      if (!event.defaultPrevented) {
        openSubmenu();
      }
    },
    onKeyDown: (event: ReactKeyboardEvent<HTMLElement>) => {
      children.props.onKeyDown?.(event);
      if (event.defaultPrevented || (event.key !== 'Enter' && event.key !== ' ')) {
        return;
      }

      event.preventDefault();
      openSubmenu();
    }
  });

  return (
    <>
      {trigger}
      {isInPath && navigation.layerHost
        ? createPortal(
            <div
              className={clsx(
                'nanokvm-mobile-menu-panel__layer',
                !isCurrent && 'pointer-events-none invisible'
              )}
              aria-hidden={!isCurrent}
            >
              {content}
            </div>,
            navigation.layerHost
          )
        : null}
    </>
  );
};

export const MenuItem = ({
  title,
  icon,
  content,
  className,
  disabled = false,
  fresh,
  mobilePlacement,
  mobilePopoverClassName,
  onOpenChange
}: MenuItemProps) => {
  const isBigScreen = useMediaQuery({ minWidth: 640 });
  const setSubmenuOpenCount = useSetAtom(submenuOpenCountAtom);
  const mobileMenuItemContext = useContext(MobileMenuItemContext);

  const effectiveMobilePlacement = mobilePlacement ?? mobileMenuItemContext.mobilePlacement;
  const effectiveMobilePopoverClassName =
    mobilePopoverClassName ?? mobileMenuItemContext.mobilePopoverClassName;
  const dismissMenuKey = mobileMenuItemContext.dismissMenuKey;

  const [isPopoverOpen, setIsPopoverOpen] = useState(false);
  const [isTooltipOpen, setIsTooltipOpen] = useState(false);
  const previousDismissMenuKey = useRef(dismissMenuKey);

  const closePopover = useCallback(() => {
    if (!isPopoverOpen) {
      return;
    }

    setIsTooltipOpen(false);
    setIsPopoverOpen(false);
    setSubmenuOpenCount((count) => Math.max(0, count - 1));
    onOpenChange?.(false);
  }, [isPopoverOpen, onOpenChange, setSubmenuOpenCount]);

  useEffect(() => {
    if (!disabled) {
      return;
    }

    closePopover();
  }, [closePopover, disabled]);

  useEffect(() => {
    if (dismissMenuKey === previousDismissMenuKey.current) {
      return;
    }

    previousDismissMenuKey.current = dismissMenuKey;
    closePopover();
  }, [closePopover, dismissMenuKey]);

  function togglePopover(open: boolean) {
    setIsTooltipOpen(false);
    setIsPopoverOpen(open);

    // Update global submenu count
    setSubmenuOpenCount((count) => (open ? count + 1 : Math.max(0, count - 1)));

    if (onOpenChange) {
      onOpenChange(open);
    }
  }

  function toggleTooltip(open: boolean) {
    if (isPopoverOpen) {
      return;
    }
    setIsTooltipOpen(open);
  }

  const popoverPlacement = effectiveMobilePlacement
    ? effectiveMobilePlacement === 'right'
      ? 'leftTop'
      : 'rightTop'
    : isBigScreen
      ? 'bottomLeft'
      : 'bottom';

  const tooltipPlacement = effectiveMobilePlacement
    ? effectiveMobilePlacement === 'right'
      ? 'left'
      : 'right'
    : 'bottom';

  const popoverContent = effectiveMobilePlacement ? (
    <MobileMenuPanel content={content} open={isPopoverOpen} onClose={closePopover} />
  ) : (
    content
  );

  const trigger = (
    <Tooltip
      title={title}
      mouseEnterDelay={0.6}
      placement={tooltipPlacement}
      overlayClassName={effectiveMobilePopoverClassName}
      open={disabled ? undefined : isTooltipOpen}
      onOpenChange={disabled ? undefined : toggleTooltip}
    >
      <div
        aria-disabled={disabled}
        className={clsx(
          'flex h-[30px] w-[30px] items-center justify-center rounded text-neutral-300',
          disabled
            ? 'cursor-not-allowed text-neutral-600 opacity-45'
            : 'cursor-pointer hover:bg-neutral-700/80 hover:text-white',
          className
        )}
      >
        {icon}
      </div>
    </Tooltip>
  );

  if (disabled) {
    return trigger;
  }

  return (
    <Popover
      content={popoverContent}
      arrow={false}
      trigger="click"
      zIndex={effectiveMobilePlacement ? mobileMenuPopoverZIndex : desktopMenuPopoverZIndex}
      placement={popoverPlacement}
      overlayClassName={effectiveMobilePopoverClassName}
      open={isPopoverOpen}
      onOpenChange={togglePopover}
      fresh={!!fresh}
    >
      {trigger}
    </Popover>
  );
};
