import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Modal, Switch } from 'antd';
import { CheckIcon, CopyIcon, EyeIcon, RefreshCcwIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as api from '@/api/ai.ts';
import type { AIConfig } from '@/api/ai.ts';
import { feedback } from '@/lib/feedback.tsx';
import { useProtectionChallengeGate } from '@/hooks/useProtectionChallengeGate.tsx';

function maskKey(key: string) {
  if (!key) {
    return '-';
  }

  if (key.length <= 16) {
    return `${key.slice(0, 4)}...${key.slice(-4)}`;
  }

  return `${key.slice(0, 12)}...${key.slice(-6)}`;
}

type MCPProps = {
  isServiceToggleLocked: boolean;
  runServiceToggle: (action: () => Promise<void>) => Promise<void>;
  onEnabledChange: (enabled: boolean) => void;
};

type MCPKeyModalContentProps = {
  value: string;
};

// Rendered inside a modal on every viewport. The key is allowed to wrap
// freely so it never stretches the layout, and copy feedback stays inside the
// modal because the global toast layer renders below nested modals.
const MCPKeyModalContent = ({ value }: MCPKeyModalContentProps) => {
  const { t } = useTranslation();
  const [isCopied, setIsCopied] = useState(false);
  const [isCopyFailed, setIsCopyFailed] = useState(false);

  async function copyKey() {
    setIsCopyFailed(false);
    try {
      await navigator.clipboard.writeText(value);
      setIsCopied(true);
      window.setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      console.log(err);
      setIsCopyFailed(true);
    }
  }

  return (
    <div className="flex flex-col space-y-3 pt-1">
      {/* The key never scrolls: it wraps freely and the panel grows with it. The
          inline-block spacer right after the text reserves the last-line slot so the
          absolutely docked copy button can never cover key characters. The px min-*
          floor keeps the reserve valid for the fixed 24px button at tiny root font
          sizes; all four classes are load-bearing (see mocked-mode test). */}
      <div className="relative rounded-lg border border-neutral-700/60 bg-neutral-900/70 px-3 py-2.5">
        <span className="select-all break-all font-mono text-sm leading-relaxed text-neutral-200">
          {value}
        </span>
        <span aria-hidden="true" className="inline-block h-5 min-h-[26px] w-10 min-w-[28px]" />
        <Button
          type="text"
          size="small"
          className="absolute bottom-2 right-2 text-neutral-400 hover:text-white"
          aria-label={t('settings.ai.copyApiKey')}
          icon={
            isCopied ? <CheckIcon size={15} className="!text-green-500" /> : <CopyIcon size={15} />
          }
          onClick={copyKey}
        />
      </div>

      {isCopyFailed && (
        <span className="w-full text-xs leading-snug text-red-400">
          {t('clipboard.copyFailed')}
        </span>
      )}
    </div>
  );
};

export const MCP = ({ isServiceToggleLocked, runServiceToggle, onEnabledChange }: MCPProps) => {
  const { t } = useTranslation();
  const [modal, contextHolder] = Modal.useModal();
  const { runWithChallenge, modal: protectionModal, isGateBusy } = useProtectionChallengeGate();

  const [config, setConfig] = useState<AIConfig>({
    mcpEnabled: false
  });
  const [mcpApiKey, setMCPAPIKey] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const isLoadingRef = useRef(false);
  const silentRefreshInFlightRef = useRef(false);
  const configActionVersionRef = useRef(0);
  const [isEndpointCopied, setIsEndpointCopied] = useState(false);
  const [isKeyCopied, setIsKeyCopied] = useState(false);
  const keyModalRef = useRef<{ destroy: () => void } | null>(null);

  const endpoint = useMemo(() => `${window.location.origin}/api/mcp`, []);
  const maskedKey = config.mcpApiKeyMasked || maskKey(mcpApiKey);
  // The full key is always shown in a wrapping modal (any viewport): the nowrap
  // key would stretch the Radix ScrollArea table wrapper and widen the whole
  // settings layout. Inline we only ever render the short masked key.

  const updateLoading = useCallback((loading: boolean) => {
    isLoadingRef.current = loading;
    setIsLoading(loading);
  }, []);

  const getConfig = useCallback(
    (silent = false) => {
      if (silent) {
        if (isLoadingRef.current || silentRefreshInFlightRef.current) {
          return;
        }
        silentRefreshInFlightRef.current = true;
      } else {
        updateLoading(true);
      }
      const actionVersion = configActionVersionRef.current;

      api
        .getAIConfig()
        .then((rsp) => {
          if (rsp.code !== 0) {
            if (!silent) {
              console.log(rsp.msg);
              feedback.error(t('settings.ai.errors.loadMcpConfig'));
            }
            return;
          }

          if (silent && actionVersion !== configActionVersionRef.current) {
            return;
          }

          setConfig(rsp.data);
          onEnabledChange(rsp.data.mcpEnabled);
        })
        .catch(() => {
          if (!silent) {
            feedback.error(t('settings.ai.errors.loadMcpConfig'));
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
    [onEnabledChange, t, updateLoading]
  );

  useEffect(() => {
    getConfig();
    const timer = window.setInterval(() => getConfig(true), 3000);

    return () => window.clearInterval(timer);
  }, [getConfig]);

  // If MCP gets disabled elsewhere while the key modal is open, the card
  // unmounts but the modal would keep showing the plaintext key.
  useEffect(() => {
    if (!config.mcpEnabled) {
      keyModalRef.current?.destroy();
    }
  }, [config.mcpEnabled]);

  async function updateMCPEnabled(mcpEnabled: boolean, challenge?: string) {
    let result: { code: number } | undefined;

    await runServiceToggle(async () => {
      configActionVersionRef.current += 1;
      updateLoading(true);

      try {
        const rsp = await api.setAIConfig(mcpEnabled, challenge);
        result = rsp;
        if (rsp.code === -3) {
          return;
        }
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.updateMcpService'));
          return;
        }

        setConfig(rsp.data);
        onEnabledChange(rsp.data.mcpEnabled);
      } catch {
        feedback.error(t('settings.ai.errors.updateMcpService'));
      } finally {
        updateLoading(false);
      }
    });
    return result;
  }

  function setMCPEnabled() {
    if (isLoadingRef.current || isServiceToggleLocked || isGateBusy) return;

    const mcpEnabled = !config.mcpEnabled;
    void runWithChallenge(mcpEnabled ? 'mcp_enable' : 'mcp_disable', 'mcp', (challenge) =>
      updateMCPEnabled(mcpEnabled, challenge)
    );
  }

  async function copyText(text: string, type: 'endpoint' | 'key') {
    if (!text) return;

    try {
      await navigator.clipboard.writeText(text);
      if (type === 'endpoint') {
        setIsEndpointCopied(true);
        setTimeout(() => setIsEndpointCopied(false), 2000);
      } else {
        setIsKeyCopied(true);
        setTimeout(() => setIsKeyCopied(false), 2000);
      }
    } catch (err) {
      console.log(err);
      feedback.error(t('clipboard.copyFailed'));
    }
  }

  async function fetchMCPAPIKey(challenge?: string) {
    const rsp = await api.getMCPAPIKey(challenge);
    if (rsp.code === 0 && typeof rsp.data?.mcpApiKey === 'string') {
      setMCPAPIKey(rsp.data.mcpApiKey);
    }
    return rsp;
  }

  function showKeyModal(key: string) {
    // Re-tapping the eye button replaces the previous instance instead of stacking.
    keyModalRef.current?.destroy();
    keyModalRef.current = modal.info({
      title: t('settings.ai.apiKey'),
      icon: null,
      centered: true,
      closable: true,
      footer: null,
      width: 420,
      content: <MCPKeyModalContent value={key} />
    });
  }

  function revealKey() {
    if (isLoading || isGateBusy || !config.mcpEnabled) return;

    void runWithChallenge('mcp_show_key', 'mcp', async (challenge) => {
      updateLoading(true);
      try {
        const rsp = await fetchMCPAPIKey(challenge);
        if (rsp.code === -3) return rsp;
        if (rsp.code !== 0 || typeof rsp.data?.mcpApiKey !== 'string') {
          if (rsp.code !== 0) console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.accessMcpApiKey'));
          return rsp;
        }
        showKeyModal(rsp.data.mcpApiKey);
        return rsp;
      } catch {
        feedback.error(t('settings.ai.errors.accessMcpApiKey'));
      } finally {
        updateLoading(false);
      }
    });
  }

  function copyMCPAPIKey() {
    if (isLoading || isGateBusy || !config.mcpEnabled) return;

    void runWithChallenge('mcp_copy_key', 'mcp', async (challenge) => {
      updateLoading(true);
      try {
        const rsp = await api.copyMCPAPIKey(challenge);
        if (rsp.code === -3) return rsp;
        if (rsp.code !== 0 || typeof rsp.data?.mcpApiKey !== 'string') {
          if (rsp.code !== 0) console.log(rsp.msg);
          feedback.error(t('settings.ai.errors.accessMcpApiKey'));
          return rsp;
        }
        setMCPAPIKey(rsp.data.mcpApiKey);
        await copyText(rsp.data.mcpApiKey, 'key');
        return rsp;
      } catch {
        feedback.error(t('settings.ai.errors.accessMcpApiKey'));
      } finally {
        updateLoading(false);
      }
    });
  }

  function regenerateKey() {
    if (isLoading || isGateBusy || !config.mcpEnabled) return;

    modal.confirm({
      title: t('settings.ai.regenerateConfirmTitle'),
      content: (
        <span className="text-xs text-neutral-500">{t('settings.ai.regenerateConfirmDesc')}</span>
      ),
      okText: t('settings.ai.okBtn'),
      cancelText: t('settings.ai.cancelBtn'),
      onOk: async () => {
        await runWithChallenge('mcp_regenerate_key', 'mcp', async (challenge) => {
          configActionVersionRef.current += 1;
          updateLoading(true);

          try {
            let rsp: Awaited<ReturnType<typeof api.regenerateMCPAPIKey>>;
            try {
              rsp = await api.regenerateMCPAPIKey(challenge);
            } catch {
              feedback.error(t('settings.ai.errors.regenerateMcpApiKey'));
              return;
            }

            if (rsp.code === -3) return rsp;
            if (rsp.code !== 0) {
              console.log(rsp.msg);
              feedback.error(t('settings.ai.errors.regenerateMcpApiKey'));
              return rsp;
            }

            setConfig(rsp.data);
            // Regeneration is a sensitive action, but leaving the newly generated
            // credential blank makes the user unable to configure their MCP client.
            // Reuse the dedicated protected read endpoint so the key is exposed only
            // to the same authenticated and authorized browser session.
            try {
              const keyRsp = await fetchMCPAPIKey();
              const newKey = keyRsp.data?.mcpApiKey;
              if (keyRsp.code !== 0 || typeof newKey !== 'string') {
                if (keyRsp.code !== 0) console.log(keyRsp.msg);
                feedback.warning(t('settings.ai.errors.regeneratedMcpApiKeyUnavailable'));
                return rsp;
              }

              // Let the confirm modal close before stacking the key modal on top.
              window.setTimeout(() => showKeyModal(newKey), 0);
              return rsp;
            } catch {
              feedback.warning(t('settings.ai.errors.regeneratedMcpApiKeyUnavailable'));
              return rsp;
            }
          } finally {
            updateLoading(false);
          }
        });
      }
    });
  }

  return (
    <>
      {contextHolder}
      <div className="flex flex-col space-y-6">
        <div className="flex items-center justify-between">
          <div className="flex flex-col space-y-1">
            <span className="text-sm font-medium">{t('settings.ai.mcpService')}</span>
            <span className="text-xs text-neutral-500">{t('settings.ai.mcpServiceDesc')}</span>
          </div>

          <Switch
            checked={config.mcpEnabled}
            disabled={isServiceToggleLocked || isGateBusy}
            loading={isLoading || isGateBusy}
            onChange={setMCPEnabled}
          />
        </div>

        {config.mcpEnabled && (
          <div className="animate-in fade-in slide-in-from-top-2 duration-300">
            <div className="flex flex-col overflow-hidden rounded-xl border border-neutral-700/50 bg-neutral-800/40 shadow-sm">
              <div className="group flex flex-col space-y-2 border-b border-neutral-800 px-4 py-3.5 transition-colors hover:bg-neutral-800/40 sm:flex-row sm:items-center sm:justify-between">
                <span className="w-24 shrink-0 text-sm font-medium text-neutral-400">
                  {t('settings.ai.endpoint')}
                </span>
                <div className="flex min-w-0 items-center justify-between gap-2">
                  <span className="min-w-0 flex-1 select-all truncate font-mono text-sm text-neutral-300">
                    {endpoint}
                  </span>
                  <div className="flex shrink-0 items-center opacity-40 transition-opacity group-hover:opacity-100 sm:ml-4 sm:space-x-1">
                    <Button
                      type="text"
                      size="small"
                      className="text-neutral-400 hover:text-white"
                      aria-label={t('settings.ai.copyEndpoint')}
                      icon={
                        isEndpointCopied ? (
                          <CheckIcon size={15} className="!text-green-500" />
                        ) : (
                          <CopyIcon size={15} />
                        )
                      }
                      onClick={() => copyText(endpoint, 'endpoint')}
                    />
                  </div>
                </div>
              </div>

              <div className="group flex flex-col space-y-2 px-4 py-3.5 transition-colors hover:bg-neutral-800/40 sm:flex-row sm:items-center sm:justify-between">
                <span className="w-24 shrink-0 text-sm font-medium text-neutral-400">
                  {t('settings.ai.apiKey')}
                </span>
                <div className="flex min-w-0 items-center justify-between gap-2">
                  <span className="min-w-0 flex-1 select-all truncate font-mono text-sm text-neutral-300">
                    {maskedKey}
                  </span>
                  <div className="flex shrink-0 items-center space-x-1 opacity-40 transition-opacity group-hover:opacity-100 sm:ml-4">
                    <Button
                      type="text"
                      size="small"
                      className="text-neutral-400 hover:text-white"
                      aria-label={t('settings.ai.showApiKey')}
                      icon={<EyeIcon size={15} />}
                      disabled={isLoading || isGateBusy}
                      onClick={revealKey}
                    />
                    <Button
                      type="text"
                      size="small"
                      className="text-neutral-400 hover:text-white"
                      aria-label={t('settings.ai.copyApiKey')}
                      icon={
                        isKeyCopied ? (
                          <CheckIcon size={15} className="!text-green-500" />
                        ) : (
                          <CopyIcon size={15} />
                        )
                      }
                      disabled={isLoading || isGateBusy}
                      onClick={copyMCPAPIKey}
                    />
                    <Button
                      type="text"
                      size="small"
                      className="text-neutral-400 hover:text-white"
                      aria-label={t('settings.ai.regenerate')}
                      loading={isLoading || isGateBusy}
                      icon={<RefreshCcwIcon size={15} />}
                      onClick={regenerateKey}
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
      {protectionModal}
    </>
  );
};
