import { useTranslation } from 'react-i18next';

import { useCountdownConfirmation } from './useCountdownConfirmation.tsx';

export function useEdidUpdateConfirmation() {
  const { t } = useTranslation();

  return useCountdownConfirmation({
    title: t('settings.screen.edid.powerConfirmTitle'),
    description: t('settings.screen.edid.powerConfirmDescription'),
    ok: t('settings.screen.edid.powerConfirmOk'),
    okCountdown: (seconds) =>
      t('settings.screen.edid.powerConfirmOkCountdown', {
        seconds
      }),
    cancel: t('settings.screen.edid.powerConfirmCancel')
  });
}
