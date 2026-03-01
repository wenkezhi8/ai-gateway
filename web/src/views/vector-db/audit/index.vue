<template>
  <div class="vector-audit-page">
    <el-card shadow="never">
      <div class="toolbar">
        <h2>向量审计日志</h2>
        <el-button :loading="loading" @click="loadLogs">刷新</el-button>
      </div>
      <div class="filters">
        <el-input v-model="filters.resource_type" placeholder="资源类型" clearable style="width: 180px" />
        <el-input v-model="filters.resource_id" placeholder="资源ID" clearable style="width: 220px" />
        <el-input v-model="filters.action" placeholder="动作" clearable style="width: 220px" />
        <el-button type="primary" @click="loadLogs">查询</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <template v-if="loading">
        <el-skeleton :rows="5" animated />
      </template>
      <template v-else-if="error">
        <el-empty description="审计日志加载失败">
          <el-button type="primary" @click="loadLogs">重试</el-button>
        </el-empty>
      </template>
      <template v-else-if="logs.length === 0">
        <el-empty description="暂无审计日志" />
      </template>
      <el-table v-else :data="logs" stripe>
        <el-table-column prop="created_at" label="时间" min-width="180" />
        <el-table-column prop="user_id" label="用户" width="120" />
        <el-table-column prop="resource_type" label="资源类型" width="140" />
        <el-table-column prop="resource_id" label="资源ID" min-width="180" show-overflow-tooltip />
        <el-table-column prop="action" label="动作" min-width="180" show-overflow-tooltip />
        <el-table-column prop="details" label="详情" min-width="260" show-overflow-tooltip />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { listVectorAuditLogs, type AuditLogItem } from '@/api/vector-db-domain'

const loading = ref(false)
const error = ref(false)
const logs = ref<AuditLogItem[]>([])
const filters = reactive({ resource_type: '', resource_id: '', action: '' })

async function loadLogs() {
  loading.value = true
  error.value = false
  try {
    const data = await listVectorAuditLogs({
      resource_type: filters.resource_type || undefined,
      resource_id: filters.resource_id || undefined,
      action: filters.action || undefined,
      limit: 100
    })
    logs.value = data.items || []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

void loadLogs()
</script>

<style scoped>
.vector-audit-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

h2 {
  margin: 0;
}

.filters {
  margin-top: 12px;
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
</style>
