import { Alert, Spin, Steps } from 'antd';
import { useTranslation } from 'react-i18next';

import type { ApplicationUpdateStatus } from '@/api/application.ts';
import { ButtonBase } from '@/components/ui/button-base.tsx';

type UpdatingProps = {
  status: ApplicationUpdateStatus;
  fileName?: string;
  fileSize?: string;
  uploadProgress?: number;
  onCancelUpload?: () => void;
};

const phaseStep: Record<ApplicationUpdateStatus['phase'], number> = {
  idle: 0,
  downloading: 0,
  uploading: 0,
  verifying: 1,
  staging: 1,
  installing: 2,
  restarting: 3,
  succeeded: 3,
  failed: 0
};

// This component is deliberately presentation-only. The page owns status
// polling, cancellation, stale-response filtering, and the shared busy lock.
export const Updating = ({
  status,
  fileName,
  fileSize,
  uploadProgress,
  onCancelUpload
}: UpdatingProps) => {
  const { t } = useTranslation();
  const isTransferPhase = status.phase === 'downloading' || status.phase === 'uploading';
  const displayedTransferProgress = Math.min(
    100,
    Math.max(
      0,
      Math.round(
        status.phase === 'uploading' && uploadProgress !== undefined
          ? uploadProgress
          : status.progress
      )
    )
  );

  return (
    <div className="flex flex-col items-center justify-center space-y-8 py-12">
      <Spin size="large" />

      <div className="text-center">
        <div className="font-medium">{t(`settings.update.phases.${status.phase}`)}</div>
        {fileName && (
          <div className="mt-1 text-xs text-neutral-500">
            {fileName}
            {fileSize ? ` · ${fileSize}` : ''}
          </div>
        )}
      </div>

      <Alert
        className="w-fit max-w-full"
        type="warning"
        showIcon
        message={t('settings.update.powerWarning')}
      />

      <div className="w-full max-w-[520px]">
        <Steps
          size="small"
          current={phaseStep[status.phase]}
          items={[
            {
              title: t(
                status.source === 'offline' ? 'settings.update.upload' : 'settings.update.download'
              ),
              icon: isTransferPhase ? (
                <span className="flex size-6 items-center justify-center rounded-full bg-blue-500 text-[8px] font-medium tabular-nums leading-none text-white">
                  {displayedTransferProgress}
                </span>
              ) : undefined
            },
            { title: t('settings.update.verify') },
            { title: t('settings.update.install') },
            { title: t('settings.update.restart') }
          ]}
        />
      </div>

      {status.phase === 'uploading' && onCancelUpload && (
        <ButtonBase
          type="button"
          className="flex h-8 items-center rounded-md bg-neutral-800/80 px-2.5 text-sm text-red-400 ring-1 ring-inset ring-neutral-700/80 transition-colors enabled:hover:bg-neutral-700/80 enabled:hover:text-red-300 enabled:active:bg-neutral-700/90"
          onClick={onCancelUpload}
        >
          {t('settings.update.cancelUpload')}
        </ButtonBase>
      )}
    </div>
  );
};
