<template>
  <div class="chat-message" :class="[message.role]">
    <div class="message-avatar">
      <el-icon v-if="message.role === 'user'" :size="20">
        <User />
      </el-icon>
      <img v-else-if="providerLogo" :src="providerLogo" class="provider-avatar" />
      <el-icon v-else :size="20">
        <Monitor />
      </el-icon>
    </div>
    <div class="message-content">
      <div class="message-header">
        <span class="role-name">{{ message.role === 'user' ? 'You' : providerName }}</span>
        <span class="timestamp">{{ formatTime(message.timestamp) }}</span>
      </div>
      
      <div class="message-images" v-if="message.images && message.images.length > 0">
        <div 
          v-for="(img, index) in message.images" 
          :key="index" 
          class="message-image"
          @click="previewImage(img)"
        >
          <img :src="img" alt="uploaded image" />
        </div>
      </div>
      
      <div class="message-files" v-if="message.files && message.files.length > 0">
        <div v-for="(file, index) in message.files" :key="index" class="message-file">
          <el-icon><Document /></el-icon>
          <span>{{ file }}</span>
        </div>
      </div>
      
      <div class="message-body" :class="{ error: message.error }">
        <template v-if="message.role === 'assistant'">
          <TypewriterText
            :content="message.content"
            :show-cursor="message.isStreaming"
          />
          <div v-if="message.error" class="error-message">
            <el-icon><WarningFilled /></el-icon>
            {{ message.error }}
          </div>
        </template>
        <template v-else>
          <div class="user-text">{{ message.content }}</div>
        </template>
      </div>
      <div v-if="!message.isStreaming && message.content" class="message-actions">
        <el-button
          text
          size="small"
          @click="copyContent"
        >
          <el-icon><DocumentCopy /></el-icon>
          {{ copied ? t('chat.copied') : t('chat.copy') }}
        </el-button>
      </div>
      <div v-if="message.role === 'assistant' && message.stats && !message.isStreaming" class="message-stats">
        <span v-if="message.stats.firstTokenTime" class="stat-item">
          <span class="stat-label">首token</span>
          <span class="stat-value">{{ message.stats.firstTokenTime.toFixed(2) }}s</span>
        </span>
        <span v-if="message.stats.totalTime" class="stat-item">
          <span class="stat-label">总耗时</span>
          <span class="stat-value">{{ message.stats.totalTime.toFixed(2) }}s</span>
        </span>
        <span v-if="message.stats.outputTokensPerSecond" class="stat-item">
          <span class="stat-label">输出</span>
          <span class="stat-value">{{ message.stats.outputTokensPerSecond }} tokens/s</span>
        </span>
        <span v-if="message.stats.totalTokens" class="stat-item">
          <span class="stat-label">共调用</span>
          <span class="stat-value">{{ message.stats.totalTokens }} tokens</span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { User, Monitor, DocumentCopy, WarningFilled, Document } from '@element-plus/icons-vue'
import type { ChatMessage } from '@/types/chat'
import { getProviderConfig } from '@/store/chat'
import TypewriterText from './TypewriterText.vue'

const props = defineProps<{
  message: ChatMessage
  provider?: string
}>()

const { t } = useI18n()
const copied = ref(false)

const providerName = computed(() => {
  if (props.provider) {
    const config = getProviderConfig(props.provider)
    return config?.label || 'AI'
  }
  return 'AI'
})

const providerLogo = computed(() => {
  if (props.provider) {
    const config = getProviderConfig(props.provider)
    return config?.logo || ''
  }
  return ''
})

function formatTime(timestamp: number): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit'
  })
}

async function copyContent(): Promise<void> {
  try {
    await navigator.clipboard.writeText(props.message.content)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (e) {
    console.error('Failed to copy:', e)
  }
}

function previewImage(src: string): void {
  window.open(src, '_blank')
}
</script>

<style lang="scss" scoped>
.chat-message {
  display: flex;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-lg);
  max-width: 900px;
  margin: 0 auto;

  &.user {
    .message-avatar {
      background: var(--color-primary);
    }

    .message-body {
      background: var(--color-primary);
      color: white;
      border-radius: var(--border-radius-lg) 0 var(--border-radius-lg) var(--border-radius-lg);
    }
  }

  &.assistant {
    .message-avatar {
      background: var(--bg-tertiary);
      color: var(--text-primary);
    }

    .message-body {
      background: var(--bg-tertiary);
      border-radius: 0 var(--border-radius-lg) var(--border-radius-lg) var(--border-radius-lg);
    }
  }
}

.message-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: white;

  .provider-avatar {
    width: 24px;
    height: 24px;
    border-radius: 4px;
    object-fit: contain;
  }
}

.message-content {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  margin-bottom: var(--spacing-xs);
}

.role-name {
  font-weight: 600;
  font-size: var(--font-size-sm);
  color: var(--text-primary);
}

.timestamp {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
}

.message-images {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-sm);
}

.message-image {
  width: 80px;
  height: 80px;
  border-radius: var(--border-radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: transform 0.2s;

  &:hover {
    transform: scale(1.05);
  }

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.message-files {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-sm);
}

.message-file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: rgba(0, 0, 0, 0.1);
  border-radius: var(--border-radius-md);
  font-size: 12px;
  color: var(--text-secondary);

  .el-icon {
    font-size: 14px;
  }
}

.message-body {
  padding: var(--spacing-sm) var(--spacing-md);
  font-size: var(--font-size-base);
  line-height: 1.6;
  word-wrap: break-word;
  overflow-wrap: break-word;

  &.error {
    border: 1px solid var(--color-danger);
    background: rgba(var(--color-danger-rgb, 255, 73, 79), 0.1);
  }
}

.user-text {
  white-space: pre-wrap;
}

.error-message {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  margin-top: var(--spacing-sm);
  color: var(--color-danger);
  font-size: var(--font-size-sm);
}

.message-actions {
  margin-top: var(--spacing-xs);
  opacity: 0;
  transition: opacity var(--transition-fast);

  .chat-message:hover & {
    opacity: 1;
  }
}

.message-stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-sm) var(--spacing-lg);
  margin-top: var(--spacing-sm);
  padding-top: var(--spacing-sm);
  border-top: 1px solid var(--border-color);
  opacity: 0.7;
  font-size: var(--font-size-xs);

  .stat-item {
    display: flex;
    align-items: center;
    gap: var(--spacing-xs);
  }

  .stat-label {
    color: var(--text-tertiary);
  }

  .stat-value {
    color: var(--text-secondary);
    font-weight: 500;
  }
}
</style>
