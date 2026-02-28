import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { ElMessage } from 'element-plus'
import { getRouterModels } from '@/api/routing-domain'
import { MODEL_FORM_DEFAULTS } from '@/constants/store/models'

export interface Model {
  id: string
  name: string
  provider: string
  type: 'chat' | 'completion' | 'embedding' | 'image'
  enabled: boolean
  maxTokens: number
  inputPrice: number
  outputPrice: number
  description?: string
  createdAt?: string
  updatedAt?: string
}

export interface ModelConfig {
  modelId: string
  temperature?: number
  maxTokens?: number
  topP?: number
  frequencyPenalty?: number
  presencePenalty?: number
}

function normalizeRemoteModels(payload: unknown): Model[] {
  const source = Array.isArray(payload)
    ? payload
    : Object.entries((payload || {}) as Record<string, any>).map(([model, data]) => ({
      model,
      ...data
    }))

  return source
    .map((item: any): Model | null => {
      const modelId = String(item?.model || item?.id || '')
      if (!modelId) return null
      const model: Model = {
        id: modelId,
        name: item?.display_name || item?.name || modelId,
        provider: String(item?.provider || 'unknown'),
        type: (item?.type || MODEL_FORM_DEFAULTS.type) as Model['type'],
        enabled: item?.enabled !== false,
        maxTokens: Number(item?.max_tokens ?? item?.maxTokens ?? MODEL_FORM_DEFAULTS.maxTokens),
        inputPrice: Number(item?.input_price ?? item?.inputPrice ?? MODEL_FORM_DEFAULTS.inputPrice),
        outputPrice: Number(item?.output_price ?? item?.outputPrice ?? MODEL_FORM_DEFAULTS.outputPrice),
        description: item?.description || undefined,
        createdAt: item?.created_at || item?.createdAt,
        updatedAt: item?.updated_at || item?.updatedAt
      }
      return model
    })
    .filter((item): item is Model => item !== null)
}

export const useModelsStore = defineStore('models', () => {
  const models = ref<Model[]>([])
  const loading = ref(false)
  const submitting = ref(false)
  const error = ref<Error | null>(null)
  const lastFetchTime = ref<number>(0)

  const enabledModels = computed(() => models.value.filter(m => m.enabled))
  
  const modelCount = computed(() => ({
    total: models.value.length,
    enabled: enabledModels.value.length,
    byProvider: models.value.reduce((acc, m) => {
      acc[m.provider] = (acc[m.provider] || 0) + 1
      return acc
    }, {} as Record<string, number>),
    byType: models.value.reduce((acc, m) => {
      acc[m.type] = (acc[m.type] || 0) + 1
      return acc
    }, {} as Record<string, number>)
  }))

  const modelsByProvider = computed(() => {
    const map: Record<string, Model[]> = {}
    models.value.forEach(model => {
      if (!map[model.provider]) {
        map[model.provider] = []
      }
      map[model.provider]!.push(model)
    })
    return map
  })

  const chatModels = computed(() => enabledModels.value.filter(m => m.type === 'chat'))
  const embeddingModels = computed(() => enabledModels.value.filter(m => m.type === 'embedding'))

  const fetchModels = async () => {
    loading.value = true
    error.value = null
    try {
      const payload = await getRouterModels()
      models.value = normalizeRemoteModels(payload)
      lastFetchTime.value = Date.now()
      return models.value
    } catch (e: any) {
      models.value = []
      error.value = e
      throw e
    } finally {
      loading.value = false
    }
  }

  const createModel = async (data: Partial<Model>): Promise<boolean> => {
    submitting.value = true
    try {
      const newModel: Model = {
        id: data.id || `model-${Date.now()}`,
        name: data.name || '',
        provider: data.provider || 'unknown',
        type: data.type || MODEL_FORM_DEFAULTS.type,
        enabled: data.enabled ?? MODEL_FORM_DEFAULTS.enabled,
        maxTokens: data.maxTokens || MODEL_FORM_DEFAULTS.maxTokens,
        inputPrice: data.inputPrice || MODEL_FORM_DEFAULTS.inputPrice,
        outputPrice: data.outputPrice || MODEL_FORM_DEFAULTS.outputPrice,
        description: data.description || undefined
      }
      models.value.push(newModel)
      ElMessage.success('模型添加成功')
      eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '添加失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const updateModel = async (id: string, data: Partial<Model>): Promise<boolean> => {
    submitting.value = true
    try {
      const index = models.value.findIndex(m => m.id === id)
      if (index >= 0) {
        const existing = models.value[index]!
        models.value[index] = { 
          ...existing,
          ...data,
          description: data.description !== undefined ? data.description : existing.description
        }
        ElMessage.success('模型更新成功')
        eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
        return true
      }
      ElMessage.error('模型不存在')
      return false
    } catch (e: any) {
      ElMessage.error(e?.message || '更新失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const deleteModel = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      const index = models.value.findIndex(m => m.id === id)
      if (index >= 0) {
        models.value.splice(index, 1)
        ElMessage.success('模型删除成功')
        eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
        return true
      }
      ElMessage.error('模型不存在')
      return false
    } catch (e: any) {
      ElMessage.error(e?.message || '删除失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const toggleModel = (id: string, enabled: boolean): boolean => {
    const model = models.value.find(m => m.id === id)
    if (model) {
      model.enabled = enabled
      eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
      return true
    }
    return false
  }

  const findById = (id: string): Model | undefined => {
    return models.value.find(m => m.id === id)
  }

  const findByProvider = (provider: string): Model[] => {
    return models.value.filter(m => m.provider === provider)
  }

  const getEnabledModelsByProvider = (provider: string): Model[] => {
    return models.value.filter(m => m.provider === provider && m.enabled)
  }

  const calculateCost = (modelId: string, inputTokens: number, outputTokens: number): number => {
    const model = findById(modelId)
    if (!model) return 0
    return (inputTokens * model.inputPrice + outputTokens * model.outputPrice) / 1000000
  }

  return {
    models,
    loading,
    submitting,
    error,
    lastFetchTime,
    enabledModels,
    modelCount,
    modelsByProvider,
    chatModels,
    embeddingModels,
    fetchModels,
    createModel,
    updateModel,
    deleteModel,
    toggleModel,
    findById,
    findByProvider,
    getEnabledModelsByProvider,
    calculateCost
  }
})
