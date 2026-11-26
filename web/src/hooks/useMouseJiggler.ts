import { useCallback, useEffect } from 'react';
import { atom, useAtomValue, useStore } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { pointerModeDeviceStatusAtom } from '@/jotai/mouse.ts';
import { feedback } from '@/lib/feedback.tsx';

type Mode = 'relative' | 'absolute';
type Hours = { maxHours: number };
type Command = 'refresh' | 'enable' | 'toggle' | 'disable' | Mode | Hours;
type Config = {
  enabled: boolean;
  mode: Mode;
  maxHours: number;
  remainingMs: number;
  elapsedMs: number;
  sampledAt: number;
  blockedByPointerMode?: boolean;
};

const configAtom = atom<Config | null>(null);
const writerAtom = atom<{ done: Promise<void> } | null>(null);
const draftAtom = atom<Hours | null>(null);
const requestAtom = atom(0);

export function useMouseJiggler(observe = true) {
  const store = useStore();
  const config = useAtomValue(configAtom);
  const pointer = useAtomValue(pointerModeDeviceStatusAtom);
  const blocked = pointer.ready
    ? !!pointer.singleEndpoint && (pointer.mode === 'touch-screen' || pointer.mode === 'digitizer-stylus')
    : !!config?.blockedByPointerMode;
  const writer = useAtomValue(writerAtom);
  const draft = useAtomValue(draftAtom);
  const { t } = useTranslation();

  const update = useCallback(
    async (command: Command) => {
      const edit = typeof command === 'object' ? command : null;
      if (edit) store.set(draftAtom, edit);
      let previous;
      while ((previous = store.get(writerAtom))) {
        if (command === 'refresh') return;
        await previous.done;
      }
      const request = store.get(requestAtom) + 1;
      store.set(requestAtom, request);

      async function send() {
        try {
          const current = store.get(configAtom);
          const rsp =
            command === 'refresh'
              ? await api.getMouseJiggler()
              : await api.setMouseJiggler(
                  typeof command === 'object'
                    ? command
                    : command === 'relative' || command === 'absolute'
                      ? { enabled: true, mode: command }
                      : { enabled: command === 'enable' || (command === 'toggle' && !current?.enabled) }
                );
          if (store.get(requestAtom) !== request) return;
          if (rsp.code === -3) {
            feedback.error(t('settings.device.mouseJiggler.noMouse'), { key: 'mouse-jiggler' });
            return;
          }
          if (rsp.code !== 0 || typeof rsp.data?.enabled !== 'boolean') throw new Error(rsp.msg);
          store.set(configAtom, { ...rsp.data, sampledAt: performance.now() });
        } catch {
          feedback.error(t('error.title'), { key: 'mouse-jiggler' });
        } finally {
          if (edit && store.get(draftAtom) === edit) store.set(draftAtom, null);
          if (command !== 'refresh') store.set(writerAtom, null);
        }
      }
      const done = Promise.resolve().then(send);
      if (command !== 'refresh') store.set(writerAtom, { done });
      await done;
    },
    [store, t]
  );

  useEffect(() => {
    if (!observe) return;
    const refresh = () => void update('refresh');
    refresh();
    window.addEventListener('focus', refresh);
    return () => window.removeEventListener('focus', refresh);
  }, [observe, update]);

  useEffect(() => {
    if (!observe || !config?.enabled || !config.maxHours) return;
    const timer = window.setTimeout(() => void update('refresh'), Math.min(config.remainingMs + 100, 2_147_483_647));
    return () => window.clearTimeout(timer);
  }, [config, observe, update]);

  return { config, blocked, busy: !!writer, hours: draft?.maxHours ?? config?.maxHours, editingHours: draft !== null, update };
}
