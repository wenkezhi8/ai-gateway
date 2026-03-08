export const TRACE_ANSWER_SOURCES = [
  'exact_raw',
  'exact_prompt',
  'semantic',
  'v2',
  'provider_chat'
] as const

export type TraceAnswerSource = (typeof TRACE_ANSWER_SOURCES)[number]

export const TRACE_ANSWER_SOURCE_LABELS: Record<TraceAnswerSource, string> = {
  exact_raw: '原始缓存',
  exact_prompt: '精确缓存',
  semantic: '语义缓存',
  v2: '向量缓存',
  provider_chat: '上游回源'
}

export const TRACE_ANSWER_SOURCE_FALLBACK: TraceAnswerSource = 'provider_chat'

export const CACHE_REQUEST_SOURCES = ['all', ...TRACE_ANSWER_SOURCES] as const

export type CacheRequestSource = (typeof CACHE_REQUEST_SOURCES)[number]
