import { useEffect, useMemo, useState } from 'react';
import { Button } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { getIntlLocale } from '@/i18n/locale.ts';

import { formatLocalizedDateTime } from './model.ts';

type SynchronizationProps = {
  timezone: string;
};

export const Synchronization = ({ timezone }: SynchronizationProps) => {
  const { t, i18n } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const [status, setStatus] = useState<'' | 'notSynced' | 'synced'>('');
  const [lastSyncTime, setLastSyncTime] = useState<number | null>(null);

  useEffect(() => {
    if (!timezone) return;
    getTimeStatus();
  }, [timezone]);

  const formatter = useMemo(() => {
    if (!timezone) return null;

    return new Intl.DateTimeFormat(getIntlLocale(i18n.language), {
      timeZone: timezone,
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: 'numeric'
    });
  }, [timezone, i18n.language]);

  function getTimeStatus() {
    api.getTimeStatus().then((rsp) => {
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      setStatus(rsp.data?.isSynchronized ? 'synced' : 'notSynced');

      const nextLastSyncTime = Number(rsp.data.lastSyncTime);
      setLastSyncTime(Number.isFinite(nextLastSyncTime) ? nextLastSyncTime : null);
    });
  }

  function sync() {
    if (isLoading) return;
    setIsLoading(true);

    api
      .syncTime()
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return;
        }
      })
      .finally(() => {
        getTimeStatus();
        setIsLoading(false);
      });
  }

  return (
    <div className="flex items-center justify-between">
      <div className="flex flex-col">
        <span>{t('settings.device.datetime.synchronize')}</span>

        {status === 'notSynced' && (
          <span className="text-xs text-red-500">
            {t('settings.device.datetime.notSynchronized')}
          </span>
        )}

        {status === 'synced' && (
          <span className="text-xs text-neutral-500">
            {t('settings.device.datetime.lastSynchronization')}
            {lastSyncTime !== null && formatter
              ? formatLocalizedDateTime(formatter, new Date(lastSyncTime))
              : '-'}
          </span>
        )}
      </div>

      <Button type="primary" loading={isLoading} onClick={sync}>
        {t('settings.device.datetime.syncNow')}
      </Button>
    </div>
  );
};
