import { ConfigProvider, theme } from 'antd';
import { useTranslation } from 'react-i18next';
import { RouterProvider } from 'react-router-dom';

import { AuthSessionNotice } from '@/components/auth-session-notice.tsx';
import { getAntdLocale } from '@/i18n/locale.ts';

import { router } from './router';

const themeConfig = {
  algorithm: theme.darkAlgorithm,
  components: {
    Collapse: {
      headerPadding: 0,
      contentPadding: 0
    },
    // Feedback at 1100 sits above standalone Ant Design Modals (CSS default
    // 1000) and the menu rail (LAYER_MENU_RAIL = 1000). See lib/layers.ts.
    Message: {
      zIndexPopup: 1100
    },
    // Notification defaults to 2050 in Ant Design; pin it to the same altitude
    // as Message so alerts no longer float above every nested modal.
    Notification: {
      zIndexPopup: 1100
    }
  }
};

// Static message.xxx() / notification.xxx() calls render through a global holder
// that only sees the theme registered here; without this they fall back to the
// light algorithm and Ant Design's default z-index (2010/2050).
ConfigProvider.config({ theme: themeConfig });

export const App = () => {
  const { i18n } = useTranslation();
  const locale = getAntdLocale(i18n.resolvedLanguage || i18n.language);

  return (
    <ConfigProvider locale={locale} theme={themeConfig}>
      <AuthSessionNotice />
      <RouterProvider router={router} />
    </ConfigProvider>
  );
};
