import { ref, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'

const THEME_KEY = 'ai-gateway-theme'

// 全局主题状态
const currentTheme = ref<ThemeMode>(getStoredTheme())

function getStoredTheme(): ThemeMode {
  const stored = localStorage.getItem(THEME_KEY)
  if (stored === 'light' || stored === 'dark' || stored === 'auto') {
    return stored
  }
  return 'auto'
}

function getSystemTheme(): 'light' | 'dark' {
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark'
  }
  return 'light'
}

function applyTheme(theme: ThemeMode) {
  const effectiveTheme = theme === 'auto' ? getSystemTheme() : theme

  if (effectiveTheme === 'dark') {
    document.documentElement.setAttribute('data-theme', 'dark')
  } else {
    document.documentElement.removeAttribute('data-theme')
  }
}

// 监听系统主题变化
if (window.matchMedia) {
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (currentTheme.value === 'auto') {
      applyTheme('auto')
    }
  })
}

export function useTheme() {
  // 初始化应用主题
  applyTheme(currentTheme.value)

  // 监听主题变化
  watch(currentTheme, (newTheme) => {
    localStorage.setItem(THEME_KEY, newTheme)
    applyTheme(newTheme)
  })

  const setTheme = (theme: ThemeMode) => {
    currentTheme.value = theme
  }

  const toggleTheme = () => {
    const themes: ThemeMode[] = ['light', 'dark', 'auto']
    const currentIndex = themes.indexOf(currentTheme.value)
    const nextIndex = (currentIndex + 1) % themes.length
    currentTheme.value = themes[nextIndex] as ThemeMode
  }

  const isDark = () => {
    const effectiveTheme = currentTheme.value === 'auto' ? getSystemTheme() : currentTheme.value
    return effectiveTheme === 'dark'
  }

  return {
    currentTheme,
    setTheme,
    toggleTheme,
    isDark
  }
}
