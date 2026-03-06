<template>
  <div class="ollama-console-page">
    <div class="hero">
      <div>
        <div class="hero-title">Ollama 控制台</div>
        <div class="hero-subtitle">
          统一管理 Ollama 服务、意图路由与向量配置
          <el-tooltip
            placement="right"
            content="首版范围说明：当前页面首版仅覆盖服务连通、意图路由与向量管理。"
          >
            <el-icon class="subtitle-info-badge"><InfoFilled /></el-icon>
          </el-tooltip>
        </div>
        <div class="hero-steps" aria-label="新手只需 3 步">
          <span class="hero-steps-label">新手只需 3 步</span>
          <el-tag effect="plain" round>启动 Ollama</el-tag>
          <el-tag effect="plain" round>下载推荐模型</el-tag>
          <el-tag effect="plain" round>预热 / 执行一次测试</el-tag>
        </div>
      </div>
      <el-button type="primary" @click="ctx.reloadAllPanels">
        <el-icon><Refresh /></el-icon>
        刷新全部
      </el-button>
    </div>

    <div class="running-overview">
      <div class="running-head">
        <span class="running-title">运行中模型总览</span>
        <el-tag :type="ctx.ollamaSetup.running_models.length > 0 ? 'success' : 'info'" effect="plain">
          {{ ctx.ollamaSetup.running_models.length }} 个模型
        </el-tag>
      </div>
      <div v-if="ctx.ollamaSetup.running_models.length > 0" class="running-list">
        <el-tag v-for="model in ctx.ollamaSetup.running_models" :key="model" type="success" class="running-tag">
          {{ model }}
        </el-tag>
        <span v-if="ctx.ollamaSetup.running_vram_bytes_total > 0" class="running-vram">
          总显存占用 {{ ctx.formatVramBytes(ctx.ollamaSetup.running_vram_bytes_total) }}
        </span>
      </div>
      <el-empty
        v-else
        :image-size="40"
        description="当前无运行模型（请先预热模型：发起一次推理请求或执行 ollama run）"
      />
    </div>

    <div class="panel service-panel">
      <OllamaServiceTab :ctx="ctx" />
    </div>

    <div class="panel tabs-panel">
      <el-tabs v-model="activeTab" class="console-tabs">
        <el-tab-pane label="意图路由" name="intent">
          <IntentRoutingTab :ctx="ctx" />
        </el-tab-pane>

        <el-tab-pane label="向量管理" name="vector">
          <VectorManagementTab :ctx="ctx" />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { proxyRefs, ref } from 'vue'
import { InfoFilled, Refresh } from '@element-plus/icons-vue'

import OllamaServiceTab from './components/OllamaServiceTab.vue'
import IntentRoutingTab from './components/IntentRoutingTab.vue'
import VectorManagementTab from './components/VectorManagementTab.vue'
import { useOllamaConsole } from './composables/useOllamaConsole'

const activeTab = ref('intent')
const ctx = proxyRefs(useOllamaConsole())
</script>

<style scoped lang="scss">
.ollama-console-page {
  .hero {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 20px 24px;
    border-radius: 16px;
    margin-bottom: 20px;
    background: linear-gradient(135deg, #ecfeff, #eff6ff);
    border: 1px solid #bfdbfe;
  }

  .hero-title {
    font-size: 22px;
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .hero-subtitle {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .subtitle-info-badge {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    border: 1px solid var(--el-border-color-lighter);
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: help;
    font-size: 13px;
    transition: all 0.2s ease;

    &:hover {
      color: var(--el-color-info);
      border-color: var(--el-color-info-light-5);
      background: var(--el-color-info-light-9);
    }
  }

  .hero-steps {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    margin-top: 12px;
  }

  .hero-steps-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .panel {
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 16px;
    padding: 16px;
  }

  .service-panel {
    margin-bottom: 14px;
  }

  .running-overview {
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    padding: 12px 14px;
    margin-bottom: 14px;
  }

  .running-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
  }

  .running-title {
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .running-list {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 8px;
  }

  .running-tag {
    margin-right: 0;
  }

  .running-vram {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  :deep(.page-card) {
    border-radius: 12px;
    border: 1px solid var(--el-border-color-lighter);
    margin-bottom: 20px;
  }

  :deep(.card-header) {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-weight: 600;
    gap: 8px;
  }

  :deep(.console-tabs > .el-tabs__header) {
    margin-bottom: 18px;
  }

  @media (max-width: 768px) {
    .hero {
      flex-direction: column;
      align-items: flex-start;
    }
  }
}
</style>
