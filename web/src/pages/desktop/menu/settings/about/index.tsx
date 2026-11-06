import { Divider } from 'antd';
import { useTranslation } from 'react-i18next';

import { Community } from './community.tsx';
import { Hostname } from './hostname.tsx';
import { Information } from './information.tsx';

export const About = () => {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="text-base font-bold">{t('settings.about.title')}</div>
      </div>
      <Divider className="opacity-50" />

      <Information />
      <Hostname />
      <Divider className="opacity-50" />

      <Community />
    </>
  );
};
