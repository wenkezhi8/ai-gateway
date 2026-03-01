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

    <el-card class="rules-card">
      <template #header>
        <div class="rules-header">
          <span>告警规则</span>
          <el-button type="primary" size="small" @click="createDefaultRule">新增默认规则</el-button>
        </div>
      </template>

      <el-alert v-if="error" :title="error" type="error" show-icon class="state" />
      <el-empty v-else-if="!loading && rules.length === 0" description="暂无告警规则" class="state" />
      <el-table v-else v-loading="loading" :data="rules" border>
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="规则名" min-width="160" />
        <el-table-column prop="metric" label="指标" min-width="140" />
        <el-table-column prop="operator" label="操作符" width="100" />
        <el-table-column prop="threshold" label="阈值" width="100" />
        <el-table-column prop="duration" label="持续时间" width="120" />
        <el-table-column label="状态" width="120">
          <template #default="scope">
            <el-tag :type="scope.row.enabled ? 'success' : 'info'">{{ scope.row.enabled ? '启用' : '停用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="scope">
            <el-button link type="danger" @click="removeRule(scope.row.id)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import type { AlertRule, VectorMetricsSummary } from '@/api/vector-db-domain'
import { createAlertRule, deleteAlertRule, getVectorMetricsSummary, listAlertRules } from '@/api/vector-db-domain'

const loading = ref(false)
const error = ref('')
const rules = ref<AlertRule[]>([])
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
    const [summaryResp, rulesResp] = await Promise.all([getVectorMetricsSummary(), listAlertRules()])
    Object.assign(summary, summaryResp)
    rules.value = rulesResp.rules || []
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载失败'
    rules.value = []
  } finally {
    loading.value = false
  }
}

async function createDefaultRule() {
  await createAlertRule({
    name: `rule-${Date.now()}`,
    metric: 'search_p95_ms',
    operator: 'gt',
    threshold: 500,
    duration: '5m',
    channels: ['webhook'],
    enabled: true
  })
  await load()
}

async function removeRule(id: number) {
  await deleteAlertRule(id)
  await load()
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

.metric {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.rules-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.state {
  margin: 10px 0;
}
</style>
