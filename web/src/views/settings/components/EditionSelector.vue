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
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'

import { useEditionStore } from '@/store/domain/edition'
import type { EditionType } from '@/api/edition-domain'

const editionStore = useEditionStore()
const selectedEdition = ref<EditionType>('basic')

const definitionsReady = computed(() => editionStore.definitions.length > 0)

onMounted(async () => {
  await Promise.all([
    editionStore.fetchEditionConfig(),
    editionStore.fetchDefinitions(),
    refreshDependencies()
  ])

  selectedEdition.value = editionStore.config?.type ?? 'basic'
})

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

@media (max-width: 960px) {
  .edition-cards {
    grid-template-columns: 1fr;
  }
}
</style>
