import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';

import { consumeAuthSessionNotice } from '@/lib/auth-session.ts';
import { feedback } from '@/lib/feedback.tsx';

export const AuthSessionNotice = () => {
  const { t } = useTranslation();

  useEffect(() => {
    const timer = window.setTimeout(() => {
      const notice = consumeAuthSessionNotice();
      if (notice === 'twoFactorEnabled') {
        feedback.warning(t('auth.twoFactorEnabledLogout.message'), 6);
      } else if (notice === 'systemTimeUpdated') {
        feedback.success(t('settings.device.datetime.setSucceeded'), 6);
      }
    }, 0);

    return () => window.clearTimeout(timer);
  }, [t]);

  return null;
};
