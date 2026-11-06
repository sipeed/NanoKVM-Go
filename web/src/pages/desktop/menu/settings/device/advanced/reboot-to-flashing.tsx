import { useState } from 'react';
import { Button, Modal } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

export const RebootToFlashing = () => {
  const { t } = useTranslation();
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();

  function rebootToFlashing() {
    if (isLoading || isGateBusy) return;

    void runWithChallenge('nanokvm_restart', 'nanoKVMRestart', async (challenge) => {
      let rebootScheduled = false;
      setIsLoading(true);
      try {
        const rsp = await api.rebootToFlashing(challenge);
        if (rsp.code === 0) {
          rebootScheduled = true;
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

  const label = t('settings.device.rebootToFlashing');

  return (
    <>
      <div className="flex items-center justify-between gap-4">
        <span>{label}</span>

        <Button
          danger
          type="primary"
          className="shrink-0"
          disabled={isLoading || isGateBusy}
          loading={isLoading}
          onClick={() => setIsConfirmOpen(true)}
        >
          {t('settings.device.rebootToFlashingButton')}
        </Button>
      </div>

      <Modal
        open={isConfirmOpen}
        title={t('settings.device.rebootToFlashingTitle')}
        centered={true}
        okText={t('settings.device.okBtn')}
        cancelText={t('settings.device.cancelBtn')}
        okButtonProps={{ danger: true }}
        onOk={() => {
          setIsConfirmOpen(false);
          rebootToFlashing();
        }}
        onCancel={() => setIsConfirmOpen(false)}
      >
        <p>{t('settings.device.rebootToFlashingDesc')}</p>
      </Modal>
      {protectionModal}
    </>
  );
};
