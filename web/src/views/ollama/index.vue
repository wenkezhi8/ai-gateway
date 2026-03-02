<template>
  <div class="ollama-page">
    <div class="page-header">
      <div>
        <h2>Ollama 管理</h2>
        <p class="subtitle">模型列表、下载/删除、服务启停与运行状态</p>
      </div>
      <el-button :loading="store.loading" @click="store.refreshStatus">
        <el-icon><RefreshRight /></el-icon>
        刷新状态
      </el-button>
    </div>

    <el-alert v-if="store.error" :title="store.error" type="error" :closable="false" style="margin-bottom: 16px" />

    <el-card class="section-card" shadow="never">
      <template #header>
        <div class="card-header">服务状态</div>
      </template>

      <div class="status-row">
        <el-tag :type="statusTagType(store.status?.installed)">安装：{{ store.status?.installed ? '已安装' : '未安装' }}</el-tag>
        <el-tag :type="statusTagType(store.status?.running)">服务：{{ store.status?.running ? '运行中' : '未运行' }}</el-tag>
        <el-tag type="info">当前模型：{{ store.status?.model || '-' }}</el-tag>
      </div>

      <div class="action-row">
        <el-button :loading="store.operating" @click="onInstall">安装 Ollama</el-button>
        <el-button :loading="store.operating" type="warning" @click="onStart">启动服务</el-button>
        <el-button :loading="store.operating" type="danger" @click="onStop">停止服务</el-button>
      </div>
    </el-card>

    <el-card class="section-card" shadow="never">
      <template #header>
        <div class="card-header">模型操作</div>
      </template>

      <div class="action-row">
        <el-input v-model="modelInput" placeholder="输入模型名，如 qwen2.5:0.5b-instruct" />
        <el-button type="primary" :loading="store.operating" @click="onPull">下载模型</el-button>
        <el-button type="danger" :loading="store.operating" @click="onDelete">删除模型</el-button>
      </div>
    </el-card>

    <el-row :gutter="16">
      <el-col :md="12" :sm="24">
        <el-card class="section-card" shadow="never">
          <template #header>
            <div class="card-header">本地模型</div>
          </template>
          <el-empty v-if="store.models.length === 0" description="暂无本地模型" />
          <el-tag v-for="item in store.models" :key="item" class="item-tag">{{ item }}</el-tag>
        </el-card>
      </el-col>

      <el-col :md="12" :sm="24">
        <el-card class="section-card" shadow="never">
          <template #header>
            <div class="card-header">运行中模型</div>
          </template>
          <el-empty v-if="store.runningModels.length === 0" description="当前无运行模型" />
          <el-tag v-for="item in store.runningModels" :key="item" type="success" class="item-tag">{{ item }}</el-tag>

          <el-descriptions v-if="store.runningModelDetails.length > 0" :column="1" border size="small" class="detail-list">
            <el-descriptions-item v-for="item in store.runningModelDetails" :key="item.name" :label="item.name">
              显存占用 {{ formatVram(item.size_vram) }}
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { useOllamaStore } from '@/store/domain/ollama'
import { ROUTING_OLLAMA_DEFAULT_MODEL } from '@/constants/routing'

const store = useOllamaStore()
const modelInput = ref(ROUTING_OLLAMA_DEFAULT_MODEL)

onMounted(async () => {
  store.model = modelInput.value
  await store.refreshStatus()
})

function statusTagType(flag: boolean | undefined) {
  return flag ? 'success' : 'info'
}

function formatVram(bytes: number) {
  if (!bytes || bytes <= 0) return '0 B'
  if (bytes >= 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GiB`
  if (bytes >= 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(2)} MiB`
  if (bytes >= 1024) return `${(bytes / 1024).toFixed(2)} KiB`
  return `${bytes} B`
}

async function onInstall() {
  const result = await store.install()
  if (result.success) ElMessage.success('安装命令已执行')
  else ElMessage.error(result.message || '安装失败')
}

async function onStart() {
  const result = await store.start()
  if (result.success) ElMessage.success('服务已启动')
  else ElMessage.error(result.message || '启动失败')
}

async function onStop() {
  const result = await store.stop()
  if (result.success) ElMessage.success('服务已停止')
  else ElMessage.error(result.message || '停止失败')
}

async function onPull() {
  const model = modelInput.value.trim()
  if (!model) {
    ElMessage.warning('请先输入模型名')
    return
  }
  store.model = model
  const result = await store.pull(model)
  if (result.success) ElMessage.success('模型下载命令已执行')
  else ElMessage.error(result.message || '下载失败')
}

async function onDelete() {
  const model = modelInput.value.trim()
  if (!model) {
    ElMessage.warning('请先输入模型名')
    return
  }
  const result = await store.remove(model)
  if (result.success) ElMessage.success('模型删除成功')
  else ElMessage.error(result.message || '删除失败')
}
</script>

<style scoped lang="scss">
.ollama-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.subtitle {
  margin: 4px 0 0;
  color: var(--el-text-color-secondary);
}

.section-card {
  margin-bottom: 16px;
}

.card-header {
  font-weight: 600;
}

.status-row,
.action-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.action-row {
  margin-top: 10px;
}

.item-tag {
  margin-right: 8px;
  margin-bottom: 8px;
}

.detail-list {
  margin-top: 12px;
}
</style>
