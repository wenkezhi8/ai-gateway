import { createI18n } from 'vue-i18n'
import zhCN from '../locales/zh-CN.json'
import enUS from '../locales/en-US.json'

export type MessageSchema = typeof zhCN

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS
}

const getStoredLocale = (): string => {
  const stored = localStorage.getItem('locale')
  if (stored && (stored === 'zh-CN' || stored === 'en-US')) {
    return stored
  }
  
  const browserLang = navigator.language
  if (browserLang.startsWith('zh')) {
    return 'zh-CN'
  }
  return 'en-US'
}

const i18n = createI18n<[MessageSchema], 'zh-CN' | 'en-US'>({
  legacy: false,
  locale: getStoredLocale(),
  fallbackLocale: 'en-US',
  messages,
  globalInjection: true
})

export const setLocale = (locale: 'zh-CN' | 'en-US') => {
  ;(i18n.global.locale as unknown as { value: string }).value = locale
  localStorage.setItem('locale', locale)
  document.querySelector('html')?.setAttribute('lang', locale)
}

export const getCurrentLocale = () => {
  return (i18n.global.locale as unknown as { value: string }).value as 'zh-CN' | 'en-US'
}

export const availableLocales = [
  { code: 'zh-CN', name: '简体中文' },
  { code: 'en-US', name: 'English' }
]

export default i18n
