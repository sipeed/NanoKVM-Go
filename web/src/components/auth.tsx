import { ReactNode } from 'react';
import { Navigate } from 'react-router-dom';

import { existToken } from '@/lib/cookie.ts';
import { isMockedDevelopment } from '@/lib/runtime-mode.ts';

export const ProtectedRoute = ({ children }: { children: ReactNode }) => {
  const hasToken = existToken();
  const mockedDevelopment =
    import.meta.env.DEV && isMockedDevelopment(import.meta.env.MODE, import.meta.env.DEV);

  if (!hasToken && !mockedDevelopment) {
    return <Navigate to={'/auth/login'} replace />;
  }

  return children;
};
