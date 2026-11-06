import { lazy, Suspense, useCallback, useEffect, useRef, useState } from 'react';
import { Divider, Spin, Switch, Tooltip } from 'antd';
import { CircleAlertIcon, HistoryIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { ScreenTimelapseState } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';
import { LAYER_OVERLAY } from '@/lib/layers.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';
import { useDismissMobileMenu } from '@/components/mobile-menu-context.ts';

const ScreenTimelapseModal = lazy(() =>
  import('./screen-timelapse-modal').then((module) => ({ default: module.ScreenTimelapseModal }))
);

const ScreenTimelapseRetentionPanel = lazy(() =>
  import('./screen-timelapse-retention').then((module) => ({
    default: module.ScreenTimelapseRetentionPanel
  }))
);

const ScreenTimelapseRecordingPanel = lazy(() =>
  import('./screen-timelapse-recording').then((module) => ({
    default: module.ScreenTimelapseRecordingPanel
  }))
);

type ScreenTimelapseProps = {
  variant?: 'settings' | 'menu';
  tooltipPlacement?: 'top' | 'bottom' | 'left' | 'right';
  isServiceToggleLocked?: boolean;
  runServiceToggle?: (action: () => Promise<void>) => Promise<void>;
};

async function runServiceToggleUnlocked(action: () => Promise<void>) {
  await action();
}

export const ScreenTimelapse = ({
  variant = 'settings',
  tooltipPlacement = 'bottom',
  isServiceToggleLocked = false,
  runServiceToggle = runServiceToggleUnlocked
}: ScreenTimelapseProps) => {
  const { t } = useTranslation();
  const dismissMobileMenu = useDismissMobileMenu();
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [playbackAuthorizationVersion, setPlaybackAuthorizationVersion] = useState(0);
  const [isTooltipOpen, setIsTooltipOpen] = useState(false);
  const [state, setState] = useState<ScreenTimelapseState>({
    enabled: false
  });
  const [isLoading, setIsLoading] = useState(false);
  const isLoadingRef = useRef(false);
  const silentRefreshInFlightRef = useRef(false);
  const stateActionVersionRef = useRef(0);

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
        .getScreenTimelapseState()
        .then((rsp) => {
          if (rsp.code !== 0) {
            if (!silent) {
              console.log(rsp.msg);
              feedback.error(t('settings.ai.errors.loadScreenTimelapseState'));
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
            feedback.error(t('settings.ai.errors.loadScreenTimelapseState'));
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
    if (variant !== 'settings') return;

    getState();
    const timer = window.setInterval(() => getState(true), 3000);

    return () => window.clearInterval(timer);
  }, [getState, variant]);

  async function updateScreenTimelapseEnabled(enabled: boolean, challenge?: string) {
    let result: { code: number } | undefined;

    await runServiceToggle(async () => {
      stateActionVersionRef.current += 1;
      updateLoading(true);

      try {
        const rsp = await api.setScreenTimelapseEnabled(enabled, challenge);
        result = rsp;
        if (rsp.code === -3) {
          return;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.updateScreenTimelapseService'));
          return;
        }

        setState(rsp.data);
      } catch {
        feedback.error(t('settings.ai.errors.updateScreenTimelapseService'));
      } finally {
        updateLoading(false);
      }
    });
    return result;
  }

  function setEnabled(enabled: boolean) {
    if (isLoadingRef.current || isServiceToggleLocked || isGateBusy) return;

    if (!enabled) {
      void updateScreenTimelapseEnabled(false);
      return;
    }

    void runWithChallenge('screen_timelapse_enable', 'screenTimelapse', (challenge) =>
      updateScreenTimelapseEnabled(true, challenge)
    );
  }

  const authorizeScreenTimelapsePlayback = useCallback(
    async (challenge?: string) => {
      try {
        const rsp = await api.authorizeScreenTimelapsePlayback(challenge);
        if (rsp.code === 0) {
          setPlaybackAuthorizationVersion((version) => version + 1);
          setIsModalOpen(true);
          return rsp;
        }
        if (rsp.code !== -3) {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.openScreenTimelapse'));
        }
        return rsp;
      } catch {
        feedback.error(t('settings.ai.errors.openScreenTimelapse'));
      }
    },
    [t]
  );

  const requestPlaybackAuthorization = useCallback(
    (onAuthorized?: () => void) => {
      void runWithChallenge('screen_timelapse_playback', 'screenTimelapse', async (challenge) => {
        const rsp = await authorizeScreenTimelapsePlayback(challenge);
        if (rsp?.code === 0) onAuthorized?.();
        // Returning rsp keeps the gate's -3/-4 reopen probe working.
        return rsp;
      });
    },
    [authorizeScreenTimelapsePlayback, runWithChallenge]
  );

  function openMenuModal() {
    setIsTooltipOpen(false);
    dismissMobileMenu();
    requestPlaybackAuthorization();
  }

  const modal = isModalOpen ? (
    <Suspense
      fallback={
        <div
          className="fixed inset-0 flex items-center justify-center bg-black/30"
          style={{ zIndex: LAYER_OVERLAY }}
        >
          <Spin />
        </div>
      }
    >
      <ScreenTimelapseModal
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        authorizationVersion={playbackAuthorizationVersion}
        onAuthorizationRequired={requestPlaybackAuthorization}
      />
    </Suspense>
  ) : null;

  if (variant === 'menu') {
    return (
      <>
        <Tooltip
          title={t('settings.ai.screenTimelapseService')}
          mouseEnterDelay={0.6}
          placement={tooltipPlacement}
          open={isTooltipOpen}
          onOpenChange={setIsTooltipOpen}
        >
          <div
            role="button"
            tabIndex={0}
            className="flex h-[30px] w-[30px] cursor-pointer items-center justify-center rounded text-neutral-300 hover:bg-neutral-700/80 hover:text-white"
            aria-label={t('settings.ai.screenTimelapseService')}
            onClick={openMenuModal}
            onKeyDown={(event) => {
              if (event.key === 'Enter' || event.key === ' ') {
                event.preventDefault();
                openMenuModal();
              }
            }}
          >
            <HistoryIcon size={18} />
          </div>
        </Tooltip>
        {modal}
        {protectionModal}
      </>
    );
  }

  return (
    <div className="flex flex-col space-y-6">
      <div className="flex items-center justify-between gap-4">
        <div className="flex min-w-0 flex-col space-y-1">
          <div className="flex items-center gap-1.5">
            <span className="text-sm font-medium">{t('settings.ai.screenTimelapseService')}</span>
            <Tooltip
              title={t('settings.ai.screenTimelapseSearchRequiresMemoryFabric')}
              placement="top"
              styles={{ root: { maxWidth: '400px' } }}
            >
              <span
                className="inline-flex cursor-pointer text-amber-400"
                tabIndex={0}
                aria-label={t('settings.ai.screenTimelapseSearchRequiresMemoryFabric')}
              >
                <CircleAlertIcon size={14} aria-hidden="true" />
              </span>
            </Tooltip>
          </div>
          <span className="text-xs text-neutral-500">
            {t('settings.ai.screenTimelapseServiceDesc')}
          </span>
        </div>

        <Switch
          checked={state.enabled}
          disabled={isServiceToggleLocked || isGateBusy}
          loading={isLoading || isGateBusy}
          onChange={setEnabled}
        />
      </div>

      {state.enabled ? (
        <Suspense
          fallback={
            <div className="flex min-h-24 items-center justify-center">
              <Spin size="small" />
            </div>
          }
        >
          <ScreenTimelapseRecordingPanel
            disabled={isLoading || isServiceToggleLocked || isGateBusy}
          />
        </Suspense>
      ) : null}

      {state.enabled ? (
        <Suspense
          fallback={
            <div className="flex min-h-24 items-center justify-center">
              <Spin size="small" />
            </div>
          }
        >
          <ScreenTimelapseRetentionPanel />
        </Suspense>
      ) : null}

      <Divider className="opacity-50" />

      {modal}
      {protectionModal}
    </div>
  );
};
