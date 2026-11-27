import { useRef, useState } from 'react';
import { message } from 'antd';
import clsx from 'clsx';
import { CheckIcon, ChevronRightIcon, KeyboardIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { getKeyboardLayout, setKeyboardLayout } from '@/api/hid.ts';
import { keyboardLayoutGroups, type KeyboardLayout } from '@/lib/keyboard-layouts.ts';
import { MenuSubmenu } from '@/components/menu-item.tsx';
import { ButtonBase } from '@/components/ui/button-base.tsx';

import { VirtualKeyboard } from './virtual-keyboard.tsx';

function rowClass(active = false) {
  return clsx(
    'flex w-full items-center gap-2 rounded px-2 py-1 text-left hover:bg-neutral-700/70',
    active ? 'text-green-400 hover:text-green-300' : 'text-neutral-200 hover:text-white'
  );
}

const popover = {
  trigger: ['hover', 'click'] as ('hover' | 'click')[],
  placement: 'rightTop' as const,
  arrow: false
};

export const KeyboardLayouts = () => {
  const { t, i18n } = useTranslation();
  const [selected, setSelected] = useState<KeyboardLayout>();
  const [saving, setSaving] = useState(false);
  const revision = useRef(0);
  const locale = i18n.language === 'cz' ? 'cs' : i18n.language.replace('_', '-');
  const names = new Intl.DisplayNames(locale, { type: 'language', languageDisplay: 'standard' });

  function layoutName(layout: KeyboardLayout) {
    switch (layout) {
      case 'fr-CA-multilingual':
        return t('keyboard.canadianMultilingual');
      case 'zh-CN':
        return `${names.of('zh-Hans')} · Microsoft Pinyin`;
      case 'fa-IR-standard':
        return `${names.of('fa-IR')} · استاندارد (ISIRI 9147)`;
      default:
        return names.of(layout);
    }
  }

  async function refresh(open: boolean) {
    if (!open || saving) return;
    const request = ++revision.current;
    try {
      const rsp = await getKeyboardLayout();
      if (rsp.code !== 0) throw new Error(rsp.msg);
      if (revision.current === request) setSelected(rsp.data.layout);
    } catch {
      if (revision.current === request) {
        setSelected(undefined);
        message.error(t('keyboard.layoutError'));
      }
    }
  }

  async function select(layout: KeyboardLayout) {
    if (saving || layout === selected) return;
    ++revision.current;
    setSaving(true);
    try {
      const rsp = await setKeyboardLayout(layout);
      if (rsp.code !== 0) throw new Error(rsp.msg);
      setSelected(rsp.data.layout);
    } catch {
      message.error(t('keyboard.layoutError'));
    } finally {
      setSaving(false);
    }
  }

  const content = (
    <div className="flex flex-col gap-1">
      {selected && (
        <div
          className="border-x-0 border-b border-t-0 border-solid border-neutral-700 px-2 pb-2"
          role="status"
        >
          <div className="text-xs text-neutral-400">{t('keyboard.currentKeyboard')}</div>
          <div className="font-medium text-green-400">{layoutName(selected)}</div>
        </div>
      )}
      <VirtualKeyboard />
      {Object.entries(keyboardLayoutGroups).map(([language, layouts]) => {
        const title = names.of(language) ?? language;
        return (
          <MenuSubmenu
            key={language}
            title={title}
            popoverProps={popover}
            content={
              <div className="flex flex-col gap-1">
                {layouts.map((layout) => (
                  <ButtonBase
                    key={layout}
                    className={rowClass(layout === selected)}
                    aria-pressed={layout === selected}
                    disabled={saving}
                    onClick={() => select(layout)}
                  >
                    <span className="flex-1 whitespace-nowrap">{layoutName(layout)}</span>
                    {selected === layout && <CheckIcon size={16} />}
                  </ButtonBase>
                ))}
              </div>
            }
          >
            <ButtonBase className={rowClass(layouts.some((layout) => layout === selected))}>
              <span className="flex-1">{title}</span>
              <ChevronRightIcon size={14} />
            </ButtonBase>
          </MenuSubmenu>
        );
      })}
    </div>
  );
  return (
    <MenuSubmenu
      title={t('keyboard.layouts')}
      content={content}
      popoverProps={{ ...popover, onOpenChange: refresh }}
    >
      <ButtonBase className={rowClass()}>
        <KeyboardIcon size={18} />
        <span className="flex-1">{t('keyboard.layouts')}</span>
        <ChevronRightIcon size={14} />
      </ButtonBase>
    </MenuSubmenu>
  );
};
