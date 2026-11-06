import { SquareDashedMousePointerIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { MenuSubmenu } from '@/components/menu-item.tsx';

import { MouseModeOptions } from './mouse-mode-options.tsx';

export const MouseMode = () => {
  const { t } = useTranslation();

  return (
    <MenuSubmenu
      title={t('mouse.mode')}
      content={<MouseModeOptions />}
      popoverProps={{ placement: 'rightTop', arrow: false, align: { offset: [14, 0] } }}
    >
      <div className="flex h-[30px] cursor-pointer items-center space-x-2 rounded px-3 text-neutral-300 hover:bg-neutral-700/70">
        <SquareDashedMousePointerIcon size={18} />
        <span>{t('mouse.mode')}</span>
      </div>
    </MenuSubmenu>
  );
};
