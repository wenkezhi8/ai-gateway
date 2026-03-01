import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import {
  getAvailableModels,
  getCascadeRules,
  getClassifierHealth,
  getClassifierModels,
  getClassifierStats,
  getClassifierSwitchTask,
  getFeedbackStats,
  getIntentEngineConfig,
  getIntentEngineHealth,
  getOllamaStatus,
  getRouterConfig,
  getRouterModels,
  getTaskModelMapping,
  getTaskTypeDistribution,
  getVectorTierConfig,
  getVectorTierStats,
  installOllama as installOllamaApi,
  promoteVectorTierEntry,
  pullOllamaModel as pullOllamaModelApi,
  putTaskModelMapping,
  startOllama as startOllamaApi,
  stopOllama as stopOllamaApi,
  switchClassifierModelAsync,
  triggerVectorTierMigrate,
  triggerFeedbackOptimization,
  updateVectorTierConfig,
  updateIntentEngineConfig,
  updateModelScore,
  updateRouterConfig
} from '@/api/routing-domain'
import {
  getCacheConfig,
  getVectorPipelineHealth,
  getVectorStats,
  rebuildVectorIndex as rebuildVectorIndexApi,
  testVectorPipeline,
  updateCacheConfig
} from '@/api/cache-domain'
import { getUiSettings, updateRoutingUiSettings } from '@/api/settings-domain'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'
import { formatDuration } from '@/utils/format-duration'
import {
  DEFAULT_CLASSIFIER_CONFIG,
  ROUTING_OLLAMA_DEFAULT_MODEL,
  createDefaultTaskModelMapping,
  createDefaultTaskTypes,
} from '@/constants/routing'
import type { RoutingPanelState } from '../routing-types'

interface ModelScore {
  model: string
  provider: string
  quality_score: number
  speed_score: number
  cost_score: number
  enabled: boolean
}

interface CascadeRule {
  task_type: string
  difficulty: string
  start_level: string
  max_level: string
}

interface ModelOption {
  id: string
  display_name?: string
}

const classifierSwitchPollIntervalMs = 2000
const classifierSwitchLoadingMessage = '正在加载模型，首次可能较慢（最多180秒）'
const classifierSwitchTimeoutMessage = '模型加载超时，请继续等待Ollama完成加载后重试'
const ollamaStatusPollIntervalMs = 8000

type PanelKey = 'policy' | 'ollama' | 'models' | 'vector'

export function useRoutingConsole() {
  const activeTab = ref<PanelKey>('policy')

  const panelState = reactive<Record<PanelKey, RoutingPanelState>>({
    policy: 'loading',
    ollama: 'loading',
    models: 'loading',
    vector: 'loading'
  })

  const panelError = reactive<Record<PanelKey, string>>({
    policy: '',
    ollama: '',
    models: '',
    vector: ''
  })

  const saving = ref(false)
  const modelSearch = ref('')
  const modelScores = ref<ModelScore[]>([])
  const availableModels = ref<ModelOption[]>([])
  const autoSaveEnabled = ref(false)
  const lastSavedAt = ref<string | null>(null)
  const isMappingReady = ref(false)
  const classifierSaving = ref(false)
  const intentEngineSaving = ref(false)
  const classifierSwitching = ref(false)
  const classifierModelsLoading = ref(false)
  const classifierSwitchModel = ref('')
  const switchPollingCancelled = ref(false)
  const ollamaInstalling = ref(false)
  const ollamaStarting = ref(false)
  const ollamaStopping = ref(false)
  const ollamaPulling = ref(false)
  const ollamaRefreshing = ref(false)
  const ollamaModelInput = ref(ROUTING_OLLAMA_DEFAULT_MODEL)
  const vectorRefreshing = ref(false)
  const vectorRebuilding = ref(false)
  const vectorTierConfigSaving = ref(false)
  const tierMigrating = ref(false)
  const tierPromoting = ref(false)
  const promoteCacheKey = ref('')

  const classifierConfig = reactive(JSON.parse(JSON.stringify(DEFAULT_CLASSIFIER_CONFIG)))

  const classifierHealth = reactive({
    healthy: false,
    latency_ms: 0,
    message: '未检查'
  })

  const classifierStats = reactive({
    total_requests: 0,
    llm_attempts: 0,
    llm_success: 0,
    fallbacks: 0,
    shadow_requests: 0,
    avg_llm_latency_ms: 0,
    avg_control_latency_ms: 0,
    parse_errors: 0,
    control_fields_missing: 0
  })

  const intentEngineConfig = reactive({
    enabled: false,
    base_url: 'http://127.0.0.1:18566',
    timeout_ms: 1500,
    language: 'zh-CN',
    expected_dimension: 1024
  })

  const intentEngineHealth = reactive({
    healthy: false,
    latency_ms: 0,
    message: '未检查'
  })

  const vectorStats = reactive({
    enabled: false,
    index_name: '',
    key_prefix: '',
    dimension: 0,
    query_timeout_ms: 0,
    message: ''
  })

  const vectorTierStats = reactive({
    enabled: false,
    cold_vector_enabled: false,
    cold_vector_query_enabled: false,
    cold_vector_backend: 'sqlite',
    cold_vector_dual_write_enabled: false,
    hot_memory_usage_percent: 0,
    hot_memory_high_watermark_percent: 75,
    hot_memory_relief_percent: 65,
    migration_runs: 0,
    migration_moved: 0,
    migration_failed: 0,
    promote_success: 0,
    promote_failed: 0,
    message: ''
  })

  const vectorTierConfig = reactive({
    cold_vector_enabled: false,
    cold_vector_query_enabled: true,
    cold_vector_backend: 'sqlite',
    cold_vector_dual_write_enabled: false,
    cold_vector_similarity_threshold: 0.92,
    cold_vector_top_k: 1,
    hot_memory_high_watermark_percent: 75,
    hot_memory_relief_percent: 65,
    hot_to_cold_batch_size: 500,
    hot_to_cold_interval_seconds: 30,
    cold_vector_qdrant_url: '',
    cold_vector_qdrant_api_key: '',
    cold_vector_qdrant_collection: 'ai_gateway_cold_vectors',
    cold_vector_qdrant_timeout_ms: 1500
  })

  const vectorPipelineSaving = ref(false)
  const vectorPipelineTesting = ref(false)
  const vectorPipelineConfig = reactive({
    vector_pipeline_enabled: true,
    vector_standard_key_version: 'v2',
    vector_embedding_provider: 'ollama',
    vector_ollama_base_url: 'http://127.0.0.1:11434',
    vector_ollama_embedding_model: 'nomic-embed-text',
    vector_ollama_embedding_dimension: 1024,
    vector_ollama_embedding_timeout_ms: 1500,
    vector_ollama_endpoint_mode: 'auto',
    vector_writeback_enabled: true
  })

  const vectorPipelineHealth = reactive({
    enabled: false,
    healthy: false,
    message: '未检查',
    embedding_latency_ms: 0,
    embedding_dimension_actual: 0,
    vector_index_dimension: 0,
    dimension_match: false
  })

  const vectorPipelineTestForm = reactive({
    query: '',
    task_type: 'qa',
    top_k: 5,
    min_similarity: 0.92
  })

  const vectorPipelineTestResult = ref<any | null>(null)

  const ollamaSetup = reactive({
    installed: false,
    running: false,
    model: ROUTING_OLLAMA_DEFAULT_MODEL,
    model_installed: false,
    running_model: '',
    running_models: [] as string[],
    running_model_details: [] as Array<{ name: string; size_vram: number }>,
    running_vram_bytes_total: 0,
    keep_alive_disabled: false,
    message: ''
  })

  const config = reactive({
    mode: 'auto',
    defaultStrategy: 'auto',
    defaultModel: 'deepseek-chat'
  })

  const taskModelMapping = reactive<Record<string, { enabled: boolean, model: string }>>({
    ...createDefaultTaskModelMapping()
  })

  const strategies = ref<Array<{ value: string; label: string; description: string }>>([])

  const feedbackStats = reactive({
    total: 0,
    positive: 0,
    positiveRate: 0,
    avgRating: 0,
    modelsTracked: 0
  })

  const cascadeRules = ref<CascadeRule[]>([])
  const taskTypes = ref(createDefaultTaskTypes())

  const cascadeLevels = computed(() => {
    const groups: Record<string, string[]> = { small: [], medium: [], large: [] }
    cascadeRules.value.forEach(rule => {
      const level = rule.start_level || 'medium'
      if (!groups[level]) return
      const item = `${rule.task_type}/${rule.difficulty}`
      if (!groups[level].includes(item)) groups[level].push(item)
    })
    return [
      { key: 'small', label: '小型', type: 'success', desc: '快速响应，低成本', models: groups.small },
      { key: 'medium', label: '中型', type: 'warning', desc: '平衡质量与速度', models: groups.medium },
      { key: 'large', label: '大型', type: 'danger', desc: '最高质量，复杂任务', models: groups.large },
    ]
  })

  const statsCards = computed(() => [
    { title: '总反馈数', value: feedbackStats.total.toString(), icon: 'ChatDotRound', color: '#007AFF' },
    { title: '好评率', value: `${feedbackStats.positiveRate}%`, icon: 'CircleCheckFilled', color: '#34C759' },
    { title: '追踪模型', value: feedbackStats.modelsTracked.toString(), icon: 'DataAnalysis', color: '#FF9500' },
    { title: '平均评分', value: feedbackStats.avgRating.toFixed(1), icon: 'StarFilled', color: '#5856D6' }
  ])

  const filteredModels = computed(() => {
    if (!modelSearch.value) return modelScores.value
    const search = modelSearch.value.toLowerCase()
    return modelScores.value.filter(m =>
      m.model.toLowerCase().includes(search) ||
      m.provider.toLowerCase().includes(search)
    )
  })

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
    return strategies.value.find(s => s.value === config.defaultStrategy)?.label || config.defaultStrategy
  })

  const lastSavedLabel = computed(() => {
    if (!lastSavedAt.value) return '未保存'
    const date = new Date(lastSavedAt.value)
    if (Number.isNaN(date.getTime())) return '未保存'
    return date.toLocaleString()
  })

  const classifierConfidencePercent = computed({
    get: () => Math.round((classifierConfig.confidence_threshold || 0.65) * 100),
    set: (value: number) => {
      classifierConfig.confidence_threshold = value / 100
    }
  })

  function ensureControlConfig() {
    if (!classifierConfig.control) {
      ;(classifierConfig as any).control = {
        enable: false,
        shadow_only: true,
        normalized_query_read_enable: false,
        cache_write_gate_enable: false,
        risk_tag_enable: false,
        risk_block_enable: false,
        tool_gate_enable: false,
        model_fit_enable: false,
        parameter_hint_enable: false
      }
      return
    }
    classifierConfig.control.enable = Boolean(classifierConfig.control.enable)
    classifierConfig.control.shadow_only = Boolean(classifierConfig.control.shadow_only)
    classifierConfig.control.normalized_query_read_enable = Boolean(classifierConfig.control.normalized_query_read_enable)
    classifierConfig.control.cache_write_gate_enable = Boolean(classifierConfig.control.cache_write_gate_enable)
    classifierConfig.control.risk_tag_enable = Boolean(classifierConfig.control.risk_tag_enable)
    classifierConfig.control.risk_block_enable = Boolean(classifierConfig.control.risk_block_enable)
    classifierConfig.control.tool_gate_enable = Boolean(classifierConfig.control.tool_gate_enable)
    classifierConfig.control.model_fit_enable = Boolean(classifierConfig.control.model_fit_enable)
    classifierConfig.control.parameter_hint_enable = Boolean(classifierConfig.control.parameter_hint_enable)
  }

  function calculateCompositeScore(row: ModelScore): number {
    return Math.round(row.quality_score * 0.4 + row.speed_score * 0.35 + row.cost_score * 0.25)
  }

  function getScoreColor(score: number): string {
    if (score >= 80) return '#67c23a'
    if (score >= 60) return '#e6a23c'
    return '#f56c6c'
  }

  function getScoreTagType(score: number): string {
    if (score >= 80) return 'success'
    if (score >= 60) return 'warning'
    return 'danger'
  }

  function formatVramBytes(value: number): string {
    const bytes = Number(value || 0)
    if (bytes <= 0) return '0 B'
    if (bytes >= 1024 * 1024 * 1024) {
      return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GiB`
    }
    if (bytes >= 1024 * 1024) {
      return `${(bytes / (1024 * 1024)).toFixed(2)} MiB`
    }
    if (bytes >= 1024) {
      return `${(bytes / 1024).toFixed(2)} KiB`
    }
    return `${bytes} B`
  }

  async function loadConfig() {
    try {
      const data: any = await getRouterConfig()
      if (data) {
        config.defaultStrategy = data.default_strategy || 'auto'
        config.defaultModel = data.default_model || 'deepseek-chat'
        const mode = data.use_auto_mode
        if (typeof mode === 'string') {
          config.mode = mode
        } else {
          config.mode = mode ? 'auto' : 'fixed'
        }
        if (data.strategies) {
          strategies.value = data.strategies
        }
        if (data.classifier) {
          Object.assign(classifierConfig, data.classifier)
          ensureControlConfig()
          classifierSwitchModel.value = classifierConfig.active_model
        }
      }

      try {
        const mappingData: any = await getTaskModelMapping()
        if (mappingData) {
          for (const [taskType, model] of Object.entries(mappingData)) {
            if (taskModelMapping[taskType]) {
              taskModelMapping[taskType].enabled = true
              taskModelMapping[taskType].model = model as string
            }
          }
        }
      } catch (e) {
        console.warn('Failed to load task model mapping:', e)
      }
    } catch (e) {
      console.warn('Failed to load config:', e)
    } finally {
      isMappingReady.value = true
    }
  }

  async function loadModelScores() {
    try {
      const data: any = await getRouterModels()
      if (data) {
        const scores = data
        if (Array.isArray(scores)) {
          modelScores.value = scores.map((item: any) => ({
            model: item.model,
            provider: item.provider || 'unknown',
            quality_score: item.quality_score || 80,
            speed_score: item.speed_score || 80,
            cost_score: item.cost_score || 80,
            enabled: item.enabled ?? true
          }))
        } else {
          modelScores.value = Object.entries(scores).map(([model, score]) => ({
            model,
            provider: (score as any).provider || 'unknown',
            quality_score: (score as any).quality_score || 80,
            speed_score: (score as any).speed_score || 80,
            cost_score: (score as any).cost_score || 80,
            enabled: (score as any).enabled ?? true
          }))
        }
        availableModels.value = modelScores.value.map(m => ({ id: m.model }))
      }
    } catch (e) {
      console.warn('Failed to load model scores:', e)
    }
  }

  async function loadAvailableModels() {
    try {
      const data: any = await getAvailableModels()
      if (Array.isArray(data)) {
        availableModels.value = data
      }
    } catch (e) {
      console.warn('Failed to load available models:', e)
    }
  }

  async function loadCascadeRules() {
    try {
      const data: any = await getCascadeRules()
      cascadeRules.value = Array.isArray(data) ? data : []
    } catch (e) {
      console.warn('Failed to load cascade rules:', e)
    }
  }

  async function loadFeedbackStats() {
    try {
      const stats: any = await getFeedbackStats()
      if (stats) {
        feedbackStats.total = stats.total_feedback || 0
        feedbackStats.positive = stats.positive_count || 0
        feedbackStats.modelsTracked = stats.models_tracked || 0
        feedbackStats.avgRating = stats.avg_rating || 0
        if (feedbackStats.total > 0) {
          feedbackStats.positiveRate = Math.round((feedbackStats.positive / feedbackStats.total) * 100)
        }
      }
    } catch (e) {
      console.warn('Failed to load feedback stats:', e)
    }
  }

  async function loadClassifierHealth() {
    try {
      const health: any = await getClassifierHealth()
      classifierHealth.healthy = Boolean(health?.healthy)
      classifierHealth.latency_ms = Number(health?.latency_ms || 0)
      classifierHealth.message = health?.message || 'ok'
    } catch (e) {
      classifierHealth.healthy = false
      classifierHealth.message = '检查失败'
      classifierHealth.latency_ms = 0
      console.warn('Failed to load classifier health:', e)
    }
  }

  async function loadClassifierModels() {
    classifierModelsLoading.value = true
    try {
      const payload: any = await getClassifierModels()
      const models = Array.isArray(payload.models) ? payload.models : []
      if (models.length > 0) {
        classifierConfig.candidate_models = models
      }
      if (payload.active_model) {
        classifierConfig.active_model = payload.active_model
      }
      if (!classifierSwitchModel.value) {
        classifierSwitchModel.value = classifierConfig.active_model
      }
    } catch (e) {
      console.warn('Failed to load classifier models:', e)
    } finally {
      classifierModelsLoading.value = false
    }
  }

  async function loadClassifierStats() {
    try {
      const stats: any = await getClassifierStats()
      classifierStats.total_requests = Number(stats.total_requests || 0)
      classifierStats.llm_attempts = Number(stats.llm_attempts || 0)
      classifierStats.llm_success = Number(stats.llm_success || 0)
      classifierStats.fallbacks = Number(stats.fallbacks || 0)
      classifierStats.shadow_requests = Number(stats.shadow_requests || 0)
      classifierStats.avg_llm_latency_ms = Number(stats.avg_llm_latency_ms || 0)
      classifierStats.avg_control_latency_ms = Number(stats.avg_control_latency_ms || 0)
      classifierStats.parse_errors = Number(stats.parse_errors || 0)
      classifierStats.control_fields_missing = Number(stats.control_fields_missing || 0)
    } catch (e) {
      console.warn('Failed to load classifier stats:', e)
    }
  }

  async function loadIntentEngineConfigData() {
    try {
      const data: any = await getIntentEngineConfig()
      if (!data) return
      intentEngineConfig.enabled = Boolean(data.enabled)
      intentEngineConfig.base_url = data.base_url || intentEngineConfig.base_url
      intentEngineConfig.timeout_ms = Number(data.timeout_ms || intentEngineConfig.timeout_ms)
      intentEngineConfig.language = data.language || intentEngineConfig.language
      intentEngineConfig.expected_dimension = Number(data.expected_dimension || intentEngineConfig.expected_dimension)
    } catch (e) {
      console.warn('Failed to load intent engine config:', e)
    }
  }

  async function loadIntentEngineHealthData() {
    try {
      const data: any = await getIntentEngineHealth()
      intentEngineHealth.healthy = Boolean(data?.healthy)
      intentEngineHealth.latency_ms = Number(data?.latency_ms || 0)
      intentEngineHealth.message = data?.message || (data?.healthy ? 'ok' : 'not ready')
    } catch (e) {
      intentEngineHealth.healthy = false
      intentEngineHealth.latency_ms = 0
      intentEngineHealth.message = '检查失败'
      console.warn('Failed to load intent engine health:', e)
    }
  }

  async function saveIntentEngineConfigData() {
    intentEngineSaving.value = true
    try {
      await updateIntentEngineConfig({
        enabled: intentEngineConfig.enabled,
        base_url: intentEngineConfig.base_url,
        timeout_ms: Number(intentEngineConfig.timeout_ms || 1500),
        language: intentEngineConfig.language,
        expected_dimension: Number(intentEngineConfig.expected_dimension || 1024)
      })
      handleSuccess('Intent Engine 配置已保存')
      await loadIntentEngineHealthData()
    } catch (e) {
      handleApiError(e, '保存 Intent Engine 配置失败')
    } finally {
      intentEngineSaving.value = false
    }
  }

  async function loadOllamaSetupStatus() {
    ollamaRefreshing.value = true
    try {
      const model = (ollamaModelInput.value || classifierConfig.active_model || ROUTING_OLLAMA_DEFAULT_MODEL).trim()
      const payload: any = await getOllamaStatus(model)
      ollamaSetup.installed = Boolean(payload.installed)
      ollamaSetup.running = Boolean(payload.running)
      ollamaSetup.model = payload.model || model
      ollamaSetup.model_installed = Boolean(payload.model_installed)
      ollamaSetup.running_models = Array.isArray(payload.running_models) ? payload.running_models : []
      ollamaSetup.running_model_details = Array.isArray(payload.running_model_details) ? payload.running_model_details : []
      ollamaSetup.running_vram_bytes_total = Number(payload.running_vram_bytes_total || 0)
      ollamaSetup.running_model = String(payload.running_model || '')
      ollamaSetup.keep_alive_disabled = Boolean(payload.keep_alive_disabled)
      ollamaSetup.message = payload.message || ''
    } catch (e) {
      console.warn('Failed to load ollama setup status:', e)
    } finally {
      ollamaRefreshing.value = false
    }
  }

  async function installOllama() {
    ollamaInstalling.value = true
    try {
      await installOllamaApi()
      handleSuccess('Ollama 安装完成')
    } catch (e) {
      handleApiError(e, '安装 Ollama 失败')
    } finally {
      ollamaInstalling.value = false
      await loadOllamaSetupStatus()
    }
  }

  async function startOllama() {
    ollamaStarting.value = true
    try {
      await startOllamaApi()
      handleSuccess('Ollama 启动成功')
    } catch (e) {
      handleApiError(e, '启动 Ollama 失败')
    } finally {
      ollamaStarting.value = false
      await loadOllamaSetupStatus()
      await loadClassifierHealth()
    }
  }

  async function stopOllama() {
    ollamaStopping.value = true
    try {
      await stopOllamaApi()
      handleSuccess('Ollama 已停止')
    } catch (e) {
      handleApiError(e, '停止 Ollama 失败')
    } finally {
      ollamaStopping.value = false
      await loadOllamaSetupStatus()
      await loadClassifierHealth()
    }
  }

  async function pullOllamaModel() {
    const model = (ollamaModelInput.value || classifierConfig.active_model || ROUTING_OLLAMA_DEFAULT_MODEL).trim()
    if (!model) {
      handleApiError(new Error('模型名不能为空'), '安装模型失败')
      return
    }
    ollamaPulling.value = true
    try {
      await pullOllamaModelApi(model)
      handleSuccess(`模型安装成功: ${model}`)
      await loadClassifierModels()
    } catch (e) {
      handleApiError(e, '安装模型失败')
    } finally {
      ollamaPulling.value = false
      await loadOllamaSetupStatus()
    }
  }

  async function saveClassifierConfig() {
    classifierSaving.value = true
    try {
      await updateRouterConfig({
        classifier: {
          ...classifierConfig,
          confidence_threshold: Number(classifierConfig.confidence_threshold || 0.65)
        }
      })
      handleSuccess('分类器配置已保存')
      await Promise.all([loadClassifierHealth(), loadClassifierStats(), loadClassifierModels()])
    } catch (e) {
      handleApiError(e, '保存分类器配置失败')
    } finally {
      classifierSaving.value = false
    }
  }

  async function switchClassifierModel() {
    if (!classifierSwitchModel.value) {
      handleApiError(new Error('请选择要切换的模型'), '切换失败')
      return
    }
    classifierSwitching.value = true
    try {
      const switchResp: any = await switchClassifierModelAsync(classifierSwitchModel.value)
      const taskId = switchResp?.task_id || switchResp?.taskId
      if (!taskId) {
        throw new Error('切换任务创建失败')
      }
      await pollClassifierSwitchTask(taskId)

      classifierConfig.active_model = classifierSwitchModel.value
      handleSuccess('分类模型切换成功')
      await Promise.all([loadClassifierHealth(), loadClassifierStats()])
    } catch (e) {
      const err = e as any
      const detailMessage = err?.response?.data?.error?.message || err?.response?.data?.message
      if (typeof detailMessage === 'string' && detailMessage.trim()) {
        handleApiError(new Error(detailMessage), '切换分类模型失败')
      } else {
        handleApiError(e, '切换分类模型失败')
      }
    } finally {
      classifierSwitching.value = false
    }
  }

  async function pollClassifierSwitchTask(taskId: string) {
    const taskPath = `/admin/router/classifier/switch-tasks/${encodeURIComponent(taskId)}`

    while (!switchPollingCancelled.value) {
      const taskResp: any = await getClassifierSwitchTask(taskPath)
      const taskData = taskResp || {}
      const status = String(taskData?.status || '').toLowerCase()

      if (status === 'success') {
        return
      }
      if (status === 'timeout') {
        throw new Error(taskData?.last_error || classifierSwitchTimeoutMessage)
      }
      if (status === 'failed') {
        throw new Error(taskData?.last_error || '切换分类模型失败')
      }

      await new Promise(resolve => window.setTimeout(resolve, classifierSwitchPollIntervalMs))
    }

    throw new Error(classifierSwitchLoadingMessage)
  }

  async function saveTaskMapping(isAuto = false) {
    saving.value = true
    try {
      const mappingData: Record<string, string> = {}
      for (const [taskType, mapping] of Object.entries(taskModelMapping)) {
        if (mapping.enabled && mapping.model) {
          mappingData[taskType] = mapping.model
        }
      }
      await putTaskModelMapping(mappingData)
      const savedAt = new Date().toISOString()
      lastSavedAt.value = savedAt
      await updateRoutingUiSettings({
        auto_save_enabled: autoSaveEnabled.value,
        last_saved_at: savedAt
      })
      if (!isAuto) {
        handleSuccess('映射已保存')
      }
    } catch (e) {
      handleApiError(e, '保存失败')
    } finally {
      saving.value = false
    }
  }

  async function toggleModelEnabled(model: ModelScore) {
    try {
      await updateModelScore(model.model, {
        provider: model.provider,
        quality_score: model.quality_score,
        speed_score: model.speed_score,
        cost_score: model.cost_score,
        enabled: model.enabled
      })
      handleSuccess(`${model.model} 已${model.enabled ? '启用' : '禁用'}`)
    } catch (e) {
      model.enabled = !model.enabled
      handleApiError(e, '操作失败')
    }
  }

  async function triggerOptimization() {
    try {
      await ElMessageBox.confirm('确定要触发自动优化吗？这将根据反馈数据调整模型评分（每个模型至少需要 10 条样本）。', '确认', { type: 'info' })
      const resp: any = await triggerFeedbackOptimization()
      const result = (resp && typeof resp === 'object' && 'data' in resp) ? (resp as any).data : resp
      const msg = resp?.message || '优化已完成'
      handleSuccess(`${msg}（扫描:${result.models_scanned || 0}，可优化:${result.models_eligible || 0}，已更新:${result.models_updated || 0}）`)
      loadModelScores()
      loadFeedbackStats()
    } catch (e) {
      if ((e as any) !== 'cancel') {
        handleApiError(e, '优化失败')
      }
    }
  }

  async function loadTaskTypeDistribution() {
    try {
      const data: any = await getTaskTypeDistribution()
      if (Array.isArray(data?.distribution) && data.distribution.length > 0) {
        const countMap: Record<string, number> = {}
        const percentMap: Record<string, number> = {}
        for (const item of data.distribution) {
          countMap[item.task_type] = item.count
          percentMap[item.task_type] = item.percent
        }
        taskTypes.value = taskTypes.value.map(task => ({
          ...task,
          count: countMap[task.type] || 0,
          percentage: percentMap[task.type] || 0
        }))
      }
    } catch (e) {
      console.warn('Failed to load task type distribution:', e)
    }
  }

  async function loadRoutingUiSettings() {
    try {
      const uiSettings = await getUiSettings()
      autoSaveEnabled.value = Boolean(uiSettings?.routing?.auto_save_enabled)
      lastSavedAt.value = uiSettings?.routing?.last_saved_at || null
    } catch (e) {
      console.warn('Failed to load routing ui settings:', e)
      autoSaveEnabled.value = false
      lastSavedAt.value = null
    }
  }

  async function loadVectorPipelineConfigData() {
    try {
      const data: any = await getCacheConfig()
      if (!data) return

      vectorPipelineConfig.vector_pipeline_enabled = data.vector_pipeline_enabled ?? vectorPipelineConfig.vector_pipeline_enabled
      vectorPipelineConfig.vector_standard_key_version = data.vector_standard_key_version || vectorPipelineConfig.vector_standard_key_version
      vectorPipelineConfig.vector_embedding_provider = data.vector_embedding_provider || vectorPipelineConfig.vector_embedding_provider
      vectorPipelineConfig.vector_ollama_base_url = data.vector_ollama_base_url || vectorPipelineConfig.vector_ollama_base_url
      vectorPipelineConfig.vector_ollama_embedding_model = data.vector_ollama_embedding_model || vectorPipelineConfig.vector_ollama_embedding_model
      vectorPipelineConfig.vector_ollama_embedding_dimension = Number(data.vector_ollama_embedding_dimension || vectorPipelineConfig.vector_ollama_embedding_dimension)
      vectorPipelineConfig.vector_ollama_embedding_timeout_ms = Number(data.vector_ollama_embedding_timeout_ms || vectorPipelineConfig.vector_ollama_embedding_timeout_ms)
      vectorPipelineConfig.vector_ollama_endpoint_mode = data.vector_ollama_endpoint_mode || vectorPipelineConfig.vector_ollama_endpoint_mode
      vectorPipelineConfig.vector_writeback_enabled = data.vector_writeback_enabled ?? vectorPipelineConfig.vector_writeback_enabled
    } catch (e) {
      console.warn('Failed to load vector pipeline config:', e)
    }
  }

  async function loadVectorPipelineHealthData() {
    try {
      const data: any = await getVectorPipelineHealth()
      vectorPipelineHealth.enabled = Boolean(data?.enabled)
      vectorPipelineHealth.healthy = Boolean(data?.healthy)
      vectorPipelineHealth.message = data?.message || (data?.healthy ? 'ok' : 'not ready')
      vectorPipelineHealth.embedding_latency_ms = Number(data?.embedding_latency_ms || 0)
      vectorPipelineHealth.embedding_dimension_actual = Number(data?.embedding_dimension_actual || 0)
      vectorPipelineHealth.vector_index_dimension = Number(data?.vector_index_dimension || 0)
      vectorPipelineHealth.dimension_match = Boolean(data?.dimension_match)
    } catch (e) {
      vectorPipelineHealth.healthy = false
      vectorPipelineHealth.message = '检查失败'
      console.warn('Failed to load vector pipeline health:', e)
    }
  }

  async function saveVectorPipelineConfigData() {
    vectorPipelineSaving.value = true
    try {
      await updateCacheConfig({
        vector_pipeline_enabled: vectorPipelineConfig.vector_pipeline_enabled,
        vector_standard_key_version: vectorPipelineConfig.vector_standard_key_version,
        vector_embedding_provider: vectorPipelineConfig.vector_embedding_provider,
        vector_ollama_base_url: vectorPipelineConfig.vector_ollama_base_url,
        vector_ollama_embedding_model: vectorPipelineConfig.vector_ollama_embedding_model,
        vector_ollama_embedding_dimension: Number(vectorPipelineConfig.vector_ollama_embedding_dimension || 1024),
        vector_ollama_embedding_timeout_ms: Number(vectorPipelineConfig.vector_ollama_embedding_timeout_ms || 1500),
        vector_ollama_endpoint_mode: vectorPipelineConfig.vector_ollama_endpoint_mode,
        vector_writeback_enabled: vectorPipelineConfig.vector_writeback_enabled
      })
      handleSuccess('向量 Pipeline 配置已保存')
      await Promise.all([loadVectorPipelineConfigData(), loadVectorPipelineHealthData(), loadVectorStatsData()])
    } catch (e) {
      handleApiError(e, '保存向量 Pipeline 配置失败')
    } finally {
      vectorPipelineSaving.value = false
    }
  }

  async function runVectorPipelineTest() {
    if (!vectorPipelineTestForm.query?.trim()) {
      handleApiError(new Error('测试文本不能为空'), '执行测试失败')
      return
    }
    vectorPipelineTesting.value = true
    vectorPipelineTestResult.value = null
    try {
      const result = await testVectorPipeline({
        query: vectorPipelineTestForm.query.trim(),
        task_type: vectorPipelineTestForm.task_type || 'qa',
        top_k: Number(vectorPipelineTestForm.top_k || 5),
        min_similarity: Number(vectorPipelineTestForm.min_similarity || 0.92)
      })
      vectorPipelineTestResult.value = result
      handleSuccess('向量 Pipeline 测试完成')
    } catch (e) {
      handleApiError(e, '执行向量 Pipeline 测试失败')
    } finally {
      vectorPipelineTesting.value = false
    }
  }

  async function loadVectorStatsData() {
    vectorRefreshing.value = true
    try {
      const stats: any = await getVectorStats()
      vectorStats.enabled = Boolean(stats?.enabled)
      vectorStats.index_name = stats?.index_name || ''
      vectorStats.key_prefix = stats?.key_prefix || ''
      vectorStats.dimension = Number(stats?.dimension || 0)
      vectorStats.query_timeout_ms = Number(stats?.query_timeout_ms || 0)
      vectorStats.message = stats?.message || ''
    } catch (e) {
      vectorStats.enabled = false
      vectorStats.message = '获取失败'
      console.warn('Failed to load vector stats:', e)
    } finally {
      vectorRefreshing.value = false
    }
  }

  async function loadVectorTierConfigData() {
    vectorRefreshing.value = true
    try {
      const configData: any = await getVectorTierConfig()
      vectorTierConfig.cold_vector_enabled = Boolean(configData?.cold_vector_enabled)
      vectorTierConfig.cold_vector_query_enabled = Boolean(configData?.cold_vector_query_enabled ?? true)
      vectorTierConfig.cold_vector_backend = configData?.cold_vector_backend || 'sqlite'
      vectorTierConfig.cold_vector_dual_write_enabled = Boolean(configData?.cold_vector_dual_write_enabled)
      vectorTierConfig.cold_vector_similarity_threshold = Number(configData?.cold_vector_similarity_threshold || 0.92)
      vectorTierConfig.cold_vector_top_k = Number(configData?.cold_vector_top_k || 1)
      vectorTierConfig.hot_memory_high_watermark_percent = Number(configData?.hot_memory_high_watermark_percent || 75)
      vectorTierConfig.hot_memory_relief_percent = Number(configData?.hot_memory_relief_percent || 65)
      vectorTierConfig.hot_to_cold_batch_size = Number(configData?.hot_to_cold_batch_size || 500)
      vectorTierConfig.hot_to_cold_interval_seconds = Number(configData?.hot_to_cold_interval_seconds || 30)
      vectorTierConfig.cold_vector_qdrant_url = String(configData?.cold_vector_qdrant_url || '')
      vectorTierConfig.cold_vector_qdrant_api_key = String(configData?.cold_vector_qdrant_api_key || '')
      vectorTierConfig.cold_vector_qdrant_collection = String(configData?.cold_vector_qdrant_collection || '')
      vectorTierConfig.cold_vector_qdrant_timeout_ms = Number(configData?.cold_vector_qdrant_timeout_ms || 1500)
    } catch (e) {
      console.warn('Failed to load vector tier config:', e)
    } finally {
      vectorRefreshing.value = false
    }
  }

  async function saveVectorTierConfigPatch(patch: Record<string, unknown>) {
    vectorTierConfigSaving.value = true
    try {
      const configData: any = await updateVectorTierConfig(patch)
      if (configData && typeof configData === 'object') {
        vectorTierConfig.cold_vector_enabled = Boolean(configData.cold_vector_enabled)
        vectorTierConfig.cold_vector_query_enabled = Boolean(configData.cold_vector_query_enabled ?? true)
        vectorTierConfig.cold_vector_backend = configData.cold_vector_backend || 'sqlite'
        vectorTierConfig.cold_vector_dual_write_enabled = Boolean(configData.cold_vector_dual_write_enabled)
        vectorTierConfig.cold_vector_similarity_threshold = Number(configData.cold_vector_similarity_threshold || 0.92)
        vectorTierConfig.cold_vector_top_k = Number(configData.cold_vector_top_k || 1)
        vectorTierConfig.hot_memory_high_watermark_percent = Number(configData.hot_memory_high_watermark_percent || 75)
        vectorTierConfig.hot_memory_relief_percent = Number(configData.hot_memory_relief_percent || 65)
        vectorTierConfig.hot_to_cold_batch_size = Number(configData.hot_to_cold_batch_size || 500)
        vectorTierConfig.hot_to_cold_interval_seconds = Number(configData.hot_to_cold_interval_seconds || 30)
        vectorTierConfig.cold_vector_qdrant_url = String(configData.cold_vector_qdrant_url || '')
        vectorTierConfig.cold_vector_qdrant_api_key = String(configData.cold_vector_qdrant_api_key || '')
        vectorTierConfig.cold_vector_qdrant_collection = String(configData.cold_vector_qdrant_collection || '')
        vectorTierConfig.cold_vector_qdrant_timeout_ms = Number(configData.cold_vector_qdrant_timeout_ms || 1500)
      }
      await loadVectorTierStatsData()
    } catch (e) {
      handleApiError(e, '更新冷热分层配置失败')
      await loadVectorTierConfigData()
    } finally {
      vectorTierConfigSaving.value = false
    }
  }

  async function loadVectorTierStatsData() {
    vectorRefreshing.value = true
    try {
      const stats: any = await getVectorTierStats()
      vectorTierStats.enabled = Boolean(stats?.enabled)
      vectorTierStats.cold_vector_enabled = Boolean(stats?.cold_vector_enabled)
      vectorTierStats.cold_vector_query_enabled = Boolean(stats?.cold_vector_query_enabled)
      vectorTierStats.cold_vector_backend = stats?.cold_vector_backend || 'sqlite'
      vectorTierStats.cold_vector_dual_write_enabled = Boolean(stats?.cold_vector_dual_write_enabled)
      vectorTierStats.hot_memory_usage_percent = Number(stats?.hot_memory_usage_percent || 0)
      vectorTierStats.hot_memory_high_watermark_percent = Number(stats?.hot_memory_high_watermark_percent || 75)
      vectorTierStats.hot_memory_relief_percent = Number(stats?.hot_memory_relief_percent || 65)
      vectorTierStats.migration_runs = Number(stats?.migration_runs || 0)
      vectorTierStats.migration_moved = Number(stats?.migration_moved || 0)
      vectorTierStats.migration_failed = Number(stats?.migration_failed || 0)
      vectorTierStats.promote_success = Number(stats?.promote_success || 0)
      vectorTierStats.promote_failed = Number(stats?.promote_failed || 0)
      vectorTierStats.message = stats?.message || ''
    } catch (e) {
      vectorTierStats.message = '获取分层状态失败'
      console.warn('Failed to load vector tier stats:', e)
    } finally {
      vectorRefreshing.value = false
    }
  }

  async function migrateHotToCold() {
    tierMigrating.value = true
    try {
      await triggerVectorTierMigrate()
      handleSuccess('冷热迁移任务执行完成')
      await loadVectorTierStatsData()
    } catch (e) {
      handleApiError(e, '执行冷热迁移失败')
    } finally {
      tierMigrating.value = false
    }
  }

  async function promoteToHotTier() {
    const cacheKey = promoteCacheKey.value.trim()
    if (!cacheKey) {
      handleApiError(new Error('请输入 cache_key'), '执行手动回暖失败')
      return
    }
    tierPromoting.value = true
    try {
      await promoteVectorTierEntry(cacheKey)
      handleSuccess('回暖完成')
      promoteCacheKey.value = ''
      await loadVectorTierStatsData()
    } catch (e) {
      handleApiError(e, '执行手动回暖失败')
    } finally {
      tierPromoting.value = false
    }
  }

  async function rebuildVectorCacheIndex() {
    vectorRebuilding.value = true
    try {
      await rebuildVectorIndexApi()
      handleSuccess('向量索引重建已完成')
      await Promise.all([loadVectorStatsData(), loadVectorTierStatsData()])
    } catch (e) {
      handleApiError(e, '重建向量索引失败')
    } finally {
      vectorRebuilding.value = false
    }
  }

  function markPanelLoading(key: PanelKey) {
    panelState[key] = 'loading'
    panelError[key] = ''
  }

  async function reloadPolicyPanel() {
    markPanelLoading('policy')
    try {
      await loadRoutingUiSettings()
      await Promise.all([
        loadConfig(),
        loadAvailableModels(),
        loadCascadeRules(),
        loadFeedbackStats(),
        loadTaskTypeDistribution(),
        loadClassifierHealth(),
        loadClassifierStats(),
        loadClassifierModels()
      ])
      const hasData = availableModels.value.length > 0 || taskTypes.value.some(t => t.count > 0)
      panelState.policy = hasData ? 'success' : 'empty'
    } catch (e) {
      panelError.policy = '路由策略加载失败'
      panelState.policy = 'error'
    }
  }

  async function reloadOllamaPanel() {
    markPanelLoading('ollama')
    try {
      await loadOllamaSetupStatus()
      panelState.ollama = 'success'
    } catch (e) {
      panelError.ollama = 'Ollama 状态加载失败'
      panelState.ollama = 'error'
    }
  }

  async function reloadModelsPanel() {
    markPanelLoading('models')
    try {
      await Promise.all([
        loadModelScores(),
        loadIntentEngineConfigData(),
        loadIntentEngineHealthData()
      ])
      panelState.models = modelScores.value.length > 0 ? 'success' : 'empty'
    } catch (e) {
      panelError.models = '模型数据加载失败'
      panelState.models = 'error'
    }
  }

  async function reloadVectorPanel() {
    markPanelLoading('vector')
    try {
      await Promise.all([
        loadVectorPipelineConfigData(),
        loadVectorPipelineHealthData(),
        loadVectorStatsData(),
        loadVectorTierConfigData(),
        loadVectorTierStatsData()
      ])
      panelState.vector = vectorStats.enabled || vectorTierStats.enabled || vectorPipelineHealth.enabled ? 'success' : 'empty'
    } catch (e) {
      panelError.vector = '向量状态加载失败'
      panelState.vector = 'error'
    }
  }

  async function reloadAllPanels() {
    await Promise.all([
      reloadPolicyPanel(),
      reloadOllamaPanel(),
      reloadModelsPanel(),
      reloadVectorPanel()
    ])
  }

  onMounted(async () => {
    switchPollingCancelled.value = false
    await reloadAllPanels()
    if (!ollamaStatusPollTimer) {
      ollamaStatusPollTimer = window.setInterval(loadOllamaSetupStatus, ollamaStatusPollIntervalMs)
    }
  })

  onUnmounted(() => {
    switchPollingCancelled.value = true
    if (autoSaveTimer) {
      window.clearTimeout(autoSaveTimer)
    }
    if (ollamaStatusPollTimer) {
      window.clearInterval(ollamaStatusPollTimer)
      ollamaStatusPollTimer = null
    }
  })

  let autoSaveTimer: number | null = null
  let ollamaStatusPollTimer: number | null = null
  const autoSaveDelayMs = 800

  function scheduleAutoSave() {
    if (!autoSaveEnabled.value || !isMappingReady.value) return
    if (autoSaveTimer) {
      window.clearTimeout(autoSaveTimer)
    }
    autoSaveTimer = window.setTimeout(() => {
      saveTaskMapping(true)
    }, autoSaveDelayMs)
  }

  watch(
    () => taskModelMapping,
    () => {
      scheduleAutoSave()
    },
    { deep: true }
  )

  watch(autoSaveEnabled, (value) => {
    updateRoutingUiSettings({
      auto_save_enabled: value,
      last_saved_at: lastSavedAt.value || ''
    }).catch((e) => {
      console.warn('Failed to persist routing auto save settings:', e)
    })
    if (value) {
      scheduleAutoSave()
    }
  })

  return {
    activeTab,
    panelState,
    panelError,
    saving,
    modelSearch,
    modelScores,
    availableModels,
    autoSaveEnabled,
    lastSavedAt,
    classifierSaving,
    intentEngineSaving,
    classifierSwitching,
    classifierModelsLoading,
    classifierSwitchModel,
    ollamaInstalling,
    ollamaStarting,
    ollamaStopping,
    ollamaPulling,
    ollamaRefreshing,
    ollamaModelInput,
    vectorRefreshing,
    vectorRebuilding,
    vectorTierConfigSaving,
    tierMigrating,
    tierPromoting,
    promoteCacheKey,
    vectorPipelineSaving,
    vectorPipelineTesting,
    classifierConfig,
    classifierHealth,
    classifierStats,
    intentEngineConfig,
    intentEngineHealth,
    vectorStats,
    vectorTierConfig,
    vectorTierStats,
    vectorPipelineConfig,
    vectorPipelineHealth,
    vectorPipelineTestForm,
    vectorPipelineTestResult,
    ollamaSetup,
    config,
    taskModelMapping,
    strategies,
    feedbackStats,
    cascadeRules,
    cascadeLevels,
    taskTypes,
    statsCards,
    filteredModels,
    modeLabel,
    strategyLabel,
    lastSavedLabel,
    classifierConfidencePercent,
    calculateCompositeScore,
    getScoreColor,
    getScoreTagType,
    formatVramBytes,
    formatDuration,
    loadClassifierHealth,
    loadClassifierModels,
    loadClassifierStats,
    loadIntentEngineHealthData,
    saveIntentEngineConfigData,
    loadOllamaSetupStatus,
    installOllama,
    startOllama,
    stopOllama,
    pullOllamaModel,
    saveClassifierConfig,
    switchClassifierModel,
    saveTaskMapping,
    toggleModelEnabled,
    triggerOptimization,
    reloadPolicyPanel,
    reloadOllamaPanel,
    reloadModelsPanel,
    reloadVectorPanel,
    reloadAllPanels,
    rebuildVectorCacheIndex,
    saveVectorTierConfigPatch,
    migrateHotToCold,
    promoteToHotTier,
    saveVectorPipelineConfigData,
    runVectorPipelineTest
  }
}
