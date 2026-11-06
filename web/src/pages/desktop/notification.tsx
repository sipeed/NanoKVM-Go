import { useEffect, useRef } from 'react';
import { Button, notification } from 'antd';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import * as aiApi from '@/api/ai.ts';
import { isPasswordUpdated } from '@/api/auth.ts';
import { getBrowserInfo } from '@/lib/browser.ts';
import * as storage from '@/lib/localstorage.ts';
import { getScreenTimelapseStorageWarning } from '@/lib/screen-timelapse-retention-warning.ts';

const screenTimelapseStorageNotificationKey = 'screen_timelapse_storage_warning';
const screenTimelapseRetentionStatusPollIntervalMs = 60 * 1000;

function formatStorageBytes(bytes: number, locale: string): string {
  const gib = bytes / (1024 * 1024 * 1024);
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(gib)} GiB`;
}

export const Notification = () => {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const [api, contextHolder] = notification.useNotification();
  const retentionRequestInFlightRef = useRef(false);
  const activeStorageWarningSignatureRef = useRef<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    async function showPasswordNotification() {
      const skip = storage.getSkipModifyPassword();
      if (skip) return;

      try {
        const rsp = await isPasswordUpdated();
        if (isMounted && rsp.code === 0 && !rsp.data.isUpdated) {
          api.warning({
            key: 'no_change_password',
            message: t('auth.changePassword'),
            description: t('auth.changePasswordDesc'),
            placement: 'topRight',
            btn: (
              <Button
                type="primary"
                onClick={() => {
                  api.destroy();
                  navigate('/auth/password');
                }}
              >
                {t('auth.ok')}
              </Button>
            ),
            duration: null,
            onClose: () => storage.setSkipModifyPassword(true)
          });
        }
      } catch (error) {
        console.error(error);
      }
    }

    function showBrowserNotification() {
      const skip = storage.getSkipBrowserNotification();
      if (skip) return;

      const info = getBrowserInfo();
      if (!info.isChrome) {
        api.info({
          key: 'recommend_browser.',
          message: t('notification.browser.title'),
          description: t('notification.browser.description'),
          placement: 'topRight',
          duration: null,
          onClose: () => storage.setSkipBrowserNotification(true)
        });
      }
    }

    showPasswordNotification();
    showBrowserNotification();

    return () => {
      isMounted = false;
    };
  }, [api, navigate, t]);

  useEffect(() => {
    let isMounted = true;

    async function refreshStorageWarning() {
      if (!isMounted || retentionRequestInFlightRef.current) return;
      retentionRequestInFlightRef.current = true;

      try {
        const rsp = await aiApi.getScreenTimelapseRetention();
        if (!isMounted || rsp.code !== 0 || !rsp.data) return;

        const warning = getScreenTimelapseStorageWarning(rsp.data);
        if (!warning) {
          if (activeStorageWarningSignatureRef.current !== null) {
            activeStorageWarningSignatureRef.current = null;
            api.destroy(screenTimelapseStorageNotificationKey);
          }
          return;
        }

        const locale = i18n.resolvedLanguage || i18n.language || 'en';
        const signature = `${locale}:${warning.usedBytes}:${warning.limitBytes}`;
        if (activeStorageWarningSignatureRef.current === signature) return;

        activeStorageWarningSignatureRef.current = signature;
        api.warning({
          key: screenTimelapseStorageNotificationKey,
          message: t('settings.ai.retention.storageWarningTitle'),
          description: t('settings.ai.retention.storageWarningDescription', {
            used: formatStorageBytes(warning.usedBytes, locale),
            limit: formatStorageBytes(warning.limitBytes, locale)
          }),
          placement: 'topRight',
          duration: null
        });
      } catch {
        // Keep the current notification during transient status-request failures.
      } finally {
        retentionRequestInFlightRef.current = false;
      }
    }

    void refreshStorageWarning();
    const timer = window.setInterval(
      () => void refreshStorageWarning(),
      screenTimelapseRetentionStatusPollIntervalMs
    );
    const refreshOnFocus = () => void refreshStorageWarning();
    const refreshWhenVisible = () => {
      if (document.visibilityState === 'visible') refreshOnFocus();
    };
    window.addEventListener('focus', refreshOnFocus);
    document.addEventListener('visibilitychange', refreshWhenVisible);

    return () => {
      isMounted = false;
      window.clearInterval(timer);
      window.removeEventListener('focus', refreshOnFocus);
      document.removeEventListener('visibilitychange', refreshWhenVisible);
      if (activeStorageWarningSignatureRef.current !== null) {
        activeStorageWarningSignatureRef.current = null;
        api.destroy(screenTimelapseStorageNotificationKey);
      }
    };
  }, [api, i18n.language, i18n.resolvedLanguage, t]);

  return <>{contextHolder}</>;
};
