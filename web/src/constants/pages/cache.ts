export const CACHE_WARMUP_DEFAULTS = {
  model: 'gpt-4o',
  provider: 'openai'
} as const

export const CACHE_DEFAULT_TASK_TTL = {
  fact: 24,
  code: 168,
  math: 720,
  chat: 1,
  creative: 0,
  reasoning: 168,
  translate: 72,
  long_text: 360,
  unknown: 24
} as const

export type CacheTaskTTLItem = {
  key: keyof typeof CACHE_DEFAULT_TASK_TTL
  name: string
  description: string
  ttl: number
}

export type CacheTaskTypeOption = {
  label: string
  value: keyof typeof CACHE_DEFAULT_TASK_TTL
}

export const CACHE_TASK_TYPE_OPTIONS: CacheTaskTypeOption[] = [
  { label: '事实查询', value: 'fact' },
  { label: '代码生成', value: 'code' },
  { label: '数学计算', value: 'math' },
  { label: '日常对话', value: 'chat' },
  { label: '创意写作', value: 'creative' },
  { label: '逻辑推理', value: 'reasoning' },
  { label: '翻译', value: 'translate' },
  { label: '长文本处理', value: 'long_text' },
  { label: '其他', value: 'unknown' }
]

export const CACHE_TASK_TTL_ITEMS: CacheTaskTTLItem[] = [
  { key: 'fact', name: '事实查询', description: '公共事实、政策、常识等，可能定期更新', ttl: CACHE_DEFAULT_TASK_TTL.fact },
  { key: 'code', name: '代码生成', description: '通用代码片段，更新频率低', ttl: CACHE_DEFAULT_TASK_TTL.code },
  { key: 'math', name: '数学计算', description: '数学题结果，几乎不会变化', ttl: CACHE_DEFAULT_TASK_TTL.math },
  { key: 'chat', name: '日常对话', description: '个性化对话，上下文相关性强', ttl: CACHE_DEFAULT_TASK_TTL.chat },
  { key: 'creative', name: '创意写作', description: '个性化创意内容，默认不缓存', ttl: CACHE_DEFAULT_TASK_TTL.creative },
  { key: 'reasoning', name: '逻辑推理', description: '推理结果，稳定性高', ttl: CACHE_DEFAULT_TASK_TTL.reasoning },
  { key: 'translate', name: '翻译', description: '标准翻译结果，仅术语更新时变化', ttl: CACHE_DEFAULT_TASK_TTL.translate },
  { key: 'long_text', name: '长文本处理', description: '文档摘要、PDF解析等，同一文本结果固定', ttl: CACHE_DEFAULT_TASK_TTL.long_text },
  { key: 'unknown', name: '其他类型', description: '未分类任务', ttl: CACHE_DEFAULT_TASK_TTL.unknown }
]

export const CACHE_RULE_MODEL_OPTIONS = [
  {
    label: 'OpenAI',
    options: [
      { label: 'gpt-4o', value: 'gpt-4o' },
      { label: 'gpt-4-turbo', value: 'gpt-4-turbo' },
      { label: 'gpt-3.5-turbo', value: 'gpt-3.5-turbo' }
    ]
  },
  {
    label: 'Anthropic',
    options: [
      { label: 'claude-3-5-sonnet', value: 'claude-3-5-sonnet' },
      { label: 'claude-3-opus', value: 'claude-3-opus' }
    ]
  },
  {
    label: '阿里云通义千问',
    options: [
      { label: 'qwen-max', value: 'qwen-max' },
      { label: 'qwen-plus', value: 'qwen-plus' },
      { label: 'qwen-turbo', value: 'qwen-turbo' }
    ]
  },
  {
    label: '百度文心一言',
    options: [
      { label: 'ernie-4.0', value: 'ernie-4.0' },
      { label: 'ernie-3.5', value: 'ernie-3.5' }
    ]
  },
  {
    label: '智谱AI',
    options: [
      { label: 'glm-4-plus', value: 'glm-4-plus' },
      { label: 'glm-4-flash', value: 'glm-4-flash' }
    ]
  },
  {
    label: '月之暗面',
    options: [
      { label: 'moonshot-v1-8k', value: 'moonshot-v1-8k' },
      { label: 'moonshot-v1-128k', value: 'moonshot-v1-128k' }
    ]
  },
  {
    label: 'DeepSeek',
    options: [
      { label: 'deepseek-chat', value: 'deepseek-chat' },
      { label: 'deepseek-reasoner', value: 'deepseek-reasoner' }
    ]
  }
] as const
