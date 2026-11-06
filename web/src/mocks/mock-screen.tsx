import { useEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { VideoStatus } from '@/types';
import { initialScreenCropState } from '@/lib/screen-crop.ts';
import {
  detectedScreenAreaAtom,
  remoteScreenAspectRatioAtom,
  screenSourceSizeAtom,
  videoParametersAtom,
  videoStatusAtom
} from '@/jotai/screen.ts';
import { ScreenViewport } from '@/pages/desktop/screen/screen-viewport.tsx';
import { useScreenLayout } from '@/pages/desktop/screen/use-screen-layout.ts';

const mockScreenWidth = 1920;
const mockScreenHeight = 1080;

export const MockScreen = () => {
  const videoParameters = useAtomValue(videoParametersAtom);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const setVideoStatus = useSetAtom(videoStatusAtom);
  const setRemoteScreenAspectRatio = useSetAtom(remoteScreenAspectRatioAtom);
  const setDetectedArea = useSetAtom(detectedScreenAreaAtom);
  const setSourceSize = useSetAtom(screenSourceSizeAtom);
  const { cropReady, viewportStyle, wrapperStyle, screenScale, mediaStyle } = useScreenLayout(
    videoParameters.scale
  );

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext('2d');
    if (context) {
      const gradient = context.createLinearGradient(0, 0, mockScreenWidth, mockScreenHeight);
      gradient.addColorStop(0, '#111827');
      gradient.addColorStop(0.5, '#1d4ed8');
      gradient.addColorStop(1, '#0f172a');
      context.fillStyle = gradient;
      context.fillRect(0, 0, mockScreenWidth, mockScreenHeight);
      context.strokeStyle = 'rgba(255, 255, 255, 0.16)';
      context.lineWidth = 2;
      for (let x = 0; x <= mockScreenWidth; x += 120) {
        context.beginPath();
        context.moveTo(x, 0);
        context.lineTo(x, mockScreenHeight);
        context.stroke();
      }
      for (let y = 0; y <= mockScreenHeight; y += 120) {
        context.beginPath();
        context.moveTo(0, y);
        context.lineTo(mockScreenWidth, y);
        context.stroke();
      }
      context.fillStyle = 'rgba(255, 255, 255, 0.9)';
      context.font = '600 48px sans-serif';
      context.fillText('NanoKVM mocked screen', 72, 120);
      context.fillStyle = 'rgba(255, 255, 255, 0.7)';
      context.font = '28px monospace';
      context.fillText('1920 × 1080 • deterministic development fixture', 72, 168);
    }

    setVideoStatus(VideoStatus.Normal);
    setRemoteScreenAspectRatio(mockScreenWidth / mockScreenHeight);
    setSourceSize({ width: mockScreenWidth, height: mockScreenHeight });
    setDetectedArea({
      sourceWidth: mockScreenWidth,
      sourceHeight: mockScreenHeight,
      contentRect: {
        left: 0,
        top: 0,
        width: mockScreenWidth,
        height: mockScreenHeight
      },
      contentAspectRatio: mockScreenWidth / mockScreenHeight,
      updatedAt: Date.now()
    });

    return () => {
      setDetectedArea(initialScreenCropState);
      setSourceSize({ width: 0, height: 0 });
      setVideoStatus(VideoStatus.Normal);
    };
  }, [setDetectedArea, setRemoteScreenAspectRatio, setSourceSize, setVideoStatus]);

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
          width={mockScreenWidth}
          height={mockScreenHeight}
          className={
            cropReady ? 'block select-none' : 'block h-full w-full select-none object-contain'
          }
          style={mediaStyle}
        />
      </div>
    </ScreenViewport>
  );
};
