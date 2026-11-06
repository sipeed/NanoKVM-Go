import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Checkbox, InputNumber } from 'antd';
import {
  CircleStopIcon,
  FlaskConicalIcon,
  LoaderCircleIcon,
  PlayIcon,
  RotateCcwIcon
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/vm.ts';
import type { FingerRobotDebugData } from '@/api/vm.ts';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import {
  createFingerRobotDebugDraft,
  fingerRobotDebugDefaults,
  formatFingerRobotDebugProgress,
  getFingerRobotDebugChallengeResult,
  getFingerRobotDebugPrimaryAction,
  isFingerRobotDebugResetDisabled,
  parseFingerRobotDebugData,
  runFingerRobotDebugStartIfActive,
  shouldPollFingerRobotDebugStatus,
  snapshotFingerRobotDebugDraft,
  stopFingerRobotDebugOnUnmount,
  validateFingerRobotDebugDraft,
  type FingerRobotDebugDraft,
  type FingerRobotDebugEditableField
} from './finger-robot-debug-model.ts';
import {
  formatFingerRobotResult,
  formatFingerRobotState,
  getFingerRobotDebugText
} from './finger-robot-debug-text.ts';

const statusPollIntervalMs = 500;

function responseError(prefix: string, message: unknown): string {
  const detail = typeof message === 'string' ? message.trim() : '';
  return detail ? `${prefix} ${detail}` : prefix;
}

export const FingerRobotDebug = () => {
  const { i18n } = useTranslation();
  const text = useMemo(
    () => getFingerRobotDebugText(i18n.resolvedLanguage ?? i18n.language),
    [i18n.language, i18n.resolvedLanguage]
  );
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();

  const [debugData, setDebugData] = useState<FingerRobotDebugData | null>(null);
  const [draft, setDraft] = useState<FingerRobotDebugDraft>({ ...fingerRobotDebugDefaults });
  const [operationError, setOperationError] = useState('');
  const [isStarting, setIsStarting] = useState(false);
  const [isStopping, setIsStopping] = useState(false);
  const [isResetting, setIsResetting] = useState(false);
  const [pollingRequested, setPollingRequested] = useState(false);

  const mountedRef = useRef(false);
  const refreshingRef = useRef(false);
  const startingRef = useRef(false);
  const stoppingRef = useRef(false);
  const resettingRef = useRef(false);
  const draftInitializedRef = useRef(false);
  const debugWasEnabledRef = useRef(false);

  const applyResponse = useCallback((value: unknown): FingerRobotDebugData | null => {
    const parsed = parseFingerRobotDebugData(value);
    if (!mountedRef.current) return parsed;

    if (!parsed) {
      setDebugData(null);
      setPollingRequested(false);
      return null;
    }

    debugWasEnabledRef.current = true;
    setDebugData(parsed);
    setPollingRequested(shouldPollFingerRobotDebugStatus(parsed.running, parsed.available));
    if (!draftInitializedRef.current) {
      draftInitializedRef.current = true;
      setDraft(createFingerRobotDebugDraft(parsed));
    }
    return parsed;
  }, []);

  const refreshStatus = useCallback(async () => {
    if (refreshingRef.current) return null;
    refreshingRef.current = true;

    try {
      const response = await api.getFingerRobotDebug();
      if (response.code !== 0) {
        if (mountedRef.current) setDebugData(null);
        return null;
      }
      return applyResponse(response.data);
    } catch (error) {
      console.log(error);
      // A failed capability/status query must hide the engineering controls.
      if (mountedRef.current) setDebugData(null);
      return null;
    } finally {
      refreshingRef.current = false;
    }
  }, [applyResponse]);

  useEffect(() => {
    mountedRef.current = true;
    void refreshStatus();

    return () => {
      mountedRef.current = false;
      if (debugWasEnabledRef.current) {
        // Best effort only. Stop returns to the configured release point; it never factory-zeros.
        stopFingerRobotDebugOnUnmount(api.stopFingerRobotDebug);
      }
    };
  }, [refreshStatus]);

  useEffect(() => {
    if (!pollingRequested) return;

    const timer = window.setInterval(() => {
      void refreshStatus();
    }, statusPollIntervalMs);
    return () => window.clearInterval(timer);
  }, [pollingRequested, refreshStatus]);

  const updateDraft = useCallback((field: FingerRobotDebugEditableField, value: number | null) => {
    setOperationError('');
    setDraft((current) => ({ ...current, [field]: value }));
  }, []);

  const updateUnlimited = useCallback((unlimited: boolean) => {
    setOperationError('');
    setDraft((current) => ({ ...current, unlimited }));
  }, []);

  const startTest = useCallback(() => {
    if (
      !debugData ||
      !debugData.available ||
      startingRef.current ||
      stoppingRef.current ||
      resettingRef.current ||
      isGateBusy
    ) {
      return;
    }

    const params = snapshotFingerRobotDebugDraft(draft, debugData);
    if (!params) {
      const error = validateFingerRobotDebugDraft(draft, debugData);
      setOperationError(error === 'direction' ? text.invalidDirection : text.invalidRange);
      return;
    }

    setOperationError('');
    void runWithChallenge('target_power', 'targetPower', async (challenge) => {
      return runFingerRobotDebugStartIfActive(mountedRef.current, startingRef.current, async () => {
        startingRef.current = true;
        if (mountedRef.current) {
          setIsStarting(true);
          // Keep checking after a timeout: the server may have accepted the sequence.
          setPollingRequested(true);
        }

        try {
          const response = await api.testFingerRobotDebug(params, challenge);
          const challengeResult = getFingerRobotDebugChallengeResult(response);
          if (challengeResult) return challengeResult;
          if (response.code !== 0) {
            if (mountedRef.current) {
              setOperationError(responseError(text.testFailed, response.msg));
            }
            // Ordinary -3/-4 operation errors must not be reinterpreted as a 2FA challenge.
            return undefined;
          }

          await refreshStatus();
          return response;
        } catch (error) {
          console.log(error);
          if (mountedRef.current) setOperationError(text.testFailed);
          return undefined;
        } finally {
          startingRef.current = false;
          if (mountedRef.current) setIsStarting(false);
        }
      });
    });
  }, [debugData, draft, isGateBusy, refreshStatus, runWithChallenge, text]);

  const stopTest = useCallback(async () => {
    if (stoppingRef.current || resettingRef.current) return;

    stoppingRef.current = true;
    setIsStopping(true);
    setOperationError('');
    setPollingRequested(true);

    try {
      const response = await api.stopFingerRobotDebug();
      if (response.code !== 0) {
        if (mountedRef.current) {
          setOperationError(responseError(text.stopFailed, response.msg));
        }
        return;
      }

      if (mountedRef.current) setOperationError(text.stopComplete);
      await refreshStatus();
    } catch (error) {
      console.log(error);
      if (mountedRef.current) setOperationError(text.stopFailed);
    } finally {
      stoppingRef.current = false;
      if (mountedRef.current) setIsStopping(false);
    }
  }, [refreshStatus, text]);

  const resetToZero = useCallback(async () => {
    if (isGateBusy || startingRef.current || stoppingRef.current || resettingRef.current) {
      return;
    }

    resettingRef.current = true;
    setIsResetting(true);
    setOperationError('');
    setPollingRequested(true);

    try {
      const response = await api.resetFingerRobotDebug();
      if (response.code !== 0) {
        if (mountedRef.current) {
          setOperationError(responseError(text.resetFailed, response.msg));
        }
        return;
      }

      if (mountedRef.current) setOperationError(text.resetComplete);
      await refreshStatus();
    } catch (error) {
      console.log(error);
      if (mountedRef.current) setOperationError(text.resetFailed);
    } finally {
      resettingRef.current = false;
      if (mountedRef.current) setIsResetting(false);
    }
  }, [isGateBusy, refreshStatus, text]);

  if (!debugData) return null;

  const draftError = validateFingerRobotDebugDraft(draft, debugData);
  const controllerBusy = debugData.running;
  const primaryAction = getFingerRobotDebugPrimaryAction(controllerBusy);
  const resetDisabled = isFingerRobotDebugResetDisabled({
    running: controllerBusy,
    isGateBusy,
    isStarting,
    isStopping,
    isResetting
  });
  const formDisabled = controllerBusy || isStarting || isStopping || isResetting || isGateBusy;
  const statusError = debugData.error.trim();
  const phase = formatFingerRobotState(debugData.state, text);
  const result = debugData.result
    ? formatFingerRobotResult(debugData.result, text)
    : debugData.running
      ? text.running
      : text.idle;

  return (
    <div className="mt-4 overflow-hidden rounded-xl border border-amber-500/40 bg-amber-950/10">
      {protectionModal}

      <div className="flex items-center space-x-2 border-b border-amber-500/20 px-4 py-3 text-amber-300">
        <FlaskConicalIcon size={18} aria-hidden="true" />
        <span className="font-semibold">{text.title}</span>
      </div>

      <div className="space-y-4 p-4">
        <Alert type="warning" showIcon message={text.warning} />
        {!debugData.available && (
          <Alert
            type="warning"
            showIcon
            message={text.unavailable}
          />
        )}

        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <DebugNumberInput
            label={text.releasePoint}
            unit="us"
            value={draft.releaseUs}
            min={debugData.releaseUs.min}
            max={debugData.releaseUs.max}
            disabled={formDisabled}
            hasError={draftError === 'releaseUs' || draftError === 'direction'}
            onChange={(value) => updateDraft('releaseUs', value)}
          />
          <DebugNumberInput
            label={text.pressPoint}
            unit="us"
            value={draft.pressUs}
            min={debugData.pressUs.min}
            max={debugData.pressUs.max}
            disabled={formDisabled}
            hasError={draftError === 'pressUs' || draftError === 'direction'}
            onChange={(value) => updateDraft('pressUs', value)}
          />
          <DebugNumberInput
            label={text.pressSpeed}
            unit="us/s"
            value={draft.pressSpeedUsPerSecond}
            min={debugData.pressSpeedUsPerSecond.min}
            max={debugData.pressSpeedUsPerSecond.max}
            step={100}
            disabled={formDisabled}
            hasError={draftError === 'pressSpeedUsPerSecond'}
            onChange={(value) => updateDraft('pressSpeedUsPerSecond', value)}
          />
          <DebugNumberInput
            label={text.pressTime}
            unit="ms"
            value={draft.pressMs}
            min={debugData.pressMs.min}
            max={debugData.pressMs.max}
            disabled={formDisabled}
            hasError={draftError === 'pressMs'}
            onChange={(value) => updateDraft('pressMs', value)}
          />
          <DebugCountInput
            label={text.count}
            unlimitedLabel={text.unlimited}
            value={draft.count}
            min={debugData.count.min}
            unlimited={draft.unlimited}
            disabled={formDisabled}
            hasError={draftError === 'count'}
            onValueChange={(value) => updateDraft('count', value)}
            onUnlimitedChange={updateUnlimited}
          />
        </div>

        {draftError && !operationError && (
          <div className="text-xs text-red-400">
            {draftError === 'direction' ? text.invalidDirection : text.invalidRange}
          </div>
        )}
        {operationError && (
          <div
            className={
              operationError === text.stopComplete || operationError === text.resetComplete
                ? 'text-xs text-emerald-400'
                : 'text-xs text-red-400'
            }
          >
            {operationError}
          </div>
        )}

        <div className="flex flex-wrap gap-2">
          <Button
            type="primary"
            danger={primaryAction === 'stop'}
            icon={
              isStarting || isStopping ? (
                <LoaderCircleIcon className="animate-spin" size={15} />
              ) : primaryAction === 'stop' ? (
                <CircleStopIcon size={15} />
              ) : (
                <PlayIcon size={15} />
              )
            }
            disabled={
              primaryAction === 'stop'
                ? isStopping || isResetting
                : formDisabled || !debugData.available || draftError !== null
            }
            loading={primaryAction === 'stop' ? isStopping : isStarting}
            onClick={primaryAction === 'stop' ? () => void stopTest() : startTest}
          >
            {primaryAction === 'stop' ? text.stop : text.start}
          </Button>
          <Button
            danger
            icon={<RotateCcwIcon size={15} />}
            loading={isResetting}
            disabled={resetDisabled}
            onClick={() => void resetToZero()}
          >
            {text.reset}
          </Button>
        </div>

        <div className="rounded-lg border border-neutral-700/70 bg-neutral-900/60 p-3">
          <div className="mb-2 flex items-center space-x-2 text-sm font-medium text-neutral-300">
            {debugData.running && (
              <LoaderCircleIcon className="animate-spin text-blue-400" size={15} />
            )}
            <span>{text.status}</span>
          </div>
          <div className="grid grid-cols-1 gap-x-6 gap-y-1 text-xs sm:grid-cols-3">
            <StatusValue
              label={text.progress}
              value={formatFingerRobotDebugProgress(
                debugData.current,
                debugData.total,
                debugData.running,
                text.infinity
              )}
            />
            <StatusValue label={text.phase} value={phase} />
            <StatusValue label={text.result} value={result} />
          </div>
          {statusError && (
            <div className="mt-2 break-words text-xs text-red-400">{statusError}</div>
          )}
        </div>
      </div>
    </div>
  );
};

type DebugNumberInputProps = {
  label: string;
  unit?: string;
  value: number | null;
  min: number;
  max?: number;
  step?: number;
  disabled: boolean;
  hasError: boolean;
  onChange: (value: number | null) => void;
};

const DebugNumberInput = ({
  label,
  unit,
  value,
  min,
  max,
  step = 1,
  disabled,
  hasError,
  onChange
}: DebugNumberInputProps) => (
  <label className="flex min-w-0 flex-col space-y-1 text-xs text-neutral-500">
    <span>{label}</span>
    <InputNumber
      className="w-full"
      value={value}
      min={min}
      max={max}
      step={step}
      precision={0}
      addonAfter={unit}
      disabled={disabled}
      status={hasError ? 'error' : undefined}
      onChange={onChange}
    />
    <span className="font-mono text-[10px] text-neutral-600">
      {max === undefined ? `≥ ${min}` : `${min}–${max}`}
      {unit ? ` ${unit}` : ''}
    </span>
  </label>
);

type DebugCountInputProps = {
  label: string;
  unlimitedLabel: string;
  value: number | null;
  min: number;
  unlimited: boolean;
  disabled: boolean;
  hasError: boolean;
  onValueChange: (value: number | null) => void;
  onUnlimitedChange: (value: boolean) => void;
};

const DebugCountInput = ({
  label,
  unlimitedLabel,
  value,
  min,
  unlimited,
  disabled,
  hasError,
  onValueChange,
  onUnlimitedChange
}: DebugCountInputProps) => (
  <div className="flex min-w-0 flex-col space-y-1 text-xs text-neutral-500">
    <span>{label}</span>
    <div className="flex min-w-0 items-center gap-3">
      <InputNumber
        className="min-w-0 flex-1"
        value={value}
        min={min}
        precision={0}
        disabled={disabled || unlimited}
        status={hasError ? 'error' : undefined}
        onChange={onValueChange}
      />
      <Checkbox
        checked={unlimited}
        disabled={disabled}
        onChange={(event) => onUnlimitedChange(event.target.checked)}
      >
        <span className="whitespace-nowrap text-neutral-300">{unlimitedLabel}</span>
      </Checkbox>
    </div>
    <span className="font-mono text-[10px] text-neutral-600">≥ {min}</span>
  </div>
);

const StatusValue = ({ label, value }: { label: string; value: string }) => (
  <div className="flex min-w-0 items-center justify-between gap-3 sm:block">
    <span className="text-neutral-500">{label}</span>
    <div className="truncate text-neutral-300 sm:mt-0.5">{value}</div>
  </div>
);
