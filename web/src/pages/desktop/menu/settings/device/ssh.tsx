import { useEffect, useState } from 'react';
import { Switch, Tooltip } from 'antd';
import { CircleAlertIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

export const Ssh = () => {
  const { t } = useTranslation();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();

  const [isEnabled, setIsEnabled] = useState(false);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    setIsLoading(true);

    api
      .getSSHState()
      .then((rsp) => {
        if (rsp.data?.enabled) {
          setIsEnabled(true);
        }
      })
      .finally(() => {
        setIsLoading(false);
      });
  }, []);

  async function update() {
    if (isLoading || isGateBusy) return;

    if (isEnabled) {
      await runWithChallenge('disable_ssh', 'ssh', async (challenge) => {
        setIsLoading(true);
        try {
          const rsp = await api.disableSSH(challenge);
          if (rsp.code === -3) {
            return rsp;
          }
          if (rsp.code !== 0) {
            console.log(rsp.msg);
            return rsp;
          }
          setIsEnabled(false);
          return rsp;
        } finally {
          setIsLoading(false);
        }
      });
      return;
    }

    await runWithChallenge('enable_ssh', 'ssh', async (challenge) => {
      setIsLoading(true);
      try {
        const rsp = await api.enableSSH(challenge);
        if (rsp.code === -3) {
          return rsp;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return rsp;
        }
        setIsEnabled(true);
        return rsp;
      } finally {
        setIsLoading(false);
      }
    });
  }

  return (
    <>
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <div className="flex items-center space-x-2">
            <span>SSH</span>

            <Tooltip
              title={t('settings.device.ssh.tip')}
              className="cursor-pointer text-neutral-500"
              placement="top"
              styles={{ root: { maxWidth: '400px' } }}
            >
              <CircleAlertIcon size={15} />
            </Tooltip>
          </div>
          <span className="text-xs text-neutral-500">{t('settings.device.ssh.description')}</span>
        </div>

        <Switch
          checked={isEnabled}
          loading={isLoading || isGateBusy}
          disabled={isLoading || isGateBusy}
          onChange={update}
        />
      </div>
      {modal}
    </>
  );
};
