import { atom } from 'jotai';

import { VideoMode, VideoParameters, VideoStatus } from '@/types';
import * as localstorage from '@/lib/localstorage.ts';
import {
  defaultPortalConfig,
  initialScreenCropState,
  PortalConfig,
  PortalDetectionPhase,
  PortalMode,
  ScreenCropState
} from '@/lib/screen-crop.ts';
import { inputCalibrationScreenAreaAtom } from '@/jotai/input-profile.ts';

export type { PortalMode } from '@/lib/screen-crop.ts';

export const videoModeAtom = atom<VideoMode | null>(null);

export const videoParametersAtom = atom<VideoParameters>({
  rateControlMode: 'vbr', // cbr | vbr
  bitrate: 8000, // 1000 - 20000
  gop: 50, // 1 - 300
  fps: 0, // 0 - 120
  scale: 1
});

export const videoStatusAtom = atom<VideoStatus>(VideoStatus.Normal);
export const videoBandwidthWarningAtom = atom(false);
export const webrtcConnectionFailedAtom = atom(false);

export const edidResolutionAtom = atom({ width: 1920, height: 1080 });

export const screenViewportPanAtom = atom({ x: 0, y: 0 });

export const videoVolumeAtom = atom(0);

export const remoteScreenAspectRatioAtom = atom(0);

export const portalConfigAtom = atom<PortalConfig>({
  ...defaultPortalConfig,
  mode: localstorage.getPortalMode()
});

export const portalModeAtom = atom(
  (get) => get(portalConfigAtom).mode,
  (get, set, mode: PortalMode) => {
    set(portalConfigAtom, { ...get(portalConfigAtom), mode });
  }
);

export const portalSelectingAtom = atom(false);

export const portalDetectionPhaseAtom = atom<PortalDetectionPhase>('idle');

export const screenSourceSizeAtom = atom({ width: 0, height: 0 });

export const detectedScreenAreaAtom = atom<ScreenCropState>(initialScreenCropState);

export const manualScreenAreaAtom = atom<ScreenCropState>(initialScreenCropState);

export const portalInputScreenAreaAtom = atom<ScreenCropState>(initialScreenCropState);

export const portalInputActiveAtom = atom(false);

export const portalBorderEnabledAtom = atom(localstorage.getPortalBorderEnabled());

export const portalBorderColorAtom = atom(localstorage.getPortalBorderColor());

export const effectiveScreenAreaAtom = atom((get) => {
  if (get(videoStatusAtom) === VideoStatus.NoImage) {
    return initialScreenCropState;
  }

  const calibrationScreenArea = get(inputCalibrationScreenAreaAtom);
  if (calibrationScreenArea) return calibrationScreenArea;

  if (get(portalInputActiveAtom)) {
    return get(portalInputScreenAreaAtom);
  }

  const mode = get(portalModeAtom);
  const selecting = get(portalSelectingAtom);

  if (mode === 'off' || selecting) {
    return initialScreenCropState;
  }

  return mode === 'auto' ? get(detectedScreenAreaAtom) : get(manualScreenAreaAtom);
});
