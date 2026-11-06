import { useCallback, useEffect, useRef, useState } from 'react';
import { Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { MemoryFabricState } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';

import { MemoryFabricConfigPanel } from './memory-fabric-config.tsx';
import { getMemoryFabricRemoteAccess } from './memory-fabric-remote-access.ts';

type MemoryFabricProps = {
  mcpEnabled: boolean;
  isServiceToggleLocked: boolean;
  runServiceToggle: (action: () => Promise<void>) => Promise<void>;
};

export const MemoryFabric = ({
  mcpEnabled,
  isServiceToggleLocked,
  runServiceToggle
}: MemoryFabricProps) => {
  const { t } = useTranslation();

  const [state, setState] = useState<MemoryFabricState>({
    enabled: false
  });
  const [isLoading, setIsLoading] = useState(false);
  const isLoadingRef = useRef(false);
  const silentRefreshInFlightRef = useRef(false);
  const stateActionVersionRef = useRef(0);
  const [isConfigSaving, setIsConfigSaving] = useState(false);

  const updateLoading = useCallback((loading: boolean) => {
    isLoadingRef.current = loading;
    setIsLoading(loading);
  }, []);

  const getState = useCallback(
    (silent = false) => {
      if (silent) {
        if (isLoadingRef.current || silentRefreshInFlightRef.current) {
          return;
        }
        silentRefreshInFlightRef.current = true;
      } else {
        updateLoading(true);
      }
      const actionVersion = stateActionVersionRef.current;

      api
        .getMemoryFabricState()
        .then((rsp) => {
          if (rsp.code !== 0) {
            if (!silent) {
              console.log(rsp.msg);
              feedback.error(t('settings.ai.errors.loadMemoryFabricState'));
            }
            return;
          }

          if (silent && actionVersion !== stateActionVersionRef.current) {
            return;
          }

          setState(rsp.data);
        })
        .catch(() => {
          if (!silent) {
            feedback.error(t('settings.ai.errors.loadMemoryFabricState'));
          }
        })
        .finally(() => {
          if (silent) {
            silentRefreshInFlightRef.current = false;
          } else {
            updateLoading(false);
          }
        });
    },
    [t, updateLoading]
  );

  useEffect(() => {
    getState();

    const timer = window.setInterval(() => getState(true), 3000);

    return () => window.clearInterval(timer);
  }, [getState]);

  function setEnabled(enabled: boolean) {
    if (isLoadingRef.current || isConfigSaving || isServiceToggleLocked) return;

    void runServiceToggle(async () => {
      stateActionVersionRef.current += 1;
      updateLoading(true);

      try {
        const rsp = await api.setMemoryFabricEnabled(enabled);
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.updateMemoryFabricService'));
          return;
        }

        setState(rsp.data);
      } catch {
        feedback.error(t('settings.ai.errors.updateMemoryFabricService'));
      } finally {
        updateLoading(false);
      }
    });
  }

  const remoteAccess = getMemoryFabricRemoteAccess(mcpEnabled, state.enabled);

  return (
    <div className="flex flex-col space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex flex-col space-y-1">
          <span className="text-sm font-medium">{t('settings.ai.memoryFabricService')}</span>
          <span className="text-xs text-neutral-500">
            {t('settings.ai.memoryFabricServiceDesc')}
          </span>
        </div>

        <Switch
          checked={state.enabled}
          disabled={isConfigSaving || isServiceToggleLocked}
          loading={isLoading}
          onChange={setEnabled}
        />
      </div>

      {remoteAccess === 'requiresMcp' && (
        <div className="rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-200">
          {t('settings.ai.memoryFabricRemoteRequiresMcp')}
        </div>
      )}

      {remoteAccess === 'available' && (
        <div className="rounded-lg border border-green-500/20 bg-green-500/10 px-3 py-2 text-xs text-green-200">
          {t('settings.ai.memoryFabricRemoteAvailable')}
        </div>
      )}

      {state.enabled && (
        <>
          <MemoryFabricConfigPanel disabled={isLoading} onSavingChange={setIsConfigSaving} />
        </>
      )}
    </div>
  );
};
