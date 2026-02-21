import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { ElMessage } from 'element-plus'

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

const MODEL_STORAGE_KEY = 'ai-gateway-models-config'

const defaultModels: Model[] = [
  { id: 'gpt-4o', name: 'GPT-4o', provider: 'openai', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 2.5, outputPrice: 10 },
  { id: 'gpt-4-turbo', name: 'GPT-4 Turbo', provider: 'openai', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 10, outputPrice: 30 },
  { id: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo', provider: 'openai', type: 'chat', enabled: true, maxTokens: 16385, inputPrice: 0.5, outputPrice: 1.5 },
  { id: 'claude-3-5-sonnet', name: 'Claude 3.5 Sonnet', provider: 'anthropic', type: 'chat', enabled: true, maxTokens: 200000, inputPrice: 3, outputPrice: 15 },
  { id: 'claude-3-opus', name: 'Claude 3 Opus', provider: 'anthropic', type: 'chat', enabled: true, maxTokens: 200000, inputPrice: 15, outputPrice: 75 },
  { id: 'glm-4-plus', name: 'GLM-4 Plus', provider: 'zhipu', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 0.05, outputPrice: 0.05 },
  { id: 'glm-4-flash', name: 'GLM-4 Flash', provider: 'zhipu', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 0.001, outputPrice: 0.001 },
  { id: 'qwen-max', name: '通义千问 Max', provider: 'qwen', type: 'chat', enabled: true, maxTokens: 32000, inputPrice: 0.04, outputPrice: 0.12 },
  { id: 'qwen-plus', name: '通义千问 Plus', provider: 'qwen', type: 'chat', enabled: true, maxTokens: 128000, inputPrice: 0.0008, outputPrice: 0.002 },
  { id: 'deepseek-chat', name: 'DeepSeek Chat', provider: 'deepseek', type: 'chat', enabled: true, maxTokens: 64000, inputPrice: 0.001, outputPrice: 0.002 },
  { id: 'deepseek-coder', name: 'DeepSeek Coder', provider: 'deepseek', type: 'chat', enabled: true, maxTokens: 64000, inputPrice: 0.001, outputPrice: 0.002 },
  { id: 'doubao-pro', name: '豆包 Pro', provider: 'volcengine', type: 'chat', enabled: true, maxTokens: 32000, inputPrice: 0.0008, outputPrice: 0.002 },
]

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

  const loadFromStorage = () => {
    try {
      const saved = localStorage.getItem(MODEL_STORAGE_KEY)
      if (saved) {
        const parsed = JSON.parse(saved)
        const merged = defaultModels.map(m => {
          const savedModel = parsed.find((s: Model) => s.id === m.id)
          return savedModel ? { ...m, ...savedModel } : m
        })
        const newModels = parsed.filter((s: Model) => !defaultModels.find(m => m.id === s.id))
        models.value = [...merged, ...newModels]
        return
      }
    } catch (e) {
      console.error('Failed to load models from storage:', e)
    }
    models.value = [...defaultModels]
  }

  const saveToStorage = () => {
    try {
      localStorage.setItem(MODEL_STORAGE_KEY, JSON.stringify(models.value))
    } catch (e) {
      console.error('Failed to save models to storage:', e)
    }
  }

  const fetchModels = async () => {
    loading.value = true
    error.value = null
    try {
      loadFromStorage()
      lastFetchTime.value = Date.now()
      return models.value
    } catch (e: any) {
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
        provider: data.provider || 'openai',
        type: data.type || 'chat',
        enabled: data.enabled ?? true,
        maxTokens: data.maxTokens || 4096,
        inputPrice: data.inputPrice || 0,
        outputPrice: data.outputPrice || 0,
        description: data.description || undefined
      }
      models.value.push(newModel)
      saveToStorage()
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
        saveToStorage()
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
        saveToStorage()
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
      saveToStorage()
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
    calculateCost,
    loadFromStorage,
    saveToStorage
  }
})
