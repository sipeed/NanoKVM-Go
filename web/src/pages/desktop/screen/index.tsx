import { lazy, Suspense, type ComponentType, type LazyExoticComponent } from 'react';
import { Flex, Result } from 'antd';
import { useAtomValue } from 'jotai';
import { useTranslation } from 'react-i18next';

import { isMockedDevelopment } from '@/lib/runtime-mode.ts';
import { isWebrtcMode, isWebrtcSupported } from '@/lib/video.ts';
import { videoModeAtom } from '@/jotai/screen.ts';

import { H264Direct } from './h264-direct.tsx';
import { H264Webrtc } from './h264-webrtc.tsx';
import { H265Direct } from './h265-direct.tsx';
import { H265Webrtc } from './h265-webrtc.tsx';
import { NoSignalScreen } from './no-signal-screen.tsx';

let mockedScreen: LazyExoticComponent<ComponentType> | undefined;

function getMockedScreen() {
  if (!mockedScreen) {
    mockedScreen = lazy(() =>
      import('@/mocks/mock-screen.tsx').then((module) => ({ default: module.MockScreen }))
    );
  }
  return mockedScreen;
}

export const Screen = () => {
  const { t } = useTranslation();
  const videoMode = useAtomValue(videoModeAtom);
  const webrtcSupported = isWebrtcSupported();

  let activeScreen;

  if (import.meta.env.DEV && isMockedDevelopment(import.meta.env.MODE, import.meta.env.DEV)) {
    const MockedScreen = getMockedScreen();
    activeScreen = (
      <Suspense fallback={null}>
        <MockedScreen />
      </Suspense>
    );
  } else if (videoMode === 'h265-direct') {
    activeScreen = <H265Direct />;
  } else if (videoMode === 'h264-direct') {
    activeScreen = <H264Direct />;
  } else if (isWebrtcMode(videoMode) && !webrtcSupported) {
    activeScreen = (
      <Flex className="h-screen w-screen" align="center" justify="center" role="alert">
        <Result
          status="warning"
          title={t('screen.webrtcUnavailable.title')}
          subTitle={t('screen.webrtcUnavailable.description')}
        />
      </Flex>
    );
  } else if (videoMode === 'h265-webrtc') {
    activeScreen = <H265Webrtc />;
  } else {
    activeScreen = <H264Webrtc />;
  }

  return (
    <>
      {activeScreen}
      <NoSignalScreen />
    </>
  );
};
