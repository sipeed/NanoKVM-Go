import clsx from 'clsx';
import { useAtom } from 'jotai';
import { ArrowLeftRightIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as storage from '@/lib/localstorage.ts';
import { swapCommandCtrlAtom } from '@/jotai/keyboard.ts';

export const SwapCommandCtrl = () => {
  const { t } = useTranslation();
  const [swapCommandCtrl, setSwapCommandCtrl] = useAtom(swapCommandCtrlAtom);

  function update() {
    const enabled = !swapCommandCtrl;
    setSwapCommandCtrl(enabled);
    storage.setKeyboardSwapCommandCtrl(enabled);
  }

  return (
    <button
      type="button"
      className={clsx(
        'flex w-full cursor-pointer select-none items-center space-x-2 rounded border-0 bg-transparent py-1 pl-2 pr-5 text-left hover:bg-neutral-700/70',
        swapCommandCtrl ? 'text-blue-500' : 'text-neutral-300'
      )}
      aria-pressed={swapCommandCtrl}
      onClick={update}
    >
      <ArrowLeftRightIcon className="shrink-0" size={18} />
      <span>{t('keyboard.swapCommandCtrl')}</span>
    </button>
  );
};
