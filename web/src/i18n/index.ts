import i18n from 'i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import { initReactI18next } from 'react-i18next'

import en from '../locales/en/common.json'
import zhCN from '../locales/zh-CN/common.json'
import zhTW from '../locales/zh-TW/common.json'
import ja from '../locales/ja/common.json'

export const defaultNS = 'common'
export const resources = {
  en: { common: en },
  'zh-CN': { common: zhCN },
  'zh-TW': { common: zhTW },
  ja: { common: ja },
} as const

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    supportedLngs: ['en', 'zh-CN', 'zh-TW', 'ja'],
    defaultNS,
    ns: [defaultNS],
    interpolation: { escapeValue: false },
    detection: {
      order: ['localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      lookupLocalStorage: 'i18nextLng',
    },
  })

i18n.on('languageChanged', (lng) => {
  document.documentElement.lang = lng
})

document.documentElement.lang = i18n.language

export { i18n }
