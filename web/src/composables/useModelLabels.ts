import { ref } from 'vue'
import { request } from '@/api/request'

export interface ModelInfo {
  id: string
  display_name?: string
}

export interface ModelLabelMap {
  [key: string]: string
}

// Shared state across all consumers
const modelLabels = ref<ModelLabelMap>({})
const labelsLoading = ref(false)

export function useModelLabels() {
  const fetchModelLabels = async (providerId?: string) => {
    labelsLoading.value = true
    try {
      const res = await request.get<{
        success: boolean
        data: Array<{
          model: string
          provider: string
          display_name?: string
          enabled: boolean
        }>
      }>('/admin/router/model-registry', { silent: true } as any)

      if ((res as any).success && (res as any).data) {
        const labels: ModelLabelMap = {}
        for (const m of (res as any).data) {
          if (!m.enabled || !m.model) continue
          if (providerId && m.provider !== providerId) continue
          
          const key = `${m.provider}::${m.model}`
          labels[key] = m.display_name || m.model
        }
        modelLabels.value = labels
      }
    } finally {
      labelsLoading.value = false
    }
  }

  const getModelLabel = (provider: string, model: string): string => {
    const key = `${provider}::${model}`
    return modelLabels.value[key] || model
  }

  const getModelLabelsForProvider = (provider: string): ModelLabelMap => {
    const result: ModelLabelMap = {}
    for (const [key, value] of Object.entries(modelLabels.value)) {
      if (key.startsWith(`${provider}::`)) {
        const model = key.replace(`${provider}::`, '')
        result[model] = value
      }
    }
    return result
  }

  const resetLabels = () => {
    modelLabels.value = {}
  }

  return {
    modelLabels,
    loading: labelsLoading,
    fetchModelLabels,
    getModelLabel,
    getModelLabelsForProvider,
    resetLabels
  }
}
