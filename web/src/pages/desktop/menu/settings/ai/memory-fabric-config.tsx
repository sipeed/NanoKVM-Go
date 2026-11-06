import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Collapse, Form, Input, InputNumber, Modal, Select, Tag } from 'antd';
import type { Rule } from 'antd/es/form';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { MemoryFabricConfig, MemoryFabricModelConfig } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';

const configTranslationRoot = 'settings.ai.memoryFabricConfig';
const maxModelTokens = 1_048_576;
const maxModelTimeoutSeconds = 3_600;

type APIKeyAction = 'keep' | 'replace' | 'clear';

type ModelFormConfig = Omit<MemoryFabricModelConfig, 'apiKey' | 'apiKeyConfigured'> & {
  apiKeyAction: APIKeyAction;
  apiKeyReplacement: string;
};

type MemoryFabricConfigFormValues = {
  models: {
    default: ModelFormConfig;
  };
};

function sanitizeModelConfig(model: MemoryFabricModelConfig): MemoryFabricModelConfig {
  return {
    model: model.model,
    baseUrl: model.baseUrl,
    apiKeyConfigured: model.apiKeyConfigured,
    maxTokens: model.maxTokens,
    timeoutSeconds: model.timeoutSeconds
  };
}

function sanitizeConfig(config: MemoryFabricConfig): MemoryFabricConfig {
  return {
    revision: config.revision,
    models: {
      default: sanitizeModelConfig(config.models.default)
    }
  };
}

function modelToFormValues(model: MemoryFabricModelConfig): ModelFormConfig {
  return {
    model: model.model,
    baseUrl: model.baseUrl,
    maxTokens: model.maxTokens,
    timeoutSeconds: model.timeoutSeconds,
    apiKeyAction: 'keep',
    apiKeyReplacement: ''
  };
}

function configToFormValues(config: MemoryFabricConfig): MemoryFabricConfigFormValues {
  return {
    models: {
      default: modelToFormValues(config.models.default)
    }
  };
}

function formValuesToConfig(
  current: MemoryFabricConfig,
  values: MemoryFabricConfigFormValues
): MemoryFabricConfig {
  const valuesForModel = values.models.default;
  const defaultModel: MemoryFabricModelConfig = {
    model: valuesForModel.model,
    baseUrl: valuesForModel.baseUrl,
    apiKeyConfigured: current.models.default.apiKeyConfigured,
    maxTokens: valuesForModel.maxTokens,
    timeoutSeconds: valuesForModel.timeoutSeconds
  };

  if (valuesForModel.apiKeyAction === 'replace') {
    defaultModel.apiKey = valuesForModel.apiKeyReplacement;
  } else if (valuesForModel.apiKeyAction === 'clear') {
    defaultModel.apiKey = '';
  }

  return {
    revision: current.revision,
    models: {
      default: defaultModel
    }
  };
}

function isValidHTTPURL(value: string) {
  if (value === '') return true;
  if (value !== value.trim()) return false;
  if (!value.startsWith('http://') && !value.startsWith('https://')) return false;
  if (value.includes('\\') || /%(?![0-9A-Fa-f]{2})/.test(value)) return false;

  try {
    const url = new URL(value);
    return (
      (url.protocol === 'http:' || url.protocol === 'https:') &&
      url.hostname !== '' &&
      url.username === '' &&
      url.password === ''
    );
  } catch {
    return false;
  }
}

type MemoryFabricConfigPanelProps = {
  disabled?: boolean;
  onSavingChange?: (saving: boolean) => void;
};

export const MemoryFabricConfigPanel = ({
  disabled = false,
  onSavingChange
}: MemoryFabricConfigPanelProps) => {
  const { t } = useTranslation();
  const [form] = Form.useForm<MemoryFabricConfigFormValues>();
  const [modal, modalContextHolder] = Modal.useModal();

  const [config, setConfig] = useState<MemoryFabricConfig>();
  const [isConfigLoading, setIsConfigLoading] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isDirty, setIsDirty] = useState(false);
  const [activeSections, setActiveSections] = useState<string[]>([]);

  const mountedRef = useRef(false);
  const loadingRef = useRef(false);
  const savingRef = useRef(false);
  const tRef = useRef(t);
  tRef.current = t;

  const applyConfig = useCallback(
    (nextConfig: MemoryFabricConfig) => {
      const safeConfig = sanitizeConfig(nextConfig);
      form.resetFields();
      form.setFieldsValue(configToFormValues(safeConfig));
      setConfig(safeConfig);
      setIsDirty(false);
    },
    [form]
  );

  const loadConfig = useCallback(async () => {
    if (loadingRef.current || savingRef.current) return false;

    loadingRef.current = true;
    if (mountedRef.current) setIsConfigLoading(true);

    try {
      const response = await api.getMemoryFabricConfig();
      if (response.code !== 0 || !response.data?.config) {
        if (mountedRef.current) {
          feedback.error(tRef.current(`${configTranslationRoot}.loadFailed`));
        }
        return false;
      }

      if (mountedRef.current) applyConfig(response.data.config);
      return true;
    } catch {
      if (mountedRef.current) {
        feedback.error(tRef.current(`${configTranslationRoot}.loadFailed`));
      }
      return false;
    } finally {
      loadingRef.current = false;
      if (mountedRef.current) setIsConfigLoading(false);
    }
  }, [applyConfig]);

  useEffect(() => {
    mountedRef.current = true;
    void loadConfig();

    return () => {
      mountedRef.current = false;
    };
  }, [loadConfig]);

  const showConflictModal = useCallback(() => {
    modal.confirm({
      title: t(`${configTranslationRoot}.conflictTitle`),
      content: t(`${configTranslationRoot}.conflictDescription`),
      okText: t(`${configTranslationRoot}.conflictReload`),
      closable: true,
      maskClosable: true,
      onOk: loadConfig
    });
  }, [loadConfig, modal, t]);

  const requestReload = useCallback(() => {
    if (disabled || loadingRef.current || savingRef.current) return;

    if (!isDirty) {
      void loadConfig();
      return;
    }

    modal.confirm({
      title: t(`${configTranslationRoot}.discardTitle`),
      content: t(`${configTranslationRoot}.discardDescription`),
      okText: t(`${configTranslationRoot}.discardConfirm`),
      closable: true,
      maskClosable: true,
      onOk: loadConfig
    });
  }, [disabled, isDirty, loadConfig, modal, t]);

  const saveConfig = useCallback(async () => {
    if (disabled || !config || loadingRef.current || savingRef.current) return;

    savingRef.current = true;
    setIsSaving(true);
    onSavingChange?.(true);

    try {
      const values = await form.validateFields();
      const response = await api.setMemoryFabricConfig(formValuesToConfig(config, values));
      if (!mountedRef.current) return;

      if (response.code === -3) {
        showConflictModal();
        return;
      }
      if (response.code !== 0 || !response.data?.config) {
        feedback.error(t(`${configTranslationRoot}.saveFailed`));
        return;
      }

      applyConfig(response.data.config);
      if (!response.data.changed) {
        feedback.success(t(`${configTranslationRoot}.noChanges`));
      } else if (response.data.restarted) {
        feedback.success(t(`${configTranslationRoot}.savedRestarted`));
      } else {
        feedback.success(t(`${configTranslationRoot}.saved`));
      }
    } catch (error) {
      if (!mountedRef.current) return;
      if (error && typeof error === 'object' && 'errorFields' in error) {
        const firstError = (error as { errorFields?: Array<{ name: Array<string | number> }> })
          .errorFields?.[0];
        if (firstError?.name.length) {
          setActiveSections(['models']);
          window.setTimeout(() => {
            if (mountedRef.current) {
              form.scrollToField(firstError.name, { block: 'center' });
            }
          }, 0);
        }
      } else {
        feedback.error(t(`${configTranslationRoot}.saveFailed`));
      }
    } finally {
      savingRef.current = false;
      if (mountedRef.current) setIsSaving(false);
      onSavingChange?.(false);
    }
  }, [applyConfig, config, disabled, form, onSavingChange, showConflictModal, t]);

  const requiredMessage = t(`${configTranslationRoot}.validation.required`);
  const requiredRule: Rule = { required: true, message: requiredMessage };
  const numberRules = (minimum: number, maximum: number): Rule[] => [
    requiredRule,
    { type: 'number', min: minimum, max: maximum, message: requiredMessage }
  ];

  const sectionItems = [
    {
      key: 'models',
      label: t(`${configTranslationRoot}.sections.models`),
      forceRender: true,
      children: (
        <div className="rounded-lg border border-neutral-700/60 bg-neutral-900/20 p-3 sm:p-4">
          <div className="mb-4 rounded-lg bg-neutral-800/40 px-3 py-2 text-xs leading-5 text-neutral-400">
            {t(`${configTranslationRoot}.openAICompatibleHint`)}
          </div>

          <div className="grid grid-cols-1 gap-x-4 md:grid-cols-2">
            <Form.Item
              name={['models', 'default', 'model']}
              label={t(`${configTranslationRoot}.fields.model`)}
              rules={[{ required: true, whitespace: true, message: requiredMessage }]}
            >
              <Input />
            </Form.Item>

            <Form.Item
              name={['models', 'default', 'baseUrl']}
              label={t(`${configTranslationRoot}.fields.baseUrl`)}
              rules={[
                {
                  validator: async (_, value: string) => {
                    if (!isValidHTTPURL(value ?? '')) {
                      throw new Error(t(`${configTranslationRoot}.validation.invalidUrl`));
                    }
                  }
                }
              ]}
            >
              <Input inputMode="url" placeholder="https://api.openai.com/v1" />
            </Form.Item>

            <Form.Item
              name={['models', 'default', 'maxTokens']}
              label={t(`${configTranslationRoot}.fields.maxTokens`)}
              rules={numberRules(1, maxModelTokens)}
            >
              <InputNumber className="w-full" min={1} max={maxModelTokens} precision={0} />
            </Form.Item>

            <Form.Item
              name={['models', 'default', 'timeoutSeconds']}
              label={t(`${configTranslationRoot}.fields.timeoutSeconds`)}
              rules={numberRules(1, maxModelTimeoutSeconds)}
            >
              <InputNumber className="w-full" min={1} max={maxModelTimeoutSeconds} precision={0} />
            </Form.Item>
          </div>

          <div className="mt-1 rounded-lg bg-neutral-800/40 p-3">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <span className="text-sm font-medium">
                {t(`${configTranslationRoot}.apiKey.title`)}
              </span>
              <Tag
                color={config?.models.default.apiKeyConfigured ? 'green' : 'default'}
                className="m-0"
              >
                {t(
                  `${configTranslationRoot}.apiKey.${
                    config?.models.default.apiKeyConfigured ? 'configured' : 'notConfigured'
                  }`
                )}
              </Tag>
            </div>

            <Form.Item name={['models', 'default', 'apiKeyAction']}>
              <Select
                options={(['keep', 'replace', 'clear'] as const).map((action) => ({
                  value: action,
                  label: t(`${configTranslationRoot}.apiKey.${action}`)
                }))}
              />
            </Form.Item>

            <Form.Item
              noStyle
              shouldUpdate={(previous, current) =>
                previous.models?.default?.apiKeyAction !== current.models?.default?.apiKeyAction
              }
            >
              {({ getFieldValue }) => {
                const action = getFieldValue(['models', 'default', 'apiKeyAction']) as APIKeyAction;

                if (action === 'replace') {
                  return (
                    <Form.Item
                      name={['models', 'default', 'apiKeyReplacement']}
                      label={t(`${configTranslationRoot}.apiKey.replacement`)}
                      preserve={false}
                      rules={[
                        {
                          required: true,
                          message: t(`${configTranslationRoot}.apiKey.required`)
                        }
                      ]}
                    >
                      <Input.Password
                        autoComplete="new-password"
                        placeholder={t(`${configTranslationRoot}.apiKey.replacementPlaceholder`)}
                      />
                    </Form.Item>
                  );
                }

                if (action !== 'clear') return null;

                return (
                  <div className="text-xs text-neutral-500">
                    {t(`${configTranslationRoot}.apiKey.clearHint`)}
                  </div>
                );
              }}
            </Form.Item>
          </div>
        </div>
      )
    }
  ];

  return (
    <div className="border-t border-neutral-700/50 pt-6">
      {modalContextHolder}

      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="text-sm font-medium">{t(`${configTranslationRoot}.title`)}</div>
          <div className="mt-1 text-xs leading-5 text-neutral-500">
            {t(`${configTranslationRoot}.description`)}
          </div>
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-2">
          {isDirty && (
            <span className="text-xs text-amber-500">{t(`${configTranslationRoot}.unsaved`)}</span>
          )}
          <Button disabled={disabled || isSaving} loading={isConfigLoading} onClick={requestReload}>
            {t(`${configTranslationRoot}.reload`)}
          </Button>
          <Button
            type="primary"
            disabled={disabled || !config || isConfigLoading}
            loading={isSaving}
            onClick={saveConfig}
          >
            {t(`${configTranslationRoot}.save`)}
          </Button>
        </div>
      </div>

      <Form<MemoryFabricConfigFormValues>
        form={form}
        layout="vertical"
        disabled={disabled || !config || isConfigLoading || isSaving}
        requiredMark={false}
        onValuesChange={() => {
          if (config && !loadingRef.current && !savingRef.current) setIsDirty(true);
        }}
      >
        <Collapse
          size="small"
          activeKey={activeSections}
          items={sectionItems}
          onChange={(keys) => {
            setActiveSections((Array.isArray(keys) ? keys : [keys]).map(String));
          }}
        />
      </Form>
    </div>
  );
};
