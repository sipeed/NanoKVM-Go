import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Form, InputNumber, Spin } from 'antd';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { ScreenTimelapseRetentionConfig } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

import {
  screenTimelapseRetentionConfigToFormValues,
  screenTimelapseRetentionFormValuesToCleanup,
  screenTimelapseRetentionGiBToMiB,
  type ScreenTimelapseRetentionFormValues
} from './screen-timelapse-retention-model.ts';

const retentionTranslationRoot = 'settings.ai.retention';
const retentionDaysMaximum = 3_650;

export const ScreenTimelapseRetentionPanel = () => {
  const { t } = useTranslation();
  const [form] = Form.useForm<ScreenTimelapseRetentionFormValues>();
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();

  const [config, setConfig] = useState<ScreenTimelapseRetentionConfig>();
  const [isLoading, setIsLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);

  const mountedRef = useRef(false);
  const loadingRef = useRef(false);
  const savingRef = useRef(false);
  const configRef = useRef<ScreenTimelapseRetentionConfig>();

  const text = useCallback(
    (key: string, fallback: string, values?: Record<string, number | string>) =>
      t(`${retentionTranslationRoot}.${key}`, { defaultValue: fallback, ...values }),
    [t]
  );

  const applyConfig = useCallback(
    (nextConfig: ScreenTimelapseRetentionConfig) => {
      configRef.current = nextConfig;
      form.resetFields();
      form.setFieldsValue(screenTimelapseRetentionConfigToFormValues(nextConfig));
      setConfig(nextConfig);
    },
    [form]
  );

  const loadConfig = useCallback(async () => {
    if (loadingRef.current || savingRef.current) return;

    loadingRef.current = true;
    if (mountedRef.current) setIsLoading(true);

    try {
      const response = await api.getScreenTimelapseRetention();
      if (response.code !== 0 || !response.data?.config) {
        if (mountedRef.current) {
          feedback.error(text('loadFailed', 'Failed to load automatic cleanup settings.'));
        }
        return;
      }
      if (mountedRef.current) applyConfig(response.data.config);
    } catch {
      if (mountedRef.current) {
        feedback.error(text('loadFailed', 'Failed to load automatic cleanup settings.'));
      }
    } finally {
      loadingRef.current = false;
      if (mountedRef.current) setIsLoading(false);
    }
  }, [applyConfig, text]);

  useEffect(() => {
    mountedRef.current = true;
    void loadConfig();

    return () => {
      mountedRef.current = false;
    };
  }, [loadConfig]);

  const requestSave = useCallback(async () => {
    const currentConfig = configRef.current;
    if (!currentConfig || loadingRef.current || savingRef.current || isGateBusy) return;

    let cleanup;
    try {
      const values = await form.validateFields();
      cleanup = screenTimelapseRetentionFormValuesToCleanup(values);
    } catch {
      return;
    }

    await runWithChallenge(
      'set_screen_timelapse_retention',
      'screenTimelapse',
      async (challenge) => {
        if (savingRef.current) return;

        savingRef.current = true;
        if (mountedRef.current) setIsSaving(true);

        try {
          const response = await api.setScreenTimelapseRetention({
            revision: currentConfig.revision,
            cleanup,
            challenge
          });
          if (!mountedRef.current) return response;

          // -3 is this endpoint's revision conflict, not its protection error.
          if (response.code === -3) {
            feedback.error(
              text(
                'conflictDescription',
                'Settings changed elsewhere. Reload the page and try again.'
              )
            );
            return undefined;
          }
          if (response.code === -4) return response;
          if (response.code !== 0 || !response.data?.config) {
            feedback.error(text('saveFailed', 'Failed to save automatic cleanup settings.'));
            return response;
          }

          applyConfig(response.data.config);
          feedback.success(
            response.data.changed
              ? text('saved', 'Automatic cleanup settings saved.')
              : text('noChanges', 'No changes to save.')
          );
          return response;
        } catch {
          if (mountedRef.current) {
            feedback.error(text('saveFailed', 'Failed to save automatic cleanup settings.'));
          }
          return undefined;
        } finally {
          savingRef.current = false;
          if (mountedRef.current) setIsSaving(false);
        }
      }
    );
  }, [applyConfig, form, isGateBusy, runWithChallenge, text]);

  const isBusy = isLoading || isSaving || isGateBusy;

  return (
    <div className="border-t border-neutral-700/50 pt-6">
      {protectionModal}

      {isLoading && !config ? (
        <div className="flex min-h-32 items-center justify-center">
          <Spin />
        </div>
      ) : (
        <Form<ScreenTimelapseRetentionFormValues>
          form={form}
          layout="vertical"
          requiredMark={false}
          disabled={isBusy || !config}
        >
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <Form.Item
              className="mb-0"
              name="retentionDays"
              label={text('daysInput', 'Retention days')}
              rules={[
                {
                  validator: async (_, value: unknown) => {
                    if (value === undefined || value === null || value === '') return;
                    if (
                      typeof value !== 'number' ||
                      !Number.isInteger(value) ||
                      value < 0 ||
                      value > retentionDaysMaximum
                    ) {
                      throw new Error(
                        text(
                          'daysInvalid',
                          `Enter a whole number from 0 to ${retentionDaysMaximum}.`
                        )
                      );
                    }
                  }
                }
              ]}
            >
              <InputNumber
                className="w-full"
                min={0}
                max={retentionDaysMaximum}
                precision={0}
                placeholder="0"
              />
            </Form.Item>

            <Form.Item
              className="mb-0"
              name="maxStorageGiB"
              label={text('capacityInput', 'Storage limit')}
              rules={[
                {
                  validator: async (_, value: unknown) => {
                    if (value === undefined || value === null || value === '') return;
                    if (
                      typeof value !== 'number' ||
                      !Number.isFinite(value) ||
                      value < 0 ||
                      (value > 0 && screenTimelapseRetentionGiBToMiB(value) === 0)
                    ) {
                      throw new Error(text('capacityInvalid', 'Enter a valid storage limit.'));
                    }
                  }
                }
              ]}
            >
              <InputNumber
                className="w-full"
                min={0}
                precision={3}
                step={0.25}
                placeholder="0"
                addonAfter="GiB"
              />
            </Form.Item>
          </div>

          <div className="mt-4 flex justify-end">
            <Button
              type="primary"
              disabled={isBusy || !config}
              loading={isSaving}
              onClick={() => void requestSave()}
            >
              {text('save', 'Save')}
            </Button>
          </div>
        </Form>
      )}
    </div>
  );
};
