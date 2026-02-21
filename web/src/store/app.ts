import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Provider {
  id: number
  name: string
  type: string
  enabled: boolean
}

export const useAppStore = defineStore('app', () => {
  const sidebarCollapsed = ref(false)
  const providers = ref<Provider[]>([])
  const loading = ref(false)

  const toggleSidebar = () => {
    sidebarCollapsed.value = !sidebarCollapsed.value
  }

  const setProviders = (list: Provider[]) => {
    providers.value = list
  }

  const setLoading = (value: boolean) => {
    loading.value = value
  }

  return {
    sidebarCollapsed,
    providers,
    loading,
    toggleSidebar,
    setProviders,
    setLoading
  }
})
