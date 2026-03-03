import type { CacheModelOptionGroup, CacheTaskTypeConfig } from '@/api/cache-domain'
import type { PublicProviderInfo } from '@/api/provider'

const FALLBACK_TASK_TYPES: CacheTaskTypeConfig[] = [
  { key: 'fact', label: '事实查询', description: '稳定事实问答与知识检索', default_ttl: 24, ttl_unit: 'hours' },
  { key: 'code', label: '代码生成', description: '代码生成、修复与重构', default_ttl: 168, ttl_unit: 'hours' },
  { key: 'math', label: '数学计算', description: '公式推导与数值计算', default_ttl: 720, ttl_unit: 'hours' },
  { key: 'chat', label: '通用对话', description: '开放式聊天与闲聊', default_ttl: 1, ttl_unit: 'hours' },
  { key: 'creative', label: '创意写作', description: '文案创作与灵感扩展', default_ttl: 0, ttl_unit: 'hours' },
  { key: 'reasoning', label: '逻辑推理', description: '多步推理与复杂分析', default_ttl: 168, ttl_unit: 'hours' },
  { key: 'translate', label: '翻译改写', description: '多语言翻译与文本改写', default_ttl: 72, ttl_unit: 'hours' },
  { key: 'long_text', label: '长文本', description: '长文总结、抽取与改写', default_ttl: 360, ttl_unit: 'hours' },
  { key: 'unknown', label: '其他任务', description: '未分类或混合场景', default_ttl: 24, ttl_unit: 'hours' }
]

export interface CacheTaskMetaLoadResult {
  mode: 'primary' | 'fallback' | 'error'
  taskTypes: CacheTaskTypeConfig[]
  modelOptions: CacheModelOptionGroup[]
  warningMessage: string
  errorMessage: string
}

interface CacheTaskMetaLoadResultInput {
  isTaskTTL404: boolean
  taskTypes: CacheTaskTypeConfig[]
  modelOptions: CacheModelOptionGroup[]
  publicProviders: PublicProviderInfo[]
  ttlDefaults: Record<string, number>
}

export function buildFallbackTaskMeta(publicProviders: PublicProviderInfo[] = []): {
  task_types: CacheTaskTypeConfig[]
  model_options: CacheModelOptionGroup[]
} {
  const modelOptions: CacheModelOptionGroup[] = publicProviders
    .map((provider) => {
      const model = provider.default_model ? String(provider.default_model).trim() : ''
      return {
        provider_id: provider.id,
        provider_label: provider.label || provider.id,
        models: model ? [model] : []
      }
    })
    .filter((group) => group.provider_id)

  return {
    task_types: FALLBACK_TASK_TYPES.map((item) => ({ ...item })),
    model_options: modelOptions
  }
}

export function mergeTtlOverrides(
  taskTypes: CacheTaskTypeConfig[],
  ttlDefaults: Record<string, number>
): CacheTaskTypeConfig[] {
  return taskTypes.map((item) => {
    const override = ttlDefaults[item.key]
    if (typeof override !== 'number' || !Number.isFinite(override) || override < 0) {
      return item
    }
    return {
      ...item,
      default_ttl: override
    }
  })
}

export function resolveCacheTaskMetaLoadResult(
  input: CacheTaskMetaLoadResultInput
): CacheTaskMetaLoadResult {
  if (input.taskTypes.length > 0) {
    return {
      mode: 'primary',
      taskTypes: mergeTtlOverrides(input.taskTypes, input.ttlDefaults),
      modelOptions: input.modelOptions,
      warningMessage: '',
      errorMessage: ''
    }
  }

  if (input.isTaskTTL404) {
    const fallback = buildFallbackTaskMeta(input.publicProviders)
    return {
      mode: 'fallback',
      taskTypes: mergeTtlOverrides(fallback.task_types, input.ttlDefaults),
      modelOptions: fallback.model_options,
      warningMessage: '已进入兼容模式：后端未提供 /admin/cache/task-ttl，使用本地默认元数据。',
      errorMessage: ''
    }
  }

  return {
    mode: 'error',
    taskTypes: [],
    modelOptions: [],
    warningMessage: '',
    errorMessage: '请检查 /admin/cache/task-ttl 接口状态'
  }
}
