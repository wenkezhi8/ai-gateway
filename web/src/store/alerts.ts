import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  acknowledgeAlert as acknowledgeAlertApi,
  acknowledgeAllAlerts,
  clearResolvedAlerts,
  getAlerts,
  resolveAlert as resolveAlertApi
} from '@/api/alert-domain'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { ElMessage } from 'element-plus'

export interface Alert {
  id: string
  type: 'warning' | 'error' | 'info' | 'success'
  title: string
  message: string
  source: string
  status: 'active' | 'acknowledged' | 'resolved'
  severity: 'low' | 'medium' | 'high' | 'critical'
  timestamp: string
  acknowledgedAt?: string
  acknowledgedBy?: string
  metadata?: Record<string, any>
}

export const useAlertsStore = defineStore('alerts', () => {
  const alerts = ref<Alert[]>([])
  const loading = ref(false)
  const submitting = ref(false)
  const error = ref<Error | null>(null)
  const lastFetchTime = ref<number>(0)

  const activeAlerts = computed(() => alerts.value.filter(a => a.status === 'active'))
  
  const acknowledgedAlerts = computed(() => alerts.value.filter(a => a.status === 'acknowledged'))
  
  const resolvedAlerts = computed(() => alerts.value.filter(a => a.status === 'resolved'))

  const alertCount = computed(() => ({
    total: alerts.value.length,
    active: activeAlerts.value.length,
    acknowledged: acknowledgedAlerts.value.length,
    resolved: resolvedAlerts.value.length,
    critical: activeAlerts.value.filter(a => a.severity === 'critical').length,
    high: activeAlerts.value.filter(a => a.severity === 'high').length
  }))

  const alertsByType = computed(() => {
    const map: Record<string, Alert[]> = {}
    alerts.value.forEach(alert => {
      if (!map[alert.type]) {
        map[alert.type] = []
      }
      map[alert.type]!.push(alert)
    })
    return map
  })

  const alertsBySource = computed(() => {
    const map: Record<string, Alert[]> = {}
    alerts.value.forEach(alert => {
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
      const res: any = await getAlerts()
      if (Array.isArray(res)) {
        alerts.value = res
      } else if (Array.isArray(res?.data)) {
        alerts.value = res.data
      } else {
        alerts.value = []
      }
      lastFetchTime.value = Date.now()
    } catch (e: any) {
      error.value = e
      if (!silent) {
        ElMessage.error(e?.message || '获取告警失败')
      }
    } finally {
      loading.value = false
    }
  }

  const acknowledgeAlert = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      await acknowledgeAlertApi(id)
      const alert = alerts.value.find(a => a.id === id)
      if (alert) {
        alert.status = 'acknowledged'
        alert.acknowledgedAt = new Date().toISOString()
      }
      ElMessage.success('告警已确认')
      eventBus.emit(DATA_EVENTS.ALERTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '确认失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const resolveAlert = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      await resolveAlertApi(id)
      const alert = alerts.value.find(a => a.id === id)
      if (alert) {
        alert.status = 'resolved'
      }
      ElMessage.success('告警已解决')
      eventBus.emit(DATA_EVENTS.ALERTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '解决失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const acknowledgeAll = async (): Promise<boolean> => {
    submitting.value = true
    try {
      await acknowledgeAllAlerts()
      activeAlerts.value.forEach(alert => {
        alert.status = 'acknowledged'
        alert.acknowledgedAt = new Date().toISOString()
      })
      ElMessage.success('所有告警已确认')
      eventBus.emit(DATA_EVENTS.ALERTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '批量确认失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const clearResolved = async (): Promise<boolean> => {
    submitting.value = true
    try {
      await clearResolvedAlerts()
      alerts.value = alerts.value.filter(a => a.status !== 'resolved')
      ElMessage.success('已清理解决的告警')
      eventBus.emit(DATA_EVENTS.ALERTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '清理失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const findById = (id: string): Alert | undefined => {
    return alerts.value.find(a => a.id === id)
  }

  const getAlertsByStatus = (status: Alert['status']): Alert[] => {
    return alerts.value.filter(a => a.status === status)
  }

  const getAlertsBySeverity = (severity: Alert['severity']): Alert[] => {
    return alerts.value.filter(a => a.severity === severity)
  }

  return {
    alerts,
    loading,
    submitting,
    error,
    lastFetchTime,
    activeAlerts,
    acknowledgedAlerts,
    resolvedAlerts,
    alertCount,
    alertsByType,
    alertsBySource,
    fetchAlerts,
    acknowledgeAlert,
    resolveAlert,
    acknowledgeAll,
    clearResolved,
    findById,
    getAlertsByStatus,
    getAlertsBySeverity
  }
})
