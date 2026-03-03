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
      
      <div class="message-reasoning" v-if="message.reasoningContent || message.reasoning">
        <div class="reasoning-header" @click="toggleReasoning">
          <el-icon><Cpu /></el-icon>
          <span>深度思考过程</span>
          <span class="reasoning-badge">推理</span>
          <!-- 改动点: 收起时显示摘要，悬浮展示完整推理 -->
          <el-tooltip v-if="!showReasoning && reasoningSummary" :content="reasoningFull" placement="top">
            <span class="reasoning-summary">
              {{ reasoningSummary }}
            </span>
          </el-tooltip>
          <el-icon class="toggle-icon" :class="{ expanded: showReasoning }"><ArrowDown /></el-icon>
        </div>
        <div class="reasoning-content" v-show="showReasoning">
          <TypewriterText
            :content="message.reasoningContent || message.reasoning || ''"
            :show-cursor="message.isStreaming && !message.content"
          />
        </div>
      </div>
      
      <div class="message-body" :class="{
        error: message.error,
        // 改动点: 受控的答案区层级样式开关
        'answer-highlight': message.role === 'assistant' && answerHighlight
      }">
        <template v-if="message.role === 'assistant'">
          <div v-if="message.reasoningContent || message.reasoning" class="answer-label">
            最终答案
          </div>
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
        <span class="stat-item">
          <span class="stat-label">本地缓存</span>
          <span class="stat-value" :class="cacheHitClass">{{ cacheHitText }}</span>
        </span>
        
        <!-- 调试详情折叠区 -->
        <div class="debug-details-toggle" @click="toggleDebugDetails">
          <el-icon class="toggle-icon" :class="{ expanded: showDebugDetails }"><ArrowRight /></el-icon>
          <span class="toggle-label">调试详情</span>
        </div>
        <div v-if="showDebugDetails" class="debug-details">
          <div class="debug-item">
            <span class="debug-label">请求模式</span>
            <span class="debug-value">{{ requestModeText }}</span>
          </div>
          <div v-if="message.stats.speedBasis" class="debug-item">
            <span class="debug-label">速度口径</span>
            <span class="debug-value">{{ speedBasisText }}</span>
          </div>
          <div v-if="message.stats.cacheLayer" class="debug-item">
            <span class="debug-label">缓存层</span>
            <span class="debug-value">{{ cacheLayerText }}</span>
          </div>
          <div class="debug-item">
            <span class="debug-label">Tokens</span>
            <span class="debug-value">
              prompt: {{ message.stats.promptTokens ?? '-' }} / 
              completion: {{ message.stats.completionTokens ?? '-' }} / 
              total: {{ message.stats.totalTokens ?? '-' }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { User, Monitor, DocumentCopy, WarningFilled, Document, Cpu, ArrowDown, ArrowRight } from '@element-plus/icons-vue'
import type { ChatMessage } from '@/types/chat'
import { getProviderConfig } from '@/store/chat'
import TypewriterText from './TypewriterText.vue'

const props = defineProps<{
  message: ChatMessage
  provider?: string
  defaultExpandReasoning?: boolean // 改动点: 推理默认展开
  answerHighlight?: boolean // 改动点: 答案区背景开关
}>()

const { t } = useI18n()
const copied = ref(false)
const showReasoning = ref(props.defaultExpandReasoning ?? true)
const showDebugDetails = ref(false) // 调试详情默认折叠

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

const answerHighlight = computed(() => props.answerHighlight ?? true) // 改动点: 默认启用答案层级样式

const cacheHitText = computed(() => {
  const cacheHit = props.message.stats?.cacheHit
  if (cacheHit === true) return '命中'
  if (cacheHit === false) return '未命中'
  return '未知'
})

const cacheHitClass = computed(() => {
  const cacheHit = props.message.stats?.cacheHit
  if (cacheHit === true) return 'cache-hit'
  if (cacheHit === false) return 'cache-miss'
  return 'cache-unknown'
})

const requestModeText = computed(() => {
  const mode = props.message.stats?.requestMode
  if (mode === 'stream') return '流式'
  if (mode === 'non_stream') return '非流式'
  return '未知'
})

const speedBasisText = computed(() => {
  const basis = props.message.stats?.speedBasis
  if (basis === 'post_first_token') return '首token后'
  if (basis === 'total_time') return '总时长'
  if (basis === 'fallback_total_time') return '回退总时长'
  return '未知'
})

const cacheLayerText = computed(() => {
  const layer = props.message.stats?.cacheLayer
  if (!layer) return '未知'
  return layer
})

const reasoningFull = computed(() => (props.message.reasoningContent || props.message.reasoning || '').trim())

const reasoningSummary = computed(() => {
  const trimmed = reasoningFull.value
  if (!trimmed) return ''
  if (trimmed.length <= 20) return trimmed
  return `${trimmed.slice(0, 20)}...`
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

function toggleReasoning(): void {
  showReasoning.value = !showReasoning.value
}

function toggleDebugDetails(): void {
  showDebugDetails.value = !showDebugDetails.value
}

watch(() => props.defaultExpandReasoning, (val) => {
  if (typeof val === 'boolean') {
    showReasoning.value = val
  }
})
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

.message-reasoning {
  margin-bottom: var(--spacing-sm);
  border: 1px dashed var(--border-secondary); /* 改动点: 强化推理区与答案区分 */
  border-radius: var(--border-radius-md);
  overflow: hidden;
}

.reasoning-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: linear-gradient(135deg, rgba(103, 194, 58, 0.1), rgba(64, 158, 255, 0.1));
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-secondary);
  transition: all 0.2s;

  &:hover {
    background: linear-gradient(135deg, rgba(103, 194, 58, 0.2), rgba(64, 158, 255, 0.2));
  }

  .el-icon {
    font-size: 14px;
  }

  .toggle-icon {
    margin-left: auto;
    transition: transform 0.2s;

    &.expanded {
      transform: rotate(180deg);
    }
  }
}

.reasoning-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  color: #2d6a4f;
  background: rgba(45, 106, 79, 0.12);
}

.reasoning-summary {
  margin-left: 4px;
  padding: 2px 6px;
  border-radius: 6px;
  font-size: 11px;
  color: var(--text-tertiary);
  background: rgba(0, 0, 0, 0.04);
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.reasoning-content {
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-secondary);
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
  border-top: 1px solid var(--border-secondary);
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

.message-body.answer-highlight {
  /* 改动点: 答案区淡色背景/边框/阴影 */
  background: color-mix(in srgb, var(--bg-tertiary) 92%, var(--color-primary) 8%);
  border: 1px solid color-mix(in srgb, var(--border-color) 60%, transparent);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.answer-label {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  margin-bottom: var(--spacing-xs);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary);
  background: color-mix(in srgb, var(--color-primary) 14%, transparent);
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

    &.cache-hit {
      color: var(--color-success);
    }

    &.cache-miss {
      color: var(--color-warning);
    }

    &.cache-unknown {
      color: var(--text-tertiary);
    }
  }
}

.debug-details-toggle {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  margin-left: auto;
  padding: 2px 6px;
  border-radius: 4px;
  transition: background 0.2s;

  &:hover {
    background: rgba(0, 0, 0, 0.04);
  }

  .toggle-icon {
    font-size: 12px;
    transition: transform 0.2s;

    &.expanded {
      transform: rotate(90deg);
    }
  }

  .toggle-label {
    font-size: 11px;
    color: var(--text-tertiary);
  }
}

.debug-details {
  width: 100%;
  margin-top: var(--spacing-xs);
  padding: var(--spacing-xs) var(--spacing-sm);
  background: rgba(0, 0, 0, 0.02);
  border-radius: var(--border-radius-sm);
  border: 1px solid var(--border-secondary);
}

.debug-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: 2px 0;
  font-size: 11px;
}

.debug-label {
  color: var(--text-tertiary);
  min-width: 60px;
}

.debug-value {
  color: var(--text-secondary);
  font-family: monospace;
}
</style>
