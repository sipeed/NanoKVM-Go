import cs_CZ from 'antd/locale/cs_CZ';
import da_DK from 'antd/locale/da_DK';
import de_DE from 'antd/locale/de_DE';
import en_US from 'antd/locale/en_US';
import es_ES from 'antd/locale/es_ES';
import fr_FR from 'antd/locale/fr_FR';
import hu_HU from 'antd/locale/hu_HU';
import id_ID from 'antd/locale/id_ID';
import it_IT from 'antd/locale/it_IT';
import ja_JP from 'antd/locale/ja_JP';
import ko_KR from 'antd/locale/ko_KR';
import nb_NO from 'antd/locale/nb_NO';
import nl_NL from 'antd/locale/nl_NL';
import pl_PL from 'antd/locale/pl_PL';
import ru_RU from 'antd/locale/ru_RU';
import th_TH from 'antd/locale/th_TH';
import uk_UA from 'antd/locale/uk_UA';
import vi_VN from 'antd/locale/vi_VN';
import zh_CN from 'antd/locale/zh_CN';
import zh_TW from 'antd/locale/zh_TW';

import { getLanguage, setLanguage } from '@/lib/localstorage.ts';

export const supportedLanguageKeys = [
  'nl',
  'da',
  'de',
  'en',
  'es',
  'fr',
  'id',
  'it',
  'pl',
  'ru',
  'ko',
  'zh',
  'zh_tw',
  'hu',
  'vi',
  'ja',
  'cz',
  'uk',
  'nb',
  'th'
] as const;

export type SupportedLanguage = (typeof supportedLanguageKeys)[number];

const supportedLanguageSet = new Set<string>(supportedLanguageKeys);

const intlLocales: Record<SupportedLanguage, string> = {
  nl: 'nl-NL',
  da: 'da-DK',
  de: 'de-DE',
  en: 'en-US',
  es: 'es-ES',
  fr: 'fr-FR',
  id: 'id-ID',
  it: 'it-IT',
  pl: 'pl-PL',
  ru: 'ru-RU',
  ko: 'ko-KR',
  zh: 'zh-CN',
  zh_tw: 'zh-TW',
  hu: 'hu-HU',
  vi: 'vi-VN',
  ja: 'ja-JP',
  cz: 'cs-CZ',
  uk: 'uk-UA',
  nb: 'nb-NO',
  th: 'th-TH'
};

const antdLocales: Partial<Record<SupportedLanguage, typeof en_US>> = {
  nl: nl_NL,
  da: da_DK,
  de: de_DE,
  en: en_US,
  es: es_ES,
  fr: fr_FR,
  id: id_ID,
  it: it_IT,
  pl: pl_PL,
  ru: ru_RU,
  ko: ko_KR,
  zh: zh_CN,
  zh_tw: zh_TW,
  hu: hu_HU,
  vi: vi_VN,
  ja: ja_JP,
  cz: cs_CZ,
  uk: uk_UA,
  nb: nb_NO,
  th: th_TH
};

export function normalizeLanguage(value?: string | null): SupportedLanguage {
  if (!value) return 'en';

  const normalized = value.trim().toLowerCase().replace(/-/g, '_');
  if (supportedLanguageSet.has(normalized)) {
    return normalized as SupportedLanguage;
  }

  if (normalized === 'cs' || normalized.startsWith('cs_')) return 'cz';
  if (normalized === 'zh' || normalized === 'zh_cn' || normalized === 'zh_sg') return 'zh';
  if (normalized === 'zh_tw' || normalized === 'zh_hk' || normalized === 'zh_mo') return 'zh_tw';

  const baseLanguage = normalized.split('_')[0];
  if (supportedLanguageSet.has(baseLanguage)) {
    return baseLanguage as SupportedLanguage;
  }

  return 'en';
}

export function getCurrentLanguage(): SupportedLanguage {
  const storedLanguage = getLanguage();
  if (storedLanguage) {
    const normalizedLanguage = normalizeLanguage(storedLanguage);
    if (storedLanguage !== normalizedLanguage) {
      setLanguage(normalizedLanguage);
    }
    return normalizedLanguage;
  }

  const initialLanguage = normalizeLanguage(window.navigator.language);
  setLanguage(initialLanguage);
  return initialLanguage;
}

export function getIntlLocale(language?: string | null) {
  return intlLocales[normalizeLanguage(language)];
}

export function getAntdLocale(language?: string | null) {
  return antdLocales[normalizeLanguage(language)] ?? en_US;
}
