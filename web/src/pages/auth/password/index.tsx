import { useEffect, useRef, useState } from 'react';
import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { Button, Card, Form, Input, Result, Spin, Typography } from 'antd';
import { useSetAtom } from 'jotai';
import { useTranslation } from 'react-i18next';
import { useLocation, useNavigate } from 'react-router-dom';

import * as api from '@/api/auth.ts';
import { removeToken } from '@/lib/cookie.ts';
import * as storage from '@/lib/localstorage.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { isSettingsDetailOpenAtom, isSettingsOpenAtom, settingTabAtom } from '@/jotai/settings.ts';
import { Head } from '@/components/head.tsx';

type PasswordLocationState = {
  sensitiveAction?: 'change_password';
  challenge?: string;
  expiresAt?: number;
};

type PasswordFormValues = {
  username: string;
  password: string;
  password2: string;
};

type PasswordPageStatus = 'loading' | 'allowed' | 'required' | 'expired';

export const Password = () => {
  const { t } = useTranslation();
  const [msg, setMsg] = useState('');
  const [status, setStatus] = useState<PasswordPageStatus>('loading');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isTwoFactorEnabled, setIsTwoFactorEnabled] = useState(false);
  const [passwordChallenge, setPasswordChallenge] = useState('');
  const [passwordChallengeExpiresAt, setPasswordChallengeExpiresAt] = useState(0);
  const navigate = useNavigate();
  const location = useLocation();
  const setIsKeyboardEnable = useSetAtom(isKeyboardEnableAtom);
  const setIsSettingsOpen = useSetAtom(isSettingsOpenAtom);
  const setIsSettingsDetailOpen = useSetAtom(isSettingsDetailOpenAtom);
  const setSettingTab = useSetAtom(settingTabAtom);
  const initialLocationStateRef = useRef<PasswordLocationState | null>(
    location.state as PasswordLocationState | null
  );

  useEffect(() => {
    if (!initialLocationStateRef.current) return;

    navigate(
      { pathname: location.pathname, search: location.search, hash: location.hash },
      { replace: true, state: null }
    );
  }, [location.hash, location.pathname, location.search, navigate]);

  useEffect(() => {
    let isMounted = true;

    api
      .getTwoFactorStatus()
      .then((rsp) => {
        if (!isMounted) return;

        if (rsp.code !== 0 || !rsp.data) {
          setStatus('required');
          return;
        }

        const twoFactorStatus = rsp.data as api.TwoFactorStatus;
        setIsTwoFactorEnabled(twoFactorStatus.enabled);
        if (!twoFactorStatus.enabled) {
          setStatus('allowed');
          setPasswordChallenge('');
          setPasswordChallengeExpiresAt(0);
          return;
        }

        const state = initialLocationStateRef.current;
        if (
          state?.sensitiveAction !== 'change_password' ||
          !state.challenge ||
          typeof state.expiresAt !== 'number' ||
          !Number.isFinite(state.expiresAt)
        ) {
          setStatus('required');
          return;
        }

        if (state.expiresAt <= Date.now()) {
          setStatus('expired');
          return;
        }

        setPasswordChallenge(state.challenge);
        setPasswordChallengeExpiresAt(state.expiresAt);
        setStatus('allowed');
      })
      .catch(() => {
        if (!isMounted) return;
        setStatus('required');
      });

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    if (msg) {
      setTimeout(() => setMsg(''), 3000);
    }
  }, [msg]);

  function changePassword(values: PasswordFormValues) {
    if (isSubmitting) return;
    if (isTwoFactorEnabled && !passwordChallenge) {
      setStatus('required');
      return;
    }
    if (isTwoFactorEnabled && passwordChallengeExpiresAt <= Date.now()) {
      setPasswordChallenge('');
      setPasswordChallengeExpiresAt(0);
      setStatus('expired');
      return;
    }

    if (values.password !== values.password2) {
      setMsg(t('auth.differentPassword'));
      return;
    }
    if (!validateString(values.username)) {
      setMsg(t('auth.illegalUsername'));
      return;
    }
    if (!validateString(values.password)) {
      setMsg(t('auth.illegalPassword'));
      return;
    }

    const username = values.username;
    const password = values.password;
    const challenge = passwordChallenge || undefined;
    if (isTwoFactorEnabled) {
      setPasswordChallenge('');
      setPasswordChallengeExpiresAt(0);
    }

    setIsSubmitting(true);

    api
      .changePassword(username, password, challenge)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          if (isTwoFactorEnabled) {
            setStatus('expired');
            setMsg(rsp.code === -2 ? t('auth.passwordVerificationExpired') : t('auth.error'));
            return;
          }

          setMsg(t('auth.error'));
          return;
        }

        storage.removeLoginPassword();
        removeToken();
        navigate('/auth/login', { replace: true });
      })
      .catch(() => {
        if (isTwoFactorEnabled) {
          setStatus('expired');
        }

        setMsg(t('auth.error'));
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  }

  function validateString(str: string) {
    const regex = /['"\\/]/;
    return !regex.test(str);
  }

  function cancel() {
    window.location.replace('/');
  }

  function returnToSettings() {
    setSettingTab('account');
    setIsSettingsDetailOpen(true);
    setIsSettingsOpen(true);
    setIsKeyboardEnable(false);
    navigate('/', { replace: true });
  }

  function renderVerificationRequired() {
    const isExpired = status === 'expired';

    return (
      <Result
        status="warning"
        title={t('auth.changePassword')}
        subTitle={
          isExpired ? t('auth.passwordVerificationExpired') : t('auth.passwordVerificationRequired')
        }
        extra={
          <Button type="primary" onClick={returnToSettings}>
            {t('auth.returnToSettings')}
          </Button>
        }
      />
    );
  }

  function renderPasswordForm() {
    if (status === 'loading') {
      return <Spin size="large" />;
    }

    if (status !== 'allowed') {
      return renderVerificationRequired();
    }

    return (
      <>
        <Form
          style={{ minWidth: 300, maxWidth: 500 }}
          initialValues={{ remember: true }}
          onFinish={changePassword}
        >
          <Form.Item
            name="username"
            rules={[{ required: true, message: t('auth.noEmptyUsername'), min: 1 }]}
          >
            <Input prefix={<UserOutlined />} placeholder={t('auth.placeholderUsername')} />
          </Form.Item>

          <Form.Item
            name="password"
            rules={[{ required: true, message: t('auth.noEmptyPassword'), min: 1 }]}
          >
            <Input
              prefix={<LockOutlined />}
              type="password"
              placeholder={t('auth.placeholderPassword')}
            />
          </Form.Item>

          <Form.Item
            name="password2"
            rules={[{ required: true, message: t('auth.noEmptyPassword'), min: 1 }]}
          >
            <Input
              prefix={<LockOutlined />}
              type="password"
              placeholder={t('auth.placeholderPassword2')}
            />
          </Form.Item>

          <span className="text-red-500">{msg}</span>
          <Form.Item>
            <div className="flex w-full space-x-2">
              <Button type="primary" htmlType="submit" className="w-1/2" loading={isSubmitting}>
                {t('auth.ok')}
              </Button>
              <Button className="w-1/2" disabled={isSubmitting} onClick={cancel}>
                {t('auth.cancel')}
              </Button>
            </div>
          </Form.Item>
        </Form>

        <Card>
          <div className="flex w-[450px] max-w-[calc(100vw-48px)] flex-col space-y-2">
            <div className="flex items-center space-x-1">
              <span>{t('auth.tips.changeReset')}</span>
              <Typography.Link
                href="https://wiki.sipeed.com/hardware/en/kvm/NanoKVM_Go/user_guide.html#Reset-Forgotten-Password"
                target="_blank"
              >
                {t('auth.tips.resetLink')}
              </Typography.Link>
            </div>
          </div>
        </Card>
      </>
    );
  }

  return (
    <>
      <Head title={t('head.changePassword')} />

      <div className="flex h-screen w-screen flex-col items-center justify-center space-y-5">
        <h2 className="text-xl font-semibold text-neutral-100">{t('auth.changePassword')}</h2>
        {renderPasswordForm()}
      </div>
    </>
  );
};
