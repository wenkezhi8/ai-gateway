import { defineStore } from 'pinia'
import { ref } from 'vue'

import type { LoadState } from './types'
import {
  getAvailableModels,
  getCascadeRules,
  getClassifierHealth,
  getClassifierModels,
  getClassifierStats,
  getFeedbackStats,
  getOllamaStatus,
  getRouterConfig,
  getRouterModels,
  getTaskModelMapping,
  getTaskTypeDistribution,
  getClassifierSwitchTask,
  putTaskModelMapping,
  switchClassifierModelAsync,
  triggerFeedbackOptimization,
  updateModelScore,
  updateRouterConfig
} from '@/api/routing-domain'

const classifierSwitchPollIntervalMs = 1500
const classifierSwitchMaxPollCount = 80

function normalizeError(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return '请求失败'
}

export const useRoutingDomainStore = defineStore('routing-domain', () => {
  const status = ref<LoadState>('idle')
  const error = ref('')
  const lastSavedAt = ref<string>('')

  const config = ref<Record<string, any>>({})
  const modelScores = ref<any[]>([])
  const availableModels = ref<any[]>([])
  const cascadeRules = ref<any[]>([])
  const feedbackStats = ref<Record<string, any>>({})
  const classifierHealth = ref<Record<string, any>>({})
  const classifierStats = ref<Record<string, any>>({})
  const classifierModels = ref<string[]>([])
  const ollamaStatus = ref<Record<string, any>>({})
  const taskModelMapping = ref<Record<string, string>>({})
  const taskTypeDistribution = ref<any[]>([])

  async function init() {
    status.value = 'loading'
    error.value = ''

    try {
      const [
        configData,
        scoresData,
        availableData,
        cascadeData,
        feedbackData,
        healthData,
        statsData,
        classifierModelsData,
        mappingData,
        distributionData
      ] = await Promise.all([
        getRouterConfig(),
        getRouterModels(),
        getAvailableModels(),
        getCascadeRules(),
        getFeedbackStats(),
        getClassifierHealth(),
        getClassifierStats(),
        getClassifierModels(),
        getTaskModelMapping(),
        getTaskTypeDistribution()
      ])

      config.value = configData || {}
      modelScores.value = Array.isArray(scoresData) ? scoresData : []
      availableModels.value = Array.isArray(availableData) ? availableData : []
      cascadeRules.value = Array.isArray(cascadeData) ? cascadeData : []
      feedbackStats.value = feedbackData || {}
      classifierHealth.value = healthData || {}
      classifierStats.value = statsData || {}
      classifierModels.value = Array.isArray(classifierModelsData?.models) ? classifierModelsData.models : []
      taskModelMapping.value = mappingData || {}
      taskTypeDistribution.value = Array.isArray(distributionData?.distribution) ? distributionData.distribution : []

      const hasData = Object.keys(config.value).length > 0 ||
        modelScores.value.length > 0 ||
        availableModels.value.length > 0
      status.value = hasData ? 'success' : 'empty'
    } catch (err) {
      status.value = 'error'
      error.value = normalizeError(err)
    }
  }

  async function refreshClassifierStatus(model: string) {
    ollamaStatus.value = await getOllamaStatus(model)
    return ollamaStatus.value
  }

  async function saveTaskMapping(mapping: Record<string, string>) {
    await putTaskModelMapping(mapping)
    taskModelMapping.value = { ...mapping }
    lastSavedAt.value = new Date().toISOString()
  }

  async function saveRouterConfig(payload: Record<string, unknown>) {
    await updateRouterConfig(payload)
    await init()
  }

  async function saveModelScore(model: string, payload: Record<string, unknown>) {
    await updateModelScore(model, payload)
    const found = modelScores.value.findIndex((item) => item.model === model)
    if (found >= 0) {
      modelScores.value[found] = {
        ...modelScores.value[found],
        ...payload
      }
      return
    }
    await init()
  }

  async function switchClassifierModel(model: string) {
    const task = await switchClassifierModelAsync(model)
    const taskId = task?.task_id || task?.taskId
    if (!taskId) {
      throw new Error('切换任务创建失败')
    }

    for (let i = 0; i < classifierSwitchMaxPollCount; i += 1) {
      const resp = await getClassifierSwitchTask(`/admin/router/classifier/switch-tasks/${encodeURIComponent(taskId)}`)
      const taskStatus = String(resp?.status || '').toLowerCase()

      if (taskStatus === 'success') {
        await init()
        return
      }
      if (taskStatus === 'failed' || taskStatus === 'timeout') {
        throw new Error(resp?.last_error || '切换分类模型失败')
      }

      await new Promise((resolve) => window.setTimeout(resolve, classifierSwitchPollIntervalMs))
    }

    throw new Error('切换超时')
  }

  async function runOptimization() {
    const result = await triggerFeedbackOptimization()
    await init()
    return result
  }

  return {
    status,
    error,
    lastSavedAt,
    config,
    modelScores,
    availableModels,
    cascadeRules,
    feedbackStats,
    classifierHealth,
    classifierStats,
    classifierModels,
    ollamaStatus,
    taskModelMapping,
    taskTypeDistribution,
    init,
    refreshClassifierStatus,
    saveTaskMapping,
    saveRouterConfig,
    saveModelScore,
    switchClassifierModel,
    runOptimization
  }
})
