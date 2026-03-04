<template>
  <div class="edition-selector">
    <div class="section-title">版本管理</div>

    <div class="edition-cards">
      <div
        v-for="edition in editionStore.definitions"
        :key="edition.type"
        class="edition-card"
        :class="{ active: selectedEdition === edition.type, disabled: !canSelectEdition(edition.type) }"
        @click="handleSelectEdition(edition.type)"
      >
        <div class="edition-header">
          <span class="edition-name">{{ edition.display_name }}</span>
          <el-tag v-if="edition.type === 'enterprise'" type="danger" size="small">推荐</el-tag>
        </div>
        <div class="edition-description">{{ edition.description }}</div>
        <div class="edition-dependencies">
          <el-tag
            v-for="dep in edition.dependencies"
            :key="dep"
            size="small"
            :type="dependencyHealthy(dep) ? 'success' : 'info'"
          >
            {{ dep.toUpperCase() }}
          </el-tag>
        </div>
      </div>
    </div>

    <div class="actions">
      <el-button type="primary" :loading="editionStore.updating" @click="handleSave">保存配置</el-button>
      <el-button @click="refreshDependencies">刷新依赖状态</el-button>
    </div>

    <div class="setup-actions">
      <el-select v-model="setupEdition" placeholder="安装目标版本" style="width: 160px" :disabled="setupRunning">
        <el-option
          v-for="edition in editionStore.definitions"
          :key="`setup-${edition.type}`"
          :label="edition.display_name"
          :value="edition.type"
        />
      </el-select>
      <el-select v-model="runtime" placeholder="安装模式" style="width: 140px">
        <el-option label="Docker" value="docker" />
        <el-option label="Native" value="native" />
      </el-select>
      <el-checkbox v-model="applyConfig">回写配置</el-checkbox>
      <el-checkbox v-model="pullEmbeddingModel">拉取 embedding 模型</el-checkbox>
      <el-button type="success" :loading="setupRunning" @click="handleSetupDependencies">安装依赖</el-button>
    </div>

    <div v-if="runtime === 'native'" class="runtime-hint">
      native 模式：缺少依赖时不会自动切换到 Docker，请先手动安装并启动 Redis / Ollama / Qdrant。
    </div>

    <div v-if="editionStore.setupTask" class="setup-status">
      <el-tag size="small" :type="setupStatusTag(editionStore.setupTask.status)">
        {{ editionStore.setupTask.status }}
      </el-tag>
      <span class="setup-summary">{{ editionStore.setupTask.summary || editionStore.setupTask.message || '-' }}</span>
    </div>

    <div v-if="editionStore.setupTask?.logs" class="setup-log-panel">
      <div class="setup-log-header">
        <span>安装过程日志</span>
        <el-button text size="small" @click="copySetupLogs">复制日志</el-button>
      </div>
      <pre ref="setupLogRef" class="setup-log-content">{{ editionStore.setupTask.logs }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'

import { useEditionStore } from '@/store/domain/edition'
import type { EditionSetupStatus, EditionSetupRuntime, EditionType } from '@/api/edition-domain'

const editionStore = useEditionStore()
const selectedEdition = ref<EditionType>('standard')
const setupEdition = ref<EditionType>('standard')
const runtime = ref<EditionSetupRuntime>('docker')
const applyConfig = ref(true)
const pullEmbeddingModel = ref(false)
const setupTaskId = ref('')
const setupLogRef = ref<HTMLElement | null>(null)
let setupPollTimer: ReturnType<typeof setTimeout> | null = null

const definitionsReady = computed(() => editionStore.definitions.length > 0)
const setupRunning = computed(() => {
  if (editionStore.setupLoading) return true
  const status = editionStore.setupTask?.status
  return status === 'pending' || status === 'running'
})

onMounted(async () => {
  await Promise.all([
    editionStore.fetchEditionConfig(),
    editionStore.fetchDefinitions(),
    refreshDependencies()
  ])

  selectedEdition.value = editionStore.config?.type ?? 'standard'
  setupEdition.value = selectedEdition.value
  runtime.value = editionStore.config?.runtime ?? 'docker'
})

watch(
  () => editionStore.setupTask?.logs,
  () => {
    nextTick(() => {
      if (!setupLogRef.value) return
      setupLogRef.value.scrollTop = setupLogRef.value.scrollHeight
    })
  }
)

function dependencyHealthy(dep: string): boolean {
  return editionStore.dependencies[dep]?.healthy ?? false
}

function canSelectEdition(type: EditionType): boolean {
  const item = editionStore.definitions.find((d) => d.type === type)
  if (!item) return false
  return item.dependencies.every((dep) => dependencyHealthy(dep))
}

function handleSelectEdition(type: EditionType) {
  if (!definitionsReady.value) return
  if (!canSelectEdition(type)) {
    ElMessage.warning('目标版本依赖未满足，请先检查依赖服务')
    return
  }
  selectedEdition.value = type
}

async function refreshDependencies() {
  await editionStore.checkDependencies()
}

async function handleSave() {
  const result = await editionStore.updateEdition(selectedEdition.value)
  if (!result.success) {
    ElMessage.error(result.message || '更新版本失败')
    return
  }
  ElMessage.success('版本已更新')
}

function setupStatusTag(status: EditionSetupStatus) {
  switch (status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'running':
      return 'warning'
    default:
      return 'info'
  }
}

function clearSetupPoll() {
  if (setupPollTimer) {
    clearTimeout(setupPollTimer)
    setupPollTimer = null
  }
}

async function pollSetupTask() {
  if (!setupTaskId.value) return
  const task = await editionStore.fetchSetupTask(setupTaskId.value)
  if (task.status === 'pending' || task.status === 'running') {
    setupPollTimer = setTimeout(() => {
      pollSetupTask()
    }, 1200)
    return
  }

  clearSetupPoll()
  await refreshDependencies()
  if (canSelectEdition(setupEdition.value)) {
    selectedEdition.value = setupEdition.value
  }
  if (task.status === 'success') {
    ElMessage.success('依赖安装完成')
    return
  }
  ElMessage.error(task.message || '依赖安装失败')
}

async function handleSetupDependencies() {
  clearSetupPoll()
  try {
    const created = await editionStore.startSetup({
      edition: setupEdition.value,
      runtime: runtime.value,
      apply_config: applyConfig.value,
      pull_embedding_model: pullEmbeddingModel.value
    })
    setupTaskId.value = created.task_id
    ElMessage.success('安装任务已创建')
    await pollSetupTask()
  } catch (err) {
    const message = err instanceof Error ? err.message : '安装任务创建失败'
    ElMessage.error(message)
  }
}

async function copySetupLogs() {
  const logs = editionStore.setupTask?.logs?.trim()
  if (!logs) {
    ElMessage.warning('暂无可复制日志')
    return
  }

  try {
    if (typeof navigator !== 'undefined' && navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(logs)
      ElMessage.success('日志已复制')
      return
    }

    const textarea = document.createElement('textarea')
    textarea.value = logs
    textarea.setAttribute('readonly', 'true')
    textarea.style.position = 'fixed'
    textarea.style.left = '-9999px'
    document.body.appendChild(textarea)
    textarea.select()
    const copied = document.execCommand('copy')
    document.body.removeChild(textarea)

    if (copied) {
      ElMessage.success('日志已复制')
      return
    }

    throw new Error('copy_failed')
  } catch {
    ElMessage.error('复制日志失败，请手动复制')
  }
}

onUnmounted(() => {
  clearSetupPoll()
})
</script>

<style scoped lang="scss">
.edition-selector {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-title {
  font-size: 18px;
  font-weight: 600;
}

.edition-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.edition-card {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 12px;
  cursor: pointer;
}

.edition-card.active {
  border-color: var(--el-color-primary);
}

.edition-card.disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.edition-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.edition-name {
  font-weight: 600;
}

.edition-description {
  color: var(--el-text-color-secondary);
  margin-bottom: 10px;
  min-height: 40px;
}

.edition-dependencies {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.actions {
  display: flex;
  gap: 8px;
}

.setup-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.setup-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.runtime-hint {
  color: var(--el-color-warning-dark-2);
  font-size: 13px;
  line-height: 1.5;
}

.setup-summary {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.setup-log-panel {
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  padding: 10px;
  background: var(--el-fill-color-lighter);
}

.setup-log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 13px;
  color: var(--el-text-color-primary);
}

.setup-log-content {
  margin: 0;
  max-height: 280px;
  overflow-y: auto;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: 6px;
  padding: 10px;
}

@media (max-width: 960px) {
  .edition-cards {
    grid-template-columns: 1fr;
  }
}
</style>
