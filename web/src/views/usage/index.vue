<template>
  <div class="usage-page">
    <div class="page-head">
      <div class="head-title">使用记录</div>
      <div class="head-subtitle">查看和分析您的 API 使用历史</div>
    </div>

    <el-row :gutter="14" class="summary-row">
      <el-col :xs="24" :sm="12" :lg="6">
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
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-amber">T</div>
            <div>
              <div class="card-label">总 Token</div>
              <div class="card-value">{{ formatCompact(overview.total_tokens) }}</div>
              <div class="card-hint">输入: {{ formatCompact(promptTokens) }} / 输出: {{ formatCompact(outputTokens) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
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
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="never" class="summary-card">
          <div class="card-inner">
            <div class="icon-wrap icon-violet">L</div>
            <div>
              <div class="card-label">缓存命中率</div>
              <div class="card-value">{{ cacheHitRate.toFixed(1) }}%</div>
              <div class="card-hint">命中 {{ formatNumber(cacheHits) }} / 未命中 {{ formatNumber(cacheMisses) }}</div>
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
            <el-select v-model="selectedModel" style="width: 220px" placeholder="全部模型" clearable>
              <el-option label="全部模型" value="" />
              <el-option
                v-for="model in modelOptions"
                :key="model"
                :label="model"
                :value="model"
              />
            </el-select>
          </div>
          <div class="filter-item">
            <div class="filter-label">时间范围</div>
            <el-select v-model="range" style="width: 140px" @change="refreshAll">
              <el-option label="近 24 小时" value="24h" />
              <el-option label="近 7 天" value="7d" />
              <el-option label="近 30 天" value="30d" />
            </el-select>
          </div>
        </div>
        <div class="filters-right">
          <el-button :loading="loading" @click="resetFilters">重置</el-button>
          <el-button type="primary" :loading="loading" @click="exportCsv">导出 CSV</el-button>
        </div>
      </div>
    </el-card>

    <el-card shadow="never" class="table-card">
      <el-table :data="pagedRows" stripe class="usage-table" v-loading="loading">
        <el-table-column prop="accountName" label="API Key账号" min-width="160" show-overflow-tooltip />
        <el-table-column prop="provider" label="服务商" width="120" />
        <el-table-column prop="time" label="最近时间" width="180" />
        <el-table-column prop="firstTokenLatency" label="首 Token 耗时" width="130" align="right" />
        <el-table-column prop="totalLatency" label="总耗时" width="110" align="right" />
        <el-table-column prop="model" label="模型" min-width="170" />
        <el-table-column label="入 Token" width="130" align="right">
          <template #default="{ row }">
            {{ formatCompact(row.inputTokens) }}
          </template>
        </el-table-column>
        <el-table-column label="出 Token" width="130" align="right">
          <template #default="{ row }">
            {{ formatCompact(row.outputTokens) }}
          </template>
        </el-table-column>
        <el-table-column label="总 Token" width="140" align="right">
          <template #default="{ row }">
            <div class="token-total">{{ formatCompact(row.totalTokens) }}</div>
          </template>
        </el-table-column>
        <el-table-column label="缓存命中" width="150" align="right">
          <template #default="{ row }">
            <span>{{ row.cacheHit }}</span>
          </template>
        </el-table-column>
        <el-table-column label="费用" width="120" align="right">
          <template #default="{ row }">
            <span class="cost">${{ row.cost.toFixed(5) }}</span>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!loading && filteredRows.length === 0" description="暂无真实使用数据" />

      <div class="pager-wrap">
        <div class="pager-text">显示 {{ pageStart }} 至 {{ pageEnd }} 共 {{ filteredRows.length }} 条结果</div>
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
import { computed, onMounted, ref } from 'vue'
import { accountApi } from '@/api/account'
import { getCacheStats } from '@/api/metrics'

type RangeType = '24h' | '7d' | '30d'

interface UsageRow {
  id: string
  accountName: string
  provider: string
  time: string
  timestamp: number
  firstTokenLatency: string
  totalLatency: string
  firstTokenSeconds: number
  totalDurationSeconds: number
  model: string
  inputTokens: number
  outputTokens: number
  totalTokens: number
  cacheHit: string
  cost: number
}

interface ChatMessageLite {
  id: string
  role: string
  timestamp: number
  stats?: {
    firstTokenTime?: number
    totalTime?: number
    totalTokens?: number
    promptTokens?: number
    completionTokens?: number
    cacheHit?: boolean
  }
}

interface ConversationLite {
  provider: string
  model: string
  messages: ChatMessageLite[]
}

const loading = ref(false)
const range = ref<RangeType>('7d')
const page = ref(1)
const pageSize = ref(20)
const selectedModel = ref('')

const usageRows = ref<UsageRow[]>([])
const accountRows = ref<Array<{ id: string; name?: string; provider?: string; provider_type?: string; is_active?: boolean }>>([])
const cacheStats = ref({ hits: 0, misses: 0, hit_rate: 0 })

const rangeStart = computed(() => {
  const now = Date.now()
  if (range.value === '24h') return now - 24 * 60 * 60 * 1000
  if (range.value === '30d') return now - 30 * 24 * 60 * 60 * 1000
  return now - 7 * 24 * 60 * 60 * 1000
})

const rangeRows = computed(() => usageRows.value.filter(row => row.timestamp >= rangeStart.value))

const modelOptions = computed(() => {
  const set = new Set<string>()
  rangeRows.value.forEach(row => set.add(row.model))
  return Array.from(set)
})

const filteredRows = computed(() => {
  const base = rangeRows.value
  if (!selectedModel.value) return base
  return base.filter(row => row.model === selectedModel.value)
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

const overview = computed(() => ({
  total_requests: rangeRows.value.length,
  total_tokens: rangeRows.value.reduce((sum, row) => sum + row.totalTokens, 0)
}))

const promptTokens = computed(() => rangeRows.value.reduce((sum, row) => sum + row.inputTokens, 0))
const outputTokens = computed(() => rangeRows.value.reduce((sum, row) => sum + row.outputTokens, 0))
const totalCost = computed(() => rangeRows.value.reduce((sum, row) => sum + row.cost, 0))

const rowCacheStats = computed(() => {
  let hits = 0
  let misses = 0
  for (const row of rangeRows.value) {
    if (row.cacheHit === '命中') {
      hits++
    } else if (row.cacheHit === '未命中') {
      misses++
    }
  }
  return { hits, misses }
})

const cacheHits = computed(() => {
  const total = rowCacheStats.value.hits + rowCacheStats.value.misses
  if (total > 0) return rowCacheStats.value.hits
  return cacheStats.value.hits
})

const cacheMisses = computed(() => {
  const total = rowCacheStats.value.hits + rowCacheStats.value.misses
  if (total > 0) return rowCacheStats.value.misses
  return cacheStats.value.misses
})

const cacheHitRate = computed(() => {
  const rowTotal = rowCacheStats.value.hits + rowCacheStats.value.misses
  if (rowTotal > 0) {
    return (rowCacheStats.value.hits / rowTotal) * 100
  }

  const direct = cacheStats.value.hit_rate
  if (direct > 0) return direct
  const total = cacheStats.value.hits + cacheStats.value.misses
  return total > 0 ? (cacheStats.value.hits / total) * 100 : 0
})

const formatNumber = (value: number) => (Number.isFinite(value) ? value.toLocaleString() : '0')

const formatCompact = (value: number) => {
  if (!value) return '0'
  if (value >= 1000000) return `${(value / 1000000).toFixed(2)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return `${value}`
}

const formatDateTime = (time: number) => {
  const d = new Date(time)
  if (Number.isNaN(d.getTime())) return '-'
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  const s = String(d.getSeconds()).padStart(2, '0')
  return `${y}/${m}/${day} ${h}:${min}:${s}`
}

const providerLabelMap: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  deepseek: 'DeepSeek',
  qwen: '通义千问',
  zhipu: '智谱AI',
  moonshot: 'Kimi',
  volcengine: '火山方舟',
  minimax: 'MiniMax',
  baichuan: '百川'
}

const resolveProviderLabel = (provider: string) => providerLabelMap[provider] || provider || '-'

const resolveAccountNameByProvider = (providerName: string) => {
  const normalized = providerName.toLowerCase()
  const exact = accountRows.value.find(account => {
    const p = (account.provider || account.provider_type || '').toLowerCase()
    return p === normalized && account.is_active
  }) || accountRows.value.find(account => {
    const p = (account.provider || account.provider_type || '').toLowerCase()
    return p === normalized
  })

  if (!exact) return '-'
  return exact.name || exact.id || '-'
}

const loadUsageRowsFromChat = () => {
  const raw = localStorage.getItem('chat_conversations')
  if (!raw) {
    usageRows.value = []
    return
  }

  let conversations: ConversationLite[] = []
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      conversations = parsed
    }
  } catch {
    conversations = []
  }

  const rows: UsageRow[] = []
  for (const conversation of conversations) {
    const providerValue = conversation.provider || ''
    const provider = resolveProviderLabel(providerValue)
    const accountName = resolveAccountNameByProvider(providerValue)

    for (const message of conversation.messages || []) {
      if (message.role !== 'assistant') continue
      const totalTokens = message.stats?.totalTokens || 0
      if (totalTokens <= 0) continue

      const inputTokens = message.stats?.promptTokens || Math.round(totalTokens * 0.6)
      const outputTokensValue = message.stats?.completionTokens || Math.max(0, totalTokens - inputTokens)
      const firstTokenSeconds = Number(message.stats?.firstTokenTime || 0)
      const totalDurationSeconds = Number(message.stats?.totalTime || 0)

      rows.push({
        id: message.id,
        accountName,
        provider,
        time: formatDateTime(message.timestamp),
        timestamp: message.timestamp,
        firstTokenLatency: `${firstTokenSeconds.toFixed(2)}s`,
        totalLatency: `${totalDurationSeconds.toFixed(2)}s`,
        firstTokenSeconds,
        totalDurationSeconds,
        model: conversation.model || '-',
        inputTokens,
        outputTokens: outputTokensValue,
        totalTokens,
        cacheHit: message.stats?.cacheHit === true ? '命中' : '未命中',
        cost: totalTokens * 0.00000035
      })
    }
  }

  usageRows.value = rows.sort((a, b) => b.timestamp - a.timestamp)
}

const fetchAccounts = async () => {
  const res = await accountApi.getList({ page: 1, pageSize: 200 })
  const data = (res as any)?.data || []
  accountRows.value = Array.isArray(data) ? data : []
}

const fetchCacheStats = async () => {
  const res = await getCacheStats()
  const data = (res as any)?.data || res || {}
  const requestCache = data.request_cache || {}
  const responseCache = data.response_cache || {}

  const hits = Number(requestCache.hits || 0) + Number(responseCache.hits || 0)
  const misses = Number(requestCache.misses || 0) + Number(responseCache.misses || 0)
  const total = hits + misses
  const hitRate = total > 0 ? (hits / total) * 100 : 0

  cacheStats.value = {
    hits,
    misses,
    hit_rate: hitRate
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchAccounts(), fetchCacheStats()])
    loadUsageRowsFromChat()
    page.value = 1
  } finally {
    loading.value = false
  }
}

const resetFilters = async () => {
  selectedModel.value = ''
  range.value = '7d'
  await refreshAll()
}

const exportCsv = () => {
  const header = ['API Key账号', '服务商', '最近时间', '首Token耗时', '总耗时', '模型', '入Token', '出Token', '总Token', '缓存命中', '费用']
  const lines = filteredRows.value.map(row => [
    row.accountName,
    row.provider,
    row.time,
    row.firstTokenLatency,
    row.totalLatency,
    row.model,
    row.inputTokens,
    row.outputTokens,
    row.totalTokens,
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
  refreshAll()
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

.token-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.token-in {
  color: #16a34a;
}

.token-out {
  color: #7c3aed;
}

.token-total {
  margin-top: 4px;
  color: #0284c7;
  font-size: 13px;
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
