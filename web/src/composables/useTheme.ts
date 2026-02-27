import { ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ThemeVariant = 'apple' | 'dashboard'
export interface ThemeSetting {
  variant: ThemeVariant
  mode: ThemeMode
}

const THEME_KEY = 'ai-gateway-theme'
const DEFAULT_THEME: ThemeSetting = { variant: 'apple', mode: 'auto' }

// 全局主题状态
const currentTheme = ref<ThemeSetting>(getStoredTheme())

function getStoredTheme(): ThemeSetting {
  const stored = localStorage.getItem(THEME_KEY)
  if (!stored) return DEFAULT_THEME

  if (stored === 'light' || stored === 'dark' || stored === 'auto') {
    return { variant: 'apple', mode: stored }
  }

  try {
    const parsed = JSON.parse(stored) as Partial<ThemeSetting>
    const variant = parsed.variant === 'dashboard' ? 'dashboard' : 'apple'
    const mode = parsed.mode === 'light' || parsed.mode === 'dark' || parsed.mode === 'auto' ? parsed.mode : 'auto'
    return { variant, mode }
  } catch {
    return DEFAULT_THEME
  }
}

function getSystemMode(): 'light' | 'dark' {
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark'
  }
  return 'light'
}

function applyTheme(setting: ThemeSetting) {
  const effectiveMode = setting.mode === 'auto' ? getSystemMode() : setting.mode
  document.documentElement.setAttribute('data-theme', setting.variant)
  document.documentElement.setAttribute('data-mode', effectiveMode)

  if (setting.variant === 'apple' && effectiveMode === 'dark') {
    document.documentElement.setAttribute('data-theme-legacy', 'dark')
  } else {
    document.documentElement.removeAttribute('data-theme-legacy')
  }
}

// 监听系统主题变化
if (window.matchMedia) {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (currentTheme.value.mode === 'auto') {
      applyTheme(currentTheme.value)
    }
  })
}

export function useTheme() {
  // 初始化应用主题
  applyTheme(currentTheme.value)

  // 监听主题变化
  watch(currentTheme, (newTheme) => {
    localStorage.setItem(THEME_KEY, JSON.stringify(newTheme))
    applyTheme(newTheme)
  }, { deep: true })

  const setTheme = (mode: ThemeMode) => {
    currentTheme.value = { ...currentTheme.value, mode }
  }

  const setVariant = (variant: ThemeVariant) => {
    currentTheme.value = { ...currentTheme.value, variant }
  }

  const toggleTheme = () => {
    const modes: ThemeMode[] = ['light', 'dark', 'auto']
    const currentIndex = modes.indexOf(currentTheme.value.mode)
    const nextIndex = (currentIndex + 1) % modes.length
    currentTheme.value = { ...currentTheme.value, mode: modes[nextIndex] ?? 'auto' }
  }

  const isDark = () => {
    const effectiveMode = currentTheme.value.mode === 'auto' ? getSystemMode() : currentTheme.value.mode
    return effectiveMode === 'dark'
  }

  return {
    currentTheme,
    setTheme,
    setVariant,
    toggleTheme,
    isDark
  }
}
