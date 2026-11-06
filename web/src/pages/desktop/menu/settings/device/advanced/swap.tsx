import { useEffect, useState } from 'react';
import { Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

export const Swap = () => {
  const { t } = useTranslation();

  const [isEnabled, setIsEnabled] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [errMsg, setErrMsg] = useState('');

  useEffect(() => {
    let isMounted = true;

    async function getSwapMode() {
      setIsLoading(true);

      try {
        const rsp = await api.getSwapMode();
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setErrMsg(t('settings.device.swap.errors.load'));
          return;
        }

        if (!isMounted) return;
        setErrMsg('');
        setIsEnabled(!!rsp.data?.enabled);
      } catch (err) {
        console.log(err);
        if (isMounted) {
          setErrMsg(t('settings.device.swap.errors.load'));
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    }

    getSwapMode();

    return () => {
      isMounted = false;
    };
  }, [t]);

  async function update(enable: boolean) {
    if (isLoading) return;
    setIsLoading(true);
    setErrMsg('');

    try {
      const rsp = await api.setSwapMode(enable);
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        setErrMsg(t('settings.device.swap.errors.update'));
        return;
      }

      setIsEnabled(enable);
    } catch (err) {
      console.log(err);
      setErrMsg(t('settings.device.swap.errors.update'));
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="flex flex-col space-y-3">
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <span>{t('settings.device.swap.title')}</span>
          <span className="text-xs text-neutral-500">{t('settings.device.swap.description')}</span>
        </div>

        <Switch checked={isEnabled} loading={isLoading} onChange={update} />
      </div>

      {errMsg && <span className="break-words text-xs text-red-500">{errMsg}</span>}
    </div>
  );
};
