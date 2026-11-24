import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode
} from 'react';
import { Modal } from 'antd';
import { useAtom, useAtomValue } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as virtualDeviceApi from '@/api/virtual-device.ts';
import { feedback } from '@/lib/feedback.tsx';
import {
  getMicrophoneErrorTranslationKey,
  hasAudioInputDevice,
  MicrophoneError,
  requestControlMicrophone,
  stopMediaStream
} from '@/lib/microphone.ts';
import { microphoneStatusAtom, type MicrophoneStatus } from '@/jotai/audio.ts';
import { videoModeAtom } from '@/jotai/screen.ts';

type AudioTransport = {
  id: symbol;
  connection: RTCPeerConnection;
  sender: RTCRtpSender;
  unregister: () => void;
};

type MicrophoneContextValue = {
  status: MicrophoneStatus;
  canEnable: boolean;
  isChecking: boolean;
  requestEnable: (beforeDialog?: () => void) => Promise<void>;
  disable: () => Promise<void>;
  registerAudioTransport: (connection: RTCPeerConnection, sender: RTCRtpSender) => () => void;
};

const MicrophoneContext = createContext<MicrophoneContextValue | null>(null);
const microphoneMessageKey = 'control-microphone';

type MicrophoneProviderProps = {
  children: ReactNode;
};

export const MicrophoneProvider = ({ children }: MicrophoneProviderProps) => {
  const { t } = useTranslation();
  const videoMode = useAtomValue(videoModeAtom);
  const [status, setStatus] = useAtom(microphoneStatusAtom);
  const [transportReady, setTransportReady] = useState(false);

  const [dialog, setDialog] = useState<'mount' | 'missing' | null>(null);
  const [isChecking, setIsChecking] = useState(false);
  const checking = useRef(false);
  const mountedRef = useRef(true);
  const statusRef = useRef<MicrophoneStatus>(status);
  const generationRef = useRef(0);
  const transportRef = useRef<AudioTransport | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const trackRef = useRef<MediaStreamTrack | null>(null);
  const senderQueueRef = useRef<Promise<void>>(Promise.resolve());
  const tRef = useRef(t);

  useEffect(() => {
    statusRef.current = status;
  }, [status]);

  useEffect(() => {
    tRef.current = t;
  }, [t]);

  const updateStatus = useCallback(
    (nextStatus: MicrophoneStatus) => {
      statusRef.current = nextStatus;
      if (mountedRef.current) {
        setStatus(nextStatus);
      }
    },
    [setStatus]
  );

  const notifyError = useCallback((error: unknown) => {
    const key = getMicrophoneErrorTranslationKey(error);
    feedback.error(tRef.current(key), { key: microphoneMessageKey });
  }, []);

  const notifyDisconnected = useCallback(() => {
    feedback.error(tRef.current('microphone.disconnected'), { key: microphoneMessageKey });
  }, []);

  const enqueueSenderOperation = useCallback((operation: () => Promise<void>) => {
    const result = senderQueueRef.current.then(operation, operation);
    senderQueueRef.current = result.catch(() => undefined);
    return result;
  }, []);

  const clearCapture = useCallback((stream: MediaStream | null, track: MediaStreamTrack | null) => {
    if (streamRef.current === stream) {
      streamRef.current = null;
    }
    if (trackRef.current === track) {
      trackRef.current = null;
    }
    if (track) {
      track.onended = null;
    }
    stopMediaStream(stream);
  }, []);

  const disable = useCallback(async () => {
    generationRef.current += 1;
    checking.current = false;
    setIsChecking(false);
    setDialog(null);
    updateStatus('disabled');

    const stream = streamRef.current;
    const track = trackRef.current;

    streamRef.current = null;
    trackRef.current = null;

    clearCapture(stream, track);
  }, [clearCapture, updateStatus]);

  const ensureVirtualMicrophoneMounted = useCallback(async (isCurrent: () => boolean) => {
    const current = await virtualDeviceApi.getVirtualDevice();
    if (!isCurrent()) return false;
    if (current.code !== 0 || !current.data) {
      throw new MicrophoneError('microphone.enableFailed');
    }
    if (!virtualDeviceApi.isVirtualAudioMode(current.data.audioMode)) {
      throw new MicrophoneError('microphone.enableFailed');
    }
    if (
      current.data.audioMode !== 'off' &&
      virtualDeviceApi.isVirtualAudioRuntimeReady(
        current.data.audioMode,
        current.data.isMicEnabled,
        current.data.isSpeakerEnabled
      )
    ) {
      return true;
    }

    const targetMode = current.data.audioMode === 'off' ? 'mic' : current.data.audioMode;
    const updated = await virtualDeviceApi.updateVirtualAudioMode(targetMode);
    if (updated.code !== 0) {
      throw new MicrophoneError('microphone.enableFailed');
    }
    if (!isCurrent()) return false;

    const verified = await virtualDeviceApi.getVirtualDevice();
    if (verified.code !== 0 || !verified.data?.isMicEnabled) {
      throw new MicrophoneError('microphone.enableFailed');
    }
    return isCurrent();
  }, []);

  const enable = useCallback(
    async () => {
      if (statusRef.current === 'enabled') return true;
      if (statusRef.current === 'enabling') return false;

      const transport = transportRef.current;
      if (!transport || transport.connection.signalingState === 'closed') {
        notifyError(new MicrophoneError('microphone.enableFailed'));
        return false;
      }

      const generation = generationRef.current + 1;
      generationRef.current = generation;
      updateStatus('enabling');

      const isCurrent = () =>
        mountedRef.current &&
        generationRef.current === generation &&
        transportRef.current?.id === transport.id &&
        transport.connection.signalingState !== 'closed';

      let stream: MediaStream | null = null;
      let track: MediaStreamTrack | null = null;

      try {
        const capture = await requestControlMicrophone();
        stream = capture.stream;
        track = capture.track;

        if (!isCurrent()) {
          clearCapture(stream, track);
          if (generationRef.current === generation) {
            updateStatus('disabled');
          }
          return false;
        }

        streamRef.current = stream;
        trackRef.current = track;
        track.onended = () => {
          if (generationRef.current !== generation || trackRef.current !== track) return;
          void disable();
          notifyDisconnected();
        };

        const mounted = await ensureVirtualMicrophoneMounted(isCurrent);
        if (!mounted || !isCurrent()) {
          clearCapture(stream, track);
          if (generationRef.current === generation) {
            updateStatus('disabled');
          }
          return false;
        }

        await enqueueSenderOperation(async () => {
          if (!isCurrent()) return;
          await transport.sender.replaceTrack(track);
        });

        if (!isCurrent() || track.readyState === 'ended' || transport.sender.track !== track) {
          clearCapture(stream, track);
          if (generationRef.current === generation) {
            updateStatus('disabled');
          }
          return false;
        }

        updateStatus('enabled');
        return true;
      } catch (error) {
        console.error('Failed to enable microphone:', error);

        clearCapture(stream, track);

        if (generationRef.current === generation) {
          updateStatus('disabled');
          notifyError(error);
        }
        return false;
      }
    },
    [
      clearCapture,
      disable,
      enqueueSenderOperation,
      ensureVirtualMicrophoneMounted,
      notifyDisconnected,
      notifyError,
      updateStatus
    ]
  );

  const registerAudioTransport = useCallback(
    (connection: RTCPeerConnection, sender: RTCRtpSender) => {
      let registered = true;
      const transport: AudioTransport = {
        id: Symbol('microphone-audio-transport'),
        connection,
        sender,
        unregister: () => undefined
      };

      const unregister = () => {
        if (!registered) return;
        registered = false;
        connection.removeEventListener('connectionstatechange', handleConnectionStateChange);
        connection.removeEventListener('signalingstatechange', handleConnectionStateChange);

        if (transportRef.current?.id !== transport.id) return;

        transportRef.current = null;
        if (mountedRef.current) {
          setTransportReady(false);
        }
        void disable();
      };

      const handleConnectionStateChange = () => {
        if (
          connection.connectionState === 'closed' ||
          connection.connectionState === 'failed' ||
          connection.signalingState === 'closed'
        ) {
          unregister();
        }
      };

      transport.unregister = unregister;
      transportRef.current?.unregister();

      transportRef.current = transport;
      connection.addEventListener('connectionstatechange', handleConnectionStateChange);
      connection.addEventListener('signalingstatechange', handleConnectionStateChange);
      if (mountedRef.current) {
        setTransportReady(true);
      }

      return unregister;
    },
    [disable]
  );

  useEffect(() => {
    const mediaDevices = typeof navigator === 'undefined' ? null : navigator.mediaDevices;
    if (!mediaDevices?.addEventListener) return;

    const handleDeviceChange = () => {
      const track = trackRef.current;
      if (!track || track.readyState !== 'ended') return;

      void disable();
      notifyDisconnected();
    };

    mediaDevices.addEventListener('devicechange', handleDeviceChange);
    return () => {
      mediaDevices.removeEventListener('devicechange', handleDeviceChange);
    };
  }, [disable, notifyDisconnected]);

  const isWebRTCMode = videoMode === 'h264-webrtc' || videoMode === 'h265-webrtc';
  const canEnable = isWebRTCMode && transportReady;

  useEffect(() => {
    if (!isWebRTCMode && statusRef.current !== 'disabled') {
      void disable();
    }
  }, [disable, isWebRTCMode]);

  useEffect(() => {
    mountedRef.current = true;
    updateStatus('disabled');

    return () => {
      void disable();
      transportRef.current?.unregister();
      transportRef.current = null;
      setTransportReady(false);
      mountedRef.current = false;
    };
  }, [disable, updateStatus]);

  const requestEnable = useCallback(async (beforeDialog?: () => void) => {
    if (statusRef.current !== 'disabled' || checking.current || dialog) return;
    if (!canEnable) {
      notifyError(new MicrophoneError('microphone.enableFailed'));
      return;
    }
    const generation = generationRef.current;
    checking.current = true;
    setIsChecking(true);
    try {
      const hasDevice = await hasAudioInputDevice();
      if (generation !== generationRef.current) return;
      if (!hasDevice) {
        beforeDialog?.();
        setDialog('missing');
        return;
      }
      const rsp = await virtualDeviceApi.getVirtualDevice();
      if (generation !== generationRef.current) return;
      if (rsp.code !== 0 || !rsp.data) throw new MicrophoneError('microphone.enableFailed');
      if (!rsp.data.isMicEnabled) {
        beforeDialog?.();
        setDialog('mount');
        return;
      }
    } catch (error) {
      if (generation === generationRef.current) notifyError(error);
      return;
    } finally {
      if (generation === generationRef.current) {
        checking.current = false;
        setIsChecking(false);
      }
    }
    await enable();
  }, [canEnable, dialog, enable, notifyError]);

  const value = useMemo<MicrophoneContextValue>(
    () => ({ status, canEnable, isChecking, requestEnable, disable, registerAudioTransport }),
    [status, canEnable, isChecking, requestEnable, disable, registerAudioTransport]
  );

  return (
    <MicrophoneContext.Provider value={value}>
      {children}
      <Modal
        title={t(dialog === 'mount' ? 'microphone.notMounted' : 'microphone.title')}
        open={dialog !== null}
        onOk={() => {
          if (dialog === 'mount') void enable();
          setDialog(null);
        }}
        onCancel={() => setDialog(null)}
        okText={t(dialog === 'mount' ? 'microphone.confirm' : 'auth.ok')}
        cancelText={t('microphone.cancel')}
        cancelButtonProps={dialog === 'missing' ? { style: { display: 'none' } } : undefined}
      >
        {dialog === 'mount' ? (
          <div className="flex flex-col space-y-3 py-3">
            <span>{t('microphone.tip1')}</span>
            <span>{t('microphone.tip2')}</span>
          </div>
        ) : (
          <div className="py-3">{t('microphone.noDevice')}</div>
        )}
      </Modal>
    </MicrophoneContext.Provider>
  );
};

// Keep the provider and its consumer hook together so they share one private context.
// eslint-disable-next-line react-refresh/only-export-components
export function useMicrophone() {
  const context = useContext(MicrophoneContext);
  if (!context) {
    throw new Error('useMicrophone must be used within MicrophoneProvider');
  }
  return context;
}
