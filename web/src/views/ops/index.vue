<template>
  <div class="ops-page">
    <el-row :gutter="20" class="header-row">
      <el-col :span="18">
        <h2 class="page-title">运维监控</h2>
        <p class="page-desc">系统运行状态与性能监控</p>
      </el-col>
      <el-col :span="6" class="header-actions">
        <el-button @click="refreshAll" :loading="loading" type="primary">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button @click="exportMetrics">
          <el-icon><Download /></el-icon>
          导出
        </el-button>
      </el-col>
    </el-row>

    <el-row :gutter="20" class="status-row">
      <el-col :span="6">
        <el-card class="status-card" :class="{ 'status-healthy': systemHealthy }">
          <div class="status-icon">
            <el-icon :size="40" color="#34C759"><CircleCheckFilled /></el-icon>
          </div>
          <div class="status-info">
            <div class="status-label">系统状态</div>
            <div class="status-value">{{ systemHealthy ? '健康' : '异常' }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-icon">
            <el-icon :size="40" color="#007AFF"><Monitor /></el-icon>
          </div>
          <div class="status-info">
            <div class="status-label">运行时间</div>
            <div class="status-value">{{ systemInfo.uptime || '-' }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-icon">
            <el-icon :size="40" color="#FF9500"><Cpu /></el-icon>
          </div>
          <div class="status-info">
            <div class="status-label">CPU 核数</div>
            <div class="status-value">{{ systemInfo.cpu_count || '-' }}</div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="status-card">
          <div class="status-icon">
            <el-icon :size="40" color="#AF52DE"><Coin /></el-icon>
          </div>
          <div class="status-info">
            <div class="status-label">内存使用</div>
            <div class="status-value">{{ systemInfo.memory_used_mb || 0 }} / {{ systemInfo.memory_mb || 0 }} MB</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <el-col :span="16">
        <el-card class="section-card">
          <template #header>
            <div class="card-header">
              <span>服务状态</span>
              <el-tag type="success" size="small">{{ healthyServicesCount }}/{{ services.length }} 正常</el-tag>
            </div>
          </template>
          <el-table :data="services" stripe>
            <el-table-column prop="name" label="服务名称" width="150">
              <template #default="{ row }">
                <div class="service-name">
                  <el-icon><Service /></el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="100">
              <template #default="{ row }">
                <el-tag :type="row.status === 'healthy' ? 'success' : 'danger'" size="small">
                  {{ row.status === 'healthy' ? '正常' : '异常' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="latency_ms" label="延迟" width="100">
              <template #default="{ row }">
                <span :class="{ 'latency-high': row.latency_ms > 100 }">{{ row.latency_ms || 0 }} ms</span>
              </template>
            </el-table-column>
            <el-table-column prop="error_count" label="错误次数" width="100">
              <template #default="{ row }">
                <span :class="{ 'error-count': row.error_count > 0 }">{{ row.error_count || 0 }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="description" label="描述" />
            <el-table-column label="操作" width="80">
              <template #default="{ row }">
                <el-button type="primary" link size="small" @click="checkService(row)">检查</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card class="section-card">
          <template #header>
            <div class="card-header">
              <span>健康检查</span>
            </div>
          </template>
          <div class="health-checks">
            <div v-for="check in healthChecks" :key="check.component" class="check-item">
              <div class="check-status" :class="check.status"></div>
              <div class="check-info">
                <div class="check-name">{{ check.component }}</div>
                <div class="check-message">{{ check.message }}</div>
              </div>
              <div class="check-latency">{{ check.latency_ms }} ms</div>
            </div>
          </div>
        </el-card>

        <el-card class="section-card" style="margin-top: 20px;">
          <template #header>
            <div class="card-header">
              <span>服务商状态</span>
            </div>
          </template>
          <div class="provider-status">
            <div v-for="provider in providerHealth" :key="provider.name" class="provider-item">
              <div class="provider-name">{{ provider.name }}</div>
              <div class="provider-info">
                <el-tag :type="provider.status === 'healthy' ? 'success' : 'danger'" size="small">
                  {{ provider.status === 'healthy' ? '正常' : '异常' }}
                </el-tag>
                <span class="provider-latency">{{ provider.latency_ms }} ms</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="12">
        <el-card class="section-card">
          <template #header>
            <div class="card-header">
              <span>性能指标</span>
              <el-tag size="small">实时</el-tag>
            </div>
          </template>
          <div class="performance-metrics">
            <div class="metric-item">
              <div class="metric-label">QPS</div>
              <div class="metric-value">{{ (performance.qps || 0).toFixed(2) }}</div>
            </div>
            <div class="metric-item">
              <div class="metric-label">平均延迟</div>
              <div class="metric-value">{{ performance.avg_latency_ms || 0 }} ms</div>
            </div>
            <div class="metric-item">
              <div class="metric-label">P99 延迟</div>
              <div class="metric-value">{{ performance.p99_latency_ms || 0 }} ms</div>
            </div>
            <div class="metric-item">
              <div class="metric-label">错误率</div>
              <div class="metric-value" :class="{ 'error-rate': performance.error_rate > 1 }">
                {{ (performance.error_rate || 0).toFixed(2) }}%
              </div>
            </div>
            <div class="metric-item">
              <div class="metric-label">活跃连接</div>
              <div class="metric-value">{{ performance.active_connections || 0 }}</div>
            </div>
            <div class="metric-item">
              <div class="metric-label">总请求</div>
              <div class="metric-value">{{ formatNumber(performance.total_requests || 0) }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card class="section-card">
          <template #header>
            <div class="card-header">
              <span>资源使用</span>
              <el-button type="primary" link size="small" @click="fetchResources">刷新</el-button>
            </div>
          </template>
          <div class="resource-usage" v-if="resources.memory">
            <div class="resource-item">
              <div class="resource-header">
                <span>内存分配</span>
                <span>{{ resources.memory.alloc_mb }} MB</span>
              </div>
              <el-progress :percentage="getMemoryPercent()" :color="getProgressColor(getMemoryPercent())" />
            </div>
            <div class="resource-item">
              <div class="resource-header">
                <span>堆内存</span>
                <span>{{ resources.memory.heap_alloc_mb }} / {{ resources.memory.heap_sys_mb }} MB</span>
              </div>
              <el-progress :percentage="getHeapPercent()" :color="getProgressColor(getHeapPercent())" />
            </div>
            <div class="resource-info">
              <div class="info-item">
                <el-icon><Cpu /></el-icon>
                <span>Goroutines: {{ resources.goroutines || 0 }}</span>
              </div>
              <div class="info-item">
                <el-icon><Timer /></el-icon>
                <span>GC 次数: {{ resources.gc?.num_gc || 0 }}</span>
              </div>
              <div class="info-item">
                <el-icon><Monitor /></el-icon>
                <span>CPU: {{ resources.cpu?.count || 0 }} 核</span>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px;">
      <el-col :span="24">
        <el-card class="section-card">
          <template #header>
            <div class="card-header">
              <span>事件日志</span>
              <div class="header-filters">
                <el-select v-model="eventLevel" size="small" placeholder="级别" clearable @change="fetchEvents" style="width: 100px; margin-right: 10px;">
                  <el-option label="全部" value="" />
                  <el-option label="信息" value="info" />
                  <el-option label="警告" value="warning" />
                  <el-option label="错误" value="error" />
                </el-select>
              </div>
            </div>
          </template>
          <el-table :data="events" stripe max-height="300">
            <el-table-column prop="timestamp" label="时间" width="180">
              <template #default="{ row }">
                {{ formatTime(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="level" label="级别" width="80">
              <template #default="{ row }">
                <el-tag :type="getLevelType(row.level)" size="small">{{ row.level }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="source" label="来源" width="120" />
            <el-table-column prop="message" label="消息" />
            <el-table-column prop="details" label="详情" width="200">
              <template #default="{ row }">
                <span class="event-details">{{ row.details || '-' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { request } from '@/api/request'

interface SystemInfo {
  hostname: string
  os: string
  arch: string
  go_version: string
  cpu_count: number
  memory_mb: number
  memory_used_mb: number
  uptime: string
  start_time: string
}

interface Service {
  name: string
  status: string
  latency_ms: number
  error_count: number
  description: string
}

interface HealthCheck {
  component: string
  status: string
  message: string
  latency_ms: number
}

interface Performance {
  qps: number
  avg_latency_ms: number
  p99_latency_ms: number
  error_rate: number
  active_connections: number
  total_requests: number
}

interface Event {
  timestamp: string
  level: string
  source: string
  message: string
  details: string
}

const loading = ref(false)
const systemInfo = ref<SystemInfo>({} as SystemInfo)
const services = ref<Service[]>([])
const healthChecks = ref<HealthCheck[]>([])
const performance = ref<Performance>({} as Performance)
const events = ref<Event[]>([])
const eventLevel = ref('')
const providerHealth = ref<any[]>([])
const resources = ref<any>({})

let refreshTimer: ReturnType<typeof setInterval> | null = null

const systemHealthy = computed(() => {
  return services.value.every(s => s.status === 'healthy')
})

const healthyServicesCount = computed(() => {
  return services.value.filter(s => s.status === 'healthy').length
})

const formatNumber = (num: number): string => {
  if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M'
  if (num >= 1000) return (num / 1000).toFixed(1) + 'K'
  return num.toString()
}

const formatTime = (timestamp: string): string => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN')
}

const getLevelType = (level: string): string => {
  switch (level) {
    case 'error': return 'danger'
    case 'warning': return 'warning'
    default: return 'info'
  }
}

const getMemoryPercent = (): number => {
  if (!resources.value.memory) return 0
  const { alloc_mb, sys_mb } = resources.value.memory
  if (!sys_mb) return 0
  return Math.min(100, Math.round((alloc_mb / sys_mb) * 100))
}

const getHeapPercent = (): number => {
  if (!resources.value.memory) return 0
  const { heap_alloc_mb, heap_sys_mb } = resources.value.memory
  if (!heap_sys_mb) return 0
  return Math.min(100, Math.round((heap_alloc_mb / heap_sys_mb) * 100))
}

const getProgressColor = (percent: number): string => {
  if (percent >= 80) return '#FF3B30'
  if (percent >= 60) return '#FF9500'
  return '#34C759'
}

const fetchSystemInfo = async () => {
  try {
    const res: any = await request.get('/admin/ops/system')
    systemInfo.value = res?.data || {}
  } catch (e) {
    console.warn('Failed to fetch system info:', e)
  }
}

const fetchServices = async () => {
  try {
    const res: any = await request.get('/admin/ops/services')
    services.value = res?.data || []
  } catch (e) {
    console.warn('Failed to fetch services:', e)
  }
}

const fetchHealthChecks = async () => {
  try {
    const res: any = await request.get('/admin/ops/health-checks')
    healthChecks.value = res?.data || []
  } catch (e) {
    console.warn('Failed to fetch health checks:', e)
  }
}

const fetchPerformance = async () => {
  try {
    const res: any = await request.get('/admin/ops/performance')
    performance.value = res?.data || {}
  } catch (e) {
    console.warn('Failed to fetch performance:', e)
  }
}

const fetchEvents = async () => {
  try {
    const params = eventLevel.value ? { level: eventLevel.value } : {}
    const res: any = await request.get('/admin/ops/events', { params })
    events.value = res?.data || []
  } catch (e) {
    console.warn('Failed to fetch events:', e)
  }
}

const fetchProviderHealth = async () => {
  try {
    const res: any = await request.get('/admin/ops/providers/health')
    providerHealth.value = res?.data || []
  } catch (e) {
    console.warn('Failed to fetch provider health:', e)
  }
}

const fetchResources = async () => {
  try {
    const res: any = await request.get('/admin/ops/resources')
    resources.value = res?.data || {}
  } catch (e) {
    console.warn('Failed to fetch resources:', e)
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    await Promise.all([
      fetchSystemInfo(),
      fetchServices(),
      fetchHealthChecks(),
      fetchPerformance(),
      fetchEvents(),
      fetchProviderHealth(),
      fetchResources()
    ])
  } finally {
    loading.value = false
  }
}

const checkService = async (service: Service) => {
  try {
    await request.post(`/admin/ops/services/${service.name}/check`)
    await fetchServices()
  } catch (e) {
    console.warn('Failed to check service:', e)
  }
}

const exportMetrics = async () => {
  try {
    const res: any = await request.get('/admin/ops/export')
    const blob = new Blob([JSON.stringify(res, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `ops-metrics-${new Date().toISOString().slice(0, 10)}.json`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    console.warn('Failed to export metrics:', e)
  }
}

onMounted(() => {
  refreshAll()
  refreshTimer = setInterval(() => {
    fetchPerformance()
    fetchResources()
  }, 5000)
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped lang="scss">
.ops-page {
  .header-row {
    margin-bottom: 20px;
    
    .page-title {
      margin: 0;
      font-size: 24px;
      font-weight: 600;
    }
    
    .page-desc {
      margin: 5px 0 0;
      color: var(--text-secondary);
      font-size: 14px;
    }
    
    .header-actions {
      display: flex;
      justify-content: flex-end;
      align-items: center;
      gap: 10px;
    }
  }
  
  .status-row {
    margin-bottom: 20px;
  }
  
  .status-card {
    display: flex;
    align-items: center;
    padding: 16px;
    border-radius: 12px;
    
    &.status-healthy {
      border-left: 4px solid #34C759;
    }
    
    .status-icon {
      margin-right: 16px;
    }
    
    .status-info {
      .status-label {
        font-size: 13px;
        color: var(--text-secondary);
        margin-bottom: 4px;
      }
      
      .status-value {
        font-size: 20px;
        font-weight: 600;
      }
    }
  }
  
  .section-card {
    border-radius: 12px;
    
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }
  }
  
  .service-name {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .latency-high {
    color: #FF9500;
    font-weight: 500;
  }
  
  .error-count {
    color: #FF3B30;
    font-weight: 500;
  }
  
  .health-checks {
    .check-item {
      display: flex;
      align-items: center;
      padding: 12px 0;
      border-bottom: 1px solid var(--border-light);
      
      &:last-child {
        border-bottom: none;
      }
      
      .check-status {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        margin-right: 12px;
        
        &.healthy {
          background: #34C759;
        }
        
        &.warning {
          background: #FF9500;
        }
        
        &.error {
          background: #FF3B30;
        }
      }
      
      .check-info {
        flex: 1;
        
        .check-name {
          font-weight: 500;
          margin-bottom: 2px;
        }
        
        .check-message {
          font-size: 12px;
          color: var(--text-secondary);
        }
      }
      
      .check-latency {
        font-size: 13px;
        color: var(--text-tertiary);
      }
    }
  }
  
  .provider-status {
    .provider-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 10px 0;
      border-bottom: 1px solid var(--border-light);
      
      &:last-child {
        border-bottom: none;
      }
      
      .provider-name {
        font-weight: 500;
        text-transform: capitalize;
      }
      
      .provider-info {
        display: flex;
        align-items: center;
        gap: 10px;
        
        .provider-latency {
          font-size: 13px;
          color: var(--text-tertiary);
        }
      }
    }
  }
  
  .performance-metrics {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 16px;
    
    .metric-item {
      text-align: center;
      padding: 16px;
      background: var(--bg-secondary);
      border-radius: 8px;
      
      .metric-label {
        font-size: 13px;
        color: var(--text-secondary);
        margin-bottom: 8px;
      }
      
      .metric-value {
        font-size: 24px;
        font-weight: 600;
        
        &.error-rate {
          color: #FF3B30;
        }
      }
    }
  }
  
  .resource-usage {
    .resource-item {
      margin-bottom: 16px;
      
      .resource-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 8px;
        font-size: 13px;
      }
    }
    
    .resource-info {
      display: flex;
      gap: 20px;
      margin-top: 16px;
      
      .info-item {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
        color: var(--text-secondary);
      }
    }
  }
  
  .header-filters {
    display: flex;
    align-items: center;
  }
  
  .event-details {
    font-size: 12px;
    color: var(--text-tertiary);
    word-break: break-all;
  }
}
</style>
