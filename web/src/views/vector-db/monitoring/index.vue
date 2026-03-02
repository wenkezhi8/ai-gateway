<template>
  <div class="vector-monitoring-page">
    <el-row :gutter="12" class="summary-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card><div class="metric"><span>集合总数</span><strong>{{ summary.collections_total }}</strong></div></el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card><div class="metric"><span>导入任务总数</span><strong>{{ summary.import_jobs.total }}</strong></div></el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card><div class="metric"><span>告警规则总数</span><strong>{{ summary.alert_rules_total }}</strong></div></el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card><div class="metric"><span>启用规则</span><strong>{{ summary.enabled_rules }}</strong></div></el-card>
      </el-col>
    </el-row>

    <el-card>
      <div class="alert-shortcut">
        <div>
          <h3>告警配置</h3>
          <p>告警规则管理已拆分到独立页面，便于按项目书结构维护。</p>
        </div>
        <el-button type="primary" @click="goAlerts">前往告警管理页</el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { VectorMetricsSummary } from '@/api/vector-db-domain'
import { getVectorMetricsSummary } from '@/api/vector-db-domain'

const router = useRouter()
const loading = ref(false)
const error = ref('')
const summary = reactive<VectorMetricsSummary>({
  collections_total: 0,
  import_jobs: { pending: 0, running: 0, retrying: 0, completed: 0, failed: 0, cancelled: 0, total: 0 },
  alert_rules_total: 0,
  enabled_rules: 0
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const summaryResp = await getVectorMetricsSummary()
    Object.assign(summary, summaryResp)
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function goAlerts() {
  void router.push('/vector-db/monitoring/alerts')
}

onMounted(() => {
  void load()
})
</script>

<style scoped>
.vector-monitoring-page {
  padding: 12px;
}

.summary-row {
  margin-bottom: 12px;
}

.alert-shortcut {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.alert-shortcut h3 {
  margin: 0;
}

.alert-shortcut p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}

.metric {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
</style>
