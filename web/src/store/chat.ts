/**
 * Chat Store - Manages conversation state
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { ChatMessage, Conversation, ProviderConfig } from '@/types/chat'
import { createConversation } from '@/types/chat'
import { request } from '@/api/request'
import { API } from '@/constants/api'
import { useModelLabels } from '@/composables/useModelLabels'

const { fetchModelLabels, resetLabels, getModelLabel } = useModelLabels()

/** Default providers configuration (fallback) - must be defined before first use */
const DEFAULT_PROVIDERS: ProviderConfig[] = [
  { label: 'OpenAI', value: 'openai', color: '#10A37F', models: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo'], logo: '/logos/openai.svg' },
  { label: 'Anthropic Claude', value: 'anthropic', color: '#CC785C', models: ['claude-3-5-sonnet-20241022', 'claude-3-5-haiku-20241022'], logo: '/logos/anthropic.svg' },
  { label: 'DeepSeek', value: 'deepseek', color: '#4D6BFE', models: ['deepseek-chat', 'deepseek-reasoner', 'deepseek-coder'], logo: '/logos/deepseek.svg' },
  { label: '阿里云通义千问', value: 'qwen', color: '#FF6A00', models: ['qwen-max', 'qwen-plus', 'qwen-turbo'], logo: '/logos/qwen.svg' },
  { label: '智谱AI', value: 'zhipu', color: '#3657ED', models: ['glm-4-plus', 'glm-4', 'glm-4-flash'], logo: '/logos/zhipu.svg' },
  { label: '月之暗面 (Kimi)', value: 'moonshot', color: '#1A1A1A', models: ['moonshot-v1-8k', 'moonshot-v1-32k'], logo: '/logos/moonshot.svg' },
  { label: '火山方舟 (豆包)', value: 'volcengine', color: '#FF4D4F', models: ['doubao-pro-128k', 'doubao-lite-128k'], logo: '/logos/volcengine.svg' },
]

/** Dynamic providers (reactive) */
export const PROVIDERS = ref<ProviderConfig[]>([...DEFAULT_PROVIDERS])

// Re-export getModelLabel from composable for compatibility
export { getModelLabel }

/** Extract date from model name for sorting (newest first) */
function extractModelDate(modelName: string): number {
  const patterns = [
    /(\d{4})-(\d{2})-(\d{2})/,
    /(\d{8})/,
    /(\d{6})$/,
  ]
  
  for (const pattern of patterns) {
    const match = modelName.match(pattern)
    if (match) {
      if (match.length === 4) {
        return parseInt(`${match[1]}${match[2]}${match[3]}`)
      } else if (match[1] && match[1].length === 8) {
        return parseInt(match[1])
      } else if (match[1] && match[1].length === 6) {
        return parseInt(`20${match[1]}`)
      }
    }
  }
  return 0
}

/** Sort models with newest first */
function sortModelsNewestFirst(models: string[]): string[] {
  return [...models].sort((a, b) => {
    const dateA = extractModelDate(a)
    const dateB = extractModelDate(b)
    // Newer dates first
    if (dateA !== dateB) return dateB - dateA
    // If same/no date, alphabetical
    return a.localeCompare(b)
  })
}

/** Load providers from public API (works without authentication) */
export async function loadProvidersFromPublicAPI(): Promise<boolean> {
  try {
    const response = await fetch(API.V1.CONFIG_PROVIDERS)
    if (!response.ok) {
      return false
    }
    const data = await response.json()
    
    if (!data.success || !data.data?.providers) {
      return false
    }
    
    const providers: ProviderConfig[] = []
    for (const p of data.data.providers as Array<{name: string; enabled: boolean; models: string[]}>) {
      if (p.enabled && p.models && p.models.length > 0) {
        const defaultConfig = DEFAULT_PROVIDERS.find(dp => dp.value === p.name)
        providers.push({
          label: defaultConfig?.label || p.name,
          value: p.name,
          color: defaultConfig?.color || '#909399',
          logo: defaultConfig?.logo || `/logos/${p.name}.svg`,
          models: sortModelsNewestFirst(p.models)
        })
      }
    }
    
    if (providers.length > 0) {
      PROVIDERS.value = providers

      // Public API doesn't provide display_name; reset to identity mapping.
      resetLabels()
      return true
    }
    return false
  } catch (e) {
    console.error('Failed to load providers from public API:', e)
    return false
  }
}

/** Load providers from backend API (requires authentication) */
export async function loadProvidersFromAdminAPI(): Promise<boolean> {
  try {
    // Fetch model labels using composable
    await fetchModelLabels()
    
    const [configsRes, modelsRes, accountsRes] = await Promise.all([
      request.get<{ success: boolean; data: Array<{ value: string; label: string; color: string; base_url: string; is_openai_compatible: boolean }> }>('/admin/providers/configs', { silent: true } as any),
      request.get<{ success: boolean; data: Array<{ model: string; provider: string; enabled: boolean }> }>('/admin/router/models', { silent: true } as any),
      request.get<{ success: boolean; data: Array<{ id: string; provider: string; enabled: boolean }> }>('/admin/accounts', { silent: true } as any)
    ])
    
    const enabledProviders = new Set<string>()
    if ((accountsRes as any).success && (accountsRes as any).data) {
      for (const acc of (accountsRes as any).data) {
        if (acc.enabled) {
          enabledProviders.add(acc.provider)
        }
      }
    }
    
    const providerConfigs: Map<string, { label: string; color: string }> = new Map()
    if ((configsRes as any).success && (configsRes as any).data) {
      for (const p of (configsRes as any).data) {
        providerConfigs.set(p.value, { label: p.label, color: p.color })
      }
    }
    
    const modelsByProvider: Record<string, string[]> = {}
    if ((modelsRes as any).success && (modelsRes as any).data) {
      for (const m of (modelsRes as any).data) {
        if (m.enabled && m.model && enabledProviders.has(m.provider)) {
          if (!modelsByProvider[m.provider]) {
            modelsByProvider[m.provider] = []
          }
          const providerModels = modelsByProvider[m.provider]
          if (providerModels && !providerModels.includes(m.model)) {
            providerModels.push(m.model)
          }
        }
      }
    }
    
    const providers: ProviderConfig[] = []
    
    for (const [providerId, models] of Object.entries(modelsByProvider)) {
      const config = providerConfigs.get(providerId)
      const defaultConfig = DEFAULT_PROVIDERS.find(p => p.value === providerId)
      providers.push({
        label: config?.label || providerId,
        value: providerId,
        color: config?.color || '#909399',
        logo: defaultConfig?.logo || `/logos/${providerId}.svg`,
        models: sortModelsNewestFirst(models)
      })
    }
    
    if (providers.length === 0) {
      PROVIDERS.value = DEFAULT_PROVIDERS.filter(p => enabledProviders.has(p.value))
    } else {
      PROVIDERS.value = providers
    }
    
    if (PROVIDERS.value.length === 0) {
      PROVIDERS.value = [...DEFAULT_PROVIDERS]
    }
    return true
  } catch (e) {
    console.error('Failed to load providers from admin API:', e)
    return false
  }
}

/** Load providers from backend API */
export async function loadProvidersFromAPI(): Promise<void> {
  const token = localStorage.getItem('token')
  
  if (token) {
    const success = await loadProvidersFromAdminAPI()
    if (success) return
  }
  
  const success = await loadProvidersFromPublicAPI()
  if (!success) {
    console.warn('Using default providers as fallback')
    PROVIDERS.value = [...DEFAULT_PROVIDERS]
  }
}

/** Initialize providers from API */
export async function initializeProviders(): Promise<void> {
  await loadProvidersFromAPI()
}

/** Get models for a specific provider */
export function getModelsForProvider(provider: string): string[] {
  const found = PROVIDERS.value.find(p => p.value === provider)
  return found?.models || []
}

/** Get provider config by value */
export function getProviderConfig(provider: string): ProviderConfig | undefined {
  return PROVIDERS.value.find(p => p.value === provider)
}

export const useChatStore = defineStore('chat', () => {
  // State
  const conversations = ref<Conversation[]>([])
  const currentConversationId = ref<string>('')
  const isLoading = ref(false)
  const abortControllers = ref<Map<string, AbortController>>(new Map())
  const streamingConversations = ref<Set<string>>(new Set())

  // Computed
  const currentConversation = computed(() => {
    return conversations.value.find(c => c.id === currentConversationId.value) || null
  })

  const currentMessages = computed(() => {
    return currentConversation.value?.messages || []
  })

  // Actions

  /** Create a new conversation */
  function createNewConversation(provider: string = 'openai', model: string = 'gpt-4o'): Conversation {
    const conversation = createConversation(provider, model)
    conversations.value.unshift(conversation)
    currentConversationId.value = conversation.id
    saveConversations()
    return conversation
  }

  /** Switch to a different conversation */
  function switchConversation(conversationId: string): void {
    const conversation = conversations.value.find(c => c.id === conversationId)
    if (conversation) {
      currentConversationId.value = conversationId
    }
  }

  /** Delete a conversation */
  function deleteConversation(conversationId: string): void {
    const index = conversations.value.findIndex(c => c.id === conversationId)
    if (index !== -1) {
      conversations.value.splice(index, 1)

      // If deleted conversation was current, switch to another or clear
      if (currentConversationId.value === conversationId) {
        if (conversations.value.length > 0 && conversations.value[0]) {
          currentConversationId.value = conversations.value[0].id
        } else {
          currentConversationId.value = ''
        }
      }
      saveConversations()
    }
  }

  /** Add a message to current conversation */
  function addMessage(message: ChatMessage): void {
    if (currentConversation.value) {
      currentConversation.value.messages.push(message)
      currentConversation.value.updatedAt = Date.now()

      // Auto-generate title from first user message
      if (!currentConversation.value.title && message.role === 'user') {
        currentConversation.value.title = message.content.slice(0, 30) + (message.content.length > 30 ? '...' : '')
      }
      saveConversations()
    }
  }

  /** Add a message to a specific conversation */
  function addMessageToConversation(conversationId: string, message: ChatMessage): void {
    const conv = conversations.value.find(c => c.id === conversationId)
    if (conv) {
      conv.messages.push(message)
      conv.updatedAt = Date.now()
      if (!conv.title && message.role === 'user') {
        conv.title = message.content.slice(0, 30) + (message.content.length > 30 ? '...' : '')
      }
      saveConversations()
    }
  }

  /** Get messages for a specific conversation */
  function getConversationMessages(conversationId: string): ChatMessage[] {
    const conv = conversations.value.find(c => c.id === conversationId)
    return conv?.messages || []
  }

  /** Update a specific message */
  function updateMessage(messageId: string, updates: Partial<ChatMessage>, conversationId?: string): void {
    const conv = conversationId 
      ? conversations.value.find(c => c.id === conversationId)
      : currentConversation.value
    if (conv) {
      const message = conv.messages.find(m => m.id === messageId)
      if (message) {
        Object.assign(message, updates)
        conv.updatedAt = Date.now()
        saveConversations()
      }
    }
  }

  /** Set streaming state for a message */
  function setMessageStreaming(messageId: string, isStreaming: boolean): void {
    updateMessage(messageId, { isStreaming })
  }

  /** Append content to a message (for streaming) */
  function appendMessageContent(messageId: string, content: string, conversationId?: string): void {
    const conv = conversationId
      ? conversations.value.find(c => c.id === conversationId)
      : currentConversation.value
    if (conv) {
      const message = conv.messages.find(m => m.id === messageId)
      if (message) {
        message.content += content
        conv.updatedAt = Date.now()
        saveConversations()
      }
    }
  }

  /** Append reasoning content to a message (for deep thinking) */
  function appendMessageReasoning(messageId: string, reasoning: string, conversationId?: string): void {
    const conv = conversationId
      ? conversations.value.find(c => c.id === conversationId)
      : currentConversation.value
    if (conv) {
      const message = conv.messages.find(m => m.id === messageId)
      if (message) {
        message.reasoningContent = (message.reasoningContent || '') + reasoning
        message.reasoning = message.reasoningContent
        conv.updatedAt = Date.now()
        saveConversations()
      }
    }
  }

  /** Set loading state */
  function setLoading(loading: boolean): void {
    isLoading.value = loading
  }

  /** Set abort controller for a conversation */
  function setAbortController(conversationId: string, controller: AbortController | null): void {
    if (controller) {
      abortControllers.value.set(conversationId, controller)
      streamingConversations.value.add(conversationId)
    } else {
      abortControllers.value.delete(conversationId)
      streamingConversations.value.delete(conversationId)
    }
  }

  /** Abort streaming request for a specific conversation */
  function abortRequest(conversationId: string): void {
    const controller = abortControllers.value.get(conversationId)
    if (controller) {
      controller.abort()
      abortControllers.value.delete(conversationId)
      streamingConversations.value.delete(conversationId)
    }
  }

  /** Check if a conversation is streaming */
  function isConversationStreaming(conversationId: string): boolean {
    return streamingConversations.value.has(conversationId)
  }

  /** Update current conversation model/provider */
  function updateCurrentModel(provider: string, model: string): void {
    if (currentConversation.value) {
      currentConversation.value.provider = provider
      currentConversation.value.model = model
      currentConversation.value.updatedAt = Date.now()
      saveConversations()
    }
  }

  /** Save conversations to localStorage */
  function saveConversations(): void {
    try {
      localStorage.setItem('chat_conversations', JSON.stringify(conversations.value))
      localStorage.setItem('chat_current_id', currentConversationId.value)
    } catch (e) {
      console.error('Failed to save conversations:', e)
    }
  }

  /** Load conversations from localStorage */
  function loadConversations(): void {
    try {
      const saved = localStorage.getItem('chat_conversations')
      const savedCurrentId = localStorage.getItem('chat_current_id')

      if (saved) {
        conversations.value = JSON.parse(saved)
      }

      if (savedCurrentId && conversations.value.find(c => c.id === savedCurrentId)) {
        currentConversationId.value = savedCurrentId
      } else if (conversations.value.length > 0 && conversations.value[0]) {
        currentConversationId.value = conversations.value[0].id
      }
    } catch (e) {
      console.error('Failed to load conversations:', e)
      conversations.value = []
      currentConversationId.value = ''
    }
  }

  /** Clear all conversations */
  function clearAllConversations(): void {
    conversations.value = []
    currentConversationId.value = ''
    saveConversations()
  }

  // Initialize on store creation
  loadConversations()

  return {
    // State
    conversations,
    currentConversationId,
    isLoading,

    // Computed
    currentConversation,
    currentMessages,

    // Actions
    createNewConversation,
    switchConversation,
    deleteConversation,
    addMessage,
    addMessageToConversation,
    getConversationMessages,
    updateMessage,
    setMessageStreaming,
    appendMessageContent,
    appendMessageReasoning,
    setLoading,
    setAbortController,
    abortRequest,
    isConversationStreaming,
    updateCurrentModel,
    saveConversations,
    loadConversations,
    clearAllConversations
  }
})
