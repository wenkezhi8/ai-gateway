<template>
  <div class="routing-page">
    <div class="cache-hero">
      <div class="hero-main">
        <div class="hero-title">路由策略控制台</div>
        <div class="hero-subtitle">统一管理分类器、Ollama、模型评分与向量状态（仅使用现有 API）</div>
      </div>
      <div class="hero-actions">
        <el-button type="primary" @click="ctx.reloadAllPanels">
          <el-icon><Refresh /></el-icon>
          刷新全部
        </el-button>
      </div>
    </div>

    <div class="stats-grid">
      <div v-for="stat in ctx.statsCards" :key="stat.title" class="stat-card">
        <div class="stat-icon" :style="{ background: stat.color + '1A', color: stat.color }">
          <el-icon :size="22"><component :is="stat.icon" /></el-icon>
        </div>
        <div class="stat-body">
          <div class="stat-label">{{ stat.title }}</div>
          <div class="stat-value">{{ stat.value }}</div>
        </div>
      </div>
    </div>

    <div class="panel config-panel">
      <el-tabs v-model="ctx.activeTab" class="routing-tabs">
        <el-tab-pane label="路由策略" name="policy">
          <RoutePolicyTab :ctx="ctx" />
        </el-tab-pane>

        <el-tab-pane label="Ollama" name="ollama">
          <OllamaTab :ctx="ctx" />
        </el-tab-pane>

        <el-tab-pane label="模型管理" name="models">
          <ModelManagementTab :ctx="ctx" />
        </el-tab-pane>

        <el-tab-pane label="向量管理" name="vector">
          <VectorManagementTab :ctx="ctx" />
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { proxyRefs } from 'vue'
import RoutePolicyTab from './components/RoutePolicyTab.vue'
import OllamaTab from './components/OllamaTab.vue'
import ModelManagementTab from './components/ModelManagementTab.vue'
import VectorManagementTab from './components/VectorManagementTab.vue'
import { useRoutingConsole } from './composables/useRoutingConsole'

const ctx = proxyRefs(useRoutingConsole())
</script>

<style scoped lang="scss">
.routing-page {
  .cache-hero {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    padding: 20px 24px;
    border-radius: 16px;
    margin-bottom: 20px;
    background: linear-gradient(135deg, #eff6ff, #f5f3ff);
    border: 1px solid #dbeafe;
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

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 16px;
    margin-bottom: 20px;
  }

  .stat-card {
    display: flex;
    align-items: center;
    gap: 12px;
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 14px;
    padding: 14px;
  }

  .stat-icon {
    width: 42px;
    height: 42px;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .stat-label {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .stat-value {
    font-size: 22px;
    font-weight: 700;
    color: var(--el-text-color-primary);
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

  :deep(.routing-tabs > .el-tabs__header) {
    margin-bottom: 18px;
  }

  @media (max-width: 1200px) {
    .stats-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 768px) {
    .cache-hero {
      flex-direction: column;
      align-items: flex-start;
    }

    .stats-grid {
      grid-template-columns: 1fr;
    }
  }
}
</style>
