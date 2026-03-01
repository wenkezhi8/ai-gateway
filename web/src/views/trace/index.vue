<template>
  <div class="trace-page">
    <div class="trace-hero">
      <div class="hero-main">
        <div class="hero-title">请求链路追踪</div>
        <div class="hero-subtitle">实时监控请求处理全流程，透明化每一步操作</div>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="loadTraces">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-header">
        <div class="panel-title">请求列表</div>
        <div class="panel-filters">
          <el-select v-model="filter.status" placeholder="状态" clearable style="width: 120px" @change="loadTraces">
            <el-option label="全部" value="" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="error" />
          </el-select>
          <el-input v-model="filter.operation" placeholder="操作类型" clearable style="width: 200px" @change="loadTraces" />
        </div>
      </div>

      <el-table :data="traces" stripe>
        <el-table-column prop="request_id" label="Request ID" width="280" show-overflow-tooltip />
        <el-table-column prop="method" label="方法" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="getMethodType(row.method)">{{ row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="200" show-overflow-tooltip />
        <el-table-column prop="step_count" label="步骤" width="90">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">{{ row.step_count }}步</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="row.status === 'success' ? 'success' : 'danger'">
              {{ row.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="duration_ms" label="耗时" width="100">
          <template #default="{ row }">{{ row.duration_ms }}ms</template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="viewDetail(row)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[20, 50, 100]"
          layout="total, sizes, prev, pager, next"
          @change="loadTraces"
        />
      </div>
    </div>

    <!-- 详情对话框 -->
    <el-dialog v-model="detailVisible" title="请求链路详情" width="900px">
      <div v-if="detailTraces.length > 0 && detailSummary">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Request ID">{{ detailSummary.request_id }}</el-descriptions-item>
          <el-descriptions-item label="方法">{{ detailSummary.method || '-' }}</el-descriptions-item>
          <el-descriptions-item label="路径" :span="2">{{ detailSummary.path || '-' }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="detailSummary.status === 'success' ? 'success' : 'danger'">
              {{ detailSummary.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="总耗时">{{ detailSummary.duration_ms }}ms</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin-top: 20px">处理步骤</h4>
        <el-timeline>
          <el-timeline-item
            v-for="trace in detailTraces"
            :key="trace.id"
            :timestamp="formatTime(trace.start_time)"
            placement="top"
          >
            <el-card>
              <div class="trace-step">
                <div class="step-header">
                  <span class="step-operation">{{ getOperationLabel(trace.operation) }}</span>
                  <el-tag size="small" :type="trace.status === 'success' ? 'success' : 'danger'">
                    {{ trace.status === 'success' ? '成功' : '失败' }}
                  </el-tag>
                  <span class="step-duration">{{ trace.duration_ms }}ms</span>
                </div>
                <div class="step-desc">{{ getOperationDesc(trace.operation) }}</div>
                <div v-if="trace.error" class="step-error">{{ trace.error }}</div>
                <div v-if="trace.model" class="step-attr">模型: {{ trace.model }}</div>
                <div v-if="trace.provider" class="step-attr">服务商: {{ trace.provider }}</div>
                <div v-if="trace.attributes?.task_type" class="step-attr">任务类型: {{ trace.attributes.task_type }}</div>
                <div v-if="trace.attributes?.difficulty" class="step-attr">任务难度: {{ trace.attributes.difficulty }}</div>
                <div v-if="trace.attributes?.recommended_ttl" class="step-attr">推荐TTL: {{ formatTTL(trace.attributes.recommended_ttl) }}</div>
                <div v-if="trace.attributes?.answer_preview" class="step-answer">
                  <div class="step-answer-title">命中答案预览</div>
                  <div class="step-answer-content">{{ trace.attributes.answer_preview }}</div>
                  <el-button
                    v-if="trace.attributes?.answer_full"
                    type="primary"
                    link
                    size="small"
                    @click="showFullAnswer(trace)"
                  >
                    查看完整命中答案
                  </el-button>
                </div>
                <div v-if="getTraceHighlights(trace).length" class="step-highlights">
                  <el-tag
                    v-for="item in getTraceHighlights(trace)"
                    :key="item"
                    size="small"
                    effect="plain"
                    class="highlight-tag"
                  >
                    {{ item }}
                  </el-tag>
                </div>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="answerVisible" title="命中答案详情" width="900px">
      <div class="answer-full" v-if="activeAnswerTrace">
        <div class="answer-meta">
          <el-tag size="small" effect="plain">{{ getOperationLabel(activeAnswerTrace.operation) }}</el-tag>
          <el-tag size="small" :type="activeAnswerTrace.attributes?.result === 'hit' ? 'success' : 'info'">
            {{ activeAnswerTrace.attributes?.result || '-' }}
          </el-tag>
        </div>
        <pre>{{ activeAnswerTrace.attributes?.answer_full || activeAnswerTrace.attributes?.answer_preview || '-' }}</pre>
      </div>
      <template #footer>
        <el-button @click="answerVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getTraces, getTraceDetail, type RequestTrace } from '@/api/trace-domain'
import { handleApiError } from '@/utils/errorHandler'

type TraceSummary = {
  request_id: string
  method: string
  path: string
  status: string
  duration_ms: number
  created_at: string
  step_count: number
}

const traces = ref<TraceSummary[]>([])
const detailTraces = ref<RequestTrace[]>([])
const detailVisible = ref(false)
const answerVisible = ref(false)
const activeAnswerTrace = ref<RequestTrace | null>(null)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filter = ref({
  status: '',
  operation: ''
})

const OPERATION_META: Record<string, { label: string; desc: string }> = {
  'http.entry': { label: '进入网关', desc: '请求进入网关，记录基础上下文' },
  'handler.parse-request': { label: '解析请求', desc: '解析请求体并提取模型/参数' },
  'classifier.assess': { label: '任务评估', desc: '评估任务类型、难度与建议策略' },
  'cache.read-v2': { label: 'V2缓存查询', desc: '向量/意图缓存查询（高优先级）' },
  'cache.read-semantic': { label: '语义缓存查询', desc: '基于相似度匹配历史响应' },
  'cache.read-exact': { label: '精确缓存查询', desc: '基于完整Key的精确命中查询' },
  'provider.select': { label: '选择服务商', desc: '根据路由策略选择 provider 与账号' },
  'provider.chat': { label: '调用上游模型', desc: '向上游模型发送请求并等待响应' },
  'cache.write': { label: '写入缓存', desc: '将响应写入缓存（便于后续命中）' },
  'http.response': { label: '返回响应', desc: '向客户端返回最终响应' }
}

onMounted(() => {
  loadTraces()
})

const detailSummary = computed(() => {
  if (!detailTraces.value.length) return null

  const entry = detailTraces.value.find(t => t.operation === 'http.entry')
  const response = detailTraces.value.find(t => t.operation === 'http.response')
  const hasError = detailTraces.value.some(t => t.status === 'error')

  return {
    request_id: detailTraces.value[0]?.request_id || '-',
    method: entry?.method || '-',
    path: entry?.path || '-',
    status: hasError ? 'error' : 'success',
    duration_ms: response?.duration_ms || Math.max(...detailTraces.value.map(t => t.duration_ms || 0)),
  }
})

async function loadTraces() {
  try {
    loading.value = true
    const params: any = {
      limit: pageSize.value,
      offset: (page.value - 1) * pageSize.value
    }
    if (filter.value.status) params.status = filter.value.status
    if (filter.value.operation) params.operation = filter.value.operation

    const data = await getTraces(params)
    const grouped = new Map<string, RequestTrace[]>()
    for (const item of data) {
      const key = item.request_id || item.id
      if (!grouped.has(key)) grouped.set(key, [])
      grouped.get(key)!.push(item)
    }

    const rows: TraceSummary[] = []
    for (const [requestID, items] of grouped.entries()) {
      const entry = items.find(i => i.operation === 'http.entry')
      const response = items.find(i => i.operation === 'http.response')
      const hasError = items.some(i => i.status === 'error')
      rows.push({
        request_id: requestID,
        method: entry?.method || '',
        path: entry?.path || '',
        status: hasError ? 'error' : 'success',
        duration_ms: response?.duration_ms || Math.max(...items.map(i => i.duration_ms || 0)),
        created_at: entry?.created_at || items[0]?.created_at || '',
        step_count: items.length,
      })
    }

    traces.value = rows.sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
    total.value = traces.value.length
  } catch (e) {
    handleApiError(e, '加载链路数据失败')
  } finally {
    loading.value = false
  }
}

async function viewDetail(row: RequestTrace) {
  try {
    const data = await getTraceDetail(row.request_id)
    detailTraces.value = data
    detailVisible.value = true
  } catch (e) {
    handleApiError(e, '加载详情失败')
  }
}

function getMethodType(method: string) {
  const types: Record<string, string> = {
    GET: 'info',
    POST: 'success',
    PUT: 'warning',
    DELETE: 'danger'
  }
  return types[method] || 'info'
}

function formatTime(timestamp: string) {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString('zh-CN')
}

function getOperationLabel(operation: string) {
  return OPERATION_META[operation]?.label || operation
}

function getOperationDesc(operation: string) {
  return OPERATION_META[operation]?.desc || '无说明'
}

function getTraceHighlights(trace: RequestTrace) {
  const attrs = trace.attributes || {}
  const result: string[] = []

  if (attrs.result) result.push(`结果: ${attrs.result}`)
  if (attrs.hit !== undefined) result.push(`缓存命中: ${attrs.hit ? '是' : '否'}`)
  if (attrs.cache_layer) result.push(`缓存层: ${attrs.cache_layer}`)
  if (attrs.layer) result.push(`命中层: ${attrs.layer}`)
  if (attrs.similarity !== undefined) {
    result.push(`相似度: ${Number(attrs.similarity).toFixed(3)}`)
  }
  if (attrs.threshold !== undefined) {
    result.push(`阈值: ${Number(attrs.threshold).toFixed(3)}`)
  }
  if (attrs.status_code !== undefined) result.push(`状态码: ${attrs.status_code}`)

  return result.slice(0, 6)
}

function formatTTL(raw: number | string) {
  const ttl = Number(raw)
  if (!Number.isFinite(ttl) || ttl <= 0) return '-'
  if (ttl > 1e12) {
    const sec = Math.floor(ttl / 1e9)
    return `${sec}s`
  }
  return `${Math.floor(ttl)}s`
}

function showFullAnswer(trace: RequestTrace) {
  activeAnswerTrace.value = trace
  answerVisible.value = true
}
</script>

<style scoped>
.trace-page {
  padding: 20px;
}

.trace-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 8px;
  color: white;
}

.hero-title {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;
}

.hero-subtitle {
  opacity: 0.9;
}

.panel {
  background: white;
  border-radius: 8px;
  padding: 20px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.panel-title {
  font-size: 18px;
  font-weight: 600;
}

.panel-filters {
  display: flex;
  gap: 12px;
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}

.trace-step {
  padding: 8px 0;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.step-operation {
  font-weight: 600;
  font-size: 16px;
}

.step-duration {
  color: #666;
  font-size: 14px;
}

.op-name {
  font-weight: 600;
}

.op-desc {
  color: #8a8f99;
  font-size: 12px;
  margin-top: 2px;
}

.step-desc {
  color: #6b7280;
  font-size: 13px;
  margin: 6px 0 8px;
}

.step-highlights {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 8px;
}

.highlight-tag {
  margin-right: 0;
}

.step-answer {
  margin-top: 10px;
  padding: 10px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #f8fafc;
}

.step-answer-title {
  font-size: 12px;
  color: #475569;
  margin-bottom: 6px;
}

.step-answer-content {
  white-space: pre-wrap;
  line-height: 1.5;
  color: #111827;
  margin-bottom: 6px;
}

.answer-full pre {
  background: #0f172a;
  color: #e2e8f0;
  padding: 14px;
  border-radius: 8px;
  max-height: 520px;
  overflow: auto;
  white-space: pre-wrap;
  line-height: 1.6;
}

.answer-meta {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.step-error {
  color: #f56c6c;
  background: #fef0f0;
  padding: 8px;
  border-radius: 4px;
  margin-top: 8px;
}

.step-attr {
  color: #666;
  font-size: 14px;
  margin-top: 4px;
}
</style>
