<template>
  <div class="vector-permissions-page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar">
        <div>
          <h2>向量权限管理</h2>
          <p>为向量检索 API 配置角色权限（reader/admin）</p>
        </div>
        <el-button @click="loadPermissions" :loading="loading">刷新</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="content-card">
      <el-form :model="createForm" inline>
        <el-form-item label="API Key">
          <el-input v-model="createForm.api_key" placeholder="输入新 API Key" style="width: 300px" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="createForm.role" style="width: 140px">
            <el-option label="reader" value="reader" />
            <el-option label="admin" value="admin" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="handleCreate">新增权限</el-button>
        </el-form-item>
      </el-form>

      <template v-if="loading">
        <el-skeleton :rows="4" animated />
      </template>
      <template v-else-if="error">
        <el-empty description="权限数据加载失败">
          <el-button type="primary" @click="loadPermissions">重试</el-button>
        </el-empty>
      </template>
      <template v-else-if="items.length === 0">
        <el-empty description="暂无权限配置" />
      </template>
      <el-table v-else :data="items" stripe>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="role" label="角色" width="120" />
        <el-table-column label="启用" width="100">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'">{{ row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-popconfirm title="确认删除该权限吗？" @confirm="handleDelete(row.id)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createVectorPermission,
  deleteVectorPermission,
  listVectorPermissions,
  type VectorPermissionItem
} from '@/api/vector-db-domain'

const loading = ref(false)
const saving = ref(false)
const error = ref(false)
const items = ref<VectorPermissionItem[]>([])

const createForm = reactive({
  api_key: '',
  role: 'reader'
})

async function loadPermissions() {
  loading.value = true
  error.value = false
  try {
    const data = await listVectorPermissions()
    items.value = data.items || []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.api_key.trim()) {
    ElMessage.warning('请输入 API Key')
    return
  }
  saving.value = true
  try {
    await createVectorPermission({ api_key: createForm.api_key, role: createForm.role })
    createForm.api_key = ''
    ElMessage.success('权限创建成功')
    await loadPermissions()
  } catch {
    ElMessage.error('权限创建失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await deleteVectorPermission(id)
    ElMessage.success('权限删除成功')
    await loadPermissions()
  } catch {
    ElMessage.error('权限删除失败')
  }
}

void loadPermissions()
</script>

<style scoped>
.vector-permissions-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.toolbar h2 {
  margin: 0;
}

.toolbar p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}
</style>
