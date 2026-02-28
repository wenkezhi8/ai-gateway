import { defineStore } from 'pinia'
import { ref } from 'vue'

import type { LoadState } from './types'
import {
  cleanupInvalidEntries,
  clearCacheByType,
  createCacheRule,
  deleteCacheEntry,
  deleteCacheEntryGroup,
  deleteCacheRule,
  getCacheConfig,
  getCacheEntries,
  getCacheEntryDetail,
  getCacheHealth,
  getCacheRules,
  getCacheStats,
  getSemanticSignatures,
  getTtlConfig,
  updateCacheConfig,
  updateCacheRule,
  updateTtlConfig
} from '@/api/cache-domain'

function normalizeError(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return '请求失败'
}

export const useCacheDomainStore = defineStore('cache-domain', () => {
  const status = ref<LoadState>('idle')
  const error = ref('')

  const stats = ref<Record<string, any>>({})
  const config = ref<Record<string, any>>({})
  const health = ref<Record<string, any>>({})
  const rules = ref<any[]>([])
  const signatures = ref<any[]>([])
  const ttlConfig = ref<Record<string, any>>({})
  const entries = ref<any[]>([])
  const entriesTotal = ref(0)
  const entryDetail = ref<Record<string, any> | null>(null)

  async function init() {
    status.value = 'loading'
    error.value = ''

    try {
      const [statsData, configData, healthData, rulesData, signaturesData, ttlData] = await Promise.all([
        getCacheStats(),
        getCacheConfig(),
        getCacheHealth(),
        getCacheRules(),
        getSemanticSignatures(12),
        getTtlConfig()
      ])

      stats.value = statsData || {}
      config.value = configData || {}
      health.value = healthData || {}
      rules.value = Array.isArray(rulesData) ? rulesData : []
      signatures.value = Array.isArray(signaturesData) ? signaturesData : []
      ttlConfig.value = ttlData || {}

      const hasData = Object.keys(stats.value).length > 0 || rules.value.length > 0
      status.value = hasData ? 'success' : 'empty'
    } catch (err) {
      status.value = 'error'
      error.value = normalizeError(err)
    }
  }

  async function loadEntries(query: string) {
    const data = await getCacheEntries(query)
    const payload = data || {}
    entries.value = Array.isArray(payload.entries) ? payload.entries : []
    entriesTotal.value = Number(payload.total || 0)
  }

  async function saveConfig(payload: Record<string, unknown>) {
    await updateCacheConfig(payload)
    config.value = {
      ...config.value,
      ...payload
    }
  }

  async function saveTtlConfig(payload: Record<string, unknown>) {
    await updateTtlConfig(payload)
    ttlConfig.value = {
      ...ttlConfig.value,
      ...payload
    }
  }

  async function clearByType(cacheType: string) {
    await clearCacheByType(cacheType)
    await init()
  }

  async function addRule(payload: Record<string, unknown>) {
    await createCacheRule(payload)
    rules.value = await getCacheRules()
  }

  async function patchRule(ruleId: number, payload: Record<string, unknown>) {
    await updateCacheRule(ruleId, payload)
    rules.value = await getCacheRules()
  }

  async function removeRule(ruleId: number) {
    await deleteCacheRule(ruleId)
    rules.value = await getCacheRules()
  }

  async function loadEntryDetail(key: string) {
    entryDetail.value = await getCacheEntryDetail(key)
    return entryDetail.value
  }

  async function removeEntry(key: string, query: string) {
    await deleteCacheEntry(key)
    await loadEntries(query)
  }

  async function removeEntryGroup(payload: Record<string, unknown>, query: string) {
    await deleteCacheEntryGroup(payload)
    await loadEntries(query)
  }

  async function runCleanupInvalid(query: string) {
    await cleanupInvalidEntries()
    await loadEntries(query)
  }

  return {
    status,
    error,
    stats,
    config,
    health,
    rules,
    signatures,
    ttlConfig,
    entries,
    entriesTotal,
    entryDetail,
    init,
    loadEntries,
    saveConfig,
    saveTtlConfig,
    clearByType,
    addRule,
    patchRule,
    removeRule,
    loadEntryDetail,
    removeEntry,
    removeEntryGroup,
    runCleanupInvalid
  }
})
