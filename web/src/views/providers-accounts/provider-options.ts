import type { ProviderType, PublicProviderInfo } from '@/api/provider'

export type ProviderCategory = 'international' | 'chinese' | 'local' | 'custom'

export interface ProviderOption {
  label: string
  value: string
  category: ProviderCategory
  color: string
  logo: string
  default_endpoint: string
  coding_endpoint?: string
}

type AccountLike = {
  provider?: string
  provider_type?: string
  base_url?: string
}

const PROVIDER_ALIAS_MAP: Record<string, string> = {
  claude: 'anthropic',
  kimi: 'moonshot'
}

const PROVIDER_META_DEFAULTS: Record<string, Omit<ProviderOption, 'value'>> = {
  openai: {
    label: 'OpenAI',
    category: 'international',
    color: '#10A37F',
    logo: '/logos/openai.svg',
    default_endpoint: 'https://api.openai.com/v1',
    coding_endpoint: 'https://api.openai.com/v1'
  },
  anthropic: {
    label: 'Anthropic Claude',
    category: 'international',
    color: '#CC785C',
    logo: '/logos/anthropic.svg',
    default_endpoint: 'https://api.anthropic.com/v1',
    coding_endpoint: 'https://api.anthropic.com/v1'
  },
  'azure-openai': {
    label: 'Azure OpenAI',
    category: 'international',
    color: '#0078D4',
    logo: '/logos/azure.svg',
    default_endpoint: 'https://your-resource.openai.azure.com',
    coding_endpoint: 'https://your-resource.openai.azure.com'
  },
  google: {
    label: 'Google Gemini',
    category: 'international',
    color: '#4285F4',
    logo: '/logos/google.svg',
    default_endpoint: 'https://generativelanguage.googleapis.com/v1beta',
    coding_endpoint: 'https://generativelanguage.googleapis.com/v1beta/openai'
  },
  deepseek: {
    label: 'DeepSeek',
    category: 'chinese',
    color: '#4D6BFE',
    logo: '/logos/deepseek.svg',
    default_endpoint: 'https://api.deepseek.com/v1',
    coding_endpoint: 'https://api.deepseek.com/v1'
  },
  qwen: {
    label: '阿里云通义千问',
    category: 'chinese',
    color: '#FF6A00',
    logo: '/logos/qwen.svg',
    default_endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    coding_endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1'
  },
  zhipu: {
    label: '智谱AI',
    category: 'chinese',
    color: '#3657ED',
    logo: '/logos/zhipu.svg',
    default_endpoint: 'https://open.bigmodel.cn/api/paas/v4',
    coding_endpoint: 'https://open.bigmodel.cn/api/paas/v4'
  },
  moonshot: {
    label: '月之暗面 (Kimi)',
    category: 'chinese',
    color: '#1A1A1A',
    logo: '/logos/moonshot.svg',
    default_endpoint: 'https://api.moonshot.cn/v1',
    coding_endpoint: 'https://api.moonshot.cn/v1'
  },
  minimax: {
    label: 'MiniMax',
    category: 'chinese',
    color: '#615CED',
    logo: '/logos/minimax.svg',
    default_endpoint: 'https://api.minimax.chat/v1',
    coding_endpoint: 'https://api.minimax.chat/v1'
  },
  baichuan: {
    label: '百川智能',
    category: 'chinese',
    color: '#0066FF',
    logo: '/logos/baichuan.svg',
    default_endpoint: 'https://api.baichuan-ai.com/v1',
    coding_endpoint: 'https://api.baichuan-ai.com/v1'
  },
  volcengine: {
    label: '火山方舟 (豆包)',
    category: 'chinese',
    color: '#FF4D4F',
    logo: '/logos/volcengine.svg',
    default_endpoint: 'https://ark.cn-beijing.volces.com/api/v3',
    coding_endpoint: 'https://ark.cn-beijing.volces.com/api/v3'
  },
  ernie: {
    label: '百度文心一言',
    category: 'chinese',
    color: '#2932E1',
    logo: '/logos/ernie.svg',
    default_endpoint: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat',
    coding_endpoint: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat'
  },
  hunyuan: {
    label: '腾讯混元',
    category: 'chinese',
    color: '#00A3FF',
    logo: '/logos/hunyuan.svg',
    default_endpoint: 'https://api.hunyuan.cloud.tencent.com/v1',
    coding_endpoint: 'https://api.hunyuan.cloud.tencent.com/v1'
  },
  spark: {
    label: '讯飞星火',
    category: 'chinese',
    color: '#E60012',
    logo: '/logos/spark.svg',
    default_endpoint: 'https://spark-api-open.xf-yun.com/v1',
    coding_endpoint: 'https://spark-api-open.xf-yun.com/v1'
  },
  yi: {
    label: '零一万物',
    category: 'chinese',
    color: '#00D4AA',
    logo: '/logos/yi.svg',
    default_endpoint: 'https://api.lingyiwanwu.com/v1',
    coding_endpoint: 'https://api.lingyiwanwu.com/v1'
  },
  ollama: {
    label: 'Ollama',
    category: 'local',
    color: '#10B981',
    logo: '/logos/ollama.svg',
    default_endpoint: 'http://localhost:11434/v1',
    coding_endpoint: 'http://localhost:11434/v1'
  },
  lmstudio: {
    label: 'LM Studio',
    category: 'local',
    color: '#3B82F6',
    logo: '/logos/lmstudio.svg',
    default_endpoint: 'http://localhost:1234/v1',
    coding_endpoint: 'http://localhost:1234/v1'
  },
  local: {
    label: '本地模型',
    category: 'local',
    color: '#6B7280',
    logo: '/logos/local.svg',
    default_endpoint: 'http://localhost:11434/v1',
    coding_endpoint: 'http://localhost:11434/v1'
  }
}

const CATEGORY_ORDER: Record<ProviderCategory, number> = {
  international: 0,
  chinese: 1,
  local: 2,
  custom: 3
}

const FALLBACK_PALETTE = ['#5B8FF9', '#5AD8A6', '#5D7092', '#F6BD16', '#E8684A', '#6DC8EC', '#9270CA', '#FF9D4D']

function fallbackColorByID(id: string): string {
  let hash = 0
  for (let i = 0; i < id.length; i += 1) {
    hash = (hash << 5) - hash + id.charCodeAt(i)
    hash |= 0
  }
  return FALLBACK_PALETTE[Math.abs(hash) % FALLBACK_PALETTE.length] || '#5B8FF9'
}

export function normalizeProviderID(input: string): string {
  const normalized = String(input || '').trim().toLowerCase()
  if (!normalized) return ''
  return PROVIDER_ALIAS_MAP[normalized] || normalized
}

function inferCategory(providerID: string): ProviderCategory {
  const normalized = normalizeProviderID(providerID)
  return PROVIDER_META_DEFAULTS[normalized]?.category || 'custom'
}

function defaultOption(providerID: string): ProviderOption {
  const normalized = normalizeProviderID(providerID)
  const known = PROVIDER_META_DEFAULTS[normalized]
  if (known) {
    return {
      value: normalized,
      ...known
    }
  }

  return {
    value: normalized,
    label: normalized,
    category: inferCategory(normalized),
    color: fallbackColorByID(normalized),
    logo: '',
    default_endpoint: '',
    coding_endpoint: ''
  }
}

export function inferProviderFromAccountBaseURL(baseURL?: string): string {
  const normalizedURL = String(baseURL || '').trim().toLowerCase()
  if (!normalizedURL) return ''

  if (normalizedURL.includes('deepseek.com')) return 'deepseek'
  if (normalizedURL.includes('openai.com')) return 'openai'
  if (normalizedURL.includes('anthropic.com')) return 'anthropic'
  if (normalizedURL.includes('volces.com') || normalizedURL.includes('volcengine')) return 'volcengine'
  if (normalizedURL.includes('dashscope.aliyuncs.com') || normalizedURL.includes('aliyun')) return 'qwen'
  if (normalizedURL.includes('zhipuai.cn') || normalizedURL.includes('bigmodel.cn')) return 'zhipu'
  if (normalizedURL.includes('moonshot.cn') || normalizedURL.includes('kimi.ai')) return 'moonshot'
  if (normalizedURL.includes('minimax')) return 'minimax'
  if (normalizedURL.includes('baichuan')) return 'baichuan'
  if (normalizedURL.includes('googleapis.com')) return 'google'
  if (normalizedURL.includes('localhost:11434') || normalizedURL.includes('127.0.0.1:11434') || normalizedURL.includes('ollama')) return 'ollama'
  if (normalizedURL.includes('localhost:1234') || normalizedURL.includes('127.0.0.1:1234') || normalizedURL.includes('lmstudio')) return 'lmstudio'
  return ''
}

function upsertProviderOption(map: Map<string, ProviderOption>, id: string, patch: Partial<ProviderOption> = {}): void {
  const normalized = normalizeProviderID(id)
  if (!normalized) return

  const current = map.get(normalized) || defaultOption(normalized)
  const merged: ProviderOption = {
    ...current,
    ...patch,
    value: normalized,
    label: String(patch.label || current.label || normalized),
    category: (patch.category || current.category || inferCategory(normalized)) as ProviderCategory,
    color: String(patch.color || current.color || fallbackColorByID(normalized)),
    logo: String(patch.logo ?? current.logo ?? ''),
    default_endpoint: String(patch.default_endpoint ?? current.default_endpoint ?? ''),
    coding_endpoint: String(patch.coding_endpoint ?? current.coding_endpoint ?? current.default_endpoint ?? '')
  }

  if (!merged.default_endpoint && PROVIDER_META_DEFAULTS[normalized]?.default_endpoint) {
    merged.default_endpoint = PROVIDER_META_DEFAULTS[normalized].default_endpoint
  }
  if (!merged.coding_endpoint) {
    merged.coding_endpoint = merged.default_endpoint
  }
  if (merged.category !== 'international' && merged.category !== 'chinese' && merged.category !== 'local' && merged.category !== 'custom') {
    merged.category = inferCategory(normalized)
  }

  map.set(normalized, merged)
}

export function buildProviderOptions(input: {
  types?: ProviderType[]
  publicProviders?: PublicProviderInfo[]
  accounts?: AccountLike[]
}): ProviderOption[] {
  const map = new Map<string, ProviderOption>()

  const types = Array.isArray(input.types) ? input.types : []
  for (const item of types) {
    upsertProviderOption(map, item.id, {
      label: item.label,
      category: item.category,
      color: item.color,
      logo: item.logo,
      default_endpoint: item.default_endpoint,
      coding_endpoint: item.coding_endpoint || item.default_endpoint
    })
  }

  const publicProviders = Array.isArray(input.publicProviders) ? input.publicProviders : []
  for (const item of publicProviders) {
    upsertProviderOption(map, item.id, {
      label: item.label || item.id,
      color: item.color,
      logo: item.logo
    })
  }

  const accounts = Array.isArray(input.accounts) ? input.accounts : []
  for (const account of accounts) {
    const inferred = inferProviderFromAccountBaseURL(account.base_url)
    const providerID = inferred || normalizeProviderID(account.provider || account.provider_type || '')
    upsertProviderOption(map, providerID)
  }

  return Array.from(map.values()).sort((left, right) => {
    const categoryDelta = CATEGORY_ORDER[left.category] - CATEGORY_ORDER[right.category]
    if (categoryDelta !== 0) return categoryDelta

    const leftLabel = left.label.toLowerCase()
    const rightLabel = right.label.toLowerCase()
    if (leftLabel !== rightLabel) return leftLabel.localeCompare(rightLabel)
    return left.value.localeCompare(right.value)
  })
}
