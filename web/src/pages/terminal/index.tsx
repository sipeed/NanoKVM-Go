import { useCallback, useEffect, useRef, useState } from 'react';
import { LockOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { AttachAddon } from '@xterm/addon-attach';
import { FitAddon } from '@xterm/addon-fit';
import { Terminal as XtermTerminal } from '@xterm/xterm';
import { useTranslation } from 'react-i18next';

import '@xterm/xterm/css/xterm.css';

import { Button, Form, Input, Modal, Typography } from 'antd';
import type { InputRef } from 'antd';

import * as authApi from '@/api/auth.ts';
import * as vmApi from '@/api/vm.ts';
import { handleAuthSessionClose } from '@/lib/auth-session.ts';
import { feedback } from '@/lib/feedback.tsx';
import { getBaseUrl } from '@/lib/service.ts';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';
import { Head } from '@/components/head.tsx';

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

type TerminalAuthRequirements = {
  secondFactorRequired: boolean;
};

async function resolveTerminalAuthRequirements(): Promise<TerminalAuthRequirements> {
  const statusRsp = await authApi.getTwoFactorStatus();
  const status = statusRsp.code === 0 && isRecord(statusRsp.data) ? statusRsp.data : null;
  if (!status || typeof status.enabled !== 'boolean') {
    throw new Error('two-factor status unavailable');
  }
  if (!status.enabled) {
    return { secondFactorRequired: false };
  }

  // The webTerminal protection policy decides whether opening the terminal also
  // requires an authentication/recovery code. Fail closed when it cannot be read.
  try {
    const protectionRsp = await authApi.getTwoFactorProtection();
    const policy =
      protectionRsp.code === 0 &&
      isRecord(protectionRsp.data) &&
      isRecord(protectionRsp.data.policy)
        ? protectionRsp.data.policy
        : null;
    const webTerminal = policy ? policy.webTerminal : null;
    return { secondFactorRequired: typeof webTerminal === 'boolean' ? webTerminal : true };
  } catch {
    return { secondFactorRequired: true };
  }
}

export const Terminal = () => {
  const { t } = useTranslation();
  const authRateLimit = useAuthRateLimit();

  const [authed, setAuthState] = useState(false);
  const [passwd, setPasswd] = useState('');
  const [code, setCode] = useState('');
  const [authRequirements, setAuthRequirements] = useState<TerminalAuthRequirements | null>(null);
  const [isResolvingOptions, setIsResolvingOptions] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [modal, setModal] = useState(true);
  const modalRef = useRef(modal);
  const authedRef = useRef(authed);
  const passwordInputRef = useRef<InputRef>(null);
  const terminalRef = useRef<XtermTerminal | null>(null);
  const terminalAttachedRef = useRef(false);
  const terminalModalClosedRef = useRef(false);
  const terminalFocusFrameRef = useRef<number | null>(null);
  const terminalHasAutoFocusedRef = useRef(false);

  modalRef.current = modal;
  authedRef.current = authed;

  const cancelPendingTerminalFocus = useCallback(() => {
    if (terminalFocusFrameRef.current === null) return;
    cancelAnimationFrame(terminalFocusFrameRef.current);
    terminalFocusFrameRef.current = null;
  }, []);

  const focusTerminalWhenReady = useCallback(() => {
    const terminal = terminalRef.current;
    if (
      terminalHasAutoFocusedRef.current ||
      terminalFocusFrameRef.current !== null ||
      !terminalAttachedRef.current ||
      !terminalModalClosedRef.current ||
      !terminal
    ) {
      return;
    }

    const focusFrame = requestAnimationFrame(() => {
      if (terminalFocusFrameRef.current !== focusFrame) return;
      terminalFocusFrameRef.current = null;
      if (
        terminalHasAutoFocusedRef.current ||
        !terminalAttachedRef.current ||
        !terminalModalClosedRef.current ||
        terminalRef.current !== terminal
      ) {
        return;
      }

      terminalHasAutoFocusedRef.current = true;
      terminal.focus();
    });
    terminalFocusFrameRef.current = focusFrame;
  }, []);

  const refreshAuthOptions = useCallback(() => {
    setIsResolvingOptions(true);
    void resolveTerminalAuthRequirements()
      .then((requirements) => {
        setAuthRequirements(requirements);
      })
      .catch(() => {
        setAuthRequirements(null);
      })
      .finally(() => {
        setIsResolvingOptions(false);
      });
  }, []);

  useEffect(() => {
    if (modal && !authed) {
      refreshAuthOptions();
    }
  }, [modal, authed, refreshAuthOptions]);

  const secondFactorRequired = authRequirements?.secondFactorRequired === true;

  useEffect(() => {
    if (
      !modal ||
      authed ||
      isResolvingOptions ||
      isSubmitting ||
      authRateLimit.isLocked ||
      authRequirements === null
    ) {
      return;
    }

    const focusFrame = requestAnimationFrame(() => {
      const input = passwordInputRef.current;
      if (!input?.input || input.input.disabled) return;
      input.focus();
    });
    return () => cancelAnimationFrame(focusFrame);
  }, [modal, authed, isResolvingOptions, isSubmitting, authRateLimit.isLocked, authRequirements]);

  useEffect(() => {
    // Keep the terminal lifecycle behind secondary authentication. Besides avoiding
    // unnecessary work, this prevents Xterm's deferred viewport setup from
    // running against a Strict Mode cleanup before the user can use it.
    if (!authed) return;

    const terminalEle = document.getElementById('terminal');
    if (!terminalEle) return;

    const terminal = new XtermTerminal({
      cursorBlink: true
    });
    terminalRef.current = terminal;
    terminalAttachedRef.current = false;
    terminalHasAutoFocusedRef.current = false;

    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalEle);
    fitAddon.fit();

    const url = `${getBaseUrl('ws')}/api/vm/terminal`;

    const ws = new WebSocket(url);
    let disposed = false;

    ws.onopen = () => {
      if (disposed || terminalRef.current !== terminal) {
        ws.close();
        return;
      }

      const attachAddon = new AttachAddon(ws);
      terminal.loadAddon(attachAddon);
      terminalAttachedRef.current = true;
      focusTerminalWhenReady();

      sendSize();
    };

    ws.onclose = (event) => {
      if (terminalRef.current === terminal) {
        terminalAttachedRef.current = false;
        terminalModalClosedRef.current = false;
        cancelPendingTerminalFocus();
      }
      if (handleAuthSessionClose(event)) return;

      if (!disposed) {
        authedRef.current = false;
        modalRef.current = true;
        setAuthState(false);
        setModal(true);
      }
    };

    const sendSize = () => {
      const windowSize = { rows: terminal.rows, cols: terminal.cols };
      const blob = new Blob([JSON.stringify(windowSize)], { type: 'application/json' });
      ws.send(blob);
    };

    const cleanupConnection = () => {
      setTimeout(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.close();
        }
      }, 100);
    };

    const resizeScreen = () => {
      fitAddon.fit();
      sendSize();
    };

    const handleBeforeUnload = () => {
      cleanupConnection();
    };

    window.addEventListener('resize', resizeScreen, false);
    window.addEventListener('beforeunload', handleBeforeUnload);

    return () => {
      disposed = true;
      if (terminalRef.current === terminal) {
        cancelPendingTerminalFocus();
        terminalRef.current = null;
        terminalAttachedRef.current = false;
        terminalHasAutoFocusedRef.current = false;
      }
      terminal.dispose();

      cleanupConnection();

      window.removeEventListener('resize', resizeScreen, false);
      window.removeEventListener('beforeunload', handleBeforeUnload);
    };
  }, [authed, cancelPendingTerminalFocus, focusTerminalWhenReady]);

  function applyAuthSuccess() {
    terminalModalClosedRef.current = false;
    cancelPendingTerminalFocus();
    authedRef.current = true;
    modalRef.current = false;
    setAuthState(true);
    setModal(false);
    setPasswd('');
    setCode('');
  }

  function applyAuthFailure(msg?: string) {
    terminalModalClosedRef.current = false;
    cancelPendingTerminalFocus();
    authedRef.current = false;
    modalRef.current = true;
    setPasswd('');
    setCode('');
    feedback.error(msg || t('terminal.authFailed'));
    setAuthState(false);
    setModal(true);
    refreshAuthOptions();
  }

  const handlerAuthSubmit = async () => {
    const trimmedCode = code.trim();
    if (
      isSubmitting ||
      isResolvingOptions ||
      authRateLimit.isLocked ||
      authRequirements === null ||
      !passwd ||
      (secondFactorRequired && !trimmedCode)
    ) {
      return;
    }
    setIsSubmitting(true);
    try {
      const verifyRsp = await authApi.verifySensitiveAction(
        'open_web_terminal',
        passwd,
        secondFactorRequired ? trimmedCode : ''
      );
      const data = verifyRsp.data as authApi.SensitiveActionData | undefined;
      if (verifyRsp.code !== 0 || !data?.challenge) {
        const rateLimit = authApi.getAuthRateLimitData(verifyRsp);
        if (rateLimit && authRateLimit.lock(rateLimit)) {
          applyAuthFailure(t('auth.rateLimited', { seconds: rateLimit.retryAfter }));
        } else {
          applyAuthFailure();
        }
        return;
      }

      const terminalRsp = await vmApi.terminalAuth(data.challenge);
      if (terminalRsp.code !== 0) {
        applyAuthFailure();
        return;
      }
      authRateLimit.clear();
      applyAuthSuccess();
    } catch {
      applyAuthFailure();
    } finally {
      setIsSubmitting(false);
    }
  };

  const submitDisabled =
    isSubmitting ||
    isResolvingOptions ||
    authRateLimit.isLocked ||
    authRequirements === null ||
    !passwd ||
    (secondFactorRequired && !code.trim());

  const statusUnavailable = authRequirements === null && !isResolvingOptions;

  function handleModalOpenChange(open: boolean) {
    if (open || modalRef.current || !authedRef.current) {
      terminalModalClosedRef.current = false;
      cancelPendingTerminalFocus();
      return;
    }

    terminalModalClosedRef.current = true;
    focusTerminalWhenReady();
  }

  return (
    <>
      <Head title={t('head.terminal')} />
      <Modal
        open={modal}
        width={360}
        centered
        title={t('terminal.authorizeTitle')}
        closable={false}
        maskClosable={false}
        keyboard={false}
        afterOpenChange={handleModalOpenChange}
        footer={
          <Button
            type="primary"
            className="w-full"
            loading={isSubmitting || isResolvingOptions}
            disabled={statusUnavailable ? isSubmitting : submitDisabled}
            onClick={statusUnavailable ? refreshAuthOptions : handlerAuthSubmit}
          >
            {statusUnavailable
              ? t('settings.account.twoFactorProtection.retry')
              : t('terminal.confirm')}
          </Button>
        }
        styles={{
          body: { paddingTop: 8 },
          footer: { marginTop: 20 }
        }}
      >
        <Form style={{ minWidth: 300, maxWidth: 500 }}>
          {statusUnavailable && (
            <Typography.Text type="danger" className="mb-4 block">
              {t('terminal.authFailed')}
            </Typography.Text>
          )}
          <Form.Item>
            <Input.Password
              ref={passwordInputRef}
              prefix={<LockOutlined />}
              placeholder={t('settings.account.currentPassword')}
              value={passwd}
              onChange={(event) => setPasswd(event.target.value)}
              autoComplete="current-password"
              aria-label={t('terminal.password')}
              onPressEnter={handlerAuthSubmit}
              disabled={
                isSubmitting ||
                isResolvingOptions ||
                authRateLimit.isLocked ||
                authRequirements === null
              }
            />
          </Form.Item>

          {secondFactorRequired && (
            <>
              <Typography.Text type="secondary" className="mb-4 block">
                {t('terminal.codeAccountHint')}
              </Typography.Text>
              <Form.Item>
                <Input
                  prefix={<SafetyCertificateOutlined />}
                  value={code}
                  autoComplete="one-time-code"
                  placeholder={t('settings.account.authCodeOrRecoveryCode')}
                  onChange={(val) => setCode(val.target.value)}
                  aria-label={t('terminal.authCode')}
                  onPressEnter={handlerAuthSubmit}
                  disabled={isSubmitting || isResolvingOptions || authRateLimit.isLocked}
                />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>

      <div className="h-full w-full overflow-hidden">
        <div id="terminal" className="h-full p-2"></div>
      </div>
    </>
  );
};
