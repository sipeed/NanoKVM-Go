import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Input, Switch } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/application.ts';

const OFFICIAL_UPDATE_SERVER = 'https://cdn.sipeed.com/nanokvm-go/v2';

type UpdateServerProps = {
  checkForUpdates: () => void;
  disabled: boolean;
  onBusyChange: (busy: boolean) => void;
  onPendingChange: (pending: boolean) => void;
};

export const UpdateServer = ({
  checkForUpdates,
  disabled,
  onBusyChange,
  onPendingChange
}: UpdateServerProps) => {
  const { t } = useTranslation();
  const [server, setServer] = useState(OFFICIAL_UPDATE_SERVER);
  const [savedServer, setSavedServer] = useState<string | null>(null);
  const [isEnabled, setIsEnabled] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const requestIdRef = useRef(0);

  const load = useCallback(async () => {
    const requestId = ++requestIdRef.current;
    setIsLoading(true);
    setError('');
    onBusyChange(true);
    try {
      const rsp = await api.getUpdateServer();
      if (requestId !== requestIdRef.current) return;
      if (rsp.code !== 0 || typeof rsp.data?.server !== 'string') {
        throw new Error('invalid update server response');
      }
      setServer(rsp.data.server || OFFICIAL_UPDATE_SERVER);
      setSavedServer(rsp.data.server);
      setIsEnabled(rsp.data.server !== '');
    } catch {
      if (requestId === requestIdRef.current) setError('loadFailed');
    } finally {
      if (requestId === requestIdRef.current) {
        setIsLoading(false);
        onBusyChange(false);
      }
    }
  }, [onBusyChange]);

  useEffect(() => {
    void load();
    return () => {
      requestIdRef.current += 1;
      onBusyChange(false);
    };
  }, [load, onBusyChange]);

  async function save(value: string) {
    if (disabled || isLoading || savedServer === null) return;
    const requestId = ++requestIdRef.current;
    let changed = false;
    setIsLoading(true);
    setError('');
    onBusyChange(true);
    try {
      const rsp = await api.setUpdateServer(value.trim());
      if (requestId !== requestIdRef.current) return;
      if (rsp.code !== 0 || typeof rsp.data?.server !== 'string') {
        setError(rsp.code === -1 ? 'invalidURL' : 'saveFailed');
        return;
      }
      if (rsp.data.server) setServer(rsp.data.server);
      setSavedServer(rsp.data.server);
      setIsEnabled(rsp.data.server !== '');
      changed = true;
    } catch {
      if (requestId === requestIdRef.current) setError('saveFailed');
    } finally {
      if (requestId === requestIdRef.current) {
        setIsLoading(false);
        onBusyChange(false);
        if (changed) checkForUpdates();
      }
    }
  }

  const unavailable = disabled || isLoading || savedServer === null;
  const hasChanges =
    savedServer !== null && (isEnabled ? server.trim() !== savedServer : savedServer !== '');

  useEffect(() => {
    onPendingChange(hasChanges);
  }, [hasChanges, onPendingChange]);

  useEffect(() => () => onPendingChange(false), [onPendingChange]);

  function toggleEnabled(enabled: boolean) {
    if (unavailable) return;
    setError('');
    if (!enabled && savedServer) {
      void save('');
      return;
    }
    setIsEnabled(enabled);
  }

  function requestSave() {
    if (unavailable || !isEnabled || !hasChanges) return;
    if (!server.trim()) {
      setError('invalidURL');
      return;
    }
    void save(server);
  }

  return (
    <div className="mt-6 flex flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0 flex-1 space-y-1">
          <div>{t('settings.update.server.title')}</div>
          <div className="text-xs text-neutral-500">{t('settings.update.server.description')}</div>
        </div>
        <Switch
          className="shrink-0"
          aria-label={t('settings.update.server.title')}
          checked={isEnabled}
          loading={isLoading}
          disabled={unavailable}
          onChange={toggleEnabled}
        />
      </div>

      {isEnabled && (
        <div className="flex flex-col gap-2 sm:flex-row">
          <div className="min-w-0 flex-1">
            <Input
              aria-label={t('settings.update.server.title')}
              value={server}
              placeholder={OFFICIAL_UPDATE_SERVER}
              disabled={unavailable}
              maxLength={2048}
              autoComplete="off"
              spellCheck={false}
              allowClear
              status={error === 'invalidURL' ? 'error' : undefined}
              onChange={(event) => {
                setServer(event.target.value);
                setError('');
              }}
              onPressEnter={requestSave}
            />
          </div>
          <Button
            className="shrink-0"
            type="primary"
            loading={isLoading}
            disabled={unavailable || !hasChanges}
            onClick={requestSave}
          >
            {t('settings.update.server.save')}
          </Button>
        </div>
      )}
      {error && (
        <p role="alert" className="text-xs text-red-500">
          {t(`settings.update.server.${error}`)}
        </p>
      )}
      {savedServer === null && !isLoading && (
        <Button className="self-start" disabled={disabled} onClick={() => void load()}>
          {t('settings.update.retry')}
        </Button>
      )}
    </div>
  );
};
