import { useCallback, useEffect, useState } from 'react';
import { Switch } from 'antd';
import { useAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import * as storage from '@/lib/localstorage.ts';
import { menuDisabledItemsAtom } from '@/jotai/settings.ts';

import { FingerRobotDebug } from './finger-robot-debug.tsx';

const powerMenuItem = 'power';

function isFingerRobotDebugVisible(disabledItems: readonly string[]): boolean {
  return !disabledItems.includes(powerMenuItem);
}

export const FingerRobot = () => {
  const { t } = useTranslation();

  const [menuDisabledItems, setMenuDisabledItems] = useAtom(menuDisabledItemsAtom);
  const [isLoading, setIsLoading] = useState(false);
  const debugVisible = isFingerRobotDebugVisible(menuDisabledItems);

  const getMenuBarConfig = useCallback(async () => {
    setIsLoading(true);

    try {
      const rsp = await api.getMenuBarConfig();
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setMenuDisabledItems(storage.normalizeMenuDisabledItems(rsp.data?.disabledItems));
    } catch (err) {
      console.log(err);
    } finally {
      setIsLoading(false);
    }
  }, [setMenuDisabledItems]);

  useEffect(() => {
    void getMenuBarConfig();
  }, [getMenuBarConfig]);

  async function update(visible: boolean) {
    if (isLoading) return;
    setIsLoading(true);

    const newItems = visible
      ? menuDisabledItems.filter((item) => item !== powerMenuItem)
      : Array.from(new Set([...menuDisabledItems, powerMenuItem]));

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
      setIsLoading(false);
    }
  }

  return (
    <div>
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.device.fingerRobot.title', { defaultValue: 'Finger Robot' })}</span>
          <span className="text-xs text-neutral-500">
            {t('settings.device.fingerRobot.description', {
              defaultValue: 'Enable finger robot control'
            })}
          </span>
        </div>

        <Switch checked={debugVisible} loading={isLoading} onChange={update} />
      </div>

      {debugVisible && <FingerRobotDebug />}
    </div>
  );
};
