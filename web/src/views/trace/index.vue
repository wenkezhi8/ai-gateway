<template>
  <div class="trace-page">
    <div class="trace-hero">
      <div class="hero-main">
        <div class="hero-title">请求链路追踪</div>
        <div class="hero-subtitle">实时监控请求处理全流程，透明化每一步操作</div>
      </div>
      <div class="hero-actions">
        <el-button
          type="danger"
          plain
          :loading="clearing"
          :disabled="loading || clearing"
          @click="handleClearTraces"
        >
          清理链路记录
        </el-button>
        <el-button type="primary" :loading="loading" @click="loadTraces">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-header">
        <div class="panel-title">请求列表</div>
        <div class="panel-filters">
          <el-select
            v-model="filter.status"
            placeholder="状态"
            clearable
            style="width: 120px"
            @change="handleFilterChange"
          >
            <el-option label="全部" value="" />
            <el-option label="成功" value="success" />
            <el-option label="失败" value="error" />
          </el-select>
          <el-input
            v-model="filter.operation"
            placeholder="操作类型"
            clearable
            style="width: 200px"
            @change="handleFilterChange"
          />
        </div>
      </div>

      <el-table :data="traces" stripe v-loading="loading">
        <el-table-column prop="created_at" label="时间" width="160">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
        <el-table-column prop="task_type" label="任务类型" width="120">
          <template #default="{ row }">
            {{ row.task_type ? formatTaskType(row.task_type) : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="method" label="方法" width="80">
          <template #default="{ row }">
            <el-tag size="small" :type="getMethodType(row.method)">{{ row.method }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路径" min-width="200" show-overflow-tooltip />
        <el-table-column prop="model" label="模型" width="120">
          <template #default="{ row }">{{ row.model || '-' }}</template>
        </el-table-column>
        <el-table-column prop="provider" label="服务商" width="180">
          <template #default="{ row }">
            <div v-if="row.provider" class="provider-cell">
              <img
                v-if="getProviderMeta(row.provider).logo"
                :src="getProviderMeta(row.provider).logo"
                :alt="getProviderLabel(row.provider)"
                class="provider-logo"
              />
              <div
                v-else
                class="provider-logo provider-fallback"
                :style="{ backgroundColor: getProviderMeta(row.provider).color }"
              >
                {{ getProviderInitial(row.provider) }}
              </div>
              <span class="provider-label">{{ getProviderLabel(row.provider) }}</span>
            </div>
            <span v-else>{{ row.provider || '-' }}</span>
          </template>
        </el-table-column>
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
          <template #default="{ row }">{{ formatDuration(row.duration_ms) }}</template>
        </el-table-column>
        <el-table-column prop="answer_source" label="AI回复来源" width="120">
          <template #default="{ row }">
            <el-tag size="small" effect="plain">
              {{ getAnswerSourceLabel(row.answer_source) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="request_id" label="Request ID" width="280" show-overflow-tooltip />
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
    <el-dialog v-model="detailVisible" title="请求链路详情" width="900px" v-loading="detailLoading">
      <div v-if="detailTraces.length > 0 && detailSummary">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Request ID">
            {{ detailSummary.request_id }}
          </el-descriptions-item>
          <el-descriptions-item label="方法">
            {{ detailSummary.method || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="路径" :span="2">
            {{ detailSummary.path || '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="detailSummary.status === 'success' ? 'success' : 'danger'">
              {{ detailSummary.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="总耗时">
            {{ formatDuration(detailSummary.duration_ms) }}
          </el-descriptions-item>
          <el-descriptions-item label="服务商" :span="2">
            <div v-if="detailSummary.provider" class="provider-cell">
              <img
                v-if="getProviderMeta(detailSummary.provider).logo"
                :src="getProviderMeta(detailSummary.provider).logo"
                :alt="getProviderLabel(detailSummary.provider)"
                class="provider-logo"
              />
              <div
                v-else
                class="provider-logo provider-fallback"
                :style="{ backgroundColor: getProviderMeta(detailSummary.provider).color }"
              >
                {{ getProviderInitial(detailSummary.provider) }}
              </div>
              <span class="provider-label">{{ getProviderLabel(detailSummary.provider) }}</span>
            </div>
            <span v-else>{{ detailSummary.provider || '-' }}</span>
          </el-descriptions-item>
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
                  <span class="step-duration">{{ formatDuration(trace.duration_ms) }}</span>
                </div>
                <div class="step-desc">{{ getOperationDesc(trace.operation) }}</div>
                <div v-if="trace.error" class="step-error">{{ trace.error }}</div>
                <div v-if="trace.model" class="step-attr">模型: {{ trace.model }}</div>
                <div v-if="trace.attributes?.provider_type" class="step-attr">
                  协议类型: {{ trace.attributes.provider_type }}
                </div>
                <div v-if="trace.attributes?.provider_name" class="step-attr">
                  服务商: {{ trace.attributes.provider_name }}
                </div>
                <div v-else-if="trace.provider" class="step-attr">服务商: {{ trace.provider }}</div>
                <div v-if="trace.attributes?.task_type" class="step-attr">
                  任务类型: {{ formatTaskType(trace.attributes.task_type) }}
                </div>
                <div v-if="trace.attributes?.difficulty" class="step-attr">
                  任务难度: {{ formatDifficulty(trace.attributes.difficulty) }}
                </div>
                <div v-if="trace.attributes?.recommended_ttl" class="step-attr">
                  推荐TTL: {{ formatTTL(trace.attributes.recommended_ttl) }}
                </div>
                <div v-if="trace.attributes?.answer_preview" class="step-answer">
                  <div class="step-answer-title">命中答案预览</div>
                  <div class="step-answer-content">{{ trace.attributes.answer_preview }}</div>
                  <el-button type="primary" link size="small" @click="showFullAnswer(trace)">
                    查看完整答案
                  </el-button>
                </div>
                <div v-if="trace.attributes?.user_message_raw_preview" class="step-answer">
                  <div class="step-answer-title">原始请求预览</div>
                  <div class="step-answer-content">
                    {{ trace.attributes.user_message_raw_preview }}
                  </div>
                  <el-button
                    type="primary"
                    link
                    size="small"
                    @click="showFullMessage(trace, 'user_raw')"
                  >
                    查看完整原始请求
                  </el-button>
                </div>
                <div v-if="trace.attributes?.user_message_preview" class="step-answer">
                  <div class="step-answer-title">清洗后问题</div>
                  <div class="step-answer-content">{{ trace.attributes.user_message_preview }}</div>
                  <el-button
                    type="primary"
                    link
                    size="small"
                    @click="showFullMessage(trace, 'user')"
                  >
                    查看完整问题
                  </el-button>
                </div>
                <div v-if="trace.attributes?.ai_response_preview" class="step-answer">
                  <div class="step-answer-title">AI 回复预览</div>
                  <div class="step-answer-content">{{ trace.attributes.ai_response_preview }}</div>
                  <el-button type="primary" link size="small" @click="showFullMessage(trace, 'ai')">
                    查看完整回复
                  </el-button>
                </div>
                <div v-if="getTraceHighlights(trace).length" class="step-highlights">
                  <el-tag
                    v-for="highlight in getTraceHighlights(trace)"
                    :key="highlight"
                    size="small"
                    effect="plain"
                    class="highlight-tag"
                  >
                    {{ highlight }}
                  </el-tag>
                </div>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-dialog>

    <el-dialog v-model="answerVisible" title="完整 AI 回复" width="760px">
      <div v-if="activeAnswerTrace" class="answer-full">
        <div class="answer-meta">
          <el-tag size="small">{{ getOperationLabel(activeAnswerTrace.operation) }}</el-tag>
          <el-tag size="small" effect="plain">
            {{ formatTime(activeAnswerTrace.start_time) }}
          </el-tag>
          <el-tag
            size="small"
            :type="activeAnswerTrace.status === 'success' ? 'success' : 'danger'"
          >
            {{ activeAnswerTrace.status === 'success' ? '成功' : '失败' }}
          </el-tag>
        </div>
        <pre>{{
          activeAnswerTrace.attributes?.answer_full ||
          activeAnswerTrace.attributes?.answer_preview ||
          '-'
        }}</pre>
      </div>
      <template #footer>
        <el-button @click="answerVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="messageVisible" :title="messageDialogTitle" width="760px">
      <pre class="message-full">{{ activeMessageContent || '-' }}</pre>
      <template #footer>
        <el-button @click="messageVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  clearTraces,
  getTraces,
  getTraceDetail,
  type RequestTrace,
  type TraceSummary
} from '@/api/trace-domain'
import {
  CHAT_PROVIDER_VISUALS,
  CHAT_PROVIDER_VISUAL_FALLBACK,
  type ChatProviderVisualMeta
} from '@/constants/store/chat'
import {
  TRACE_ANSWER_SOURCE_FALLBACK,
  TRACE_ANSWER_SOURCE_LABELS
} from '@/constants/trace-answer-source'
import { handleApiError } from '@/utils/errorHandler'

const traces = ref<TraceSummary[]>([])
const detailTraces = ref<RequestTrace[]>([])
const detailVisible = ref(false)
const answerVisible = ref(false)
const activeAnswerTrace = ref<RequestTrace | null>(null)
const messageVisible = ref(false)
const messageDialogTitle = ref('')
const activeMessageContent = ref('')
const loading = ref(false)
const clearing = ref(false)
const detailLoading = ref(false)
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

const TASK_TYPE_LABELS: Record<string, string> = {
  coding: '编程',
  math: '数学',
  creative: '创作',
  analysis: '分析',
  chat: '通用对话'
}

const DIFFICULTY_LABELS: Record<string, string> = {
  low: '低',
  medium: '中',
  high: '高'
}

onMounted(() => {
  loadTraces()
})

const detailSummary = computed(() => {
  if (!detailTraces.value.length) return null

  const entry = detailTraces.value.find(t => t.operation === 'http.entry')
  const response = detailTraces.value.find(t => t.operation === 'http.response')
  const providerTrace = response || detailTraces.value.find(t => t.provider)
  const hasError = detailTraces.value.some(t => t.status === 'error')

  return {
    request_id: detailTraces.value[0]?.request_id || '-',
    method: entry?.method || '-',
    path: entry?.path || '-',
    status: hasError ? 'error' : 'success',
    duration_ms:
      response?.duration_ms || Math.max(...detailTraces.value.map(t => t.duration_ms || 0)),
    provider: providerTrace?.provider || ''
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

    const result = await getTraces(params)
    traces.value = result.data
    total.value = result.total
  } catch (e) {
    handleApiError(e, '加载链路数据失败')
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  page.value = 1
  void loadTraces()
}

async function handleClearTraces() {
  try {
    await ElMessageBox.confirm('确定要清理全部链路记录吗？该操作不可恢复。', '清理链路记录', {
      type: 'warning',
      confirmButtonText: '确认清理',
      cancelButtonText: '取消'
    })
  } catch {
    return
  }

  try {
    clearing.value = true
    const result = await clearTraces()
    ElMessage.success(`已清理链路记录，共删除 ${result.deleted} 条`)
    page.value = 1
    detailVisible.value = false
    detailTraces.value = []
    activeAnswerTrace.value = null
    answerVisible.value = false
    messageVisible.value = false
    activeMessageContent.value = ''
    await loadTraces()
  } catch (e) {
    handleApiError(e, '清理链路记录失败')
  } finally {
    clearing.value = false
  }
}

async function viewDetail(row: TraceSummary) {
  try {
    detailLoading.value = true
    const data = await getTraceDetail(row.request_id)
    detailTraces.value = data
    detailVisible.value = true
  } catch (e) {
    handleApiError(e, '加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

function getAnswerSourceLabel(source: TraceSummary['answer_source']) {
  return TRACE_ANSWER_SOURCE_LABELS[source] || source || TRACE_ANSWER_SOURCE_FALLBACK
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

function getProviderMeta(provider?: string): ChatProviderVisualMeta {
  const key = provider?.trim().toLowerCase()
  if (!key) {
    return {
      ...CHAT_PROVIDER_VISUAL_FALLBACK,
      logo: ''
    }
  }

  const meta = CHAT_PROVIDER_VISUALS[key]
  if (meta) return meta

  return {
    label: provider || CHAT_PROVIDER_VISUAL_FALLBACK.label,
    color: CHAT_PROVIDER_VISUAL_FALLBACK.color,
    logo: ''
  }
}

function getProviderLabel(provider?: string) {
  return getProviderMeta(provider).label
}

function getProviderInitial(provider?: string) {
  const source = (provider || '').trim()
  if (!source) return 'A'
  return source.charAt(0).toUpperCase()
}

function formatTime(timestamp: string) {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleString('zh-CN')
}

function formatDuration(durationMs: number) {
  const seconds = Number(durationMs || 0) / 1000
  return `${seconds.toFixed(3)}s`
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

function showFullMessage(trace: RequestTrace, kind: 'user_raw' | 'user' | 'ai') {
  const attrs = trace.attributes || {}
  if (kind === 'user_raw') {
    messageDialogTitle.value = '原始请求全文'
    activeMessageContent.value = String(
      attrs.user_message_raw_full || attrs.user_message_raw_preview || '-'
    )
  } else if (kind === 'user') {
    messageDialogTitle.value = '清洗后问题全文'
    activeMessageContent.value = String(
      attrs.user_message_full || attrs.user_message_preview || '-'
    )
  } else {
    messageDialogTitle.value = 'AI 回复全文'
    activeMessageContent.value = String(attrs.ai_response_full || attrs.ai_response_preview || '-')
  }
  messageVisible.value = true
}

function formatTaskType(taskType: string) {
  return TASK_TYPE_LABELS[taskType] || taskType
}

function formatDifficulty(difficulty: string) {
  return DIFFICULTY_LABELS[difficulty] || difficulty
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

.provider-cell {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.provider-logo {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  object-fit: cover;
  flex: 0 0 auto;
}

.provider-fallback {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
}

.provider-label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.answer-full pre,
.message-full {
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
