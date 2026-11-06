import { useState } from 'react';
import { UserSwitchOutlined } from '@ant-design/icons';
import { Button, Card } from 'antd';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';

type LoginProps = {
  onSuccess: () => void;
};

export const Login = ({ onSuccess }: LoginProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();

  const [isLoading, setIsLoading] = useState(false);
  const [loginUrl, setLoginUrl] = useState('');
  const [errMsg, setErrMsg] = useState('');

  function login() {
    if (isLoading || isGateBusy) return;

    void runWithChallenge('tailscale_login', 'tailscale', async (challenge) => {
      setIsLoading(true);
      setErrMsg('');
      try {
        const rsp = await api.login(challenge);
        if (rsp.code === -3) {
          return rsp;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          setErrMsg(t('settings.tailscale.failed'));
          return rsp;
        }

        const url = rsp.data?.url;
        if (!url) {
          onSuccess();
          return rsp;
        }

        setLoginUrl(url);
        window.open(url, '_blank', 'noopener,noreferrer');
        setTimeout(() => setLoginUrl(''), 10 * 60 * 1000);
        return rsp;
      } finally {
        setIsLoading(false);
      }
    });
  }

  return (
    <>
      <div className="flex flex-col items-center justify-center space-y-10">
        <Card>{t('settings.tailscale.notLogin')}</Card>

        {loginUrl === '' ? (
          <Button
            type="primary"
            size="large"
            shape="round"
            icon={<UserSwitchOutlined />}
            loading={isLoading || isGateBusy}
            onClick={login}
          >
            {t('settings.tailscale.login')}
          </Button>
        ) : (
          <div className="flex w-full flex-col items-center justify-center space-y-5">
            <Button type="link" href={loginUrl} target="_blank" rel="noopener noreferrer">
              {loginUrl}
            </Button>

            <span className="text-xs text-neutral-600">{t('settings.tailscale.urlPeriod')}</span>

            <Button type="primary" size="large" shape="round" onClick={onSuccess}>
              {t('settings.tailscale.loginSuccess')}
            </Button>
          </div>
        )}

        {errMsg && <span className="text-red-500">{errMsg}</span>}
      </div>
      {modal}
    </>
  );
};
