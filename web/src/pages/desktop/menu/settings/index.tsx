import { useEffect, useState, type ReactNode } from 'react';
import { Badge, Modal, Tooltip } from 'antd';
import clsx from 'clsx';
import { useAtom, useSetAtom, useStore } from 'jotai';
import {
  ArrowLeftIcon,
  BadgeInfoIcon,
  BotIcon,
  ChevronRightIcon,
  CircleArrowUpIcon,
  MonitorIcon,
  NetworkIcon,
  PackageIcon,
  PaletteIcon,
  SettingsIcon,
  SmartphoneIcon,
  UserRoundIcon,
  XIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/application.ts';
import * as ls from '@/lib/localstorage.ts';
import { isMockedDevelopment } from '@/lib/runtime-mode.ts';
import { isNewerVersion } from '@/lib/version.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import {
  isSettingsDetailOpenAtom,
  isSettingsOpenAtom,
  settingTabAtom,
  submenuOpenCountAtom,
  type SettingTab
} from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';
import { ScrollArea } from '@/components/ui/scroll-area.tsx';

import { About } from './about';
import { Account } from './account';
import { AI } from './ai';
import { Appearance } from './appearance';
import { Apps } from './apps';
import { Device } from './device';
import { RemoteNetwork } from './remote-network';
import { Screen } from './screen';
import { Update } from './update';

const mobileIconButtonClass =
  'flex size-9 items-center justify-center rounded bg-transparent text-neutral-300 transition-colors enabled:hover:bg-neutral-700 enabled:hover:text-white enabled:active:bg-neutral-600/70';

type SettingsProps = {
  tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
};

export const Settings = ({ tooltipPlacement = 'bottom' }: SettingsProps) => {
  const { t } = useTranslation();
  const responsiveDevice = useResponsiveDevice();
  const store = useStore();

  const [isSettingsOpen, setIsSettingsOpen] = useAtom(isSettingsOpenAtom);
  const [isMobileDetailOpen, setIsMobileDetailOpen] = useAtom(isSettingsDetailOpenAtom);
  const [settingTab, setSettingTab] = useAtom(settingTabAtom);
  const setIsKeyboardEnable = useSetAtom(isKeyboardEnableAtom);
  const setSubmenuOpenCount = useSetAtom(submenuOpenCountAtom);

  const [isLocked, setIsLocked] = useState(false);

  const [isUpdateAvailable, setIsUpdateAvailable] = useState(false);
  const tabs: Array<{ id: SettingTab; icon: ReactNode; component: ReactNode }> = [
    { id: 'about', icon: <BadgeInfoIcon size={16} />, component: <About /> },
    { id: 'appearance', icon: <PaletteIcon size={16} />, component: <Appearance /> },
    { id: 'screen', icon: <MonitorIcon size={16} />, component: <Screen /> },
    { id: 'device', icon: <SmartphoneIcon size={16} />, component: <Device /> },
    { id: 'ai', icon: <BotIcon size={16} />, component: <AI /> },
    { id: 'apps', icon: <PackageIcon size={16} />, component: <Apps /> },
    {
      id: 'remoteNetwork',
      icon: <NetworkIcon size={16} />,
      component: <RemoteNetwork setIsLocked={setIsLocked} />
    },
    {
      id: 'update',
      icon: <CircleArrowUpIcon size={16} />,
      component: <Update setIsLocked={setIsLocked} />
    },
    { id: 'account', icon: <UserRoundIcon size={16} />, component: <Account /> }
  ];

  useEffect(() => {
    if (import.meta.env.DEV && isMockedDevelopment(import.meta.env.MODE, import.meta.env.DEV)) {
      let cancelled = false;
      // The bootstrap pre-seeds Jotai before React renders. Reapply against the
      // mounted store as well because Settings is lazy-loaded with the menu tree.
      void import('@/mocks/target.ts').then(({ applyMockTarget }) => {
        if (!cancelled) applyMockTarget(window.location.hash, store);
      });
      return () => {
        cancelled = true;
      };
    }
  }, [store]);

  useEffect(() => {
    const skip = ls.getSkipUpdate();
    if (!skip) {
      checkForUpdates();
    }
  }, []);

  function checkForUpdates() {
    api.getVersion().then((rsp: any) => {
      if (rsp.code !== 0) {
        return;
      }

      if (isNewerVersion(rsp.data.latest, rsp.data.current)) {
        setIsUpdateAvailable(true);
      }
    });
  }

  function changeTab(tab: SettingTab) {
    if (isLocked) {
      return;
    }

    setSettingTab(tab);

    if (isUpdateAvailable && tab === 'update') {
      setIsUpdateAvailable(false);
      ls.setSkipUpdate(true);
    }
  }

  function openMobileDetail(tab: SettingTab) {
    changeTab(tab);

    if (!isLocked) {
      setIsMobileDetailOpen(true);
    }
  }

  function openModal() {
    setIsSettingsOpen(true);
    setIsMobileDetailOpen(false);
    setIsKeyboardEnable(false);
    setSubmenuOpenCount((count) => count + 1);
  }

  function closeModal() {
    if (isLocked) {
      return;
    }

    setIsKeyboardEnable(true);
    setIsSettingsOpen(false);
    setIsMobileDetailOpen(false);
    setSettingTab('about');
    setSubmenuOpenCount((count) => Math.max(0, count - 1));
  }

  const activeTab = tabs.find((tab) => tab.id === settingTab) ?? tabs[0];
  const isMobileSettingsLayout = responsiveDevice.isMobilePortrait;

  const renderTabTitle = (tab: (typeof tabs)[number], className?: string) => {
    if (isUpdateAvailable && tab.id === 'update') {
      return (
        <Badge className="max-w-full" dot color="blue" offset={[6, 3]}>
          <span className={className}>{t(`settings.${tab.id}.title`)}</span>
        </Badge>
      );
    }

    return <span className={className}>{t(`settings.${tab.id}.title`)}</span>;
  };

  const renderDesktopModal = () => (
    <Modal
      open={isSettingsOpen}
      width={'80%'}
      centered={true}
      footer={null}
      destroyOnHidden={true}
      onCancel={closeModal}
      style={{ maxWidth: '1080px' }}
      styles={{ content: { padding: 0 } }}
    >
      <div className="flex h-[80vh] max-h-[700px] rounded-lg outline outline-1 outline-neutral-700">
        <div className="flex h-full min-h-0 max-w-[260px] flex-col space-y-0.5 rounded-l-lg bg-neutral-800 px-1 sm:w-1/5 md:w-1/4 md:px-2">
          <div className="hidden shrink-0 px-3 pt-10 text-xl font-semibold sm:block">{t('settings.title')}</div>
          <div className="h-10 shrink-0 sm:h-5" />
          <ScrollArea className="min-h-0 flex-1">
            <div className="space-y-0.5">
              {tabs.map((tab) => (
                <div
                  key={tab.id}
                  className={clsx(
                    'flex cursor-pointer select-none items-center space-x-2 rounded-lg p-2 sm:px-3',
                    tab.id === settingTab ? 'bg-neutral-700/70' : 'hover:bg-neutral-700'
                  )}
                  onClick={() => changeTab(tab.id)}
                >
                  <div className="h-[16px] w-[16px]">{tab.icon}</div>
                  {renderTabTitle(tab, 'hidden truncate text-sm sm:block')}
                </div>
              ))}
            </div>
          </ScrollArea>
        </div>

        <ScrollArea className="h-full w-full rounded-r-lg bg-neutral-900/50 px-3 md:px-5">
          <div className="flex h-full w-full justify-center">
            <div className="w-full max-w-[600px] pb-10 pt-14">
              <>{activeTab.component}</>
            </div>
          </div>
        </ScrollArea>
      </div>
    </Modal>
  );

  const renderMobileModal = () => (
    <Modal
      open={isSettingsOpen}
      width="100%"
      centered={false}
      footer={null}
      closable={false}
      destroyOnHidden={true}
      onCancel={closeModal}
      style={{ top: 8, maxWidth: '100vw', padding: '0 8px' }}
      styles={{
        content: { padding: 0, overflow: 'hidden' },
        body: { height: 'calc(100dvh - 16px)' }
      }}
    >
      <div className="flex h-full min-h-0 flex-col overflow-hidden rounded-lg bg-neutral-900 outline outline-1 outline-neutral-700">
        <div className="flex h-12 shrink-0 items-center justify-between border-b border-neutral-700/80 bg-neutral-800 px-3">
          {isMobileDetailOpen ? (
            <ButtonBase
              className={mobileIconButtonClass}
              onClick={() => setIsMobileDetailOpen(false)}
              disabled={isLocked}
              aria-label={t('settings.back')}
            >
              <ArrowLeftIcon size={19} aria-hidden="true" />
            </ButtonBase>
          ) : (
            <div className="size-9" />
          )}

          <div className="min-w-0 truncate px-2 text-base font-medium text-neutral-100">
            {isMobileDetailOpen ? t(`settings.${activeTab.id}.title`) : t('settings.title')}
          </div>

          <ButtonBase
            className={mobileIconButtonClass}
            onClick={closeModal}
            disabled={isLocked}
            aria-label={t('settings.close')}
          >
            <XIcon size={19} aria-hidden="true" />
          </ButtonBase>
        </div>

        {isMobileDetailOpen ? (
          <ScrollArea className="min-h-0 flex-1 bg-neutral-900 px-4">
            <div className="mx-auto w-full max-w-[620px] pb-10 pt-5">
              <>{activeTab.component}</>
            </div>
          </ScrollArea>
        ) : (
          <ScrollArea className="min-h-0 flex-1 bg-neutral-900 px-2 py-2">
            <div className="space-y-1 pb-3">
              {tabs.map((tab) => (
                <ButtonBase
                  key={tab.id}
                  aria-current={tab.id === settingTab ? 'page' : undefined}
                  className={clsx(
                    'flex h-12 w-full items-center rounded px-3 text-left transition-colors enabled:active:bg-neutral-700/70 enabled:active:text-white',
                    tab.id === settingTab
                      ? 'bg-neutral-800 text-white'
                      : 'bg-transparent text-neutral-200 enabled:hover:bg-neutral-800/80'
                  )}
                  onClick={() => openMobileDetail(tab.id)}
                  disabled={isLocked}
                >
                  <span
                    aria-hidden="true"
                    className="flex size-8 shrink-0 items-center justify-center text-neutral-300"
                  >
                    {tab.icon}
                  </span>
                  <span className="min-w-0 flex-1 overflow-hidden pl-2 text-sm">
                    {renderTabTitle(tab, 'block truncate')}
                  </span>
                  <ChevronRightIcon
                    size={18}
                    aria-hidden="true"
                    className="shrink-0 text-neutral-500"
                  />
                </ButtonBase>
              ))}
            </div>
          </ScrollArea>
        )}
      </div>
    </Modal>
  );

  return (
    <>
      <Tooltip
        title={isMobileSettingsLayout ? null : t('settings.title')}
        placement={tooltipPlacement}
        mouseEnterDelay={0.6}
      >
        <ButtonBase
          aria-label={
            isUpdateAvailable
              ? `${t('settings.title')}: ${t('settings.update.title')}`
              : t('settings.title')
          }
          className="group flex h-[30px] w-[30px] items-center justify-center rounded text-neutral-300 transition-colors enabled:hover:bg-neutral-700/80 enabled:hover:text-white enabled:active:bg-neutral-600/70"
          onClick={openModal}
        >
          <Badge dot={isUpdateAvailable} color="blue" offset={[0, 2]}>
            <div className="pt-[3px] text-neutral-300 transition-colors group-hover:text-white group-active:text-white">
              <SettingsIcon size={18} aria-hidden="true" />
            </div>
          </Badge>
        </ButtonBase>
      </Tooltip>

      {isMobileSettingsLayout ? renderMobileModal() : renderDesktopModal()}
    </>
  );
};
