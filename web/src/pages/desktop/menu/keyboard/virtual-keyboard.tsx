import { useSetAtom } from 'jotai';
import { KeyboardIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { isKeyboardOpenAtom } from '@/jotai/keyboard.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

export const VirtualKeyboard = () => {
  const { t } = useTranslation();
  const setIsKeyboardOpen = useSetAtom(isKeyboardOpenAtom);

  return (
    <ButtonBase
      className="flex items-center space-x-2 rounded py-1 pl-2 pr-5 text-left text-neutral-200 hover:bg-neutral-700/70 hover:text-white"
      onClick={() => setIsKeyboardOpen((o) => !o)}
    >
      <KeyboardIcon size={18} />
      <span>{t('keyboard.virtual')}</span>
    </ButtonBase>
  );
};
