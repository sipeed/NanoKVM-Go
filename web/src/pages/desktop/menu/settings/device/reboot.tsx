import { useState } from 'react';
import { ReloadOutlined } from '@ant-design/icons';
import { Button, Popconfirm } from 'antd';
import clsx from 'clsx';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

export const Reboot = ({ disable = false }: { disable?: boolean }) => {
  const { t } = useTranslation();

  const [isLoading, setIsLoading] = useState(false);
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();

  function reboot() {
    if (disable || isLoading || isGateBusy) return;

    void runWithChallenge('nanokvm_restart', 'nanoKVMRestart', async (challenge) => {
      let rebootScheduled = false;
      setIsLoading(true);
      try {
        const rsp = await api.reboot(challenge);
        if (rsp.code === 0) {
          rebootScheduled = true;
          window.setTimeout(() => {
            window.location.reload();
          }, 30000);
        } else {
          console.log(rsp.msg);
        }
        return rsp;
      } catch (err) {
        console.log(err);
        throw err;
      } finally {
        if (!rebootScheduled) setIsLoading(false);
      }
    });
  }

  return (
    <div className={clsx('flex justify-center pt-1', disable && 'cursor-not-allowed opacity-50')}>
      <Popconfirm
        placement="bottom"
        disabled={disable || isLoading || isGateBusy}
        title={t('settings.device.rebootDesc')}
        okText={t('settings.device.okBtn')}
        cancelText={t('settings.device.cancelBtn')}
        onConfirm={reboot}
      >
        <Button
          danger
          type="primary"
          size="large"
          shape="round"
          disabled={disable || isLoading || isGateBusy}
          loading={isLoading}
          icon={<ReloadOutlined />}
        >
          {t('settings.device.reboot')}
        </Button>
      </Popconfirm>
      {protectionModal}
    </div>
  );
};
