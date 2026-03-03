<template>
  <div class="usage-page">
    <div class="page-head">
      <div class="head-title">使用记录</div>
      <div class="head-subtitle">查看和分析您的 API 使用历史</div>
    </div>

    <el-row :gutter="14" class="summary-row">
      <el-col :xs="24" :sm="12" :lg="4">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-blue">R</div>
            <div>
              <div class="card-label">总请求数</div>
              <div class="card-value">{{ formatNumber(overview.total_requests) }}</div>
              <div class="card-hint">所选范围内</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="4">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-amber">T</div>
            <div>
              <div class="card-label">总 Token</div>
              <div class="card-value">{{ formatCompact(overview.total_tokens) }}</div>
              <div class="card-hint">
                输入: {{ formatCompact(promptTokens) }} / 输出: {{ formatCompact(outputTokens) }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="4">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-green">$</div>
            <div>
              <div class="card-label">总消费</div>
              <div class="card-value success">${{ totalCost.toFixed(4) }}</div>
              <div class="card-hint">按估算单价计算</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="4">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-cyan">S</div>
            <div>
              <div class="card-label">命中节省 Token</div>
              <div class="card-value">{{ formatCompact(savedTokens) }}</div>
              <div class="card-hint">
                节省费用: ${{ savedCost.toFixed(4) }} · 命中请求 {{ formatNumber(savedRequests) }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="4">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-violet">L</div>
            <div>
              <div class="card-label">缓存命中率</div>
              <div class="card-value">{{ cacheHitRate.toFixed(1) }}%</div>
              <div class="card-hint">
                命中 {{ formatNumber(cacheHits) }} / 未命中 {{ formatNumber(cacheMisses) }}
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-card shadow="never" class="filters-card">
      <div class="filters-wrap">
        <div class="filters-left">
          <div class="filter-item">
            <div class="filter-label">模型</div>
            <el-select
              v-model="selectedModel"
              style="width: 220px"
              placeholder="全部模型"
              clearable
            >
              <el-option label="全部模型" value="" />
              <el-option v-for="model in modelOptions" :key="model" :label="model" :value="model" />
            </el-select>
          </div>
          <div class="filter-item">
            <div class="filter-label">任务类型</div>
            <el-select
              v-model="selectedTaskType"
              style="width: 180px"
              placeholder="全部类型"
              clearable
            >
              <el-option label="全部类型" value="" />
              <el-option v-for="tt in taskTypeOptions" :key="tt" :label="tt" :value="tt" />
            </el-select>
          </div>
          <div class="filter-item">
            <div class="filter-label">时间范围</div>
            <el-select v-model="range" style="width: 140px">
              <el-option label="近 24 小时" value="24h" />
              <el-option label="近 7 天" value="7d" />
              <el-option label="近 30 天" value="30d" />
            </el-select>
          </div>
        </div>
        <div class="filters-right">
          <el-button type="danger" plain :loading="loading" @click="clearUsageLogs">
            清空使用记录
          </el-button>
          <el-button :loading="loading" @click="resetFilters">重置</el-button>
          <el-button type="primary" :loading="loading" @click="exportCsv">导出 CSV</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="table-card">
      <el-table
        :data="pagedRows"
        stripe
        class="usage-table"
        v-loading="loading"
        table-layout="fixed"
      >
        <el-table-column
          prop="accountName"
          label="账号"
          min-width="160"
          class-name="cell-single-line"
          show-overflow-tooltip
        />
        <el-table-column prop="provider" label="服务商" min-width="110" class-name="cell-single-line" />
        <el-table-column prop="time" label="最近时间" min-width="168" class-name="cell-single-line" />
        <el-table-column
          prop="firstTokenLatency"
          label="首 Token 耗时"
          width="118"
          align="right"
          class-name="cell-num"
        />
        <el-table-column prop="totalLatency" label="总耗时" width="108" align="right" class-name="cell-num" />
        <el-table-column prop="model" label="模型" min-width="190" class-name="cell-single-line" show-overflow-tooltip />
        <el-table-column label="任务类型" width="120" class-name="cell-single-line">
          <template #default="{ row }">
            <el-tooltip
              placement="top"
              :content="row.taskTypeRaw"
              :disabled="!row.taskTypeRaw || row.taskTypeRaw === '-'"
            >
              <span class="cell-single-line">{{ row.taskTypeLabel }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="requestType" label="请求类型" width="108" class-name="cell-single-line" />
        <el-table-column prop="inferenceIntensity" label="推理强度" width="100" class-name="cell-single-line" />
        <el-table-column prop="userAgent" label="用户代理" min-width="240" class-name="cell-single-line" show-overflow-tooltip />
        <el-table-column label="入 Token" width="112" align="right" class-name="cell-num">
          <template #default="{ row }">
            {{ formatCompact(row.inputTokens) }}
          </template>
        </el-table-column>
        <el-table-column label="出 Token" width="112" align="right" class-name="cell-num">
          <template #default="{ row }">
            {{ formatCompact(row.outputTokens) }}
          </template>
        </el-table-column>
        <el-table-column label="总 Token" width="120" align="right" class-name="cell-num">
          <template #default="{ row }">
            <span class="token-total">{{ formatCompact(row.totalTokens) }}</span>
          </template>
        </el-table-column>
        <el-table-column label="Token来源" width="96" align="center" class-name="cell-center">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.usageSourceLabel }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="节省 Token" width="120" align="right" class-name="cell-num">
          <template #default="{ row }">
            <div class="token-total">{{ formatCompact(row.savedTokens) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="缓存命中" width="92" align="center" class-name="cell-center">
          <template #default="{ row }">
            <span>{{ row.cacheHit }}</span>
          </template>
        </el-table-column>
        <el-table-column label="费用" width="112" align="right" class-name="cell-num">
          <template #default="{ row }">
            <span class="cost">${{ row.cost.toFixed(5) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && filteredRows.length === 0" description="暂无真实使用数据" />

      <div class="pager-wrap">
        <div class="pager-text">
          显示 {{ pageStart }} 至 {{ pageEnd }} 共 {{ filteredRows.length }} 条结果
        </div>
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="filteredRows.length"
          :page-sizes="[10, 20, 50, 100]"
          layout="sizes, prev, pager, next"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { request } from '@/api/request'
import { API } from '@/constants/api'
import { filterUsageRows } from '@/utils/usage-filters'
import { accountApi, type Account } from '@/api/account'
import { USAGE_CSV_HEADER } from '@/constants/pages/usage'
import {
  pickUsageOverview,
  type UsageStatsPayload
} from './usage-overview'
import { mapUsageLogToRow, type UsageRow } from './usage-row-mapper'

type RangeType = '24h' | '7d' | '30d'

const loading = ref(false)
const range = ref<RangeType>('7d')
const page = ref(1)
const pageSize = ref(20)
const selectedModel = ref('')
const selectedTaskType = ref('')

const usageRows = ref<UsageRow[]>([])
const usageStats = ref<UsageStatsPayload | null>(null)
const accounts = ref<Account[]>([])

const accountNameMap = computed(() => {
  const map = new Map<string, string>()
  accounts.value.forEach(acc => {
    if (acc.provider && acc.name) {
      map.set(acc.provider, acc.name)
    }
  })
  return map
})

const modelOptions = computed(() => {
  const set = new Set<string>()
  usageRows.value.forEach(row => set.add(row.model))
  return Array.from(set)
})

const taskTypeOptions = computed(() => {
  const set = new Set<string>()
  usageRows.value.forEach(row => set.add(row.taskType))
  return Array.from(set)
})

const filteredRows = computed(() => {
  return filterUsageRows(usageRows.value, {
    model: selectedModel.value,
    taskType: selectedTaskType.value
  })
})

const pagedRows = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})

const pageStart = computed(() => {
  if (!filteredRows.value.length) return 0
  return (page.value - 1) * pageSize.value + 1
})

const pageEnd = computed(() => {
  const end = page.value * pageSize.value
  return Math.min(end, filteredRows.value.length)
})

const overviewSummary = computed(() =>
  pickUsageOverview(
    usageStats.value,
    filteredRows.value.map(row => ({
      inputTokens: row.inputTokens,
      outputTokens: row.outputTokens,
      totalTokens: row.totalTokens,
      cacheHit: row.cacheHit,
      success: row.success
    }))
  )
)

const overview = computed(() => ({
  total_requests: overviewSummary.value.totalRequests,
  total_tokens: overviewSummary.value.totalTokens
}))

const promptTokens = computed(() =>
  filteredRows.value.reduce((sum, row) => sum + row.inputTokens, 0)
)
const outputTokens = computed(() =>
  filteredRows.value.reduce((sum, row) => sum + row.outputTokens, 0)
)
const totalCost = computed(() => overviewSummary.value.totalCost)
const cacheHits = computed(() => overviewSummary.value.cacheHits)
const cacheMisses = computed(() => overviewSummary.value.cacheMisses)
const cacheHitRate = computed(() => overviewSummary.value.cacheHitRate)
const savedTokens = computed(() => overviewSummary.value.savedTokens)
const savedRequests = computed(() => overviewSummary.value.savedRequests)
const savedCost = computed(() => overviewSummary.value.savedCost)

const formatNumber = (value: number) => (Number.isFinite(value) ? value.toLocaleString() : '0')

const formatCompact = (value: number) => {
  if (!value) return '0'
  if (value >= 1000000) return `${(value / 1000000).toFixed(2)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return `${value}`
}

const fetchUsageLogs = async () => {
  try {
    const res = await request.get(API.USAGE.LOGS, {
      params: {
        range: range.value,
        limit: 1000,
        model: selectedModel.value || undefined,
        task_type: selectedTaskType.value || undefined
      }
    })
    const data = (res as any)?.data || []
    const rows: UsageRow[] = data.map((log: any) => mapUsageLogToRow(log, accountNameMap.value))

    usageRows.value = rows.sort((a, b) => b.timestamp - a.timestamp)
  } catch (e) {
    console.warn('Failed to fetch usage logs from API:', e)
    usageRows.value = []
  }
}

const fetchUsageStats = async () => {
  try {
    const res = await request.get(API.USAGE.STATS, {
      params: {
        range: range.value,
        model: selectedModel.value || undefined,
        task_type: selectedTaskType.value || undefined
      }
    })
    usageStats.value = ((res as any)?.data || null) as UsageStatsPayload | null
  } catch (e) {
    console.warn('Failed to fetch usage stats from API, fallback to local rows:', e)
    usageStats.value = null
  }
}

const fetchAccounts = async () => {
  try {
    const res = await accountApi.getList()
    accounts.value = (res as any)?.data || []
  } catch (e) {
    console.warn('Failed to fetch accounts:', e)
    accounts.value = []
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchUsageLogs(), fetchUsageStats()])
    page.value = 1
  } finally {
    loading.value = false
  }
}

const resetFilters = async () => {
  const changed =
    selectedModel.value !== '' || selectedTaskType.value !== '' || range.value !== '7d'
  selectedModel.value = ''
  selectedTaskType.value = ''
  range.value = '7d'
  if (!changed) {
    await refreshAll()
  }
}

const clearUsageLogs = async () => {
  try {
    await ElMessageBox.confirm(
      '确定清空全部使用记录吗？该操作不可恢复。',
      '清空使用记录',
      {
        type: 'warning',
        confirmButtonText: '确认清空',
        cancelButtonText: '取消'
      }
    )
  } catch {
    return
  }

  loading.value = true
  try {
    const res = await request.delete(API.USAGE.CLEAR)
    const deleted = Number((res as any)?.data?.deleted || 0)
    ElMessage.success(`已清空使用记录，共删除 ${formatNumber(deleted)} 条`)
    await Promise.all([fetchUsageLogs(), fetchUsageStats()])
    page.value = 1
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || e?.message || '清空失败')
  } finally {
    loading.value = false
  }
}

const exportCsv = () => {
  const header = [...USAGE_CSV_HEADER]
  const lines = filteredRows.value.map(row => [
    row.accountName,
    row.provider,
    row.time,
    row.firstTokenLatency,
    row.totalLatency,
    row.model,
    row.taskType,
    row.requestType,
    row.inferenceIntensity,
    row.userAgent,
    row.inputTokens,
    row.outputTokens,
    row.totalTokens,
    row.usageSourceLabel,
    row.savedTokens,
    row.cacheHit,
    row.cost.toFixed(5)
  ])

  const csv = [header, ...lines]
    .map(cols => cols.map(col => `"${String(col).replace(/"/g, '""')}"`).join(','))
    .join('\n')

  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `usage-records-${Date.now()}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

onMounted(() => {
  loading.value = true
  fetchAccounts()
    .then(refreshAll)
    .finally(() => {
      loading.value = false
    })
})

watch([range, selectedModel, selectedTaskType], () => {
  void refreshAll()
})
</script>

<style scoped>
.usage-page {
  min-height: calc(100vh - 130px);
  background: linear-gradient(180deg, #eef5f7 0%, #f6f8fa 56%, #f8fafc 100%);
  border-radius: 16px;
  padding: 18px;
}

.page-head {
  margin-bottom: 12px;
}

.head-title {
  font-size: 28px;
  font-weight: 700;
  color: #111827;
  letter-spacing: 0.2px;
}

.head-subtitle {
  margin-top: 6px;
  color: #6b7280;
  font-size: 14px;
}

.summary-row {
  margin-bottom: 14px;
}

.summary-card {
  border-radius: 14px;
  border: 1px solid #e9eef2;
}

.card-inner {
  display: flex;
  align-items: center;
  gap: 12px;
}

.icon-wrap {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
}

.icon-blue {
  color: #3b82f6;
  background: #dbeafe;
}

.icon-amber {
  color: #d97706;
  background: #fef3c7;
}

.icon-green {
  color: #16a34a;
  background: #dcfce7;
}

.icon-cyan {
  color: #0891b2;
  background: #cffafe;
}

.icon-violet {
  color: #7c3aed;
  background: #ede9fe;
}

.card-label {
  color: #6b7280;
  font-size: 13px;
}

.card-value {
  margin-top: 2px;
  font-size: 34px;
  line-height: 1.1;
  color: #111827;
  font-weight: 700;
}

.card-value.success {
  color: #16a34a;
}

.card-hint {
  margin-top: 4px;
  color: #9ca3af;
  font-size: 12px;
}

.filters-card,
.table-card {
  border-radius: 14px;
  border: 1px solid #e8edf2;
}

.filters-wrap {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.filters-left {
  display: flex;
  gap: 14px;
  flex-wrap: wrap;
}

.filter-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.filter-label {
  font-size: 13px;
  font-weight: 600;
  color: #111827;
}

.filters-right {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.table-card {
  margin-top: 14px;
}

.token-total {
  margin-top: 0;
  color: #0284c7;
  font-size: 13px;
}

.cell-single-line {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.cell-num {
  font-variant-numeric: tabular-nums;
  font-feature-settings: 'tnum';
}

.cell-center {
  text-align: center;
}

:deep(.usage-table .el-table__header-wrapper th),
:deep(.usage-table .el-table__body-wrapper td) {
  padding: 10px 12px;
}

:deep(.usage-table .el-table__row) {
  height: 42px;
}

.cost {
  color: #16a34a;
  font-weight: 600;
}

.pager-wrap {
  margin-top: 14px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.pager-text {
  color: #6b7280;
  font-size: 13px;
}

@media (max-width: 900px) {
  .usage-page {
    padding: 12px;
  }

  .head-title {
    font-size: 22px;
  }

  .card-value {
    font-size: 24px;
  }
}
</style>
