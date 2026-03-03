/**
 * AI Chat Types
 */

/** Message role type */
export type MessageRole = 'user' | 'assistant' | 'system'

/** Single chat message */
export interface ChatMessage {
  id: string
  role: MessageRole
  content: string
  timestamp: number
  isStreaming?: boolean
  error?: string
  stats?: MessageStats
  images?: string[]
  files?: string[]
  reasoningContent?: string // DeepSeek R1 深度思考内容
  reasoning?: string // 兼容字段
}

/** Message statistics */
export interface MessageStats {
  firstTokenTime?: number
  totalTime?: number
  outputTokensPerSecond?: number
  totalTokens?: number
  promptTokens?: number
  completionTokens?: number
  cacheHit?: boolean
  cacheLayer?: string
  requestMode?: 'stream' | 'non_stream'
  speedBasis?: 'post_first_token' | 'total_time' | 'fallback_total_time'
  reasoningEffortDowngraded?: boolean
}

/** Completion metadata returned by API layer */
export interface CompletionMeta {
  totalTokens?: number
  promptTokens?: number
  completionTokens?: number
  cacheHit?: boolean
  cacheLayer?: string
  requestMode: 'stream' | 'non_stream'
  reasoningEffortDowngraded?: boolean
}

export type ReasoningEffort = 'low' | 'medium' | 'high' | 'xhigh'

/** Conversation session */
export interface Conversation {
  id: string
  title: string
  messages: ChatMessage[]
  model: string
  provider: string
  createdAt: number
  updatedAt: number
}

/** Chat completion API request parameters */
export interface ChatCompletionParams {
  model: string
  messages: Array<{
    role: MessageRole
    content: string | Array<{ type: string; text?: string; image_url?: { url: string } }>
  }>
  provider?: string
  temperature?: number
  max_tokens?: number
  top_p?: number
  frequency_penalty?: number
  presence_penalty?: number
  stream?: boolean
  deepThink?: boolean
  reasoning_effort?: ReasoningEffort
}

/** SSE stream chunk data */
export interface StreamChunk {
  id?: string
  object?: string
  created?: number
  model?: string
  choices?: Array<{
    index: number
    delta: {
      role?: string
      content?: string
      reasoning?: string
      reasoning_content?: string
    }
    finish_reason?: string | null
  }>
  usage?: {
    prompt_tokens?: number
    completion_tokens?: number
    total_tokens?: number
  }
  gateway_meta?: {
    reasoning_effort_downgraded?: boolean
  }
  error?: {
    message: string
    type: string
    code: string
  }
}

/** Provider configuration */
export interface ProviderConfig {
  label: string
  value: string
  color: string
  models: string[]
  logo?: string
}

/** Generate unique ID */
export function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`
}

/** Create a new message */
export function createMessage(role: MessageRole, content: string): ChatMessage {
  return {
    id: generateId(),
    role,
    content,
    timestamp: Date.now()
  }
}

/** Create a new conversation */
export function createConversation(provider: string, model: string): Conversation {
  return {
    id: generateId(),
    title: '',
    messages: [],
    model,
    provider,
    createdAt: Date.now(),
    updatedAt: Date.now()
  }
}
