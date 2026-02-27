export type CacheTypeId =
  | 'response'
  | 'request'
  | 'route'
  | 'context'
  | 'usage'
  | 'semantic'

export interface CacheTypeMeta {
  id: CacheTypeId
  name: string
  alias: string
  description: string
  prefix: string
  tone: string
  icon: string
}

export interface CacheTypeState {
  id: CacheTypeId
  enabled?: boolean
  hitRate?: number
  entries?: number
  size?: string
}

export interface CacheTypeCard extends CacheTypeMeta {
  enabled: boolean
  hitRate: number
  entries: number
  size: string
}

export const CACHE_TYPE_META: Record<CacheTypeId, CacheTypeMeta> = {
  response: {
    id: 'response',
    name: '内容缓存',
    alias: 'Response Cache',
    description: '缓存最终模型响应体，命中后直接返回结果。',
    prefix: 'ai-gateway:ai-response:*',
    tone: 'ocean',
    icon: 'MagicStick'
  },
  request: {
    id: 'request',
    name: '请求缓存',
    alias: 'Request Cache',
    description: '缓存请求参数 + 响应 + token 用量，用于去重与加速。',
    prefix: 'ai-gateway:req:*',
    tone: 'sunset',
    icon: 'Connection'
  },
  route: {
    id: 'route',
    name: '路由缓存',
    alias: 'Route Cache',
    description: '缓存模型/服务商路由决策，降低路由判断成本。',
    prefix: 'ai-gateway:route:*',
    tone: 'violet',
    icon: 'Share'
  },
  context: {
    id: 'context',
    name: '上下文缓存',
    alias: 'Context Cache',
    description: '缓存多轮会话消息与摘要，复用历史上下文。',
    prefix: 'ai-gateway:session:*',
    tone: 'forest',
    icon: 'ChatDotRound'
  },
  usage: {
    id: 'usage',
    name: 'Usage 缓存',
    alias: 'Usage Cache',
    description: '缓存面板聚合统计，降低实时计算压力。',
    prefix: 'ai-gateway:usage:*',
    tone: 'ember',
    icon: 'TrendCharts'
  },
  semantic: {
    id: 'semantic',
    name: '语义缓存',
    alias: 'Semantic Cache',
    description: '基于向量相似度匹配，相似请求可复用缓存结果。',
    prefix: '语义索引',
    tone: 'neon',
    icon: 'Compass'
  }
}

export const CACHE_TYPE_ORDER: CacheTypeId[] = [
  'response',
  'request',
  'route',
  'context',
  'usage',
  'semantic'
]

export const listCacheTypeMeta = (): CacheTypeMeta[] =>
  CACHE_TYPE_ORDER.map(id => CACHE_TYPE_META[id])

export const buildCacheTypeCards = (states: CacheTypeState[] = []): CacheTypeCard[] => {
  const byId = new Map(states.map(state => [state.id, state]))
  return CACHE_TYPE_ORDER.map(id => {
    const meta = CACHE_TYPE_META[id]
    const state = byId.get(id)
    return {
      ...meta,
      enabled: state?.enabled ?? true,
      hitRate: state?.hitRate ?? 0,
      entries: state?.entries ?? 0,
      size: state?.size ?? '0 MB'
    }
  })
}

export const getCacheTypeMeta = (id: string): CacheTypeMeta | undefined =>
  CACHE_TYPE_META[id as CacheTypeId]
