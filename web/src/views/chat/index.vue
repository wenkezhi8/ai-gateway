<template>
  <div class="chat-page" :class="{ 'public-page': isPublicPage }">
    <!-- Sidebar: Conversation List -->
    <aside class="sidebar" :class="{ collapsed: sidebarCollapsed }">
      <div class="sidebar-header">
        <el-button type="primary" class="new-chat-btn" @click="createNewChat">
          <el-icon><Plus /></el-icon>
          {{ t('chat.newChat') }}
        </el-button>
        <div v-if="conversations.length > 0" class="batch-actions">
          <el-checkbox v-model="selectAll" @change="handleSelectAll">全选</el-checkbox>
          <el-button 
            v-if="selectedConversationIds.size > 0" 
            type="success" 
            size="small"
            @click="showBatchDialog"
          >
            批量提问 ({{ selectedConversationIds.size }})
          </el-button>
        </div>
      </div>
      <div class="conversation-list">
        <div
          v-for="conversation in conversations"
          :key="conversation.id"
          class="conversation-item"
          :class="{ active: conversation.id === currentConversationId }"
          @click="switchChat(conversation.id)"
        >
          <el-checkbox 
            :model-value="selectedConversationIds.has(conversation.id)"
            @change="(val: boolean) => toggleConversationSelection(conversation.id, val)"
            @click.stop
            class="conversation-checkbox"
          />
          <img v-if="getProviderLogo(conversation.provider)" :src="getProviderLogo(conversation.provider)" class="provider-logo-sidebar" />
          <el-icon v-else class="icon"><ChatDotRound /></el-icon>
          <div class="conversation-content">
            <span class="title">{{ conversation.title || t('chat.newConversation') }}</span>
            <span class="model-badge">
              {{ getModelDisplayName(conversation.model) }}
            </span>
          </div>
          <el-button
            text
            class="delete-btn"
            @click.stop="confirmDelete(conversation.id)"
          >
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
      </div>
    </aside>

    <!-- Main Content -->
    <main class="main-content">
      <!-- Mobile sidebar toggle -->
      <button class="mobile-menu-btn" @click="toggleSidebar">
        <el-icon><Operation /></el-icon>
      </button>

      <!-- Welcome Screen -->
      <div v-if="!currentConversation" class="welcome-screen">
        <div class="welcome-content">
          <h1>{{ t('chat.welcome') }}</h1>
          <p>{{ t('chat.welcomeDesc') }}</p>
          <el-button type="primary" size="large" @click="createNewChat">
            <el-icon><Plus /></el-icon>
            {{ t('chat.newChat') }}
          </el-button>
        </div>
      </div>

      <!-- Chat Area -->
      <template v-else>
        <!-- Messages -->
        <div ref="messagesContainer" class="messages-area">
          <ChatMessage
            v-for="message in currentMessages"
            :key="message.id"
            :message="message"
            :provider="currentConversation?.provider"
            :default-expand-reasoning="defaultExpandReasoning"
            :answer-highlight="answerHighlightEnabled"
          />
          <div v-if="isLoading && !isStreaming" class="loading-indicator">
            <el-icon class="is-loading"><Loading /></el-icon>
          </div>
        </div>

        <!-- Model Selector & Input -->
        <div class="input-area">
          <div class="chat-settings">
            <el-tooltip content="默认展开/收起深度思考过程" placement="top">
              <div class="setting-item">
                <span>推理默认展开</span>
                <el-switch v-model="defaultExpandReasoning" />
              </div>
            </el-tooltip>
            <el-tooltip content="答案区域淡色背景" placement="top">
              <div class="setting-item">
                <span>答案背景</span>
                <el-switch v-model="answerHighlightEnabled" />
              </div>
            </el-tooltip>
            <el-tooltip content="关闭后使用非流式响应（便于测试缓存）" placement="top">
              <div class="setting-item">
                <span>流式响应</span>
                <el-switch v-model="streamEnabled" />
              </div>
            </el-tooltip>
          </div>
          <ModelSelector
            v-model:provider="selectedProvider"
            v-model:model="selectedModel"
            @change="handleModelChange"
          />
          <ChatInput
            ref="chatInputRef"
            :is-loading="isStreaming"
            @send="handleSend"
            @stop="handleStop"
            @update:web-search-enabled="webSearchEnabled = $event"
            @update:deep-think-enabled="deepThinkEnabled = $event"
            @update:multimodal-enabled="multimodalEnabled = $event"
          />
        </div>
      </template>
    </main>

    <!-- Batch Send Dialog -->
    <el-dialog v-model="batchDialogVisible" title="批量提问" width="600px">
      <div class="batch-dialog-content">
        <div class="selected-conversations">
          <div class="label">将向以下 {{ selectedConversationIds.size }} 个会话发送相同问题：</div>
          <div class="conversation-tags">
            <el-tag 
              v-for="conv in selectedConversationsList" 
              :key="conv.id"
              class="conv-tag"
            >
              <img v-if="getProviderLogo(conv.provider)" :src="getProviderLogo(conv.provider)" class="tag-logo" />
              {{ conv.title || '新对话' }} ({{ getModelDisplayName(conv.model) }})
            </el-tag>
          </div>
        </div>
        <el-input
          v-model="batchQuestion"
          type="textarea"
          :rows="4"
          placeholder="请输入要批量发送的问题..."
          class="batch-input"
        />
      </div>
      <template #footer>
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleBatchSend" :loading="batchSending">
          发送 ({{ selectedConversationIds.size }})
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessageBox, ElMessage } from 'element-plus'
import { Plus, ChatDotRound, Delete, Operation, Loading } from '@element-plus/icons-vue'
import { useRoute } from 'vue-router'
import { useChatStore, initializeProviders, PROVIDERS } from '@/store/chat'
import { streamCompletion, search } from '@/api/chat'
import { createMessage, type ChatMessage as ChatMessageType } from '@/types/chat'
import ChatMessage from './components/ChatMessage.vue'
import ChatInput from './components/ChatInput.vue'
import ModelSelector from './components/ModelSelector.vue'

const route = useRoute()
const { t } = useI18n()
const chatStore = useChatStore()

const isPublicPage = computed(() => route.meta.public === true)

// Refs
const messagesContainer = ref<HTMLElement>()
const chatInputRef = ref()
const sidebarCollapsed = ref(window.innerWidth < 768)
const selectedProvider = ref('')
const selectedModel = ref('')

// Store state
const conversations = computed(() => chatStore.conversations)
const currentConversationId = computed(() => chatStore.currentConversationId)
const currentConversation = computed(() => chatStore.currentConversation)
const currentMessages = computed(() => chatStore.currentMessages)
const isLoading = computed(() => chatStore.isLoading)
const isStreaming = computed(() => {
  if (!currentConversationId.value) return false
  return chatStore.isConversationStreaming(currentConversationId.value)
})

// Batch selection state
const selectedConversationIds = ref<Set<string>>(new Set())
const selectAll = ref(false)
const batchDialogVisible = ref(false)
const batchQuestion = ref('')
const batchSending = ref(false)

// Feature toggles state
const webSearchEnabled = ref(false)
const deepThinkEnabled = ref(false)
const multimodalEnabled = ref(false)
const defaultExpandReasoning = ref(true)
const answerHighlightEnabled = ref(true)
const streamEnabled = ref(true)  // 流式响应开关

const selectedConversationsList = computed(() => {
  return conversations.value.filter(c => selectedConversationIds.value.has(c.id))
})

function handleSelectAll(val: boolean): void {
  if (val) {
    selectedConversationIds.value = new Set(conversations.value.map(c => c.id))
  } else {
    selectedConversationIds.value = new Set()
  }
}

function toggleConversationSelection(conversationId: string, selected: boolean): void {
  const newSet = new Set(selectedConversationIds.value)
  if (selected) {
    newSet.add(conversationId)
  } else {
    newSet.delete(conversationId)
  }
  selectedConversationIds.value = newSet
  selectAll.value = newSet.size === conversations.value.length
}

function showBatchDialog(): void {
  if (selectedConversationIds.value.size === 0) {
    ElMessage.warning('请先选择要发送的会话')
    return
  }
  batchQuestion.value = ''
  batchDialogVisible.value = true
}

async function handleBatchSend(): Promise<void> {
  const question = batchQuestion.value.trim()
  if (!question) {
    ElMessage.warning('请输入问题')
    return
  }

  batchSending.value = true
  const convIds = Array.from(selectedConversationIds.value)
  
  // 并行发送所有请求
  const promises = convIds.map(async (convId) => {
    try {
      const conv = conversations.value.find(c => c.id === convId)
      if (!conv) return { success: false, convId }

      const userMessage = createMessage('user', question)
      chatStore.addMessageToConversation(convId, userMessage)

      const assistantMessage: ChatMessageType = {
        id: `assistant-${Date.now()}-${convId}`,
        role: 'assistant',
        content: '',
        timestamp: Date.now(),
        isStreaming: true
      }
      chatStore.addMessageToConversation(convId, assistantMessage)

      const messages = chatStore.getConversationMessages(convId)
        .filter(m => m.id !== assistantMessage.id && m.content)
        .map(m => ({
          role: m.role,
          content: m.content
        }))

      return new Promise<{ success: boolean; convId: string }>((resolve) => {
        const controller = streamCompletion(
          {
            model: conv.model,
            provider: conv.provider,
            messages,
            stream: streamEnabled.value
          },
          (chunk) => {
            const content = chunk.choices?.[0]?.delta?.content
            if (content) {
              chatStore.appendMessageContent(assistantMessage.id, content, convId)
            }
          },
          (error) => {
            chatStore.updateMessage(assistantMessage.id, { 
              isStreaming: false, 
              error: error.message 
            }, convId)
            chatStore.setAbortController(convId, null)
            resolve({ success: false, convId })
          },
          () => {
            chatStore.updateMessage(assistantMessage.id, { isStreaming: false }, convId)
            chatStore.setAbortController(convId, null)
            resolve({ success: true, convId })
          }
        )
        chatStore.setAbortController(convId, controller)
      })
    } catch (e) {
      return { success: false, convId }
    }
  })

  // 等待所有请求完成
  const results = await Promise.all(promises)
  
  const successCount = results.filter(r => r?.success).length
  const failCount = results.length - successCount

  batchSending.value = false
  batchDialogVisible.value = false
  selectedConversationIds.value = new Set()
  selectAll.value = false
  
  if (successCount > 0) {
    ElMessage.success(`成功发送 ${successCount} 个会话${failCount > 0 ? `，失败 ${failCount} 个` : ''}`)
  } else {
    ElMessage.error('发送失败')
  }
}

// Methods
function toggleSidebar(): void {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

const providerLogos: Record<string, string> = {
  openai: '/logos/openai.svg',
  deepseek: '/logos/deepseek.svg',
  anthropic: '/logos/anthropic.svg',
  qwen: '/logos/qwen.svg',
  zhipu: '/logos/zhipu.svg',
  moonshot: '/logos/moonshot.svg',
  minimax: '/logos/minimax.svg',
  baichuan: '/logos/baichuan.svg',
  volcengine: '/logos/volcengine.svg',
  google: '/logos/google.svg'
}

function getProviderLogo(provider: string): string {
  return providerLogos[provider] || ''
}

function getModelDisplayName(model: string): string {
  // Truncate long model names
  if (model.length > 20) {
    return model.substring(0, 17) + '...'
  }
  return model
}

function createNewChat(): void {
  chatStore.createNewConversation(selectedProvider.value, selectedModel.value)
  sidebarCollapsed.value = true
  nextTick(() => {
    chatInputRef.value?.focus()
  })
}

function switchChat(conversationId: string): void {
  chatStore.switchConversation(conversationId)
  sidebarCollapsed.value = true
  nextTick(() => {
    scrollToBottom()
    chatInputRef.value?.focus()
  })
}

async function confirmDelete(conversationId: string): Promise<void> {
  try {
    await ElMessageBox.confirm(t('chat.confirmDelete'), t('common.warning'), {
      confirmButtonText: t('common.delete'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    chatStore.deleteConversation(conversationId)
    ElMessage.success(t('common.success'))
  } catch {
    // User cancelled
  }
}

function handleModelChange(provider: string, model: string): void {
  selectedProvider.value = provider
  selectedModel.value = model
  if (chatStore.currentConversation) {
    chatStore.updateCurrentModel(provider, model)
  }
}

function notifyDeepThinkUnsupported(): void {
  if (!deepThinkEnabled.value || !chatStore.currentConversation) return

  const currentProvider = chatStore.currentConversation.provider
  const currentModel = chatStore.currentConversation.model
  const isReasonerModel = currentModel.includes('reasoner') || currentModel.includes('r1')

  if (currentProvider === 'deepseek' && isReasonerModel) return

  // 改动点: 不自动切换模型，仅提示当前模型可能不输出推理内容
  ElMessage.warning('当前模型可能不输出深度思考过程，请手动切换到支持推理的模型')
}

async function handleSend(text: string, files: any[] = []): Promise<void> {
  if ((!text.trim() && files.length === 0) || !chatStore.currentConversation) return

  notifyDeepThinkUnsupported()

  const conversationId = chatStore.currentConversation.id

  // Extract images from files
  const imageFiles = files.filter(f => f.isImage && f.base64)
  const imageUrls = imageFiles.map(f => f.base64)
  const fileNames = files.filter(f => !f.isImage).map(f => f.name)

  // Add user message with images/files info
  const userMessage = createMessage('user', text || (imageUrls.length > 0 ? '[图片]' : ''))
  if (imageUrls.length > 0) {
    userMessage.images = imageUrls
  }
  if (fileNames.length > 0) {
    userMessage.files = fileNames
  }
  chatStore.addMessage(userMessage)

  // Create assistant message placeholder
  const assistantMessage: ChatMessageType = {
    id: `assistant-${Date.now()}`,
    role: 'assistant',
    content: '',
    timestamp: Date.now(),
    isStreaming: true
  }
  chatStore.addMessage(assistantMessage)

  chatStore.setLoading(true)

  // Handle web search
  let searchContext = ''
  if (webSearchEnabled.value && text.trim()) {
    try {
      const searchResult = await search(text, 3)
      if (searchResult.success && searchResult.data && searchResult.data.length > 0) {
        searchContext = '\n\n### 联网搜索结果:\n'
        for (let i = 0; i < searchResult.data.length; i++) {
          const result = searchResult.data[i]
          if (result) {
            searchContext += `${i + 1}. **${result.title}**\n   ${result.snippet}\n   来源: ${result.link}\n`
          }
        }
        searchContext += '\n请基于以上搜索结果回答问题。'
      }
    } catch (e) {
      console.error('Search failed:', e)
    }
  }

  // Build messages with search context if available
  if (searchContext) {
    const lastUserMsg = chatStore.currentMessages
      .filter(m => m.role === 'user')
      .pop()
    if (lastUserMsg) {
      // Update the last user message content to include search results
      const msgIndex = chatStore.currentMessages.findIndex(m => m.id === lastUserMsg.id)
      if (msgIndex >= 0) {
        const updatedContent = lastUserMsg.content + ' [联网搜索已启用]'
        chatStore.updateMessage(lastUserMsg.id, { content: updatedContent })
      }
    }
  }

  // Prepare messages for API
  const messages = chatStore.currentMessages
    .filter(m => m.id !== assistantMessage.id && (m.content || m.images?.length))
    .map(m => {
      const content = m.content || ''
      const hasSearch = content.includes('[联网搜索已启用]')
      const finalContent = (hasSearch && searchContext)
        ? content.replace(' [联网搜索已启用]', '') + searchContext
        : content

      if (m.images && m.images.length > 0) {
        return {
          role: m.role,
          content: [
            { type: 'text', text: finalContent },
            ...m.images.map((url: string) => ({
              type: 'image_url',
              image_url: { url }
            }))
          ]
        }
      }
      return {
        role: m.role,
        content: finalContent
      }
    })

  // Track stats
  const startTime = Date.now()
  let firstTokenTime: number | undefined
  let outputChars = 0
  
  const promptChars = messages.reduce((sum, m) => {
    if (typeof m.content === 'string') {
      return sum + m.content.length
    }
    if (Array.isArray(m.content)) {
      return sum + m.content.reduce((s: number, p: any) => s + (p.text?.length || 0), 0)
    }
    return sum
  }, 0)

  // Start streaming
  const controller = streamCompletion(
    {
      model: chatStore.currentConversation.model,
      provider: chatStore.currentConversation.provider,
      messages,
      stream: streamEnabled.value,
      deepThink: deepThinkEnabled.value
    },
    // onChunk
    (chunk) => {
      const delta = chunk.choices?.[0]?.delta
      const content = delta?.content
      const reasoning = delta?.reasoning || delta?.reasoning_content

      if (reasoning) {
        if (!firstTokenTime) {
          firstTokenTime = Date.now()
        }
        chatStore.appendMessageReasoning(assistantMessage.id, reasoning)
        scrollToBottom()
      }

      if (content) {
        if (!firstTokenTime) {
          firstTokenTime = Date.now()
        }
        outputChars += content.length
        chatStore.appendMessageContent(assistantMessage.id, content)
        scrollToBottom()
      }
    },
    // onError
    (error) => {
      chatStore.updateMessage(assistantMessage.id, {
        isStreaming: false,
        error: error.message
      })
      chatStore.setLoading(false)
      chatStore.setAbortController(conversationId, null)
      ElMessage.error(error.message)
    },
    // onComplete
    (totalTokens?: number, promptTokens?: number, completionTokens?: number) => {
      const endTime = Date.now()
      const totalTimeMs = endTime - startTime
      const firstTokenMs = firstTokenTime ? firstTokenTime - startTime : undefined
      
      const completionTimeSec = firstTokenTime ? (endTime - firstTokenTime) / 1000 : 0
      const estimatedPromptTokens = promptTokens || Math.round(promptChars / 2)
      const estimatedCompletionTokens = completionTokens || Math.round(outputChars / 2)
      const estimatedTotalTokens = totalTokens || (estimatedPromptTokens + estimatedCompletionTokens)
      const tokensPerSecond = completionTimeSec > 0 ? Math.round(estimatedCompletionTokens / completionTimeSec) : undefined
      
      chatStore.updateMessage(assistantMessage.id, {
        isStreaming: false,
        stats: {
          firstTokenTime: firstTokenMs ? firstTokenMs / 1000 : undefined,
          totalTime: totalTimeMs / 1000,
          outputTokensPerSecond: tokensPerSecond,
          totalTokens: estimatedTotalTokens,
          promptTokens: estimatedPromptTokens,
          completionTokens: estimatedCompletionTokens
        }
      })
      chatStore.setLoading(false)
      chatStore.setAbortController(conversationId, null)
    }
  )

  chatStore.setAbortController(conversationId, controller)
  scrollToBottom()
}

function handleStop(): void {
  if (currentConversationId.value) {
    chatStore.abortRequest(currentConversationId.value)
    chatStore.setLoading(false)
  }
}

function scrollToBottom(): void {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

// Watch for conversation changes to update selected model
watch(currentConversation, (conv) => {
  if (conv) {
    selectedProvider.value = conv.provider
    selectedModel.value = conv.model
  }
}, { immediate: true })

function applyPrimaryColor(color: string) {
  document.documentElement.style.setProperty('--color-primary', color)
  const r = parseInt(color.slice(1, 3), 16)
  const g = parseInt(color.slice(3, 5), 16)
  const b = parseInt(color.slice(5, 7), 16)
  const lightColor = `rgba(${r}, ${g}, ${b}, 0.8)`
  document.documentElement.style.setProperty('--color-primary-light', lightColor)
}

function selectFirstAvailableProvider(): void {
  if (PROVIDERS.value.length > 0 && !selectedProvider.value) {
    const firstProvider = PROVIDERS.value[0]
    if (firstProvider) {
      selectedProvider.value = firstProvider.value
      if (firstProvider.models && firstProvider.models.length > 0) {
        selectedModel.value = firstProvider.models[0] ?? ''
      }
    }
  }
}

onMounted(async () => {
  const savedPrimaryColor = localStorage.getItem('ai-gateway-primary-color')
  if (savedPrimaryColor) {
    applyPrimaryColor(savedPrimaryColor)
  }

  const savedExpand = localStorage.getItem('chat_reasoning_default_expand')
  if (savedExpand !== null) {
    defaultExpandReasoning.value = savedExpand === 'true'
  }

  const savedAnswerHighlight = localStorage.getItem('chat_answer_highlight')
  if (savedAnswerHighlight !== null) {
    answerHighlightEnabled.value = savedAnswerHighlight === 'true'
  }
  
  await initializeProviders()
  selectFirstAvailableProvider()
  
  window.addEventListener('resize', () => {
    if (window.innerWidth < 768) {
      sidebarCollapsed.value = true
    }
  })
})

watch(defaultExpandReasoning, (val) => {
  localStorage.setItem('chat_reasoning_default_expand', String(val))
})

watch(answerHighlightEnabled, (val) => {
  localStorage.setItem('chat_answer_highlight', String(val))
})
</script>

<style lang="scss" scoped>
.chat-page {
  display: flex;
  height: calc(100vh - 60px);
  background: var(--bg-primary);

  &.public-page {
    height: 100vh;
  }
}

.sidebar {
  width: 260px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  transition: transform var(--transition-base), width var(--transition-base);

  &.collapsed {
    transform: translateX(-100%);
    width: 0;
    border: none;
  }

  @media (min-width: 768px) {
    &.collapsed {
      transform: none;
      width: 260px;
      border-right: 1px solid var(--border-color);
    }
  }
}

.sidebar-header {
  padding: var(--spacing-lg);
  border-bottom: 1px solid var(--border-color);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.new-chat-btn {
  width: 100%;
  height: 44px;
  border-radius: var(--border-radius-md);
  font-weight: 600;
}

.batch-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-sm);
  padding-top: var(--spacing-xs);
}

.conversation-checkbox {
  margin-right: 4px;
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-sm);
}

.conversation-item {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  padding: var(--spacing-md);
  border-radius: var(--border-radius-md);
  cursor: pointer;
  transition: background-color var(--transition-fast);
  margin-bottom: var(--spacing-xs);

  &:hover {
    background: var(--bg-tertiary);

    .delete-btn {
      opacity: 1;
    }
  }

  &.active {
    background: color-mix(in srgb, var(--color-primary) 12%, transparent);
    border-left: 3px solid var(--color-primary);
    padding-left: calc(var(--spacing-md) - 3px);

    .title,
    .icon {
      color: var(--color-primary);
    }

    .model-badge {
      background: color-mix(in srgb, var(--color-primary) 20%, transparent);
      color: var(--color-primary);
    }

    .delete-btn {
      color: var(--color-primary);

      &:hover {
        background: color-mix(in srgb, var(--color-primary) 20%, transparent);
      }
    }

    .provider-logo-sidebar {
      filter: none;
    }
  }

  .icon {
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .provider-logo-sidebar {
    width: 20px;
    height: 20px;
    border-radius: 4px;
    object-fit: contain;
    flex-shrink: 0;
  }

  .conversation-content {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .title {
    font-size: var(--font-size-sm);
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .model-badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 10px;
    background: var(--el-fill-color-light);
    color: var(--el-text-color-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: inline-block;
    max-width: fit-content;
  }

  .delete-btn {
    opacity: 0;
    padding: 4px;
    color: var(--text-secondary);
    transition: opacity var(--transition-fast);

    &:hover {
      background: var(--bg-primary);
      color: var(--color-danger);
    }
  }
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  position: relative;
}

.mobile-menu-btn {
  position: absolute;
  top: var(--spacing-md);
  left: var(--spacing-md);
  z-index: 10;
  width: 40px;
  height: 40px;
  border-radius: var(--border-radius-md);
  background: var(--bg-glass);
  backdrop-filter: blur(10px);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;

  @media (min-width: 768px) {
    display: none;
  }
}

.welcome-screen {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-xl);
}

.welcome-content {
  text-align: center;
  max-width: 500px;

  h1 {
    font-size: var(--font-size-3xl);
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: var(--spacing-md);
  }

  p {
    font-size: var(--font-size-lg);
    color: var(--text-secondary);
    margin-bottom: var(--spacing-xl);
  }
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md) 0;
  scroll-behavior: smooth;

  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--border-color);
    border-radius: 3px;

    &:hover {
      background: var(--text-tertiary);
    }
  }
}

.loading-indicator {
  display: flex;
  justify-content: center;
  padding: var(--spacing-lg);

  .el-icon {
    font-size: 24px;
    color: var(--color-primary);
  }
}

.input-area {
  flex-shrink: 0;
  padding-bottom: 80px;
}

.chat-settings {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
  padding: var(--spacing-sm) var(--spacing-lg) 0 var(--spacing-lg);
  max-width: 900px;
  margin: 0 auto;

  .setting-item {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--text-secondary);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: var(--border-radius-md);
    padding: 6px 10px;
  }
}

.batch-dialog-content {
  .selected-conversations {
    margin-bottom: var(--spacing-lg);
    
    .label {
      font-size: 14px;
      color: var(--el-text-color-regular);
      margin-bottom: var(--spacing-sm);
    }
    
    .conversation-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      max-height: 150px;
      overflow-y: auto;
      
      .conv-tag {
        display: flex;
        align-items: center;
        gap: 4px;
        
        .tag-logo {
          width: 14px;
          height: 14px;
          border-radius: 2px;
          object-fit: contain;
        }
      }
    }
  }
  
  .batch-input {
    :deep(.el-textarea__inner) {
      font-size: 14px;
    }
  }
}
</style>
