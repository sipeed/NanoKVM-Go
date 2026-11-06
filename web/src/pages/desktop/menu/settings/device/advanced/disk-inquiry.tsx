import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

import { BootStringSetting } from './boot-string-setting.tsx';

export const DiskInquiry = () => {
  const { t } = useTranslation();

  return (
    <BootStringSetting
      asciiOnly
      maxLength={28}
      api={{
        get: api.getDiskInquiry,
        set: api.setDiskInquiry
      }}
      labels={{
        title: t('settings.device.diskInquiry.title'),
        config: t('settings.device.diskInquiry.config'),
        tip: t('settings.device.diskInquiry.tip'),
        enable: t('settings.device.diskInquiry.enable'),
        placeholder: t('settings.device.diskInquiry.placeholder'),
        notEmpty: t('settings.device.diskInquiry.notEmpty'),
        notValid: t('settings.device.diskInquiry.notValid'),
        failed: t('settings.device.diskInquiry.failed'),
        success: t('settings.device.diskInquiry.success'),
        okBtn: t('settings.device.diskInquiry.okBtn'),
        cancelBtn: t('settings.device.diskInquiry.cancelBtn')
      }}
    />
  );
};
