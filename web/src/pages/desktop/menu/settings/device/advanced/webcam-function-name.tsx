import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

import { BootStringSetting } from './boot-string-setting.tsx';

export const WebcamFunctionName = () => {
  const { t } = useTranslation();

  return (
    <BootStringSetting
      maxLength={63}
      api={{
        get: api.getWebcamFunctionName,
        set: api.setWebcamFunctionName
      }}
      labels={{
        title: t('settings.device.webcamFunctionName.title'),
        config: t('settings.device.webcamFunctionName.config'),
        tip: t('settings.device.webcamFunctionName.tip'),
        enable: t('settings.device.webcamFunctionName.enable'),
        placeholder: t('settings.device.webcamFunctionName.placeholder'),
        notEmpty: t('settings.device.webcamFunctionName.notEmpty'),
        notValid: t('settings.device.webcamFunctionName.notValid'),
        failed: t('settings.device.webcamFunctionName.failed'),
        success: t('settings.device.webcamFunctionName.success'),
        okBtn: t('settings.device.webcamFunctionName.okBtn'),
        cancelBtn: t('settings.device.webcamFunctionName.cancelBtn')
      }}
    />
  );
};
