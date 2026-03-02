<template>
  <div class="vector-import-page">
    <el-card shadow="never">
      <div class="header-row">
        <div>
          <h2>向量导入中心</h2>
          <p>统一查看导入任务状态，并快速跳转到集合页发起新任务。</p>
        </div>
        <el-button type="primary" @click="goToCollections">前往集合页新建任务</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span>导入器</span>
      </template>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="JSON 导入" name="json">
          <JsonImporter :collections="collections" :submitting="creating" @create="onCreateJob" />
        </el-tab-pane>
        <el-tab-pane label="CSV 导入" name="csv">
          <CsvImporter :collections="collections" :submitting="creating" @create="onCreateJob" />
        </el-tab-pane>
        <el-tab-pane label="PDF 导入" name="pdf">
          <PdfImporter :collections="collections" :submitting="creating" @create="onCreateJob" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="header-row">
          <span>文件上传与字段映射</span>
          <el-tag type="info">字段映射</el-tag>
        </div>
      </template>
      <el-upload
        drag
        action="#"
        :auto-upload="false"
        :show-file-list="false"
        :on-change="handleUploadChange"
      >
        <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
        <div class="el-upload__text">拖拽文件到此处，或 <em>点击上传</em></div>
      </el-upload>
      <el-table :data="fieldMappings" stripe style="margin-top: 12px">
        <el-table-column prop="source" label="源字段" min-width="180" />
        <el-table-column prop="target" label="目标字段" min-width="180" />
      </el-table>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <div class="header-row">
          <span>最近导入任务</span>
          <el-button :loading="loading" @click="loadJobs">刷新</el-button>
        </div>
      </template>

      <template v-if="loading">
        <el-skeleton :rows="4" animated />
      </template>
      <template v-else-if="error">
        <el-empty description="任务加载失败">
          <el-button type="primary" @click="loadJobs">重试</el-button>
        </el-empty>
      </template>
      <template v-else-if="jobs.length === 0">
        <el-empty description="暂无导入任务" />
      </template>
      <el-table v-else :data="jobs" stripe>
        <el-table-column prop="id" label="任务ID" min-width="180" show-overflow-tooltip />
        <el-table-column prop="collection_name" label="Collection" min-width="150" />
        <el-table-column prop="file_name" label="文件" min-width="160" show-overflow-tooltip />
        <el-table-column prop="status" label="状态" width="120" />
        <el-table-column label="进度" width="140">
          <template #default="{ row }">{{ row.processed_records }}/{{ row.total_records }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button
              v-if="canCancel(row.status)"
              link
              type="danger"
              :loading="cancelingId === row.id"
              @click="onCancelJob(row.id)"
            >
              取消
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import {
  cancelImportJob,
  createImportJob,
  listImportJobs,
  listVectorCollections,
  type CreateImportJobPayload,
  type ImportJob,
  type ImportJobStatus
} from '@/api/vector-db-domain'
import JsonImporter from './JsonImporter.vue'
import CsvImporter from './CsvImporter.vue'
import PdfImporter from './PdfImporter.vue'

const router = useRouter()
const loading = ref(false)
const error = ref(false)
const jobs = ref<ImportJob[]>([])
const collections = ref<string[]>([])
const activeTab = ref('json')
const creating = ref(false)
const cancelingId = ref('')
const fieldMappings = reactive([
  { source: 'text', target: 'content' },
  { source: 'title', target: 'title' },
  { source: 'tags', target: 'tags' }
])

function handleUploadChange(file: { name?: string }) {
  const fileName = file.name || ''
  if (!fileName) {
    return
  }
  ElMessage.success(`已选择文件：${fileName}`)
}

async function loadJobs() {
  loading.value = true
  error.value = false
  try {
    const data = await listImportJobs({ limit: 20 })
    jobs.value = data.jobs || []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

async function loadCollections() {
  try {
    const data = await listVectorCollections({ limit: 200 })
    collections.value = (data.collections || []).map((item) => item.name)
  } catch {
    collections.value = []
  }
}

function canCancel(status: ImportJobStatus) {
  return status === 'pending' || status === 'running' || status === 'retrying'
}

async function onCreateJob(payload: CreateImportJobPayload) {
  if (!payload.collection_name || !payload.file_name || !payload.file_path) {
    ElMessage.warning('请先补全导入参数')
    return
  }
  creating.value = true
  try {
    await createImportJob(payload)
    ElMessage.success('导入任务已创建')
    await loadJobs()
  } finally {
    creating.value = false
  }
}

async function onCancelJob(id: string) {
  cancelingId.value = id
  try {
    await cancelImportJob(id)
    ElMessage.success('导入任务已取消')
    await loadJobs()
  } finally {
    cancelingId.value = ''
  }
}

function goToCollections() {
  void router.push('/vector-db/collections')
}

void loadCollections()
void loadJobs()
</script>

<style scoped>
.vector-import-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

h2 {
  margin: 0;
  font-size: 20px;
}

p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}
</style>
