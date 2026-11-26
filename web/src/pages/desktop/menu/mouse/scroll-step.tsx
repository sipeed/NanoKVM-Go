import { Slider } from 'antd';
import { useAtom } from 'jotai';
import { MoveVerticalIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as storage from '@/lib/localstorage.ts';
import {
  clampPixelsPerNotch,
  WHEEL_PIXELS_PER_NOTCH_MAX,
  WHEEL_PIXELS_PER_NOTCH_MIN
} from '@/lib/wheel.ts';
import { scrollStepAtom } from '@/jotai/mouse.ts';
import { MenuSubmenu } from '@/components/menu-item.tsx';

// The slider shows sensitivity (0 = slow, 100 = fast); the stored value is the
// browser scroll distance per wheel notch, so a higher sensitivity means fewer pixels.
// The scale is logarithmic so each slider step changes the speed by the same ratio,
// which keeps the fast end (small distances) as easy to fine-tune as the slow end.
const RATIO = WHEEL_PIXELS_PER_NOTCH_MIN / WHEEL_PIXELS_PER_NOTCH_MAX;

function stepToSensitivity(step: number): number {
  const fraction =
    Math.log(clampPixelsPerNotch(step) / WHEEL_PIXELS_PER_NOTCH_MAX) / Math.log(RATIO);
  return Math.round(fraction * 100);
}

function sensitivityToStep(sensitivity: number): number {
  return Math.round(WHEEL_PIXELS_PER_NOTCH_MAX * RATIO ** (sensitivity / 100) * 10) / 10;
}

export const ScrollStep = () => {
  const { t } = useTranslation();
  const [scrollStep, setScrollStep] = useAtom(scrollStepAtom);

  function update(sensitivity: number): void {
    const step = sensitivityToStep(sensitivity);
    setScrollStep(step);
    storage.setMouseScrollStep(step);
  }

  const content = (
    <div className="h-[150px] min-w-[60px] py-3">
      <Slider
        vertical
        marks={{
          0: <span>{t('mouse.slow')}</span>,
          100: <span>{t('mouse.fast')}</span>
        }}
        range={false}
        included={false}
        step={1}
        value={stepToSensitivity(scrollStep)}
        onChange={update}
      />
    </div>
  );

  return (
    <MenuSubmenu
      title={t('mouse.scrollStep')}
      content={content}
      popoverProps={{ placement: 'rightTop', arrow: false, align: { offset: [14, 0] } }}
    >
      <div className="flex h-[30px] cursor-pointer items-center space-x-2 rounded px-3 text-neutral-300 hover:bg-neutral-700/70">
        <MoveVerticalIcon size={18} />
        <span>{t('mouse.scrollStep')}</span>
      </div>
    </MenuSubmenu>
  );
};
