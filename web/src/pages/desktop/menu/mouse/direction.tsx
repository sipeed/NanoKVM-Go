import { useAtom } from 'jotai';
import { ArrowDownUpIcon, CheckIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as storage from '@/lib/localstorage.ts';
import { scrollDirectionAtom } from '@/jotai/mouse.ts';
import { MenuSubmenu } from '@/components/menu-item.tsx';

export const Direction = () => {
  const { t } = useTranslation();

  const [scrollDirection, setScrollDirection] = useAtom(scrollDirectionAtom);

  const directions = [
    // Which option feels natural depends on the scroll settings of both the browser's
    // machine and the target. These labels match a natural-scrolling touchpad driving a
    // macOS target with its default (natural) scrolling; the stored values are unchanged.
    { name: t('mouse.scrollNatural'), value: 1 },
    { name: t('mouse.scrollConventional'), value: -1 }
  ];

  function update(direction: number): void {
    const value = Number(direction);

    setScrollDirection(value);
    storage.setMouseScrollDirection(value);
  }

  const content = (
    <>
      {directions.map((direction) => (
        <div
          key={direction.value}
          className="flex cursor-pointer items-center space-x-1 rounded py-1.5 pl-2 pr-5 hover:bg-neutral-700/70"
          onClick={() => update(direction.value)}
        >
          <div className="flex h-[16px] w-[16px] items-end text-blue-500">
            {direction.value === scrollDirection && <CheckIcon strokeWidth={3} size={16} />}
          </div>
          <span>{direction.name}</span>
        </div>
      ))}
    </>
  );

  return (
    <MenuSubmenu
      title={t('mouse.direction')}
      content={content}
      popoverProps={{ placement: 'rightTop', arrow: false, align: { offset: [13, 0] } }}
    >
      <div className="flex h-[30px] cursor-pointer items-center space-x-1 rounded px-3 text-neutral-300 hover:bg-neutral-700/50">
        <div className="flex h-[14px] w-[20px] items-end">
          <ArrowDownUpIcon size={16} />
        </div>
        <span>{t('mouse.direction')}</span>
      </div>
    </MenuSubmenu>
  );
};
