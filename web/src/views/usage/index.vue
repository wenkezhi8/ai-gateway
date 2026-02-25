<template>
  <div class="usage-page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar">
        <div>
          <div class="title">API 使用统计</div>
          <div class="subtitle">查看请求量、Token 消耗、模型与服务商使用分布</div>
        </div>
        <div class="actions">
          <el-radio-group v-model="range" size="small">
            <el-radio-button label="24h">24小时</el-radio-button>
            <el-radio-button label="7d">7天</el-radio-button>
          </el-radio-group>
          <el-button type="primary" :loading="loading" @click="refreshAll">刷新</el-button>
        </div>
      </div>
    </el-card>

    <el-row :gutter="16" class="stats-row">
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="label">总请求数</div>
          <div class="value">{{ formatNumber(overview.total_requests) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="label">今日请求</div>
          <div class="value">{{ formatNumber(overview.requests_today) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="label">Token 总量</div>
          <div class="value">{{ formatNumber(overview.total_tokens) }}</div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-card shadow="hover" class="stat-card">
          <div class="label">成功率</div>
          <div class="value">{{ formatPercent(overview.success_rate) }}%</div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="charts-row">
      <el-col :xs="24" :lg="16">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">请求趋势</div>
          </template>
          <div ref="trendChartRef" class="chart"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="8">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">服务商分布</div>
          </template>
          <div ref="providerChartRef" class="chart"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">模型使用排行</div>
          </template>
          <el-table :data="modelRows" stripe>
            <el-table-column prop="name" label="模型" min-width="180" />
            <el-table-column prop="requests" label="请求数" width="120" align="right">
              <template #default="{ row }">{{ formatNumber(row.requests) }}</template>
            </el-table-column>
            <el-table-column prop="tokens" label="Tokens" width="130" align="right">
              <template #default="{ row }">{{ formatNumber(row.tokens) }}</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :xs="24" :lg="12">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">服务商明细</div>
          </template>
          <el-table :data="providerRows" stripe>
            <el-table-column prop="name" label="服务商" min-width="120" />
            <el-table-column prop="requests" label="请求数" width="120" align="right">
              <template #default="{ row }">{{ formatNumber(row.requests) }}</template>
            </el-table-column>
            <el-table-column prop="success_rate" label="成功率" width="120" align="right">
              <template #default="{ row }">{{ formatPercent(row.success_rate) }}%</template>
            </el-table-column>
            <el-table-column prop="avg_latency_ms" label="平均延迟" width="120" align="right">
              <template #default="{ row }">{{ row.avg_latency_ms || 0 }}ms</template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import * as echarts from 'echarts'
import { getOverview, getProviders, getRequestTrend } from '@/api/metrics'

type RangeType = '24h' | '7d'

const loading = ref(false)
const range = ref<RangeType>('24h')

const overview = ref({
  total_requests: 0,
  requests_today: 0,
  total_tokens: 0,
  success_rate: 0,
  top_models: [] as Array<{ name: string; requests: number; tokens: number }>
})

const providerRows = ref<Array<{
  name: string
  requests: number
  success_rate: number
  avg_latency_ms: number
}>>([])

const requestTrend = ref<Array<{
  timestamp: string
  requests: number
  success: number
  failed: number
}>>([])

const providerDistribution = ref<Record<string, number>>({})

const modelRows = computed(() => overview.value.top_models || [])

const trendChartRef = ref<HTMLElement | null>(null)
const providerChartRef = ref<HTMLElement | null>(null)

let trendChart: echarts.ECharts | null = null
let providerChart: echarts.ECharts | null = null
let refreshTimer: number | null = null

const formatNumber = (value: number) => {
  if (!value) return '0'
  if (value >= 1000000) return `${(value / 1000000).toFixed(1)}M`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}K`
  return `${value}`
}

const formatPercent = (value: number) => (Number.isFinite(value) ? value.toFixed(1) : '0.0')

const fetchOverview = async () => {
  const res = await getOverview()
  const data = (res as any)?.data || res || {}
  overview.value = {
    total_requests: data.total_requests || 0,
    requests_today: data.requests_today || 0,
    total_tokens: data.total_tokens || 0,
    success_rate: data.success_rate || 0,
    top_models: Array.isArray(data.top_models) ? data.top_models : []
  }
}

const fetchProviders = async () => {
  const res = await getProviders()
  const data = (res as any)?.data || res || {}
  providerRows.value = Array.isArray(data.providers) ? data.providers : []
  providerDistribution.value = data.distribution || {}
}

const fetchRequestTrend = async () => {
  const period = range.value
  const interval = period === '7d' ? 'day' : 'hour'
  const res = await getRequestTrend({ period, interval })
  const data = (res as any)?.data || res || []
  requestTrend.value = Array.isArray(data) ? data : []
}

const renderTrendChart = () => {
  if (!trendChartRef.value) return
  if (!trendChart) trendChart = echarts.init(trendChartRef.value)

  const points = requestTrend.value
  const labels = points.map(item => {
    const d = new Date(item.timestamp)
    if (Number.isNaN(d.getTime())) return item.timestamp
    return range.value === '7d'
      ? `${d.getMonth() + 1}/${d.getDate()}`
      : `${String(d.getHours()).padStart(2, '0')}:00`
  })

  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['请求', '成功', '失败'] },
    grid: { left: 40, right: 16, top: 36, bottom: 28 },
    xAxis: { type: 'category', data: labels },
    yAxis: { type: 'value' },
    series: [
      { name: '请求', type: 'line', smooth: true, data: points.map(i => i.requests || 0) },
      { name: '成功', type: 'line', smooth: true, data: points.map(i => i.success || 0) },
      { name: '失败', type: 'line', smooth: true, data: points.map(i => i.failed || 0) }
    ]
  })
}

const renderProviderChart = () => {
  if (!providerChartRef.value) return
  if (!providerChart) providerChart = echarts.init(providerChartRef.value)

  const data = Object.entries(providerDistribution.value).map(([name, value]) => ({ name, value }))
  providerChart.setOption({
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [
      {
        type: 'pie',
        radius: ['45%', '72%'],
        center: ['50%', '45%'],
        data,
        label: { formatter: '{b}: {d}%' }
      }
    ]
  })
}

const refreshAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchOverview(), fetchProviders(), fetchRequestTrend()])
    await nextTick()
    renderTrendChart()
    renderProviderChart()
  } finally {
    loading.value = false
  }
}

watch(range, async () => {
  await fetchRequestTrend()
  renderTrendChart()
})

onMounted(() => {
  refreshAll()
  refreshTimer = window.setInterval(refreshAll, 30000)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
  window.removeEventListener('resize', handleResize)
  trendChart?.dispose()
  providerChart?.dispose()
  trendChart = null
  providerChart = null
})

const handleResize = () => {
  trendChart?.resize()
  providerChart?.resize()
}
</script>

<style scoped>
.usage-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar-card {
  border-radius: 12px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.title {
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.subtitle {
  margin-top: 4px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stats-row,
.charts-row {
  margin: 0;
}

.stat-card .label {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.stat-card .value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.card-header {
  font-weight: 600;
}

.chart {
  height: 320px;
}

@media (max-width: 768px) {
  .toolbar {
    flex-direction: column;
    align-items: flex-start;
  }

  .chart {
    height: 260px;
  }
}
</style>
