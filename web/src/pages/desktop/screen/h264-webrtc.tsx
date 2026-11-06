import { MutableRefObject, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { Spin } from 'antd';
import clsx from 'clsx';
import { useAtom, useAtomValue, useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/stream.ts';
import { VideoStatus } from '@/types';
import { handleAuthSessionClose } from '@/lib/auth-session.ts';
import {
  remoteScreenAspectRatioAtom,
  videoParametersAtom,
  videoStatusAtom,
  videoVolumeAtom,
  webrtcConnectionFailedAtom
} from '@/jotai/screen.ts';

import { useMicrophone } from '../microphone-context.tsx';
import {
  attachAudioStream,
  disposeAudioVolumeSynchronization,
  resolveVolumeAfterAutoplayBlock,
  synchronizeAudioVolume
} from './audio-volume.ts';
import { ScreenViewport } from './screen-viewport.tsx';
import {
  usePortalDetectionFromElement,
  useScreenLayout,
  useScreenSourceSize
} from './use-screen-layout.ts';
import { parseStreamEvent, parseVideoStatusUpdate, type StreamEvent } from './video-status.ts';

const createAnswerHandler = (
  connection: RTCPeerConnection,
  offerSentRef: MutableRefObject<boolean>,
  candidatesRef: MutableRefObject<RTCIceCandidate[]>,
  type: 'video' | 'audio'
) => {
  return (data: any) => {
    if (connection.signalingState !== 'have-local-offer') {
      offerSentRef.current = false;
      console.warn(`${type} signaling state incorrect for answer: ${connection.signalingState}`);
      return;
    }

    connection
      .setRemoteDescription(new RTCSessionDescription(data))
      .then(() => {
        offerSentRef.current = false;
        candidatesRef.current.forEach((candidate) => {
          connection
            .addIceCandidate(candidate)
            .catch((e) => console.error(`${type} candidate failed to add:`, e.message));
        });
        candidatesRef.current = [];
      })
      .catch((error) => {
        console.error(`${type} answer set failed:`, error);
        offerSentRef.current = false;
      });
  };
};

const createCandidateHandler = (
  connection: RTCPeerConnection,
  candidatesRef: MutableRefObject<RTCIceCandidate[]>,
  type: 'video' | 'audio'
) => {
  return (data: any) => {
    if (!data.candidate) {
      return;
    }

    const candidate = new RTCIceCandidate(data);
    if (connection.remoteDescription) {
      connection
        .addIceCandidate(candidate)
        .catch((e) => console.error(`${type} candidate failed to add:`, e.message));
    } else {
      candidatesRef.current.push(candidate);
    }
  };
};

export const H264Webrtc = () => {
  const { t } = useTranslation();
  const videoParameters = useAtomValue(videoParametersAtom);
  const setVideoStatus = useSetAtom(videoStatusAtom);
  const setWebrtcConnectionFailed = useSetAtom(webrtcConnectionFailedAtom);
  const setRemoteScreenAspectRatio = useSetAtom(remoteScreenAspectRatioAtom);
  const [volume, setVolume] = useAtom(videoVolumeAtom);
  const { registerAudioTransport } = useMicrophone();
  const { cropReady, viewportStyle, wrapperStyle, screenScale, mediaStyle } = useScreenLayout(
    videoParameters.scale
  );

  const [isPlaying, setIsPlaying] = useState(true);
  const [isLoading, setIsLoading] = useState(true);

  const videoRef = useRef<HTMLVideoElement | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const volumeRef = useRef(volume);
  const videoOfferSent = useRef(false);
  const audioOfferSent = useRef(false);

  const videoIceCandidates = useRef<RTCIceCandidate[]>([]);
  const audioIceCandidates = useRef<RTCIceCandidate[]>([]);

  usePortalDetectionFromElement(videoRef);
  useScreenSourceSize(videoRef);

  function updateRemoteScreenAspectRatio() {
    const video = videoRef.current;
    if (!video?.videoWidth || !video.videoHeight) return;
    setRemoteScreenAspectRatio(video.videoWidth / video.videoHeight);
  }

  useEffect(() => {
    setVideoStatus(VideoStatus.Normal);
    setWebrtcConnectionFailed(false);

    let disposed = false;
    let unregisterAudioTransport: (() => void) | null = null;
    const audioElement = audioRef.current;

    const ws = api.webrtcH264();

    const iceServers = [{ urls: ['stun:stun.l.google.com:19302'] }];
    const video = new RTCPeerConnection({ iceServers });
    const audio = new RTCPeerConnection({ iceServers });

    const handleVideoAnswer = createAnswerHandler(
      video,
      videoOfferSent,
      videoIceCandidates,
      'video'
    );
    const handleAudioAnswer = createAnswerHandler(
      audio,
      audioOfferSent,
      audioIceCandidates,
      'audio'
    );
    const handleVideoCandidate = createCandidateHandler(video, videoIceCandidates, 'video');
    const handleAudioCandidate = createCandidateHandler(audio, audioIceCandidates, 'audio');

    const sendMsg = (event: string, data: string) => {
      if (ws.readyState !== WebSocket.OPEN) {
        return;
      }

      try {
        const message: StreamEvent = { event, data };
        ws.send(JSON.stringify(message));
      } catch (err) {
        console.error('Error sending event:', err);
      }
    };

    // --- Init Video ---
    video.onnegotiationneeded = async () => {
      if (videoOfferSent.current || video.signalingState !== 'stable') {
        console.log('Skipping video negotiation - Waiting for answer or state unstable');
        return;
      }

      try {
        videoOfferSent.current = true;
        const offer = await video.createOffer({
          offerToReceiveVideo: true,
          offerToReceiveAudio: false
        });

        await video.setLocalDescription(offer);
        sendMsg('video-offer', JSON.stringify(video.localDescription));
      } catch (error) {
        videoOfferSent.current = false;
        console.error('Video negotiation failed:', error);
      }
    };

    video.onconnectionstatechange = () => {
      if (disposed) return;

      if (video.connectionState === 'connected') {
        clearTimeout(loadingTimer);
        setWebrtcConnectionFailed(false);
        setIsLoading(false);
      } else if (video.connectionState === 'failed') {
        clearTimeout(loadingTimer);
        setWebrtcConnectionFailed(true);
        setIsLoading(false);
      }
    };

    video.ontrack = (event) => {
      if (videoRef.current && event.track.kind === 'video') {
        videoRef.current.srcObject = new MediaStream([event.track]);
      }
    };

    // --- Init Audio ---
    audio.onnegotiationneeded = async () => {
      if (audioOfferSent.current || audio.signalingState !== 'stable') {
        console.log('Skipping audio negotiation - Waiting for answer or state unstable');
        return;
      }

      try {
        audioOfferSent.current = true;
        const offer = await audio.createOffer({
          offerToReceiveVideo: false,
          offerToReceiveAudio: true
        });

        await audio.setLocalDescription(offer);
        sendMsg('audio-offer', JSON.stringify(audio.localDescription));
      } catch (error) {
        audioOfferSent.current = false;
        console.error('Audio negotiation failed:', error);
      }
    };

    audio.ontrack = (event) => {
      if (!disposed && event.track.kind === 'audio') {
        const attemptedVolume = volumeRef.current;
        void attachAudioStream(audioElement, new MediaStream([event.track]), attemptedVolume).then(
          (result) => {
            if (disposed || result !== 'blocked') return;

            setVolume((currentVolume) => {
              const nextVolume = resolveVolumeAfterAutoplayBlock(currentVolume, attemptedVolume);
              volumeRef.current = nextVolume;
              return nextVolume;
            });
          }
        );
      }
    };

    // --- WebSocket Message Handling ---
    ws.onopen = () => {
      if (disposed) {
        ws.close();
        return;
      }

      videoOfferSent.current = false;
      audioOfferSent.current = false;

      video.onicecandidate = (event) => {
        if (event.candidate) {
          sendMsg('video-candidate', JSON.stringify(event.candidate));
        }
      };

      audio.onicecandidate = (event) => {
        if (event.candidate) {
          sendMsg('audio-candidate', JSON.stringify(event.candidate));
        }
      };

      video.addTransceiver('video', { direction: 'recvonly' });
      const audioTransceiver = audio.addTransceiver('audio', { direction: 'sendrecv' });
      unregisterAudioTransport = registerAudioTransport(audio, audioTransceiver.sender);
    };

    ws.onmessage = (event) => {
      try {
        const msg = parseStreamEvent(event.data);
        if (!msg) return;

        const videoStatus = parseVideoStatusUpdate(msg);
        if (videoStatus) {
          setVideoStatus(videoStatus.status);
          setIsPlaying(videoStatus.isPlaying);
          setIsLoading(false);
          return;
        }

        if (!msg.data) return;

        const data = JSON.parse(msg.data);
        if (!data) return;

        switch (msg.event) {
          case 'video-answer':
            handleVideoAnswer(data);
            break;
          case 'video-candidate':
            handleVideoCandidate(data);
            break;
          case 'audio-answer':
            handleAudioAnswer(data);
            break;
          case 'audio-candidate':
            handleAudioCandidate(data);
            break;
          case 'heartbeat':
            break;
          default:
            console.log('Unhandled event:', msg.event);
        }
      } catch (err) {
        console.error('Message processing error:', err);
      }
    };

    ws.onclose = (event) => {
      unregisterAudioTransport?.();
      unregisterAudioTransport = null;
      handleAuthSessionClose(event);
    };

    const heartbeatTimer = setInterval(() => {
      sendMsg('heartbeat', '');
    }, 60 * 1000);

    const loadingTimer = setTimeout(() => {
      setWebrtcConnectionFailed(video.connectionState !== 'connected');
      setIsLoading(false);
    }, 10 * 1000);

    return () => {
      disposed = true;
      unregisterAudioTransport?.();
      unregisterAudioTransport = null;
      disposeAudioVolumeSynchronization(audioElement);

      if (ws.readyState === WebSocket.CONNECTING || ws.readyState === WebSocket.OPEN) {
        ws.close();
      }

      video.close();
      audio.close();

      videoOfferSent.current = false;
      audioOfferSent.current = false;

      clearInterval(heartbeatTimer);
      clearTimeout(loadingTimer);
      setVideoStatus(VideoStatus.Normal);
      setWebrtcConnectionFailed(false);
    };
  }, [registerAudioTransport, setVideoStatus, setVolume, setWebrtcConnectionFailed]);

  useLayoutEffect(() => {
    let active = true;
    const attemptedVolume = volume;
    volumeRef.current = volume;
    void synchronizeAudioVolume(audioRef.current, attemptedVolume).then((result) => {
      if (!active || result !== 'blocked') return;

      setVolume((currentVolume) => {
        const nextVolume = resolveVolumeAfterAutoplayBlock(currentVolume, attemptedVolume);
        volumeRef.current = nextVolume;
        return nextVolume;
      });
    });

    return () => {
      active = false;
    };
  }, [setVolume, volume]);

  return (
    <Spin
      size="large"
      tip={t('screen.loading')}
      spinning={isLoading}
      wrapperClassName="nanokvm-screen-loading"
    >
      <ScreenViewport contentStyle={viewportStyle} screenScale={screenScale}>
        <div
          className={cropReady ? 'relative' : 'relative h-full w-full'}
          style={wrapperStyle}
          data-input-region-host="screen"
        >
          <video
            id="screen"
            ref={videoRef}
            className={clsx(
              cropReady ? 'block select-none' : 'block h-full w-full select-none object-contain',
              isPlaying ? 'opacity-100' : 'opacity-0'
            )}
            style={mediaStyle}
            muted
            autoPlay
            playsInline
            controls={false}
            onLoadedMetadata={updateRemoteScreenAspectRatio}
            onPlaying={() => {
              setIsLoading(false);
              updateRemoteScreenAspectRatio();
            }}
            onResize={updateRemoteScreenAspectRatio}
            onClick={(e) => {
              e.stopPropagation();
              e.preventDefault();
            }}
          />
        </div>

        <audio ref={audioRef} muted autoPlay playsInline />
      </ScreenViewport>
    </Spin>
  );
};
