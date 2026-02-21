import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { request } from '@/api/request'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'

export interface OverviewStats {
  total_requests: number
  requests_today: number
  success_rate: number
  avg_latency_ms: number
  total_tokens: number
  active_accounts: number
  active_providers: number
  cache_hit_rate: number
}

export interface ProviderStat {
  name: string
  requests: number
  tokens: number
  success_rate: number
  avg_latency_ms: number
}

export interface RequestTrendPoint {
  timestamp: string
  requests: number
  success: number
  failed: number
  avg_latency_ms: number
}

export interface RealtimeData {
  timestamp: string
  active_connections: number
  requests_per_minute: number
  tokens_per_minute: number
  avg_latency_ms: number
  error_rate: number
}

export const useStatsStore = defineStore('stats', () => {
  const overview = ref<OverviewStats | null>(null)
  const providerStats = ref<ProviderStat[]>([])
  const requestTrend = ref<RequestTrendPoint[]>([])
  const realtime = ref<RealtimeData | null>(null)
  
  const loading = ref(false)
  const error = ref<Error | null>(null)
  const lastFetchTime = ref<number>(0)

  const formattedOverview = computed(() => {
    if (!overview.value) return null
    return {
      totalRequests: formatNumber(overview.value.total_requests),
      requestsToday: formatNumber(overview.value.requests_today),
      successRate: `${(overview.value.success_rate * 100).toFixed(1)}%`,
      avgLatency: `${overview.value.avg_latency_ms.toFixed(0)}ms`,
      totalTokens: formatTokens(overview.value.total_tokens),
      activeAccounts: overview.value.active_accounts,
      activeProviders: overview.value.active_providers,
      cacheHitRate: `${(overview.value.cache_hit_rate * 100).toFixed(1)}%`
    }
  })

  const topProviders = computed(() => {
    return [...providerStats.value]
      .sort((a, b) => b.requests - a.requests)
      .slice(0, 5)
  })

  function formatNumber(num: number): string {
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
    return num.toString()
  }

  function formatTokens(num: number): string {
    if (num >= 1000000000) return (num / 1000000000).toFixed(2) + 'B'
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
    return num.toString()
  }

  const fetchOverview = async (silent = true) => {
    loading.value = !silent
    error.value = null
    try {
      const res = await request.get<{ success: boolean; data: any }>('/admin/dashboard/stats', { 
        silent: true 
      } as any)
      if ((res as any).data) {
        overview.value = (res as any).data
        providerStats.value = (res as any).data.provider_stats || []
        lastFetchTime.value = Date.now()
      }
    } catch (e: any) {
      error.value = e
      if (!silent) throw e
    } finally {
      loading.value = false
    }
  }

  const fetchRequestTrend = async (params?: { period?: string; interval?: string }) => {
    try {
      const res = await request.get<{ success: boolean; data: any }>('/admin/dashboard/requests', { 
        params,
        silent: true 
      } as any)
      if ((res as any).data?.data) {
        requestTrend.value = (res as any).data.data
      }
    } catch (e) {
      console.error('Failed to fetch request trend:', e)
    }
  }

  const fetchRealtime = async () => {
    try {
      const res = await request.get<{ success: boolean; data: any }>('/admin/dashboard/stats', { 
        silent: true 
      } as any)
      if ((res as any).data) {
        realtime.value = {
          timestamp: new Date().toISOString(),
          active_connections: (res as any).data.active_accounts || 0,
          requests_per_minute: Math.floor((res as any).data.requests_today / 1440) || 0,
          tokens_per_minute: Math.floor((res as any).data.total_tokens / 1440) || 0,
          avg_latency_ms: (res as any).data.avg_latency_ms || 0,
          error_rate: 1 - ((res as any).data.success_rate || 1)
        }
      }
    } catch (e) {
      console.error('Failed to fetch realtime data:', e)
    }
  }

  const refresh = async () => {
    await Promise.all([
      fetchOverview(true),
      fetchRequestTrend(),
      fetchRealtime()
    ])
    eventBus.emit(DATA_EVENTS.STATS_CHANGED)
  }

  return {
    overview,
    providerStats,
    requestTrend,
    realtime,
    loading,
    error,
    lastFetchTime,
    formattedOverview,
    topProviders,
    fetchOverview,
    fetchRequestTrend,
    fetchRealtime,
    refresh
  }
})
