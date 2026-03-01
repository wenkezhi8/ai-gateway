<template>
  <div class="vector-backup-page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar-row">
        <div>
          <h2>备份恢复管理</h2>
          <p>管理向量集合的备份任务与恢复任务</p>
        </div>
        <div class="toolbar-actions">
          <el-button @click="loadTasks" :loading="loading">刷新</el-button>
        </div>
      </div>

      <el-form :model="createForm" inline>
        <el-form-item label="Collection">
          <el-input v-model="createForm.collection_name" placeholder="如 docs" style="width: 200px" />
        </el-form-item>
        <el-form-item label="快照名">
          <el-input v-model="createForm.snapshot_name" placeholder="可选，默认自动生成" style="width: 260px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="creating" @click="handleCreate">创建备份</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="content-card">
      <template v-if="loading">
        <el-skeleton :rows="5" animated />
      </template>
      <template v-else-if="error">
        <el-empty description="备份任务加载失败">
          <el-button type="primary" @click="loadTasks">重试</el-button>
        </el-empty>
      </template>
      <template v-else-if="tasks.length === 0">
        <el-empty description="暂无备份任务" />
      </template>
      <el-table v-else :data="tasks" stripe>
        <el-table-column prop="id" label="任务ID" width="100" />
        <el-table-column prop="collection_name" label="Collection" min-width="140" />
        <el-table-column prop="snapshot_name" label="快照名" min-width="180" show-overflow-tooltip />
        <el-table-column prop="action" label="动作" width="100" />
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updated_at" label="更新时间" min-width="180" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="handleRestore(row.id)">恢复</el-button>
            <el-button link type="warning" @click="handleRetry(row.id)">重试</el-button>
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
  createBackupTask,
  listBackupTasks,
  retryBackupTask,
  triggerBackupRestore,
  type BackupTask
} from '@/api/vector-db-domain'

const loading = ref(false)
const creating = ref(false)
const error = ref(false)
const tasks = ref<BackupTask[]>([])

const createForm = reactive({
  collection_name: '',
  snapshot_name: ''
})

async function loadTasks() {
  loading.value = true
  error.value = false
  try {
    const data = await listBackupTasks({ limit: 100 })
    tasks.value = data.items || []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.collection_name.trim()) {
    ElMessage.warning('请输入 Collection 名称')
    return
  }
  creating.value = true
  try {
    await createBackupTask({
      collection_name: createForm.collection_name,
      snapshot_name: createForm.snapshot_name || undefined
    })
    ElMessage.success('备份任务创建成功')
    createForm.snapshot_name = ''
    await loadTasks()
  } catch {
    ElMessage.error('备份任务创建失败')
  } finally {
    creating.value = false
  }
}

async function handleRestore(id: number) {
  try {
    await triggerBackupRestore(id)
    ElMessage.success('恢复任务已触发')
    await loadTasks()
  } catch {
    ElMessage.error('恢复任务触发失败')
  }
}

async function handleRetry(id: number) {
  try {
    await retryBackupTask(id)
    ElMessage.success('任务重试已触发')
    await loadTasks()
  } catch {
    ElMessage.error('任务重试失败')
  }
}

function statusType(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'danger'
  if (status === 'running') return 'warning'
  return 'info'
}

void loadTasks()
</script>

<style scoped>
.vector-backup-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.toolbar-row h2 {
  margin: 0;
}

.toolbar-row p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}
</style>
