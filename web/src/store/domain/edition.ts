import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import {
  checkEditionDependencies,
  getEditionSetupTask,
  getEditionConfig,
  getEditionDefinitions,
  setupEditionEnvironment,
  type DependencyStatus,
  type EditionConfig,
  type EditionSetupRequest,
  type EditionSetupTask,
  type EditionType,
  updateEditionConfig
} from '@/api/edition-domain'

function standardFallback(): EditionConfig {
  return {
    type: 'standard',
    features: {
      vector_cache: true,
      vector_db_management: false,
      knowledge_base: false,
      cold_hot_tiering: false
    },
    display_name: '标准版',
    description: '网关 + 语义缓存，中大规模场景',
    dependencies: ['redis', 'ollama'],
    runtime: 'docker',
    dependency_versions: {
      redis: '7.2.0-v18',
      ollama: 'latest',
      qdrant: 'latest'
    }
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
  const setupTask = ref<EditionSetupTask | null>(null)
  const loading = ref(false)
  const updating = ref(false)
  const setupLoading = ref(false)
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
      config.value = standardFallback()
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

  async function startSetup(payload: EditionSetupRequest) {
    setupLoading.value = true
    try {
      return await setupEditionEnvironment(payload)
    } finally {
      setupLoading.value = false
    }
  }

  async function fetchSetupTask(taskId: string) {
    const task = await getEditionSetupTask(taskId)
    setupTask.value = task
    return task
  }

  return {
    config,
    definitions,
    dependencies,
    setupTask,
    loading,
    updating,
    setupLoading,
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
    updateEdition,
    startSetup,
    fetchSetupTask
  }
})
