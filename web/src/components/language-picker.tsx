import { useState } from 'react';
import { Dropdown, type MenuProps } from 'antd';
import clsx from 'clsx';
import { ChevronDownIcon, GlobeIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import languages from '@/i18n/languages.ts';
import { normalizeLanguage, type SupportedLanguage } from '@/i18n/locale.ts';
import { setLanguage } from '@/lib/localstorage.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

type LanguagePickerProps = {
  className?: string;
};

export const LanguagePicker = ({ className }: LanguagePickerProps) => {
  const { t, i18n } = useTranslation();
  const [isOpen, setIsOpen] = useState(false);
  const selectedLanguage = normalizeLanguage(i18n.resolvedLanguage || i18n.language);

  const selectedLanguageName =
    languages.find((language) => language.key === selectedLanguage)?.name ??
    languages.find((language) => language.key === 'en')!.name;

  async function changeLanguage(key: SupportedLanguage) {
    const nextLanguage = normalizeLanguage(key);
    if (selectedLanguage === nextLanguage) return;

    try {
      await i18n.changeLanguage(nextLanguage);
      setLanguage(nextLanguage);
    } catch (error) {
      console.error(error);
    }
  }

  const menuItems: MenuProps['items'] = languages.map((language) => ({
    key: language.key,
    label: language.name
  }));

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    setIsOpen(false);
    void changeLanguage(key as SupportedLanguage);
  };

  return (
    <Dropdown
      autoFocus
      menu={{
        items: menuItems,
        selectable: true,
        selectedKeys: [selectedLanguage],
        onClick: handleMenuClick,
        style: { maxHeight: 'calc(100dvh - 72px)', overflowY: 'auto' }
      }}
      open={isOpen}
      placement="bottomRight"
      trigger={['click']}
      onOpenChange={setIsOpen}
    >
      <ButtonBase
        aria-label={`${t('auth.language')}: ${selectedLanguageName}`}
        aria-haspopup="menu"
        aria-expanded={isOpen}
        className={clsx(
          'flex h-8 items-center gap-1.5 whitespace-nowrap rounded-md bg-neutral-800/80 px-2.5 font-sans text-sm text-neutral-300 ring-1 ring-inset ring-neutral-700/80 transition-colors enabled:hover:bg-neutral-700/80 enabled:hover:text-white enabled:active:bg-neutral-700/90',
          className
        )}
      >
        <GlobeIcon size={16} aria-hidden="true" />
        <span>{selectedLanguageName}</span>
        <ChevronDownIcon
          size={14}
          aria-hidden="true"
          className={clsx('transition-transform duration-200', isOpen && 'rotate-180')}
        />
      </ButtonBase>
    </Dropdown>
  );
};
