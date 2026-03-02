<template>
  <div class="vector-monitoring-alerts-page">
    <el-card>
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
import { onMounted, ref } from 'vue'
import type { AlertRule } from '@/api/vector-db-domain'
import { createAlertRule, deleteAlertRule, listAlertRules } from '@/api/vector-db-domain'

const loading = ref(false)
const error = ref('')
const rules = ref<AlertRule[]>([])

async function loadRules() {
  loading.value = true
  error.value = ''
  try {
    const rulesResp = await listAlertRules()
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
  await loadRules()
}

async function removeRule(id: number) {
  await deleteAlertRule(id)
  await loadRules()
}

onMounted(() => {
  void loadRules()
})
</script>

<style scoped>
.vector-monitoring-alerts-page {
  padding: 12px;
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
