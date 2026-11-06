import { ChangeEvent, useRef, useState } from 'react';
import { UploadOutlined } from '@ant-design/icons';
import { Button, Divider, Input, Modal, Popconfirm } from 'antd';
import clsx from 'clsx';
import { ChevronRightIcon, FileJsonIcon } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import * as authApi from '@/api/auth.ts';
import * as scriptApi from '@/api/script.ts';
import { feedback } from '@/lib/feedback.tsx';
import { MenuItem } from '@/components/menu-item.tsx';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';

import { Run } from './run';

type PendingAction =
  | { kind: 'run'; script: string; type: string }
  | { kind: 'upload'; file: File }
  | { kind: 'delete'; script: string };

function isTwoFactorStatus(value: unknown): value is authApi.TwoFactorStatus {
  if (!value || typeof value !== 'object') return false;
  return typeof (value as authApi.TwoFactorStatus).enabled === 'boolean';
}

function isTwoFactorProtectionStatus(value: unknown): value is authApi.TwoFactorProtectionStatus {
  if (!value || typeof value !== 'object') return false;
  const policy = (value as authApi.TwoFactorProtectionStatus).policy;
  if (!policy || typeof policy !== 'object') return false;
  return typeof policy.scripts === 'boolean';
}

async function resolveScriptGates(): Promise<{
  twoFactorEnabled: boolean;
  requireRunChallenge: boolean;
  requireUploadChallenge: boolean;
  requireDeleteChallenge: boolean;
}> {
  try {
    const statusRsp = await authApi.getTwoFactorStatus();
    const status = isTwoFactorStatus(statusRsp.data) ? statusRsp.data : null;
    const twoFactorEnabled = statusRsp.code === 0 && status?.enabled === true;
    if (!twoFactorEnabled) {
      // Match backend: execution still password-gated; upload/delete stay open.
      return {
        twoFactorEnabled: false,
        requireRunChallenge: true,
        requireUploadChallenge: false,
        requireDeleteChallenge: false
      };
    }

    try {
      const protectionRsp = await authApi.getTwoFactorProtection();
      const protection = isTwoFactorProtectionStatus(protectionRsp.data)
        ? protectionRsp.data
        : null;
      if (protectionRsp.code === 0 && protection) {
        const requireChallenge = protection.policy.scripts;
        return {
          twoFactorEnabled: true,
          requireRunChallenge: requireChallenge,
          requireUploadChallenge: requireChallenge,
          requireDeleteChallenge: requireChallenge
        };
      }
    } catch {
      // fail closed when protection cannot be loaded
    }

    return {
      twoFactorEnabled: true,
      requireRunChallenge: true,
      requireUploadChallenge: true,
      requireDeleteChallenge: true
    };
  } catch {
    return {
      twoFactorEnabled: true,
      requireRunChallenge: true,
      requireUploadChallenge: true,
      requireDeleteChallenge: true
    };
  }
}

function sensitiveActionFor(action: PendingAction): 'run_script' | 'upload_script' | 'delete_script' {
  switch (action.kind) {
    case 'run':
      return 'run_script';
    case 'upload':
      return 'upload_script';
    case 'delete':
      return 'delete_script';
  }
}

export const Script = () => {
  const { t } = useTranslation();

  const [scripts, setScripts] = useState<string[]>([]);
  const [currentScript, setCurrentScript] = useState('');
  const [isRunning, setIsRunning] = useState(false);
  const [runningScript, setRunningScript] = useState('');
  const [runChallenge, setRunChallenge] = useState('');
  const [isUploading, setIsUploading] = useState(false);
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [verifyPassword, setVerifyPassword] = useState('');
  const [verifyCode, setVerifyCode] = useState('');
  const [verifyError, setVerifyError] = useState('');
  const [isVerifying, setIsVerifying] = useState(false);
  const [isTwoFactorEnabled, setIsTwoFactorEnabled] = useState(false);
  const [isGateLoading, setIsGateLoading] = useState(false);
  const verifyRateLimit = useAuthRateLimit();
  const inputRef = useRef<HTMLInputElement | null>(null);

  function handleOpenChange(open: boolean) {
    if (open) {
      getScripts();
    } else {
      setCurrentScript('');
    }
  }

  function selectFile() {
    if (inputRef.current) {
      inputRef.current.value = '';
    }

    inputRef.current?.click();
  }

  function clearVerification() {
    setPendingAction(null);
    setVerifyPassword('');
    setVerifyCode('');
    setVerifyError('');
    setIsTwoFactorEnabled(false);
    setIsGateLoading(false);
    verifyRateLimit.clear();
  }

  function closeVerification() {
    if (isVerifying) return;
    clearVerification();
  }

  function openVerification(action: PendingAction, twoFactorEnabled: boolean) {
    setPendingAction(action);
    setVerifyPassword('');
    setVerifyCode('');
    setVerifyError('');
    setIsTwoFactorEnabled(twoFactorEnabled);
    setIsGateLoading(false);
    verifyRateLimit.clear();
  }

  function completeUpload(file: File, challenge?: string) {
    setIsUploading(true);

    const formData = new FormData();
    formData.append('file', file);
    if (challenge) {
      formData.append('challenge', challenge);
    }

    scriptApi
      .uploadScript(formData)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('script.uploadFailed'));
          return;
        }

        if (!scripts.includes(rsp.data.file)) {
          setScripts([...scripts, rsp.data.file]);
        }
      })
      .catch(() => {
        feedback.error(t('script.uploadFailed'));
      })
      .finally(() => {
        setIsUploading(false);
      });
  }

  function executeScript(script: string, type: string, challenge?: string) {
    if (type === 'foreground') {
      setRunningScript(script);
      setRunChallenge(challenge ?? '');
      setIsRunning(true);
      return;
    }

    scriptApi
      .runScript(script, type, challenge)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('script.runFailed'));
        }
      })
      .catch(() => {
        feedback.error(t('script.runFailed'));
      });
  }

  function uploadFile(e: ChangeEvent<HTMLInputElement>) {
    if (!e.target?.files?.length) return;
    const file = e.target.files[0];
    if (isUploading || isGateLoading || isVerifying) return;

    setIsGateLoading(true);
    void resolveScriptGates()
      .then((gates) => {
        if (!gates.requireUploadChallenge) {
          completeUpload(file);
          return;
        }
        openVerification({ kind: 'upload', file }, gates.twoFactorEnabled);
      })
      .finally(() => {
        setIsGateLoading(false);
      });
  }

  function runScript(type: string) {
    if (!currentScript || isGateLoading || isVerifying) return;

    const script = currentScript;
    setIsGateLoading(true);
    void resolveScriptGates()
      .then((gates) => {
        if (!gates.requireRunChallenge) {
          executeScript(script, type);
          return;
        }
        openVerification({ kind: 'run', script, type }, gates.twoFactorEnabled);
      })
      .finally(() => {
        setIsGateLoading(false);
      });
  }

  function submitVerification() {
    const password = verifyPassword;
    const code = verifyCode.trim();
    const action = pendingAction;
    if (isVerifying || isGateLoading || verifyRateLimit.isLocked || !action) {
      return;
    }
    // 2FA-on protection gates: code only. TOTP-off historical run gate: password only.
    if (isTwoFactorEnabled) {
      if (!code) return;
    } else if (!password) {
      return;
    }

    const sensitiveAction = sensitiveActionFor(action);

    setIsVerifying(true);
    setVerifyError('');

    authApi
      .verifySensitiveAction(
        sensitiveAction,
        isTwoFactorEnabled ? undefined : password,
        isTwoFactorEnabled ? code : ''
      )
      .then((rsp) => {
        const data = rsp.data as authApi.SensitiveActionData | undefined;
        if (rsp.code !== 0 || !data?.challenge) {
          const rateLimit = authApi.getAuthRateLimitData(rsp);
          if (rateLimit && verifyRateLimit.lock(rateLimit)) {
            setVerifyCode('');
            setVerifyError(t('auth.rateLimited', { seconds: rateLimit.retryAfter }));
            return;
          }

          setVerifyCode('');
          setVerifyError(t('auth.verificationFailed'));
          return;
        }

        const challenge = data.challenge;
        clearVerification();
        if (action.kind === 'run') {
          executeScript(action.script, action.type, challenge);
        } else if (action.kind === 'upload') {
          completeUpload(action.file, challenge);
        } else {
          completeDelete(action.script, challenge);
        }
      })
      .catch(() => {
        setVerifyCode('');
        setVerifyError(t('auth.verificationFailed'));
      })
      .finally(() => {
        setIsVerifying(false);
      });
  }

  function getScripts() {
    scriptApi.getScripts().then((rsp) => {
      if (rsp.code !== 0) {
        console.log(rsp.msg);
        return;
      }

      if (rsp.data?.files?.length > 0) {
        setScripts(rsp.data.files);
      }
    });
  }

  function completeDelete(script: string, challenge?: string) {
    scriptApi
      .deleteScript(script, challenge)
      .then((rsp) => {
        if (rsp.code !== 0) {
          console.log(rsp.msg);
          feedback.error(t('script.deleteFailed'));
          return;
        }

        setScripts((prev) => prev.filter((name) => name !== script));
        setCurrentScript((current) => (current === script ? '' : current));
      })
      .catch(() => {
        feedback.error(t('script.deleteFailed'));
      });
  }

  function deleteScript() {
    if (!currentScript || isGateLoading || isVerifying) return;

    const script = currentScript;
    setIsGateLoading(true);
    void resolveScriptGates()
      .then((gates) => {
        if (!gates.requireDeleteChallenge) {
          completeDelete(script);
          return;
        }
        openVerification({ kind: 'delete', script }, gates.twoFactorEnabled);
      })
      .finally(() => {
        setIsGateLoading(false);
      });
  }

  function activate(script: string) {
    setCurrentScript(script === currentScript ? '' : script);
  }

  const content = (
    <div className="min-w-[250px]">
      <div className="flex items-center justify-between">
        <span className="text-base font-bold text-neutral-300">{t('script.title')}</span>

        <input
          ref={inputRef}
          type="file"
          accept=".sh,.py"
          className="hidden"
          onChange={uploadFile}
        />
        <Button
          ghost
          type="primary"
          size="small"
          icon={<UploadOutlined />}
          loading={isUploading || isGateLoading}
          onClick={selectFile}
        >
          {t('script.upload')}
        </Button>
      </div>
      <Divider style={{ margin: '10px 0 15px 0' }} />
      {scripts.map((script) => (
        <div
          key={script}
          className={clsx(
            'my-1 cursor-pointer rounded',
            script === currentScript ? 'bg-neutral-700/50' : 'hover:bg-neutral-700/70'
          )}
        >
          <div
            className="flex items-center justify-between space-x-5 px-2 py-1.5"
            onClick={() => activate(script)}
          >
            <div className="max-w-[300px] select-none truncate">{script}</div>
            <div className={clsx('h-[16px] w-[16px]', script === currentScript && 'rotate-90')}>
              <ChevronRightIcon size={16} />
            </div>
          </div>

          {script === currentScript && (
            <div className="flex items-center justify-end space-x-2 p-3">
              <Button
                type="primary"
                size="small"
                loading={isGateLoading}
                onClick={() => runScript('foreground')}
              >
                {t('script.run')}
              </Button>
              <Button
                type="primary"
                size="small"
                loading={isGateLoading}
                onClick={() => runScript('background')}
              >
                {t('script.runBackground')}
              </Button>

              <Popconfirm
                title={t('script.attention')}
                description={t('script.delDesc')}
                onConfirm={deleteScript}
                okText={t('script.confirm')}
                cancelText={t('script.cancel')}
                placement="bottom"
                disabled={isGateLoading || isVerifying}
              >
                <Button danger type="primary" size="small" loading={isGateLoading}>
                  {t('script.delete')}
                </Button>
              </Popconfirm>
            </div>
          )}
        </div>
      ))}
    </div>
  );

  return (
    <>
      <MenuItem
        title={t('script.title')}
        icon={<FileJsonIcon size={18} />}
        content={content}
        onOpenChange={handleOpenChange}
      />
      <Modal
        open={pendingAction !== null}
        title={isTwoFactorEnabled ? t('auth.twoFactorTitle') : t('settings.account.currentPassword')}
        okText={t('auth.verify')}
        cancelText={t('auth.cancel')}
        confirmLoading={isVerifying || isGateLoading}
        okButtonProps={{
          danger: true,
          disabled:
            isGateLoading ||
            verifyRateLimit.isLocked ||
            (isTwoFactorEnabled ? !verifyCode.trim() : !verifyPassword)
        }}
        onOk={submitVerification}
        onCancel={closeVerification}
        destroyOnHidden={true}
      >
        <div className="space-y-3 pt-2">
          {isTwoFactorEnabled ? (
            <>
              <div className="text-sm text-neutral-300">{t('auth.sensitiveActionDesc')}</div>
              <Input
                autoFocus
                value={verifyCode}
                autoComplete="one-time-code"
                placeholder={t('settings.account.authCodeOrRecoveryCode')}
                disabled={isVerifying || isGateLoading || verifyRateLimit.isLocked}
                onChange={(event) => setVerifyCode(event.target.value)}
                onPressEnter={submitVerification}
              />
            </>
          ) : (
            <Input.Password
              autoFocus
              value={verifyPassword}
              autoComplete="current-password"
              placeholder={t('settings.account.currentPassword')}
              disabled={isVerifying || isGateLoading || verifyRateLimit.isLocked}
              onChange={(event) => setVerifyPassword(event.target.value)}
              onPressEnter={submitVerification}
            />
          )}

          {verifyError && <div className="text-sm text-red-500">{verifyError}</div>}
        </div>
      </Modal>
      {isRunning && (
        <Run script={runningScript} challenge={runChallenge} setIsRunning={setIsRunning} />
      )}
    </>
  );
};
