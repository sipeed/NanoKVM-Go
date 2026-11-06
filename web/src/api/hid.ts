import { http } from '@/lib/http.ts';
import type { PointerMode } from '@/jotai/mouse.ts';

export const POINTER_MODES = ['absolute', 'digitizer-stylus', 'touch-screen'] as const;

export type PointerModeStatusResponse = {
  mode: PointerMode | 'unknown';
  switching: boolean;
  degraded?: boolean;
  error?: string;
};

export type HidApiResponse<T> = {
  code: number;
  msg: string;
  data: T;
};

// paste
export function paste(content: string) {
  return http.post('/api/hid/paste', { content });
}

// reset hid
export function reset() {
  return http.post('/api/hid/reset');
}

// get hid mode
export function getHidMode() {
  return http.get('/api/hid/mode');
}

// set hid mode
export function setHidMode(mode: string) {
  const data = {
    mode
  };
  return http.post('/api/hid/mode', data);
}

export function getPointerMode() {
  return http.get('/api/hid/pointer-mode') as Promise<HidApiResponse<PointerModeStatusResponse>>;
}

export function setPointerMode(mode: PointerMode) {
  return http.post('/api/hid/pointer-mode', { mode }) as Promise<
    HidApiResponse<PointerModeStatusResponse>
  >;
}

// get shortcuts
export function getShortcuts() {
  return http.get('/api/hid/shortcuts');
}

// add shortcut
export function addShortcut(keys: any[]) {
  const data = {
    keys
  };
  return http.post('/api/hid/shortcut', data);
}

// delete shortcut
export function deleteShortcut(id: string) {
  const data = {
    id
  };
  return http.delete('/api/hid/shortcut', data);
}

// get shortcut leader key
export function getLeaderKey() {
  return http.get('/api/hid/shortcut/leader-key');
}

// set shortcut leader key
export function setLeaderKey(key: string) {
  return http.post('/api/hid/shortcut/leader-key', { key });
}
