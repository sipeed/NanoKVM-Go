import { Fragment, useEffect, useId, useState } from 'react';
import { Button, Checkbox, Switch } from 'antd';
import { useAtom } from 'jotai';
import {
  FileJsonIcon,
  HandIcon,
  HardDriveIcon,
  HistoryIcon,
  MaximizeIcon,
  MicIcon,
  MousePointerClickIcon,
  NetworkIcon,
  PictureInPicture2Icon,
  TerminalSquareIcon,
  VideoIcon,
  Volume2Icon,
  XIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as aiAPI from '@/api/ai.ts';
import * as api from '@/api/vm.ts';
import { connectFeatures } from '@/lib/localstorage.ts';
import { isPictureInPictureSupported } from '@/lib/picture-in-picture.ts';
import { connectPreferencesAtom, menuDisabledItemsAtom } from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

import { useDesktopWebcam } from '../../../connect-features.tsx';
import { useMicrophone } from '../../../microphone-context.tsx';

import { filterMenuIconItems, getSupportedServiceIDs } from './menu-icon-items.ts';

export const MenuIcons = () => {
  const { t } = useTranslation();

  const [menuDisabledItems, setMenuDisabledItems] = useAtom(menuDisabledItemsAtom);

  const [preferences, setPreference] = useAtom(connectPreferencesAtom);
  const webcam = useDesktopWebcam();
  const microphone = useMicrophone();
  const id = useId();

  const [loadingItem, setLoadingItem] = useState('');
  const [supportedServiceIDs, setSupportedServiceIDs] = useState<Set<string>>(new Set());

  const responsiveDevice = useResponsiveDevice();

  const allItems = [
    { key: 'volume', icon: <Volume2Icon size={16} /> },
    { key: 'microphone', icon: <MicIcon size={16} /> },
    { key: 'webcam', icon: <VideoIcon size={16} /> },
    {
      key: 'mouseJiggler',
      icon: <MousePointerClickIcon size={16} />,
      label: 'settings.device.mouseJiggler.title'
    },
    {
      key: 'pictureInPicture',
      icon: <PictureInPicture2Icon size={16} />,
      label: 'pictureInPicture.title'
    },
    { key: 'image', icon: <HardDriveIcon size={16} />, label: 'storage.title' },
    { key: 'script', icon: <FileJsonIcon size={16} /> },
    { key: 'terminal', icon: <TerminalSquareIcon size={16} /> },
    { key: 'wol', icon: <NetworkIcon size={16} /> },
    {
      key: 'power',
      icon: <HandIcon size={16} />,
      label: 'settings.device.fingerRobot.title'
    },
    {
      key: 'screenTimelapse',
      icon: <HistoryIcon size={16} />,
      label: 'settings.ai.screenTimelapseService'
    },
    { key: 'fullscreen', icon: <MaximizeIcon size={16} />, label: 'fullscreen.toggle' },
    { key: 'collapse', icon: <XIcon size={16} />, label: 'menu.collapse' }
  ];

  const items = filterMenuIconItems(
    allItems.filter((item) => item.key !== 'pictureInPicture' || isPictureInPictureSupported()),
    {
      isMobilePortrait: responsiveDevice.isMobilePortrait,
      supportedServiceIDs
    }
  );

  useEffect(() => {
    let isCancelled = false;

    async function loadMenuBarConfig() {
      try {
        const rsp = await api.getMenuBarConfig();
        if (!isCancelled && rsp.code === 0 && rsp.data?.disabledItems?.length) {
          setMenuDisabledItems(rsp.data.disabledItems);
        }
      } catch (err) {
        console.log(err);
      }
    }

    async function loadSupportedServices() {
      try {
        const rsp = await aiAPI.getAIServices();
        if (isCancelled || rsp.code !== 0) {
          return;
        }

        setSupportedServiceIDs(getSupportedServiceIDs(rsp.data?.services));
      } catch (err) {
        console.log(err);
      }
    }

    void loadMenuBarConfig();
    void loadSupportedServices();

    return () => {
      isCancelled = true;
    };
  }, [setMenuDisabledItems]);

  async function updateItems(item: string) {
    if (loadingItem) return;
    setLoadingItem(item);

    const exist = menuDisabledItems.includes(item);
    const newItems = exist
      ? menuDisabledItems.filter((disabledItem) => disabledItem !== item)
      : [...menuDisabledItems, item];

    try {
      const rsp = await api.setMenuBarConfig(newItems);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setMenuDisabledItems(newItems);
    } catch (err) {
      console.log(err);
    } finally {
      setLoadingItem('');
    }
  }

  return (
    <div className="mt-8 flex flex-col space-y-5">
      <div className="flex flex-col">
        <span>{t('settings.appearance.menuBar.icons')}</span>
        <span className="text-xs text-neutral-500">
          {t('settings.appearance.menuBar.iconsDesc')}
        </span>
      </div>

      <div className="grid grid-cols-[minmax(0,1fr)_minmax(3rem,16%)_minmax(4.5rem,22%)] items-center gap-x-2 gap-y-4">
        <span />
        <span id={id + '-show'} className="text-center text-xs text-neutral-400">
          {t('settings.appearance.menuBar.showIcon')}
        </span>
        <span id={id + '-connect'} className="text-center text-xs text-neutral-400">
          {t('settings.appearance.menuBar.activateOnConnect')}
        </span>
        {items.map((item) => {
          const feature = connectFeatures.find((key) => key === item.key);
          const labelId = id + '-' + item.key;
          const hidden = menuDisabledItems.includes(item.key);
          const stop = hidden && item.key === 'webcam' && webcam.status !== 'off'
            ? webcam.stop
            : hidden && item.key === 'microphone' && microphone.status !== 'disabled'
              ? microphone.disable
              : undefined;
          return (
            <Fragment key={item.key}>
              <div className="flex flex-wrap items-center gap-2 text-neutral-400">
                {item.icon}
                <span id={labelId} className="text-neutral-300">
                  {item.label ? t(item.label) : t(`${item.key}.title`)}
                </span>
                {stop && (
                  <Button size="small" onClick={() => void stop()}>
                    {t('settings.appearance.menuBar.stopSharing')}
                  </Button>
                )}
              </div>
              <Switch
                className="justify-self-center"
                aria-labelledby={labelId + ' ' + id + '-show'}
                value={!hidden}
                loading={item.key === loadingItem}
                onChange={() => updateItems(item.key)}
              />
              <div className="justify-self-center">
                {feature && (
                  <Checkbox
                    aria-labelledby={labelId + ' ' + id + '-connect'}
                    checked={preferences[feature]}
                    onChange={(event) => setPreference(feature, event.target.checked)}
                  />
                )}
              </div>
            </Fragment>
          );
        })}
      </div>
    </div>
  );
};
