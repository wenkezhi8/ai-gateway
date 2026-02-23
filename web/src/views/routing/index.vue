<template>
  <div class="routing-page">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6" v-for="stat in statsCards" :key="stat.title">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" :style="{ background: stat.color + '15' }">
              <el-icon :size="28" :color="stat.color"><component :is="stat.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stat.value }}</div>
              <div class="stat-title">{{ stat.title }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="24">
      <!-- 左侧：智能路由配置 + 模型评分 -->
      <el-col :span="16">
        <!-- 智能路由配置 -->
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>智能路由配置</span>
              <el-button type="primary" size="small" @click="saveConfig" :loading="saving">
                <el-icon><Check /></el-icon>
                保存配置
              </el-button>
            </div>
          </template>

          <el-form label-width="120px">
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="路由模式">
                  <el-switch v-model="config.useAutoMode" active-text="自动" inactive-text="手动" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="默认策略">
                  <el-select v-model="config.defaultStrategy" style="width: 100%">
                    <el-option
                      v-for="s in strategies"
                      :key="s.value"
                      :label="s.label"
                      :value="s.value"
                    >
                      <div class="strategy-option">
                        <span>{{ s.label }}</span>
                        <span class="strategy-desc">{{ s.description }}</span>
                      </div>
                    </el-option>
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="默认模型">
                  <el-select v-model="config.defaultModel" filterable style="width: 100%">
                    <el-option
                      v-for="m in availableModels"
                      :key="m"
                      :label="m"
                      :value="m"
                    />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </el-card>

        <!-- 模型评分 -->
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>模型评分管理</span>
              <el-input
                v-model="modelSearch"
                placeholder="搜索模型..."
                style="width: 200px"
                clearable
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
              </el-input>
            </div>
          </template>

          <el-table :data="filteredModels" stripe max-height="400">
            <el-table-column prop="model" label="模型" width="180" fixed />
            <el-table-column prop="provider" label="服务商" width="100" />
            <el-table-column label="效果" width="120">
              <template #default="{ row }">
                <div class="score-cell">
                  <el-progress
                    :percentage="row.quality_score"
                    :color="getScoreColor(row.quality_score)"
                    :stroke-width="8"
                    :show-text="false"
                  />
                  <span class="score-text">{{ row.quality_score }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="速度" width="120">
              <template #default="{ row }">
                <div class="score-cell">
                  <el-progress
                    :percentage="row.speed_score"
                    :color="getScoreColor(row.speed_score)"
                    :stroke-width="8"
                    :show-text="false"
                  />
                  <span class="score-text">{{ row.speed_score }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="成本" width="120">
              <template #default="{ row }">
                <div class="score-cell">
                  <el-progress
                    :percentage="row.cost_score"
                    :color="getScoreColor(row.cost_score)"
                    :stroke-width="8"
                    :show-text="false"
                  />
                  <span class="score-text">{{ row.cost_score }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="综合" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="getScoreTagType(calculateCompositeScore(row))" size="small">
                  {{ calculateCompositeScore(row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="toggleModelEnabled(row)" />
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <!-- 右侧：级联路由 + 任务类型 + 反馈统计 -->
      <el-col :span="8">
        <!-- 级联路由策略 -->
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>级联路由策略</span>
              <el-tag size="small" type="success">自动升级</el-tag>
            </div>
          </template>

          <div class="cascade-levels">
            <div v-for="level in cascadeLevels" :key="level.key" class="cascade-level">
              <div class="level-header">
                <el-tag :type="level.type" size="small">{{ level.label }}</el-tag>
                <span class="level-desc">{{ level.desc }}</span>
              </div>
              <div class="level-models">
                <el-tag
                  v-for="model in level.models"
                  :key="model"
                  size="small"
                  class="model-tag"
                >
                  {{ model }}
                </el-tag>
              </div>
            </div>
          </div>

          <el-alert type="info" :closable="false" show-icon style="margin-top: 16px">
            <template #title>
              当小模型无法处理时，自动升级到大模型
            </template>
          </el-alert>
        </el-card>

        <!-- 任务类型分布 -->
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>任务类型分布</span>
            </div>
          </template>

          <div class="task-types">
            <div v-for="task in taskTypes" :key="task.type" class="task-type-item">
              <div class="task-header">
                <span class="task-name">{{ task.name }}</span>
                <span class="task-percent">{{ task.percentage }}%</span>
              </div>
              <el-progress
                :percentage="task.percentage"
                :color="task.color"
                :stroke-width="8"
                :show-text="false"
              />
            </div>
          </div>
        </el-card>

        <!-- 反馈统计 -->
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>效果评估</span>
              <el-button type="primary" link size="small" @click="loadFeedbackStats">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>

          <div class="feedback-stats">
            <div class="feedback-item">
              <span class="label">总反馈数</span>
              <span class="value">{{ feedbackStats.total }}</span>
            </div>
            <div class="feedback-item">
              <span class="label">好评率</span>
              <span class="value positive">{{ feedbackStats.positiveRate }}%</span>
            </div>
            <div class="feedback-item">
              <span class="label">追踪模型数</span>
              <span class="value">{{ feedbackStats.modelsTracked }}</span>
            </div>
            <div class="feedback-item">
              <span class="label">平均评分</span>
              <span class="value">{{ feedbackStats.avgRating.toFixed(1) }}</span>
            </div>
          </div>

          <el-button type="primary" style="width: 100%; margin-top: 16px" @click="triggerOptimization">
            <el-icon><MagicStick /></el-icon>
            触发自动优化
          </el-button>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessageBox } from 'element-plus'
import { request } from '@/api/request'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'

interface ModelScore {
  model: string
  provider: string
  quality_score: number
  speed_score: number
  cost_score: number
  enabled: boolean
}

const saving = ref(false)
const modelSearch = ref('')
const modelScores = ref<ModelScore[]>([])
const availableModels = ref<string[]>([])

const config = reactive({
  defaultStrategy: 'auto',
  defaultModel: 'deepseek-chat',
  useAutoMode: true
})

const strategies = ref([
  { value: 'auto', label: '智能平衡', description: '综合效果 + 速度 + 成本' },
  { value: 'quality', label: '效果优先', description: '选择效果最好的模型' },
  { value: 'speed', label: '速度优先', description: '选择响应最快的模型' },
  { value: 'cost', label: '成本优先', description: '选择成本最低的模型' }
])

const feedbackStats = reactive({
  total: 0,
  positive: 0,
  positiveRate: 0,
  avgRating: 0,
  modelsTracked: 0
})

const ttlConfig = reactive({
  taskTypeDefaults: {} as Record<string, number>,
  difficultyMultipliers: { low: 0.5, medium: 1.0, high: 2.0 }
})

const cascadeLevels = [
  { key: 'small', label: '小型', type: 'success', desc: '快速响应，低成本', models: ['gpt-4o-mini', 'deepseek-chat', 'glm-4-flash', 'qwen-turbo'] },
  { key: 'medium', label: '中型', type: 'warning', desc: '平衡质量与速度', models: ['gpt-4o', 'deepseek-coder', 'claude-3-5-haiku', 'qwen-plus'] },
  { key: 'large', label: '大型', type: 'danger', desc: '最高质量，复杂任务', models: ['deepseek-reasoner', 'o1', 'claude-3-5-sonnet', 'gpt-4-turbo'] }
]

const taskTypes = ref([
  { type: 'code', name: '代码生成', count: 0, percentage: 0, color: '#007AFF' },
  { type: 'chat', name: '日常对话', count: 0, percentage: 0, color: '#34C759' },
  { type: 'reasoning', name: '逻辑推理', count: 0, percentage: 0, color: '#FF9500' },
  { type: 'math', name: '数学计算', count: 0, percentage: 0, color: '#FF3B30' },
  { type: 'fact', name: '事实查询', count: 0, percentage: 0, color: '#34C759' },
  { type: 'creative', name: '创意写作', count: 0, percentage: 0, color: '#AF52DE' },
  { type: 'translate', name: '翻译', count: 0, percentage: 0, color: '#5856D6' },
  { type: 'other', name: '其他', count: 0, percentage: 0, color: '#8E8E93' }
])

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

async function loadConfig() {
  try {
    const data: any = await request.get('/api/admin/router/config')
    if (data?.data) {
      config.defaultStrategy = data.data.default_strategy || 'auto'
      config.defaultModel = data.data.default_model || 'deepseek-chat'
      config.useAutoMode = data.data.use_auto_mode ?? true
      if (data.data.strategies) {
        strategies.value = data.data.strategies
      }
    }
  } catch (e) {
    console.warn('Failed to load config:', e)
  }
}

async function loadModelScores() {
  try {
    const data: any = await request.get('/api/admin/router/models')
    if (data) {
      const scores = data.data || data
      modelScores.value = Object.entries(scores).map(([model, score]) => ({
        model,
        provider: (score as any).provider || 'unknown',
        quality_score: (score as any).quality_score || 80,
        speed_score: (score as any).speed_score || 80,
        cost_score: (score as any).cost_score || 80,
        enabled: (score as any).enabled ?? true
      }))
      availableModels.value = modelScores.value.map(m => m.model)
    }
  } catch (e) {
    console.warn('Failed to load model scores:', e)
  }
}

async function loadAvailableModels() {
  try {
    const data: any = await request.get('/api/admin/router/available-models')
    if (data?.data) {
      availableModels.value = data.data
    }
  } catch (e) {
    console.warn('Failed to load available models:', e)
  }
}

async function loadFeedbackStats() {
  try {
    const data: any = await request.get('/api/admin/feedback/stats')
    if (data) {
      const stats = data.data || data
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

async function saveConfig() {
  saving.value = true
  try {
    await request.put('/api/admin/router/config', {
      default_strategy: config.defaultStrategy,
      default_model: config.defaultModel,
      use_auto_mode: config.useAutoMode
    })
    handleSuccess('配置已保存')
  } catch (e) {
    handleApiError(e, '保存失败')
  } finally {
    saving.value = false
  }
}

async function toggleModelEnabled(model: ModelScore) {
  try {
    await request.put(`/api/admin/router/models/${model.model}`, {
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
    await ElMessageBox.confirm('确定要触发自动优化吗？这将根据反馈数据调整模型评分。', '确认', { type: 'info' })
    await request.post('/api/admin/feedback/optimize')
    handleSuccess('优化已完成')
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
    const data: any = await request.get('/api/admin/feedback/task-type-distribution')
    if (data?.distribution) {
      const colorMap: Record<string, string> = {
        code: '#007AFF',
        chat: '#34C759',
        reasoning: '#FF9500',
        math: '#FF3B30',
        fact: '#34C759',
        creative: '#AF52DE',
        translate: '#5856D6',
        other: '#8E8E93'
      }
      const nameMap: Record<string, string> = {
        code: '代码生成',
        chat: '日常对话',
        reasoning: '逻辑推理',
        math: '数学计算',
        fact: '事实查询',
        creative: '创意写作',
        translate: '翻译',
        other: '其他'
      }
      taskTypes.value = data.distribution.map((item: any) => ({
        type: item.task_type,
        name: nameMap[item.task_type] || item.task_type,
        count: item.count,
        percentage: item.percent,
        color: colorMap[item.task_type] || '#8E8E93'
      }))
    }
  } catch (e) {
    console.warn('Failed to load task type distribution:', e)
  }
}

async function loadTTLConfig() {
  try {
    const data: any = await request.get('/api/admin/router/ttl-config')
    if (data?.data) {
      ttlConfig.taskTypeDefaults = data.data.task_type_defaults || {}
      ttlConfig.difficultyMultipliers = data.data.difficulty_multipliers || { low: 0.5, medium: 1.0, high: 2.0 }
    }
  } catch (e) {
    console.warn('Failed to load TTL config:', e)
  }
}

onMounted(() => {
  loadConfig()
  loadModelScores()
  loadAvailableModels()
  loadFeedbackStats()
  loadTaskTypeDistribution()
  loadTTLConfig()
})
</script>

<style scoped lang="scss">
.routing-page {
  .stats-row {
    margin-bottom: 20px;
  }

  .stat-card {
    border-radius: 12px;
    border: none;

    .stat-content {
      display: flex;
      align-items: center;
      gap: 16px;

      .stat-icon {
        width: 56px;
        height: 56px;
        border-radius: 12px;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .stat-info {
        .stat-value {
          font-size: 24px;
          font-weight: 600;
          color: var(--el-text-color-primary);
        }

        .stat-title {
          font-size: 14px;
          color: var(--el-text-color-secondary);
        }
      }
    }
  }

  .page-card {
    border-radius: 12px;
    border: none;
    margin-bottom: 20px;

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-weight: 600;
    }
  }

  .strategy-option {
    display: flex;
    flex-direction: column;
    gap: 4px;

    .strategy-desc {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .score-cell {
    display: flex;
    align-items: center;
    gap: 8px;

    .el-progress {
      flex: 1;
    }

    .score-text {
      width: 24px;
      text-align: right;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .cascade-levels {
    display: flex;
    flex-direction: column;
    gap: 16px;

    .cascade-level {
      .level-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;

        .level-desc {
          font-size: 12px;
          color: var(--el-text-color-secondary);
        }
      }

      .level-models {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;

        .model-tag {
          font-size: 11px;
        }
      }
    }
  }

  .task-types {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .task-type-item {
      .task-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 4px;

        .task-name {
          font-size: 14px;
          color: var(--el-text-color-primary);
        }

        .task-percent {
          font-size: 14px;
          font-weight: 500;
          color: var(--el-text-color-secondary);
        }
      }
    }
  }

  .feedback-stats {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;

    .feedback-item {
      display: flex;
      flex-direction: column;
      gap: 4px;

      .label {
        font-size: 12px;
        color: var(--el-text-color-secondary);
      }

      .value {
        font-size: 20px;
        font-weight: 600;
        color: var(--el-text-color-primary);

        &.positive {
          color: #67c23a;
        }
      }
    }
  }
}
</style>
