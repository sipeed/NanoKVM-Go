import { Tooltip } from 'antd';
import { LoaderCircleIcon, MousePointerClickIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { useMouseJiggler } from '@/hooks/useMouseJiggler.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

export function MouseJigglerButton({
  tooltipPlacement
}: {
  tooltipPlacement: 'bottom' | 'left' | 'right';
}) {
  const { t } = useTranslation();
  const { config, busy, update } = useMouseJiggler();
  const title = t('settings.device.mouseJiggler.title');

  return (
    <Tooltip title={title} placement={tooltipPlacement} mouseEnterDelay={0.6}>
      <ButtonBase
        className="flex size-[30px] items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white"
        aria-label={title}
        aria-pressed={config?.enabled}
        disabled={busy}
        onClick={() => void update('toggle')}
      >
        {busy ? (
          <LoaderCircleIcon size={18} className="animate-spin" />
        ) : (
          <MousePointerClickIcon
            size={18}
            className={config?.enabled ? 'text-green-400' : undefined}
          />
        )}
      </ButtonBase>
    </Tooltip>
  );
}
