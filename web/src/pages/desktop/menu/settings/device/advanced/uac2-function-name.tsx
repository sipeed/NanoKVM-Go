import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

import { BootStringSetting } from './boot-string-setting.tsx';

export const Uac2FunctionName = () => {
  const { t } = useTranslation();

  return (
    <BootStringSetting
      maxLength={64}
      api={{
        get: api.getUac2FunctionName,
        set: api.setUac2FunctionName
      }}
      labels={{
        title: t('settings.device.uac2FunctionName.title'),
        config: t('settings.device.uac2FunctionName.config'),
        tip: t('settings.device.uac2FunctionName.tip'),
        enable: t('settings.device.uac2FunctionName.enable'),
        placeholder: t('settings.device.uac2FunctionName.placeholder'),
        notEmpty: t('settings.device.uac2FunctionName.notEmpty'),
        notValid: t('settings.device.uac2FunctionName.notValid'),
        failed: t('settings.device.uac2FunctionName.failed'),
        success: t('settings.device.uac2FunctionName.success'),
        okBtn: t('settings.device.uac2FunctionName.okBtn'),
        cancelBtn: t('settings.device.uac2FunctionName.cancelBtn')
      }}
    />
  );
};
