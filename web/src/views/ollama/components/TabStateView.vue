<template>
  <div v-if="state === 'loading'" class="state-wrap">
    <el-skeleton :rows="5" animated />
  </div>
  <el-result
    v-else-if="state === 'error'"
    icon="error"
    title="加载失败"
    :sub-title="errorText || '请稍后重试'"
    class="state-wrap"
  >
    <template #extra>
      <el-button type="primary" @click="$emit('retry')">重试</el-button>
    </template>
  </el-result>
  <el-empty
    v-else-if="state === 'empty'"
    :description="emptyText || '暂无数据'"
    class="state-wrap"
  />
  <slot v-else />
</template>

<script setup lang="ts">
type TabPanelState = 'loading' | 'error' | 'empty' | 'success'

defineProps<{
  state: TabPanelState
  errorText?: string
  emptyText?: string
}>()

defineEmits<{
  (e: 'retry'): void
}>()
</script>

<style scoped lang="scss">
.state-wrap {
  background: var(--el-fill-color-blank);
  border-radius: 12px;
  padding: 24px;
}
</style>
