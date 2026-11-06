import { useEffect, useState } from 'react';
import { Switch } from 'antd';
import { useAtom } from 'jotai';
import {
  FileJsonIcon,
  HandIcon,
  HardDriveIcon,
  HistoryIcon,
  MaximizeIcon,
  MicIcon,
  NetworkIcon,
  PictureInPicture2Icon,
  TerminalSquareIcon,
  Volume2Icon,
  XIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as aiAPI from '@/api/ai.ts';
import * as api from '@/api/vm.ts';
import { isPictureInPictureSupported } from '@/lib/picture-in-picture.ts';
import { menuDisabledItemsAtom } from '@/jotai/settings.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

import { filterMenuIconItems, getSupportedServiceIDs } from './menu-icon-items.ts';

export const MenuIcons = () => {
  const { t } = useTranslation();

  const [menuDisabledItems, setMenuDisabledItems] = useAtom(menuDisabledItemsAtom);

  const [loadingItem, setLoadingItem] = useState('');
  const [supportedServiceIDs, setSupportedServiceIDs] = useState<Set<string>>(new Set());

  const responsiveDevice = useResponsiveDevice();

  const allItems = [
    { key: 'volume', icon: <Volume2Icon size={16} /> },
    { key: 'microphone', icon: <MicIcon size={16} /> },
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

      <div className="mt-5 flex flex-col space-y-5">
        {items.map((item) => (
          <div key={item.key} className="flex items-center justify-between">
            <div className="flex items-center space-x-2 text-neutral-400">
              {item.icon}
              <span className="text-neutral-300">
                {item.label ? t(item.label) : t(`${item.key}.title`)}
              </span>
            </div>

            <Switch
              value={!menuDisabledItems.includes(item.key)}
              loading={item.key === loadingItem}
              onChange={() => updateItems(item.key)}
            />
          </div>
        ))}
      </div>
    </div>
  );
};
