import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { Divider } from 'antd';
import clsx from 'clsx';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { ChevronDownIcon, ChevronUpIcon, GripVerticalIcon } from 'lucide-react';
import Draggable, { DraggableData, DraggableEvent } from 'react-draggable';
import { useTranslation } from 'react-i18next';

import * as aiAPI from '@/api/ai.ts';
import * as vmAPI from '@/api/vm.ts';
import { LAYER_MENU_RAIL } from '@/lib/layers.ts';
import * as storage from '@/lib/localstorage.ts';
import { getBaseUrl } from '@/lib/service.ts';
import { onboardingActiveAtom } from '@/jotai/onboarding.ts';
import { videoModeAtom } from '@/jotai/screen.ts';
import {
  menuConfigInitializedAtom,
  menuDisabledItemsAtom,
  screenTimelapseEnabledAtom
} from '@/jotai/settings.ts';
import { useMenuBounds } from '@/hooks/useMenuBounds.ts';
import { useMenuVisibility } from '@/hooks/useMenuVisibility.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';
import { MobileMenuItemProvider } from '@/components/menu-item.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { FingerRobot } from './finger-robot';
import { Fullscreen } from './fullscreen';
import { Keyboard } from './keyboard';
import { Microphone } from './microphone';
import { Mouse } from './mouse';
import { Collapse, Expand } from './operations';
import { PictureInPicture } from './picture-in-picture';
import { Portal } from './portal';
import { Preview } from './preview';
import { Screen } from './screen';
import { Script } from './script';
import { Settings } from './settings';
import { ScreenTimelapse } from './settings/ai/screen-timelapse';
import { Storage } from './storage';
import { Terminal } from './terminal';
import { Volume } from './volume';
import { Wol } from './wol';

const screenTimelapseServiceID = 'screenTimelapse';
const mobileMenuPopoverClassName = 'nanokvm-mobile-menu-popover';
const mobileRailToggleButtonClass =
  'flex size-[30px] items-center justify-center bg-transparent text-neutral-300 transition-colors enabled:hover:bg-neutral-700/80 enabled:hover:text-white enabled:active:bg-neutral-600/70';
const mobileMenuMargin = 8;
const mobileMenuDefaultTop = 48;

type MenuDivider = 'desktop' | 'mobile';
type MenuPosition = { x: number; y: number };
type MenuDragBounds = { left: number; right: number; top: number; bottom: number };

const initialMobileMenuBounds: MenuDragBounds = {
  left: mobileMenuMargin,
  right: mobileMenuMargin,
  top: mobileMenuMargin,
  bottom: mobileMenuMargin
};

function clampToRange(value: number, min: number, max: number): number {
  if (min > max) return (min + max) / 2;
  return Math.min(max, Math.max(min, value));
}

function clampMenuPosition(position: MenuPosition, bounds: MenuDragBounds): MenuPosition {
  return {
    x: Math.min(bounds.right, Math.max(bounds.left, position.x)),
    y: Math.min(bounds.bottom, Math.max(bounds.top, position.y))
  };
}

function normalizeMenuBounds(bounds: MenuDragBounds): MenuDragBounds {
  if (bounds.left <= bounds.right) return bounds;

  const midpoint = (bounds.left + bounds.right) / 2;
  return { ...bounds, left: midpoint, right: midpoint };
}

export const Menu = () => {
  const { t } = useTranslation();
  const nodeRef = useRef<HTMLDivElement | null>(null);
  const mobileRailRef = useRef<HTMLDivElement | null>(null);
  const mobileRailPositionInitialized = useRef(false);

  const videoMode = useAtomValue(videoModeAtom);
  const menuDisabledItems = useAtomValue(menuDisabledItemsAtom);
  const setMenuDisabledItems = useSetAtom(menuDisabledItemsAtom);
  const [isInitialized, setIsInitialized] = useAtom(menuConfigInitializedAtom);
  const [isScreenTimelapseEnabled, setIsScreenTimelapseEnabled] = useAtom(
    screenTimelapseEnabledAtom
  );
  const [isMobileRailCollapsed, setIsMobileRailCollapsed] = useState(false);
  const [mobileRailPosition, setMobileRailPosition] = useState<MenuPosition>({
    x: mobileMenuMargin,
    y: mobileMenuDefaultTop
  });
  const [mobileMenuBounds, setMobileMenuBounds] = useState<MenuDragBounds>(initialMobileMenuBounds);
  const [desktopMenuOffset, setDesktopMenuOffset] = useState({ x: 0, y: 0 });
  const [dismissMenuKey, setDismissMenuKey] = useState(0);

  const { isMenuExpanded, isMenuHidden, handleHovered, handleMoved, setIsMenuExpanded } =
    useMenuVisibility();

  // While the onboarding tour runs the rail is pinned: useMenuVisibility
  // forces it expanded and suspends auto-hide; dragging and mobile-rail
  // collapse are disabled here so the spotlight anchors stay stable.
  const onboardingActive = useAtomValue(onboardingActiveAtom);
  // The tour never runs on mobile devices in v1, but keep the rail expanded if
  // the flag is ever set so spotlight anchors cannot collapse away.
  const isRailCollapsed = isMobileRailCollapsed && !onboardingActive;

  const responsiveDevice = useResponsiveDevice();
  const isMobileRailActive = responsiveDevice.isMobilePortrait;
  const isTouchLandscapeMenu =
    responsiveDevice.isTouchLikeDevice && responsiveDevice.isLandscapeViewport;
  const menuBounds = useMenuBounds(nodeRef, isMenuExpanded, !isMobileRailActive);
  const draggableMenuBounds = normalizeMenuBounds(menuBounds);

  const updateMobileMenuBounds = useCallback(() => {
    const rail = mobileRailRef.current;
    if (!rail) return;

    const railRect = rail.getBoundingClientRect();
    const viewport = window.visualViewport;
    const viewportLeft = viewport?.offsetLeft ?? 0;
    const viewportTop = viewport?.offsetTop ?? 0;
    const viewportWidth = viewport?.width ?? window.innerWidth;
    const viewportHeight = viewport?.height ?? window.innerHeight;
    const left = viewportLeft + mobileMenuMargin;
    const top = viewportTop + mobileMenuMargin;
    const bounds = {
      left,
      right: Math.max(left, viewportLeft + viewportWidth - railRect.width - mobileMenuMargin),
      top,
      bottom: Math.max(top, viewportTop + viewportHeight - railRect.height - mobileMenuMargin)
    };

    setMobileMenuBounds(bounds);
    setMobileRailPosition((position) => {
      if (!mobileRailPositionInitialized.current) {
        mobileRailPositionInitialized.current = true;
        return clampMenuPosition(
          { x: bounds.right, y: viewportTop + mobileMenuDefaultTop },
          bounds
        );
      }
      return clampMenuPosition(position, bounds);
    });
  }, []);

  useEffect(() => {
    if (!isMobileRailActive) return;

    updateMobileMenuBounds();
    window.addEventListener('resize', updateMobileMenuBounds);
    window.visualViewport?.addEventListener('resize', updateMobileMenuBounds);
    window.visualViewport?.addEventListener('scroll', updateMobileMenuBounds);
    const resizeObserver = new ResizeObserver(updateMobileMenuBounds);
    if (mobileRailRef.current) resizeObserver.observe(mobileRailRef.current);

    return () => {
      window.removeEventListener('resize', updateMobileMenuBounds);
      window.visualViewport?.removeEventListener('resize', updateMobileMenuBounds);
      window.visualViewport?.removeEventListener('scroll', updateMobileMenuBounds);
      resizeObserver.disconnect();
    };
  }, [isMobileRailActive, updateMobileMenuBounds]);

  useEffect(() => {
    if (isMobileRailActive || !menuBounds.isMeasured) return;

    setDesktopMenuOffset((offset) => ({
      x: clampToRange(offset.x, menuBounds.left, menuBounds.right),
      y: clampToRange(offset.y, menuBounds.top, menuBounds.bottom)
    }));
  }, [isMobileRailActive, menuBounds]);

  useEffect(() => {
    vmAPI
      .getMenuBarConfig()
      .then((rsp) => {
        if (rsp.code === 0) {
          setMenuDisabledItems(storage.normalizeMenuDisabledItems(rsp.data?.disabledItems));
          return;
        }

        // compatible with old versions
        const items = storage.normalizeMenuDisabledItems(storage.getMenuDisabledItems());
        setMenuDisabledItems(items);
      })
      .finally(() => {
        setIsInitialized(true);
      });

    return () => {
      // Restore the pre-lift local-state semantics: a remounted menu is not
      // initialized (and gates the onboarding trigger) until the config fetch
      // resolves again.
      setIsInitialized(false);
    };
  }, [setIsInitialized, setMenuDisabledItems]);

  useEffect(() => {
    const eventSource = new EventSource(`${getBaseUrl('http')}${aiAPI.aiServiceEventsPath}`, {
      withCredentials: true
    });

    const parseEventData = <T,>(event: MessageEvent<string>): T | null => {
      try {
        return JSON.parse(event.data) as T;
      } catch (error) {
        console.log(error);
        return null;
      }
    };

    const handleSnapshot = (event: MessageEvent<string>) => {
      const snapshot = parseEventData<aiAPI.AIServiceStateSnapshotEvent>(event);
      setIsScreenTimelapseEnabled(Boolean(snapshot?.services?.[screenTimelapseServiceID]?.enabled));
    };

    const handleServiceChanged = (event: MessageEvent<string>) => {
      const change = parseEventData<aiAPI.AIServiceStateChangedEvent>(event);
      if (change?.service === screenTimelapseServiceID) {
        setIsScreenTimelapseEnabled(Boolean(change.enabled));
      }
    };

    eventSource.addEventListener('snapshot', handleSnapshot);
    eventSource.addEventListener('service_changed', handleServiceChanged);
    eventSource.onerror = (error) => {
      console.log(error);
      setIsScreenTimelapseEnabled(false);
    };

    return () => {
      eventSource.close();
      // Restore the pre-lift local-state semantics: a remounted menu starts
      // without the timelapse item until the next service snapshot arrives.
      setIsScreenTimelapseEnabled(false);
    };
  }, [setIsScreenTimelapseEnabled]);

  function onDesktopDrag(_e: DraggableEvent, data: DraggableData) {
    setDesktopMenuOffset({ x: data.x, y: data.y });
  }

  function onDragStop(_e: DraggableEvent, data: DraggableData) {
    setDesktopMenuOffset({ x: data.x, y: data.y });
    if (data.x === 0 && data.y === 0) return;
    handleMoved();
  }

  function onMobileDrag(_e: DraggableEvent, data: DraggableData) {
    setMobileRailPosition({ x: data.x, y: data.y });
  }

  const dismissMenus = useCallback(() => {
    setDismissMenuKey((key) => key + 1);
  }, []);

  function onMenuDragStart() {
    dismissMenus();
  }

  function isEnabled(item: string) {
    return !menuDisabledItems.includes(item);
  }

  function renderDivider(variant: MenuDivider, key: string) {
    if (variant === 'mobile') {
      return <div key={key} className="my-1 h-px w-7 shrink-0 bg-neutral-600/60" />;
    }

    return <Divider key={key} type="vertical" />;
  }

  // Group wrappers carry the onboarding spotlight anchors. They are
  // display:contents (no layout box), so the tour engine measures the union
  // of each group's children rects instead of the wrapper itself. `key`
  // discriminates repeated groups (automation has two conditional blocks).
  function renderGroup(group: string, children: ReactNode, key: string = group) {
    return (
      <div key={`group-${key}`} data-menu-group={group} className="contents">
        {children}
      </div>
    );
  }

  function renderMenuItems(variant: MenuDivider, mobilePlacement?: 'left' | 'right'): ReactNode[] {
    const items: ReactNode[] = [];
    const isWebRTCVideoMode = videoMode === 'h264-webrtc' || videoMode === 'h265-webrtc';
    const showVolume = isEnabled('volume') && isWebRTCVideoMode;
    const showMicrophone = isEnabled('microphone') && isWebRTCVideoMode;
    const tooltipPlacement =
      variant === 'mobile' ? (mobilePlacement === 'right' ? 'left' : 'right') : 'bottom';

    items.push(
      renderGroup(
        'display',
        <>
          <Screen key="screen" />
          <Portal key="portal" tooltipPlacement={tooltipPlacement} />
          {isEnabled('pictureInPicture') && (
            <PictureInPicture key="picture-in-picture" tooltipPlacement={tooltipPlacement} />
          )}
        </>
      )
    );
    if (showVolume || showMicrophone) {
      items.push(
        renderGroup(
          'media',
          <>
            {showVolume && <Volume key="volume" />}
            {showMicrophone && <Microphone key="microphone" tooltipPlacement={tooltipPlacement} />}
          </>
        )
      );
      items.push(renderDivider(variant, 'divider-media'));
    }

    items.push(
      renderGroup(
        'input',
        <>
          <Keyboard key="keyboard" />
          <Mouse key="mouse" />
          <Preview key="preview" />
        </>
      )
    );
    items.push(renderDivider(variant, 'divider-input'));

    if (['image', 'script', 'terminal', 'wol'].some(isEnabled)) {
      items.push(
        renderGroup(
          'tools',
          <>
            {isEnabled('image') && <Storage key="image" />}
            {isEnabled('script') && <Script key="script" />}
            {isEnabled('terminal') && <Terminal key="terminal" />}
            {isEnabled('wol') && <Wol key="wol" />}
          </>
        )
      );
      items.push(renderDivider(variant, 'divider-tools'));
    }

    if (isScreenTimelapseEnabled && isEnabled('screenTimelapse')) {
      items.push(
        renderGroup(
          'automation',
          <ScreenTimelapse
            key="screen-timelapse"
            variant="menu"
            tooltipPlacement={tooltipPlacement}
          />,
          'automation-timelapse'
        )
      );
      items.push(renderDivider(variant, 'divider-timelapse'));
    }

    if (isEnabled('power')) {
      // The persisted menu-bar key remains "power"; only the product UI was renamed.
      items.push(renderGroup('automation', <FingerRobot key="power" />, 'automation-power'));
      items.push(renderDivider(variant, 'divider-finger-robot'));
    }

    items.push(
      renderGroup(
        'system',
        <>
          <Settings key="settings" tooltipPlacement={tooltipPlacement} />
          {variant === 'desktop' && isEnabled('fullscreen') && <Fullscreen key="fullscreen" />}
          {variant === 'desktop' && isEnabled('collapse') && (
            <Collapse key="collapse" toggleMenu={setIsMenuExpanded} />
          )}
        </>
      )
    );

    return items;
  }

  if (isMobileRailActive) {
    const viewport = window.visualViewport;
    const viewportCenter = (viewport?.offsetLeft ?? 0) + (viewport?.width ?? window.innerWidth) / 2;
    const mobileMenuMaxHeight = Math.max(
      1,
      (viewport?.height ?? window.innerHeight) - mobileMenuMargin * 2
    );
    const mobilePlacement = mobileRailPosition.x < viewportCenter ? 'left' : 'right';

    return (
      <Draggable
        nodeRef={mobileRailRef}
        bounds={mobileMenuBounds}
        handle="strong"
        position={mobileRailPosition}
        disabled={onboardingActive}
        onStart={onMenuDragStart}
        onDrag={onMobileDrag}
        onStop={onMobileDrag}
      >
        <div
          ref={mobileRailRef}
          className={clsx(
            'fixed left-0 top-0 transition-opacity duration-300',
            isInitialized ? 'opacity-100' : 'opacity-0'
          )}
          style={{ zIndex: LAYER_MENU_RAIL }}
        >
          <MobileMenuItemProvider
            mobilePlacement={mobilePlacement}
            mobilePopoverClassName={mobileMenuPopoverClassName}
            dismissMenuKey={dismissMenuKey}
            onRequestMobileMenuDismiss={dismissMenus}
          >
            {isRailCollapsed ? (
              <div className="flex flex-col items-center rounded-full bg-neutral-800/90 p-1 shadow-lg shadow-black/30 outline outline-1 outline-neutral-700/80 backdrop-blur">
                <strong>
                  <div className="flex size-[28px] cursor-move select-none items-center justify-center rounded-full text-neutral-500">
                    <GripVerticalIcon size={18} />
                  </div>
                </strong>
                <ButtonBase
                  className={clsx(mobileRailToggleButtonClass, 'rounded-full')}
                  onClick={() => setIsMobileRailCollapsed(false)}
                  aria-label={t('menu.expand')}
                >
                  <ChevronDownIcon size={18} aria-hidden="true" />
                </ButtonBase>
              </div>
            ) : (
              <div
                data-menu-rail="mobile"
                className="flex flex-col items-center overflow-y-auto rounded bg-neutral-800/90 px-1 py-1 shadow-lg shadow-black/30 outline outline-1 outline-neutral-700/80 backdrop-blur transition-all duration-200"
                style={{ maxHeight: mobileMenuMaxHeight }}
              >
                <strong>
                  <div className="flex size-[30px] cursor-move select-none items-center justify-center rounded text-neutral-500">
                    <GripVerticalIcon size={18} />
                  </div>
                </strong>

                {renderDivider('mobile', 'divider-handle')}
                {renderMenuItems('mobile', mobilePlacement)}

                {isEnabled('collapse') && (
                  <>
                    {renderDivider('mobile', 'divider-collapse')}
                    <ButtonBase
                      className={clsx(mobileRailToggleButtonClass, 'rounded')}
                      onClick={() => setIsMobileRailCollapsed(true)}
                      aria-label={t('menu.collapse')}
                    >
                      <ChevronUpIcon size={18} aria-hidden="true" />
                    </ButtonBase>
                  </>
                )}
              </div>
            )}
          </MobileMenuItemProvider>
        </div>
      </Draggable>
    );
  }

  return (
    <>
      <Draggable
        nodeRef={nodeRef}
        bounds={draggableMenuBounds}
        handle="strong"
        position={desktopMenuOffset}
        disabled={onboardingActive}
        onStart={onMenuDragStart}
        onDrag={onDesktopDrag}
        onStop={onDragStop}
      >
        <div
          ref={nodeRef}
          className={clsx(
            'pointer-events-none fixed left-1/2 top-[10px] w-max transition-opacity duration-300',
            isInitialized ? 'opacity-100' : 'opacity-0'
          )}
          style={{ zIndex: LAYER_MENU_RAIL }}
        >
          <div
            className="relative -translate-x-1/2"
            onMouseEnter={() => handleHovered(true)}
            onMouseLeave={() => handleHovered(false)}
            onFocus={() => handleHovered(true)}
            onBlur={() => handleHovered(false)}
          >
            {/* Trigger area for auto-show when hidden */}
            {isMenuExpanded && (
              <ButtonBase
                aria-label={t('menu.expand')}
                className={clsx(
                  'absolute z-10 transition-all duration-300',
                  isTouchLandscapeMenu
                    ? 'left-1/2 top-0 flex h-6 w-16 -translate-x-1/2 items-center justify-center rounded-b bg-neutral-800/80 text-neutral-300 shadow-lg shadow-black/30 backdrop-blur'
                    : '-top-[10px] left-0 h-[46px] w-full bg-transparent',
                  isMenuHidden ? 'pointer-events-auto opacity-100' : 'pointer-events-none opacity-0'
                )}
                onPointerDown={(event) => {
                  handleHovered(true);
                  event.currentTarget.setPointerCapture(event.pointerId);
                }}
                onPointerUp={(event) => {
                  handleHovered(false);
                  if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                    event.currentTarget.releasePointerCapture(event.pointerId);
                  }
                }}
                onPointerCancel={(event) => {
                  handleHovered(false);
                  if (event.currentTarget.hasPointerCapture(event.pointerId)) {
                    event.currentTarget.releasePointerCapture(event.pointerId);
                  }
                }}
              >
                {isTouchLandscapeMenu && <ChevronDownIcon size={16} aria-hidden="true" />}
              </ButtonBase>
            )}

            {/* Menubar */}
            <div className="sticky top-[10px] flex w-full justify-center">
              <div
                className={clsx(
                  'pointer-events-auto box-border flex h-[36px] w-max max-w-none items-center overflow-visible rounded bg-neutral-800/80 pl-1 pr-2 transition-all duration-300 [&>*]:shrink-0 [&_[data-menu-group]>*]:shrink-0',
                  isMenuExpanded ? 'flex' : 'hidden',
                  isMenuHidden ? '-translate-y-[110%] opacity-80' : 'translate-y-0 opacity-100'
                )}
              >
                <strong>
                  <div className="flex h-[30px] cursor-move select-none items-center justify-center pl-1 text-neutral-500">
                    <GripVerticalIcon size={18} />
                  </div>
                </strong>
                <Divider type="vertical" />

                <MobileMenuItemProvider dismissMenuKey={dismissMenuKey}>
                  {renderMenuItems('desktop')}
                </MobileMenuItemProvider>
              </div>
            </div>

            {/* Menubar expand button */}
            {!isMenuExpanded && (
              <div className="pointer-events-auto inline-flex w-max">
                <Expand toggleMenu={setIsMenuExpanded} />
              </div>
            )}
          </div>
        </div>
      </Draggable>
    </>
  );
};
