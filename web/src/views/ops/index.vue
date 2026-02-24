<template>
  <div class="ops-page">
    <div class="ops-header">
      <div class="header-left">
        <h2>运维监控</h2>
        <div class="status-badge" :class="realtime.status">
          <span class="status-dot"></span>
          {{ realtime.health_status || '就绪' }}
        </div>
        <span class="refresh-time">刷新: {{ realtime.timestamp || '-' }}</span>
      </div>
      <div class="header-right">
        <el-select v-model="timeRange" size="small" @change="refreshAll" style="width: 100px;">
          <el-option label="近1分钟" value="1min" />
          <el-option label="近5分钟" value="5min" />
          <el-option label="近30分钟" value="30min" />
          <el-option label="近1小时" value="1h" />
        </el-select>
        <el-button size="small" @click="refreshAll" :loading="loading">
          <el-icon><Refresh /></el-icon>
        </el-button>
        <el-button size="small" @click="exportData">
          <el-icon><Download /></el-icon>
        </el-button>
      </div>
    </div>

    <div class="ops-content">
      <div class="top-row">
        <div class="diagnosis-card" :class="diagnosis.status">
          <div class="diagnosis-header">
            <el-icon :size="20"><Cpu /></el-icon>
            <span>智能诊断</span>
          </div>
          <div class="diagnosis-title">{{ diagnosis.title }}</div>
          <div class="diagnosis-message">{{ diagnosis.message }}</div>
          <div class="diagnosis-suggestions">
            <div v-for="(s, i) in diagnosis.suggestions" :key="i" class="suggestion-item">
              <el-icon><InfoFilled /></el-icon>
              {{ s }}
            </div>
          </div>
        </div>

        <div class="realtime-card">
          <div class="card-header">
            <span>实时信息</span>
            <div class="time-tabs">
              <span v-for="t in timeTabs" :key="t" :class="{ active: timeRange === t }" @click="timeRange = t; refreshAll()">{{ t }}</span>
            </div>
          </div>
          <div class="realtime-grid">
            <div class="realtime-section">
              <div class="section-label">当前</div>
              <div class="section-values">
                <div class="value-item">
                  <span class="value">{{ realtime.current_qps?.toFixed(1) || '0.0' }}</span>
                  <span class="label">QPS</span>
                </div>
                <div class="value-item">
                  <span class="value">{{ realtime.current_tps?.toFixed(1) || '0.0' }}</span>
                  <span class="label">TPS(千)</span>
                </div>
              </div>
            </div>
            <div class="realtime-section">
              <div class="section-label">峰值</div>
              <div class="section-values">
                <div class="value-item">
                  <span class="value">{{ realtime.peak_qps?.toFixed(1) || '0.0' }}</span>
                  <span class="label">QPS</span>
                </div>
                <div class="value-item">
                  <span class="value">{{ realtime.peak_tps?.toFixed(1) || '0.0' }}</span>
                  <span class="label">TPS(千)</span>
                </div>
              </div>
            </div>
            <div class="realtime-section">
              <div class="section-label">平均</div>
              <div class="section-values">
                <div class="value-item">
                  <span class="value">{{ realtime.avg_qps?.toFixed(1) || '0.0' }}</span>
                  <span class="label">QPS</span>
                </div>
                <div class="value-item">
                  <span class="value">{{ realtime.avg_tps?.toFixed(1) || '0.0' }}</span>
                  <span class="label">TPS(千)</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="metrics-row">
        <div class="metric-card">
          <div class="metric-header">
            <span>请求</span>
            <el-button type="primary" link size="small">明细</el-button>
          </div>
          <div class="metric-content">
            <div class="metric-row"><span>请求数:</span><span>{{ formatNumber(realtime.total_requests) }}</span></div>
            <div class="metric-row"><span>Token数:</span><span>{{ formatNumber(realtime.total_tokens) }}</span></div>
            <div class="metric-row"><span>平均 QPS:</span><span>{{ realtime.avg_qps?.toFixed(1) || '0.0' }}</span></div>
            <div class="metric-row"><span>平均 TPS:</span><span>{{ realtime.avg_tps?.toFixed(1) || '0.0' }}</span></div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <span>SLA（排除业务限制）</span>
            <el-button type="primary" link size="small">明细</el-button>
          </div>
          <div class="metric-content sla">
            <div class="sla-value" :class="{ warning: realtime.sla_percent < 99 }">
              {{ (realtime.sla_percent || 100).toFixed(3) }}%
            </div>
            <div class="metric-row"><span>异常数:</span><span>{{ realtime.error_count || 0 }}</span></div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <span>请求时长</span>
            <el-button type="primary" link size="small">明细</el-button>
          </div>
          <div class="metric-content">
            <div class="latency-main">{{ realtime.latency_p99 || '-' }} ms <span class="p-label">(P99)</span></div>
            <div class="metric-row"><span>P95:</span><span>{{ realtime.latency_p95 || '-' }} ms</span></div>
            <div class="metric-row"><span>P90:</span><span>{{ realtime.latency_p90 || '-' }} ms</span></div>
            <div class="metric-row"><span>P50:</span><span>{{ realtime.latency_p50 || '-' }} ms</span></div>
            <div class="metric-row"><span>Avg:</span><span>{{ realtime.latency_avg || '-' }} ms</span></div>
            <div class="metric-row"><span>Max:</span><span>{{ realtime.latency_max || '-' }} ms</span></div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <span>TTFT</span>
            <el-button type="primary" link size="small">明细</el-button>
          </div>
          <div class="metric-content">
            <div class="latency-main">{{ realtime.ttft_p99 || '-' }} ms <span class="p-label">(P99)</span></div>
            <div class="metric-row"><span>P95:</span><span>{{ realtime.ttft_p95 || '-' }} ms</span></div>
            <div class="metric-row"><span>P90:</span><span>{{ realtime.ttft_p90 || '-' }} ms</span></div>
            <div class="metric-row"><span>P50:</span><span>{{ realtime.ttft_p50 || '-' }} ms</span></div>
            <div class="metric-row"><span>Avg:</span><span>{{ realtime.ttft_avg || '-' }} ms</span></div>
            <div class="metric-row"><span>Max:</span><span>{{ realtime.ttft_max || '-' }} ms</span></div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <span>请求错误</span>
            <el-button type="primary" link size="small">明细</el-button>
          </div>
          <div class="metric-content">
            <div class="error-value" :class="{ warning: realtime.request_error_rate > 1 }">
              {{ (realtime.request_error_rate || 0).toFixed(2) }}%
            </div>
            <div class="metric-row"><span>错误数:</span><span>{{ realtime.error_count || 0 }}</span></div>
            <div class="metric-row"><span>业务限制:</span><span>{{ realtime.business_limit || 0 }}</span></div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <span>上游错误</span>
            <el-button type="primary" link size="small">明细</el-button>
          </div>
          <div class="metric-content">
            <div class="error-value" :class="{ warning: realtime.upstream_error_rate > 1 }">
              {{ (realtime.upstream_error_rate || 0).toFixed(2) }}%
            </div>
            <div class="metric-row"><span>错误数(排除429/529):</span><span>{{ realtime.upstream_error_count || 0 }}</span></div>
            <div class="metric-row"><span>429/529:</span><span>{{ realtime.error_429_count || 0 }}</span></div>
          </div>
        </div>
      </div>

      <div class="resources-row">
        <div class="resource-card">
          <div class="resource-header">
            <el-icon><Cpu /></el-icon>
            <span>CPU</span>
          </div>
          <div class="resource-value">{{ (resources.cpu_usage || 0).toFixed(1) }}%</div>
          <el-progress :percentage="resources.cpu_usage || 0" :show-text="false" :stroke-width="6" />
          <div class="resource-thresholds">警告 {{ resources.cpu_warning }}% · 严重 {{ resources.cpu_critical }}%</div>
        </div>

        <div class="resource-card">
          <div class="resource-header">
            <el-icon><Coin /></el-icon>
            <span>内存</span>
          </div>
          <div class="resource-value">{{ (resources.memory_usage || 0).toFixed(1) }}%</div>
          <div class="resource-detail">{{ resources.memory_used_mb || 0 }} / {{ resources.memory_total_mb || 0 }} MB</div>
          <el-progress :percentage="resources.memory_usage || 0" :show-text="false" :stroke-width="6" />
        </div>

        <div class="resource-card">
          <div class="resource-header">
            <el-icon><Connection /></el-icon>
            <span>协程</span>
          </div>
          <div class="resource-value" :class="{ warning: resources.goroutines > resources.goroutine_warning }">
            {{ resources.goroutines || 0 }}
          </div>
          <div class="resource-status">正常</div>
          <div class="resource-thresholds">警告 {{ resources.goroutine_warning }} · 严重 {{ resources.goroutine_critical }}</div>
        </div>

        <div class="resource-card">
          <div class="resource-header">
            <el-icon><Timer /></el-icon>
            <span>GC</span>
          </div>
          <div class="resource-value">{{ resources.gc_count || 0 }}</div>
          <div class="resource-status">正常</div>
          <div class="resource-detail">暂停总时长: {{ ((resources.gc_pause_total_ns || 0) / 1e6).toFixed(2) }} ms</div>
        </div>
      </div>

      <div class="system-row">
        <div class="system-card">
          <div class="card-title">系统信息</div>
          <div class="system-grid">
            <div class="system-item"><span class="label">主机名</span><span class="value">{{ system.hostname }}</span></div>
            <div class="system-item"><span class="label">操作系统</span><span class="value">{{ system.os }} / {{ system.arch }}</span></div>
            <div class="system-item"><span class="label">Go 版本</span><span class="value">{{ system.go_version }}</span></div>
            <div class="system-item"><span class="label">CPU 核数</span><span class="value">{{ system.cpu_count }}</span></div>
            <div class="system-item"><span class="label">运行时间</span><span class="value">{{ system.uptime }}</span></div>
            <div class="system-item"><span class="label">启动时间</span><span class="value">{{ formatTime(system.start_time) }}</span></div>
          </div>
        </div>

        <div class="services-card">
          <div class="card-title">服务状态</div>
          <div class="services-list">
            <div v-for="s in services" :key="s.name" class="service-item">
              <span class="service-name">{{ s.name }}</span>
              <el-tag :type="s.status === 'healthy' ? 'success' : 'danger'" size="small">{{ s.status === 'healthy' ? '正常' : '异常' }}</el-tag>
            </div>
          </div>
        </div>

        <div class="providers-card">
          <div class="card-title">服务商状态</div>
          <div class="providers-list">
            <div v-for="p in providers" :key="p.name" class="provider-item">
              <span class="provider-name">{{ p.name }}</span>
              <div class="provider-info">
                <el-tag :type="p.status === 'healthy' ? 'success' : 'danger'" size="small">{{ p.status === 'healthy' ? '正常' : '异常' }}</el-tag>
                <span class="latency">{{ p.latency_ms }} ms</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { request } from '@/api/request'

const loading = ref(false)
const timeRange = ref('1h')
const timeTabs = ['1min', '5min', '30min', '1h']

const system = ref<any>({})
const realtime = ref<any>({})
const resources = ref<any>({})
const diagnosis = ref<any>({ status: 'idle', title: '待机', message: '', suggestions: [] })
const services = ref<any[]>([])
const providers = ref<any[]>([])

let timer: ReturnType<typeof setInterval> | null = null

const formatNumber = (n: number): string => {
  if (!n) return '0'
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'K'
  return n.toString()
}

const formatTime = (t: string): string => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const fetchData = async () => {
  try {
    const res: any = await request.get(`/admin/ops/dashboard?range=${timeRange.value}`)
    const data = res?.data || {}
    system.value = data.system || {}
    realtime.value = data.realtime || {}
    resources.value = data.resources || {}
    diagnosis.value = data.diagnosis || {}
  } catch (e) {
    console.warn('Failed to fetch dashboard:', e)
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

const fetchProviders = async () => {
  try {
    const res: any = await request.get('/admin/ops/providers/health')
    providers.value = res?.data || []
  } catch (e) {
    console.warn('Failed to fetch providers:', e)
  }
}

const refreshAll = async () => {
  loading.value = true
  try {
    await Promise.all([fetchData(), fetchServices(), fetchProviders()])
  } finally {
    loading.value = false
  }
}

const exportData = async () => {
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
    console.warn('Export failed:', e)
  }
}

onMounted(() => {
  refreshAll()
  timer = setInterval(fetchData, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped lang="scss">
.ops-page {
  padding: 20px;
  background: var(--bg-primary);
  min-height: 100vh;
}

.ops-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  
  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
    
    h2 {
      margin: 0;
      font-size: 20px;
    }
    
    .status-badge {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: 16px;
      font-size: 13px;
      background: rgba(52, 199, 89, 0.1);
      color: #34C759;
      
      &.warning {
        background: rgba(255, 149, 0, 0.1);
        color: #FF9500;
      }
      
      &.critical {
        background: rgba(255, 59, 48, 0.1);
        color: #FF3B30;
      }
      
      .status-dot {
        width: 8px;
        height: 8px;
        border-radius: 50%;
        background: currentColor;
      }
    }
    
    .refresh-time {
      font-size: 12px;
      color: var(--text-tertiary);
    }
  }
  
  .header-right {
    display: flex;
    gap: 8px;
  }
}

.ops-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.top-row {
  display: grid;
  grid-template-columns: 280px 1fr;
  gap: 16px;
}

.diagnosis-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 16px;
  
  &.idle {
    border-left: 4px solid #8E8E93;
  }
  
  &.healthy {
    border-left: 4px solid #34C759;
  }
  
  &.warning {
    border-left: 4px solid #FF9500;
  }
  
  .diagnosis-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 12px;
  }
  
  .diagnosis-title {
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 4px;
  }
  
  .diagnosis-message {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 12px;
  }
  
  .diagnosis-suggestions {
    .suggestion-item {
      display: flex;
      align-items: flex-start;
      gap: 6px;
      font-size: 12px;
      color: var(--text-tertiary);
      margin-bottom: 4px;
    }
  }
}

.realtime-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 16px;
  
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    font-weight: 500;
    
    .time-tabs {
      display: flex;
      gap: 4px;
      
      span {
        padding: 4px 12px;
        font-size: 12px;
        border-radius: 4px;
        cursor: pointer;
        color: var(--text-secondary);
        
        &.active {
          background: var(--color-primary);
          color: #fff;
        }
      }
    }
  }
  
  .realtime-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 20px;
    
    .realtime-section {
      .section-label {
        font-size: 12px;
        color: var(--text-tertiary);
        margin-bottom: 8px;
      }
      
      .section-values {
        display: flex;
        gap: 24px;
        
        .value-item {
          .value {
            display: block;
            font-size: 24px;
            font-weight: 600;
          }
          .label {
            font-size: 12px;
            color: var(--text-tertiary);
          }
        }
      }
    }
  }
}

.metrics-row {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 12px;
}

.metric-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 14px;
  
  .metric-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    font-size: 13px;
    font-weight: 500;
  }
  
  .metric-content {
    font-size: 12px;
    
    .metric-row {
      display: flex;
      justify-content: space-between;
      padding: 4px 0;
      color: var(--text-secondary);
      
      span:last-child {
        color: var(--text-primary);
        font-weight: 500;
      }
    }
    
    .sla-value {
      font-size: 28px;
      font-weight: 600;
      color: #34C759;
      margin-bottom: 8px;
      
      &.warning {
        color: #FF9500;
      }
    }
    
    .latency-main {
      font-size: 20px;
      font-weight: 600;
      margin-bottom: 8px;
      
      .p-label {
        font-size: 12px;
        font-weight: normal;
        color: var(--text-tertiary);
      }
    }
    
    .error-value {
      font-size: 24px;
      font-weight: 600;
      margin-bottom: 8px;
      
      &.warning {
        color: #FF3B30;
      }
    }
  }
}

.resources-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
}

.resource-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 16px;
  
  .resource-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 500;
    margin-bottom: 8px;
  }
  
  .resource-value {
    font-size: 28px;
    font-weight: 600;
    margin-bottom: 4px;
    
    &.warning {
      color: #FF9500;
    }
  }
  
  .resource-detail {
    font-size: 12px;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }
  
  .resource-status {
    font-size: 12px;
    color: #34C759;
    margin-bottom: 4px;
  }
  
  .resource-thresholds {
    font-size: 11px;
    color: var(--text-tertiary);
    margin-top: 8px;
  }
}

.system-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 12px;
}

.system-card, .services-card, .providers-card {
  background: var(--bg-secondary);
  border-radius: 12px;
  padding: 16px;
  
  .card-title {
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 12px;
  }
}

.system-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
  
  .system-item {
    .label {
      font-size: 12px;
      color: var(--text-tertiary);
      display: block;
    }
    .value {
      font-size: 13px;
      font-weight: 500;
    }
  }
}

.services-list, .providers-list {
  .service-item, .provider-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-light);
    
    &:last-child {
      border-bottom: none;
    }
    
    .service-name, .provider-name {
      font-size: 13px;
      text-transform: capitalize;
    }
    
    .provider-info {
      display: flex;
      align-items: center;
      gap: 8px;
      
      .latency {
        font-size: 12px;
        color: var(--text-tertiary);
      }
    }
  }
}
</style>
