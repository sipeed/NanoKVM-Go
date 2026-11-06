import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';

import { BootStringSetting } from './boot-string-setting.tsx';

export const DiskInquiryCdrom = () => {
  const { t } = useTranslation();

  return (
    <BootStringSetting
      asciiOnly
      maxLength={28}
      api={{
        get: api.getDiskInquiryCdrom,
        set: api.setDiskInquiryCdrom
      }}
      labels={{
        title: t('settings.device.diskInquiryCdrom.title'),
        config: t('settings.device.diskInquiryCdrom.config'),
        tip: t('settings.device.diskInquiryCdrom.tip'),
        enable: t('settings.device.diskInquiryCdrom.enable'),
        placeholder: t('settings.device.diskInquiryCdrom.placeholder'),
        notEmpty: t('settings.device.diskInquiryCdrom.notEmpty'),
        notValid: t('settings.device.diskInquiryCdrom.notValid'),
        failed: t('settings.device.diskInquiryCdrom.failed'),
        success: t('settings.device.diskInquiryCdrom.success'),
        okBtn: t('settings.device.diskInquiryCdrom.okBtn'),
        cancelBtn: t('settings.device.diskInquiryCdrom.cancelBtn')
      }}
    />
  );
};
