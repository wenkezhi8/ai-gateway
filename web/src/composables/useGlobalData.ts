import { onMounted, onUnmounted } from 'vue'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { useProvidersStore } from '@/store/providers'
import { useAccountsStore } from '@/store/accounts'
import { useModelsStore } from '@/store/models'
import { useStatsStore } from '@/store/stats'
import { useAlertsStore } from '@/store/alerts'

interface UseGlobalDataOptions {
  providers?: boolean
  accounts?: boolean
  models?: boolean
  stats?: boolean
  alerts?: boolean
  autoRefresh?: boolean
  refreshInterval?: number
}

export function useGlobalData(options: UseGlobalDataOptions = {}) {
  const {
    providers: loadProviders = true,
    accounts: loadAccounts = true,
    models: loadModels = true,
    stats: loadStats = true,
    alerts: loadAlerts = false,
    autoRefresh = true,
    refreshInterval = 30000
  } = options

  const providersStore = loadProviders ? useProvidersStore() : null
  const accountsStore = loadAccounts ? useAccountsStore() : null
  const modelsStore = loadModels ? useModelsStore() : null
  const statsStore = loadStats ? useStatsStore() : null
  const alertsStore = loadAlerts ? useAlertsStore() : null

  let refreshTimer: number | null = null

  const initialize = async () => {
    const promises: Promise<any>[] = []
    
    if (providersStore) promises.push(providersStore.fetchProviders())
    if (accountsStore) promises.push(accountsStore.fetchAccounts())
    if (modelsStore) promises.push(modelsStore.fetchModels())
    if (statsStore) promises.push(statsStore.refresh())
    if (alertsStore) promises.push(alertsStore.fetchAlerts())
    
    await Promise.allSettled(promises)
  }

  const refresh = async () => {
    const promises: Promise<any>[] = []
    
    if (providersStore) promises.push(providersStore.fetchProviders(true))
    if (accountsStore) promises.push(accountsStore.fetchAccounts(true))
    if (modelsStore) promises.push(modelsStore.fetchModels())
    if (statsStore) promises.push(statsStore.refresh())
    if (alertsStore) promises.push(alertsStore.fetchAlerts(true))
    
    await Promise.allSettled(promises)
  }

  const startAutoRefresh = () => {
    if (!autoRefresh || refreshTimer) return
    refreshTimer = window.setInterval(refresh, refreshInterval)
  }

  const stopAutoRefresh = () => {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }

  onMounted(() => {
    initialize()
    if (autoRefresh) {
      startAutoRefresh()
    }
  })

  onUnmounted(() => {
    stopAutoRefresh()
  })

  return {
    providersStore,
    accountsStore,
    modelsStore,
    statsStore,
    alertsStore,
    initialize,
    refresh,
    startAutoRefresh,
    stopAutoRefresh
  }
}

export function useDataSync(stores: {
  providers?: ReturnType<typeof useProvidersStore>
  accounts?: ReturnType<typeof useAccountsStore>
  models?: ReturnType<typeof useModelsStore>
  stats?: ReturnType<typeof useStatsStore>
  alerts?: ReturnType<typeof useAlertsStore>
}) {
  const handleDataChange = async (event: string) => {
    switch (event) {
      case DATA_EVENTS.PROVIDERS_CHANGED:
        if (stores.providers) await stores.providers.fetchProviders(true)
        if (stores.stats) await stores.stats.refresh()
        break
      case DATA_EVENTS.ACCOUNTS_CHANGED:
        if (stores.accounts) await stores.accounts.fetchAccounts(true)
        if (stores.stats) await stores.stats.refresh()
        break
      case DATA_EVENTS.MODELS_CHANGED:
        if (stores.models) await stores.models.fetchModels()
        break
      case DATA_EVENTS.ALERTS_CHANGED:
        if (stores.alerts) await stores.alerts.fetchAlerts(true)
        if (stores.stats) await stores.stats.refresh()
        break
      case DATA_EVENTS.STATS_CHANGED:
        if (stores.stats) await stores.stats.refresh()
        break
      case DATA_EVENTS.ALL_DATA_REFRESH:
        await Promise.allSettled([
          stores.providers?.fetchProviders(true),
          stores.accounts?.fetchAccounts(true),
          stores.models?.fetchModels(),
          stores.stats?.refresh(),
          stores.alerts?.fetchAlerts(true)
        ].filter(Boolean))
        break
    }
  }

  const eventsToListen = [
    DATA_EVENTS.PROVIDERS_CHANGED,
    DATA_EVENTS.ACCOUNTS_CHANGED,
    DATA_EVENTS.MODELS_CHANGED,
    DATA_EVENTS.ALERTS_CHANGED,
    DATA_EVENTS.STATS_CHANGED,
    DATA_EVENTS.ALL_DATA_REFRESH
  ]

  const unsubscribers: (() => void)[] = []

  onMounted(() => {
    eventsToListen.forEach(event => {
      const unsub = eventBus.on(event, () => handleDataChange(event))
      unsubscribers.push(unsub)
    })
  })

  onUnmounted(() => {
    unsubscribers.forEach(unsub => unsub())
  })
}

export function emitDataChange(event: keyof typeof DATA_EVENTS) {
  eventBus.emit(DATA_EVENTS[event])
}
