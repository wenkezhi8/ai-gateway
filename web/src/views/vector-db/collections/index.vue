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
            <el-popconfirm title="确认删除该 Collection 吗？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
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
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createVectorCollection,
  deleteVectorCollection,
  listVectorCollections,
  updateVectorCollection,
  type VectorCollection
} from '@/api/vector-db-domain'

const loading = ref(false)
const saving = ref(false)
const error = ref(false)
const collections = ref<VectorCollection[]>([])

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

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value || 0)
}

void loadCollections()
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
</style>
