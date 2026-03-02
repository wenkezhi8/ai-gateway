<template>
  <div class="ollama-console-page">
    <div class="hero">
      <div>
        <div class="hero-title">Ollama 控制台</div>
        <div class="hero-subtitle">统一管理 Ollama 服务、意图路由与向量配置</div>
      </div>
      <el-button type="primary" @click="ctx.reloadAllPanels">
        <el-icon><Refresh /></el-icon>
        刷新全部
      </el-button>
    </div>

    <div class="panel">
      <el-tabs v-model="activeTab" class="console-tabs">
        <el-tab-pane label="Ollama" name="ollama">
          <OllamaServiceTab :ctx="ctx" />
        </el-tab-pane>

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
import { Refresh } from '@element-plus/icons-vue'

import OllamaServiceTab from './components/OllamaServiceTab.vue'
import IntentRoutingTab from './components/IntentRoutingTab.vue'
import VectorManagementTab from './components/VectorManagementTab.vue'
import { useOllamaConsole } from './composables/useOllamaConsole'

const activeTab = ref('ollama')
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
    margin-top: 4px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
  }

  .panel {
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 16px;
    padding: 16px;
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
