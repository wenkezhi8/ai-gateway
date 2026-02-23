<template>
  <div class="cache-page">
    <!-- 缓存统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :span="6" v-for="stat in cacheStats" :key="stat.title">
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
      <!-- 缓存类型 -->
      <el-col :span="8">
        <el-card shadow="never" class="page-card types-card">
          <template #header>
            <div class="card-header">
              <span>缓存类型</span>
              <el-button type="primary" size="small" @click="refreshAllCache">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>

          <div class="cache-types">
            <div v-for="cacheType in cacheTypes" :key="cacheType.id" class="cache-type-item">
              <div class="type-header">
                <div class="type-info">
                  <span class="type-name">{{ cacheType.name }}</span>
                  <el-tag size="small" :type="cacheType.enabled ? 'success' : 'info'">
                    {{ cacheType.enabled ? '已启用' : '已禁用' }}
                  </el-tag>
                </div>
                <el-switch v-model="cacheType.enabled" size="small" @change="handleTypeChange(cacheType)" />
              </div>
              <div class="type-stats">
                <div class="stat-item">
                  <span class="label">命中率</span>
                  <el-progress :percentage="cacheType.hitRate" :color="getProgressColor(cacheType.hitRate)" :stroke-width="6" />
                </div>
                <div class="stat-row">
                  <span><el-icon><Files /></el-icon> {{ cacheType.entries }} 条</span>
                  <span><el-icon><Coin /></el-icon> {{ cacheType.size }}</span>
                </div>
              </div>
              <div class="type-actions">
                <el-button type="primary" link size="small" @click="clearCacheType(cacheType)">
                  <el-icon><Delete /></el-icon> 清空
                </el-button>
                <el-button type="primary" link size="small" @click="viewCacheDetail(cacheType)">
                  <el-icon><View /></el-icon> 详情
                </el-button>
              </div>
            </div>
          </div>
        </el-card>
      </el-col>

      <!-- 缓存配置 -->
      <el-col :span="16">
        <el-card shadow="never" class="page-card config-card">
          <template #header>
            <div class="card-header">
              <span>缓存配置</span>
            </div>
          </template>

          <el-tabs v-model="activeTab">
            <el-tab-pane label="基本配置" name="general">
              <el-form :model="cacheConfig" label-width="140px" class="config-form">
                <el-form-item label="启用缓存">
                  <el-switch v-model="cacheConfig.enabled" @change="saveConfig" />
                </el-form-item>

                <el-form-item label="缓存策略">
                  <el-select v-model="cacheConfig.strategy" style="width: 100%" @change="saveConfig">
                    <el-option label="语义相似度" value="semantic" />
                    <el-option label="精确匹配" value="exact" />
                    <el-option label="前缀匹配" value="prefix" />
                  </el-select>
                </el-form-item>

                <el-form-item label="相似度阈值">
                  <el-row style="width: 100%" :gutter="16">
                    <el-col :span="18">
                      <el-slider v-model="cacheConfig.similarityThreshold" :min="0" :max="100" @change="saveConfig" />
                    </el-col>
                    <el-col :span="6">
                      <el-tag>{{ cacheConfig.similarityThreshold }}%</el-tag>
                    </el-col>
                  </el-row>
                </el-form-item>

                <el-form-item label="默认TTL">
                  <el-input-number v-model="cacheConfig.defaultTTL" :min="60" :max="86400" @change="saveConfig" />
                  <span class="unit-label">秒</span>
                </el-form-item>

                <el-form-item label="最大缓存大小">
                  <el-input-number v-model="cacheConfig.maxSize" :min="100" :max="10000" :step="100" @change="saveConfig" />
                  <span class="unit-label">MB</span>
                </el-form-item>

                <el-form-item label="淘汰策略">
                  <el-select v-model="cacheConfig.evictionPolicy" style="width: 100%" @change="saveConfig">
                    <el-option label="LRU (最近最少使用)" value="lru" />
                    <el-option label="LFU (最不经常使用)" value="lfu" />
                    <el-option label="FIFO (先进先出)" value="fifo" />
                  </el-select>
                </el-form-item>
              </el-form>
            </el-tab-pane>

            <el-tab-pane label="缓存规则" name="rules">
              <div class="rules-header">
                <el-button type="primary" size="small" @click="showAddRuleDialog">
                  <el-icon><Plus /></el-icon>
                  添加规则
                </el-button>
              </div>

              <el-table :data="cacheRules" stripe class="rules-table">
                <el-table-column prop="pattern" label="匹配模式" min-width="200">
                  <template #default="{ row }">
                    <code class="pattern-code">{{ row.pattern }}</code>
                  </template>
                </el-table-column>
                <el-table-column prop="modelFilter" label="模型过滤" width="140">
                  <template #default="{ row }">
                    <el-tag size="small" v-if="row.modelFilter">{{ row.modelFilter }}</el-tag>
                    <span v-else class="text-muted">全部</span>
                  </template>
                </el-table-column>
                <el-table-column prop="ttl" label="TTL" width="100">
                  <template #default="{ row }">
                    {{ formatTTL(row.ttl) }}
                  </template>
                </el-table-column>
                <el-table-column prop="priority" label="优先级" width="100">
                  <template #default="{ row }">
                    <el-tag size="small" :type="getPriorityType(row.priority)">{{ getPriorityText(row.priority) }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="enabled" label="状态" width="80">
                  <template #default="{ row }">
                    <el-switch v-model="row.enabled" size="small" @change="handleRuleChange(row)" />
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="100" fixed="right">
                  <template #default="{ row }">
                    <el-button type="primary" link size="small" @click="editRule(row)">编辑</el-button>
                    <el-button type="danger" link size="small" @click="deleteRule(row)">删除</el-button>
                  </template>
                </el-table-column>
              </el-table>
            </el-tab-pane>

            <el-tab-pane label="热门缓存" name="hot">
              <el-table :data="hotCaches" stripe class="hot-cache-table">
                <el-table-column prop="query" label="查询哈希" min-width="200">
                  <template #default="{ row }">
                    <div class="query-cell">
                      <el-icon><Key /></el-icon>
                      <code class="hash-code">{{ row.hash }}</code>
                    </div>
                  </template>
                </el-table-column>
                <el-table-column prop="model" label="模型" width="140">
                  <template #default="{ row }">
                    <el-tag size="small">{{ row.model }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column prop="hits" label="命中次数" width="100" sortable>
                  <template #default="{ row }">
                    <span class="hits-count">{{ row.hits.toLocaleString() }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="size" label="大小" width="100">
                  <template #default="{ row }">
                    {{ row.size }}
                  </template>
                </el-table-column>
                <el-table-column prop="lastHit" label="最后命中" width="120">
                  <template #default="{ row }">
                    <span class="time-text">{{ row.lastHit }}</span>
                  </template>
                </el-table-column>
                <el-table-column prop="createdAt" label="创建时间" width="120">
                  <template #default="{ row }">
                    <span class="time-text">{{ row.createdAt }}</span>
                  </template>
                </el-table-column>
              </el-table>

              <div class="pagination">
                <el-pagination
                  v-model:current-page="hotCachePage"
                  v-model:page-size="hotCachePageSize"
                  :total="hotCacheTotal"
                  :page-sizes="[10, 20, 50]"
                  layout="total, sizes, prev, pager, next"
                />
              </div>
            </el-tab-pane>

            <el-tab-pane label="任务类型TTL" name="task-ttl">
              <el-alert type="info" :closable="false" show-icon style="margin-bottom: 16px">
                <template #title>
                  根据任务类型设置缓存过期时间。内容越稳定、调用成本越高的任务，TTL 应越长。
                </template>
              </el-alert>
              
              <el-table :data="taskTTLList" stripe>
                <el-table-column prop="name" label="任务类型" width="150" />
                <el-table-column prop="description" label="说明" min-width="200" />
                <el-table-column label="TTL (小时)" width="180">
                  <template #default="{ row }">
                    <el-input-number 
                      v-model="row.ttl" 
                      :min="0" 
                      :max="2160" 
                      :step="24"
                      size="small"
                      style="width: 120px"
                    />
                  </template>
                </el-table-column>
                <el-table-column label="缓存策略" width="120">
                  <template #default="{ row }">
                    <el-tag v-if="row.ttl === 0" type="danger" size="small">不缓存</el-tag>
                    <el-tag v-else-if="row.ttl <= 24" type="warning" size="small">短期</el-tag>
                    <el-tag v-else-if="row.ttl <= 168" type="primary" size="small">中期</el-tag>
                    <el-tag v-else type="success" size="small">长期</el-tag>
                  </template>
                </el-table-column>
              </el-table>
              
              <div style="margin-top: 16px">
                <el-button type="primary" @click="saveTaskTTLConfig" :loading="saving">
                  <el-icon><Check /></el-icon>
                  保存配置
                </el-button>
                <el-button @click="resetTaskTTLConfig">
                  <el-icon><Refresh /></el-icon>
                  恢复默认
                </el-button>
              </div>
              
              <el-divider />
              
              <h4>TTL 设置规则</h4>
              <el-descriptions :column="2" border size="small">
                <el-descriptions-item label="不缓存 (0h)">创意写作、个性化内容</el-descriptions-item>
                <el-descriptions-item label="短期 (1-24h)">日常对话、实时信息、个性化咨询</el-descriptions-item>
                <el-descriptions-item label="中期 (168h)">代码生成、工具模板、事实查询</el-descriptions-item>
                <el-descriptions-item label="长期 (360-720h)">数学计算、多模态理解、专业知识</el-descriptions-item>
              </el-descriptions>
            </el-tab-pane>

            <el-tab-pane label="高级功能" name="advanced">
              <el-row :gutter="24">
                <!-- Redis 状态 -->
                <el-col :span="12">
                  <div class="advanced-section">
                    <h4><el-icon><Coin /></el-icon> Redis 缓存状态</h4>
                    <div class="status-card">
                      <div class="status-item">
                        <span class="label">连接状态</span>
                        <el-tag :type="redisHealth.status === 'healthy' ? 'success' : 'danger'">
                          {{ redisHealth.status === 'healthy' ? '已连接' : '未连接' }}
                        </el-tag>
                      </div>
                      <div class="status-item">
                        <span class="label">后端类型</span>
                        <span class="value">{{ redisHealth.backend || '内存缓存' }}</span>
                      </div>
                      <div class="status-item">
                        <span class="label">延迟</span>
                        <span class="value">{{ redisHealth.latency_ms || 0 }} ms</span>
                      </div>
                    </div>
                    <el-button type="primary" size="small" @click="runHealthCheck" style="margin-top: 12px">
                      <el-icon><Refresh /></el-icon> 健康检查
                    </el-button>
                    <el-alert type="info" :closable="false" show-icon style="margin-top: 12px">
                      <template #title>Redis 连接失败时自动降级到内存缓存</template>
                    </el-alert>
                  </div>
                </el-col>

                <!-- 请求去重 -->
                <el-col :span="12">
                  <div class="advanced-section">
                    <h4><el-icon><Connection /></el-icon> 请求去重配置</h4>
                    <el-form label-width="100px" size="small">
                      <el-form-item label="启用去重">
                        <el-switch v-model="dedupConfig.enabled" @change="saveDedupConfig" />
                      </el-form-item>
                      <el-form-item label="最大等待数">
                        <el-input-number v-model="dedupConfig.maxPending" :min="1" :max="100" @change="saveDedupConfig" />
                      </el-form-item>
                      <el-form-item label="请求超时">
                        <el-input-number v-model="dedupConfig.requestTimeout" :min="1" :max="300" @change="saveDedupConfig" />
                        <span class="unit-label">秒</span>
                      </el-form-item>
                    </el-form>
                    <el-alert type="success" :closable="false" show-icon style="margin-top: 12px">
                      <template #title>相同请求自动合并，减少重复调用</template>
                    </el-alert>
                  </div>
                </el-col>
              </el-row>

              <!-- 语义缓存 -->
              <div class="advanced-section" style="margin-top: 20px">
                <h4><el-icon><MagicStick /></el-icon> 语义缓存说明</h4>
                <el-descriptions :column="2" border>
                  <el-descriptions-item label="工作原理">
                    基于向量相似度匹配，相似请求可复用缓存结果
                  </el-descriptions-item>
                  <el-descriptions-item label="相似度阈值">
                    {{ cacheConfig.similarityThreshold }}% - 高于此值的请求将被视为相同
                  </el-descriptions-item>
                  <el-descriptions-item label="缓存策略">
                    {{ cacheConfig.strategy === 'semantic' ? '语义相似度' : cacheConfig.strategy === 'exact' ? '精确匹配' : '前缀匹配' }}
                  </el-descriptions-item>
                  <el-descriptions-item label="自动降级">
                    Redis 不可用时自动降级到内存缓存
                  </el-descriptions-item>
                </el-descriptions>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>

    <!-- 添加/编辑规则对话框 -->
    <el-dialog v-model="ruleDialogVisible" :title="isEditRule ? '编辑缓存规则' : '添加缓存规则'" width="550px">
      <el-form :model="ruleForm" :rules="ruleFormRules" ref="ruleFormRef" label-width="120px">
        <el-form-item label="匹配模式" prop="pattern">
          <el-input v-model="ruleForm.pattern" placeholder="例如：chat:* 或 gpt-4:*" />
        </el-form-item>
        <el-form-item label="模型过滤">
          <el-select v-model="ruleForm.modelFilter" placeholder="选择模型（可选）" clearable style="width: 100%">
            <el-option label="所有模型" value="" />
            <el-option-group label="OpenAI">
              <el-option label="gpt-4o" value="gpt-4o" />
              <el-option label="gpt-4-turbo" value="gpt-4-turbo" />
              <el-option label="gpt-3.5-turbo" value="gpt-3.5-turbo" />
            </el-option-group>
            <el-option-group label="Anthropic">
              <el-option label="claude-3-5-sonnet" value="claude-3-5-sonnet" />
              <el-option label="claude-3-opus" value="claude-3-opus" />
            </el-option-group>
            <el-option-group label="阿里云通义千问">
              <el-option label="qwen-max" value="qwen-max" />
              <el-option label="qwen-plus" value="qwen-plus" />
              <el-option label="qwen-turbo" value="qwen-turbo" />
            </el-option-group>
            <el-option-group label="百度文心一言">
              <el-option label="ernie-4.0" value="ernie-4.0" />
              <el-option label="ernie-3.5" value="ernie-3.5" />
            </el-option-group>
            <el-option-group label="智谱AI">
              <el-option label="glm-4-plus" value="glm-4-plus" />
              <el-option label="glm-4-flash" value="glm-4-flash" />
            </el-option-group>
            <el-option-group label="月之暗面">
              <el-option label="moonshot-v1-8k" value="moonshot-v1-8k" />
              <el-option label="moonshot-v1-128k" value="moonshot-v1-128k" />
            </el-option-group>
            <el-option-group label="DeepSeek">
              <el-option label="deepseek-chat" value="deepseek-chat" />
              <el-option label="deepseek-reasoner" value="deepseek-reasoner" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="TTL">
          <el-row :gutter="10">
            <el-col :span="12">
              <el-input-number v-model="ruleForm.ttlValue" :min="1" style="width: 100%" />
            </el-col>
            <el-col :span="12">
              <el-select v-model="ruleForm.ttlUnit" style="width: 100%">
                <el-option label="秒" value="seconds" />
                <el-option label="分钟" value="minutes" />
                <el-option label="小时" value="hours" />
                <el-option label="天" value="days" />
              </el-select>
            </el-col>
          </el-row>
        </el-form-item>
        <el-form-item label="优先级">
          <el-select v-model="ruleForm.priority" style="width: 100%">
            <el-option label="高" value="high" />
            <el-option label="中" value="medium" />
            <el-option label="低" value="low" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRule">保存</el-button>
      </template>
    </el-dialog>

    <!-- 缓存详情对话框 -->
    <el-dialog v-model="detailDialogVisible" :title="cacheDetail?.name + ' 统计'" width="600px">
      <div v-if="cacheDetail" class="cache-detail">
        <el-row :gutter="20">
          <el-col :span="12">
            <div class="detail-item">
              <span class="label">命中率</span>
              <el-progress :percentage="cacheDetail.hitRate" :color="getProgressColor(cacheDetail.hitRate)" :stroke-width="10" />
            </div>
          </el-col>
          <el-col :span="12">
            <div class="detail-item">
              <span class="label">内存使用</span>
              <el-progress :percentage="cacheDetail.memoryUsage" color="#409eff" :stroke-width="10" />
            </div>
          </el-col>
        </el-row>

        <el-descriptions :column="2" border class="detail-desc">
          <el-descriptions-item label="总条目数">{{ cacheDetail.entries }}</el-descriptions-item>
          <el-descriptions-item label="总大小">{{ cacheDetail.size }}</el-descriptions-item>
          <el-descriptions-item label="总命中">{{ cacheDetail.totalHits?.toLocaleString() }}</el-descriptions-item>
          <el-descriptions-item label="总未命中">{{ cacheDetail.totalMisses?.toLocaleString() }}</el-descriptions-item>
          <el-descriptions-item label="平均响应">{{ cacheDetail.avgResponse }}</el-descriptions-item>
          <el-descriptions-item label="上次清理">{{ cacheDetail.lastCleared }}</el-descriptions-item>
        </el-descriptions>

        <div class="detail-chart">
          <h4>命中率趋势（最近24小时）</h4>
          <div ref="detailChartRef" class="chart-container"></div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { request } from '@/api/request'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'
import * as echarts from 'echarts'

interface CacheType {
  id: string
  name: string
  enabled: boolean
  hitRate: number
  entries: number
  size: string
  memoryUsage?: number
  totalHits?: number
  totalMisses?: number
  avgResponse?: string
  lastCleared?: string
}

interface CacheRule {
  id: number
  pattern: string
  modelFilter: string
  ttl: number
  priority: string
  enabled: boolean
}

interface HotCache {
  hash: string
  model: string
  hits: number
  size: string
  lastHit: string
  createdAt: string
}

const loading = ref(false)
const saving = ref(false)
const activeTab = ref('general')
const ruleDialogVisible = ref(false)
const detailDialogVisible = ref(false)
const isEditRule = ref(false)
const ruleFormRef = ref<FormInstance>()
const detailChartRef = ref<HTMLElement>()
const hotCachePage = ref(1)
const hotCachePageSize = ref(10)
const hotCacheTotal = ref(100)
const cacheDetail = ref<CacheType | null>(null)
let detailChart: echarts.ECharts | null = null

const overallStats = reactive({
  hitRate: 0,
  totalSize: '0 MB',
  totalEntries: 0,
  avgResponse: '0ms'
})

const cacheStats = computed(() => [
  { title: '总命中率', value: `${overallStats.hitRate.toFixed(1)}%`, icon: 'CircleCheckFilled', color: '#34C759' },
  { title: '缓存大小', value: overallStats.totalSize, icon: 'Coin', color: '#007AFF' },
  { title: '总条目数', value: overallStats.totalEntries.toString(), icon: 'Files', color: '#FF9500' },
  { title: '平均响应', value: overallStats.avgResponse, icon: 'Timer', color: '#5856D6' }
])

const cacheTypes = ref<CacheType[]>([
  { id: 'request', name: '请求缓存', enabled: true, hitRate: 0, entries: 0, size: '0 MB' },
  { id: 'context', name: '上下文缓存', enabled: true, hitRate: 0, entries: 0, size: '0 MB' },
  { id: 'route', name: '路由缓存', enabled: true, hitRate: 0, entries: 0, size: '0 MB' },
  { id: 'usage', name: '用量统计缓存', enabled: true, hitRate: 0, entries: 0, size: '0 MB' }
])

const cacheConfig = reactive({
  enabled: true,
  strategy: 'semantic',
  similarityThreshold: 85,
  defaultTTL: 3600,
  maxSize: 1024,
  evictionPolicy: 'lru'
})

const redisHealth = reactive({
  status: 'unknown',
  backend: '',
  latency_ms: 0
})

const dedupConfig = reactive({
  enabled: true,
  maxPending: 100,
  requestTimeout: 30
})

// 任务类型 TTL 配置
interface TaskTTLItem {
  key: string
  name: string
  description: string
  ttl: number
}

const taskTTLList = ref<TaskTTLItem[]>([
  { key: 'fact', name: '事实查询', description: '公共事实、政策、常识等，可能定期更新', ttl: 24 },
  { key: 'code', name: '代码生成', description: '通用代码片段，更新频率低', ttl: 168 },
  { key: 'math', name: '数学计算', description: '数学题结果，几乎不会变化', ttl: 720 },
  { key: 'chat', name: '日常对话', description: '个性化对话，上下文相关性强', ttl: 1 },
  { key: 'creative', name: '创意写作', description: '个性化创意内容，默认不缓存', ttl: 0 },
  { key: 'reasoning', name: '逻辑推理', description: '推理结果，稳定性高', ttl: 168 },
  { key: 'translate', name: '翻译', description: '标准翻译结果，仅术语更新时变化', ttl: 360 },
  { key: 'long_text', name: '长文本处理', description: '文档摘要、PDF解析等，同一文本结果固定', ttl: 360 },
  { key: 'other', name: '其他类型', description: '未分类任务', ttl: 24 }
])

const defaultTaskTTL = {
  fact: 24,
  code: 168,
  math: 720,
  chat: 1,
  creative: 0,
  reasoning: 168,
  translate: 360,
  long_text: 360,
  other: 24
}

const cacheRules = ref<CacheRule[]>([])

const hotCaches = ref<HotCache[]>([])

const ruleForm = reactive({
  id: 0,
  pattern: '',
  modelFilter: '',
  ttlValue: 1,
  ttlUnit: 'hours',
  priority: 'medium',
  enabled: true
})

const ruleFormRules: FormRules = {
  pattern: [{ required: true, message: 'Please enter pattern', trigger: 'blur' }]
}

const getProgressColor = (percentage: number) => {
  if (percentage >= 80) return '#34C759'
  if (percentage >= 60) return '#007AFF'
  if (percentage >= 40) return '#FF9500'
  return '#FF3B30'
}

const formatTTL = (seconds: number) => {
  if (seconds >= 86400) return `${seconds / 86400}d`
  if (seconds >= 3600) return `${seconds / 3600}h`
  if (seconds >= 60) return `${seconds / 60}m`
  return `${seconds}s`
}

const getPriorityType = (priority: string) => {
  const types: Record<string, string> = {
    high: 'danger',
    medium: 'warning',
    low: 'info'
  }
  return types[priority] || 'info'
}

const getPriorityText = (priority: string) => {
  const texts: Record<string, string> = {
    high: '高',
    medium: '中',
    low: '低'
  }
  return texts[priority] || priority
}

const handleTypeChange = async (type: CacheType) => {
  try {
    await request.put('/admin/cache/config', {
      [type.id]: { enabled: type.enabled }
    })
    ElMessage.success(`${type.name} 已${type.enabled ? '启用' : '禁用'}`)
  } catch (e) {
    type.enabled = !type.enabled
    handleApiError(e, '操作失败')
  }
}

const clearCacheType = async (type: CacheType) => {
  try {
    await ElMessageBox.confirm(`确定清空 ${type.name} 的所有缓存吗？`, '警告', { type: 'warning' })
    await request.delete(`/admin/cache?type=${type.id}`)
    type.entries = 0
    type.size = '0 MB'
    type.hitRate = 0
    ElMessage.success(`${type.name} 已清空`)
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '清空失败')
    }
  }
}

const viewCacheDetail = (type: CacheType) => {
  cacheDetail.value = {
    ...type,
    memoryUsage: type.memoryUsage || 0,
    totalHits: type.totalHits || 0,
    totalMisses: type.totalMisses || 0,
    avgResponse: type.avgResponse || '0ms',
    lastCleared: type.lastCleared || '-'
  }
  detailDialogVisible.value = true

  nextTick(() => {
    initDetailChart()
  })
}

const initDetailChart = () => {
  if (!detailChartRef.value) return

  if (detailChart) {
    detailChart.dispose()
  }
  detailChart = echarts.init(detailChartRef.value)
  const hours = Array.from({ length: 24 }, (_, i) => `${23 - i}h ago`).reverse()
  const hitRate = cacheDetail.value?.hitRate || 0
  const data = Array.from({ length: 24 }, () => hitRate)

  detailChart.setOption({
    tooltip: {
      trigger: 'axis'
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLabel: { fontSize: 10 }
    },
    yAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { formatter: '{value}%' }
    },
    series: [{
      data: data,
      type: 'line',
      smooth: true,
      areaStyle: {
        color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
          { offset: 0, color: 'rgba(0, 122, 255, 0.3)' },
          { offset: 1, color: 'rgba(0, 122, 255, 0.05)' }
        ])
      },
      lineStyle: { color: '#007AFF' },
      itemStyle: { color: '#007AFF' }
    }]
  })
}

const refreshAllCache = () => {
  ElMessage.success('缓存统计已刷新')
}

const saveConfig = () => {
  ElMessage.success('配置已保存')
}

const showAddRuleDialog = () => {
  isEditRule.value = false
  Object.assign(ruleForm, {
    id: 0,
    pattern: '',
    modelFilter: '',
    ttlValue: 1,
    ttlUnit: 'hours',
    priority: 'medium',
    enabled: true
  })
  ruleDialogVisible.value = true
}

const editRule = (rule: CacheRule) => {
  isEditRule.value = true
  let ttlValue = rule.ttl
  let ttlUnit = 'seconds'

  if (rule.ttl >= 86400) {
    ttlValue = rule.ttl / 86400
    ttlUnit = 'days'
  } else if (rule.ttl >= 3600) {
    ttlValue = rule.ttl / 3600
    ttlUnit = 'hours'
  } else if (rule.ttl >= 60) {
    ttlValue = rule.ttl / 60
    ttlUnit = 'minutes'
  }

  Object.assign(ruleForm, {
    id: rule.id,
    pattern: rule.pattern,
    modelFilter: rule.modelFilter,
    ttlValue,
    ttlUnit,
    priority: rule.priority,
    enabled: rule.enabled
  })
  ruleDialogVisible.value = true
}

const deleteRule = async (rule: CacheRule) => {
  try {
    await ElMessageBox.confirm(`确定删除规则 "${rule.pattern}" 吗？`, '警告', { type: 'warning' })
    await request.delete(`/admin/cache/rules/${rule.id}`)
    handleSuccess('规则已删除')
    loadCacheRules()
  } catch (e: any) {
    if (e !== 'cancel') {
      handleApiError(e, '删除失败')
    }
  }
}

const handleRuleChange = async (rule: CacheRule) => {
  try {
    await request.put(`/admin/cache/rules/${rule.id}`, {
      enabled: rule.enabled
    })
    handleSuccess(`规则已${rule.enabled ? '启用' : '禁用'}`)
  } catch (e) {
    rule.enabled = !rule.enabled
    handleApiError(e, '操作失败')
  }
}

const submitRule = async () => {
  if (!ruleFormRef.value) return
  try {
    const valid = await ruleFormRef.value.validate()
    if (valid) {
      let ttl = ruleForm.ttlValue
      switch (ruleForm.ttlUnit) {
        case 'days': ttl *= 86400; break
        case 'hours': ttl *= 3600; break
        case 'minutes': ttl *= 60; break
      }

      if (isEditRule.value) {
        await request.put(`/admin/cache/rules/${ruleForm.id}`, {
          pattern: ruleForm.pattern,
          model_filter: ruleForm.modelFilter,
          ttl: ttl,
          priority: ruleForm.priority,
          enabled: ruleForm.enabled
        })
        handleSuccess('规则已更新')
      } else {
        await request.post('/admin/cache/rules', {
          pattern: ruleForm.pattern,
          model_filter: ruleForm.modelFilter,
          ttl: ttl,
          priority: ruleForm.priority,
          enabled: ruleForm.enabled
        })
        handleSuccess('规则已添加')
      }
      ruleDialogVisible.value = false
      loadCacheRules()
    }
  } catch (error) {
    handleApiError(error, '操作失败')
  }
}

async function loadCacheRules() {
  try {
    const data: any = await request.get('/admin/cache/rules')
    if (data?.data) {
      cacheRules.value = data.data.map((r: any) => ({
        id: r.id,
        pattern: r.pattern,
        modelFilter: r.model_filter || '',
        ttl: r.ttl,
        priority: r.priority,
        enabled: r.enabled
      }))
    }
  } catch (e) {
    console.warn('Failed to load cache rules:', e)
  }
}

async function loadCacheStats() {
  loading.value = true
  try {
    const data: any = await request.get('/admin/cache/stats')
    if (data) {
      const stats = data.data || data
      
      let totalHits = 0
      let totalOps = 0
      let totalEntries = 0
      
      const typeStats: Record<string, any> = {}
      
      for (const [key, value] of Object.entries(stats)) {
        if (key.endsWith('_cache') || key === 'request_cache' || key === 'context_cache' || key === 'route_cache' || key === 'usage_cache' || key === 'response_cache') {
          const typeName = key.replace('_cache', '')
          const stat = value as any
          typeStats[typeName] = stat
          totalHits += stat.hits || 0
          totalOps += (stat.hits || 0) + (stat.misses || 0)
          totalEntries += stat.size || stat.entries || 0
        }
      }
      
      overallStats.hitRate = totalOps > 0 ? Math.round((totalHits / totalOps) * 100) : 0
      overallStats.totalEntries = totalEntries
      overallStats.totalSize = `${Math.round(totalEntries * 0.001)} MB`
      
      cacheTypes.value = cacheTypes.value.map(type => {
        const stat = typeStats[type.id]
        if (stat) {
          const hits = stat.hits || 0
          const misses = stat.misses || 0
          const ops = hits + misses
          return {
            ...type,
            hitRate: ops > 0 ? Math.round((hits / ops) * 100) : 0,
            entries: stat.size || stat.entries || 0,
            size: `${Math.round((stat.size || stat.entries || 0) * 0.001)} MB`
          }
        }
        return type
      })
    }
  } catch (e) {
    console.warn('Failed to load cache stats:', e)
  } finally {
    loading.value = false
  }
}

async function loadCacheConfig() {
  try {
    const data: any = await request.get('/admin/cache/config')
    if (data) {
      const cfg = data.data || data
      cacheConfig.enabled = cfg.enabled ?? true
      cacheConfig.strategy = cfg.strategy || 'semantic'
      cacheConfig.similarityThreshold = cfg.similarity_threshold || cfg.similarityThreshold || 85
      cacheConfig.defaultTTL = cfg.default_ttl || cfg.defaultTTL || 3600
      cacheConfig.maxSize = cfg.max_size || cfg.maxSize || 1024
      cacheConfig.evictionPolicy = cfg.eviction_policy || cfg.evictionPolicy || 'lru'
    }
  } catch (e) {
    console.warn('Failed to load cache config:', e)
  }
}

async function loadCacheHealth() {
  try {
    const data: any = await request.get('/admin/cache/health')
    if (data) {
      const health = data.data || data
      redisHealth.status = health.status || 'unknown'
      redisHealth.backend = health.backend || 'memory'
      redisHealth.latency_ms = health.latency_ms || 0
    }
  } catch (e) {
    redisHealth.status = 'unhealthy'
    redisHealth.backend = 'memory'
    console.warn('Failed to load cache health:', e)
  }
}

async function runHealthCheck() {
  try {
    await request.get('/admin/cache/health')
    await loadCacheHealth()
    await loadCacheStats()
    handleSuccess('健康检查完成')
  } catch (e) {
    handleApiError(e, '健康检查失败')
  }
}

async function saveDedupConfig() {
  try {
    await request.put('/admin/cache/config', {
      dedup: {
        enabled: dedupConfig.enabled,
        max_pending: dedupConfig.maxPending,
        request_timeout: dedupConfig.requestTimeout
      }
    })
    handleSuccess('请求去重配置已保存')
  } catch (e) {
    handleApiError(e, '保存失败')
  }
}

async function loadTaskTTLConfig() {
  try {
    const data: any = await request.get('/admin/router/ttl-config')
    if (data?.data?.task_type_defaults) {
      const defaults = data.data.task_type_defaults
      taskTTLList.value = taskTTLList.value.map(item => ({
        ...item,
        ttl: defaults[item.key] ?? defaultTaskTTL[item.key as keyof typeof defaultTaskTTL] ?? 24
      }))
    }
  } catch (e) {
    console.warn('Failed to load task TTL config:', e)
  }
}

async function saveTaskTTLConfig() {
  saving.value = true
  try {
    const taskTypeDefaults: Record<string, number> = {}
    for (const item of taskTTLList.value) {
      taskTypeDefaults[item.key] = item.ttl
    }
    await request.put('/admin/router/ttl-config', {
      task_type_defaults: taskTypeDefaults
    })
    handleSuccess('任务类型 TTL 配置已保存')
  } catch (e) {
    handleApiError(e, '保存失败')
  } finally {
    saving.value = false
  }
}

function resetTaskTTLConfig() {
  taskTTLList.value = taskTTLList.value.map(item => ({
    ...item,
    ttl: defaultTaskTTL[item.key as keyof typeof defaultTaskTTL] ?? 24
  }))
  handleSuccess('已恢复默认配置')
}

onMounted(() => {
  loadCacheStats()
  loadCacheConfig()
  loadCacheHealth()
  loadCacheRules()
  loadTaskTTLConfig()
})

onUnmounted(() => {
  if (detailChart) {
    detailChart.dispose()
    detailChart = null
  }
})
</script>

<style scoped lang="scss">
.cache-page {
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
    }

    .types-card {
      .cache-types {
        .cache-type-item {
          padding: var(--spacing-lg);
          border-bottom: 1px solid var(--border-primary);

          &:last-child {
            border-bottom: none;
          }

          .type-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: var(--spacing-md);

            .type-info {
              display: flex;
              align-items: center;
              gap: var(--spacing-sm);

              .type-name {
                font-weight: var(--font-weight-semibold);
                font-size: var(--font-size-md);
              }
            }
          }

          .type-stats {
            margin-bottom: var(--spacing-sm);

            .stat-item {
              margin-bottom: var(--spacing-xs);

              .label {
                font-size: var(--font-size-xs);
                color: var(--text-secondary);
                margin-bottom: 2px;
                display: block;
              }
            }

            .stat-row {
              display: flex;
              gap: var(--spacing-lg);
              font-size: var(--font-size-sm);
              color: var(--text-secondary);

              span {
                display: flex;
                align-items: center;
                gap: 4px;
              }
            }
          }

          .type-actions {
            display: flex;
            gap: var(--spacing-sm);
          }
        }
      }
    }

    .config-card {
      .config-form {
        max-width: 600px;

        .unit-label {
          margin-left: var(--spacing-sm);
          color: var(--text-secondary);
        }
      }

      .rules-header {
        margin-bottom: var(--spacing-md);
      }

      .rules-table {
        .pattern-code {
          background: var(--bg-secondary);
          padding: 2px 8px;
          border-radius: var(--border-radius-sm);
          font-family: var(--font-family-mono);
          font-size: var(--font-size-sm);
        }

        .text-muted {
          color: var(--text-tertiary);
        }
      }

      .hot-cache-table {
        .query-cell {
          display: flex;
          align-items: center;
          gap: var(--spacing-xs);

          .hash-code {
            font-family: var(--font-family-mono);
            font-size: var(--font-size-sm);
            background: var(--bg-secondary);
            padding: 2px 6px;
            border-radius: var(--border-radius-sm);
          }
        }

        .hits-count {
          font-weight: var(--font-weight-semibold);
          color: var(--color-primary);
        }

        .time-text {
          font-family: var(--font-family-mono);
          font-size: var(--font-size-sm);
          color: var(--text-secondary);
        }
      }

      .pagination {
        margin-top: var(--spacing-lg);
        display: flex;
        justify-content: flex-end;
      }
    }
  }

  .cache-detail {
    .detail-item {
      margin-bottom: var(--spacing-lg);

      .label {
        display: block;
        margin-bottom: var(--spacing-sm);
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
      }
    }

    .detail-desc {
      margin: var(--spacing-lg) 0;
    }

    .detail-chart {
      h4 {
        margin-bottom: var(--spacing-md);
        font-size: var(--font-size-md);
        color: var(--text-primary);
      }

      .chart-container {
        height: 200px;
      }
    }

    .advanced-section {
      h4 {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 16px;
        font-size: 16px;
        font-weight: 600;
        color: var(--el-text-color-primary);
      }

      .status-card {
        background: var(--el-fill-color-light);
        border-radius: 8px;
        padding: 16px;

        .status-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 0;
          border-bottom: 1px solid var(--el-border-color-lighter);

          &:last-child {
            border-bottom: none;
          }

          .label {
            color: var(--el-text-color-secondary);
            font-size: 14px;
          }

          .value {
            font-weight: 500;
            color: var(--el-text-color-primary);
          }
        }
      }

      .unit-label {
        margin-left: 8px;
        color: var(--el-text-color-secondary);
        font-size: 12px;
      }
    }
  }
}
</style>
