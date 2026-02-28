import { defineStore } from 'pinia'
import { ref } from 'vue'

import type { LoadState } from './types'
import {
  getOpsDashboard,
  getOpsExportMetrics,
  getOpsProviderHealth,
  getOpsServices
} from '@/api/ops-domain'

function normalizeError(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return '请求失败'
}

export const useOpsDomainStore = defineStore('ops-domain', () => {
  const status = ref<LoadState>('idle')
  const error = ref('')

  const dashboard = ref<Record<string, any>>({})
  const system = ref<Record<string, any>>({})
  const realtime = ref<Record<string, any>>({})
  const resources = ref<Record<string, any>>({})
  const diagnosis = ref<Record<string, any>>({})
  const services = ref<any[]>([])
  const providers = ref<any[]>([])

  async function loadDashboard(range: string) {
    const data = await getOpsDashboard(range)
    dashboard.value = data || {}
    system.value = dashboard.value.system || {}
    realtime.value = dashboard.value.realtime || {}
    resources.value = dashboard.value.resources || {}
    diagnosis.value = dashboard.value.diagnosis || {}
  }

  async function loadServices() {
    services.value = await getOpsServices()
  }

  async function loadProviders() {
    providers.value = await getOpsProviderHealth()
  }

  async function init(range = '1h') {
    status.value = 'loading'
    error.value = ''

    try {
      await Promise.all([
        loadDashboard(range),
        loadServices(),
        loadProviders()
      ])
      const hasData = Object.keys(dashboard.value).length > 0 || services.value.length > 0
      status.value = hasData ? 'success' : 'empty'
    } catch (err) {
      status.value = 'error'
      error.value = normalizeError(err)
    }
  }

  async function exportMetrics() {
    return getOpsExportMetrics()
  }

  return {
    status,
    error,
    dashboard,
    system,
    realtime,
    resources,
    diagnosis,
    services,
    providers,
    init,
    loadDashboard,
    loadServices,
    loadProviders,
    exportMetrics
  }
})
