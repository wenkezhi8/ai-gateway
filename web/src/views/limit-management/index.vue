<template>
  <div class="limit-management-page">
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.totalAccounts }}</div>
              <div class="stat-label">账号总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon active">
              <el-icon><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.activeAccounts }}</div>
              <div class="stat-label">活跃账号</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon warning">
              <el-icon><Warning /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.warningAccounts }}</div>
              <div class="stat-label">预警账号</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon exceeded">
              <el-icon><CircleClose /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.exceededAccounts }}</div>
              <div class="stat-label">超限账号</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <el-col :xs="24" :lg="16">
        <el-card shadow="never" class="accounts-card">
          <template #header>
            <div class="card-header">
              <span>账号限额状态</span>
              <div class="header-actions">
                <el-select v-model="providerFilter" placeholder="服务商筛选" clearable size="small" style="width: 150px">
                  <el-option label="全部服务商" value="" />
                  <el-option v-for="p in providers" :key="p" :label="getProviderLabel(p)" :value="p" />
                </el-select>
                <el-button type="primary" size="small" @click="refreshAccounts" :loading="loading">
                  <el-icon><Refresh /></el-icon>
                  刷新
                </el-button>
              </div>
            </div>
          </template>

          <el-table :data="filteredAccounts" stripe v-loading="loading" class="accounts-table">
            <el-table-column prop="name" label="账号名称" min-width="160">
              <template #default="{ row }">
                <div class="account-name-cell">
                  <img v-if="getProviderLogo(row.provider)" :src="getProviderLogo(row.provider)" class="provider-logo-avatar" />
                  <el-avatar v-else :size="28" :style="{ background: getProviderColor(row.provider) }">
                    {{ row.name?.charAt(0) || 'A' }}
                  </el-avatar>
                  <div class="account-info">
                    <span class="name">{{ row.name }}</span>
                    <div class="account-meta">
                      <el-tag v-if="row.plan_type" type="info" size="small" effect="plain">{{ row.plan_type }}</el-tag>
                      <el-tag v-if="row.is_active" type="success" size="small" effect="dark">活跃</el-tag>
                    </div>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="provider" label="服务商" width="120">
              <template #default="{ row }">
                <div class="provider-cell">
                  <img v-if="getProviderLogo(row.provider)" :src="getProviderLogo(row.provider)" class="provider-logo-small" />
                  <span>{{ getProviderLabel(row.provider) }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="多级限额状态" min-width="280">
              <template #default="{ row }">
                <div class="multi-limit-cell">
                  <div v-if="row.usage?.hour5 || row.limits?.hour5" class="limit-item">
                    <span class="limit-label">5小时</span>
                    <el-progress
                      :percentage="row.usage?.hour5?.percent_used || 0"
                      :status="getProgressStatus(row.usage?.hour5?.warning_level)"
                      :stroke-width="8"
                      :show-text="false"
                      style="width: 60px"
                    />
                    <span class="limit-value">{{ formatUsage({ used: row.usage?.hour5?.used || 0, limit: row.usage?.hour5?.limit || row.limits?.hour5?.limit || 0 }) }}</span>
                  </div>
                  <div v-if="row.usage?.week || row.limits?.week" class="limit-item">
                    <span class="limit-label">周</span>
                    <el-progress
                      :percentage="row.usage?.week?.percent_used || 0"
                      :status="getProgressStatus(row.usage?.week?.warning_level)"
                      :stroke-width="8"
                      :show-text="false"
                      style="width: 60px"
                    />
                    <span class="limit-value">{{ formatUsage({ used: row.usage?.week?.used || 0, limit: row.usage?.week?.limit || row.limits?.week?.limit || 0 }) }}</span>
                  </div>
                  <div v-if="row.usage?.month || row.limits?.month" class="limit-item">
                    <span class="limit-label">月</span>
                    <el-progress
                      :percentage="row.usage?.month?.percent_used || 0"
                      :status="getProgressStatus(row.usage?.month?.warning_level)"
                      :stroke-width="8"
                      :show-text="false"
                      style="width: 60px"
                    />
                    <span class="limit-value">{{ formatUsage({ used: row.usage?.month?.used || 0, limit: row.usage?.month?.limit || row.limits?.month?.limit || 0 }) }}</span>
                  </div>
                  <div v-if="(row.usage?.token || row.limits?.token) && !row.usage?.hour5 && !row.limits?.hour5 && !row.usage?.week && !row.limits?.week" class="limit-item">
                    <span class="limit-label">Token</span>
                    <el-progress
                      :percentage="row.usage?.token?.percent_used || 0"
                      :status="getProgressStatus(row.usage?.token?.warning_level)"
                      :stroke-width="8"
                      :show-text="false"
                      style="width: 60px"
                    />
                    <span class="limit-value">{{ formatUsage({ used: row.usage?.token?.used || 0, limit: row.usage?.token?.limit || row.limits?.token?.limit || 0 }) }}</span>
                  </div>
                  <div v-if="!hasAnyUsage(row)" class="no-limit">未配置限额</div>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="RPM" width="120">
              <template #default="{ row }">
                <div v-if="row.usage?.rpm || row.limits?.rpm" class="rpm-cell">
                  <span :class="{ 'rpm-warning': (row.usage?.rpm?.percent_used || 0) >= 90 }">
                    {{ row.usage?.rpm?.used || 0 }}/{{ row.usage?.rpm?.limit || row.limits?.rpm?.limit || 0 }}
                  </span>
                </div>
                <span v-else class="no-limit">-</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="90" align="center">
              <template #default="{ row }">
                <el-tag :type="getAccountStatusType(row)" size="small">
                  {{ getAccountStatusText(row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="160" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link size="small" @click="showLimitDialog(row)">
                  配置限额
                </el-button>
                <el-button
                  v-if="!row.is_active && row.enabled"
                  type="success"
                  link
                  size="small"
                  @click="handleForceSwitch(row)"
                >
                  激活
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :xs="24" :lg="8">
        <el-card shadow="never" class="history-card">
          <template #header>
            <div class="card-header">
              <span>切换历史</span>
              <el-button type="primary" link size="small" @click="refreshHistory">
                <el-icon><Refresh /></el-icon>
              </el-button>
            </div>
          </template>

          <el-timeline v-if="switchHistory.length > 0" class="switch-timeline">
            <el-timeline-item
              v-for="(event, index) in switchHistory"
              :key="index"
              :timestamp="formatTime(event.timestamp)"
              placement="top"
              :type="getSwitchType(event.reason)"
            >
              <div class="switch-event">
                <div class="switch-accounts">
                  <span class="from">{{ getAccountName(event.from_account) }}</span>
                  <el-icon><Right /></el-icon>
                  <span class="to">{{ getAccountName(event.to_account) }}</span>
                </div>
                <div class="switch-reason">{{ event.reason }}</div>
                <div v-if="event.duration" class="switch-duration">
                  耗时: {{ event.duration }}ms
                </div>
              </div>
            </el-timeline-item>
          </el-timeline>
          <el-empty v-else description="暂无切换记录" :image-size="80" />
        </el-card>

        <el-card shadow="never" class="alerts-card" style="margin-top: 20px">
          <template #header>
            <div class="card-header">
              <span>限额预警</span>
              <el-badge :value="activeAlerts.length" :hidden="activeAlerts.length === 0" type="danger" />
            </div>
          </template>

          <div v-if="activeAlerts.length > 0" class="alerts-list">
            <div v-for="alert in activeAlerts" :key="alert.timestamp" class="alert-item" :class="alert.type">
              <el-icon class="alert-icon">
                <Warning v-if="alert.type === 'warning'" />
                <CircleClose v-else />
              </el-icon>
              <div class="alert-content">
                <div class="alert-message">{{ alert.message }}</div>
                <div class="alert-time">{{ formatTime(alert.timestamp) }}</div>
              </div>
            </div>
          </div>
          <el-empty v-else description="暂无预警" :image-size="60" />
        </el-card>
      </el-col>
    </el-row>

    <el-dialog
      v-model="limitDialogVisible"
      title="配置账号限额"
      width="680px"
      destroy-on-close
    >
      <el-form :model="limitForm" label-width="100px" v-if="selectedAccount">
        <div class="account-info-header">
          <el-avatar :size="40" :style="{ background: getProviderColor(selectedAccount.provider) }">
            {{ selectedAccount.name?.charAt(0) || 'A' }}
          </el-avatar>
          <div class="info">
            <div class="name">{{ selectedAccount.name }}</div>
            <div class="provider">{{ getProviderLabel(selectedAccount.provider) }}</div>
          </div>
          <div class="plan-quick-select">
            <el-select v-model="selectedPlanTemplate" placeholder="快速选择套餐模板" clearable @change="applyPlanTemplate">
              <el-option-group label="智谱AI Coding Plan">
                <el-option label="Lite 套餐 (80次/5h, 400次/周)" value="zhipu-lite" />
                <el-option label="Pro 套餐 (400次/5h, 2000次/周)" value="zhipu-pro" />
                <el-option label="Max 套餐 (1600次/5h, 8000次/周)" value="zhipu-max" />
              </el-option-group>
              <el-option-group label="阿里云百炼 Coding Plan">
                <el-option label="Lite 套餐 (1200次/5h, 9000次/周, 18000次/月)" value="bailian-lite" />
                <el-option label="Pro 套餐 (6000次/5h, 45000次/周, 90000次/月)" value="bailian-pro" />
              </el-option-group>
              <el-option-group label="火山方舟 Coding Plan">
                <el-option label="基础套餐" value="volcengine-basic" />
                <el-option label="标准套餐" value="volcengine-standard" />
                <el-option label="专业套餐" value="volcengine-pro" />
              </el-option-group>
            </el-select>
          </div>
        </div>

        <el-divider content-position="left">
          <el-icon><Timer /></el-icon>
          多级请求限额 (Coding Plan 模式)
        </el-divider>
        
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="每5小时">
              <el-input-number v-model="limitForm.hour5.limit" :min="0" :step="100" controls-position="right" class="limit-input" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="每周限额">
              <el-input-number v-model="limitForm.week.limit" :min="0" :step="500" controls-position="right" class="limit-input" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="每月限额">
              <el-input-number v-model="limitForm.month.limit" :min="0" :step="1000" controls-position="right" class="limit-input" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="预警阈值">
          <el-slider v-model="limitForm.hour5.warning" :min="50" :max="99" :format-tooltip="(val: number) => val + '%'" />
        </el-form-item>

        <el-divider content-position="left">
          <el-icon><Coin /></el-icon>
          Token 限额 (传统模式)
        </el-divider>
        
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="周期">
              <el-select v-model="limitForm.token.period" style="width: 100%">
                <el-option label="每分钟" value="minute" />
                <el-option label="每小时" value="hour" />
                <el-option label="每天" value="day" />
                <el-option label="每月" value="month" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="限额">
              <el-input-number v-model="limitForm.token.limit" :min="0" :step="1000" controls-position="right" class="limit-input" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="预警阈值">
          <el-slider v-model="limitForm.token.warning" :min="50" :max="99" :format-tooltip="(val: number) => val + '%'" />
        </el-form-item>

        <el-divider content-position="left">
          <el-icon><Odometer /></el-icon>
          RPM 限额
        </el-divider>
        
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="每分钟">
              <el-input-number v-model="limitForm.rpm.limit" :min="0" :step="10" controls-position="right" class="limit-input" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="预警阈值">
              <el-slider v-model="limitForm.rpm.warning" :min="50" :max="99" :format-tooltip="(val: number) => val + '%'" />
            </el-form-item>
          </el-col>
        </el-row>

        <el-alert
          v-if="limitForm.hour5.limit === 0 && limitForm.week.limit === 0 && limitForm.month.limit === 0 && limitForm.token.limit === 0 && limitForm.rpm.limit === 0"
          title="提示：所有限额为0时，将不会进行限额检查"
          type="info"
          :closable="false"
          show-icon
        />
      </el-form>
      <template #footer>
        <el-button @click="limitDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveLimits" :loading="saving">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Key, CircleCheck, Warning, CircleClose, Refresh, Right, Timer, Coin, Odometer
} from '@element-plus/icons-vue'
import { accountApi, type Account, type SwitchEvent, type LimitAlert, type LimitConfig } from '@/api/account'

const loading = ref(false)
const saving = ref(false)
const accounts = ref<Account[]>([])
const switchHistory = ref<SwitchEvent[]>([])
const activeAlerts = ref<LimitAlert[]>([])
const providerFilter = ref('')
const limitDialogVisible = ref(false)
const selectedAccount = ref<Account | null>(null)
const selectedPlanTemplate = ref('')

let refreshTimer: number | null = null

const stats = computed(() => {
  const total = accounts.value.length
  const active = accounts.value.filter(a => a.is_active).length
  const warning = accounts.value.filter(a => 
    a.usage?.hour5?.warning_level === 'warning' || 
    a.usage?.week?.warning_level === 'warning' ||
    a.usage?.month?.warning_level === 'warning' ||
    a.usage?.token?.warning_level === 'warning' || 
    a.usage?.rpm?.warning_level === 'warning'
  ).length
  const exceeded = accounts.value.filter(a => 
    a.usage?.hour5?.warning_level === 'critical' || 
    a.usage?.week?.warning_level === 'critical' ||
    a.usage?.month?.warning_level === 'critical' ||
    a.usage?.token?.warning_level === 'critical' || 
    a.usage?.rpm?.warning_level === 'critical' ||
    (a.usage?.hour5?.percent_used ?? 0) >= 100 ||
    (a.usage?.week?.percent_used ?? 0) >= 100 ||
    (a.usage?.month?.percent_used ?? 0) >= 100 ||
    (a.usage?.token?.percent_used ?? 0) >= 100 || 
    (a.usage?.rpm?.percent_used ?? 0) >= 100
  ).length
  return { totalAccounts: total, activeAccounts: active, warningAccounts: warning, exceededAccounts: exceeded }
})

const providers = computed(() => {
  const set = new Set(accounts.value.map(a => a.provider))
  return Array.from(set)
})

const filteredAccounts = computed(() => {
  if (!providerFilter.value) return accounts.value
  return accounts.value.filter(a => a.provider === providerFilter.value)
})

interface LimitFormItem {
  type: 'request' | 'token' | 'rpm'
  period: 'minute' | 'hour' | '5hour' | 'day' | 'week' | 'month'
  limit: number
  warning: number
}

const limitForm = reactive<{
  hour5: LimitFormItem
  week: LimitFormItem
  month: LimitFormItem
  token: LimitFormItem
  rpm: LimitFormItem
}>({
  hour5: { type: 'request', period: '5hour', limit: 0, warning: 90 },
  week: { type: 'request', period: 'week', limit: 0, warning: 90 },
  month: { type: 'request', period: 'month', limit: 0, warning: 90 },
  token: { type: 'token', period: 'day', limit: 0, warning: 90 },
  rpm: { type: 'rpm', period: 'minute', limit: 0, warning: 90 }
})

const planTemplates: Record<string, Partial<typeof limitForm>> = {
  'zhipu-lite': { hour5: { ...limitForm.hour5, limit: 80 }, week: { ...limitForm.week, limit: 400 }, month: { ...limitForm.month, limit: 0 } },
  'zhipu-pro': { hour5: { ...limitForm.hour5, limit: 400 }, week: { ...limitForm.week, limit: 2000 }, month: { ...limitForm.month, limit: 0 } },
  'zhipu-max': { hour5: { ...limitForm.hour5, limit: 1600 }, week: { ...limitForm.week, limit: 8000 }, month: { ...limitForm.month, limit: 0 } },
  'bailian-lite': { hour5: { ...limitForm.hour5, limit: 1200 }, week: { ...limitForm.week, limit: 9000 }, month: { ...limitForm.month, limit: 18000 } },
  'bailian-pro': { hour5: { ...limitForm.hour5, limit: 6000 }, week: { ...limitForm.week, limit: 45000 }, month: { ...limitForm.month, limit: 90000 } },
  'volcengine-basic': { hour5: { ...limitForm.hour5, limit: 500 }, week: { ...limitForm.week, limit: 3000 }, month: { ...limitForm.month, limit: 6000 } },
  'volcengine-standard': { hour5: { ...limitForm.hour5, limit: 2000 }, week: { ...limitForm.week, limit: 12000 }, month: { ...limitForm.month, limit: 24000 } },
  'volcengine-pro': { hour5: { ...limitForm.hour5, limit: 5000 }, week: { ...limitForm.week, limit: 30000 }, month: { ...limitForm.month, limit: 60000 } },
}

const applyPlanTemplate = (template: string) => {
  if (!template) return
  const config = planTemplates[template]
  if (config) {
    if (config.hour5) limitForm.hour5 = { ...limitForm.hour5, ...config.hour5 }
    if (config.week) limitForm.week = { ...limitForm.week, ...config.week }
    if (config.month) limitForm.month = { ...limitForm.month, ...config.month }
    ElMessage.success(`已应用套餐模板配置`)
  }
}

const accountNameMap = computed(() => {
  const map: Record<string, string> = {}
  accounts.value.forEach(a => { map[a.id] = a.name })
  return map
})

const providerLabels: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Claude',
  zhipu: '智谱AI',
  qwen: '通义千问',
  deepseek: 'DeepSeek',
  ernie: '文心一言',
  volcengine: '火山方舟',
  moonshot: '月之暗面',
  minimax: 'MiniMax',
  baichuan: '百川',
  yi: '零一万物',
  bailian: '阿里百炼'
}

const providerColors: Record<string, string> = {
  openai: '#10a37f',
  anthropic: '#cc785c',
  zhipu: '#1a73e8',
  qwen: '#ff6a00',
  deepseek: '#0066ff',
  ernie: '#2932e1',
  volcengine: '#ff4d4f',
  moonshot: '#722ed1',
  minimax: '#00b96b',
  baichuan: '#1890ff',
  yi: '#faad14',
  bailian: '#ff9500'
}

const providerLogos: Record<string, string> = {
  openai: '/logos/openai.svg',
  anthropic: '/logos/anthropic.svg',
  zhipu: '/logos/zhipu.svg',
  qwen: '/logos/qwen.svg',
  deepseek: '/logos/deepseek.svg',
  ernie: '/logos/ernie.svg',
  volcengine: '/logos/volcengine.svg',
  moonshot: '/logos/moonshot.svg',
  minimax: '/logos/minimax.svg',
  baichuan: '/logos/baichuan.svg'
}

const getProviderLabel = (provider: string) => providerLabels[provider] || provider
const getProviderColor = (provider: string) => providerColors[provider] || '#666'
const getProviderLogo = (provider: string) => providerLogos[provider] || ''

const formatNumber = (num: number) => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatUsage = (usage: { used: number; limit: number }) => {
  return `${formatNumber(usage.used)}/${formatNumber(usage.limit)}`
}

const hasAnyUsage = (account: Account) => {
  if (account.usage?.hour5 || account.usage?.week || account.usage?.month || account.usage?.token || account.usage?.rpm) {
    return true
  }
  return account.limits?.hour5 || account.limits?.week || account.limits?.month || account.limits?.token || account.limits?.rpm
}

const formatTime = (timestamp: string) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return date.toLocaleString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

const getProgressStatus = (level?: string) => {
  if (level === 'critical') return 'exception'
  if (level === 'warning') return 'warning'
  return ''
}

const getAccountStatusType = (account: Account) => {
  if (!account.enabled) return 'info'
  const usages = [account.usage?.hour5, account.usage?.week, account.usage?.month, account.usage?.token, account.usage?.rpm]
  if (usages.some(u => (u?.percent_used ?? 0) >= 100 || u?.warning_level === 'critical')) return 'danger'
  if (usages.some(u => u?.warning_level === 'warning')) return 'warning'
  return 'success'
}

const getAccountStatusText = (account: Account) => {
  if (!account.enabled) return '已禁用'
  const usages = [account.usage?.hour5, account.usage?.week, account.usage?.month, account.usage?.token, account.usage?.rpm]
  if (usages.some(u => (u?.percent_used ?? 0) >= 100 || u?.warning_level === 'critical')) return '已超限'
  if (usages.some(u => u?.warning_level === 'warning')) return '预警中'
  return '正常'
}

const getSwitchType = (reason: string) => {
  if (reason.includes('exceeded') || reason.includes('超限')) return 'danger'
  if (reason.includes('disabled') || reason.includes('禁用')) return 'warning'
  return 'primary'
}

const getAccountName = (id: string) => accountNameMap.value[id] || id

const refreshAccounts = async () => {
  loading.value = true
  try {
    const res = await accountApi.getList()
    if (res.success && Array.isArray(res.data)) {
      accounts.value = res.data
    }
  } catch (e) {
    console.error('Failed to load accounts:', e)
  } finally {
    loading.value = false
  }
}

const refreshHistory = async () => {
  try {
    const res = await accountApi.getSwitchHistory(20)
    if (res.success && Array.isArray(res.data)) {
      switchHistory.value = res.data
    }
  } catch (e) {
    console.error('Failed to load switch history:', e)
  }
}

const refreshAlerts = async () => {
  try {
    const res = await accountApi.getLimitAlerts()
    if (res.success && Array.isArray(res.data)) {
      activeAlerts.value = res.data
    }
  } catch (e) {
    console.error('Failed to load alerts:', e)
  }
}

const showLimitDialog = (account: Account) => {
  selectedAccount.value = account
  selectedPlanTemplate.value = ''
  
  const parseLimit = (limits: Record<string, LimitConfig> | undefined, key: string, period: string): LimitFormItem => {
    if (limits?.[key]) {
      const l = limits[key]
      return { type: l.type as 'request', period: l.period as any, limit: l.limit, warning: l.warning }
    }
    return { type: 'request' as const, period: period as any, limit: 0, warning: 90 }
  }
  
  limitForm.hour5 = parseLimit(account.limits, 'hour5', '5hour')
  limitForm.week = parseLimit(account.limits, 'week', 'week')
  limitForm.month = parseLimit(account.limits, 'month', 'month')
  limitForm.token = parseLimit(account.limits, 'token', 'day')
  limitForm.rpm = parseLimit(account.limits, 'rpm', 'minute')
  
  limitDialogVisible.value = true
}

const saveLimits = async () => {
  if (!selectedAccount.value) return
  
  saving.value = true
  try {
    const limits: Record<string, LimitConfig> = {}
    
    if (limitForm.hour5.limit > 0) {
      limits.hour5 = { ...limitForm.hour5, type: 'request' }
    }
    if (limitForm.week.limit > 0) {
      limits.week = { ...limitForm.week, type: 'request' }
    }
    if (limitForm.month.limit > 0) {
      limits.month = { ...limitForm.month, type: 'request' }
    }
    if (limitForm.token.limit > 0) {
      limits.token = limitForm.token
    }
    if (limitForm.rpm.limit > 0) {
      limits.rpm = limitForm.rpm
    }
    
    await accountApi.updateLimits(selectedAccount.value.id, limits)
    ElMessage.success('限额配置已保存')
    limitDialogVisible.value = false
    await refreshAccounts()
  } catch (e) {
    ElMessage.error('保存失败')
    console.error(e)
  } finally {
    saving.value = false
  }
}

const handleForceSwitch = async (account: Account) => {
  try {
    await ElMessageBox.confirm(
      `确定要将 ${account.name} 设为活跃账号吗？`,
      '切换账号',
      { type: 'warning' }
    )
    await accountApi.forceSwitch(account.provider, account.id)
    ElMessage.success('账号已激活')
    await refreshAccounts()
    await refreshHistory()
  } catch (e) {
    if (e !== 'cancel') {
      ElMessage.error('切换失败')
      console.error(e)
    }
  }
}

const refreshAll = () => {
  refreshAccounts()
  refreshHistory()
  refreshAlerts()
}

onMounted(() => {
  refreshAll()
  refreshTimer = window.setInterval(refreshAll, 30000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.limit-management-page {
  padding: 0;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  margin-bottom: 10px;
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: white;
}

.stat-icon.total { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
.stat-icon.active { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); }
.stat-icon.warning { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }
.stat-icon.exceeded { background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%); }

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.stat-label {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.accounts-card {
  margin-bottom: 20px;
}

.account-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.provider-logo-avatar {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: contain;
}

.provider-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}

.provider-logo-small {
  width: 14px;
  height: 14px;
  border-radius: 2px;
  object-fit: contain;
}

.account-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.account-info .name {
  font-weight: 500;
}

.account-meta {
  display: flex;
  gap: 4px;
}

.multi-limit-cell {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.limit-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.limit-label {
  color: var(--el-text-color-secondary);
  width: 36px;
  flex-shrink: 0;
}

.limit-value {
  color: var(--el-text-color-primary);
  min-width: 80px;
}

.rpm-cell {
  font-size: 12px;
}

.rpm-warning {
  color: var(--el-color-danger);
  font-weight: 500;
}

.no-limit {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}

.history-card,
.alerts-card {
  height: fit-content;
}

.switch-timeline {
  max-height: 400px;
  overflow-y: auto;
}

.switch-event {
  padding: 4px 0;
}

.switch-accounts {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.switch-accounts .from {
  color: var(--el-text-color-secondary);
}

.switch-accounts .to {
  color: var(--el-color-primary);
  font-weight: 500;
}

.switch-reason {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.switch-duration {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
}

.alerts-list {
  max-height: 300px;
  overflow-y: auto;
}

.alert-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  margin-bottom: 8px;
}

.alert-item.warning {
  background: var(--el-color-warning-light-9);
}

.alert-item.critical,
.alert-item.exceeded {
  background: var(--el-color-danger-light-9);
}

.alert-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.alert-item.warning .alert-icon {
  color: var(--el-color-warning);
}

.alert-item.critical .alert-icon,
.alert-item.exceeded .alert-icon {
  color: var(--el-color-danger);
}

.alert-content {
  flex: 1;
  min-width: 0;
}

.alert-message {
  font-size: 13px;
  color: var(--el-text-color-primary);
  word-break: break-all;
}

.alert-time {
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  margin-top: 4px;
}

.account-info-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
  padding: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}

.account-info-header .info {
  flex: 1;
}

.account-info-header .info .name {
  font-weight: 600;
  font-size: 15px;
}

.account-info-header .info .provider {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
}

.plan-quick-select {
  width: 280px;
}

.limit-input {
  width: 100%;
}

.limit-input :deep(.el-input__inner) {
  text-align: left;
}

@media (max-width: 768px) {
  .stats-row .el-col {
    margin-bottom: 10px;
  }
  
  .plan-quick-select {
    width: 100%;
    margin-top: 12px;
  }
  
  .account-info-header {
    flex-wrap: wrap;
  }
}
</style>
