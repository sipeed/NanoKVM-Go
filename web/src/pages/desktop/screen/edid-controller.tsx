import { useEffect } from 'react';
import { useSetAtom } from 'jotai';

import * as api from '@/api/vm.ts';
import { validScreenResolution } from '@/lib/no-signal-screen.ts';
import { edidResolutionAtom } from '@/jotai/screen.ts';

export const EdidController = () => {
  const setEdidResolution = useSetAtom(edidResolutionAtom);

  useEffect(() => {
    const load = async () => {
      try {
        const rsp = await api.getEdid();
        const resolution = validScreenResolution(rsp.data?.width, rsp.data?.height);
        if (rsp.code === 0 && resolution) {
          setEdidResolution(resolution);
        }
      } catch (error) {
        console.log(error);
      }
    };

    void load();
  }, [setEdidResolution]);

  return null;
};
