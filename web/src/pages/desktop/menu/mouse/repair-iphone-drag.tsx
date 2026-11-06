import { useEffect, useRef, useState } from 'react';
import clsx from 'clsx';
import { useAtomValue } from 'jotai';
import { WrenchIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { inputRouter, InputSource } from '@/lib/input-router.ts';
import { canRepairIphoneDrag } from '@/lib/pointer-mode.ts';
import { mouseModeAtom, pointerModeInputEnabledAtom } from '@/jotai/mouse.ts';

const IOS_REPAIR_NUDGE_DELAY_MS = 30;

export const RepairIphoneDrag = () => {
  const { t } = useTranslation();
  const mouseMode = useAtomValue(mouseModeAtom);
  const inputEnabled = useAtomValue(pointerModeInputEnabledAtom);
  const [isRepairing, setIsRepairing] = useState(false);
  const repairTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (canRepairIphoneDrag(mouseMode, inputEnabled)) return;
    if (repairTimerRef.current) {
      clearTimeout(repairTimerRef.current);
      repairTimerRef.current = null;
    }
    setIsRepairing(false);
  }, [inputEnabled, mouseMode]);

  if (!canRepairIphoneDrag(mouseMode, inputEnabled)) return null;

  function repairIphoneDrag() {
    if (isRepairing) return;

    setIsRepairing(true);
    inputRouter.sendAbsolutePointer(InputSource.VirtualPointer, 0, 0.5, 0.5, 1);

    if (repairTimerRef.current) {
      clearTimeout(repairTimerRef.current);
    }

    repairTimerRef.current = setTimeout(() => {
      repairTimerRef.current = null;
      inputRouter.sendAbsolutePointer(InputSource.VirtualPointer, 0, 0.5, 0.5, -1);
      setIsRepairing(false);
    }, IOS_REPAIR_NUDGE_DELAY_MS);
  }

  return (
    <div
      className="flex h-[30px] cursor-pointer select-none items-center space-x-2 rounded px-3 text-neutral-300 hover:bg-neutral-700/70"
      onClick={repairIphoneDrag}
    >
      <WrenchIcon className={clsx({ 'text-blue-500': isRepairing })} size={18} />
      <span>{t('mouse.repairIphoneDrag', { defaultValue: 'Repair iPhone drag' })}</span>
    </div>
  );
};
