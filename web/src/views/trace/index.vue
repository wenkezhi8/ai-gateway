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
        <el-table-column prop="operation" label="操作" width="150" />
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
      <div v-if="detailTraces.length > 0 && detailTraces[0]">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Request ID">{{ detailTraces[0]?.request_id }}</el-descriptions-item>
          <el-descriptions-item label="方法">{{ detailTraces[0]?.method }}</el-descriptions-item>
          <el-descriptions-item label="路径" :span="2">{{ detailTraces[0]?.path }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="detailTraces[0]?.status === 'success' ? 'success' : 'danger'">
              {{ detailTraces[0]?.status === 'success' ? '成功' : '失败' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="总耗时">{{ detailTraces[0]?.duration_ms }}ms</el-descriptions-item>
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
                  <span class="step-operation">{{ trace.operation }}</span>
                  <el-tag size="small" :type="trace.status === 'success' ? 'success' : 'danger'">
                    {{ trace.status === 'success' ? '成功' : '失败' }}
                  </el-tag>
                  <span class="step-duration">{{ trace.duration_ms }}ms</span>
                </div>
                <div v-if="trace.error" class="step-error">{{ trace.error }}</div>
                <div v-if="trace.model" class="step-attr">模型: {{ trace.model }}</div>
                <div v-if="trace.provider" class="step-attr">服务商: {{ trace.provider }}</div>
              </div>
            </el-card>
          </el-timeline-item>
        </el-timeline>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
import { getTraces, getTraceDetail, type RequestTrace } from '@/api/trace-domain'
import { handleApiError } from '@/utils/errorHandler'

const traces = ref<RequestTrace[]>([])
const detailTraces = ref<RequestTrace[]>([])
const detailVisible = ref(false)
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const filter = ref({
  status: '',
  operation: ''
})

onMounted(() => {
  loadTraces()
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
    traces.value = data
    total.value = data.length
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
