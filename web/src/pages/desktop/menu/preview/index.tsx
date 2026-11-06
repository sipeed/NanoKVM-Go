import { Tooltip } from 'antd';
import clsx from 'clsx';
import { useAtom } from 'jotai';
import { EyeIcon, MousePointer2Icon } from 'lucide-react';
import type { KeyboardEvent } from 'react';
import { useTranslation } from 'react-i18next';

import { isPreviewModeAtom } from '@/jotai/preview.ts';

export const Preview = () => {
  const { t } = useTranslation();
  const [isPreviewMode, setIsPreviewMode] = useAtom(isPreviewModeAtom);

  function togglePreviewMode() {
    setIsPreviewMode((value) => !value);
  }

  function handleKeyDown(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key !== 'Enter' && event.key !== ' ') {
      return;
    }

    event.preventDefault();
    togglePreviewMode();
  }

  return (
    <Tooltip title={t('preview.title')} placement="bottom" mouseEnterDelay={0.6}>
      <div
        role="button"
        tabIndex={0}
        className={clsx(
          'flex h-[30px] w-[30px] cursor-pointer items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white'
        )}
        aria-pressed={isPreviewMode}
        onClick={togglePreviewMode}
        onKeyDown={handleKeyDown}
      >
        {isPreviewMode ? (
          <EyeIcon size={18} className="text-red-400" />
        ) : (
          <MousePointer2Icon size={18} className="text-green-400" />
        )}
      </div>
    </Tooltip>
  );
};
