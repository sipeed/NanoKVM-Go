import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Divider, InputNumber, Popconfirm, Select, Slider } from 'antd';
import clsx from 'clsx';
import { CircleStopIcon, HandIcon, LoaderCircleIcon, PlayIcon, WrenchIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import type { FingerRobotArm, FingerRobotConfig, FingerRobotSpeed } from '@/api/vm.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';
import { MenuItem } from '@/components/menu-item.tsx';

import {
  armImageMap,
  fingerRobotImageLevels,
  type FingerRobotImageLevel
} from './finger-robot-images.ts';
import {
  cloneFingerRobotConfig,
  getActiveFingerRobotStrength,
  isFingerRobotChallengeResponse,
  parseFingerRobotConfig,
  parseFingerRobotData,
  resolveFingerRobotDisplayState,
  selectFingerRobotArm,
  selectFingerRobotPressMS,
  selectFingerRobotSpeed,
  selectFingerRobotStrength,
  shouldPollFingerRobotStatus
} from './finger-robot-model.ts';
import {
  formatFingerRobotDisplayState,
  getFingerRobotText,
  type FingerRobotText
} from './finger-robot-text.ts';

const fingerRobotStatusPollIntervalMS = 500;

type FingerRobotError =
  | ''
  | 'load'
  | 'config'
  | 'save'
  | 'conflict'
  | 'invalidDuration'
  | 'busy'
  | 'press'
  | 'release'
  | 'replaceArm';

function fingerRobotErrorText(error: FingerRobotError, text: FingerRobotText): string {
  switch (error) {
    case 'load':
      return text.loadFailed;
    case 'config':
      return text.configUnavailable;
    case 'save':
      return text.saveFailed;
    case 'conflict':
      return text.revisionConflict;
    case 'invalidDuration':
      return text.invalidDuration;
    case 'busy':
      return text.busy;
    case 'press':
      return text.pressFailed;
    case 'release':
      return text.releaseFailed;
    case 'replaceArm':
      return text.replaceArmFailed;
    default:
      return '';
  }
}

export const FingerRobot = () => {
  const { t, i18n } = useTranslation();
  const text = useMemo(
    () => getFingerRobotText(i18n.resolvedLanguage ?? i18n.language),
    [i18n.language, i18n.resolvedLanguage]
  );

  // The physical actuations (press and arm replacement) require a target_power
  // challenge when the protection policy demands it. Config save and release run
  // directly as plain authenticated operations.
  const challengeGate = useProtectionChallengeGate();

  const [data, setData] = useState<api.FingerRobotData | null>(null);
  const [draft, setDraft] = useState<FingerRobotConfig | null>(null);
  const [pressMSInput, setPressMSInput] = useState<number | null>(null);
  const [error, setError] = useState<FingerRobotError>('');
  const [panelOpen, setPanelOpen] = useState(false);
  const [initialLoading, setInitialLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [pressPending, setPressPending] = useState(false);
  const [releasePending, setReleasePending] = useState(false);
  const [replaceArmPending, setReplaceArmPending] = useState(false);
  const [failedImageSources, setFailedImageSources] = useState<ReadonlySet<string>>(
    () => new Set()
  );

  const mountedRef = useRef(false);
  const refreshingRef = useRef(false);
  const trailingRefreshRef = useRef({ pending: false, syncDraft: false });
  const draftDirtyRef = useRef(false);
  const configCommitRef = useRef(false);
  const pressActionRef = useRef(false);
  const releaseActionRef = useRef(false);
  const replaceArmActionRef = useRef(false);
  const operationGenerationRef = useRef(0);
  const runtimeEpochRef = useRef(0);
  const confirmedConfigRef = useRef<FingerRobotConfig | null>(null);
  const challengeGateWasBusyRef = useRef(false);

  const applyConfig = useCallback((config: FingerRobotConfig) => {
    const next = cloneFingerRobotConfig(config);
    confirmedConfigRef.current = next;
    draftDirtyRef.current = false;
    setDraft(next);
    setPressMSInput(next.pressMs);
    setData((current) => (current ? { ...current, config: next } : current));
  }, []);

  const refresh = useCallback(async (syncDraft: boolean): Promise<api.FingerRobotData | null> => {
    if (refreshingRef.current) {
      trailingRefreshRef.current.pending = true;
      trailingRefreshRef.current.syncDraft ||= syncDraft;
      return null;
    }
    refreshingRef.current = true;
    const requestEpoch = runtimeEpochRef.current;

    try {
      const response = await api.getFingerRobot();
      if (!mountedRef.current || requestEpoch !== runtimeEpochRef.current) return null;
      const parsed = parseFingerRobotData(response.data);
      if (!parsed) {
        if (mountedRef.current) setError('load');
        return null;
      }
      let accepted = parsed;
      const confirmedConfig = confirmedConfigRef.current;
      if (parsed.config && confirmedConfig && parsed.config.revision < confirmedConfig.revision) {
        accepted = { ...parsed, config: cloneFingerRobotConfig(confirmedConfig) };
      } else if (parsed.config) {
        confirmedConfigRef.current = cloneFingerRobotConfig(parsed.config);
      } else {
        confirmedConfigRef.current = null;
      }

      setData(accepted);
      if (accepted.config && (syncDraft || !draftDirtyRef.current)) {
        const next = cloneFingerRobotConfig(accepted.config);
        draftDirtyRef.current = false;
        setDraft(next);
        setPressMSInput(next.pressMs);
      }

      setError((current) => {
        if (!accepted.config) return 'config';
        if (current === 'busy' && !accepted.status.running) return '';
        if (response.code === 0 && (current === 'load' || current === 'config')) return '';
        return current;
      });
      return accepted;
    } catch (requestError) {
      console.log(requestError);
      if (mountedRef.current && requestEpoch === runtimeEpochRef.current) setError('load');
      return null;
    } finally {
      refreshingRef.current = false;
      if (mountedRef.current) setInitialLoading(false);
      if (mountedRef.current && trailingRefreshRef.current.pending) {
        const trailingSyncDraft = trailingRefreshRef.current.syncDraft;
        trailingRefreshRef.current = { pending: false, syncDraft: false };
        window.queueMicrotask(() => void refresh(trailingSyncDraft));
      }
    }
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    void refresh(true);
    return () => {
      mountedRef.current = false;
      operationGenerationRef.current += 1;
    };
  }, [refresh]);

  const activeArm = draft?.activeArm;

  useEffect(() => {
    if (!activeArm) return;

    const preloaders = fingerRobotImageLevels.flatMap((level) => {
      const source = armImageMap[activeArm][level];
      if (!source || failedImageSources.has(source)) return [];

      const image = new Image();
      image.onerror = () => {
        setFailedImageSources((current) => {
          if (current.has(source)) return current;
          return new Set(current).add(source);
        });
      };
      image.src = source;
      return [image];
    });

    return () => {
      for (const image of preloaders) image.onerror = null;
    };
  }, [activeArm, failedImageSources]);

  const pollStatus = shouldPollFingerRobotStatus(
    panelOpen,
    pressPending,
    releasePending || replaceArmPending,
    Boolean(data?.status.running)
  );

  useEffect(() => {
    if (!pollStatus) return;

    const timer = window.setInterval(() => {
      void refresh(false);
    }, fingerRobotStatusPollIntervalMS);
    return () => window.clearInterval(timer);
  }, [pollStatus, refresh]);

  useEffect(() => {
    if (challengeGate.isGateBusy) {
      challengeGateWasBusyRef.current = true;
      return;
    }
    if (challengeGateWasBusyRef.current) {
      challengeGateWasBusyRef.current = false;
      pressActionRef.current = false;
      replaceArmActionRef.current = false;
    }
  }, [challengeGate.isGateBusy]);

  const commitConfig = useCallback(
    (nextConfig: FingerRobotConfig) => {
      if (!data || configCommitRef.current) return;

      const validated = parseFingerRobotConfig(
        nextConfig,
        data.strength,
        data.pressMs,
        data.speedOptions
      );
      if (!validated) {
        setError('save');
        return;
      }
      if (data.config && JSON.stringify(validated) === JSON.stringify(data.config)) {
        applyConfig(data.config);
        setError('');
        return;
      }

      configCommitRef.current = true;
      draftDirtyRef.current = true;
      runtimeEpochRef.current += 1;
      setDraft(cloneFingerRobotConfig(validated));
      setPressMSInput(validated.pressMs);
      setError('');

      void (async () => {
        setIsSaving(true);
        try {
          const response = await api.setFingerRobot(validated);
          runtimeEpochRef.current += 1;

          const responseConfig = parseFingerRobotConfig(
            response.data?.config,
            data.strength,
            data.pressMs,
            data.speedOptions
          );
          if (response.code === 0 && responseConfig) {
            if (mountedRef.current) applyConfig(responseConfig);
            return;
          }
          if (response.code === -6 && responseConfig) {
            if (mountedRef.current) {
              applyConfig(responseConfig);
              setError('conflict');
            }
            return;
          }

          if (mountedRef.current) {
            setError('save');
            void refresh(true);
          }
        } catch (requestError) {
          console.log(requestError);
          runtimeEpochRef.current += 1;
          if (mountedRef.current) {
            setError('save');
            void refresh(true);
          }
        } finally {
          configCommitRef.current = false;
          if (mountedRef.current) setIsSaving(false);
        }
      })();
    },
    [applyConfig, data, refresh]
  );

  const previewStrength = useCallback(
    (value: number) => {
      if (!draft || !data) return;
      const next = selectFingerRobotStrength(draft, value, data.strength);
      if (!next) return;
      draftDirtyRef.current = true;
      setDraft(next);
      setError('');
    },
    [data, draft]
  );

  const updateArm = useCallback(
    (arm: FingerRobotArm) => {
      if (!draft || draft.activeArm === arm) return;
      commitConfig(selectFingerRobotArm(draft, arm));
    },
    [commitConfig, draft]
  );

  const updateStrength = useCallback(
    (value: number) => {
      if (!draft || !data) return;
      const next = selectFingerRobotStrength(draft, value, data.strength);
      if (next) commitConfig(next);
    },
    [commitConfig, data, draft]
  );

  const updateSpeed = useCallback(
    (speed: FingerRobotSpeed) => {
      if (!draft || !data || draft.speed === speed) return;
      const next = selectFingerRobotSpeed(draft, speed, data.speedOptions);
      if (next) commitConfig(next);
    },
    [commitConfig, data, draft]
  );

  const previewPressMS = useCallback(
    (value: number | null) => {
      setPressMSInput(value);
      draftDirtyRef.current = true;
      setError('');
      if (!draft || !data || value === null) return;

      const next = selectFingerRobotPressMS(draft, value, data.pressMs);
      if (next) setDraft(next);
    },
    [data, draft]
  );

  const commitPressMS = useCallback(() => {
    if (!draft || !data || pressMSInput === null) {
      setError('invalidDuration');
      return;
    }

    const next = selectFingerRobotPressMS(draft, pressMSInput, data.pressMs);
    if (!next) {
      setError('invalidDuration');
      return;
    }
    commitConfig(next);
  }, [commitConfig, data, draft, pressMSInput]);

  const press = useCallback(() => {
    if (
      !data?.config ||
      !data.enabled ||
      !data.available ||
      configCommitRef.current ||
      draftDirtyRef.current ||
      pressActionRef.current ||
      challengeGate.isGateBusy ||
      releaseActionRef.current ||
      releasePending ||
      replaceArmActionRef.current ||
      replaceArmPending ||
      data.status.running
    ) {
      return;
    }

    const generation = ++operationGenerationRef.current;
    runtimeEpochRef.current += 1;
    pressActionRef.current = true;
    setError('');

    void challengeGate
      .runWithChallenge('target_power', 'targetPower', async (challenge) => {
        if (generation !== operationGenerationRef.current) {
          pressActionRef.current = false;
          return;
        }

        let waitingForChallenge = false;
        setPressPending(true);
        try {
          const request = api.pressFingerRobot(challenge);
          void refresh(false);
          const response = await request;
          if (generation !== operationGenerationRef.current) return;
          if (isFingerRobotChallengeResponse(response.code)) {
            waitingForChallenge = true;
            return response;
          }
          runtimeEpochRef.current += 1;

          if (response.code === -2) {
            setError('busy');
          } else if (response.code !== 0) {
            setError('press');
          }
          await refresh(false);
          return response;
        } catch (requestError) {
          console.log(requestError);
          runtimeEpochRef.current += 1;
          if (mountedRef.current && generation === operationGenerationRef.current) {
            setError('press');
            void refresh(false);
          }
        } finally {
          if (!waitingForChallenge) pressActionRef.current = false;
          if (mountedRef.current) setPressPending(false);
        }
      })
      .catch((requestError) => {
        console.log(requestError);
        pressActionRef.current = false;
        if (mountedRef.current && generation === operationGenerationRef.current) setError('press');
      });
  }, [data, challengeGate, refresh, releasePending, replaceArmPending]);

  const release = useCallback(() => {
    if (
      !data?.enabled ||
      !data.available ||
      releaseActionRef.current ||
      replaceArmActionRef.current
    ) {
      return;
    }

    const generation = ++operationGenerationRef.current;
    runtimeEpochRef.current += 1;
    releaseActionRef.current = true;
    setError('');

    void (async () => {
      if (generation !== operationGenerationRef.current) {
        releaseActionRef.current = false;
        return;
      }

      setReleasePending(true);
      try {
        const request = api.releaseFingerRobot();
        void refresh(false);
        const response = await request;
        if (generation !== operationGenerationRef.current) return;
        runtimeEpochRef.current += 1;

        setError(response.code === 0 ? '' : 'release');
        await refresh(false);
      } catch (requestError) {
        console.log(requestError);
        runtimeEpochRef.current += 1;
        if (mountedRef.current && generation === operationGenerationRef.current) {
          setError('release');
          void refresh(false);
        }
      } finally {
        releaseActionRef.current = false;
        if (mountedRef.current) setReleasePending(false);
      }
    })();
  }, [data, refresh]);

  const replaceArm = useCallback(() => {
    if (
      !data?.enabled ||
      !data.available ||
      replaceArmActionRef.current ||
      challengeGate.isGateBusy ||
      releaseActionRef.current
    ) {
      return;
    }

    const generation = ++operationGenerationRef.current;
    runtimeEpochRef.current += 1;
    replaceArmActionRef.current = true;
    setError('');

    void challengeGate
      .runWithChallenge('target_power', 'targetPower', async (challenge) => {
        if (generation !== operationGenerationRef.current) {
          replaceArmActionRef.current = false;
          return;
        }

        let waitingForChallenge = false;
        setReplaceArmPending(true);
        try {
          const request = api.replaceFingerRobotArm(challenge);
          void refresh(false);
          const response = await request;
          if (generation !== operationGenerationRef.current) return;
          if (isFingerRobotChallengeResponse(response.code)) {
            waitingForChallenge = true;
            return response;
          }
          runtimeEpochRef.current += 1;

          setError(response.code === 0 ? '' : 'replaceArm');
          await refresh(false);
          return response;
        } catch (requestError) {
          console.log(requestError);
          runtimeEpochRef.current += 1;
          if (mountedRef.current && generation === operationGenerationRef.current) {
            setError('replaceArm');
            void refresh(false);
          }
        } finally {
          if (!waitingForChallenge) replaceArmActionRef.current = false;
          if (mountedRef.current) setReplaceArmPending(false);
        }
      })
      .catch((requestError) => {
        console.log(requestError);
        replaceArmActionRef.current = false;
        if (mountedRef.current && generation === operationGenerationRef.current) {
          setError('replaceArm');
        }
      });
  }, [data, challengeGate, refresh]);

  const formDisabled = initialLoading || isSaving || configCommitRef.current;
  const pressDisabled =
    formDisabled ||
    pressPending ||
    challengeGate.isGateBusy ||
    releasePending ||
    replaceArmPending ||
    !data?.enabled ||
    !data.available ||
    !data.config ||
    !draft ||
    draftDirtyRef.current ||
    Boolean(data.status.running);
  const releaseDisabled = releasePending || replaceArmPending || !data?.enabled || !data.available;
  const replaceArmDisabled =
    replaceArmPending ||
    releasePending ||
    challengeGate.isGateBusy ||
    !data?.enabled ||
    !data.available;
  const waitingForAvailability = Boolean(data?.enabled && !data.available);

  const statusState = releasePending
    ? 'releasing'
    : data
      ? resolveFingerRobotDisplayState(data.status, {
          busy: error === 'busy',
          failed: ['load', 'config', 'save', 'press', 'release', 'replaceArm'].includes(error),
          pressPending,
          releasePending,
          replaceArmPending
        })
      : initialLoading
        ? 'idle'
        : 'failed';
  const statusText =
    initialLoading && !data
      ? text.loading
      : waitingForAvailability
        ? text.reconnecting
        : formatFingerRobotDisplayState(statusState, text);
  const statusColor = waitingForAvailability
    ? 'bg-amber-400'
    : statusState === 'failed'
      ? 'bg-red-400'
      : statusState === 'busy'
        ? 'bg-amber-400'
        : statusState === 'completed' || statusState === 'armReplacement'
          ? 'bg-emerald-400'
          : data?.status.running || pressPending || releasePending || replaceArmPending
            ? 'bg-blue-400'
            : 'bg-neutral-500';
  const errorText = fingerRobotErrorText(error, text);

  const marks = useMemo(() => {
    if (!data) return {};
    return Object.fromEntries(
      Array.from({ length: data.strength.max - data.strength.min + 1 }, (_, index) => {
        const value = data.strength.min + index;
        return [
          value,
          <span className="whitespace-nowrap" key={value}>
            {text.levelMark(value)}
          </span>
        ];
      })
    );
  }, [data, text]);

  const iconBusy =
    initialLoading ||
    isSaving ||
    pressPending ||
    releasePending ||
    replaceArmPending ||
    data?.status.running;
  const icon = (
    <div className="h-[18px] w-[18px] text-neutral-300 hover:text-white">
      {iconBusy ? <LoaderCircleIcon className="animate-spin" size={18} /> : <HandIcon size={18} />}
    </div>
  );

  const content = (
    <div className="max-h-[calc(100dvh-80px)] w-[340px] max-w-full overflow-x-hidden overflow-y-auto overscroll-contain [-webkit-overflow-scrolling:touch]">
      {challengeGate.modal}

      <div className="flex items-center justify-between px-1">
        <span className="text-base font-bold text-neutral-200">{text.title}</span>
        <div className="flex items-center space-x-1.5 text-xs text-neutral-500">
          <span className={clsx('size-2 rounded-full', statusColor)} />
          <span>{statusText}</span>
        </div>
      </div>

      <Divider style={{ margin: '10px 0 14px 0' }} />

      <div className="space-y-4">
        {initialLoading && !data && (
          <div className="flex min-h-20 items-center justify-center text-neutral-500">
            <LoaderCircleIcon className="animate-spin" size={20} />
          </div>
        )}
        {!initialLoading && !data && <Alert type="error" showIcon message={text.loadFailed} />}
        {data && (
          <>
            {data && !data.enabled && <Alert type="warning" showIcon message={text.disabled} />}
            {data?.enabled && !data.available && (
              <Alert
                type="warning"
                showIcon
                message={text.unavailable}
              />
            )}
            {data && !data.config && (
              <Alert type="error" showIcon message={text.configUnavailable} />
            )}

            {data.config && draft && (
              <>
                <fieldset disabled={formDisabled} className="m-0 min-w-0 space-y-2 border-0 p-0">
                  <legend className="text-xs text-neutral-500">{text.arm}</legend>
                  <div className="grid grid-cols-2 gap-2">
                    {data.profiles.map((profile) => {
                      const selected = draft.activeArm === profile.id;
                      const strength = draft.strengthByArm[profile.id] as FingerRobotImageLevel;
                      const armName = profile.id === 'long' ? text.longArm : text.shortArm;
                      const imageSource = armImageMap[profile.id][strength];
                      const showImage = imageSource && !failedImageSources.has(imageSource);
                      return (
                        <button
                          key={profile.id}
                          type="button"
                          className={clsx(
                            'overflow-hidden rounded-lg border text-left transition-colors',
                            !showImage && 'px-3 py-2',
                            selected
                              ? 'border-blue-500 bg-blue-500/15 text-blue-200'
                              : 'border-neutral-700 bg-neutral-800/70 text-neutral-300 hover:border-neutral-500 hover:bg-neutral-700/70',
                            formDisabled && 'cursor-not-allowed opacity-50'
                          )}
                          aria-pressed={selected}
                          onClick={() => updateArm(profile.id)}
                        >
                          {showImage && (
                            <span className="mt-2 block aspect-[8/5] w-full overflow-hidden bg-[#202225]">
                              <img
                                src={imageSource}
                                alt={`${armName} - ${text.level(strength)}`}
                                className="h-full w-full object-contain"
                                onError={() => {
                                  setFailedImageSources((current) => {
                                    if (current.has(imageSource)) return current;
                                    return new Set(current).add(imageSource);
                                  });
                                }}
                              />
                            </span>
                          )}
                          <span
                            className={clsx(
                              'flex items-center justify-between gap-2',
                              showImage && 'px-3 py-2'
                            )}
                          >
                            <span className="min-w-0 truncate text-sm font-medium">{armName}</span>
                            <span className="shrink-0 text-xs text-neutral-500">
                              {text.level(strength)}
                            </span>
                          </span>
                        </button>
                      );
                    })}
                  </div>
                </fieldset>

                <label className="block space-y-1 text-xs text-neutral-500">
                  <span className="flex items-center justify-between">
                    <span>{text.strength}</span>
                    <span className="text-neutral-300">
                      {text.level(getActiveFingerRobotStrength(draft))}
                    </span>
                  </span>
                  <div className="px-1 pb-3">
                    <Slider
                      value={getActiveFingerRobotStrength(draft)}
                      min={data.strength.min}
                      max={data.strength.max}
                      step={null}
                      marks={marks}
                      tooltip={{ open: false }}
                      disabled={formDisabled}
                      onChange={previewStrength}
                      onChangeComplete={updateStrength}
                    />
                  </div>
                </label>

                <div className="!mt-2 grid grid-cols-2 gap-3">
                  <label className="flex min-w-0 flex-col space-y-1 text-xs text-neutral-500">
                    <span>{text.holdDuration}</span>
                    <InputNumber
                      className="w-full"
                      value={pressMSInput}
                      min={data.pressMs.min}
                      max={data.pressMs.max}
                      step={100}
                      precision={0}
                      addonAfter="ms"
                      disabled={formDisabled}
                      status={error === 'invalidDuration' ? 'error' : undefined}
                      onChange={previewPressMS}
                      onBlur={commitPressMS}
                      onPressEnter={(event) => event.currentTarget.blur()}
                    />
                  </label>

                  <label className="flex min-w-0 flex-col space-y-1 text-xs text-neutral-500">
                    <span>{text.speed}</span>
                    <Select
                      value={draft.speed}
                      disabled={formDisabled}
                      options={data.speedOptions.map((speed) => ({
                        value: speed,
                        label: speed === 'slow' ? text.slow : text.fast
                      }))}
                      onChange={updateSpeed}
                    />
                  </label>
                </div>
              </>
            )}
          </>
        )}

        {errorText && <div className="break-words text-xs text-red-400">{errorText}</div>}

        <div className="flex gap-2">
          <Button
            type="primary"
            className="flex-1"
            icon={
              pressPending ? (
                <LoaderCircleIcon className="animate-spin" size={15} />
              ) : (
                <PlayIcon size={15} />
              )
            }
            disabled={pressDisabled}
            loading={pressPending}
            onClick={press}
          >
            {text.pressOnce}
          </Button>
          <Button
            danger
            className="flex-1"
            icon={
              releasePending ? (
                <LoaderCircleIcon className="animate-spin" size={15} />
              ) : (
                <CircleStopIcon size={15} />
              )
            }
            disabled={releaseDisabled}
            loading={releasePending}
            onClick={release}
          >
            {text.releaseNow}
          </Button>
        </div>
        <Popconfirm
          title={text.replaceArmConfirm}
          okText={t('settings.device.okBtn')}
          cancelText={t('settings.device.cancelBtn')}
          disabled={replaceArmDisabled}
          okButtonProps={{ disabled: replaceArmDisabled }}
          onConfirm={replaceArm}
        >
          <Button
            block
            icon={<WrenchIcon size={15} />}
            title={text.replaceArmHint}
            disabled={replaceArmDisabled}
            loading={replaceArmPending}
          >
            {text.replaceArm}
          </Button>
        </Popconfirm>
      </div>
    </div>
  );

  return (
    <MenuItem
      title={text.title}
      icon={icon}
      content={content}
      fresh
      onOpenChange={(open) => {
        setPanelOpen(open);
        if (open) void refresh(true);
      }}
    />
  );
};
