import i18n from 'i18next';
import type { Resource, ResourceLanguage } from 'i18next';
import { initReactI18next } from 'react-i18next';

import appMessages from './apps.ts';
import fullscreenMessages from './fullscreen.ts';
import inputProfileMessages from './input-profile.ts';
import { getCurrentLanguage } from './locale.ts';
import onboardingMessages from './onboarding.ts';
import portalMessages from './portal.ts';
import remoteNetworkMessages from './remote-network.ts';
import screenTimelapseRetentionLocaleMessages from './retention.ts';
import screenTimelapseRecordingMessages from './timelapse-recording.ts';
import updateMessages from './update.ts';

type TranslationObject = Record<string, unknown>;

function asTranslationObject(value: unknown): TranslationObject {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as TranslationObject)
    : {};
}

function withScreenTimelapseMessages(
  language: string,
  resource: ResourceLanguage
): ResourceLanguage {
  const message =
    screenTimelapseRetentionLocaleMessages[
      language as keyof typeof screenTimelapseRetentionLocaleMessages
    ];
  if (!message) return resource;

  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  const settings = asTranslationObject(translation.settings);
  const ai = asTranslationObject(settings.ai);

  return {
    ...source,
    translation: {
      ...translation,
      settings: {
        ...settings,
        ai: {
          ...ai,
          screenTimelapseRecording: screenTimelapseRecordingMessages(language),
          retention: message.retention,
          screenTimelapseRecordingCleaned: message.recordingCleaned
        }
      }
    }
  } as ResourceLanguage;
}

function withAppMessages(language: string, resource: ResourceLanguage): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  const settings = asTranslationObject(translation.settings);
  return {
    ...source,
    translation: {
      ...translation,
      settings: { ...settings, apps: appMessages(language) }
    }
  } as ResourceLanguage;
}

function withFullscreenMessages(language: string, resource: ResourceLanguage): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  const fullscreen = asTranslationObject(translation.fullscreen);

  return {
    ...source,
    translation: {
      ...translation,
      fullscreen: { ...fullscreen, ...fullscreenMessages(language) }
    }
  } as ResourceLanguage;
}

function withUpdateMessages(language: string, resource: ResourceLanguage): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  const settings = asTranslationObject(translation.settings);
  const existingUpdate = asTranslationObject(settings.update);
  return {
    ...source,
    translation: {
      ...translation,
      settings: {
        ...settings,
        update: { ...existingUpdate, ...updateMessages(language) }
      }
    }
  } as ResourceLanguage;
}

function withRemoteNetworkMessages(
  language: string,
  resource: ResourceLanguage
): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  const settings = asTranslationObject(translation.settings);

  return {
    ...source,
    translation: {
      ...translation,
      settings: { ...settings, remoteNetwork: remoteNetworkMessages(language) }
    }
  } as ResourceLanguage;
}

function withPortalMessages(language: string, resource: ResourceLanguage): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  return {
    ...source,
    translation: { ...translation, portal: portalMessages(language) }
  } as ResourceLanguage;
}

function withOnboardingMessages(language: string, resource: ResourceLanguage): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  return {
    ...source,
    translation: { ...translation, onboarding: onboardingMessages(language) }
  } as ResourceLanguage;
}

function withInputProfileMessages(language: string, resource: ResourceLanguage): ResourceLanguage {
  const source = resource as unknown as TranslationObject;
  const translation = asTranslationObject(source.translation);
  const mouse = asTranslationObject(translation.mouse);
  return {
    ...source,
    translation: {
      ...translation,
      mouse: { ...mouse, calibration: inputProfileMessages(language) }
    }
  } as ResourceLanguage;
}

function getResources(): Resource {
  const resources: Resource = {};

  const modules: Record<string, Resource> = import.meta.glob('./locales/*.ts', { eager: true });

  for (const path in modules) {
    const moduleName = path.split('/').pop()?.replace('.ts', '');
    if (moduleName) {
      resources[moduleName] = withRemoteNetworkMessages(
        moduleName,
        withInputProfileMessages(
          moduleName,
          withOnboardingMessages(
            moduleName,
            withPortalMessages(
              moduleName,
              withUpdateMessages(
                moduleName,
                withFullscreenMessages(
                  moduleName,
                  withAppMessages(
                    moduleName,
                    withScreenTimelapseMessages(moduleName, modules[path].default)
                  )
                )
              )
            )
          )
        )
      );
    }
  }

  return resources;
}

const resources = getResources();
const lng = getCurrentLanguage();

i18n
  .use(initReactI18next)
  .init({
    resources,
    lng,
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false
    }
  })
  .then();

export default i18n;
