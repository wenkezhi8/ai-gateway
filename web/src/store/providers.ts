import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { providerApi } from '@/api/provider'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { ElMessage } from 'element-plus'

export interface Provider {
  id: string
  name: string
  type: string
  endpoint: string
  enabled: boolean
  accounts: number
  latency: string
  models: string[]
  createdAt?: string
  updatedAt?: string
  testing?: boolean
}

export const useProvidersStore = defineStore('providers', () => {
  const providers = ref<Provider[]>([])
  const loading = ref(false)
  const error = ref<Error | null>(null)
  const lastFetchTime = ref<number>(0)
  const cacheTimeout = 30000

  const enabledProviders = computed(() => providers.value.filter(p => p.enabled))
  
  const providerCount = computed(() => ({
    total: providers.value.length,
    enabled: enabledProviders.value.length,
    disabled: providers.value.length - enabledProviders.value.length
  }))

  const providerTypes = computed(() => {
    const types = new Set(providers.value.map(p => p.type))
    return Array.from(types)
  })

  const fetchProviders = async (force = false) => {
    const now = Date.now()
    if (!force && now - lastFetchTime.value < cacheTimeout && providers.value.length > 0) {
      return providers.value
    }

    loading.value = true
    error.value = null
    try {
      const res = await providerApi.getList()
      providers.value = (res as any).data || []
      lastFetchTime.value = now
      return providers.value
    } catch (e: any) {
      error.value = e
      throw e
    } finally {
      loading.value = false
    }
  }

  const createProvider = async (data: Partial<Provider>): Promise<boolean> => {
    try {
      await providerApi.create(data as any)
      ElMessage.success('服务商创建成功')
      await fetchProviders(true)
      eventBus.emit(DATA_EVENTS.PROVIDERS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '创建失败')
      return false
    }
  }

  const updateProvider = async (id: string, data: Partial<Provider>): Promise<boolean> => {
    try {
      await providerApi.update(id, data as any)
      ElMessage.success('服务商更新成功')
      await fetchProviders(true)
      eventBus.emit(DATA_EVENTS.PROVIDERS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '更新失败')
      return false
    }
  }

  const deleteProvider = async (id: string): Promise<boolean> => {
    try {
      await providerApi.delete(id)
      ElMessage.success('服务商删除成功')
      await fetchProviders(true)
      eventBus.emit(DATA_EVENTS.PROVIDERS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '删除失败')
      return false
    }
  }

  const toggleProvider = async (id: string, enabled: boolean): Promise<boolean> => {
    try {
      await providerApi.toggleStatus(id, enabled)
      const provider = providers.value.find(p => p.id.toString() === id.toString())
      if (provider) {
        provider.enabled = enabled
      }
      eventBus.emit(DATA_EVENTS.PROVIDERS_CHANGED)
      eventBus.emit(DATA_EVENTS.STATS_CHANGED)
      return true
    } catch (e: any) {
      const provider = providers.value.find(p => p.id.toString() === id.toString())
      if (provider) {
        provider.enabled = !enabled
      }
      ElMessage.error(e?.message || '状态切换失败')
      return false
    }
  }

  const testConnection = async (id: string): Promise<{ success: boolean; latency?: number; error?: string }> => {
    const provider = providers.value.find(p => p.id.toString() === id.toString())
    if (provider) {
      provider.testing = true
    }
    
    try {
      const res = await providerApi.testConnection(id)
      return { success: true, latency: (res as any).response_time_ms }
    } catch (e: any) {
      return { success: false, error: e?.message || '连接测试失败' }
    } finally {
      if (provider) {
        provider.testing = false
      }
    }
  }

  const findById = (id: string): Provider | undefined => {
    return providers.value.find(p => p.id.toString() === id.toString())
  }

  const findByType = (type: string): Provider[] => {
    return providers.value.filter(p => p.type === type)
  }

  return {
    providers,
    loading,
    error,
    lastFetchTime,
    enabledProviders,
    providerCount,
    providerTypes,
    fetchProviders,
    createProvider,
    updateProvider,
    deleteProvider,
    toggleProvider,
    testConnection,
    findById,
    findByType
  }
})
