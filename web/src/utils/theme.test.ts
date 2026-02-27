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

  it('applies stored theme and mode to dataset on init', async () => {
    setSystemTheme(false)
    localStorage.setItem('ai-gateway-theme', JSON.stringify({ variant: 'dashboard', mode: 'dark' }))
    const { initTheme } = await import('../composables/useTheme')
    initTheme()

    expect(document.documentElement.dataset.theme).toBe('dashboard')
    expect(document.documentElement.dataset.mode).toBe('dark')
  })

  it('setVariant updates dataset and localStorage', async () => {
    setSystemTheme(false)
    localStorage.setItem('ai-gateway-theme', JSON.stringify({ variant: 'apple', mode: 'light' }))
    const { initTheme, setVariant } = await import('../composables/useTheme')
    initTheme()
    setVariant('dashboard')
    await nextTick()

    expect(document.documentElement.dataset.theme).toBe('dashboard')
    const stored = JSON.parse(localStorage.getItem('ai-gateway-theme') || '{}')
    expect(stored.variant).toBe('dashboard')
  })

  it('auto mode follows system scheme changes', async () => {
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

    const { initTheme } = await import('../composables/useTheme')
    initTheme()
    expect(document.documentElement.dataset.mode).toBe('light')

    isDark = true
    listeners.forEach(cb => cb({ matches: true } as MediaQueryListEvent))
    await nextTick()
    expect(document.documentElement.dataset.mode).toBe('dark')
  })
})
