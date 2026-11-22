import { http } from '@/lib/http.ts';

export type VirtualAudioMode = 'off' | 'mic' | 'duplex';
type ToggleVirtualDevice = 'network' | 'disk';

export const VIRTUAL_AUDIO_MODE_CHANGED_EVENT = 'nanokvm:virtual-audio-mode-changed';

export function isVirtualAudioMode(value: unknown): value is VirtualAudioMode {
  return value === 'off' || value === 'mic' || value === 'duplex';
}

export function isVirtualAudioRuntimeReady(
  mode: VirtualAudioMode,
  isMicEnabled: boolean,
  isSpeakerEnabled: boolean
) {
  if (mode === 'off') return !isMicEnabled && !isSpeakerEnabled;
  if (mode === 'mic') return isMicEnabled && !isSpeakerEnabled;
  return isMicEnabled && isSpeakerEnabled;
}

// get virtual devices status
export function getVirtualDevice() {
  return http.get('/api/vm/device/virtual');
}

// toggle virtual devices that still use legacy on/off behavior
export function updateVirtualDevice(device: ToggleVirtualDevice, diskSizeMiB?: number) {
  const data = {
    device,
    ...(diskSizeMiB === undefined ? {} : { diskSizeMiB })
  };

  return http.post('/api/vm/device/virtual', data);
}

export function disableVirtualDisk() {
  return http.post('/api/vm/device/virtual/disk/disable');
}

export async function updateVirtualAudioMode(mode: VirtualAudioMode) {
  const rsp = await http.post('/api/vm/device/virtual', {
    device: 'audio',
    mode
  });

  if (rsp.code === 0 && typeof window !== 'undefined') {
    window.dispatchEvent(new Event(VIRTUAL_AUDIO_MODE_CHANGED_EVENT));
  }

  return rsp;
}

// refresh virtual device
export function refreshVirtualDevice(device: string) {
  const data = {
    device
  };

  return http.post('/api/vm/device/virtual/refresh', data);
}

export const WEBCAM_CHANGED = 'nanokvm:webcam-changed';

export async function setWebcamEnabled(enabled: boolean) {
  const rsp = await http.post('/api/vm/device/virtual', {
    device: 'webcam',
    mode: enabled ? 'on' : 'off'
  });
  window.dispatchEvent(new Event(WEBCAM_CHANGED));
  return rsp;
}
