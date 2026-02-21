import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { accountApi, type Account, type LimitConfig } from '@/api/account'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { ElMessage } from 'element-plus'

export const useAccountsStore = defineStore('accounts', () => {
  const accounts = ref<Account[]>([])
  const loading = ref(false)
  const submitting = ref(false)
  const error = ref<Error | null>(null)
  const lastFetchTime = ref<number>(0)
  const cacheTimeout = 30000

  const enabledAccounts = computed(() => accounts.value.filter(a => a.enabled))
  
  const activeAccounts = computed(() => accounts.value.filter(a => a.is_active))
  
  const accountCount = computed(() => ({
    total: accounts.value.length,
    enabled: enabledAccounts.value.length,
    active: activeAccounts.value.length,
    warning: accounts.value.filter(a => 
      a.usage?.token?.warning_level === 'warning' || a.usage?.rpm?.warning_level === 'warning'
    ).length,
    exceeded: accounts.value.filter(a => 
      (a.usage?.token?.percent_used ?? 0) >= 100 || (a.usage?.rpm?.percent_used ?? 0) >= 100
    ).length
  }))

  const accountsByProvider = computed(() => {
    const map: Record<string, Account[]> = {}
    accounts.value.forEach(account => {
      if (!map[account.provider]) {
        map[account.provider] = []
      }
      map[account.provider]!.push(account)
    })
    return map
  })

  const fetchAccounts = async (force = false) => {
    const now = Date.now()
    if (!force && now - lastFetchTime.value < cacheTimeout && accounts.value.length > 0) {
      return accounts.value
    }

    loading.value = true
    error.value = null
    try {
      const res = await accountApi.getList()
      accounts.value = (res as any).data || []
      lastFetchTime.value = now
      return accounts.value
    } catch (e: any) {
      error.value = e
      throw e
    } finally {
      loading.value = false
    }
  }

  const createAccount = async (data: Partial<Account>): Promise<boolean> => {
    submitting.value = true
    try {
      await accountApi.create(data as any)
      ElMessage.success('账号创建成功')
      await fetchAccounts(true)
      eventBus.emit(DATA_EVENTS.ACCOUNTS_CHANGED)
      eventBus.emit(DATA_EVENTS.STATS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '创建失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const updateAccount = async (id: string, data: Partial<Account>): Promise<boolean> => {
    submitting.value = true
    try {
      await accountApi.update(id, data as any)
      ElMessage.success('账号更新成功')
      await fetchAccounts(true)
      eventBus.emit(DATA_EVENTS.ACCOUNTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '更新失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const deleteAccount = async (id: string): Promise<boolean> => {
    submitting.value = true
    try {
      await accountApi.delete(id)
      ElMessage.success('账号删除成功')
      await fetchAccounts(true)
      eventBus.emit(DATA_EVENTS.ACCOUNTS_CHANGED)
      eventBus.emit(DATA_EVENTS.STATS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '删除失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const toggleAccount = async (id: string, enabled: boolean): Promise<boolean> => {
    try {
      await accountApi.toggleStatus(id, enabled)
      const account = accounts.value.find(a => a.id === id)
      if (account) {
        account.enabled = enabled
      }
      eventBus.emit(DATA_EVENTS.ACCOUNTS_CHANGED)
      eventBus.emit(DATA_EVENTS.STATS_CHANGED)
      return true
    } catch (e: any) {
      const account = accounts.value.find(a => a.id === id)
      if (account) {
        account.enabled = !enabled
      }
      ElMessage.error(e?.message || '状态切换失败')
      return false
    }
  }

  const updateAccountLimits = async (id: string, limits: Record<string, LimitConfig>): Promise<boolean> => {
    submitting.value = true
    try {
      await accountApi.updateLimits(id, limits)
      ElMessage.success('限额配置已保存')
      await fetchAccounts(true)
      eventBus.emit(DATA_EVENTS.ACCOUNTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '保存失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const forceSwitch = async (provider: string, accountId: string): Promise<boolean> => {
    try {
      await accountApi.forceSwitch(provider, accountId)
      ElMessage.success('账号已激活')
      await fetchAccounts(true)
      eventBus.emit(DATA_EVENTS.ACCOUNTS_CHANGED)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '切换失败')
      return false
    }
  }

  const findById = (id: string): Account | undefined => {
    return accounts.value.find(a => a.id === id)
  }

  const findByProvider = (provider: string): Account[] => {
    return accounts.value.filter(a => a.provider === provider)
  }

  const getActiveAccountForProvider = (provider: string): Account | undefined => {
    return accounts.value.find(a => a.provider === provider && a.is_active)
  }

  return {
    accounts,
    loading,
    submitting,
    error,
    lastFetchTime,
    enabledAccounts,
    activeAccounts,
    accountCount,
    accountsByProvider,
    fetchAccounts,
    createAccount,
    updateAccount,
    deleteAccount,
    toggleAccount,
    updateAccountLimits,
    forceSwitch,
    findById,
    findByProvider,
    getActiveAccountForProvider
  }
})
