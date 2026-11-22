import { useCallback, useEffect } from 'react';
import { atom, useAtomValue, useStore } from 'jotai';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import { feedback } from '@/lib/feedback.tsx';

type Mode = 'relative' | 'absolute';
type Command = 'refresh' | 'toggle' | 'disable' | Mode;
type Config = { enabled: boolean; mode: Mode };

const configAtom = atom<Config | null>(null);
const busyAtom = atom(false);

export function useMouseJiggler() {
  const store = useStore();
  const config = useAtomValue(configAtom);
  const busy = useAtomValue(busyAtom);
  const { t } = useTranslation();

  const update = useCallback(
    async (command: Command) => {
      if (store.get(busyAtom)) return;
      store.set(busyAtom, true);

      try {
        let current = store.get(configAtom);
        if (!current || command === 'refresh') {
          const rsp = await api.getMouseJiggler();
          if (rsp.code !== 0 || typeof rsp.data?.enabled !== 'boolean') {
            throw new Error(rsp.msg);
          }

          // The device reports relative mode while idle. Keep the last active
          // mode for a quick off/on cycle in this session.
          current = {
            enabled: rsp.data.enabled,
            mode:
              !rsp.data.enabled && current
                ? current.mode
                : rsp.data.mode === 'absolute'
                  ? 'absolute'
                  : 'relative'
          };
          store.set(configAtom, current);
        }
        if (command === 'refresh') return;

        const enabled = command === 'toggle' ? !current.enabled : command !== 'disable';
        const mode = command === 'absolute' || command === 'relative' ? command : current.mode;
        const rsp = await api.setMouseJiggler(enabled, mode);
        if (rsp.code !== 0) throw new Error(rsp.msg);
        store.set(configAtom, { enabled, mode });
      } catch {
        feedback.error(t('error.title'), { key: 'mouse-jiggler' });
      } finally {
        store.set(busyAtom, false);
      }
    },
    [store, t]
  );

  useEffect(() => {
    void update('refresh');
  }, [update]);

  return { config, busy, update };
}
