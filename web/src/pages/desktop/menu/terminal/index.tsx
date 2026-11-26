import { Tooltip } from 'antd';
import { SquareTerminalIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export const Terminal = ({ tooltipPlacement }: { tooltipPlacement: 'bottom' | 'left' | 'right' }) => {
  const { t } = useTranslation();
  const title = t('terminal.title');

  return (
    <Tooltip title={title} placement={tooltipPlacement} mouseEnterDelay={0.6}>
      <a
        href="/#terminal"
        target="_blank"
        rel="noopener noreferrer"
        aria-label={title}
        className="flex size-[30px] items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white"
        onKeyDown={(event) => event.stopPropagation()}
        onKeyUp={(event) => event.stopPropagation()}
      >
        <SquareTerminalIcon size={18} />
      </a>
    </Tooltip>
  );
};
