import { useState } from 'react';
import { Tooltip } from 'antd';
import clsx from 'clsx';
import { LoaderCircleIcon, MicIcon, MicOffIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { useDismissMobileMenu } from '@/components/mobile-menu-context.ts';

import { useMicrophone } from '../../microphone-context.tsx';

type MicrophoneProps = {
  tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
};

export const Microphone = ({ tooltipPlacement = 'bottom' }: MicrophoneProps) => {
  const { t } = useTranslation();
  const dismissMobileMenu = useDismissMobileMenu();
  const { status, requestEnable, disable, isChecking } = useMicrophone();
  const [isTooltipOpen, setIsTooltipOpen] = useState(false);

  const toggleMicrophone = () => {
    setIsTooltipOpen(false);
    if (status === 'disabled') void requestEnable(dismissMobileMenu);
    else void disable();
  };

  const isEnabled = status === 'enabled';
  const isEnabling = status === 'enabling';
  const isLoading = isChecking || isEnabling;

  return (
    <Tooltip
      title={t('microphone.title')}
      placement={tooltipPlacement}
      mouseEnterDelay={0.6}
      open={isTooltipOpen}
      onOpenChange={setIsTooltipOpen}
    >
      <div
        className={clsx(
          'flex h-[30px] w-[30px] cursor-pointer items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white',
          isChecking && 'pointer-events-none opacity-50'
        )}
        aria-disabled={isChecking}
        onClick={() => void toggleMicrophone()}
      >
        {isLoading ? (
          <LoaderCircleIcon size={18} className="animate-spin" />
        ) : isEnabled ? (
          <MicIcon size={18} className="text-green-400" />
        ) : (
          <MicOffIcon size={18} />
        )}
      </div>
    </Tooltip>
  );
};
