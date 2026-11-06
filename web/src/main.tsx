import React, { Suspense } from 'react';
import { Spin } from 'antd';
import ReactDOM from 'react-dom/client';
import { ErrorBoundary } from 'react-error-boundary';
import { HelmetProvider } from 'react-helmet-async';

import { App } from './app.tsx';
import { MainError } from './components/main-error.tsx';
import { isMockedDevelopment } from './lib/runtime-mode.ts';

import './i18n';
import './assets/styles/index.css';

const renderApp = () => {
  return ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <Suspense
        fallback={
          <div className="flex h-screen w-screen items-center justify-center">
            <Spin size="large" />
          </div>
        }
      >
        <ErrorBoundary FallbackComponent={MainError}>
          <HelmetProvider>
            <App />
          </HelmetProvider>
        </ErrorBoundary>
      </Suspense>
    </React.StrictMode>
  );
};

const renderBootstrapError = () => {
  return ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode>
      <MainError />
    </React.StrictMode>
  );
};

async function bootstrap() {
  // Keep DEV explicit so production builds can discard the mock import graph.
  if (import.meta.env.DEV && isMockedDevelopment(import.meta.env.MODE, import.meta.env.DEV)) {
    try {
      const { startMockedFrontend } = await import('./mocks/browser.ts');
      await startMockedFrontend();
    } catch (error) {
      console.error('[mocked] Failed to start the frontend mock runtime', error);
      renderBootstrapError();
      return;
    }
  }

  renderApp();
}

void bootstrap();
