import { useEffect } from 'react';
import { useAtomValue } from 'jotai';
import { CheckIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { requestPointerModeSync } from '@/lib/pointer-mode.ts';
import { mouseModeAtom, type MouseMode } from '@/jotai/mouse.ts';
import { usePointerModeSelection } from '@/hooks/usePointerModeSelection.ts';

type MouseModeOptionsProps = {
  labels?: Partial<Record<MouseMode, string>>;
  showSelected?: boolean;
  onSelect?: (mode: MouseMode) => void;
};

export const MouseModeOptions = ({
  labels,
  showSelected = true,
  onSelect
}: MouseModeOptionsProps) => {
  const { t } = useTranslation();
  const mouseMode = useAtomValue(mouseModeAtom);
  const { selectMode, switching } = usePointerModeSelection();

  useEffect(() => {
    // Opening this submenu is also an explicit resynchronization point for
    // changes made by another browser/client.
    requestPointerModeSync();
  }, []);

  const mouseModes: { name: string; value: MouseMode }[] = [
    { name: labels?.absolute ?? t('mouse.absolute'), value: 'absolute' },
    {
      name: labels?.['digitizer-stylus'] ?? t('mouse.absoluteAndroid'),
      value: 'digitizer-stylus'
    },
    {
      name:
        labels?.['touch-screen'] ?? t('mouse.touchScreen', { defaultValue: 'Multi-touch screen' }),
      value: 'touch-screen'
    },
    { name: labels?.relative ?? t('mouse.relative'), value: 'relative' }
  ];

  async function updateMouseMode(mode: MouseMode) {
    if (switching) return;
    if (await selectMode(mode)) onSelect?.(mode);
  }

  return (
    <>
      {mouseModes.map((mode) => (
        <div
          key={mode.value}
          data-mouse-mode={mode.value}
          aria-disabled={switching}
          className={`flex items-center space-x-1 rounded py-1.5 pl-2 pr-5 ${
            switching ? 'cursor-not-allowed opacity-50' : 'cursor-pointer hover:bg-neutral-700/70'
          }`}
          onClick={() => void updateMouseMode(mode.value)}
        >
          <div className="flex h-[16px] w-[16px] items-end text-blue-500">
            {showSelected && mode.value === mouseMode && <CheckIcon strokeWidth={3} size={16} />}
          </div>
          <span>{mode.name}</span>
        </div>
      ))}
    </>
  );
};
