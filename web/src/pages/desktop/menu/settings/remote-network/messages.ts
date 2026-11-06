import { createContext, useCallback, useContext } from 'react';
import type { InterpolationOptions, i18n as I18n, TOptions } from 'i18next';
import { useTranslation } from 'react-i18next';

import type { SupportedLanguage } from '@/i18n/locale.ts';

import type { RemoteNetworkComponentMessages, RemoteNetworkMessageBundle } from './types.ts';

/** Translate function handed to remote network components. */
export type RemoteNetworkTranslate = (key: string, options?: TOptions) => string;

/** Messages of the component currently open; set by the section shell. */
export const RemoteNetworkMessagesContext = createContext<
  RemoteNetworkComponentMessages | undefined
>(undefined);

/**
 * Component bundle first, then its English bundle. `undefined` means the key is
 * not component specific, so the caller falls back to the global i18n bundle.
 */
export function resolveRemoteNetworkMessage(
  messages: RemoteNetworkComponentMessages | undefined,
  language: string,
  key: string
): string | undefined {
  const bundle: RemoteNetworkMessageBundle | undefined =
    messages?.[language as SupportedLanguage] ?? messages?.en;

  return bundle?.[key] ?? messages?.en?.[key];
}

function interpolate(i18n: I18n, message: string, options?: TOptions): string {
  if (!options) return message;

  const interpolator = i18n.services?.interpolator;
  if (!interpolator) return message;

  // Same call shape i18next uses internally: data plus interpolation options.
  return interpolator.interpolate(message, options, i18n.language, options as InterpolationOptions);
}

/**
 * Resolve component text. Keys shipped in the component bundle win, everything
 * else (for example `settings.back`) comes from the global i18n bundle, so a new
 * component only needs its own `messages.ts`.
 */
export function useRemoteNetworkMessages(): RemoteNetworkTranslate {
  const { t: translate, i18n } = useTranslation();
  const messages = useContext(RemoteNetworkMessagesContext);
  const language = i18n.language;

  return useCallback<RemoteNetworkTranslate>(
    (key, options) => {
      const message = resolveRemoteNetworkMessage(messages, language, key);
      if (message === undefined) {
        return translate(key, options) as string;
      }

      return interpolate(i18n, message, options);
    },
    [i18n, language, messages, translate]
  );
}
