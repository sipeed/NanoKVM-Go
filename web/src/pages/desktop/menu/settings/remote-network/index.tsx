import { useCallback, useEffect, useRef, useState } from 'react';
import { Spin } from 'antd';

import { syncServerFeatures } from '@/api/application.ts';

import { RemoteNetworkHome } from './home.tsx';
import { RemoteNetworkMessagesContext } from './messages.ts';
import { getRemoteNetworkComponent, getRemoteNetworkComponents } from './registry.ts';

type RemoteNetworkProps = {
  setIsLocked: (isLocked: boolean) => void;
};

/**
 * Remote network section shell.
 *
 * The home page lists every registered component and each component owns the
 * page opened from its card, so the shell never imports a component directly:
 * Tailscale and FRP are registered today; other providers register the same way.
 *
 * The shell also owns the lock. Components only report long running work through
 * `setLocked`; the shell alone decides what a lock means, so a component cannot
 * be left while it is still installing.
 */
export const RemoteNetwork = ({ setIsLocked }: RemoteNetworkProps) => {
  const [activeId, setActiveId] = useState<string | null>(null);
  const [isLocked, setIsSectionLocked] = useState(false);
  const [isSyncingFeatures, setIsSyncingFeatures] = useState(true);
  const featureSyncRef = useRef<ReturnType<typeof syncServerFeatures> | null>(null);

  const activeComponent = activeId ? getRemoteNetworkComponent(activeId) : undefined;

  useEffect(() => {
    let cancelled = false;
    // Reuse the request across effect replay; a new visit gets a new ref.
    featureSyncRef.current ??= syncServerFeatures();
    void featureSyncRef.current
      // A failed sync keeps the local flags; the status APIs still read them.
      .catch(() => undefined)
      .finally(() => {
        if (!cancelled) setIsSyncingFeatures(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  // Mirror the component reported lock onto the settings modal.
  useEffect(() => {
    setIsLocked(isLocked);
  }, [isLocked, setIsLocked]);

  // Never leave the settings modal locked once this section is gone.
  useEffect(() => {
    return () => setIsLocked(false);
  }, [setIsLocked]);

  const openComponent = useCallback((id: string) => {
    setIsSectionLocked(false);
    setActiveId(id);
  }, []);

  // The only way back to the home page: ignored while a component is busy.
  const backToHome = useCallback(() => {
    if (isLocked) return;

    setIsSectionLocked(false);
    setActiveId(null);
  }, [isLocked]);

  if (isSyncingFeatures) {
    return (
      <div className="flex justify-center pt-10">
        <Spin size="large" />
      </div>
    );
  }

  if (!activeComponent) {
    return <RemoteNetworkHome components={getRemoteNetworkComponents()} onOpen={openComponent} />;
  }

  const View = activeComponent.View;

  return (
    <RemoteNetworkMessagesContext.Provider value={activeComponent.messages}>
      <View onBack={backToHome} setLocked={setIsSectionLocked} />
    </RemoteNetworkMessagesContext.Provider>
  );
};
