import { ref, reactive, computed, onMounted, watch } from 'vue'
import { ElMessageBox } from 'element-plus'

import {
  getFeedbackStats,
  getRouterConfig,
  getTaskTypeDistribution,
  triggerFeedbackOptimization
} from '@/api/routing-domain'
import { getUiSettings, updateRoutingUiSettings } from '@/api/settings-domain'
import { createDefaultTaskTypes } from '@/constants/routing'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'

import type { RoutingPanelState } from '../routing-types'

type PanelKey = 'policy'

export function useRoutingConsole() {
  const panelState = reactive<Record<PanelKey, RoutingPanelState>>({
    policy: 'loading'
  })
  const panelError = reactive<Record<PanelKey, string>>({
    policy: ''
  })

  const autoSaveEnabled = ref(false)
  const lastSavedAt = ref<string | null>(null)

  const config = reactive({
    mode: 'auto',
    defaultStrategy: 'auto',
    defaultModel: ''
  })

  const strategies = ref<Array<{ value: string; label: string; description: string }>>([])
  const taskTypes = ref(createDefaultTaskTypes())

  const feedbackStats = reactive({
    total: 0,
    positive: 0,
    positiveRate: 0,
    avgRating: 0,
    modelsTracked: 0
  })

  const statsCards = computed(() => [
    { title: '总反馈数', value: feedbackStats.total.toString(), icon: 'ChatDotRound', color: '#007AFF' },
    { title: '好评率', value: `${feedbackStats.positiveRate}%`, icon: 'CircleCheckFilled', color: '#34C759' },
    { title: '追踪模型', value: feedbackStats.modelsTracked.toString(), icon: 'DataAnalysis', color: '#FF9500' },
    { title: '平均评分', value: feedbackStats.avgRating.toFixed(1), icon: 'StarFilled', color: '#5856D6' }
  ])

  const modeLabel = computed(() => {
    const labels: Record<string, string> = {
      auto: 'Auto 智能选择',
      default: 'Default 服务商默认',
      fixed: '固定模型',
      latest: 'Latest 最新'
    }
    return labels[config.mode] || config.mode
  })

  const strategyLabel = computed(() => {
    return strategies.value.find((s) => s.value === config.defaultStrategy)?.label || config.defaultStrategy
  })

  const lastSavedLabel = computed(() => {
    if (!lastSavedAt.value) return '未保存'
    const date = new Date(lastSavedAt.value)
    if (Number.isNaN(date.getTime())) return '未保存'
    return date.toLocaleString()
  })

  async function loadConfig() {
    const data: any = await getRouterConfig()
    if (!data) return

    config.defaultStrategy = data.default_strategy || 'auto'
    config.defaultModel = data.default_model || ''
    const mode = data.use_auto_mode
    config.mode = typeof mode === 'string' ? mode : (mode ? 'auto' : 'fixed')
    if (data.strategies) {
      strategies.value = data.strategies
    }
  }

  async function loadFeedbackStats() {
    const stats: any = await getFeedbackStats()
    if (!stats) return

    feedbackStats.total = stats.total_feedback || 0
    feedbackStats.positive = stats.positive_count || 0
    feedbackStats.modelsTracked = stats.models_tracked || 0
    feedbackStats.avgRating = stats.avg_rating || 0
    feedbackStats.positiveRate = feedbackStats.total > 0
      ? Math.round((feedbackStats.positive / feedbackStats.total) * 100)
      : 0
  }

  async function loadTaskTypeDistribution() {
    const data: any = await getTaskTypeDistribution()
    if (!Array.isArray(data?.distribution) || data.distribution.length === 0) return

    const countMap: Record<string, number> = {}
    const percentMap: Record<string, number> = {}
    for (const item of data.distribution) {
      countMap[item.task_type] = item.count
      percentMap[item.task_type] = item.percent
    }
    taskTypes.value = taskTypes.value.map((task) => ({
      ...task,
      count: countMap[task.type] || 0,
      percentage: percentMap[task.type] || 0
    }))
  }

  async function loadRoutingUiSettings() {
    const uiSettings = await getUiSettings()
    autoSaveEnabled.value = Boolean(uiSettings?.routing?.auto_save_enabled)
    lastSavedAt.value = uiSettings?.routing?.last_saved_at || null
  }

  async function reloadPolicyPanel() {
    panelState.policy = 'loading'
    panelError.policy = ''
    try {
      await loadRoutingUiSettings()
      await Promise.all([loadConfig(), loadFeedbackStats(), loadTaskTypeDistribution()])
      const hasData = taskTypes.value.some((t) => t.count > 0) || feedbackStats.total > 0
      panelState.policy = hasData ? 'success' : 'empty'
    } catch (e) {
      panelError.policy = '路由策略加载失败'
      panelState.policy = 'error'
    }
  }

  async function reloadAllPanels() {
    await reloadPolicyPanel()
  }

  async function triggerOptimization() {
    try {
      await ElMessageBox.confirm('确定要触发自动优化吗？这将根据反馈数据调整模型评分（每个模型至少需要 10 条样本）。', '确认', { type: 'info' })
      const resp: any = await triggerFeedbackOptimization()
      const result = (resp && typeof resp === 'object' && 'data' in resp) ? (resp as any).data : resp
      const msg = resp?.message || '优化已完成'
      handleSuccess(`${msg}（扫描:${result.models_scanned || 0}，可优化:${result.models_eligible || 0}，已更新:${result.models_updated || 0}）`)
      await loadFeedbackStats()
    } catch (e) {
      if ((e as any) !== 'cancel') {
        handleApiError(e, '优化失败')
      }
    }
  }

  watch(autoSaveEnabled, (value) => {
    updateRoutingUiSettings({
      auto_save_enabled: value,
      last_saved_at: lastSavedAt.value || ''
    }).catch(() => {})
  })

  onMounted(async () => {
    await reloadAllPanels()
  })

  return {
    panelState,
    panelError,
    autoSaveEnabled,
    lastSavedAt,
    config,
    taskTypes,
    feedbackStats,
    statsCards,
    modeLabel,
    strategyLabel,
    lastSavedLabel,
    loadTaskTypeDistribution,
    triggerOptimization,
    reloadPolicyPanel,
    reloadAllPanels
  }
}
