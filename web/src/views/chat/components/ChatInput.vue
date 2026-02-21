<template>
  <div class="chat-input">
    <div class="input-wrapper">
      <el-input
        ref="inputRef"
        v-model="inputText"
        type="textarea"
        :rows="1"
        :autosize="{ minRows: 1, maxRows: 6 }"
        :placeholder="placeholder"
        :disabled="disabled"
        resize="none"
        @keydown="handleKeydown"
      />
      <div class="actions">
        <span class="hint">{{ t('chat.sendHint') }}</span>
        <el-button
          v-if="!isLoading"
          type="primary"
          :disabled="!inputText.trim() || disabled"
          @click="handleSend"
        >
          <el-icon><Promotion /></el-icon>
          {{ t('chat.send') }}
        </el-button>
        <el-button
          v-else
          type="danger"
          @click="handleStop"
        >
          <el-icon><VideoPause /></el-icon>
          {{ t('chat.stop') }}
        </el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Promotion, VideoPause } from '@element-plus/icons-vue'

const props = defineProps<{
  disabled?: boolean
  isLoading?: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  send: [text: string]
  stop: []
}>()

const { t } = useI18n()
const inputText = ref('')
const inputRef = ref()

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    handleSend()
  }
}

function handleSend(): void {
  const text = inputText.value.trim()
  if (text && !props.disabled && !props.isLoading) {
    emit('send', text)
    inputText.value = ''
    // Reset textarea height
    nextTick(() => {
      if (inputRef.value?.textarea) {
        inputRef.value.textarea.style.height = 'auto'
      }
    })
  }
}

function handleStop(): void {
  emit('stop')
}

function focus(): void {
  inputRef.value?.focus()
}

defineExpose({ focus })
</script>

<style lang="scss" scoped>
.chat-input {
  padding: var(--spacing-xs) var(--spacing-md) var(--spacing-sm);
  background: var(--bg-glass);
  backdrop-filter: blur(20px);
  border-top: 1px solid var(--border-color);
}

.input-wrapper {
  max-width: 900px;
  margin: 0 auto;

  :deep(.el-textarea__inner) {
    padding: var(--spacing-sm) var(--spacing-md);
    padding-bottom: 40px;
    font-size: var(--font-size-base);
    line-height: 1.6;
    border-radius: var(--border-radius-md);
    border: 1px solid var(--border-color);
    background: var(--bg-primary);
    transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
    resize: none;
    overflow: hidden;

    &:focus {
      border-color: var(--color-primary);
      box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.1);
    }

    &:disabled {
      background: var(--bg-tertiary);
      cursor: not-allowed;
    }

    &::placeholder {
      color: var(--text-tertiary);
    }
  }
}

.actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: calc(-1 * var(--spacing-sm) - 32px);
  padding: 0 var(--spacing-sm);
  height: 32px;
  position: relative;
  z-index: 1;
}

.hint {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}
</style>
