import { useAtomValue } from 'jotai';

import { VideoStatus } from '@/types';
import { NO_SIGNAL_SCREEN_ID } from '@/lib/no-signal-screen.ts';
import {
  edidResolutionAtom,
  portalBorderColorAtom,
  portalBorderEnabledAtom,
  videoStatusAtom
} from '@/jotai/screen.ts';
import { useResponsiveDevice } from '@/hooks/useResponsiveDevice.ts';

export const NoSignalScreen = () => {
  const videoStatus = useAtomValue(videoStatusAtom);
  const resolution = useAtomValue(edidResolutionAtom);
  const borderEnabled = useAtomValue(portalBorderEnabledAtom);
  const borderColor = useAtomValue(portalBorderColorAtom);
  const { isMobileDevice } = useResponsiveDevice();

  if (videoStatus !== VideoStatus.NoImage) return null;

  const aspectRatio = resolution.width / resolution.height;
  const borderClearance = borderEnabled ? 12 : 0;

  return (
    <div
      className={`${isMobileDevice ? 'absolute' : 'fixed'} pointer-events-none inset-0 z-10 flex items-center justify-center bg-black`}
    >
      <div
        id={NO_SIGNAL_SCREEN_ID}
        className="relative h-screen w-screen bg-black"
        data-input-region-host="no-signal"
        data-source-height={resolution.height}
        data-source-width={resolution.width}
        style={{
          width: `calc(100dvw - ${borderClearance}px)`,
          height: `calc(100dvh - ${borderClearance}px)`,
          maxHeight: `calc((100dvw - ${borderClearance}px) / ${aspectRatio})`,
          maxWidth: `calc((100dvh - ${borderClearance}px) * ${aspectRatio})`,
          outline: borderEnabled ? `3px solid ${borderColor}` : 'none',
          outlineOffset: '2px'
        }}
      />
    </div>
  );
};
