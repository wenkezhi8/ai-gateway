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
              <!-- FIX: 保存任务映射 -->
              <el-button type="primary" size="small" @click="saveTaskMapping" :loading="saving">
                <el-icon><Check /></el-icon>
                保存映射
              </el-button>
            </div>
          </template>

          <el-form label-width="120px">
            <!-- FIX: 基础路由配置仅展示，避免与 API 管理页重复 -->
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="当前路由模式">
                  <el-tag size="small" type="info">{{ modeLabel }}</el-tag>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="默认策略">
                  <el-tag size="small" type="info">{{ strategyLabel }}</el-tag>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="默认模型">
                  <el-tag size="small" type="info">{{ config.defaultModel }}</el-tag>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="基础配置">
                  <!-- FIX: 跳转到 API 管理页面配置基础路由 -->
                  <el-button type="primary" link @click="$router.push('/api-management')">
                    前往 API 管理设置
                  </el-button>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="自动保存">
                  <el-switch v-model="autoSaveEnabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="最近保存">
                  <span class="last-saved">{{ lastSavedLabel }}</span>
                </el-form-item>
              </el-col>
            </el-row>

            <el-divider content-position="left">0.5B 分类控制器</el-divider>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="启用分类器">
                  <el-switch v-model="classifierConfig.enabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Shadow模式">
                  <el-switch v-model="classifierConfig.shadow_mode" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-divider content-position="left">控制面开关</el-divider>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="控制层总开关">
                  <el-switch v-model="classifierConfig.control.enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="控制层Shadow">
                  <el-switch v-model="classifierConfig.control.shadow_only" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="归一化查询读">
                  <el-switch v-model="classifierConfig.control.normalized_query_read_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="缓存写门禁">
                  <el-switch v-model="classifierConfig.control.cache_write_gate_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="风险打标">
                  <el-switch v-model="classifierConfig.control.risk_tag_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="风险拦截">
                  <el-switch v-model="classifierConfig.control.risk_block_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="工具门控">
                  <el-switch v-model="classifierConfig.control.tool_gate_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="Model Fit 选模">
                  <el-switch v-model="classifierConfig.control.model_fit_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="参数建议">
                  <el-switch v-model="classifierConfig.control.parameter_hint_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="运行模型">
                  <el-tag size="small" :type="classifierHealth.healthy ? 'success' : 'warning'">
                    {{ classifierConfig.active_model || '-' }}
                  </el-tag>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="超时(ms)">
                  <el-input-number v-model="classifierConfig.timeout_ms" :min="50" :max="10000" :step="10" style="width: 180px" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="置信度阈值">
                  <el-slider v-model="classifierConfidencePercent" :min="30" :max="95" :step="1" show-input style="max-width: 320px" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="14">
                <el-form-item label="手动切换模型">
                  <el-select v-model="classifierSwitchModel" filterable clearable style="width: 100%" placeholder="选择分类模型">
                    <el-option v-for="model in classifierConfig.candidate_models" :key="model" :label="model" :value="model" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="10">
                <el-form-item label="操作">
                  <el-button type="primary" :loading="classifierSaving" @click="saveClassifierConfig">保存配置</el-button>
                  <el-button :loading="classifierSwitching" @click="switchClassifierModel">切换模型</el-button>
                  <el-button :loading="classifierModelsLoading" @click="loadClassifierModels">刷新模型列表</el-button>
                  <el-button link @click="loadClassifierHealth">健康检查</el-button>
                </el-form-item>
              </el-col>
            </el-row>
            <el-alert
              :title="`健康状态: ${classifierHealth.message || 'unknown'} (延迟 ${formatDuration(classifierHealth.latency_ms)})`"
              :type="classifierHealth.healthy ? 'success' : 'warning'"
              :closable="false"
              style="margin-bottom: 16px"
            />
            <el-divider content-position="left">Ollama 一键安装与模型安装</el-divider>
            <el-row :gutter="12" style="margin-bottom: 12px">
              <el-col :span="24">
                <el-tag :type="ollamaSetup.installed ? 'success' : 'warning'" style="margin-right: 8px">
                  Ollama安装: {{ ollamaSetup.installed ? '已安装' : '未安装' }}
                </el-tag>
                <el-tag :type="ollamaSetup.running ? 'success' : 'danger'" style="margin-right: 8px">
                  服务状态: {{ ollamaSetup.running ? '运行中' : '未运行' }}
                </el-tag>
                <el-tag :type="ollamaSetup.model_installed ? 'success' : 'info'">
                  模型({{ ollamaSetup.model }}): {{ ollamaSetup.model_installed ? '已安装' : '未安装' }}
                </el-tag>
                <el-tag :type="ollamaSetup.running_model ? 'success' : 'warning'" style="margin-left: 8px">
                  当前运行模型: {{ ollamaSetup.running_model || '无' }}
                </el-tag>
              </el-col>
            </el-row>
            <el-row :gutter="12" style="margin-bottom: 12px">
              <el-col :span="12">
                <el-input v-model="ollamaModelInput" placeholder="模型名，如 qwen2.5:0.5b-instruct" />
              </el-col>
              <el-col :span="12">
                <el-button :loading="ollamaInstalling" @click="installOllama">安装 Ollama</el-button>
                <el-button :loading="ollamaStarting" type="warning" @click="startOllama">启动 Ollama</el-button>
                <el-button :loading="ollamaStopping" type="danger" @click="stopOllama">停止 Ollama</el-button>
                <el-button :loading="ollamaPulling" type="primary" @click="pullOllamaModel">安装模型</el-button>
                <el-button :loading="ollamaRefreshing" link @click="loadOllamaSetupStatus">刷新状态</el-button>
              </el-col>
            </el-row>
            <el-alert
              v-if="ollamaSetup.message"
              :title="`Ollama状态: ${ollamaSetup.message}`"
              :type="ollamaSetup.running ? 'success' : 'warning'"
              :closable="false"
              style="margin-bottom: 16px"
            />
            <el-alert
              v-if="ollamaSetup.keep_alive_disabled"
              title="已禁用模型自动休眠（keep_alive=-1）"
              type="success"
              :closable="false"
              style="margin-bottom: 16px"
            />
            <el-alert
              v-if="ollamaSetup.running_models.length > 0"
              :title="`运行模型列表: ${ollamaSetup.running_models.join(', ')}`"
              type="info"
              :closable="false"
              style="margin-bottom: 16px"
            />
            <el-alert
              v-if="ollamaSetup.running_vram_bytes_total > 0"
              :title="`显存占用: ${formatVramBytes(ollamaSetup.running_vram_bytes_total)}`"
              type="warning"
              :closable="false"
              style="margin-bottom: 16px"
            />
            <el-descriptions :column="2" border size="small" style="margin-bottom: 16px">
              <el-descriptions-item label="总请求">{{ classifierStats.total_requests }}</el-descriptions-item>
              <el-descriptions-item label="LLM尝试">{{ classifierStats.llm_attempts }}</el-descriptions-item>
              <el-descriptions-item label="LLM成功">{{ classifierStats.llm_success }}</el-descriptions-item>
              <el-descriptions-item label="回退次数">{{ classifierStats.fallbacks }}</el-descriptions-item>
              <el-descriptions-item label="Shadow请求">{{ classifierStats.shadow_requests }}</el-descriptions-item>
              <el-descriptions-item label="平均延迟">{{ formatDuration(classifierStats.avg_llm_latency_ms) }}</el-descriptions-item>
              <el-descriptions-item label="控制层延迟">{{ formatDuration(classifierStats.avg_control_latency_ms) }}</el-descriptions-item>
              <el-descriptions-item label="解析错误">{{ classifierStats.parse_errors }}</el-descriptions-item>
              <el-descriptions-item label="控制字段缺失">{{ classifierStats.control_fields_missing }}</el-descriptions-item>
            </el-descriptions>

            <!-- 任务类型模型映射 -->
            <el-divider content-position="left">任务类型模型映射</el-divider>
            <el-alert type="info" :closable="false" style="margin-bottom: 16px">
              <template #title>
                开启后将根据任务类型自动选择对应模型，关闭则使用默认策略
              </template>
            </el-alert>
            <el-row :gutter="16">
              <el-col :span="12" v-for="task in taskTypes" :key="task.type">
                <div class="task-model-item">
                  <div class="task-header">
                    <el-switch v-model="taskModelMapping[task.type]!.enabled" size="small" />
                    <span class="task-name">{{ task.name }}</span>
                  </div>
                  <el-select 
                    v-model="taskModelMapping[task.type]!.model" 
                    :disabled="!taskModelMapping[task.type]?.enabled"
                    filterable
                    size="small"
                    style="width: 100%"
                    placeholder="选择模型"
                  >
                    <el-option
                      v-for="m in availableModels"
                      :key="m.id"
                      :label="m.display_name || m.id"
                      :value="m.id"
                    />
                  </el-select>
                </div>
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
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import { request } from '@/api/request'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'
import { formatDuration } from '@/utils/format-duration'
import {
  DEFAULT_CLASSIFIER_CONFIG,
  ROUTING_OLLAMA_DEFAULT_MODEL,
  createDefaultTaskModelMapping,
  createDefaultTaskTypes,
} from '@/constants/routing'

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

const saving = ref(false)
const modelSearch = ref('')
const modelScores = ref<ModelScore[]>([])
interface ModelOption {
  id: string
  display_name?: string
}
const availableModels = ref<ModelOption[]>([])
const autoSaveEnabled = ref(false) // FIX: 自动保存开关
const lastSavedAt = ref<string | null>(null) // FIX: 最近保存时间
const isMappingReady = ref(false) // FIX: 防止初始化阶段触发自动保存
const classifierSaving = ref(false)
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

const classifierSwitchPollIntervalMs = 2000
const classifierSwitchLoadingMessage = '正在加载模型，首次可能较慢（最多180秒）'
const classifierSwitchTimeoutMessage = '模型加载超时，请继续等待Ollama完成加载后重试'

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

const config = reactive({
  // FIX: 使用字符串模式用于只读展示
  mode: 'auto',
  defaultStrategy: 'auto',
  defaultModel: 'deepseek-chat'
})

// 任务类型模型映射
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

const taskTypes = ref(createDefaultTaskTypes())

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

// FIX: 展示当前路由模式与策略标签
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
    const data: any = await request.get('/admin/router/config')
    if (data?.data) {
      config.defaultStrategy = data.data.default_strategy || 'auto'
      config.defaultModel = data.data.default_model || 'deepseek-chat'
      const mode = data.data.use_auto_mode
      if (typeof mode === 'string') {
        config.mode = mode
      } else {
        config.mode = mode ? 'auto' : 'fixed'
      }
      if (data.data.strategies) {
        strategies.value = data.data.strategies
      }
      if (data.data.classifier) {
        Object.assign(classifierConfig, data.data.classifier)
        ensureControlConfig()
        classifierSwitchModel.value = classifierConfig.active_model
      }
    }
    
    // 加载任务类型模型映射
    try {
      const mappingData: any = await request.get('/admin/router/task-model-mapping')
      if (mappingData?.data) {
        for (const [taskType, model] of Object.entries(mappingData.data)) {
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
    const data: any = await request.get('/admin/router/models')
    if (data) {
      const scores = data.data || data
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
    const data: any = await request.get('/admin/router/available-models?format=object')
    if (data?.data) {
      availableModels.value = data.data
    }
  } catch (e) {
    console.warn('Failed to load available models:', e)
  }
}

async function loadCascadeRules() {
  try {
    const data: any = await request.get('/admin/router/cascade-rules')
    cascadeRules.value = Array.isArray(data?.data) ? data.data : []
  } catch (e) {
    console.warn('Failed to load cascade rules:', e)
  }
}

async function loadFeedbackStats() {
  try {
    const data: any = await request.get('/admin/feedback/stats')
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

async function loadClassifierHealth() {
  try {
    const data: any = await request.get('/admin/router/classifier/health')
    const health = data?.data || data
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
    const data: any = await request.get('/admin/router/classifier/models')
    const payload = data?.data || {}
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
    const data: any = await request.get('/admin/router/classifier/stats')
    const stats = data?.data || {}
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

async function loadOllamaSetupStatus() {
  ollamaRefreshing.value = true
  try {
    const model = (ollamaModelInput.value || classifierConfig.active_model || ROUTING_OLLAMA_DEFAULT_MODEL).trim()
    const data: any = await request.get(`/admin/router/ollama/status?model=${encodeURIComponent(model)}`)
    const payload = data?.data || data || {}
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
    await request.post('/admin/router/ollama/install')
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
    await request.post('/admin/router/ollama/start')
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
    await request.post('/admin/router/ollama/stop')
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
    await request.post('/admin/router/ollama/pull', { model })
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
    await request.put('/admin/router/config', {
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
    const switchResp: any = await request.post('/admin/router/classifier/switch-async', {
      model: classifierSwitchModel.value
    })
    const taskId = switchResp?.data?.task_id || switchResp?.task_id
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
    const taskResp: any = await request.get(taskPath)
    const taskData = taskResp?.data || taskResp
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
    // FIX: 仅保存任务映射，基础配置在 API 管理页设置
    const mappingData: Record<string, string> = {}
    for (const [taskType, mapping] of Object.entries(taskModelMapping)) {
      if (mapping.enabled && mapping.model) {
        mappingData[taskType] = mapping.model
      }
    }
    await request.put('/admin/router/task-model-mapping', mappingData)
    const savedAt = new Date().toISOString()
    lastSavedAt.value = savedAt
    localStorage.setItem('routing_task_mapping_last_saved', savedAt)
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
    await request.put(`/admin/router/models/${model.model}`, {
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
    const resp: any = await request.post('/admin/feedback/optimize')
    const result = resp?.data || {}
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
    const data: any = await request.get('/admin/feedback/task-type-distribution')
    if (data?.distribution && data.distribution.length > 0) {
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

onMounted(() => {
  switchPollingCancelled.value = false
  const storedAutoSave = localStorage.getItem('routing_task_mapping_auto_save')
  autoSaveEnabled.value = storedAutoSave === '1'
  lastSavedAt.value = localStorage.getItem('routing_task_mapping_last_saved')
  loadConfig()
  loadModelScores()
  loadAvailableModels()
  loadCascadeRules()
  loadFeedbackStats()
  loadTaskTypeDistribution()
  loadClassifierHealth()
  loadClassifierStats()
  loadClassifierModels()
  loadOllamaSetupStatus()
})

onUnmounted(() => {
  switchPollingCancelled.value = true
  if (autoSaveTimer) {
    window.clearTimeout(autoSaveTimer)
  }
})

let autoSaveTimer: number | null = null
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
  localStorage.setItem('routing_task_mapping_auto_save', value ? '1' : '0')
  if (value) {
    scheduleAutoSave()
  }
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

  .task-model-item {
    margin-bottom: 12px;
    padding: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: var(--el-fill-color-light);

    .task-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 8px;

      .task-name {
        font-size: 13px;
        font-weight: 500;
        color: var(--el-text-color-primary);
      }
    }
  }

  .last-saved {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }
}
</style>
