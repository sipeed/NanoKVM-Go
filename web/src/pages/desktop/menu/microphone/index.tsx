import { useState } from 'react';
import { Modal, Tooltip } from 'antd';
import clsx from 'clsx';
import { LoaderCircleIcon, MicIcon, MicOffIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/virtual-device.ts';
import { feedback } from '@/lib/feedback.tsx';
import { hasAudioInputDevice } from '@/lib/microphone.ts';
import { useDismissMobileMenu } from '@/components/mobile-menu-context.ts';

import { useMicrophone } from '../../microphone-context.tsx';

type MicrophoneProps = {
  tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
};

export const Microphone = ({ tooltipPlacement = 'bottom' }: MicrophoneProps) => {
  const { t } = useTranslation();
  const dismissMobileMenu = useDismissMobileMenu();
  const { status, canEnable, enable, disable } = useMicrophone();

  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isNoDeviceModalOpen, setIsNoDeviceModalOpen] = useState(false);
  const [isChecking, setIsChecking] = useState(false);
  const [isTooltipOpen, setIsTooltipOpen] = useState(false);

  const toggleMicrophone = async () => {
    setIsTooltipOpen(false);

    if (status === 'enabled' || status === 'enabling') {
      await disable();
      return;
    }
    if (isChecking) return;

    setIsChecking(true);
    try {
      const hasMicrophone = await hasAudioInputDevice();
      if (!hasMicrophone) {
        dismissMobileMenu();
        setIsNoDeviceModalOpen(true);
        return;
      }

      if (!canEnable) {
        feedback.error(t('microphone.enableFailed'), { key: 'control-microphone' });
        return;
      }

      const rsp = await api.getVirtualDevice();
      if (rsp.code !== 0 || !rsp.data) {
        feedback.error(t('microphone.enableFailed'), { key: 'control-microphone' });
        return;
      }

      if (rsp.data.isMicEnabled) {
        await enable({ ensureVirtualMounted: true });
      } else {
        dismissMobileMenu();
        setIsModalOpen(true);
      }
    } catch (err) {
      console.log(err);
      feedback.error(t('microphone.enableFailed'), { key: 'control-microphone' });
    } finally {
      setIsChecking(false);
    }
  };

  const handleConfirm = async () => {
    if (status === 'enabling') return;

    await enable({ ensureVirtualMounted: true });
    setIsModalOpen(false);
  };

  const handleCancel = () => {
    void disable();
    setIsModalOpen(false);
  };

  const isEnabled = status === 'enabled';
  const isEnabling = status === 'enabling';
  const isLoading = isChecking || isEnabling;

  return (
    <>
      <Tooltip
        title={t('microphone.title')}
        placement={tooltipPlacement}
        mouseEnterDelay={0.6}
        open={isTooltipOpen}
        onOpenChange={setIsTooltipOpen}
      >
        <div
          className={clsx(
            'flex h-[30px] w-[30px] cursor-pointer items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white',
            isChecking && 'pointer-events-none opacity-50'
          )}
          aria-disabled={isChecking}
          onClick={() => void toggleMicrophone()}
        >
          {isLoading ? (
            <LoaderCircleIcon size={18} className="animate-spin" />
          ) : isEnabled ? (
            <MicIcon size={18} className="text-green-400" />
          ) : (
            <MicOffIcon size={18} />
          )}
        </div>
      </Tooltip>

      <Modal
        title={t('microphone.notMounted')}
        open={isModalOpen}
        onOk={handleConfirm}
        onCancel={handleCancel}
        okText={t('microphone.confirm')}
        cancelText={t('microphone.cancel')}
        confirmLoading={isLoading}
      >
        <div className="flex flex-col space-y-3 py-3">
          <span>{t('microphone.tip1')}</span>
          <span>{t('microphone.tip2')}</span>
        </div>
      </Modal>

      <Modal
        title={t('microphone.title')}
        open={isNoDeviceModalOpen}
        onOk={() => setIsNoDeviceModalOpen(false)}
        onCancel={() => setIsNoDeviceModalOpen(false)}
        okText={t('auth.ok')}
        cancelButtonProps={{ style: { display: 'none' } }}
      >
        <div className="py-3">
          <span>{t('microphone.noDevice')}</span>
        </div>
      </Modal>
    </>
  );
};
