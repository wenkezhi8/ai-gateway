<template>
  <div class="dashboard-page">
    <!-- 加载状态 - 骨架屏 -->
    <template v-if="loading">
      <el-row :gutter="20" class="stats-row">
        <el-col :span="6" v-for="i in 4" :key="i">
          <el-card class="stat-card skeleton-card" shadow="hover">
            <div class="stat-content">
              <div class="stat-info">
                <el-skeleton-item variant="text" style="width: 40%; height: 16px; margin-bottom: 12px;" />
                <el-skeleton-item variant="text" style="width: 60%; height: 36px; margin-bottom: 8px;" />
                <el-skeleton-item variant="text" style="width: 30%; height: 14px;" />
              </div>
              <el-skeleton-item variant="circle" style="width: 56px; height: 56px;" />
            </div>
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="20" class="chart-row">
        <el-col :span="16">
          <el-card shadow="hover" class="chart-card">
            <template #header>
              <div class="card-header">
                <el-skeleton-item variant="text" style="width: 100px; height: 20px;" />
                <el-skeleton-item variant="text" style="width: 180px; height: 32px;" />
              </div>
            </template>
            <el-skeleton :rows="8" animated />
          </el-card>
        </el-col>
        <el-col :span="8">
          <el-card shadow="hover" class="chart-card">
            <template #header>
              <el-skeleton-item variant="text" style="width: 100px; height: 20px;" />
            </template>
            <el-skeleton :rows="8" animated />
          </el-card>
        </el-col>
      </el-row>

      <el-row :gutter="20" class="metrics-row">
        <el-col :span="8" v-for="i in 3" :key="i">
          <el-card shadow="hover" class="chart-card">
            <template #header>
              <el-skeleton-item variant="text" style="width: 100px; height: 20px;" />
            </template>
            <el-skeleton :rows="5" animated />
          </el-card>
        </el-col>
      </el-row>
    </template>

    <!-- 加载失败状态 -->
    <template v-else-if="loadError">
      <div class="error-state">
        <div class="error-content">
          <div class="error-icon">
            <el-icon :size="80"><WarningFilled /></el-icon>
          </div>
          <h3 class="error-title">数据加载失败</h3>
          <p class="error-desc">抱歉,无法获取仪表盘数据,请检查网络连接后重试</p>
          <el-button type="primary" size="large" @click="retryFetch" class="retry-btn">
            <el-icon><Refresh /></el-icon>
            重新加载
          </el-button>
        </div>
      </div>
    </template>

    <!-- 空数据状态 -->
    <template v-else-if="isEmptyData">
      <div class="empty-state">
        <div class="empty-content">
          <div class="empty-icon">
            <el-icon :size="80"><DataBoard /></el-icon>
          </div>
          <h3 class="empty-title">暂无数据</h3>
          <p class="empty-desc">系统运行后将在此显示监控数据</p>
          <div class="empty-tips">
            <div class="tip-item">
              <el-icon><InfoFilled /></el-icon>
              <span>配置 AI 服务商后即可开始使用</span>
            </div>
            <div class="tip-item">
              <el-icon><InfoFilled /></el-icon>
              <span>请求将通过网关进行路由和监控</span>
            </div>
          </div>
          <el-button type="primary" size="large" @click="$router.push('/providers')" class="action-btn">
            <el-icon><Setting /></el-icon>
            前往配置
          </el-button>
        </div>
      </div>
    </template>

    <!-- 正常数据展示 -->
    <template v-else>
      <!-- 统计卡片 -->
      <el-row :gutter="20" class="stats-row">
        <el-col :span="6" v-for="stat in stats" :key="stat.title">
          <el-card class="stat-card" shadow="hover">
            <div class="stat-content">
              <div class="stat-info">
                <div class="stat-title">{{ stat.title }}</div>
                <div class="stat-value">{{ stat.value }}</div>
                <div class="stat-change" :class="stat.trend">
                  <el-icon><component :is="stat.trend === 'up' ? 'Top' : 'Bottom'" /></el-icon>
                  {{ stat.change }}
                </div>
              </div>
              <div class="stat-icon" :style="{ background: stat.color + '15' }">
                <el-icon :size="28" :color="stat.color"><component :is="stat.icon" /></el-icon>
              </div>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 图表区域 -->
      <el-row :gutter="20" class="chart-row">
      <el-col :span="16">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="card-header">
              <span>请求趋势</span>
              <el-radio-group v-model="requestTrendRange" size="small" @change="fetchRequestTrend">
                <el-radio-button value="24h">24小时</el-radio-button>
                <el-radio-button value="7d">7天</el-radio-button>
              </el-radio-group>
            </div>
          </template>
          <div ref="lineChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <span>服务商分布</span>
          </template>
          <div ref="pieChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

      <!-- 中间指标 -->
      <el-row :gutter="20" class="metrics-row">
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <div class="card-header">
              <span>缓存性能</span>
              <el-tag size="small" type="success">{{ cacheHitRate }}% 命中率</el-tag>
            </div>
          </template>
          <div ref="cacheChartRef" class="chart-container-sm"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <span>Token使用量</span>
          </template>
          <div ref="tokenChartRef" class="chart-container-sm"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover" class="chart-card">
          <template #header>
            <span>模型使用排行</span>
          </template>
          <div class="model-ranking" v-if="modelRanking.length > 0">
            <div v-for="(item, index) in modelRanking" :key="item.name" class="rank-item">
              <span class="rank-num" :class="'top-' + (index + 1)">{{ index + 1 }}</span>
              <span class="model-name">{{ item.name }}</span>
              <el-progress :percentage="item.percentage" :show-text="false" :stroke-width="8" />
              <span class="model-count">{{ formatNumber(item.tokens) }}</span>
            </div>
          </div>
          <el-empty v-else description="暂无模型使用数据" :image-size="80" />
        </el-card>
      </el-col>
    </el-row>

      <!-- 实时请求和告警 -->
      <el-row :gutter="20" class="table-row">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>实时指标</span>
              <el-tag type="success" size="small">
                <el-icon class="pulse"><Connection /></el-icon>
                {{ realtimeData.requests_per_minute || 0 }} 请求/分钟
              </el-tag>
            </div>
          </template>
          <div class="realtime-stats">
            <div class="realtime-item">
              <el-icon><User /></el-icon>
              <span class="value">{{ realtimeData.active_connections || 0 }}</span>
              <span class="label">活跃连接</span>
            </div>
            <div class="realtime-item">
              <el-icon><Coin /></el-icon>
              <span class="value">{{ formatNumber(realtimeData.tokens_per_minute || 0) }}</span>
              <span class="label">Token/分钟</span>
            </div>
            <div class="realtime-item">
              <el-icon><Timer /></el-icon>
              <span class="value">{{ realtimeData.avg_latency_ms || 0 }}ms</span>
              <span class="label">平均延迟</span>
            </div>
            <div class="realtime-item" :class="{ 'has-error': realtimeData.error_rate > 1 }">
              <el-icon><Warning /></el-icon>
              <span class="value">{{ formatPercent(realtimeData.error_rate || 0) }}%</span>
              <span class="label">错误率</span>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="card-header">
              <span>最近错误</span>
              <el-button type="primary" link @click="$router.push('/alerts')">
                查看全部
              </el-button>
            </div>
          </template>
          <el-table :data="recentErrors" stripe size="small" max-height="200" v-if="recentErrors.length > 0">
            <el-table-column prop="timestamp" label="时间" width="100">
              <template #default="{ row }">
                {{ formatTime(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="provider" label="服务商" width="90">
              <template #default="{ row }">
                <el-tag size="small">{{ row.provider }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="error" label="错误" />
            <el-table-column prop="count" label="次数" width="60" />
          </el-table>
          <div v-else class="no-errors">
            <el-icon :size="48" color="#34C759"><CircleCheckFilled /></el-icon>
            <p>运行正常,暂无错误</p>
          </div>
        </el-card>
      </el-col>
      </el-row>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import * as echarts from 'echarts'
import {
  getOverview,
  getRequestTrend,
  getProviders,
  getRealtime,
  getCacheStats,
  type OverviewData,
  type RealtimeData
} from '@/api/metrics'

const loading = ref(false)
const loadError = ref(false)
const requestTrendRange = ref('24h')
const overviewData = ref<OverviewData | null>(null)
const realtimeData = ref<RealtimeData>({
  timestamp: '',
  active_connections: 0,
  requests_per_minute: 0,
  tokens_per_minute: 0,
  avg_latency_ms: 0,
  error_rate: 0,
  top_models: [],
  recent_errors: []
})

// 判断是否为空数据状态
const isEmptyData = computed(() => {
  if (!overviewData.value) return true
  const data = overviewData.value
  // 如果所有关键指标都是0,则认为是空数据
  return (
    data.requests_today === 0 &&
    data.success_rate === 0 &&
    data.active_accounts === 0 &&
    (!data.top_models || data.top_models.length === 0)
  )
})

const cacheHitRate = computed(() => overviewData.value?.cache_hit_rate?.toFixed(1) || '0')

const stats = computed(() => [
  {
    title: '今日请求',
    value: formatNumber(overviewData.value?.requests_today || 0),
    change: '+12.5%',
    trend: 'up',
    icon: 'TrendCharts',
    color: '#007AFF'
  },
  {
    title: '成功率',
    value: `${(overviewData.value?.success_rate || 0).toFixed(1)}%`,
    change: '+0.2%',
    trend: 'up',
    icon: 'CircleCheck',
    color: '#34C759'
  },
  {
    title: '平均延迟',
    value: `${overviewData.value?.avg_latency_ms || 0}ms`,
    change: '-8ms',
    trend: 'up',
    icon: 'Timer',
    color: '#FF9500'
  },
  {
    title: '活跃账号',
    value: `${overviewData.value?.active_accounts || 0}`,
    change: `+${overviewData.value?.active_providers || 0}服务商`,
    trend: 'up',
    icon: 'User',
    color: '#AF52DE'
  }
])

const modelRanking = computed(() => {
  const models = overviewData.value?.top_models || []
  const maxTokens = Math.max(...models.map(m => m.tokens), 1)
  return models.map(m => ({
    ...m,
    percentage: Math.round((m.tokens / maxTokens) * 100)
  }))
})

const recentErrors = computed(() => realtimeData.value?.recent_errors || [])

const formatNumber = (num: number): string => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatPercent = (num: number): string => {
  // 保留最多2位小数，去掉尾部的0
  return num.toFixed(2).replace(/\.?0+$/, '')
}

const formatTime = (timestamp: string): string => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

// 图表相关
const lineChartRef = ref<HTMLElement>()
const pieChartRef = ref<HTMLElement>()
const cacheChartRef = ref<HTMLElement>()
const tokenChartRef = ref<HTMLElement>()

let lineChart: echarts.ECharts | null = null
let pieChart: echarts.ECharts | null = null
let cacheChart: echarts.ECharts | null = null
let tokenChart: echarts.ECharts | null = null
let realtimeTimer: ReturnType<typeof setInterval> | null = null
let themeObserver: MutationObserver | null = null

const getChartTheme = () => {
  const isDark = document.documentElement.getAttribute('data-theme') === 'dark'
  return {
    textColor: isDark ? '#AEAEB2' : '#6E6E73',
    axisLineColor: isDark ? '#3A3A3C' : '#E8E8ED',
    splitLineColor: isDark ? '#2C2C2E' : '#F5F5F7'
  }
}

const initLineChart = (data: any[] = []) => {
  if (!lineChartRef.value) return

  if (!lineChart) {
    lineChart = echarts.init(lineChartRef.value)
  }
  const theme = getChartTheme()

  const hasData = data && data.length > 0
  const timestamps = hasData ? data.map(d => {
    const date = new Date(d.timestamp)
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }) : []
  const requests = hasData ? data.map(d => d.requests) : []
  const successRates = hasData ? data.map(d => {
    const rate = d.requests > 0 ? (d.success / d.requests) * 100 : 0
    return rate.toFixed(1)
  }) : []

  const defaultTimestamps = ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00']
  const defaultRequests = [0, 0, 0, 0, 0, 0]
  const defaultSuccessRates = [0, 0, 0, 0, 0, 0]

  const option = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(0,0,0,0.7)',
      borderColor: 'transparent',
      textStyle: { color: '#fff' }
    },
    legend: {
      data: ['请求数', '成功率'],
      textStyle: { color: theme.textColor },
      top: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: timestamps.length > 0 ? timestamps : defaultTimestamps,
      axisLine: { lineStyle: { color: theme.axisLineColor } },
      axisLabel: { color: theme.textColor }
    },
    yAxis: [
      {
        type: 'value',
        name: '请求数',
        axisLine: { lineStyle: { color: theme.axisLineColor } },
        axisLabel: { color: theme.textColor },
        splitLine: { lineStyle: { color: theme.splitLineColor } }
      },
      {
        type: 'value',
        name: '成功率',
        min: 0,
        max: 100,
        axisLine: { lineStyle: { color: theme.axisLineColor } },
        axisLabel: { color: theme.textColor, formatter: '{value}%' },
        splitLine: { show: false }
      }
    ],
    series: [
      {
        name: '请求数',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { width: 3, color: '#007AFF' },
        itemStyle: { color: '#007AFF' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(0, 122, 255, 0.3)' },
            { offset: 1, color: 'rgba(0, 122, 255, 0.05)' }
          ])
        },
        data: requests.length > 0 ? requests : defaultRequests
      },
      {
        name: '成功率',
        type: 'line',
        smooth: true,
        symbol: 'circle',
        symbolSize: 4,
        yAxisIndex: 1,
        lineStyle: { width: 2, color: '#34C759' },
        itemStyle: { color: '#34C759' },
        data: successRates.length > 0 ? successRates : defaultSuccessRates
      }
    ]
  }

  lineChart.setOption(option)
  lineChart.resize()
}

const initPieChart = (distribution: Record<string, number> = {}) => {
  if (!pieChartRef.value) return

  if (!pieChart) {
    pieChart = echarts.init(pieChartRef.value)
  }

  const theme = getChartTheme()
  const colors: Record<string, string> = {
    openai: '#007AFF',
    anthropic: '#AF52DE',
    azure: '#00C7BE',
    google: '#FF9500',
    volcengine: '#FF3B30',
    qwen: '#FF6A00',
    ernie: '#2932E1',
    zhipu: '#3657ED',
    hunyuan: '#00A3FF',
    moonshot: '#1A1A1A',
    minimax: '#615CED',
    baichuan: '#0066FF',
    spark: '#E60012',
    deepseek: '#4D6BFE'
  }

  const hasData = distribution && Object.keys(distribution).length > 0
  const pieData = hasData
    ? Object.entries(distribution).map(([name, value]) => ({
        value,
        name: name.charAt(0).toUpperCase() + name.slice(1),
        itemStyle: { color: colors[name.toLowerCase()] || '#6B7280' }
      }))
    : [
        { value: 1, name: '暂无数据', itemStyle: { color: theme.axisLineColor } }
      ]

  const option = {
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(0,0,0,0.7)',
      borderColor: 'transparent',
      textStyle: { color: '#fff' },
      formatter: '{b}: {d}%'
    },
    legend: {
      orient: 'vertical',
      right: '5%',
      top: 'center',
      textStyle: { color: theme.textColor }
    },
    series: [
      {
        name: '服务商',
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['35%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: 'transparent',
          borderWidth: 2
        },
        label: { show: false },
        emphasis: {
          label: { show: true, fontSize: 14, fontWeight: 'bold' }
        },
        labelLine: { show: false },
        data: pieData
      }
    ]
  }

  pieChart.setOption(option)
  pieChart.resize()
}

const initCacheChart = (cacheStats?: any) => {
  if (!cacheChartRef.value) return

  if (!cacheChart) {
    cacheChart = echarts.init(cacheChartRef.value)
  }

  const theme = getChartTheme()
  
  // Use real data if available
  let cacheData = [0, 0, 0, 0, 0]
  if (cacheStats) {
    cacheData = [
      (cacheStats.request_cache?.hit_rate || 0),
      (cacheStats.context_cache?.hit_rate || 0),
      (cacheStats.route_cache?.hit_rate || 0),
      (cacheStats.usage_cache?.hit_rate || 0),
      (cacheStats.response_cache?.hit_rate || 0),
    ]
  }

  const option = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(0,0,0,0.7)',
      borderColor: 'transparent',
      textStyle: { color: '#fff' }
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: ['请求缓存', '上下文缓存', '路由缓存', '用量缓存', '响应缓存'],
      axisLine: { lineStyle: { color: theme.axisLineColor } },
      axisLabel: { color: theme.textColor, fontSize: 11 }
    },
    yAxis: {
      type: 'value',
      max: 100,
      axisLine: { lineStyle: { color: theme.axisLineColor } },
      axisLabel: { color: theme.textColor, formatter: '{value}%' },
      splitLine: { lineStyle: { color: theme.splitLineColor } }
    },
    series: [
      {
        data: cacheData,
        type: 'bar',
        barWidth: '50%',
        itemStyle: {
          borderRadius: [4, 4, 0, 0],
          color: (params: any) => {
            const value = params.value
            if (value >= 70) return '#34C759'
            if (value >= 40) return '#FF9500'
            return '#FF3B30'
          }
        }
      }
    ]
  }

  cacheChart.setOption(option)
  cacheChart.resize()
}

const initTokenChart = (tokenData?: { prompt: number[]; completion: number[] }) => {
  if (!tokenChartRef.value) return

  if (!tokenChart) {
    tokenChart = echarts.init(tokenChartRef.value)
  }

  const theme = getChartTheme()
  
  // Use real data or generate from overview
  const promptTokens = tokenData?.prompt || [0, 0, 0, 0, 0, 0, 0]
  const completionTokens = tokenData?.completion || [0, 0, 0, 0, 0, 0, 0]

  const option = {
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(0,0,0,0.7)',
      borderColor: 'transparent',
      textStyle: { color: '#fff' }
    },
    legend: {
      data: ['Prompt Tokens', 'Completion Tokens'],
      textStyle: { color: theme.textColor },
      top: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      top: '15%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: ['周一', '周二', '周三', '周四', '周五', '周六', '周日'],
      axisLine: { lineStyle: { color: theme.axisLineColor } },
      axisLabel: { color: theme.textColor }
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: theme.axisLineColor } },
      axisLabel: { color: theme.textColor, formatter: '{value}K' },
      splitLine: { lineStyle: { color: theme.splitLineColor } }
    },
    series: [
      {
        name: 'Prompt Tokens',
        type: 'bar',
        stack: 'total',
        barWidth: '40%',
        itemStyle: { borderRadius: [0, 0, 0, 0], color: '#007AFF' },
        data: promptTokens
      },
      {
        name: 'Completion Tokens',
        type: 'bar',
        stack: 'total',
        itemStyle: { borderRadius: [4, 4, 0, 0], color: '#34C759' },
        data: completionTokens
      }
    ]
  }

  tokenChart.setOption(option)
  tokenChart.resize()
}

// 数据获取
const fetchOverview = async () => {
  try {
    const res = await getOverview()
    overviewData.value = (res as any)?.data || res
  } catch (error) {
    console.warn('Overview data not available', error)
    throw error
  }
}

const fetchRequestTrend = async () => {
  try {
    const period = requestTrendRange.value === '7d' ? '7d' : '24h'
    const interval = period === '7d' ? 'day' : 'hour'
    const res = await getRequestTrend({ period, interval })
    const data = (res as any)?.data || res
    initLineChart(Array.isArray(data) ? data : [])
  } catch (error) {
    console.warn('Request trend data not available', error)
    initLineChart([])
  }
}

const fetchProviders = async () => {
  try {
    const res = await getProviders()
    const data = (res as any)?.data || res
    initPieChart(data?.distribution || {})
  } catch (error) {
    console.warn('Provider data not available', error)
    initPieChart({})
  }
}

const fetchCacheStats = async () => {
  try {
    const res = await getCacheStats()
    const data = (res as any)?.data || res
    initCacheChart(data)
  } catch (error) {
    console.warn('Cache stats not available', error)
    initCacheChart()
  }
}

const fetchRealtime = async () => {
  try {
    const res = await getRealtime()
    realtimeData.value = (res as any)?.data || res
  } catch (error) {
    console.warn('Realtime data not available', error)
  }
}

const fetchAllData = async () => {
  loading.value = true
  loadError.value = false
  try {
    await Promise.all([
      fetchOverview(),
      fetchRequestTrend(),
      fetchProviders(),
      fetchRealtime(),
      fetchCacheStats()
    ])
    // Initialize token chart with overview data
    if (overviewData.value) {
      initTokenChart()
    }
  } catch (error) {
    loadError.value = true
  } finally {
    loading.value = false
  }
}

const retryFetch = async () => {
  try {
    await fetchAllData()
    if (!loadError.value && !isEmptyData.value) {
      await nextTick()
      fetchRequestTrend()
      fetchProviders()
      fetchCacheStats()
    }
  } catch (error) {
    console.error('重试数据加载失败:', error)
    loadError.value = true
  }
}

const handleResize = () => {
  lineChart?.resize()
  pieChart?.resize()
  cacheChart?.resize()
  tokenChart?.resize()
}

const updateChartsTheme = () => {
  initLineChart([])
  initPieChart({})
  initCacheChart()
  initTokenChart()
  fetchRequestTrend()
  fetchProviders()
  fetchCacheStats()
}

onMounted(async () => {
  try {
    await nextTick()

    // Fetch data and initialize charts
    await fetchAllData()

    // Initialize charts with empty data first (will be updated by fetchAllData)
    initLineChart([])
    initPieChart({})
    initCacheChart()
    initTokenChart()

    if (typeof window !== 'undefined') {
      window.addEventListener('resize', handleResize)
    }

    if (typeof document !== 'undefined') {
      themeObserver = new MutationObserver(() => {
        updateChartsTheme()
      })
      themeObserver.observe(document.documentElement, {
        attributes: true,
        attributeFilter: ['data-theme']
      })
    }

    // Refresh realtime data every 10 seconds
    realtimeTimer = setInterval(fetchRealtime, 10000)
    
    // Refresh overview data every 5 seconds
    setInterval(() => {
      fetchOverview()
      fetchCacheStats()
    }, 5000)
  } catch (error) {
    console.error('Dashboard初始化失败:', error)
    loadError.value = true
  }
})

onUnmounted(() => {
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
  }
  themeObserver?.disconnect()
  lineChart?.dispose()
  pieChart?.dispose()
  cacheChart?.dispose()
  tokenChart?.dispose()
  if (realtimeTimer) {
    clearInterval(realtimeTimer)
  }
})

// 监听时间范围变化
watch(requestTrendRange, () => {
  fetchRequestTrend()
})
</script>

<style scoped lang="scss">
.dashboard-page {
  .stats-row,
  .chart-row,
  .metrics-row,
  .table-row {
    margin-bottom: var(--spacing-xl);
  }

  // 骨架屏样式
  .skeleton-card {
    .stat-content {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
    }
  }

  // 错误状态样式
  .error-state {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 200px);
    padding: var(--spacing-2xl);

    .error-content {
      text-align: center;
      max-width: 400px;

      .error-icon {
        margin-bottom: var(--spacing-xl);
        color: #FF3B30;
        animation: shake 0.5s ease-in-out;
      }

      .error-title {
        font-size: var(--font-size-2xl);
        font-weight: var(--font-weight-bold);
        color: var(--text-primary);
        margin-bottom: var(--spacing-md);
      }

      .error-desc {
        font-size: var(--font-size-md);
        color: var(--text-secondary);
        margin-bottom: var(--spacing-xl);
        line-height: 1.6;
      }

      .retry-btn {
        padding: 12px 32px;
        border-radius: var(--border-radius-lg);
        font-weight: var(--font-weight-medium);
      }
    }
  }

  // 空数据状态样式
  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 200px);
    padding: var(--spacing-2xl);

    .empty-content {
      text-align: center;
      max-width: 500px;

      .empty-icon {
        margin-bottom: var(--spacing-xl);
        color: var(--color-primary);
        animation: float 3s ease-in-out infinite;
      }

      .empty-title {
        font-size: var(--font-size-2xl);
        font-weight: var(--font-weight-bold);
        color: var(--text-primary);
        margin-bottom: var(--spacing-md);
      }

      .empty-desc {
        font-size: var(--font-size-md);
        color: var(--text-secondary);
        margin-bottom: var(--spacing-xl);
        line-height: 1.6;
      }

      .empty-tips {
        background: rgba(255, 255, 255, 0.72);
        backdrop-filter: blur(20px);
        border-radius: var(--border-radius-lg);
        padding: var(--spacing-lg);
        margin-bottom: var(--spacing-xl);

        .tip-item {
          display: flex;
          align-items: center;
          gap: var(--spacing-sm);
          padding: var(--spacing-sm) 0;
          color: var(--text-secondary);
          font-size: var(--font-size-sm);

          .el-icon {
            color: var(--color-primary);
          }
        }
      }

      .action-btn {
        padding: 12px 32px;
        border-radius: var(--border-radius-lg);
        font-weight: var(--font-weight-medium);
      }
    }
  }

  .stat-card {
    border-radius: var(--border-radius-lg);
    border: none;

    .stat-content {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;

      .stat-info {
        .stat-title {
          font-size: var(--font-size-md);
          color: var(--text-secondary);
          font-weight: var(--font-weight-medium);
        }

        .stat-value {
          font-size: var(--font-size-4xl);
          font-weight: var(--font-weight-bold);
          margin: var(--spacing-sm) 0;
          color: var(--text-primary);
        }

        .stat-change {
          font-size: var(--font-size-sm);
          display: flex;
          align-items: center;
          gap: 4px;
          font-weight: var(--font-weight-medium);

          &.up {
            color: var(--color-success);
          }

          &.down {
            color: var(--color-danger);
          }
        }
      }

      .stat-icon {
        width: 56px;
        height: 56px;
        border-radius: var(--border-radius-lg);
        display: flex;
        align-items: center;
        justify-content: center;
      }
    }
  }

  .chart-card {
    border-radius: var(--border-radius-lg);
    border: none;

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
  }

  .chart-container {
    height: 300px;
  }

  .chart-container-sm {
    height: 200px;
  }

  .model-ranking {
    padding: var(--spacing-sm) 0;

    .rank-item {
      display: flex;
      align-items: center;
      gap: var(--spacing-md);
      padding: var(--spacing-sm) 0;

      .rank-num {
        width: 24px;
        height: 24px;
        border-radius: var(--border-radius-sm);
        background: var(--bg-tertiary);
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: var(--font-size-sm);
        font-weight: var(--font-weight-semibold);
        color: var(--text-secondary);

        &.top-1 {
          background: linear-gradient(135deg, #FFD700, #FFA500);
          color: #fff;
        }

        &.top-2 {
          background: linear-gradient(135deg, #C0C0C0, #A0A0A0);
          color: #fff;
        }

        &.top-3 {
          background: linear-gradient(135deg, #CD7F32, #A0522D);
          color: #fff;
        }
      }

      .model-name {
        width: 120px;
        font-weight: var(--font-weight-medium);
      }

      .el-progress {
        flex: 1;
      }

      .model-count {
        width: 60px;
        text-align: right;
        color: var(--text-tertiary);
        font-size: var(--font-size-sm);
      }
    }
  }

  .realtime-stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--spacing-lg);
    padding: var(--spacing-lg) 0;

    .realtime-item {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: var(--spacing-xs);
      padding: var(--spacing-md);
      background: var(--bg-secondary);
      border-radius: var(--border-radius-lg);
      transition: all var(--transition-fast);

      &:hover {
        background: var(--bg-tertiary);
        transform: translateY(-2px);
      }

      .el-icon {
        font-size: 28px;
        color: var(--color-primary);
        margin-bottom: var(--spacing-xs);
      }

      .value {
        font-size: var(--font-size-2xl);
        font-weight: var(--font-weight-bold);
        color: var(--text-primary);
        line-height: 1.2;
      }

      .label {
        font-size: var(--font-size-sm);
        color: var(--text-tertiary);
        font-weight: var(--font-weight-medium);
      }

      &.has-error {
        .el-icon {
          color: var(--color-danger);
        }
        .value {
          color: var(--color-danger);
        }
        background: rgba(255, 59, 48, 0.08);
      }
    }
  }

  .pulse {
    animation: pulse 2s infinite;
  }

  .no-errors {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: var(--spacing-2xl);
    color: var(--text-secondary);

    p {
      margin-top: var(--spacing-md);
      font-size: var(--font-size-md);
    }
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.5; }
  }

  @keyframes shake {
    0%, 100% { transform: translateX(0); }
    25% { transform: translateX(-10px); }
    75% { transform: translateX(10px); }
  }

  @keyframes float {
    0%, 100% { transform: translateY(0); }
    50% { transform: translateY(-10px); }
  }
}
</style>
