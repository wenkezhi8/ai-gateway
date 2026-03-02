import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import {
  checkEditionDependencies,
  getEditionConfig,
  getEditionDefinitions,
  type DependencyStatus,
  type EditionConfig,
  type EditionType,
  updateEditionConfig
} from '@/api/edition-domain'

function basicFallback(): EditionConfig {
  return {
    type: 'basic',
    features: {
      vector_cache: false,
      vector_db_management: false,
      knowledge_base: false,
      cold_hot_tiering: false
    },
    display_name: '基础版',
    description: '纯AI网关功能',
    dependencies: ['redis']
  }
}

function normalizeError(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return '请求失败'
}

export const useEditionStore = defineStore('edition-domain', () => {
  const config = ref<EditionConfig | null>(null)
  const definitions = ref<EditionConfig[]>([])
  const dependencies = ref<Record<string, DependencyStatus>>({})
  const loading = ref(false)
  const updating = ref(false)
  const error = ref('')

  const isBasic = computed(() => config.value?.type === 'basic')
  const isStandard = computed(() => config.value?.type === 'standard')
  const isEnterprise = computed(() => config.value?.type === 'enterprise')

  const hasVectorCache = computed(() => config.value?.features.vector_cache ?? false)
  const hasVectorDBManagement = computed(() => config.value?.features.vector_db_management ?? false)
  const hasKnowledgeBase = computed(() => config.value?.features.knowledge_base ?? false)
  const hasColdHotTiering = computed(() => config.value?.features.cold_hot_tiering ?? false)

  async function fetchEditionConfig() {
    loading.value = true
    error.value = ''
    try {
      config.value = await getEditionConfig()
    } catch (err) {
      config.value = basicFallback()
      error.value = normalizeError(err)
    } finally {
      loading.value = false
    }
  }

  async function fetchDefinitions() {
    definitions.value = await getEditionDefinitions()
  }

  async function checkDependencies() {
    dependencies.value = await checkEditionDependencies()
  }

  async function updateEdition(type: EditionType) {
    updating.value = true
    try {
      const data = await updateEditionConfig(type)
      config.value = data.edition
      return {
        success: true,
        restartRequired: data.restart_required
      }
    } catch (err) {
      return {
        success: false,
        message: normalizeError(err)
      }
    } finally {
      updating.value = false
    }
  }

  return {
    config,
    definitions,
    dependencies,
    loading,
    updating,
    error,
    isBasic,
    isStandard,
    isEnterprise,
    hasVectorCache,
    hasVectorDBManagement,
    hasKnowledgeBase,
    hasColdHotTiering,
    fetchEditionConfig,
    fetchDefinitions,
    checkDependencies,
    updateEdition
  }
})
