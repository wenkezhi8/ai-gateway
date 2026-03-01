<template>
  <div class="knowledge-documents-page">
    <el-card>
      <template #header>
        <div class="header-row">
          <div>
            <div class="title">知识库文档管理</div>
            <div class="subtitle">上传文档后自动分块，支持后续问答检索。</div>
          </div>
          <div class="actions">
            <el-input
              v-model="search"
              clearable
              placeholder="按文档名搜索"
              style="width: 220px"
              @keyup.enter="reload"
            />
            <el-select v-model="status" clearable placeholder="状态筛选" style="width: 160px" @change="reload">
              <el-option label="处理中" value="processing" />
              <el-option label="已完成" value="completed" />
              <el-option label="失败" value="failed" />
            </el-select>
            <el-upload
              :show-file-list="false"
              :before-upload="beforeUpload"
              :http-request="uploadDocument"
              accept=".txt,.md,.pdf,.doc,.docx,.xls,.xlsx,.html"
            >
              <el-button type="primary" :loading="uploading">上传文档</el-button>
            </el-upload>
          </div>
        </div>
      </template>

      <div v-if="error" class="error-box">
        <el-alert :title="error" type="error" show-icon :closable="false" />
        <el-button style="margin-top: 10px" @click="reload">重试</el-button>
      </div>

      <el-skeleton v-else-if="loading" :rows="6" animated />

      <el-empty v-else-if="documents.length === 0" description="暂无知识库文档，先上传一份文件" />

      <div v-else>
        <el-table :data="documents" stripe>
          <el-table-column prop="name" label="文档名" min-width="220" />
          <el-table-column prop="type" label="类型" width="100" />
          <el-table-column prop="size" label="大小" width="120">
            <template #default="{ row }">{{ prettySize(row.size) }}</template>
          </el-table-column>
          <el-table-column prop="chunk_count" label="分块" width="80" />
          <el-table-column prop="status" label="状态" width="110">
            <template #default="{ row }">
              <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="updated_at" label="更新时间" width="180">
            <template #default="{ row }">{{ formatTime(row.updated_at) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="220" fixed="right">
            <template #default="{ row }">
              <el-button link type="primary" @click="viewDocument(row.id)">详情</el-button>
              <el-button link type="warning" @click="reVectorize(row.id)">重建向量</el-button>
              <el-button link type="danger" @click="removeDocument(row.id)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="pager">
          <el-pagination
            v-model:current-page="page"
            v-model:page-size="pageSize"
            :total="total"
            :page-sizes="[10, 20, 50]"
            layout="total, sizes, prev, pager, next"
            @change="reload"
          />
        </div>
      </div>
    </el-card>

    <el-drawer v-model="drawerVisible" title="文档详情" size="50%">
      <div v-if="drawerLoading">加载中...</div>
      <div v-else-if="drawerDoc">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="文档名">{{ drawerDoc.name }}</el-descriptions-item>
          <el-descriptions-item label="集合">{{ drawerDoc.collection_id }}</el-descriptions-item>
          <el-descriptions-item label="分块数">{{ drawerDoc.chunk_count }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ drawerDoc.status }}</el-descriptions-item>
        </el-descriptions>
        <h4 class="chunk-title">分块预览</h4>
        <el-empty v-if="!drawerDoc.chunks || drawerDoc.chunks.length === 0" description="暂无分块数据" />
        <el-timeline v-else>
          <el-timeline-item v-for="item in drawerDoc.chunks" :key="item.id" :timestamp="item.id">
            <pre class="chunk-content">{{ item.content }}</pre>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox, type UploadRequestOptions } from 'element-plus'
import {
  deleteKnowledgeDocument,
  getKnowledgeDocument,
  listKnowledgeDocuments,
  uploadKnowledgeDocument,
  vectorizeKnowledgeDocument,
  type KnowledgeDocument
} from '@/api/knowledge-domain'

const loading = ref(false)
const uploading = ref(false)
const error = ref('')

const documents = ref<KnowledgeDocument[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const search = ref('')
const status = ref('')

const drawerVisible = ref(false)
const drawerLoading = ref(false)
const drawerDoc = ref<any>(null)

onMounted(() => {
  reload()
})

async function reload() {
  loading.value = true
  error.value = ''
  try {
    const data = await listKnowledgeDocuments({
      page: page.value,
      page_size: pageSize.value,
      search: search.value || undefined,
      status: status.value || undefined
    })
    documents.value = data.items || []
    total.value = data.total || 0
  } catch (e: any) {
    error.value = e?.message || '加载文档列表失败'
  } finally {
    loading.value = false
  }
}

function beforeUpload(rawFile: File) {
  const allow = rawFile.size <= 20 * 1024 * 1024
  if (!allow) {
    ElMessage.error('单个文件不能超过20MB')
  }
  return allow
}

async function uploadDocument(options: UploadRequestOptions) {
  uploading.value = true
  try {
    const file = options.file as File
    await uploadKnowledgeDocument(file)
    ElMessage.success('上传成功')
    await reload()
    options.onSuccess?.({})
  } catch (e: any) {
    options.onError?.(e)
    ElMessage.error(e?.message || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function viewDocument(id: string) {
  drawerVisible.value = true
  drawerLoading.value = true
  try {
    drawerDoc.value = await getKnowledgeDocument(id)
  } finally {
    drawerLoading.value = false
  }
}

async function reVectorize(id: string) {
  await vectorizeKnowledgeDocument(id)
  ElMessage.success('已触发向量重建')
  await reload()
}

async function removeDocument(id: string) {
  await ElMessageBox.confirm('删除后不可恢复，确认删除？', '警告', { type: 'warning' })
  await deleteKnowledgeDocument(id)
  ElMessage.success('删除成功')
  await reload()
}

function formatTime(value: string) {
  if (!value) return '-'
  return new Date(value).toLocaleString('zh-CN')
}

function prettySize(size: number) {
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${(size / 1024 / 1024).toFixed(1)} MB`
}

function statusType(s: string) {
  if (s === 'completed') return 'success'
  if (s === 'processing') return 'warning'
  if (s === 'failed') return 'danger'
  return 'info'
}
</script>

<style scoped>
.knowledge-documents-page {
  padding: 8px;
}

.header-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
}

.title {
  font-size: 18px;
  font-weight: 600;
}

.subtitle {
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.error-box {
  margin-bottom: 12px;
}

.pager {
  margin-top: 14px;
  display: flex;
  justify-content: flex-end;
}

.chunk-title {
  margin: 18px 0 12px;
}

.chunk-content {
  white-space: pre-wrap;
  margin: 0;
  font-size: 12px;
  line-height: 1.6;
}
</style>
