import { useEffect, useRef } from 'react';
import { useAtomValue, useSetAtom } from 'jotai';

import { getInputProfileConfig, setInputProfileConfig } from '@/api/vm.ts';
import { normalizeInputProfileStore } from '@/lib/input-profile.ts';
import type { InputProfileStore } from '@/lib/input-profile.ts';
import { hydrateInputProfileStoreAtom, inputProfileStoreAtom } from '@/jotai/input-profile.ts';

type InputProfileConfigResponse = InputProfileStore & { configured: boolean };

export const InputProfileController = () => {
  const localStore = useAtomValue(inputProfileStoreAtom);
  const hydrateStore = useSetAtom(hydrateInputProfileStoreAtom);
  const loadedRef = useRef(false);

  useEffect(() => {
    if (loadedRef.current) return;
    loadedRef.current = true;

    void getInputProfileConfig()
      .then(async (rsp) => {
        if (rsp.code !== 0 || !rsp.data) return;
        const response = rsp.data as InputProfileConfigResponse;
        if (response.configured) {
          hydrateStore(normalizeInputProfileStore(response));
          return;
        }

        const hasLocalProfiles = localStore.profiles.some((profile) => profile.origin === 'user');
        if (hasLocalProfiles) {
          await setInputProfileConfig(localStore);
        }
      })
      .catch(() => undefined);
  }, [hydrateStore, localStore]);

  return null;
};
