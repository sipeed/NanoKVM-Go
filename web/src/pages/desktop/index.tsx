import { useEffect } from 'react';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';
import { useMediaQuery } from 'react-responsive';

import { VideoMode } from '@/types';
import * as storage from '@/lib/localstorage.ts';
import { getSupportedVideoModes, isWebrtcMode, isWebrtcSupported } from '@/lib/video.ts';
import { client } from '@/lib/websocket.ts';
import { videoBandwidthWarningAtom, videoModeAtom, videoParametersAtom } from '@/jotai/screen.ts';
import { Head } from '@/components/head.tsx';

import { Keyboard } from './keyboard';
import { Menu } from './menu';
import { Message } from './message.tsx';
import { MicrophoneProvider } from './microphone-context.tsx';
import { Mouse } from './mouse';
import { Notification } from './notification.tsx';
import { Onboarding } from './onboarding/index.tsx';
import { OnboardingMobile } from './onboarding/mobile/index.tsx';
import { PointerModeController } from './pointer-mode-controller.tsx';
import { Screen } from './screen';
import { EdidController } from './screen/edid-controller.tsx';
import { PortalController } from './screen/portal-controller.tsx';
import { PortalSelectionOverlay } from './screen/portal-selection-overlay.tsx';
import { VirtualKeyboard } from './virtual-keyboard';
import { WebHIDInputRuntime } from './webhid-input-runtime.tsx';

export const Desktop = () => {
  const { t } = useTranslation();
  const isBigScreen = useMediaQuery({ minWidth: 850 });

  const [videoMode, setVideoMode] = useAtom(videoModeAtom);
  const bitrate = useAtomValue(videoParametersAtom).bitrate;
  const setBandwidthWarning = useSetAtom(videoBandwidthWarningAtom);

  useEffect(() => {
    const connection = (
      navigator as Navigator & { connection?: EventTarget & { downlink?: number } }
    ).connection;
    const update = () =>
      setBandwidthWarning(
        Boolean(connection?.downlink && connection.downlink * 1000 < bitrate * 1.2)
      );
    update();
    connection?.addEventListener('change', update);
    return () => {
      connection?.removeEventListener('change', update);
      setBandwidthWarning(false);
    };
  }, [bitrate, setBandwidthWarning]);

  useEffect(() => {
    client.connect();

    const mode = getVideoMode() as VideoMode;
    setVideoMode(mode);

    return () => {
      client.close();
    };
  }, []);

  function getVideoMode() {
    const supportedModes = getSupportedVideoModes();

    const cookieVideoMode = storage.getVideoMode();
    if (cookieVideoMode && supportedModes.includes(cookieVideoMode)) {
      return cookieVideoMode;
    }

    if (isWebrtcMode(cookieVideoMode) && !isWebrtcSupported()) {
      return cookieVideoMode;
    }

    if (supportedModes.includes('h264-webrtc')) {
      return 'h264-webrtc';
    }

    if (supportedModes.includes('h264-direct')) {
      return 'h264-direct';
    }

    return 'h264-webrtc';
  }

  return (
    <MicrophoneProvider>
      <PointerModeController />
      <Head title={t('head.desktop')} />

      <Message />

      {isBigScreen && <Notification />}

      {videoMode && (
        <>
          <Menu />
          <EdidController />
          <Screen />
          <PortalController />
          <PortalSelectionOverlay />
          <Mouse />
          <Keyboard />
          <Onboarding />
          <OnboardingMobile />
          <WebHIDInputRuntime />
        </>
      )}

      <VirtualKeyboard />
    </MicrophoneProvider>
  );
};
