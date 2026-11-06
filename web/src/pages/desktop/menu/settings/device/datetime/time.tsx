import { useEffect, useMemo, useState } from 'react';
import { Button, DatePicker, Modal } from 'antd';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import { InfoIcon, LoaderCircleIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { getIntlLocale } from '@/i18n/locale.ts';
import { expireAuthSession } from '@/lib/auth-session.ts';
import { getBaseUrl } from '@/lib/service.ts';

import {
  formatDateTimeInTimeZone,
  formatLocalizedDateTime,
  normalizeDateTimeLocal
} from './model.ts';

dayjs.extend(utc);

type TimeProps = {
  timezone: string;
};

export const Time = ({ timezone }: TimeProps) => {
  const { t, i18n } = useTranslation();
  const [time, setTime] = useState<number | null | undefined>(undefined);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [datetime, setDatetime] = useState('');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  const formatter = useMemo(() => {
    if (!timezone) return null;

    return new Intl.DateTimeFormat(getIntlLocale(i18n.language), {
      timeZone: timezone,
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: 'numeric',
      minute: 'numeric',
      second: 'numeric'
    });
  }, [timezone, i18n.language]);

  const pickerValue = useMemo(() => {
    const normalizedDatetime = normalizeDateTimeLocal(datetime);
    return normalizedDatetime ? dayjs.utc(normalizedDatetime) : null;
  }, [datetime]);

  useEffect(() => {
    const baseURL = getBaseUrl('http');
    const url = `${baseURL}/api/vm/time`;
    const eventSource = new EventSource(url);

    eventSource.onmessage = (event) => {
      const timestamp = Number(event.data);
      if (!Number.isFinite(timestamp)) {
        setTime(null);
        return;
      }

      setTime(timestamp);
    };

    eventSource.onerror = (error) => {
      console.log(error);
      setTime(null);
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, []);

  function useBrowserTime() {
    const nextDatetime = formatDateTimeInTimeZone(new Date(), timezone);
    if (!nextDatetime) {
      setError(t('settings.device.datetime.invalidTime'));
      return;
    }

    setDatetime(nextDatetime);
    setError('');
  }

  function openManualTime() {
    if (!timezone) return;

    const nextDatetime = formatDateTimeInTimeZone(new Date(), timezone);
    setDatetime(nextDatetime ?? '');
    setError(nextDatetime ? '' : t('settings.device.datetime.invalidTime'));
    setIsModalOpen(true);
  }

  function closeManualTime() {
    if (isSaving) return;

    setIsModalOpen(false);
    setDatetime('');
    setError('');
  }

  async function saveManualTime() {
    if (isSaving) return;

    const normalizedDatetime = normalizeDateTimeLocal(datetime);
    if (!normalizedDatetime) {
      setError(t('settings.device.datetime.invalidTime'));
      return;
    }

    setIsSaving(true);
    setError('');

    try {
      const rsp = await api.setTime(normalizedDatetime);
      const timestamp = rsp.data?.timestamp;
      if (rsp.code !== 0 || typeof timestamp !== 'number' || !Number.isFinite(timestamp)) {
        setError(t('settings.device.datetime.setFailed'));
        return;
      }

      setTime(timestamp);
      setIsModalOpen(false);
      expireAuthSession('systemTimeUpdated');
    } catch (err) {
      console.log(err);
      setError(t('settings.device.datetime.setFailed'));
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <>
      <div className="flex items-center justify-between gap-4">
        <span>{t('settings.device.datetime.datetime')}</span>

        <div className="flex min-w-0 items-center justify-end gap-3">
          {time === undefined ? (
            <LoaderCircleIcon className="animate-spin text-neutral-500" size={18} />
          ) : time === null || !formatter ? (
            <div className="font-mono">-</div>
          ) : (
            <div className="truncate font-mono">
              {formatLocalizedDateTime(formatter, new Date(time))}
            </div>
          )}

          <Button className="shrink-0" size="small" disabled={!timezone} onClick={openManualTime}>
            {t('settings.device.datetime.manualSet')}
          </Button>
        </div>
      </div>

      <Modal
        open={isModalOpen}
        title={t('settings.device.datetime.manualSetTitle')}
        centered={true}
        width={420}
        okText={t('settings.device.datetime.setTime')}
        cancelText={t('settings.account.cancelBtn')}
        confirmLoading={isSaving}
        okButtonProps={{ disabled: isSaving || !normalizeDateTimeLocal(datetime) }}
        cancelButtonProps={{ disabled: isSaving }}
        closable={!isSaving}
        maskClosable={!isSaving}
        onOk={() => void saveManualTime()}
        onCancel={closeManualTime}
        destroyOnHidden={true}
      >
        <div className="space-y-4 pt-2">
          <div className="space-y-2">
            <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1">
              <label htmlFor="manual-system-time" className="text-sm font-medium text-neutral-200">
                {t('settings.device.datetime.dateAndTime')}
              </label>

              <Button
                type="link"
                size="small"
                className="h-auto shrink-0 p-0 text-xs"
                disabled={isSaving}
                onClick={useBrowserTime}
              >
                {t('settings.device.datetime.useBrowserTime')}
              </Button>
            </div>

            <DatePicker
              id="manual-system-time"
              className="w-full"
              value={pickerValue}
              format="YYYY-MM-DD HH:mm:ss"
              showTime={{ format: 'HH:mm:ss', changeOnScroll: true }}
              classNames={{ popup: { root: 'nanokvm-time-wheel-popup' } }}
              allowClear={false}
              inputReadOnly={true}
              needConfirm={false}
              showNow={false}
              aria-label={t('settings.device.datetime.dateAndTime')}
              disabled={isSaving}
              onChange={(value) => {
                setDatetime(value ? value.utc(true).format('YYYY-MM-DDTHH:mm:ss') : '');
                setError('');
              }}
            />

            {error && <div className="text-sm text-red-500">{error}</div>}
          </div>

          <div className="flex items-start gap-2 rounded-lg bg-neutral-800 px-3 py-2.5 text-xs text-neutral-500">
            <InfoIcon className="mt-0.5 shrink-0" size={16} aria-hidden="true" />
            <span>{t('settings.device.datetime.manualSetDescription')}</span>
          </div>
        </div>
      </Modal>
    </>
  );
};
