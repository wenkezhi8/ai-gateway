<template>
  <div class="alerts-page">
    <!-- 告警统计卡片 -->
    <el-row :gutter="20" class="stats-row">
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

          <div class="rules-list">
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
                  <el-option label="全部" value="" />
                  <el-option label="严重" value="critical" />
                  <el-option label="警告" value="warning" />
                  <el-option label="信息" value="info" />
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
              </div>
            </div>
          </template>

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
            <el-table-column prop="status" label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.status === 'resolved' ? 'success' : 'warning'" size="small">
                  {{ row.status === 'resolved' ? '已处理' : '待处理' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button v-if="row.status !== 'resolved'" type="primary" link size="small" @click="resolveAlert(row)">
                  处理
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
            <el-option label="API延迟" value="latency" />
            <el-option label="错误率" value="error_rate" />
            <el-option label="额度使用率" value="quota" />
            <el-option label="服务可用性" value="availability" />
            <el-option label="缓存命中率" value="cache_hit_rate" />
          </el-select>
        </el-form-item>
        <el-form-item label="触发条件">
          <el-row :gutter="10">
            <el-col :span="8">
              <el-select v-model="ruleForm.operator" placeholder="操作符">
                <el-option label="大于" value=">" />
                <el-option label="小于" value="<" />
                <el-option label="等于" value="=" />
                <el-option label="大于等于" value=">=" />
                <el-option label="小于等于" value="<=" />
              </el-select>
            </el-col>
            <el-col :span="16">
              <el-input v-model="ruleForm.threshold" placeholder="阈值">
                <template #append>
                  <el-select v-model="ruleForm.unit" style="width: 80px">
                    <el-option label="ms" value="ms" />
                    <el-option label="%" value="%" />
                    <el-option label="次" value="count" />
                  </el-select>
                </template>
              </el-input>
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="告警级别" prop="level">
          <el-radio-group v-model="ruleForm.level">
            <el-radio value="critical">严重</el-radio>
            <el-radio value="warning">警告</el-radio>
            <el-radio value="info">信息</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="通知渠道">
          <el-checkbox-group v-model="ruleForm.channels">
            <el-checkbox value="email">邮件</el-checkbox>
            <el-checkbox value="dingtalk">钉钉</el-checkbox>
            <el-checkbox value="wechat">企业微信</el-checkbox>
            <el-checkbox value="webhook">Webhook</el-checkbox>
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
import { alertApi } from '@/api/alert'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'

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
const loading = ref(false)

const alertStats = computed(() => [
  { title: '严重告警', value: alerts.value.filter(a => a.level === 'critical' && a.status !== 'resolved').length, icon: 'WarningFilled', color: '#FF3B30' },
  { title: '警告', value: alerts.value.filter(a => a.level === 'warning' && a.status !== 'resolved').length, icon: 'Warning', color: '#FF9500' },
  { title: '今日告警', value: alerts.value.length, icon: 'BellFilled', color: '#007AFF' },
  { title: '已处理', value: alerts.value.filter(a => a.status === 'resolved').length, icon: 'CircleCheckFilled', color: '#34C759' }
])

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
    if ((error as any) !== 'cancel') {
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
      const ruleData = {
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
        await alertApi.updateRule(ruleForm.id, ruleData as any)
        handleSuccess('规则更新成功')
      } else {
        await alertApi.createRule(ruleData as any)
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
  } catch (error) {
    handleApiError(error, '处理失败')
  }
}

const viewDetail = (alert: Alert) => {
  selectedAlert.value = alert
  detailDialogVisible.value = true
}

const fetchRules = async () => {
  try {
    const res = await alertApi.getRules()
    const data = (res as any)?.data || res
    if (Array.isArray(data)) {
      alertRules.value = data.map((r: any) => ({
        id: r.id,
        name: r.name,
        enabled: r.enabled ?? true,
        condition: {
          type: r.condition?.type || 'latency',
          typeLabel: getConditionLabel(r.condition?.type || 'latency'),
          text: `${getConditionLabel(r.condition?.type || 'latency')} ${r.condition?.operator || '>'} ${r.condition?.threshold || 80}`
        },
        channels: r.notifyChannels || []
      }))
    }
  } catch (error) {
    console.warn('Failed to fetch alert rules:', error)
  }
}

const fetchAlerts = async () => {
  loading.value = true
  try {
    const params: any = {}
    if (selectedLevel.value) params.level = selectedLevel.value
    if (dateRange.value && dateRange.value.length === 2) {
      params.startDate = dateRange.value[0]
      params.endDate = dateRange.value[1]
    }
    const res = await alertApi.getHistory(params)
    const data = (res as any)?.data || res
    if (data?.list) {
      alerts.value = data.list.map((a: any) => ({
        id: a.id,
        time: a.time,
        level: a.level,
        source: a.source,
        message: a.message,
        status: a.status
      }))
      total.value = data.total || alerts.value.length
    } else if (Array.isArray(data)) {
      alerts.value = data.map((a: any) => ({
        id: a.id,
        time: a.time,
        level: a.level,
        source: a.source,
        message: a.message,
        status: a.status
      }))
    }
  } catch (error) {
    console.warn('Failed to fetch alerts:', error)
  } finally {
    loading.value = false
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
})
</script>

<style scoped lang="scss">
.alerts-page {
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
