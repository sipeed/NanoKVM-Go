import { Select } from 'antd';
import { useTranslation } from 'react-i18next';

import languages from '@/i18n/languages.ts';
import { normalizeLanguage } from '@/i18n/locale.ts';
import { feedback } from '@/lib/feedback.tsx';
import { setLanguage } from '@/lib/localstorage.ts';

export const Language = () => {
  const { t, i18n } = useTranslation();
  const selectedLanguage = normalizeLanguage(i18n.resolvedLanguage || i18n.language);

  const options = languages.map((language) => ({
    value: language.key,
    label: language.name
  }));

  async function changeLanguage(value: string) {
    const nextLanguage = normalizeLanguage(value);
    if (selectedLanguage === nextLanguage) return;

    try {
      await i18n.changeLanguage(nextLanguage);
      setLanguage(nextLanguage);
    } catch (error) {
      console.error(error);
      feedback.error(t('settings.appearance.languageChangeFailed'));
    }
  }

  return (
    <div className="mt-5 flex items-center justify-between">
      <div className="flex flex-col">
        <span>{t('settings.appearance.language')}</span>
        <span className="text-xs text-neutral-500">{t('settings.appearance.languageDesc')}</span>
      </div>

      <Select
        value={selectedLanguage}
        style={{ width: 180 }}
        options={options}
        onChange={changeLanguage}
      />
    </div>
  );
};
