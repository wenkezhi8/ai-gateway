import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { alertApi, type Alert as ApiAlert } from '@/api/alert'
import { DATA_EVENTS, eventBus } from '@/utils/eventBus'

export type Alert = ApiAlert

interface AlertHistoryEnvelope {
  data?: {
    list?: Alert[]
    total?: number
  }
}

function extractHistoryList(payload: AlertHistoryEnvelope | Alert[]): Alert[] {
  if (Array.isArray(payload)) {
    return payload
  }

  if (Array.isArray(payload.data?.list)) {
    return payload.data.list
  }

  return []
}

export const useAlertsStore = defineStore('alerts', () => {
  const alerts = ref<Alert[]>([])
  const loading = ref(false)
  const submitting = ref(false)
  const error = ref<Error | null>(null)
  const lastFetchTime = ref<number>(0)

  const pendingAlerts = computed(() => alerts.value.filter((alert) => alert.status === 'pending'))
  const resolvedAlerts = computed(() => alerts.value.filter((alert) => alert.status === 'resolved'))

  const alertCount = computed(() => ({
    total: alerts.value.length,
    pending: pendingAlerts.value.length,
    resolved: resolvedAlerts.value.length,
    critical: pendingAlerts.value.filter((alert) => alert.level === 'critical').length,
    warning: pendingAlerts.value.filter((alert) => alert.level === 'warning').length
  }))

  const alertsBySource = computed(() => {
    const map: Record<string, Alert[]> = {}
    alerts.value.forEach((alert) => {
      if (!map[alert.source]) {
        map[alert.source] = []
      }
      map[alert.source]!.push(alert)
    })

    return map
  })

  const fetchAlerts = async (silent = false) => {
    loading.value = !silent
    error.value = null

    try {
      const response = await alertApi.getHistory()
      alerts.value = extractHistoryList(response as AlertHistoryEnvelope)
      lastFetchTime.value = Date.now()
    } catch (err) {
      const fetchError = err instanceof Error ? err : new Error('获取告警失败')
      error.value = fetchError
      if (!silent) {
        ElMessage.error(fetchError.message || '获取告警失败')
      }
    } finally {
      loading.value = false
    }
  }

  const resolveAlert = async (id: string): Promise<boolean> => {
    submitting.value = true

    try {
      await alertApi.resolveAlert(id)
      const alert = alerts.value.find((item) => item.id === id)
      if (alert) {
        alert.status = 'resolved'
      }

      ElMessage.success('告警已处理')
      eventBus.emit(DATA_EVENTS.ALERTS_CHANGED)
      return true
    } catch (err) {
      const resolveError = err instanceof Error ? err : new Error('处理失败')
      ElMessage.error(resolveError.message || '处理失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const findById = (id: string): Alert | undefined => alerts.value.find((item) => item.id === id)

  const getAlertsByStatus = (status: Alert['status']): Alert[] => {
    return alerts.value.filter((alert) => alert.status === status)
  }

  const getAlertsByLevel = (level: Alert['level']): Alert[] => {
    return alerts.value.filter((alert) => alert.level === level)
  }

  return {
    alerts,
    loading,
    submitting,
    error,
    lastFetchTime,
    pendingAlerts,
    resolvedAlerts,
    alertCount,
    alertsBySource,
    fetchAlerts,
    resolveAlert,
    findById,
    getAlertsByStatus,
    getAlertsByLevel
  }
})
