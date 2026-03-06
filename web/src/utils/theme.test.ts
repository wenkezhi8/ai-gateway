import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const createLocalStorageMock = () => {
  let store: Record<string, string> = {}
  return {
    getItem: (key: string) => (key in store ? store[key] : null),
    setItem: (key: string, value: string) => {
      store[key] = value
    },
    removeItem: (key: string) => {
      delete store[key]
    },
    clear: () => {
      store = {}
    }
  }
}

const setSystemTheme = (isDark: boolean) => {
  const matchMedia = vi.fn().mockReturnValue({
    matches: isDark,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn()
  })
  window.matchMedia = matchMedia
}

describe('theme', () => {
  beforeEach(() => {
    vi.resetModules()
    const dataset: Record<string, string> = {}
    const documentElement = {
      dataset,
      setAttribute: (key: string, value: string) => {
        if (key.startsWith('data-')) {
          const dataKey = key.replace('data-', '').replace(/-([a-z])/g, (_, c) => c.toUpperCase())
          dataset[dataKey] = value
        }
      },
      removeAttribute: (key: string) => {
        if (key.startsWith('data-')) {
          const dataKey = key.replace('data-', '').replace(/-([a-z])/g, (_, c) => c.toUpperCase())
          delete dataset[dataKey]
        }
      }
    }
    const documentMock = {
      documentElement,
      createElement: () => ({})
    }
    // @ts-expect-error minimal test window mock
    globalThis.window = {
      matchMedia: vi.fn().mockReturnValue({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn()
      })
    }
    // @ts-expect-error minimal test document mock
    globalThis.document = documentMock
    // @ts-expect-error minimal localStorage mock
    globalThis.localStorage = createLocalStorageMock()
  })

  it('normalizes legacy dashboard variant and applies mode theme on init', async () => {
    setSystemTheme(false)
    localStorage.setItem('ai-gateway-theme', JSON.stringify({ variant: 'dashboard', mode: 'dark' }))
    const { useTheme } = await import('../composables/useTheme')
    useTheme()

    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.dataset.mode).toBe('dark')
    expect(document.documentElement.dataset.themeVariant).toBe('apple')
  })

  it('setTheme updates dataset mode and keeps apple variant', async () => {
    setSystemTheme(false)
    localStorage.setItem('ai-gateway-theme', JSON.stringify({ variant: 'apple', mode: 'light' }))
    const { useTheme } = await import('../composables/useTheme')
    const { setTheme } = useTheme()
    setTheme('dark')
    await nextTick()

    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.dataset.themeVariant).toBe('apple')
    const stored = JSON.parse(localStorage.getItem('ai-gateway-theme') || '{}')
    expect(stored.variant).toBe('apple')
    expect(stored.mode).toBe('dark')
  })

  it('auto mode follows system scheme changes and updates theme attribute', async () => {
    const listeners: Array<(event: MediaQueryListEvent) => void> = []
    let isDark = false
    const matchMedia = vi.fn().mockReturnValue({
      get matches() {
        return isDark
      },
      addEventListener: (_: string, cb: (event: MediaQueryListEvent) => void) => listeners.push(cb),
      removeEventListener: vi.fn()
    })
    window.matchMedia = matchMedia

    localStorage.setItem('ai-gateway-theme', JSON.stringify({ variant: 'dashboard', mode: 'auto' }))

    const { useTheme } = await import('../composables/useTheme')
    useTheme()
    expect(document.documentElement.dataset.theme).toBe('light')
    expect(document.documentElement.dataset.mode).toBe('light')
    expect(document.documentElement.dataset.themeVariant).toBe('apple')

    isDark = true
    listeners.forEach(cb => cb({ matches: true } as MediaQueryListEvent))
    await nextTick()
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.dataset.mode).toBe('dark')
  })

  it('registers system theme listener once even when useTheme is called multiple times', async () => {
    const addEventListener = vi.fn()
    window.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener,
      removeEventListener: vi.fn()
    })

    const { useTheme } = await import('../composables/useTheme')
    useTheme()
    useTheme()

    expect(addEventListener).toHaveBeenCalledTimes(1)
  })
})
