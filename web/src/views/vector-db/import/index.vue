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
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { listImportJobs, type ImportJob } from '@/api/vector-db-domain'

const router = useRouter()
const loading = ref(false)
const error = ref(false)
const jobs = ref<ImportJob[]>([])

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

function goToCollections() {
  void router.push('/vector-db/collections')
}

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
