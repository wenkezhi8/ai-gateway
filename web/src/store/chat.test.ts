import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { PROVIDERS, useChatStore } from './chat'

const getModelLabelsForProviderMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/routing-domain', () => ({
  getModelRegistry: vi.fn()
}))

vi.mock('@/api/chat-domain', () => ({
  getAdminAccounts: vi.fn(),
  getAdminProviderConfigs: vi.fn(),
  getPublicProvidersConfig: vi.fn()
}))

vi.mock('@/composables/useModelLabels', () => ({
  useModelLabels: () => ({
    fetchModelLabels: vi.fn(),
    resetLabels: vi.fn(),
    getModelLabel: (_provider: string, model: string) => model,
    getModelLabelsForProvider: getModelLabelsForProviderMock
  })
}))

function buildConversation(id: string, provider: string, model: string) {
  return {
    id,
    title: '',
    messages: [],
    provider,
    model,
    createdAt: Date.now(),
    updatedAt: Date.now()
  }
}

function createMemoryStorage() {
  const data = new Map<string, string>()
  return {
    getItem(key: string) {
      return data.has(key) ? data.get(key)! : null
    },
    setItem(key: string, value: string) {
      data.set(key, String(value))
    },
    removeItem(key: string) {
      data.delete(key)
    },
    clear() {
      data.clear()
    }
  }
}

describe('chat store', () => {
  beforeEach(() => {
    const localStorageMock = createMemoryStorage()
    const windowMock = {
      location: {
        pathname: '/chat'
      },
      history: {
        replaceState: (_state: unknown, _title: string, url: string) => {
          const nextPath = typeof url === 'string'
            ? new URL(url, 'http://localhost').pathname
            : '/chat'
          windowMock.location.pathname = nextPath
        }
      }
    }

    vi.stubGlobal('localStorage', localStorageMock)
    vi.stubGlobal('window', windowMock)

    setActivePinia(createPinia())
    localStorage.clear()
    PROVIDERS.value = []
    getModelLabelsForProviderMock.mockReset()
    window.history.replaceState({}, '', '/chat')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('should isolate localStorage keys for public chat route', () => {
    const privateConversation = buildConversation('private-1', 'openai', 'gpt-4o')
    const publicConversation = buildConversation('public-1', 'openai', 'gpt-5.3-codex-spark')

    localStorage.setItem('chat_conversations', JSON.stringify([privateConversation]))
    localStorage.setItem('chat_current_id', 'private-1')
    localStorage.setItem('chat_conversations_public', JSON.stringify([publicConversation]))
    localStorage.setItem('chat_current_id_public', 'public-1')

    window.history.replaceState({}, '', '/p/chat')
    const store = useChatStore()

    expect(store.currentConversationId).toBe('public-1')
    expect(store.conversations).toHaveLength(1)
    expect(store.conversations[0]?.id).toBe('public-1')

    store.createNewConversation('openai', 'gpt-5.2-codex')
    const storedPublic = localStorage.getItem('chat_conversations_public') || ''
    const storedPrivate = localStorage.getItem('chat_conversations') || ''

    expect(storedPublic).toContain('public-1')
    expect(storedPrivate).toContain('private-1')
  })

  it('should normalize legacy display name model to canonical model id', () => {
    const legacyConversation = buildConversation('legacy-1', 'openai', 'GPT-5.3 Codex Spark')
    localStorage.setItem('chat_conversations_public', JSON.stringify([legacyConversation]))
    localStorage.setItem('chat_current_id_public', 'legacy-1')
    window.history.replaceState({}, '', '/p/chat')

    PROVIDERS.value = [
      {
        label: 'OpenAI',
        value: 'openai',
        color: '#10a37f',
        models: ['gpt-5.3-codex-spark', 'gpt-5.2-codex']
      }
    ]
    getModelLabelsForProviderMock.mockReturnValue({
      'gpt-5.3-codex-spark': 'GPT-5.3 Codex Spark',
      'gpt-5.2-codex': 'GPT-5.2 Codex'
    })

    const store = useChatStore()
    const changedCount = store.normalizeConversationModels()

    expect(changedCount).toBe(1)
    expect(store.currentConversation?.model).toBe('gpt-5.3-codex-spark')
  })
})
