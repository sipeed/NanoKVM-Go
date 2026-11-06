import { createHashRouter, Navigate } from 'react-router-dom';

import { ProtectedRoute } from '@/components/auth';
import { Root } from '@/components/root';

export const router = createHashRouter([
  {
    path: '/auth/login',
    lazy: async () => {
      const { Login } = await import('./pages/auth/login');
      return { Component: Login };
    }
  },
  {
    path: '/',
    element: (
      <ProtectedRoute>
        <Root />
      </ProtectedRoute>
    ),
    children: [
      {
        path: '',
        lazy: async () => {
          const { Desktop } = await import('./pages/desktop');
          return { Component: Desktop };
        }
      },
      {
        // The storage page became the desktop menu's mount and file-manager
        // modals; old links land on the desktop.
        path: 'storage',
        element: <Navigate to="/" replace />
      },
      {
        path: 'terminal',
        lazy: async () => {
          const { Terminal } = await import('./pages/terminal');
          return { Component: Terminal };
        }
      },
      {
        path: 'auth/password',
        lazy: async () => {
          const { Password } = await import('./pages/auth/password');
          return { Component: Password };
        }
      }
    ]
  },
  {
    path: '/wifi',
    caseSensitive: false,
    lazy: async () => {
      const { Wifi } = await import('./pages/wifi');
      return { Component: Wifi };
    }
  }
]);
