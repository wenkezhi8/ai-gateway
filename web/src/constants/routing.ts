export type TaskModelMappingItem = {
  enabled: boolean
  model: string
}

export type TaskTypeItem = {
  type: string
  name: string
  count: number
  percentage: number
  color: string
}

export const ROUTING_OLLAMA_DEFAULT_BASE_URL = 'http://127.0.0.1:11434'
export const ROUTING_OLLAMA_DEFAULT_MODEL = 'qwen2.5:0.5b-instruct'
export const ROUTING_OLLAMA_DEFAULT_EMBEDDING_MODEL = 'nomic-embed-text'
export const ROUTING_OLLAMA_DEFAULT_EMBEDDING_DIMENSION = 1024
export const ROUTING_OLLAMA_DEFAULT_EMBEDDING_TIMEOUT_MS = 3000
export const ROUTING_OLLAMA_DEFAULT_EMBEDDING_ENDPOINT_MODE = 'auto'

export const DEFAULT_CLASSIFIER_CONFIG = {
  enabled: true,
  shadow_mode: false,
  provider: 'ollama',
  base_url: ROUTING_OLLAMA_DEFAULT_BASE_URL,
  active_model: ROUTING_OLLAMA_DEFAULT_MODEL,
  candidate_models: [ROUTING_OLLAMA_DEFAULT_MODEL],
  timeout_ms: 5000,
  confidence_threshold: 0.65,
  fail_open: true,
  max_input_chars: 4000,
  control: {
    enable: false,
    shadow_only: true,
    normalized_query_read_enable: false,
    cache_write_gate_enable: false,
    risk_tag_enable: false,
    risk_block_enable: false,
    tool_gate_enable: false,
    model_fit_enable: false,
    parameter_hint_enable: false
  }
}

export const DEFAULT_TASK_MODEL_MAPPING: Record<string, TaskModelMappingItem> = {
  code: { enabled: false, model: '' },
  chat: { enabled: false, model: '' },
  reasoning: { enabled: false, model: '' },
  math: { enabled: false, model: '' },
  fact: { enabled: false, model: '' },
  creative: { enabled: false, model: '' },
  translate: { enabled: false, model: '' },
  other: { enabled: false, model: '' }
}

export const DEFAULT_TASK_TYPES: TaskTypeItem[] = [
  { type: 'code', name: '代码生成', count: 0, percentage: 0, color: '#007AFF' },
  { type: 'chat', name: '日常对话', count: 0, percentage: 0, color: '#34C759' },
  { type: 'reasoning', name: '逻辑推理', count: 0, percentage: 0, color: '#FF9500' },
  { type: 'math', name: '数学计算', count: 0, percentage: 0, color: '#FF3B30' },
  { type: 'fact', name: '事实查询', count: 0, percentage: 0, color: '#34C759' },
  { type: 'creative', name: '创意写作', count: 0, percentage: 0, color: '#AF52DE' },
  { type: 'translate', name: '翻译', count: 0, percentage: 0, color: '#5856D6' },
  { type: 'other', name: '其他', count: 0, percentage: 0, color: '#8E8E93' }
]

export function createDefaultTaskModelMapping(): Record<string, TaskModelMappingItem> {
  return Object.fromEntries(
    Object.entries(DEFAULT_TASK_MODEL_MAPPING).map(([key, value]) => [
      key,
      { ...value }
    ])
  )
}

export function createDefaultTaskTypes(): TaskTypeItem[] {
  return DEFAULT_TASK_TYPES.map(item => ({ ...item }))
}
