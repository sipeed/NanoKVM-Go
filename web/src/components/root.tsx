import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';

import { feedback } from '@/lib/feedback.tsx';

export const Root = () => {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  useEffect(() => {
    const state = location.state as { recoveryCodeUsed?: boolean } | null;
    if (!state?.recoveryCodeUsed) return;

    feedback.warning(t('auth.recoveryCodeUsed'));
    navigate(
      { pathname: location.pathname, search: location.search, hash: location.hash },
      { replace: true, state: null }
    );
  }, [location.hash, location.pathname, location.search, location.state, navigate, t]);

  return (
    <div className="h-screen w-screen">
      <Outlet />
    </div>
  );
};
