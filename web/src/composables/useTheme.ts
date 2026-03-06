import { ref, watch, type WatchStopHandle } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'auto'
export type ThemeVariant = 'apple'
export interface ThemeSetting {
  variant: ThemeVariant
  selectedMode: ThemeMode
  effectiveMode: 'light' | 'dark'
}

const THEME_KEY = 'ai-gateway-theme'
const DEFAULT_THEME: ThemeSetting = {
  variant: 'apple',
  selectedMode: 'auto',
  effectiveMode: getSystemMode()
}

// 全局主题状态
const currentTheme = ref<ThemeSetting>(getStoredTheme())
let stopThemeWatcher: WatchStopHandle | null = null
let mediaQueryList: MediaQueryList | null = null
let hasThemeInitialized = false

function getStoredTheme(): ThemeSetting {
  const stored = localStorage.getItem(THEME_KEY)
  if (!stored) return DEFAULT_THEME

  if (stored === 'light' || stored === 'dark' || stored === 'auto') {
    return createThemeSetting('apple', stored)
  }

  try {
    const parsed = JSON.parse(stored) as Partial<ThemeSetting> & { mode?: unknown }
    const variant: ThemeVariant = 'apple'
    const selectedMode = toThemeMode(parsed.selectedMode) ?? toThemeMode(parsed.mode) ?? 'auto'
    return createThemeSetting(variant, selectedMode)
  } catch {
    return DEFAULT_THEME
  }
}

function toThemeMode(value: unknown): ThemeMode | null {
  if (value === 'light' || value === 'dark' || value === 'auto') {
    return value
  }
  return null
}

function getSystemMode(): 'light' | 'dark' {
  if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'dark'
  }
  return 'light'
}

function resolveEffectiveMode(selectedMode: ThemeMode): 'light' | 'dark' {
  return selectedMode === 'auto' ? getSystemMode() : selectedMode
}

function createThemeSetting(variant: ThemeVariant, selectedMode: ThemeMode): ThemeSetting {
  return {
    variant,
    selectedMode,
    effectiveMode: resolveEffectiveMode(selectedMode)
  }
}

function applyTheme(setting: ThemeSetting) {
  const effectiveMode = resolveEffectiveMode(setting.selectedMode)

  // CSS 样式统一基于 data-theme=light|dark 生效，theme-variant 仅作为风格标识保留。
  document.documentElement.setAttribute('data-theme', effectiveMode)
  document.documentElement.setAttribute('data-mode', effectiveMode)
  document.documentElement.setAttribute('data-theme-variant', setting.variant)
}

function persistTheme(setting: ThemeSetting) {
  // 保持存储兼容：沿用 mode 字段，避免破坏历史读取方。
  localStorage.setItem(THEME_KEY, JSON.stringify({
    variant: setting.variant,
    mode: setting.selectedMode
  }))
}

function refreshEffectiveModeIfAuto() {
  if (currentTheme.value.selectedMode !== 'auto') return
  currentTheme.value = createThemeSetting(currentTheme.value.variant, 'auto')
}

function ensureSystemThemeListener() {
  if (!window.matchMedia || mediaQueryList) return
  mediaQueryList = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQueryList.addEventListener('change', () => {
    refreshEffectiveModeIfAuto()
  })
}

function ensureThemeWatcher() {
  if (stopThemeWatcher) return
  stopThemeWatcher = watch(currentTheme, (newTheme) => {
    persistTheme(newTheme)
    applyTheme(newTheme)
  }, { deep: true })
}

function ensureThemeInitialized() {
  if (hasThemeInitialized) return
  hasThemeInitialized = true
  ensureSystemThemeListener()
  ensureThemeWatcher()
  applyTheme(currentTheme.value)
}

export function useTheme() {
  ensureThemeInitialized()

  const setTheme = (mode: ThemeMode) => {
    currentTheme.value = createThemeSetting(currentTheme.value.variant, mode)
  }

  const setVariant = (variant: ThemeVariant) => {
    currentTheme.value = createThemeSetting(variant, currentTheme.value.selectedMode)
  }

  const toggleTheme = () => {
    const modes: ThemeMode[] = ['light', 'dark', 'auto']
    const currentIndex = modes.indexOf(currentTheme.value.selectedMode)
    const nextIndex = (currentIndex + 1) % modes.length
    currentTheme.value = createThemeSetting(currentTheme.value.variant, modes[nextIndex] ?? 'auto')
  }

  const isDark = () => {
    return currentTheme.value.effectiveMode === 'dark'
  }

  return {
    currentTheme,
    setTheme,
    setVariant,
    toggleTheme,
    isDark
  }
}
