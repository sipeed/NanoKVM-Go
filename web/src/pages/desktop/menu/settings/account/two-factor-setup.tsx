import { useState } from 'react';
import { Button, Input } from 'antd';
import { QRCodeSVG } from 'qrcode.react';
import { useTranslation } from 'react-i18next';

import { type TwoFactorSetupData } from '@/api/auth.ts';

type TwoFactorSetupProps = {
  setup: TwoFactorSetupData;
  code: string;
  error?: string;
  isVerifying?: boolean;
  isCancelling?: boolean;
  isLocked?: boolean;
  onCodeChange: (code: string) => void;
  onVerify: () => void;
  onCancel: () => void;
};

export const TwoFactorSetup = ({
  setup,
  code,
  error,
  isVerifying = false,
  isCancelling = false,
  isLocked = false,
  onCodeChange,
  onVerify,
  onCancel
}: TwoFactorSetupProps) => {
  const { t } = useTranslation();
  const isBusy = isVerifying || isCancelling || isLocked;
  const [isSetupKeyVisible, setIsSetupKeyVisible] = useState(false);

  return (
    <div className="space-y-4 rounded border border-neutral-700 bg-neutral-800/40 p-4">
      <div className="text-sm text-neutral-300">{t('settings.account.scanTwoFactorQRCode')}</div>

      <div className="flex justify-center">
        <div className="flex h-56 w-56 shrink-0 items-center justify-center rounded bg-white p-3">
          <QRCodeSVG
            value={setup.uri}
            size={192}
            level="M"
            bgColor="#ffffff"
            fgColor="#111827"
            marginSize={4}
            title={t('settings.account.scanTwoFactorQRCode')}
          />
        </div>
      </div>

      <div className="space-y-2">
        <Button
          type="link"
          className="h-auto max-w-full whitespace-normal p-0 text-left text-xs"
          disabled={isBusy}
          onClick={() => setIsSetupKeyVisible((visible) => !visible)}
        >
          {isSetupKeyVisible
            ? t('settings.account.hideSetupKey')
            : t('settings.account.showSetupKey')}
        </Button>

        {isSetupKeyVisible && (
          <label className="block space-y-2">
            <span className="text-xs text-neutral-500">{t('settings.account.setupKey')}</span>
            <Input value={setup.secret} readOnly={true} className="font-mono text-xs" />
          </label>
        )}
      </div>

      <Input
        value={code}
        disabled={isBusy}
        autoComplete="one-time-code"
        placeholder={t('settings.account.authCode')}
        onChange={(event) => onCodeChange(event.target.value)}
        onPressEnter={onVerify}
      />

      <div className="min-h-5 text-sm text-red-500" aria-live="polite">
        {error}
      </div>

      <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
        <Button
          className="w-full sm:w-auto"
          disabled={isBusy}
          loading={isCancelling}
          onClick={onCancel}
        >
          {t('settings.account.cancelBtn')}
        </Button>
        <Button
          type="primary"
          className="w-full sm:w-auto"
          disabled={!code.trim() || isBusy}
          loading={isVerifying}
          onClick={onVerify}
        >
          {t('settings.account.verifyAndEnable')}
        </Button>
      </div>
    </div>
  );
};
