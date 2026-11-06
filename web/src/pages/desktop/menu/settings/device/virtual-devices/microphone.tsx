import { useRef, useState } from 'react';
import { Segmented } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/virtual-device.ts';
import type { VirtualAudioMode } from '@/api/virtual-device.ts';
import { useCountdownConfirmation } from '@/hooks/useCountdownConfirmation.tsx';

import { useMicrophone } from '../../../../microphone-context.tsx';

interface MicrophoneProps {
  isDisabled: boolean;
  audioMode: VirtualAudioMode;
  isAudioRuntimeReady: boolean;
  getVirtualDevices: () => void | Promise<void>;
}

export const Microphone = ({
  isDisabled,
  audioMode,
  isAudioRuntimeReady,
  getVirtualDevices
}: MicrophoneProps) => {
  const { t } = useTranslation();
  const { status, disable } = useMicrophone();
  const { confirm: confirmVirtualAudioUpdate, contextHolder } = useCountdownConfirmation({
    title: t('settings.device.virtualAudioConfirm.title'),
    description: t('settings.device.virtualAudioConfirm.description'),
    ok: t('settings.device.virtualAudioConfirm.ok'),
    okCountdown: (seconds) =>
      t('settings.device.virtualAudioConfirm.okCountdown', {
        seconds
      }),
    cancel: t('settings.device.virtualAudioConfirm.cancel')
  });

  const [isLoading, setIsLoading] = useState(false);
  const statusRef = useRef(status);
  statusRef.current = status;

  async function update(nextMode: VirtualAudioMode) {
    if (isLoading) return;
    setIsLoading(true);

    try {
      const current = await api.getVirtualDevice();
      if (current.code !== 0 || !current.data || !api.isVirtualAudioMode(current.data.audioMode)) {
        return;
      }

      if (
        current.data.audioMode === nextMode &&
        api.isVirtualAudioRuntimeReady(
          nextMode,
          current.data.isMicEnabled,
          current.data.isSpeakerEnabled
        )
      ) {
        return;
      }

      const wasTransmitting = statusRef.current !== 'disabled';
      await disable();
      if (wasTransmitting) {
        await new Promise((resolve) => setTimeout(resolve, 1000));
      }

      const rsp = await api.updateVirtualAudioMode(nextMode);
      if (rsp.code !== 0) {
        return;
      }
    } finally {
      try {
        await getVirtualDevices();
      } finally {
        setIsLoading(false);
      }
    }
  }

  function requestUpdate(nextMode: VirtualAudioMode) {
    const speakerStateChanges = (audioMode === 'duplex') !== (nextMode === 'duplex');
    if (speakerStateChanges) {
      confirmVirtualAudioUpdate(() => update(nextMode));
      return;
    }

    void update(nextMode);
  }

  function repairCurrentMode(mode: VirtualAudioMode) {
    if (mode !== audioMode || isAudioRuntimeReady || isDisabled || isLoading) return;
    requestUpdate(mode);
  }

  const options = [
    {
      value: 'off' as const,
      label: (
        <span onClick={() => repairCurrentMode('off')}>
          {t('settings.device.virtualAudioMode.off')}
        </span>
      )
    },
    {
      value: 'mic' as const,
      label: (
        <span onClick={() => repairCurrentMode('mic')}>
          {t('settings.device.virtualAudioMode.mic')}
        </span>
      )
    },
    {
      value: 'duplex' as const,
      label: (
        <span onClick={() => repairCurrentMode('duplex')}>
          {t('settings.device.virtualAudioMode.duplex')}
        </span>
      )
    }
  ];

  return (
    <>
      <div className="flex w-full items-center justify-between">
        <div className="flex flex-col">
          <span>{t('settings.device.mic')}</span>
          <span className="text-xs text-neutral-500">{t('settings.device.micDesc')}</span>
        </div>

        <Segmented<VirtualAudioMode>
          disabled={isDisabled || isLoading}
          value={audioMode}
          options={options}
          onChange={requestUpdate}
        />
      </div>
      {contextHolder}
    </>
  );
};
