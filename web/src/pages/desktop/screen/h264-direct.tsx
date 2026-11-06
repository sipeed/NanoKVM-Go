import { useEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import * as api from '@/api/stream.ts';
import { VideoStatus } from '@/types';
import { handleAuthSessionClose } from '@/lib/auth-session.ts';
import { initialScreenCropState } from '@/lib/screen-crop.ts';
import {
  detectedScreenAreaAtom,
  portalDetectionPhaseAtom,
  portalModeAtom,
  remoteScreenAspectRatioAtom,
  screenSourceSizeAtom,
  videoParametersAtom,
  videoStatusAtom
} from '@/jotai/screen.ts';

import DirectWorker from './direct.worker.ts?worker';
import { ScreenViewport } from './screen-viewport.tsx';
import { useScreenLayout } from './use-screen-layout.ts';
import { parseStreamEvent, parseVideoStatusUpdate } from './video-status.ts';

export const H264Direct = () => {
  const videoParameters = useAtomValue(videoParametersAtom);
  const portalMode = useAtomValue(portalModeAtom);
  const currentVideoStatus = useAtomValue(videoStatusAtom);
  const setRemoteScreenAspectRatio = useSetAtom(remoteScreenAspectRatioAtom);
  const setDetectedArea = useSetAtom(detectedScreenAreaAtom);
  const setDetectionPhase = useSetAtom(portalDetectionPhaseAtom);
  const setSourceSize = useSetAtom(screenSourceSizeAtom);
  const setVideoStatus = useSetAtom(videoStatusAtom);
  const { cropReady, viewportStyle, wrapperStyle, screenScale, mediaStyle } = useScreenLayout(
    videoParameters.scale
  );

  const canvasRef = useRef<HTMLCanvasElement>(null);
  const workerRef = useRef<Worker | null>(null);
  const portalDetectionEnabledRef = useRef(
    portalMode === 'auto' && currentVideoStatus === VideoStatus.Normal
  );

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }

    setVideoStatus(VideoStatus.Normal);

    const worker = new DirectWorker();
    workerRef.current = worker;
    worker.onmessage = (event: MessageEvent) => {
      if (event.data?.type === 'screen_size_changed' && event.data.aspectRatio) {
        canvas.dataset.mediaWidth = String(event.data.width);
        canvas.dataset.mediaHeight = String(event.data.height);
        setRemoteScreenAspectRatio(Number(event.data.aspectRatio));
        setSourceSize({ width: Number(event.data.width), height: Number(event.data.height) });
      }
      if (
        portalDetectionEnabledRef.current &&
        event.data?.type === 'portal_effective_area' &&
        event.data.area
      ) {
        setDetectedArea(event.data.area);
        setDetectionPhase(event.data.phase ?? 'idle');
      }
    };

    const offscreen = canvas.transferControlToOffscreen();
    worker.postMessage(
      {
        type: 'init_h264',
        canvas: offscreen,
        enabled: portalDetectionEnabledRef.current
      },
      [offscreen]
    );

    const ws = api.directH264();
    ws.binaryType = 'arraybuffer';

    ws.onmessage = (event) => {
      if (typeof event.data === 'string') {
        const streamEvent = parseStreamEvent(event.data);
        const videoStatus = streamEvent && parseVideoStatusUpdate(streamEvent);
        if (videoStatus) {
          setVideoStatus(videoStatus.status);
        }
        return;
      }

      if (!(event.data instanceof ArrayBuffer)) {
        return;
      }

      try {
        worker.postMessage({ type: 'ws_message', data: event.data }, [event.data]);
      } catch (error) {
        console.error('Error processing WebSocket message:', error);
      }
    };

    ws.onerror = () => {
      worker.postMessage({ type: 'error' });
    };

    ws.onclose = (event) => {
      if (handleAuthSessionClose(event)) return;

      worker.postMessage({ type: 'close' });
    };

    return () => {
      workerRef.current = null;
      if (ws.readyState === 1) {
        ws.close();
      }
      worker.terminate();
      setDetectedArea(initialScreenCropState);
      setDetectionPhase('idle');
      setSourceSize({ width: 0, height: 0 });
      delete canvas.dataset.mediaWidth;
      delete canvas.dataset.mediaHeight;
      setVideoStatus(VideoStatus.Normal);
    };
  }, [
    setDetectedArea,
    setDetectionPhase,
    setRemoteScreenAspectRatio,
    setSourceSize,
    setVideoStatus
  ]);

  useEffect(() => {
    const enabled = portalMode === 'auto' && currentVideoStatus === VideoStatus.Normal;
    portalDetectionEnabledRef.current = enabled;
    workerRef.current?.postMessage({ type: 'set_portal_detection', enabled });

    if (!enabled) {
      setDetectedArea(initialScreenCropState);
      setDetectionPhase('idle');
    }
  }, [currentVideoStatus, portalMode, setDetectedArea, setDetectionPhase]);

  return (
    <ScreenViewport contentStyle={viewportStyle} screenScale={screenScale}>
      <div
        className={cropReady ? 'relative' : 'relative h-full w-full'}
        style={wrapperStyle}
        data-input-region-host="screen"
      >
        <canvas
          id="screen"
          ref={canvasRef}
          className={
            cropReady ? 'block select-none' : 'block h-full w-full select-none object-contain'
          }
          style={mediaStyle}
        ></canvas>
      </div>
    </ScreenViewport>
  );
};
