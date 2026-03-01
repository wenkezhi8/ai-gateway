<template>
  <div class="knowledge-chat-page">
    <el-card>
      <template #header>
        <div class="chat-header">
          <div>
            <div class="title">知识库问答</div>
            <div class="subtitle">输入问题后，系统从向量知识库检索并返回引用来源。</div>
          </div>
          <div class="controls">
            <el-input-number v-model="topK" :min="1" :max="20" />
            <el-input-number v-model="threshold" :min="0" :max="1" :step="0.1" />
          </div>
        </div>
      </template>

      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />

      <div class="messages" ref="messagesRef">
        <el-empty v-if="messages.length === 0" description="还没有问答记录，先提一个问题" />
        <div v-for="item in messages" :key="item.id" class="message" :class="item.role">
          <div class="bubble">
            <div class="role">{{ item.role === 'user' ? '你' : '助手' }}</div>
            <div class="content">{{ item.content }}</div>
            <div v-if="item.sources && item.sources.length > 0" class="sources">
              <el-collapse>
                <el-collapse-item v-for="(s, idx) in item.sources" :key="idx" :title="`${s.document_name}（score=${s.score}）`">
                  <pre>{{ s.content }}</pre>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
        </div>
      </div>

      <div class="input-box">
        <el-input
          v-model="query"
          type="textarea"
          :rows="3"
          placeholder="请输入问题，例如：默认向量后端是什么？"
          @keydown.enter.ctrl="submit"
        />
        <div class="actions">
          <span>Ctrl + Enter 发送</span>
          <el-button type="primary" :loading="loading" :disabled="!query.trim()" @click="submit">发送</el-button>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { sendKnowledgeChatMessage, type KnowledgeSource } from '@/api/knowledge-domain'

interface ChatItem {
  id: string
  role: 'user' | 'assistant'
  content: string
  sources?: KnowledgeSource[]
}

const query = ref('')
const loading = ref(false)
const error = ref('')
const topK = ref(5)
const threshold = ref(0.7)
const messages = ref<ChatItem[]>([])
const messagesRef = ref<HTMLElement | null>(null)

async function submit() {
  const text = query.value.trim()
  if (!text || loading.value) return
  error.value = ''
  messages.value.push({ id: `${Date.now()}-q`, role: 'user', content: text })
  query.value = ''
  loading.value = true
  try {
    const data = await sendKnowledgeChatMessage({
      query: text,
      top_k: topK.value,
      similarity_threshold: threshold.value
    })
    messages.value.push({
      id: `${Date.now()}-a`,
      role: 'assistant',
      content: data.answer,
      sources: data.sources || []
    })
  } catch (e: any) {
    error.value = e?.message || '问答请求失败'
  } finally {
    loading.value = false
    await nextTick()
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
    }
  }
}
</script>

<style scoped>
.knowledge-chat-page {
  padding: 8px;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.title {
  font-size: 18px;
  font-weight: 600;
}

.subtitle {
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.controls {
  display: flex;
  gap: 8px;
}

.messages {
  min-height: 260px;
  max-height: 50vh;
  overflow: auto;
  padding: 8px 0;
}

.message {
  display: flex;
  margin-bottom: 12px;
}

.message.user {
  justify-content: flex-end;
}

.message.assistant {
  justify-content: flex-start;
}

.bubble {
  max-width: 72%;
  background: #f5f7fb;
  border-radius: 10px;
  padding: 10px 12px;
}

.message.user .bubble {
  background: #ecf5ff;
}

.role {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 4px;
}

.content {
  white-space: pre-wrap;
  line-height: 1.6;
}

.sources {
  margin-top: 8px;
}

.sources pre {
  white-space: pre-wrap;
  margin: 0;
}

.input-box {
  margin-top: 12px;
}

.actions {
  margin-top: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: var(--el-text-color-secondary);
}
</style>
