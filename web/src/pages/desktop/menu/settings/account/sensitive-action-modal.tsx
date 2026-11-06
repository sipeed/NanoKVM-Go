import { useEffect, useRef } from 'react';
import type { ReactNode } from 'react';
import { Input, Modal } from 'antd';
import type { InputRef } from 'antd';
import { useTranslation } from 'react-i18next';

type SensitiveActionModalProps = {
  open: boolean;
  title: string;
  description: ReactNode;
  confirmText: string;
  cancelText?: string;
  danger?: boolean;
  /** When false, hide password field (code-only protection re-auth). Default true. */
  requirePassword?: boolean;
  password?: string;
  onPasswordChange?: (value: string) => void;
  error: string;
  isLoading: boolean;
  isLocked: boolean;
  onSubmit: () => void;
  onCancel: () => void;
  submitDisabled?: boolean;
  extra?: ReactNode;
};

export const SensitiveActionModal = ({
  open,
  title,
  description,
  confirmText,
  cancelText,
  danger = false,
  requirePassword = true,
  password = '',
  onPasswordChange,
  error,
  isLoading,
  isLocked,
  onSubmit,
  onCancel,
  submitDisabled = false,
  extra
}: SensitiveActionModalProps) => {
  const { t } = useTranslation();
  const inputRef = useRef<InputRef>(null);

  useEffect(() => {
    if (!open) return;

    const timer = setTimeout(() => {
      inputRef.current?.focus();
    }, 50);

    return () => clearTimeout(timer);
  }, [open, requirePassword]);

  const passwordMissing = requirePassword && !password;
  const disabled = isLoading || isLocked || passwordMissing || submitDisabled;

  return (
    <Modal
      open={open}
      title={title}
      okText={confirmText}
      cancelText={cancelText ?? t('settings.account.cancelBtn')}
      confirmLoading={isLoading}
      okButtonProps={{ danger, disabled }}
      onOk={onSubmit}
      onCancel={onCancel}
      destroyOnHidden={true}
    >
      <div className="space-y-3 pt-2">
        <div className="text-sm text-neutral-300">{description}</div>

        {requirePassword && (
          <Input.Password
            ref={inputRef}
            value={password}
            autoComplete="current-password"
            placeholder={t('settings.account.currentPassword')}
            disabled={isLoading || isLocked}
            onChange={(event) => onPasswordChange?.(event.target.value)}
            onPressEnter={onSubmit}
          />
        )}

        {extra}

        {error && <div className="text-sm text-red-500">{error}</div>}
      </div>
    </Modal>
  );
};
