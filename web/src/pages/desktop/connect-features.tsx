import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from 'react';
import { useAtomValue, useSetAtom, useStore } from 'jotai';

import { feedback } from '@/lib/feedback.tsx';
import { getSavedVideoVolume, getUnmuteVolume } from '@/lib/localstorage.ts';
import { isWebrtcMode } from '@/lib/video.ts';
import { client } from '@/lib/websocket.ts';
import { videoModeAtom, videoVolumeAtom } from '@/jotai/screen.ts';
import { connectPreferencesAtom, connectStartsAtom, refreshConnectStartsAtom } from '@/jotai/settings.ts';
import { useMouseJiggler } from '@/hooks/useMouseJiggler.ts';

import { useWebcamShare } from './menu/webcam.tsx';
import { useMicrophone } from './microphone-context.tsx';

const WebcamContext = createContext<ReturnType<typeof useWebcamShare> | null>(null);

// A readiness change may release a pending start. After an attempt, only a
// new connection or explicitly rechecking the preference should rearm it.
function useConnectStart(session: number, checked: boolean, ready: boolean, start: () => void) {
  const attempted = useRef(0);
  useEffect(() => {
    if (!checked) {
      attempted.current = 0;
      return;
    }
    if (!session || !ready || attempted.current === session) return;
    attempted.current = session;
    start();
  }, [session, checked, ready, start]);
}

export function ConnectFeatures({ children }: { children: ReactNode }) {
  const store = useStore();
  const starts = useAtomValue(connectStartsAtom);
  const preferences = useAtomValue(connectPreferencesAtom);
  const videoMode = useAtomValue(videoModeAtom);
  const setVolume = useSetAtom(videoVolumeAtom);
  const microphone = useMicrophone();
  const { disable } = microphone;
  const webcam = useWebcamShare();
  const jiggler = useMouseJiggler(false);
  const [session, setSession] = useState(0);
  const sequence = useRef(0);

  useEffect(() => {
    function open() {
      if (!store.set(refreshConnectStartsAtom).volume) setVolume(0);
      setSession(++sequence.current);
    }
    const unsubscribe = client.onConnectionState((state) => {
      if (state === 'open') open();
      if (state === 'closed') {
        setSession(0);
        void disable();
      }
    });
    if (client.isConnected) open();
    return unsubscribe;
  }, [disable, setVolume, store]);

  useConnectStart(session, starts.volume, preferences.volume && isWebrtcMode(videoMode), () => {
    setVolume(getSavedVideoVolume() || getUnmuteVolume());
  });
  useConnectStart(session, starts.microphone, preferences.microphone && microphone.canEnable, () => {
    void microphone.requestEnable();
  });
  useConnectStart(session, starts.webcam, preferences.webcam && webcam.enabled, () => {
    void webcam.start('', (message) => feedback.error(message, { key: 'connect-webcam' }));
  });
  useConnectStart(session, starts.mouseJiggler, preferences.mouseJiggler, () => void jiggler.update('enable'));

  return <WebcamContext.Provider value={webcam}>{children}</WebcamContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useDesktopWebcam() {
  const webcam = useContext(WebcamContext);
  if (!webcam) throw new Error('Webcam controls require ConnectFeatures');
  return webcam;
}
