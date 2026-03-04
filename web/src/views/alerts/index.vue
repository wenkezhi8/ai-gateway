<template>
  <div class="alerts-page">
    <!-- 告警统计卡片 -->
    <div class="stats-section">
      <div v-if="statsRequest.loading" class="request-state">
        <el-skeleton :rows="1" animated />
      </div>
      <div v-else-if="statsRequest.error" class="request-state">
        <el-empty description="告警统计加载失败">
          <el-button type="primary" size="small" @click="fetchStats">重试</el-button>
        </el-empty>
      </div>
      <div v-else-if="isStatsEmpty" class="request-state">
        <el-empty description="暂无告警统计" />
      </div>
      <el-row v-else :gutter="20" class="stats-row">
        <el-col :span="6" v-for="stat in alertStats" :key="stat.title">
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
    </div>

    <el-row :gutter="24" class="content-row">
      <!-- 告警规则 -->
      <el-col :span="8">
        <el-card shadow="never" class="page-card rules-card">
          <template #header>
            <div class="card-header">
              <span>告警规则</span>
              <el-button type="primary" size="small" @click="showAddRuleDialog">
                <el-icon><Plus /></el-icon>
                添加
              </el-button>
            </div>
          </template>

          <div v-if="rulesRequest.loading" class="request-state">
            <el-skeleton :rows="4" animated />
          </div>
          <div v-else-if="rulesRequest.error" class="request-state">
            <el-empty description="告警规则加载失败">
              <el-button type="primary" size="small" @click="fetchRules">重试</el-button>
            </el-empty>
          </div>
          <div v-else-if="!alertRules.length" class="request-state">
            <el-empty description="暂无告警规则" />
          </div>
          <div v-else class="rules-list">
            <div v-for="rule in alertRules" :key="rule.id" class="rule-item">
              <div class="rule-header">
                <div class="rule-info">
                  <span class="rule-name">{{ rule.name }}</span>
                  <el-tag size="small" :type="getConditionType(rule.condition.type)">
                    {{ rule.condition.typeLabel }}
                  </el-tag>
                </div>
                <el-switch v-model="rule.enabled" size="small" @change="handleRuleChange(rule)" />
              </div>
              <div class="rule-condition">
                <el-icon><Operation /></el-icon>
                <span>{{ rule.condition.text }}</span>
              </div>
              <div class="rule-channels">
                <el-tag v-for="channel in rule.channels" :key="channel" size="small" type="info">
                  {{ channel }}
                </el-tag>
              </div>
              <div class="rule-actions">
                <el-button type="primary" link size="small" @click="editRule(rule)">编辑</el-button>
                <el-button type="danger" link size="small" @click="deleteRule(rule)">删除</el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 告警历史 -->
      <el-col :span="16">
        <el-card shadow="never" class="page-card history-card">
          <template #header>
            <div class="card-header">
              <span>告警历史</span>
              <div class="filter-group">
                <el-select v-model="selectedLevel" placeholder="告警级别" clearable size="small">
                  <el-option v-for="opt in alertLevelOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
                </el-select>
                <el-date-picker
                  v-model="dateRange"
                  type="daterange"
                  range-separator="至"
                  start-placeholder="开始日期"
                  end-placeholder="结束日期"
                  size="small"
                  style="margin-left: 10px"
                />
                <el-button type="danger" plain size="small" style="margin-left: 10px" @click="clearHistory">
                  清空告警历史
                </el-button>
              </div>
            </div>
          </template>

          <div v-if="historyRequest.loading" class="request-state">
            <el-skeleton :rows="6" animated />
          </div>
          <div v-else-if="historyRequest.error" class="request-state">
            <el-empty description="告警历史加载失败">
              <el-button type="primary" size="small" @click="fetchAlerts">重试</el-button>
            </el-empty>
          </div>
          <div v-else-if="!filteredAlerts.length" class="request-state">
            <el-empty description="暂无告警历史" />
          </div>
          <template v-else>
            <el-table :data="filteredAlerts" stripe class="alerts-table">
              <el-table-column prop="time" label="时间" width="160">
                <template #default="{ row }">
                  <span class="time-text">{{ row.time }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="level" label="级别" width="80">
                <template #default="{ row }">
                  <el-tag :type="getAlertTagType(row.level)" size="small" effect="dark">
                    {{ getLevelText(row.level) }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="source" label="来源" width="100">
                <template #default="{ row }">
                  <el-tag size="small">{{ row.source }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="message" label="告警信息" min-width="200" show-overflow-tooltip />
              <el-table-column prop="trigger_count" label="持续次数" width="90">
                <template #default="{ row }">
                  <span>{{ row.trigger_count || 1 }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="last_triggered_at" label="最后触发" width="160">
                <template #default="{ row }">
                  <span class="time-text">{{ row.last_triggered_at || row.time }}</span>
                </template>
              </el-table-column>
              <el-table-column prop="status" label="状态" width="90">
                <template #default="{ row }">
                  <el-tag :type="row.status === 'resolved' ? 'success' : 'warning'" size="small">
                    {{ row.status === 'resolved' ? '已处理' : '待处理' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="200" fixed="right">
                <template #default="{ row }">
                  <el-button v-if="row.status !== 'resolved'" type="primary" link size="small" @click="resolveAlert(row)">
                    处理
                  </el-button>
                  <el-button v-if="row.status !== 'resolved'" type="warning" link size="small" @click="resolveSimilar(row)">
                    处理同类
                  </el-button>
                  <el-button type="primary" link size="small" @click="viewDetail(row)">
                    详情
                  </el-button>
                </template>
              </el-table-column>
            </el-table>

            <div class="pagination">
              <el-pagination
                v-model:current-page="currentPage"
                v-model:page-size="pageSize"
                :total="total"
                :page-sizes="[10, 20, 50]"
                layout="total, sizes, prev, pager, next"
              />
            </div>
          </template>
        </el-card>
      </el-col>
    </el-row>

    <!-- 添加/编辑规则对话框 -->
    <el-dialog v-model="ruleDialogVisible" :title="isEditRule ? '编辑告警规则' : '添加告警规则'" width="600px">
      <el-form :model="ruleForm" :rules="ruleFormRules" ref="ruleFormRef" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="ruleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="监控指标" prop="conditionType">
          <el-select v-model="ruleForm.conditionType" placeholder="选择监控指标" style="width: 100%">
            <el-option v-for="opt in alertMetricOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="触发条件">
          <el-row :gutter="10">
            <el-col :span="8">
              <el-select v-model="ruleForm.operator" placeholder="操作符">
                <el-option v-for="opt in alertOperatorOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
              </el-select>
            </el-col>
            <el-col :span="16">
              <el-input v-model="ruleForm.threshold" placeholder="阈值">
                <template #append>
                  <el-select v-model="ruleForm.unit" style="width: 80px">
                    <el-option v-for="opt in alertUnitOptions" :key="opt.value" :label="opt.label" :value="opt.value" />
                  </el-select>
                </template>
              </el-input>
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="告警级别" prop="level">
          <el-radio-group v-model="ruleForm.level">
            <el-radio v-for="opt in alertRuleLevelRadios" :key="opt.value" :value="opt.value">{{ opt.label }}</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-checkbox-group v-model="ruleForm.channels">
            <el-checkbox v-for="opt in alertNotifyChannelOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="钉钉Webhook" v-if="ruleForm.channels.includes('dingtalk')">
          <el-input v-model="ruleForm.dingtalkWebhook" placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." />
        </el-form-item>
        <el-form-item label="接收邮箱" v-if="ruleForm.channels.includes('email')">
          <el-select v-model="ruleForm.emails" multiple filterable allow-create placeholder="输入邮箱地址" style="width: 100%">
          </el-select>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRule">确定</el-button>
      </template>
    </el-dialog>

    <!-- 告警详情对话框 -->
    <el-dialog v-model="detailDialogVisible" title="告警详情" width="500px">
      <div v-if="selectedAlert" class="alert-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="告警时间">{{ selectedAlert.time }}</el-descriptions-item>
          <el-descriptions-item label="告警级别">
            <el-tag :type="getAlertTagType(selectedAlert.level)" size="small">
              {{ getLevelText(selectedAlert.level) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="告警来源">{{ selectedAlert.source }}</el-descriptions-item>
          <el-descriptions-item label="告警信息">{{ selectedAlert.message }}</el-descriptions-item>
          <el-descriptions-item label="处理状态">
            <el-tag :type="selectedAlert.status === 'resolved' ? 'success' : 'warning'" size="small">
              {{ selectedAlert.status === 'resolved' ? '已处理' : '待处理' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <template #footer>
        <el-button @click="detailDialogVisible = false">关闭</el-button>
        <el-button v-if="selectedAlert?.status !== 'resolved'" type="primary" @click="selectedAlert && resolveAlert(selectedAlert)">
          标记为已处理
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { alertApi, type AlertRule as AlertRulePayload } from '@/api/alert'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'
import {
  ALERT_LEVEL_OPTIONS,
  ALERT_METRIC_OPTIONS,
  ALERT_OPERATOR_OPTIONS,
  ALERT_UNIT_OPTIONS,
  ALERT_NOTIFY_CHANNEL_OPTIONS,
  ALERT_RULE_LEVEL_RADIOS
} from '@/constants/pages/alerts'

const alertLevelOptions = [...ALERT_LEVEL_OPTIONS]
const alertMetricOptions = [...ALERT_METRIC_OPTIONS]
const alertOperatorOptions = [...ALERT_OPERATOR_OPTIONS]
const alertUnitOptions = [...ALERT_UNIT_OPTIONS]
const alertNotifyChannelOptions = [...ALERT_NOTIFY_CHANNEL_OPTIONS]
const alertRuleLevelRadios = [...ALERT_RULE_LEVEL_RADIOS]

interface AlertRule {
  id: string
  name: string
  enabled: boolean
  condition: {
    type: string
    typeLabel: string
    text: string
  }
  channels: string[]
}

interface Alert {
  id: string
  time: string
  level: string
  source: string
  message: string
  status: string
  dedup_key?: string
  trigger_count?: number
  last_triggered_at?: string
}

interface RequestState {
  loading: boolean
  error: string | null
}

interface AlertStatsData {
  critical: number
  warning: number
  todayTotal: number
  resolved: number
}

interface AlertRuleApiModel {
  id: string
  name: string
  enabled?: boolean
  condition?: {
    type?: string
    operator?: string
    threshold?: number
  }
  notifyChannels?: string[]
}

interface AlertStatsApiModel {
  critical?: number
  warning?: number
  todayTotal?: number
  resolved?: number
}

const selectedLevel = ref('')
const dateRange = ref([])
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(100)
const ruleDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const isEditRule = ref(false)
const ruleFormRef = ref<FormInstance>()
const selectedAlert = ref<Alert | null>(null)
const rulesRequest = ref<RequestState>({ loading: false, error: null })
const historyRequest = ref<RequestState>({ loading: false, error: null })
const statsRequest = ref<RequestState>({ loading: false, error: null })
const statsData = ref<AlertStatsData>({ critical: 0, warning: 0, todayTotal: 0, resolved: 0 })

const alertStats = computed(() => [
  { title: '严重告警', value: statsData.value.critical, icon: 'WarningFilled', color: '#FF3B30' },
  { title: '警告', value: statsData.value.warning, icon: 'Warning', color: '#FF9500' },
  { title: '今日告警', value: statsData.value.todayTotal, icon: 'BellFilled', color: '#007AFF' },
  { title: '已处理', value: statsData.value.resolved, icon: 'CircleCheckFilled', color: '#34C759' }
])

const isStatsEmpty = computed(() =>
  alertStats.value.every((item) => Number(item.value || 0) === 0)
)

const alertRules = ref<AlertRule[]>([])

const alerts = ref<Alert[]>([])

const ruleForm = reactive({
  id: '',
  name: '',
  conditionType: '',
  operator: '>',
  threshold: '',
  unit: '%',
  level: 'warning',
  channels: [] as string[],
  dingtalkWebhook: '',
  emails: [] as string[],
  enabled: true
})

const ruleFormRules: FormRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  conditionType: [{ required: true, message: '请选择监控指标', trigger: 'change' }],
  level: [{ required: true, message: '请选择告警级别', trigger: 'change' }]
}

const filteredAlerts = computed(() => {
  if (!selectedLevel.value) return alerts.value
  return alerts.value.filter(a => a.level === selectedLevel.value)
})

const getConditionType = (type: string) => {
  const types: Record<string, string> = {
    latency: 'warning',
    error_rate: 'danger',
    quota: 'warning',
    availability: 'danger',
    cache_hit_rate: 'info'
  }
  return types[type] || 'info'
}

const getAlertTagType = (level: string) => {
  const types: Record<string, string> = {
    critical: 'danger',
    warning: 'warning',
    info: 'info'
  }
  return types[level] || 'info'
}

const getLevelText = (level: string) => {
  const texts: Record<string, string> = {
    critical: '严重',
    warning: '警告',
    info: '信息'
  }
  return texts[level] || level
}

const showAddRuleDialog = () => {
  isEditRule.value = false
  Object.assign(ruleForm, {
    id: '',
    name: '',
    conditionType: '',
    operator: '>',
    threshold: '',
    unit: '%',
    level: 'warning',
    channels: [],
    dingtalkWebhook: '',
    emails: [],
    enabled: true
  })
  ruleDialogVisible.value = true
}

const editRule = (rule: AlertRule) => {
  isEditRule.value = true
  Object.assign(ruleForm, {
    id: rule.id,
    name: rule.name,
    conditionType: rule.condition.type,
    operator: '>',
    threshold: '80',
    unit: '%',
    level: 'warning',
    channels: [],
    enabled: rule.enabled
  })
  ruleDialogVisible.value = true
}

const deleteRule = async (rule: AlertRule) => {
  try {
    await ElMessageBox.confirm(`确定删除规则 ${rule.name} 吗？`, '提示', { type: 'warning' })
    await alertApi.deleteRule(rule.id)
    handleSuccess('删除成功')
    fetchRules()
  } catch (error) {
    if (!isCancelError(error)) {
      handleApiError(error, '删除失败')
    }
  }
}

const handleRuleChange = async (rule: AlertRule) => {
  try {
    await alertApi.updateRule(rule.id, { enabled: rule.enabled })
    handleSuccess(`${rule.name} 已${rule.enabled ? '启用' : '禁用'}`)
  } catch (error) {
    rule.enabled = !rule.enabled
    handleApiError(error, '状态更新失败')
  }
}

const submitRule = async () => {
  if (!ruleFormRef.value) return
  try {
    const valid = await ruleFormRef.value.validate()
    if (valid) {
      const ruleData: Omit<AlertRulePayload, 'id' | 'createdAt' | 'updatedAt'> = {
        name: ruleForm.name,
        enabled: ruleForm.enabled,
        condition: {
          type: ruleForm.conditionType as 'error_rate' | 'latency' | 'quota' | 'availability',
          operator: ruleForm.operator,
          threshold: parseFloat(ruleForm.threshold) || 80
        },
        notifyChannels: ruleForm.channels
      }
      
      if (isEditRule.value) {
        await alertApi.updateRule(ruleForm.id, ruleData)
        handleSuccess('规则更新成功')
      } else {
        await alertApi.createRule(ruleData)
        handleSuccess('规则添加成功')
      }
      ruleDialogVisible.value = false
      fetchRules()
    }
  } catch (error) {
    handleApiError(error, '操作失败')
  }
}

const resolveAlert = async (alert: Alert) => {
  try {
    await alertApi.resolveAlert(alert.id)
    alert.status = 'resolved'
    handleSuccess('告警已处理')
    detailDialogVisible.value = false
    fetchAlerts()
    fetchStats()
  } catch (error) {
    handleApiError(error, '处理失败')
  }
}

const resolveSimilar = async (alert: Alert) => {
  try {
    await ElMessageBox.confirm(
      `确定处理同类告警吗？\n级别：${getLevelText(alert.level)}\n来源：${alert.source}\n信息：${alert.message}`,
      '批量处理同类告警',
      { type: 'warning' }
    )

    const raw = await alertApi.resolveSimilar({
      dedup_key: alert.dedup_key || undefined,
      level: alert.level,
      source: alert.source,
      message: alert.message
    })
    const data = extractResolveSimilarPayload(raw)
    const affected = Number(data.affected || 0)

    handleSuccess(`已处理同类告警 ${affected} 条`)
    await Promise.all([fetchAlerts(), fetchStats()])
  } catch (error) {
    if (!isCancelError(error)) {
      handleApiError(error, '批量处理失败')
    }
  }
}

const clearHistory = async () => {
  try {
    await ElMessageBox.confirm('确定清空全部告警历史吗？该操作不可恢复。', '清空告警历史', {
      type: 'warning'
    })

    const raw = await alertApi.clearHistory()
    const data = extractClearHistoryPayload(raw)
    const affected = Number(data.affected || 0)

    handleSuccess(`已清空告警历史 ${affected} 条`)
    await Promise.all([fetchAlerts(), fetchStats()])
  } catch (error) {
    if (!isCancelError(error)) {
      handleApiError(error, '清空告警历史失败')
    }
  }
}

const viewDetail = (alert: Alert) => {
  selectedAlert.value = alert
  detailDialogVisible.value = true
}

const buildRequestError = (error: unknown, fallback: string): string => {
  if (error instanceof Error && error.message) {
    return error.message
  }

  return fallback
}

const isCancelError = (error: unknown): boolean => error === 'cancel'

const extractRulesPayload = (
  payload: AlertRuleApiModel[] | { data?: AlertRuleApiModel[] }
): AlertRuleApiModel[] => {
  if (Array.isArray(payload)) {
    return payload
  }

  if (Array.isArray(payload.data)) {
    return payload.data
  }

  return []
}

type AlertHistoryListPayload = { list?: Alert[]; total?: number }
type AlertHistoryEnvelopePayload = { data?: AlertHistoryListPayload }
type AlertHistoryPayloadInput = Alert[] | AlertHistoryListPayload | AlertHistoryEnvelopePayload

type AlertStatsEnvelopePayload = { data?: AlertStatsApiModel }
type AlertStatsPayloadInput = AlertStatsApiModel | AlertStatsEnvelopePayload

type ResolveSimilarPayload = { affected?: number }
type ResolveSimilarEnvelopePayload = { data?: ResolveSimilarPayload }
type ResolveSimilarPayloadInput = ResolveSimilarPayload | ResolveSimilarEnvelopePayload

type ClearHistoryPayload = { affected?: number }
type ClearHistoryEnvelopePayload = { data?: ClearHistoryPayload }
type ClearHistoryPayloadInput = ClearHistoryPayload | ClearHistoryEnvelopePayload

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null

const isAlertHistoryEnvelopePayload = (
  payload: AlertHistoryPayloadInput
): payload is AlertHistoryEnvelopePayload => isRecord(payload) && 'data' in payload

const isAlertStatsEnvelopePayload = (
  payload: AlertStatsPayloadInput
): payload is AlertStatsEnvelopePayload => isRecord(payload) && 'data' in payload

const isResolveSimilarEnvelopePayload = (
  payload: ResolveSimilarPayloadInput
): payload is ResolveSimilarEnvelopePayload => isRecord(payload) && 'data' in payload

const isClearHistoryEnvelopePayload = (
  payload: ClearHistoryPayloadInput
): payload is ClearHistoryEnvelopePayload => isRecord(payload) && 'data' in payload

const extractHistoryPayload = (payload: AlertHistoryPayloadInput): { list: Alert[]; total: number } => {
  if (Array.isArray(payload)) {
    return { list: payload, total: payload.length }
  }

  if (isAlertHistoryEnvelopePayload(payload)) {
    const list = Array.isArray(payload.data?.list) ? payload.data.list : []
    return {
      list,
      total: Number(payload.data?.total ?? list.length)
    }
  }

  const list = Array.isArray(payload.list) ? payload.list : []
  return {
    list,
    total: Number(payload.total ?? list.length)
  }
}

const extractStatsPayload = (payload: AlertStatsPayloadInput): AlertStatsApiModel => {
  if (isAlertStatsEnvelopePayload(payload)) {
    return payload.data || {}
  }

  return payload
}

const extractResolveSimilarPayload = (payload: ResolveSimilarPayloadInput): ResolveSimilarPayload => {
  if (isResolveSimilarEnvelopePayload(payload)) {
    return payload.data || {}
  }

  return payload
}

const extractClearHistoryPayload = (payload: ClearHistoryPayloadInput): ClearHistoryPayload => {
  if (isClearHistoryEnvelopePayload(payload)) {
    return payload.data || {}
  }

  return payload
}

const mapRuleModel = (rule: AlertRuleApiModel): AlertRule => {
  const conditionType = rule.condition?.type || 'latency'
  const conditionOperator = rule.condition?.operator || '>'
  const conditionThreshold = rule.condition?.threshold ?? 80

  return {
    id: rule.id,
    name: rule.name,
    enabled: rule.enabled ?? true,
    condition: {
      type: conditionType,
      typeLabel: getConditionLabel(conditionType),
      text: `${getConditionLabel(conditionType)} ${conditionOperator} ${conditionThreshold}`
    },
    channels: rule.notifyChannels || []
  }
}

const fetchRules = async () => {
  rulesRequest.value = { loading: true, error: null }

  try {
    const response = await alertApi.getRules()
    const rules = extractRulesPayload(response as AlertRuleApiModel[] | { data?: AlertRuleApiModel[] })
    alertRules.value = rules.map(mapRuleModel)
  } catch (error) {
    rulesRequest.value.error = buildRequestError(error, '告警规则加载失败')
  } finally {
    rulesRequest.value.loading = false
  }
}

const fetchAlerts = async () => {
  historyRequest.value = { loading: true, error: null }

  try {
    const params: {
      level?: string
      startDate?: string
      endDate?: string
    } = {}
    if (selectedLevel.value) params.level = selectedLevel.value
    if (dateRange.value && dateRange.value.length === 2) {
      params.startDate = String(dateRange.value[0])
      params.endDate = String(dateRange.value[1])
    }

    const response = await alertApi.getHistory(params)
    const history = extractHistoryPayload(
      response as Alert[] | { list?: Alert[]; total?: number } | { data?: { list?: Alert[]; total?: number } }
    )

    alerts.value = history.list.map((alert) => normalizeAlert(alert))
    total.value = history.total
  } catch (error) {
    historyRequest.value.error = buildRequestError(error, '告警历史加载失败')
  } finally {
    historyRequest.value.loading = false
  }
}

const normalizeAlert = (alert: Partial<Alert>): Alert => ({
  id: alert.id || '',
  time: alert.time || '',
  level: alert.level || 'info',
  source: alert.source || '-',
  message: alert.message || '-',
  status: alert.status || 'pending',
  dedup_key: alert.dedup_key,
  trigger_count: Number(alert.trigger_count || 1),
  last_triggered_at: alert.last_triggered_at || alert.time || ''
})

const fetchStats = async () => {
  statsRequest.value = { loading: true, error: null }

  try {
    const response = await alertApi.getStats()
    const data = extractStatsPayload(response as AlertStatsApiModel | { data?: AlertStatsApiModel })

    statsData.value = {
      critical: Number(data?.critical || 0),
      warning: Number(data?.warning || 0),
      todayTotal: Number(data?.todayTotal || 0),
      resolved: Number(data?.resolved || 0)
    }
  } catch (error) {
    statsRequest.value.error = buildRequestError(error, '告警统计加载失败')
    statsData.value = { critical: 0, warning: 0, todayTotal: 0, resolved: 0 }
  } finally {
    statsRequest.value.loading = false
  }
}

const getConditionLabel = (type: string) => {
  const labels: Record<string, string> = {
    latency: '延迟',
    error_rate: '错误率',
    quota: '配额',
    availability: '可用性'
  }
  return labels[type] || type
}

onMounted(() => {
  fetchRules()
  fetchAlerts()
  fetchStats()
})
</script>

<style scoped lang="scss">
.alerts-page {
  .request-state {
    min-height: 180px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .page-card {
    border-radius: var(--border-radius-lg);
    border: none;
  }

  .stats-row {
    margin-bottom: var(--spacing-xl);
  }

  .stat-card {
    border-radius: var(--border-radius-lg);
    border: none;

    .stat-content {
      display: flex;
      align-items: center;
      gap: var(--spacing-lg);

      .stat-icon {
        width: 56px;
        height: 56px;
        border-radius: var(--border-radius-lg);
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .stat-info {
        .stat-value {
          font-size: var(--font-size-3xl);
          font-weight: var(--font-weight-bold);
          color: var(--text-primary);
        }

        .stat-title {
          font-size: var(--font-size-md);
          color: var(--text-secondary);
        }
      }
    }
  }

  .content-row {
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;

      .filter-group {
        display: flex;
        align-items: center;
      }
    }

    .rules-card {
      .rules-list {
        .rule-item {
          padding: var(--spacing-lg);
          border-bottom: 1px solid var(--border-primary);

          &:last-child {
            border-bottom: none;
          }

          .rule-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: var(--spacing-sm);

            .rule-info {
              display: flex;
              align-items: center;
              gap: var(--spacing-sm);

              .rule-name {
                font-weight: var(--font-weight-semibold);
                font-size: var(--font-size-md);
              }
            }
          }

          .rule-condition {
            display: flex;
            align-items: center;
            gap: var(--spacing-xs);
            font-size: var(--font-size-sm);
            color: var(--text-secondary);
            margin-bottom: var(--spacing-sm);
          }

          .rule-channels {
            display: flex;
            gap: var(--spacing-xs);
            margin-bottom: var(--spacing-sm);
          }

          .rule-actions {
            display: flex;
            gap: var(--spacing-sm);
          }
        }
      }
    }

    .alerts-table {
      .time-text {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-sm);
      }
    }

    .pagination {
      margin-top: var(--spacing-lg);
      display: flex;
      justify-content: flex-end;
    }
  }

  .alert-detail {
    :deep(.el-descriptions__label) {
      width: 100px;
    }
  }
}
</style>
