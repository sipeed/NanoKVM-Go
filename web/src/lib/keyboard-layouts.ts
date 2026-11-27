export const keyboardLayoutGroups = {
  en: ['en-US', 'en-GB'],
  fr: ['fr-CA', 'fr-CA-multilingual', 'fr-FR'],
  de: ['de-DE'],
  es: ['es-ES'],
  zh: ['zh-CN'],
  ru: ['ru-RU'],
  fa: ['fa-IR', 'fa-IR-standard']
} as const;

export type KeyboardLayout =
  (typeof keyboardLayoutGroups)[keyof typeof keyboardLayoutGroups][number];
