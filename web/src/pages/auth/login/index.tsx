import { useEffect, useRef, useState } from 'react';
import { KeyOutlined, LockOutlined, UserOutlined } from '@ant-design/icons';
import { Button, Checkbox, Form, Input } from 'antd';
import type { InputRef } from 'antd';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';

import * as api from '@/api/auth.ts';
import { existToken, setToken } from '@/lib/cookie.ts';
import * as storage from '@/lib/localstorage.ts';
import { isMockedDevelopment } from '@/lib/runtime-mode.ts';
import { safeLoginRedirect } from '@/lib/safe-redirect.ts';
import { useAuthRateLimit } from '@/hooks/useAuthRateLimit.ts';
import { Head } from '@/components/head.tsx';
import { LanguagePicker } from '@/components/language-picker.tsx';

import { Tips } from './tips.tsx';

type LoginStep = 'password' | 'twoFactor';

type Pending2FA = {
  challenge: string;
  username: string;
};

type LoginValues = {
  username?: string;
  password?: string;
  code?: string;
};

function parseDirectTwoFactorLoginStatus(value: unknown): api.DirectTwoFactorLoginStatus | null {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) return null;
  const enabled = (value as Record<string, unknown>).enabled;
  return typeof enabled === 'boolean' ? { enabled } : null;
}

export const Login = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [form] = Form.useForm();

  const [isLoading, setIsLoading] = useState(false);
  const [msg, setMsg] = useState('');
  const [step, setStep] = useState<LoginStep>('password');
  const [directLoginEnabled, setDirectLoginEnabled] = useState<boolean | null>(null);
  const [pending2FA, setPending2FA] = useState<Pending2FA | null>(null);
  const authRateLimit = useAuthRateLimit();
  const isFormDisabled = isLoading || authRateLimit.isLocked || directLoginEnabled === null;
  const usernameInputRef = useRef<InputRef>(null);
  const passwordInputRef = useRef<InputRef>(null);
  const codeInputRef = useRef<InputRef>(null);
  const initialValuesRef = useRef({
    username: storage.getLoginUsername(),
    password: ''
  });
  const initialValues = initialValuesRef.current;
  const [rememberPassword, setRememberPassword] = useState(false);
  const mockedDevelopment =
    import.meta.env.DEV && isMockedDevelopment(import.meta.env.MODE, import.meta.env.DEV);

  useEffect(() => {
    let cancelled = false;

    if (existToken() && !mockedDevelopment) {
      navigate(safeLoginRedirect(searchParams.get('redirect')), { replace: true });
      return;
    }

    function enablePasswordLogin() {
      const password = storage.getLoginPassword();
      form.setFieldsValue({ password });
      setRememberPassword(Boolean(password));
      setDirectLoginEnabled(false);
    }

    void api
      .getDirectTwoFactorLoginStatus()
      .then((rsp) => {
        if (cancelled) return;
        const status = rsp.code === 0 ? parseDirectTwoFactorLoginStatus(rsp.data) : null;
        if (status?.enabled) {
          setDirectLoginEnabled(true);
          return;
        }

        enablePasswordLogin();
      })
      .catch(() => {
        if (!cancelled) enablePasswordLogin();
      });

    return () => {
      cancelled = true;
    };
  }, [form, mockedDevelopment, navigate, searchParams]);

  useEffect(() => {
    if (msg) {
      const timer = setTimeout(() => setMsg(''), 3000);
      return () => clearTimeout(timer);
    }
  }, [msg]);

  useEffect(() => {
    if (directLoginEnabled === null) return;

    requestAnimationFrame(() => {
      if (directLoginEnabled || step === 'twoFactor') {
        codeInputRef.current?.focus();
        return;
      }

      if (initialValues.username) {
        passwordInputRef.current?.focus();
        const input = passwordInputRef.current?.input;
        input?.setSelectionRange(0, 0);
        return;
      }

      usernameInputRef.current?.focus();
    });
  }, [directLoginEnabled, initialValues.username, step]);

  useEffect(() => {
    if (!authRateLimit.isLocked) return;
    setMsg(t('auth.rateLimited', { seconds: authRateLimit.retryAfter }));
  }, [authRateLimit.isLocked, authRateLimit.retryAfter, t]);

  function saveRememberedPassword(username: string, password: string) {
    if (!rememberPassword) return;

    if (!storage.setLoginPassword(password)) {
      setRememberPassword(false);
      return;
    }

    if (!storage.setLoginUsername(username)) {
      storage.removeLoginPassword();
      setRememberPassword(false);
    }
  }

  function changeRememberPassword(checked: boolean) {
    setRememberPassword(checked);
    if (!checked) storage.removeLoginPassword();
  }

  function login(values: LoginValues) {
    if (isLoading || authRateLimit.isLocked) return;
    setIsLoading(true);

    const username = values.username ?? '';
    const password = values.password ?? '';

    api
      .login(username, password)
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          const rateLimit = api.getAuthRateLimitData(rsp);
          if (rateLimit && authRateLimit.lock(rateLimit)) {
            setMsg(t('auth.rateLimited', { seconds: rateLimit.retryAfter }));
            return;
          }

          setMsg(rsp.code === -2 ? t('auth.invalidUser') : t('auth.error'));
          return;
        }

        const data = rsp.data as api.LoginResponseData;
        if ((mockedDevelopment && !data.requires2FA) || data.token) {
          saveRememberedPassword(username, password);
          finishLogin(data, username, true);
          return;
        }

        if (data.requires2FA && data.challenge) {
          saveRememberedPassword(username, password);
          form.setFieldValue('password', '');
          setPending2FA({ challenge: data.challenge, username });
          setStep('twoFactor');
          setMsg('');
          return;
        }

        setMsg(t('auth.error'));
      })
      .catch(() => {
        setMsg(t('auth.error'));
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function finishLogin(data: api.LoginResponseData, username?: string, reload = false) {
    if (!data.token && !mockedDevelopment) {
      setMsg(t('auth.error'));
      return;
    }

    setMsg('');
    if (data.token) setToken(data.token);
    if (username) storage.setLoginUsername(username);

    navigate(safeLoginRedirect(searchParams.get('redirect')), {
      replace: true,
      state: data.recoveryCodeUsed ? { recoveryCodeUsed: true } : undefined
    });

    if (reload) {
      window.location.reload();
    }
  }

  function handleCodeVerificationFailure(rsp?: { data?: unknown }) {
    form.resetFields(['code']);
    const rateLimit = rsp ? api.getAuthRateLimitData(rsp) : null;
    if (rateLimit && authRateLimit.lock(rateLimit)) {
      setMsg(t('auth.rateLimited', { seconds: rateLimit.retryAfter }));
      return;
    }

    setMsg(t('auth.verificationFailed'));
    requestAnimationFrame(() => codeInputRef.current?.focus());
  }

  function verifyCode(values: LoginValues) {
    if (isLoading || authRateLimit.isLocked || !pending2FA) return;
    setIsLoading(true);

    api
      .verifyLogin2FA(pending2FA.challenge, values.code ?? '')
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          handleCodeVerificationFailure(rsp);
          return;
        }

        finishLogin(rsp.data as api.LoginResponseData, pending2FA.username);
      })
      .catch(() => {
        handleCodeVerificationFailure();
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function loginWithDirectTwoFactor(values: LoginValues) {
    if (isLoading || authRateLimit.isLocked || !directLoginEnabled) return;
    setIsLoading(true);

    api
      .loginWithDirectTwoFactor(values.code ?? '')
      .then((rsp: any) => {
        if (rsp.code !== 0) {
          handleCodeVerificationFailure(rsp);
          return;
        }

        finishLogin(rsp.data as api.LoginResponseData);
      })
      .catch(() => {
        handleCodeVerificationFailure();
      })
      .finally(() => {
        setIsLoading(false);
      });
  }

  function submit(values: LoginValues) {
    if (authRateLimit.isLocked) return;

    if (step === 'twoFactor') {
      verifyCode(values);
      return;
    }

    if (directLoginEnabled) {
      loginWithDirectTwoFactor(values);
      return;
    }

    login(values);
  }

  function backToLogin() {
    form.resetFields(['code']);
    authRateLimit.clear();
    setPending2FA(null);
    setStep('password');
    setMsg('');
  }

  const isDirectLoginLoading = directLoginEnabled === null;
  const isCodeStep = directLoginEnabled === true || step === 'twoFactor';

  return (
    <>
      <Head title={t('head.login')} />

      <div className="relative flex h-screen w-screen flex-col items-center justify-center">
        <div className="absolute right-4 top-4">
          <LanguagePicker />
        </div>

        <Form
          form={form}
          style={{ minWidth: 300, maxWidth: 500 }}
          initialValues={initialValues}
          onFinish={submit}
        >
          <div className="flex flex-col items-center justify-center pb-4">
            <img
              id="logo"
              src="/sipeed.ico"
              alt="Sipeed"
              onClick={(evt) => {
                evt.preventDefault();
                (evt.target as HTMLImageElement).classList.add('animate-spin');
                setTimeout(() => {
                  (evt.target as HTMLImageElement).classList.remove('animate-spin');
                }, 1000);
              }}
            />
          </div>

          {isDirectLoginLoading ? (
            <div aria-busy="true" className="mb-6 space-y-3">
              <div className="h-10 animate-pulse bg-neutral-800" />
              <div className="h-10 animate-pulse bg-neutral-800" />
            </div>
          ) : !isCodeStep ? (
            <>
              <Form.Item
                name="username"
                rules={[{ required: true, message: t('auth.noEmptyUsername'), min: 1 }]}
              >
                <Input
                  ref={usernameInputRef}
                  prefix={<UserOutlined />}
                  placeholder={t('auth.placeholderUsername')}
                  autoComplete="username"
                  disabled={isFormDisabled}
                />
              </Form.Item>

              <Form.Item
                name="password"
                rules={[{ required: true, message: t('auth.noEmptyPassword'), min: 1 }]}
              >
                <Input
                  ref={passwordInputRef}
                  prefix={<LockOutlined />}
                  type="password"
                  placeholder={t('auth.placeholderPassword')}
                  autoComplete="current-password"
                  disabled={isFormDisabled}
                />
              </Form.Item>

              <Form.Item className="-mt-2">
                <Checkbox
                  checked={rememberPassword}
                  disabled={isFormDisabled}
                  onChange={(event) => changeRememberPassword(event.target.checked)}
                >
                  {t('auth.savePassword')}
                </Checkbox>
              </Form.Item>
            </>
          ) : (
            <>
              <div className="mb-4 text-center">
                <div className="text-lg font-medium">{t('auth.twoFactorTitle')}</div>
                <div className="mt-2 text-xs text-neutral-500">{t('auth.twoFactorDesc')}</div>
              </div>

              <Form.Item
                name="code"
                rules={[{ required: true, message: t('auth.noEmptyAuthCode'), min: 1 }]}
              >
                <Input
                  ref={codeInputRef}
                  prefix={<KeyOutlined />}
                  inputMode="text"
                  autoComplete="one-time-code"
                  placeholder={t('auth.placeholderAuthCode')}
                  disabled={isFormDisabled}
                />
              </Form.Item>
            </>
          )}

          <div className="text-red-500">{msg}</div>

          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              className="w-full"
              loading={isLoading}
              disabled={isDirectLoginLoading || isLoading || authRateLimit.isLocked}
            >
              {isCodeStep ? t('auth.verify') : t('auth.loginButtonText')}
            </Button>
          </Form.Item>

          {step === 'twoFactor' && pending2FA && (
            <Form.Item>
              <Button className="w-full" disabled={isLoading} onClick={backToLogin}>
                {t('auth.backToLogin')}
              </Button>
            </Form.Item>
          )}

          {!isDirectLoginLoading && !isCodeStep && (
            <div className="flex justify-end pb-4 text-sm">
              <Tips />
            </div>
          )}
        </Form>
      </div>
    </>
  );
};
