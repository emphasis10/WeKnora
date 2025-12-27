import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN'
import ruRU from './locales/ru-RU'
import enUS from './locales/en-US'
import koKR from './locales/ko-KR'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'ko-KR': koKR
}

// Get saved language from localStorage or use Chinese as default
const savedLocale = localStorage.getItem('locale') || 'zh-CN'
console.log('i18n initialization with language:', savedLocale)

const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'zh-CN',
  globalInjection: true,
  messages
})

export default i18n