<template>
  <div class="vector-db-page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar-row">
        <div class="toolbar-title">
          <h2>向量集合管理</h2>
          <p>统一管理通用向量数据库 Collection，支持内部业务与系统级检索场景</p>
        </div>
        <div class="toolbar-actions">
          <el-button @click="loadCollections" :loading="loading">刷新</el-button>
          <el-button @click="openImportDialog">新建导入任务</el-button>
          <el-button type="primary" @click="openCreateDialog">新建 Collection</el-button>
        </div>
      </div>
      <div class="toolbar-filters">
        <el-input
          v-model="filters.search"
          placeholder="搜索名称或描述"
          clearable
          style="width: 280px"
          @keyup.enter="loadCollections"
        />
        <el-select v-model="filters.environment" clearable placeholder="环境" style="width: 150px">
          <el-option label="生产" value="production" />
          <el-option label="测试" value="staging" />
          <el-option label="开发" value="dev" />
        </el-select>
        <el-select v-model="filters.status" clearable placeholder="状态" style="width: 150px">
          <el-option label="激活" value="active" />
          <el-option label="禁用" value="inactive" />
          <el-option label="归档" value="archived" />
        </el-select>
        <el-button type="primary" plain @click="loadCollections">查询</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <template v-if="loading">
        <el-skeleton :rows="6" animated />
      </template>

      <template v-else-if="error">
        <el-empty description="加载失败，请重试">
          <el-button type="primary" @click="loadCollections">重新加载</el-button>
        </el-empty>
      </template>

      <template v-else-if="collections.length === 0">
        <el-empty description="暂无 Collection 数据">
          <el-button type="primary" @click="openCreateDialog">创建第一个 Collection</el-button>
        </el-empty>
      </template>

      <el-table v-else :data="collections" stripe>
        <el-table-column prop="name" label="名称" min-width="180" />
        <el-table-column prop="description" label="描述" min-width="220" show-overflow-tooltip />
        <el-table-column prop="dimension" label="维度" width="90" />
        <el-table-column prop="distance_metric" label="距离度量" width="120" />
        <el-table-column prop="index_type" label="索引" width="100" />
        <el-table-column prop="storage_backend" label="存储" width="100" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="向量数" width="120">
          <template #default="{ row }">{{ formatNumber(row.vector_count) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditDialog(row)">编辑</el-button>
            <el-button link type="warning" @click="openIndexDialog(row)">索引配置</el-button>
            <el-popconfirm title="确认清空该 Collection 的全部向量吗？" @confirm="handleEmpty(row)">
              <template #reference>
                <el-button link type="warning">清空</el-button>
              </template>
            </el-popconfirm>
            <el-popconfirm title="确认删除该 Collection 吗？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" class="content-card">
      <template #header>
        <div class="section-header">
          <span>导入任务</span>
          <div class="section-actions">
            <el-button size="small" plain :type="jobFilters.status === 'failed' ? 'danger' : 'default'" @click="toggleFailedOnly">
              {{ jobFilters.status === 'failed' ? '取消仅看失败' : '仅看失败任务' }}
            </el-button>
            <el-select v-model="jobFilters.status" clearable placeholder="任务状态" style="width: 140px" @change="loadImportJobs">
              <el-option label="待处理" value="pending" />
              <el-option label="运行中" value="running" />
              <el-option label="已完成" value="completed" />
              <el-option label="失败" value="failed" />
              <el-option label="重试中" value="retrying" />
            </el-select>
            <el-button size="small" type="warning" plain @click="handleRetryFailedJobs" :loading="batchRetryLoading">批量重试失败任务</el-button>
            <el-button size="small" @click="loadImportJobs" :loading="jobsLoading">刷新任务</el-button>
          </div>
        </div>
      </template>

      <div class="job-summary-grid">
        <div class="job-summary-card is-pending">
          <span>待处理</span>
          <strong>{{ jobSummary.pending }}</strong>
        </div>
        <div class="job-summary-card is-running">
          <span>运行中/重试中</span>
          <strong>{{ jobSummary.running + jobSummary.retrying }}</strong>
        </div>
        <div class="job-summary-card is-failed">
          <span>失败</span>
          <strong>{{ jobSummary.failed }}</strong>
        </div>
        <div class="job-summary-card is-completed">
          <span>已完成</span>
          <strong>{{ jobSummary.completed }}</strong>
        </div>
      </div>

      <template v-if="jobsLoading">
        <el-skeleton :rows="4" animated />
      </template>

      <template v-else-if="jobsError">
        <el-empty description="导入任务加载失败">
          <el-button type="primary" @click="loadImportJobs">重试</el-button>
        </el-empty>
      </template>

      <template v-else-if="importJobs.length === 0">
        <el-empty description="暂无导入任务">
          <el-button type="primary" @click="openImportDialog">创建导入任务</el-button>
        </el-empty>
      </template>

      <el-table v-else :data="importJobs" stripe>
        <el-table-column prop="id" label="任务ID" min-width="180" show-overflow-tooltip />
        <el-table-column prop="collection_name" label="Collection" min-width="140" />
        <el-table-column prop="file_name" label="文件" min-width="160" show-overflow-tooltip />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="jobStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="进度" width="140">
          <template #default="{ row }">{{ row.processed_records }}/{{ row.total_records }}</template>
        </el-table-column>
        <el-table-column label="失败" width="90">
          <template #default="{ row }">{{ row.failed_records }}</template>
        </el-table-column>
        <el-table-column label="重试" width="100">
          <template #default="{ row }">{{ row.retry_count }}/{{ row.max_retries }}</template>
        </el-table-column>
        <el-table-column label="重试上限" width="120">
          <template #default="{ row }">
            <el-tooltip v-if="isRetryExceeded(row)" content="已达最大重试次数，需人工处理源文件或配置后再新建任务" placement="top">
              <el-tag type="danger">已达上限</el-tag>
            </el-tooltip>
            <el-tag v-else type="success">可重试</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button link @click="openJobDetail(row)">详情</el-button>
            <el-button link type="primary" @click="handleRunJob(row.id)">运行</el-button>
            <el-button link type="warning" :disabled="isRetryExceeded(row)" @click="handleRetryJob(row.id)">重试</el-button>
            <el-button link type="info" @click="openErrorDialog(row.id)">错误</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="editing ? '编辑 Collection' : '新建 Collection'" width="560px">
      <el-form :model="form" label-width="110px">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" :disabled="editing" />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="维度" required>
          <el-input-number v-model="form.dimension" :min="1" :disabled="editing" />
        </el-form-item>
        <el-form-item label="距离度量">
          <el-select v-model="form.distance_metric">
            <el-option label="cosine" value="cosine" />
            <el-option label="euclid" value="euclid" />
            <el-option label="dot" value="dot" />
          </el-select>
        </el-form-item>
        <el-form-item label="索引类型">
          <el-select v-model="form.index_type">
            <el-option label="hnsw" value="hnsw" />
            <el-option label="ivf" value="ivf" />
          </el-select>
        </el-form-item>
        <el-form-item label="环境">
          <el-input v-model="form.environment" />
        </el-form-item>
        <el-form-item label="公开访问">
          <el-switch v-model="form.is_public" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="submitForm">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="importDialogVisible" title="新建导入任务" width="560px">
      <el-form :model="importForm" label-width="120px">
        <el-form-item label="Collection" required>
          <el-select v-model="importForm.collection_name" filterable placeholder="选择 Collection">
            <el-option v-for="item in collections" :key="item.name" :label="item.name" :value="item.name" />
          </el-select>
        </el-form-item>
        <el-form-item label="文件名" required>
          <el-input v-model="importForm.file_name" placeholder="如 docs.json" />
        </el-form-item>
        <el-form-item label="文件路径" required>
          <el-input v-model="importForm.file_path" placeholder="如 /tmp/docs.json" />
        </el-form-item>
        <el-form-item label="文件大小(Byte)" required>
          <el-input-number v-model="importForm.file_size" :min="1" :step="1024" />
        </el-form-item>
        <el-form-item label="记录数" required>
          <el-input-number v-model="importForm.total_records" :min="1" />
        </el-form-item>
        <el-form-item label="最大重试">
          <el-input-number v-model="importForm.max_retries" :min="1" :max="10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="importDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="importSaving" @click="submitImportJob">提交任务</el-button>
      </template>
    </el-dialog>

    <IndexSettingsDialog
      v-model="indexDialogVisible"
      :loading="indexSaving"
      :collection="indexTarget"
      @save="submitIndexSettings"
    />

    <el-dialog v-model="errorDialogVisible" title="导入错误摘要" width="680px">
      <template v-if="errorLogs.length === 0">
        <el-empty description="暂无错误记录" />
      </template>
      <el-timeline v-else>
        <el-timeline-item v-for="item in errorLogs" :key="item.id" :timestamp="formatDate(item.created_at)">
          <div class="error-log-item">
            <strong>{{ item.action }}</strong>
            <p>{{ item.details || '-' }}</p>
          </div>
        </el-timeline-item>
      </el-timeline>
      <template #footer>
        <el-button @click="errorDialogVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="jobDetailVisible" title="导入任务详情" width="760px">
      <template v-if="jobDetail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="任务ID">{{ jobDetail.id }}</el-descriptions-item>
          <el-descriptions-item label="Collection">{{ jobDetail.collection_name || jobDetail.collection_id }}</el-descriptions-item>
          <el-descriptions-item label="文件名">{{ jobDetail.file_name }}</el-descriptions-item>
          <el-descriptions-item label="文件路径">{{ jobDetail.file_path }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="jobStatusType(jobDetail.status)">{{ jobDetail.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="最近运行时间">{{ formatDate(resolveLastRunAt(jobDetail)) }}</el-descriptions-item>
          <el-descriptions-item label="重试策略" :span="2">{{ buildRetryHint(jobDetail) }}</el-descriptions-item>
          <el-descriptions-item label="错误摘要" :span="2">
            <div class="detail-error-toolbar">
              <el-select v-model="detailErrorAction" clearable placeholder="按动作筛选" style="width: 220px" @change="reloadJobDetailErrors">
                <el-option label="运行失败" value="import_run_failed" />
                <el-option label="写入失败" value="import_upsert_failed" />
                <el-option label="重试超限" value="import_retry_exceeded" />
              </el-select>
              <el-button size="small" @click="copyDetailErrors">复制错误摘要</el-button>
              <el-button size="small" type="primary" plain @click="exportDetailErrors">导出 .txt</el-button>
              <el-segmented
                v-model="detailErrorLimit"
                :options="[
                  { label: '最近5条', value: 5 },
                  { label: '最近20条', value: 20 }
                ]"
                @change="reloadJobDetailErrors"
              />
            </div>
            <div class="detail-error-meta">
              <span>筛选条件：{{ detailErrorActionLabel() }}</span>
              <span>已加载：{{ jobDetailErrors.length }} 条</span>
            </div>
            <template v-if="detailErrorLoading">加载中...</template>
            <template v-else-if="jobDetailErrors.length === 0">暂无错误记录</template>
            <template v-else>
              <div class="detail-error-scroll" @scroll.passive="onDetailErrorScroll">
                <div v-for="group in groupedDetailErrorLogs" :key="group.date" class="detail-error-group">
                  <div class="detail-error-group-title">{{ group.date }}</div>
                  <ul class="detail-error-list">
                    <li v-for="item in group.items" :key="item.id">{{ item.action }} - {{ item.details || '-' }}</li>
                  </ul>
                </div>
              </div>
              <div class="detail-error-footer">
                <el-button link :disabled="!detailErrorsHasMore || detailErrorLoadingMore" @click="loadMoreJobDetailErrors">
                  {{ detailErrorLoadingMore ? '加载中...' : detailErrorsHasMore ? '加载更多' : '没有更多了' }}
                </el-button>
                <el-button v-if="detailLoadMoreFailed" link type="danger" @click="loadMoreJobDetailErrors">加载失败，点击重试</el-button>
              </div>
            </template>
          </el-descriptions-item>
        </el-descriptions>
      </template>
      <template #footer>
        <el-button @click="jobDetailVisible = false">关闭</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import IndexSettingsDialog from './IndexSettingsDialog.vue'
import {
  createImportJob,
  createVectorCollection,
  emptyVectorCollection,
  deleteVectorCollection,
  getImportJobErrors,
  getImportJobSummary,
  listImportJobs,
  listVectorCollections,
  retryFailedImportJobs,
  retryImportJob,
  runImportJob,
  updateVectorCollection,
  type ImportJob,
  type ImportJobErrorLog,
  type ImportJobSummary,
  type ImportJobStatus,
  type VectorCollection
} from '@/api/vector-db-domain'
import { canRetryImportJob, normalizeImportJobStatus } from './import-job-utils'
import {
  buildImportJobErrorSummaryText,
  buildImportJobErrorExportFileName,
  buildRetryHint,
  filterImportJobErrorsByDateRange,
  groupImportJobErrorsByDate,
  mergeImportJobErrorLogs,
  normalizeImportJobErrorAction,
  resolveImportJobErrorDateRange,
  resolveLastRunAt
} from './import-job-utils'

const loading = ref(false)
const saving = ref(false)
const error = ref(false)
const collections = ref<VectorCollection[]>([])
const importJobs = ref<ImportJob[]>([])
const jobsLoading = ref(false)
const jobsError = ref(false)
const batchRetryLoading = ref(false)
const importDialogVisible = ref(false)
const importSaving = ref(false)
const errorDialogVisible = ref(false)
const errorLogs = ref<ImportJobErrorLog[]>([])
const jobDetailVisible = ref(false)
const jobDetail = ref<ImportJob | null>(null)
const jobDetailErrors = ref<ImportJobErrorLog[]>([])
const detailErrorLoading = ref(false)
const detailErrorLoadingMore = ref(false)
const detailErrorsHasMore = ref(false)
const detailLoadMoreFailed = ref(false)
const detailErrorAction = ref('')
const detailErrorLimit = ref(5)
const detailErrorOffset = ref(0)
const groupedDetailErrorLogs = computed(() => groupImportJobErrorsByDate(jobDetailErrors.value))
const indexDialogVisible = ref(false)
const indexSaving = ref(false)
const indexTarget = ref<VectorCollection | null>(null)
const jobSummary = ref<ImportJobSummary>({
  pending: 0,
  running: 0,
  retrying: 0,
  completed: 0,
  failed: 0,
  cancelled: 0,
  total: 0
})
const jobFilters = reactive({
  status: '' as '' | ImportJobStatus
})

const filters = reactive({
  search: '',
  environment: '',
  status: ''
})

const dialogVisible = ref(false)
const editing = ref(false)
const editingName = ref('')
const form = reactive({
  name: '',
  description: '',
  dimension: 1536,
  distance_metric: 'cosine',
  index_type: 'hnsw',
  storage_backend: 'qdrant',
  environment: 'production',
  is_public: false
})

const importForm = reactive({
  collection_name: '',
  file_name: '',
  file_path: '',
  file_size: 1024,
  total_records: 100,
  max_retries: 3
})

function resetForm() {
  form.name = ''
  form.description = ''
  form.dimension = 1536
  form.distance_metric = 'cosine'
  form.index_type = 'hnsw'
  form.storage_backend = 'qdrant'
  form.environment = 'production'
  form.is_public = false
}

function openCreateDialog() {
  editing.value = false
  editingName.value = ''
  resetForm()
  dialogVisible.value = true
}

function openEditDialog(row: VectorCollection) {
  editing.value = true
  editingName.value = row.name
  form.name = row.name
  form.description = row.description || ''
  form.dimension = row.dimension
  form.distance_metric = row.distance_metric || 'cosine'
  form.index_type = row.index_type || 'hnsw'
  form.storage_backend = row.storage_backend || 'qdrant'
  form.environment = row.environment || 'production'
  form.is_public = !!row.is_public
  dialogVisible.value = true
}

function openIndexDialog(row: VectorCollection) {
  indexTarget.value = row
  indexDialogVisible.value = true
}

async function loadCollections() {
  loading.value = true
  error.value = false
  try {
    const data = await listVectorCollections({
      search: filters.search || undefined,
      environment: filters.environment || undefined,
      status: filters.status || undefined
    })
    collections.value = data.collections || []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

async function loadImportJobs() {
  jobsLoading.value = true
  jobsError.value = false
  try {
    const data = await listImportJobs({
      limit: 50,
      status: normalizeImportJobStatus(jobFilters.status)
    })
    importJobs.value = data.jobs || []
    const summary = await getImportJobSummary()
    jobSummary.value = {
      pending: summary.pending || 0,
      running: summary.running || 0,
      retrying: summary.retrying || 0,
      completed: summary.completed || 0,
      failed: summary.failed || 0,
      cancelled: summary.cancelled || 0,
      total: summary.total || 0
    }
  } catch {
    jobsError.value = true
  } finally {
    jobsLoading.value = false
  }
}

async function submitForm() {
  if (!form.name.trim()) {
    ElMessage.warning('请输入 Collection 名称')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateVectorCollection(editingName.value, {
        description: form.description,
        distance_metric: form.distance_metric,
        index_type: form.index_type,
        storage_backend: form.storage_backend,
        environment: form.environment,
        is_public: form.is_public
      })
      ElMessage.success('Collection 更新成功')
    } else {
      await createVectorCollection({
        name: form.name,
        description: form.description,
        dimension: form.dimension,
        distance_metric: form.distance_metric,
        index_type: form.index_type,
        storage_backend: form.storage_backend,
        environment: form.environment,
        is_public: form.is_public
      })
      ElMessage.success('Collection 创建成功')
    }
    dialogVisible.value = false
    await loadCollections()
  } catch {
    ElMessage.error('保存失败，请重试')
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: VectorCollection) {
  try {
    await deleteVectorCollection(row.name)
    ElMessage.success('删除成功')
    await loadCollections()
  } catch {
    ElMessage.error('删除失败，请重试')
  }
}

async function handleEmpty(row: VectorCollection) {
  try {
    await emptyVectorCollection(row.name)
    ElMessage.success('清空成功')
    await loadCollections()
  } catch {
    ElMessage.error('清空失败，请重试')
  }
}

async function submitIndexSettings(payload: {
  index_type: string
  hnsw_m: number
  hnsw_ef_construct: number
  ivf_nlist: number
}) {
  if (!indexTarget.value) {
    return
  }
  indexSaving.value = true
  try {
    await updateVectorCollection(indexTarget.value.name, payload)
    ElMessage.success('索引配置更新成功')
    indexDialogVisible.value = false
    await loadCollections()
  } catch {
    ElMessage.error('索引配置更新失败')
  } finally {
    indexSaving.value = false
  }
}

function openImportDialog() {
  importForm.collection_name = collections.value[0]?.name || ''
  importForm.file_name = ''
  importForm.file_path = ''
  importForm.file_size = 1024
  importForm.total_records = 100
  importForm.max_retries = 3
  importDialogVisible.value = true
}

async function submitImportJob() {
  if (!importForm.collection_name || !importForm.file_name || !importForm.file_path) {
    ElMessage.warning('请完整填写导入任务信息')
    return
  }
  importSaving.value = true
  try {
    await createImportJob({
      collection_name: importForm.collection_name,
      file_name: importForm.file_name,
      file_path: importForm.file_path,
      file_size: importForm.file_size,
      total_records: importForm.total_records,
      max_retries: importForm.max_retries
    })
    ElMessage.success('导入任务创建成功')
    importDialogVisible.value = false
    await loadImportJobs()
  } catch {
    ElMessage.error('导入任务创建失败')
  } finally {
    importSaving.value = false
  }
}

async function handleRunJob(id: string) {
  try {
    await runImportJob(id)
    ElMessage.success('任务已执行')
    await loadImportJobs()
  } catch {
    ElMessage.error('任务执行失败')
  }
}

async function handleRetryJob(id: string) {
  try {
    await retryImportJob(id)
    ElMessage.success('任务已重试')
    await loadImportJobs()
  } catch {
    ElMessage.error('任务重试失败')
  }
}

async function handleRetryFailedJobs() {
  batchRetryLoading.value = true
  try {
    const data = await retryFailedImportJobs(20)
    ElMessage.success(`批量重试完成，本次触发 ${data.total || 0} 个任务`)
    await loadImportJobs()
  } catch {
    ElMessage.error('批量重试失败')
  } finally {
    batchRetryLoading.value = false
  }
}

function toggleFailedOnly() {
  jobFilters.status = jobFilters.status === 'failed' ? '' : 'failed'
  void loadImportJobs()
}

async function openErrorDialog(id: string) {
  try {
    const data = await getImportJobErrors(id, 20)
    errorLogs.value = data.logs || []
    errorDialogVisible.value = true
  } catch {
    ElMessage.error('错误日志加载失败')
  }
}

async function openJobDetail(job: ImportJob) {
  jobDetail.value = job
  detailErrorAction.value = ''
  detailErrorLimit.value = 5
  detailErrorOffset.value = 0
  detailErrorsHasMore.value = false
  detailErrorLoadingMore.value = false
  detailLoadMoreFailed.value = false
  jobDetailVisible.value = true
  await reloadJobDetailErrors()
}

async function reloadJobDetailErrors() {
  if (!jobDetail.value) {
    return
  }
  detailErrorLoading.value = true
  detailLoadMoreFailed.value = false
  try {
    detailErrorOffset.value = 0
    const data = await getImportJobErrors(jobDetail.value.id, detailErrorLimit.value, detailErrorAction.value || undefined, detailErrorOffset.value)
    jobDetailErrors.value = data.logs || []
    detailErrorsHasMore.value = (data.logs || []).length >= detailErrorLimit.value
  } catch {
    jobDetailErrors.value = []
    detailErrorsHasMore.value = false
  } finally {
    detailErrorLoading.value = false
  }
}

async function loadMoreJobDetailErrors() {
  if (!jobDetail.value || detailErrorLoadingMore.value || !detailErrorsHasMore.value) {
    return
  }

  detailErrorLoadingMore.value = true
  detailLoadMoreFailed.value = false
  try {
    const nextOffset = jobDetailErrors.value.length
    const data = await getImportJobErrors(jobDetail.value.id, detailErrorLimit.value, detailErrorAction.value || undefined, nextOffset)
    const incoming = data.logs || []
    const merged = mergeImportJobErrorLogs(jobDetailErrors.value, incoming)
    jobDetailErrors.value = merged
    detailErrorOffset.value = merged.length
    detailErrorsHasMore.value = incoming.length >= detailErrorLimit.value
  } catch {
    detailLoadMoreFailed.value = true
  } finally {
    detailErrorLoadingMore.value = false
  }
}

function onDetailErrorScroll(event: Event) {
  if (!detailErrorsHasMore.value || detailErrorLoadingMore.value) {
    return
  }
  const target = event.target
  if (!(target instanceof HTMLElement)) {
    return
  }
  const remain = target.scrollHeight - target.scrollTop - target.clientHeight
  if (remain <= 24) {
    void loadMoreJobDetailErrors()
  }
}

function detailErrorActionLabel() {
  if (detailErrorAction.value === 'import_run_failed') return '运行失败'
  if (detailErrorAction.value === 'import_upsert_failed') return '写入失败'
  if (detailErrorAction.value === 'import_retry_exceeded') return '重试超限'
  return '全部动作'
}

async function copyDetailErrors() {
  const text = buildImportJobErrorSummaryText(jobDetailErrors.value)
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.focus()
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
    ElMessage.success('错误摘要已复制')
  } catch {
    ElMessage.error('复制失败，请手动复制')
  }
}

function exportDetailErrors() {
  const dateRange = resolveImportJobErrorDateRange(groupedDetailErrorLogs.value)
  const visibleLogs = filterImportJobErrorsByDateRange(jobDetailErrors.value, dateRange)
  const summary = buildImportJobErrorSummaryText(visibleLogs)
  const generatedAt = new Date().toISOString()
  const collectionName = jobDetail.value?.collection_name || jobDetail.value?.collection_id || 'unknown'
  const status = jobDetail.value?.status || 'unknown'
  const retryProgress = `${jobDetail.value?.retry_count ?? 0}/${jobDetail.value?.max_retries ?? 0}`
  const visibleRange = dateRange ? `${dateRange.startDate} ~ ${dateRange.endDate}` : '未知'
  const normalizedAction = normalizeImportJobErrorAction(detailErrorAction.value || undefined)
  const content = [
    '# 导入任务错误摘要',
    `任务ID: ${jobDetail.value?.id || 'unknown'}`,
    `Collection: ${collectionName}`,
    `任务状态: ${status}`,
    `重试进度: ${retryProgress}`,
    `筛选条件: ${detailErrorActionLabel()}`,
    `筛选动作值: ${normalizedAction}`,
    `可见日期范围: ${visibleRange}`,
    `导出条数: ${visibleLogs.length}`,
    `导出时间: ${generatedAt}`,
    '',
    summary
  ].join('\n')

  const fileName = buildImportJobErrorExportFileName(jobDetail.value?.id || 'unknown', detailErrorAction.value || undefined)
  const blob = new Blob(['\uFEFF', content], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')

  anchor.href = url
  anchor.download = fileName
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(url)

  ElMessage.success('错误摘要已导出')
}

function jobStatusType(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running' || status === 'retrying') return 'warning'
  return 'info'
}

function isRetryExceeded(job: ImportJob) {
  return !canRetryImportJob(job)
}

function formatDate(value?: string) {
  if (!value) return '-'
  const time = new Date(value)
  if (Number.isNaN(time.getTime())) return value
  return time.toLocaleString('zh-CN')
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

void loadCollections()
void loadImportJobs()
</script>

<style scoped>
.vector-db-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar-card,
.content-card {
  border-radius: 14px;
}

.toolbar-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
}

.toolbar-title h2 {
  margin: 0;
  font-size: 20px;
}

.toolbar-title p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}

.toolbar-actions {
  display: flex;
  gap: 8px;
}

.toolbar-filters {
  margin-top: 14px;
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.job-summary-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 12px;
}

.job-summary-card {
  border-radius: 10px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.job-summary-card span {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.job-summary-card strong {
  font-size: 20px;
}

.job-summary-card.is-pending {
  background: #f4f7ff;
}

.job-summary-card.is-running {
  background: #fff8ec;
}

.job-summary-card.is-failed {
  background: #fff1f0;
}

.job-summary-card.is-completed {
  background: #effcf4;
}

@media (max-width: 960px) {
  .job-summary-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.error-log-item p {
  margin: 6px 0 0;
  color: var(--el-text-color-regular);
}

.detail-error-list {
  margin: 0;
  padding-left: 18px;
}

.detail-error-group {
  padding-bottom: 8px;
}

.detail-error-group-title {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.detail-error-scroll {
  max-height: 220px;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  padding: 8px 10px;
}

.detail-error-meta {
  margin-bottom: 8px;
  display: flex;
  justify-content: space-between;
  gap: 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.detail-error-footer {
  margin-top: 8px;
}

.detail-error-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}
</style>
