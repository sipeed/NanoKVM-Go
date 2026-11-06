import { useState } from 'react';
import { DownloadOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { Button, Card, Result } from 'antd';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';

type InstallProps = {
  isInstalling: boolean;
  onBusyChange: (isBusy: boolean) => void;
  onSuccess: () => void | Promise<void>;
};

export const Install = ({ isInstalling, onBusyChange, onSuccess }: InstallProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();

  const [state, setState] = useState('');
  const isBusy = isInstalling || state === 'installing' || isGateBusy;

  function install() {
    if (isBusy) return;

    void runWithChallenge('tailscale_install', 'tailscale', async (challenge) => {
      setState('installing');
      onBusyChange(true);

      try {
        const rsp = await api.install(challenge);
        // Let the gate reopen the 2FA modal; do not treat as install failure.
        if (rsp.code === -3) {
          return rsp;
        }
        if (rsp.code !== 0) {
          setState('failed');
          return rsp;
        }
        await onSuccess();
        setState('');
        return rsp;
      } catch {
        setState('failed');
        return { code: -1 };
      } finally {
        onBusyChange(false);
        setState((prev) => (prev === 'installing' ? '' : prev));
      }
    });
  }

  if (state === 'failed') {
    return (
      <>
        <Result
          status="warning"
          title={t('settings.tailscale.failed')}
          subTitle={t('settings.tailscale.retry')}
          icon={<InfoCircleOutlined />}
          extra={
            <div className="flex w-full flex-col items-center gap-4">
              <Button type="primary" onClick={() => setState('')}>
                {t('settings.tailscale.retry')}
              </Button>
              <Card key="tips" styles={{ body: { padding: 0 } }} className="w-full max-w-md">
                <ul className="list-decimal text-left font-mono text-sm text-neutral-300">
                  <li>
                    {t('settings.tailscale.download')}
                    <a className="px-1" href="https://pkgs.tailscale.com/stable/" target="_blank">
                      {t('settings.tailscale.package')}
                    </a>
                    {t('settings.tailscale.unzip')}
                  </li>
                  <li>{t('settings.tailscale.upTailscale')}</li>
                  <li>{t('settings.tailscale.upTailscaled')}</li>
                  <li>{t('settings.tailscale.refresh')}</li>
                </ul>
              </Card>
            </div>
          }
        />
        {modal}
      </>
    );
  }

  return (
    <>
      <Card>
        <Result
          icon={<DownloadOutlined />}
          subTitle={t('settings.tailscale.notInstall')}
          extra={
            <Button
              key="install"
              type="primary"
              loading={isBusy}
              disabled={isBusy}
              onClick={install}
            >
              {isInstalling || state === 'installing'
                ? t('settings.tailscale.installing')
                : t('settings.tailscale.install')}
            </Button>
          }
        />
      </Card>
      {modal}
    </>
  );
};
