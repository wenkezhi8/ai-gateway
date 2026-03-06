<template>
  <TabStateView
    :state="ctx.panelState.ollama"
    :error-text="ctx.panelError.ollama"
    empty-text="暂无 Ollama 状态数据"
    @retry="ctx.reloadOllamaPanel"
  >
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="card-header">
          <span>Ollama 服务管理</span>
          <el-button link :loading="ctx.ollamaRefreshing" @click="ctx.reloadOllamaPanel">刷新状态</el-button>
        </div>
      </template>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        class="novice-alert"
        title="系统已使用推荐运行参数，非必要无需调整"
      />

      <section class="service-section">
        <div class="section-title">基础设置</div>
        <div class="status-tags">
          <el-tag :type="ctx.ollamaSetup.installed ? 'success' : 'warning'" effect="plain">
            Ollama安装: {{ ctx.ollamaSetup.installed ? '已安装' : '未安装' }}
          </el-tag>
          <el-tag :type="ctx.ollamaSetup.running ? 'success' : 'danger'" effect="plain">
            服务状态: {{ ctx.ollamaSetup.running ? '运行中' : '未运行' }}
          </el-tag>
          <el-tag type="info" effect="plain">
            当前运行模型: {{ ctx.ollamaSetup.running_model || '无' }}
          </el-tag>
        </div>

        <div class="preset-row">
          <span class="preset-title">常用模型</span>
          <el-button
            v-for="item in commonModels"
            :key="item"
            text
            type="primary"
            :disabled="ctx.ollamaPulling || ctx.ollamaDeleting"
            @click="ctx.ollamaModelInput = item"
          >
            {{ item }}
          </el-button>
        </div>

        <el-row :gutter="12" class="action-block">
          <el-col :span="12">
            <el-input v-model="ctx.ollamaModelInput" placeholder="模型名，如 qwen2.5:0.5b-instruct" />
          </el-col>
          <el-col :span="12" class="action-row">
            <el-button :loading="ctx.ollamaInstalling" @click="ctx.installOllama">安装 Ollama</el-button>
            <el-button :loading="ctx.ollamaStarting" type="warning" @click="ctx.startOllama">启动服务</el-button>
            <el-button :loading="ctx.ollamaStopping" type="danger" @click="ctx.stopOllama">停止服务</el-button>
            <el-button :loading="ctx.ollamaPulling" type="primary" @click="ctx.pullOllamaModel">下载模型</el-button>
            <el-button :loading="ctx.ollamaDeleting" type="danger" plain @click="ctx.deleteOllamaModel">删除模型</el-button>
            <el-button type="primary" plain :loading="ctx.ollamaPreloading" @click="ctx.preloadConfiguredOllamaModels">
              立即预热
            </el-button>
          </el-col>
        </el-row>
      </section>

      <section class="service-section">
        <div class="section-title">运行状态</div>

        <el-alert
          v-if="ctx.ollamaSetup.message"
          :title="`Ollama状态: ${ctx.ollamaSetup.message}`"
          :type="ctx.ollamaSetup.running ? 'success' : 'warning'"
          :closable="false"
          class="state-alert"
        />

        <el-alert
          v-if="ctx.ollamaSetup.last_error"
          :title="`启动错误: ${ctx.ollamaSetup.last_error}`"
          type="error"
          :closable="false"
          class="state-alert"
        />

        <el-row :gutter="16">
          <el-col :span="12">
            <el-card shadow="never" class="inner-card">
              <template #header>
                <div class="card-header">本地模型</div>
              </template>
              <el-empty v-if="ctx.ollamaSetup.models.length === 0" description="暂无本地模型" />
              <el-tag v-for="item in ctx.ollamaSetup.models" :key="item" class="item-tag">{{ item }}</el-tag>
            </el-card>
          </el-col>
          <el-col :span="12">
            <el-card shadow="never" class="inner-card">
              <template #header>
                <div class="card-header">运行中模型</div>
              </template>
              <el-empty v-if="ctx.ollamaSetup.running_models.length === 0" description="当前无运行模型（请先预热模型：发起一次推理请求或执行 ollama run）" />
              <el-tag v-for="item in ctx.ollamaSetup.running_models" :key="item" type="success" class="item-tag">{{ item }}</el-tag>

              <el-descriptions v-if="ctx.ollamaSetup.running_model_details.length > 0" :column="1" border size="small" style="margin-top: 12px">
                <el-descriptions-item v-for="item in ctx.ollamaSetup.running_model_details" :key="item.name" :label="item.name">
                  显存占用 {{ ctx.formatVramBytes(item.size_vram) }}
                </el-descriptions-item>
              </el-descriptions>
            </el-card>
          </el-col>
        </el-row>

        <el-descriptions v-if="ctx.ollamaPreloadResults.length > 0" border :column="1" size="small" class="preload-results">
          <el-descriptions-item v-for="item in ctx.ollamaPreloadResults" :key="`${item.kind}-${item.model}`" :label="`预热结果 · ${item.kind}`">
            <el-tag :type="item.status === 'success' ? 'success' : 'danger'" style="margin-right: 8px">
              {{ item.status === 'success' ? '成功' : '失败' }}
            </el-tag>
            <span style="margin-right: 8px">{{ item.model }}</span>
            <span style="margin-right: 8px">耗时 {{ item.duration_ms }} ms</span>
            <span v-if="item.error">{{ item.error }}</span>
          </el-descriptions-item>
        </el-descriptions>
      </section>

      <section class="service-section advanced-section">
        <div class="section-title">高级设置</div>
        <el-collapse>
          <el-collapse-item name="advanced" title="高级设置">
            <div class="poll-row">
              <el-switch v-model="pollEnabled" active-text="自动轮询" inactive-text="手动刷新" />
              <el-select v-model="pollIntervalSeconds" style="width: 180px" :disabled="!pollEnabled">
                <el-option v-for="option in pollIntervalOptions" :key="option" :label="`${option} 秒`" :value="option" />
              </el-select>
              <span class="poll-label">轮询间隔</span>
            </div>

            <el-row :gutter="12" class="config-row">
              <el-col :span="10">
                <el-form-item label="启动方式" class="runtime-item">
                  <el-select v-model="ctx.ollamaRuntimeConfig.startup_mode" style="width: 100%">
                    <el-option label="自动检测（优先 App）" value="auto" />
                    <el-option label="使用 Ollama.app" value="app" />
                    <el-option label="使用 CLI" value="cli" />
                    <el-option label="手动启动" value="manual" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="10">
                <div class="monitor-switch-row">
                  <el-switch v-model="ctx.ollamaRuntimeConfig.monitoring.enabled" active-text="启用监控" inactive-text="关闭监控" />
                  <el-switch v-model="ctx.ollamaRuntimeConfig.monitoring.auto_restart" active-text="自动重启" inactive-text="不自动重启" />
                </div>
              </el-col>
              <el-col :span="4" class="config-action-col">
                <el-button type="primary" plain :loading="ctx.ollamaConfigSaving" @click="ctx.saveOllamaRuntimeConfig">
                  保存配置
                </el-button>
              </el-col>
            </el-row>

            <el-card shadow="never" class="inner-card preload-card">
              <template #header>
                <div class="card-header">模型预热</div>
              </template>
              <el-row :gutter="12" class="preload-row">
                <el-col :span="8">
                  <el-switch
                    v-model="ctx.ollamaRuntimeConfig.preload.auto_on_startup"
                    active-text="启动时自动预热"
                    inactive-text="仅手动预热"
                  />
                </el-col>
                <el-col :span="10">
                  <el-checkbox-group v-model="ctx.ollamaRuntimeConfig.preload.targets">
                    <el-checkbox label="intent">意图模型</el-checkbox>
                    <el-checkbox label="embedding">Embedding 模型</el-checkbox>
                  </el-checkbox-group>
                </el-col>
                <el-col :span="6" class="preload-timeout-col">
                  <el-input-number
                    v-model="ctx.ollamaRuntimeConfig.preload.timeout_seconds"
                    :min="30"
                    :max="600"
                    :step="30"
                    controls-position="right"
                  />
                  <span class="preload-timeout-label">单模型超时(秒)</span>
                </el-col>
              </el-row>
            </el-card>

            <el-descriptions border :column="3" size="small" style="margin-bottom: 12px">
              <el-descriptions-item label="监控状态">
                <el-tag :type="ctx.ollamaSetup.monitoring_stats.enabled ? 'success' : 'info'">
                  {{ ctx.ollamaSetup.monitoring_stats.enabled ? '已启用' : '已关闭' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="健康状态">
                <el-tag :type="ctx.ollamaSetup.monitoring_stats.health_status === 'healthy' ? 'success' : 'warning'">
                  {{ ctx.ollamaSetup.monitoring_stats.health_status || 'unknown' }}
                </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="自动重启次数">
                {{ ctx.ollamaSetup.monitoring_stats.restart_attempts || 0 }}
              </el-descriptions-item>
            </el-descriptions>
          </el-collapse-item>
        </el-collapse>
      </section>
    </el-card>
  </TabStateView>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'

import TabStateView from './TabStateView.vue'

const props = defineProps<{
  ctx: any
}>()

const commonModels = computed(() => {
  const models = Array.isArray(props.ctx.ollamaSetup?.models) ? props.ctx.ollamaSetup.models : []
  const current = String(props.ctx.ollamaSetup?.model || '').trim()
  const merged = Array.from(new Set([...models, current].filter(Boolean)))
  return merged.slice(0, 6)
})
const pollIntervalOptions = [5, 10, 15, 30, 60]
const pollEnabled = ref(true)
const pollIntervalSeconds = ref(10)

let pollTimer: number | null = null

function stopPolling() {
  if (pollTimer !== null) {
    window.clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling() {
  stopPolling()
  if (!pollEnabled.value) return
  pollTimer = window.setInterval(() => {
    void props.ctx.loadOllamaSetupStatus()
  }, pollIntervalSeconds.value * 1000)
}

onMounted(() => {
  startPolling()
})

onUnmounted(() => {
  stopPolling()
})

watch([pollEnabled, pollIntervalSeconds], () => {
  startPolling()
})
</script>

<style scoped lang="scss">
.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.action-block {
  margin-top: 12px;
}

.novice-alert,
.state-alert,
.preload-results {
  margin-bottom: 12px;
}

.service-section {
  margin-bottom: 18px;
}

.section-title {
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.status-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}

.poll-row {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.config-row {
  margin-bottom: 12px;
}

.runtime-item {
  margin-bottom: 0;
}

.monitor-switch-row {
  min-height: 32px;
  display: flex;
  align-items: center;
  gap: 12px;
}

.config-action-col {
  display: flex;
  justify-content: flex-end;
}

.preload-card {
  margin-bottom: 12px;
}

.preload-row {
  margin-bottom: 10px;
}

.preload-timeout-col {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.preload-timeout-label,
.poll-label,
.preset-title {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.preset-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 8px;
}

.inner-card {
  border: 1px solid var(--el-border-color-lighter);
}

.item-tag {
  margin-right: 8px;
  margin-bottom: 8px;
}

.advanced-section {
  margin-bottom: 0;
}
</style>
