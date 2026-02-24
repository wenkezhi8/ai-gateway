<template>
  <div class="model-selector">
    <el-select
      v-model="selectedProvider"
      :placeholder="t('chat.selectProvider')"
      class="provider-select"
      @change="handleProviderChange"
    >
      <template #prefix>
        <img
          v-if="currentProviderLogo"
          :src="currentProviderLogo"
          class="provider-logo-prefix"
        />
        <span
          v-else
          class="provider-dot"
          :style="{ background: currentProviderColor }"
        ></span>
      </template>
      <el-option
        v-for="provider in providers"
        :key="provider.value"
        :label="provider.label"
        :value="provider.value"
      >
        <span class="provider-option">
          <img v-if="provider.logo" :src="provider.logo" class="provider-logo" />
          <span v-else class="dot" :style="{ background: provider.color }"></span>
          {{ provider.label }}
        </span>
      </el-option>
    </el-select>

    <el-select
      v-model="selectedModel"
      :placeholder="t('chat.selectModel')"
      class="model-select"
      :disabled="!availableModels.length"
      @change="handleModelChange"
    >
      <el-option
        v-for="model in availableModels"
        :key="model"
        :label="model"
        :value="model"
      >
        <!-- 改动点: 标注支持推理的模型 -->
        <span class="model-option">
          <span class="model-name">{{ model }}</span>
          <span v-if="isReasoningModel(model)" class="model-badge">支持推理</span>
        </span>
      </el-option>
    </el-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PROVIDERS } from '@/store/chat'

const props = defineProps<{
  provider?: string
  model?: string
}>()

const emit = defineEmits<{
  'update:provider': [value: string]
  'update:model': [value: string]
  change: [provider: string, model: string]
}>()

const { t } = useI18n()
const providers = PROVIDERS

const selectedProvider = ref(props.provider || '')
const selectedModel = ref(props.model || '')

const availableModels = computed(() => {
  const found = PROVIDERS.value.find(p => p.value === selectedProvider.value)
  return found?.models || []
})

function selectFirstAvailable(): void {
  if (!selectedProvider.value && PROVIDERS.value.length > 0) {
    const firstProvider = PROVIDERS.value[0]
    if (firstProvider) {
      selectedProvider.value = firstProvider.value
      if (firstProvider.models && firstProvider.models.length > 0) {
        selectedModel.value = firstProvider.models[0] ?? ''
      }
    }
  }
}

selectFirstAvailable()

const currentProviderColor = computed(() => {
  const config = PROVIDERS.value.find(p => p.value === selectedProvider.value)
  return config?.color || '#666'
})

const currentProviderLogo = computed(() => {
  const config = PROVIDERS.value.find(p => p.value === selectedProvider.value)
  return config?.logo || ''
})

function handleProviderChange(provider: string): void {
  selectedProvider.value = provider
  const found = PROVIDERS.value.find(p => p.value === provider)
  const models = found?.models || []
  if (models.length > 0 && !models.includes(selectedModel.value)) {
    selectedModel.value = models[0] || ''
  }
  emitChange()
}

function handleModelChange(model: string): void {
  selectedModel.value = model
  emitChange()
}

function emitChange(): void {
  emit('update:provider', selectedProvider.value)
  emit('update:model', selectedModel.value)
  emit('change', selectedProvider.value, selectedModel.value)
}

function isReasoningModel(model: string): boolean {
  return model.includes('reasoner') || model.includes('r1')
}

// Watch for external prop changes
watch(() => props.provider, (val) => {
  if (val && val !== selectedProvider.value) {
    selectedProvider.value = val
  }
})

watch(() => props.model, (val) => {
  if (val && val !== selectedModel.value) {
    selectedModel.value = val
  }
})

watch(() => PROVIDERS.value, (newProviders) => {
  if (newProviders && newProviders.length > 0 && !selectedProvider.value) {
    selectFirstAvailable()
  }
}, { immediate: true })
</script>

<style lang="scss" scoped>
.model-selector {
  display: flex;
  gap: var(--spacing-sm);
  padding: var(--spacing-md) var(--spacing-lg);
  max-width: 900px;
  margin: 0 auto;
}

.provider-select,
.model-select {
  :deep(.el-input__wrapper) {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
    box-shadow: none;
    transition: border-color var(--transition-fast);

    &:hover {
      border-color: var(--color-primary);
    }
  }

  :deep(.el-input__inner) {
    color: var(--text-primary);
  }
}

.provider-select {
  width: 160px;
}

.model-select {
  flex: 1;
  max-width: 240px;
}

.provider-dot {
  width: 8px;
  height: 8px;
  border-radius: 2px;
  display: inline-block;
}

.provider-logo-prefix {
  height: 18px;
  width: auto;
  max-width: 60px;
  border-radius: 4px;
  object-fit: contain;
}

.provider-logo {
  height: 20px;
  width: auto;
  max-width: 70px;
  border-radius: 4px;
  object-fit: contain;
  flex-shrink: 0;
}

.provider-option {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;

  .dot {
    width: 10px;
    height: 10px;
    border-radius: 3px;
    flex-shrink: 0;
  }
}

.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
}

.model-name {
  flex: 1;
  min-width: 0;
}

.model-badge {
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  color: #1d4ed8;
  background: rgba(29, 78, 216, 0.12);
}
</style>
