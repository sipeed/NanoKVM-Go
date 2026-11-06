import { useState } from 'react';
import { Modal } from 'antd';
import { Trash2Icon } from 'lucide-react';

import * as api from '@/api/extensions/tailscale.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import { useRemoteNetworkMessages } from '../../messages.ts';

type UninstallProps = {
  onSuccess: () => void;
};

export const Uninstall = ({ onSuccess }: UninstallProps) => {
  const t = useRemoteNetworkMessages();
  const { runWithChallenge, modal, isGateBusy } = useProtectionChallengeGate();

  const [isLoading, setIsLoading] = useState(false);
  const [isModalOpen, setIsModalOpen] = useState(false);

  function uninstall() {
    if (isLoading || isGateBusy) return;

    void runWithChallenge('tailscale_uninstall', 'tailscale', async (challenge) => {
      setIsLoading(true);
      try {
        const rsp = await api.uninstall(challenge);
        if (rsp.code === -3) {
          return rsp;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          return rsp;
        }
        setIsModalOpen(false);
        onSuccess();
        return rsp;
      } finally {
        setIsLoading(false);
      }
    });
  }

  const title = (
    <div className="flex items-center space-x-1 text-red-500">
      <Trash2Icon size={18} />
      <span>{t('settings.tailscale.uninstall')}</span>
    </div>
  );

  return (
    <>
      <div
        className="flex h-[30px] cursor-pointer items-center space-x-1 rounded px-3 py-0.5 text-neutral-300 hover:bg-neutral-700/70"
        onClick={() => setIsModalOpen(true)}
      >
        <span>{t('settings.tailscale.uninstall')}</span>
      </div>

      <Modal
        title={title}
        open={isModalOpen}
        okType="danger"
        okText={t('settings.tailscale.okBtn')}
        cancelText={t('settings.tailscale.cancelBtn')}
        onOk={uninstall}
        onCancel={() => {
          if (isLoading || isGateBusy) return;
          setIsModalOpen(false);
        }}
        confirmLoading={isLoading || isGateBusy}
        // Rendered inside the settings Modal subtree; Ant Design's zindexContext
        // stacking keeps it above settings without an explicit zIndex.
      >
        <div className="py-5">
          <p className="text-base">{t('settings.tailscale.uninstallDesc')}</p>
        </div>
      </Modal>
      {modal}
    </>
  );
};
