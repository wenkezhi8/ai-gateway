<template>
  <div class="chat-input">
    <div class="uploaded-files" v-if="uploadedFiles.length > 0">
      <div
        v-for="(file, index) in uploadedFiles"
        :key="index"
        class="file-item"
        :class="{ 'is-image': file.isImage }"
      >
        <div class="file-preview" v-if="file.isImage">
          <img :src="file.preview" :alt="file.name" />
        </div>
        <div class="file-icon" v-else>
          <el-icon><Document /></el-icon>
        </div>
        <div class="file-info">
          <span class="file-name">{{ file.name }}</span>
          <span class="file-size">{{ formatFileSize(file.size) }}</span>
        </div>
        <button class="file-remove" @click="removeFile(index)">
          <el-icon><Close /></el-icon>
        </button>
      </div>
    </div>

    <div
      class="input-box"
      :class="{ 'drag-over': isDragOver }"
      @drop.prevent="handleDrop"
      @dragover.prevent="isDragOver = true"
      @dragleave.prevent="isDragOver = false"
    >
      <div class="input-inner">
        <el-input
          ref="inputRef"
          v-model="inputText"
          type="textarea"
          :rows="1"
          :autosize="{ minRows: 1, maxRows: 6 }"
          :placeholder="placeholder || (uploadedFiles.length > 0 ? '描述文件内容或提出问题...' : '向AI提问...')"
          :disabled="disabled"
          resize="none"
          @keydown="handleKeydown"
          class="main-input"
        />

        <div class="input-bottom-bar">
          <div class="feature-buttons">
            <el-tooltip content="上传文件" placement="top">
              <button
                class="feature-btn"
                @click="triggerUpload"
                :disabled="disabled || isLoading || uploadedFiles.length >= 10"
              >
                <el-icon><Upload /></el-icon>
              </button>
            </el-tooltip>
            <el-tooltip content="联网搜索" placement="top">
              <button
                class="feature-btn"
                :class="{ active: webSearchEnabled }"
                @click="toggleWebSearch"
              >
                <el-icon><Search /></el-icon>
              </button>
            </el-tooltip>
            <el-tooltip content="深度思考" placement="top">
              <button
                class="feature-btn"
                :class="{ active: deepThinkEnabled }"
                @click="toggleDeepThink"
              >
                <el-icon><Cpu /></el-icon>
              </button>
            </el-tooltip>
          </div>

          <div class="send-area">
            <el-button
              v-if="!isLoading"
              type="primary"
              :disabled="(!inputText.trim() && uploadedFiles.length === 0) || disabled"
              @click="handleSend"
              circle
              size="large"
            >
              <el-icon><Promotion /></el-icon>
            </el-button>
            <el-button
              v-else
              type="danger"
              @click="handleStop"
              circle
              size="large"
            >
              <el-icon><VideoPause /></el-icon>
            </el-button>
          </div>
        </div>
      </div>

      <input
        ref="fileInputRef"
        type="file"
        multiple
        accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.md,.png,.jpg,.jpeg,.gif,.webp"
        style="display: none"
        @change="handleFileSelect"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import {
  Promotion, VideoPause, Search, Cpu,
  Upload, Document, Close
} from '@element-plus/icons-vue'

export interface UploadedFile {
  file: File
  name: string
  size: number
  type: string
  isImage: boolean
  preview?: string
  base64?: string
}

const props = defineProps<{
  disabled?: boolean
  isLoading?: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  send: [text: string, files: UploadedFile[]]
  stop: []
  'update:multimodalEnabled': [value: boolean]
  'update:webSearchEnabled': [value: boolean]
  'update:deepThinkEnabled': [value: boolean]
}>()

const inputText = ref('')
const inputRef = ref()
const fileInputRef = ref()
const isDragOver = ref(false)

const multimodalEnabled = ref(false)
const webSearchEnabled = ref(false)
const deepThinkEnabled = ref(false)
const uploadedFiles = ref<UploadedFile[]>([])

const MAX_FILES = 10
const MAX_FILE_SIZE = 50 * 1024 * 1024
const MAX_IMAGE_SIZE = 5 * 1024 * 1024

function toggleWebSearch() {
  webSearchEnabled.value = !webSearchEnabled.value
  emit('update:webSearchEnabled', webSearchEnabled.value)
}

function toggleDeepThink() {
  deepThinkEnabled.value = !deepThinkEnabled.value
  emit('update:deepThinkEnabled', deepThinkEnabled.value)
}

function triggerUpload() {
  fileInputRef.value?.click()
}

async function processFiles(files: FileList | File[]) {
  const remainingSlots = MAX_FILES - uploadedFiles.value.length
  const filesArray = Array.from(files).slice(0, remainingSlots)

  for (const file of filesArray) {
    const isImage = file.type.startsWith('image/')
    const maxSize = isImage ? MAX_IMAGE_SIZE : MAX_FILE_SIZE

    if (file.size > maxSize) {
      ElMessage.warning(`${file.name} 超过大小限制 (${isImage ? '5MB' : '50MB'})`)
      continue
    }

    const uploadedFile: UploadedFile = {
      file,
      name: file.name,
      size: file.size,
      type: file.type,
      isImage
    }

    if (isImage) {
      try {
        uploadedFile.preview = URL.createObjectURL(file)
        uploadedFile.base64 = await fileToBase64(file)
      } catch (e) {
        console.error('Failed to read image:', e)
      }
    }

    uploadedFiles.value.push(uploadedFile)
  }

  if (uploadedFiles.value.length > 0) {
    multimodalEnabled.value = true
    emit('update:multimodalEnabled', true)
  }
}

async function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files) return

  await processFiles(files)
  target.value = ''
}

async function handleDrop(event: DragEvent) {
  isDragOver.value = false
  const files = event.dataTransfer?.files
  if (!files || files.length === 0) return

  await processFiles(files)
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve(result)
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

function removeFile(index: number) {
  const file = uploadedFiles.value[index]
  if (file?.preview) {
    URL.revokeObjectURL(file.preview)
  }
  uploadedFiles.value.splice(index, 1)

  if (uploadedFiles.value.length === 0) {
    multimodalEnabled.value = false
    emit('update:multimodalEnabled', false)
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    handleSend()
  }
}

function handleSend(): void {
  const text = inputText.value.trim()
  const hasFiles = uploadedFiles.value.length > 0

  if ((text || hasFiles) && !props.disabled && !props.isLoading) {
    const files = [...uploadedFiles.value]
    emit('send', text, files)
    inputText.value = ''

    files.forEach(file => {
      if (file.preview) {
        URL.revokeObjectURL(file.preview)
      }
    })
    uploadedFiles.value = []

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

<script lang="ts">
import { ElMessage } from 'element-plus'
</script>

<style lang="scss" scoped>
.chat-input {
  padding: var(--spacing-md);
  background: transparent;
}

.uploaded-files {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-sm);
  max-height: 120px;
  overflow-y: auto;
  max-width: 900px;
  margin-left: auto;
  margin-right: auto;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border-secondary);
  border-radius: var(--border-radius-md);
  font-size: 12px;
  max-width: 200px;

  &.is-image {
    border-color: var(--color-primary-light);
    background: rgba(0, 122, 255, 0.05);
  }
}

.file-preview {
  width: 32px;
  height: 32px;
  border-radius: 4px;
  overflow: hidden;
  flex-shrink: 0;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.file-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-secondary);
  border-radius: 4px;
  color: var(--text-secondary);
}

.file-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-name {
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-size {
  color: var(--text-tertiary);
  font-size: 10px;
}

.file-remove {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  background: none;
  color: var(--text-tertiary);
  cursor: pointer;
  border-radius: 4px;
  flex-shrink: 0;

  &:hover {
    background: rgba(245, 108, 108, 0.1);
    color: var(--color-danger);
  }
}

.input-box {
  max-width: 900px;
  margin: 0 auto;
  position: relative;

  &.drag-over {
    .input-inner {
      border-color: var(--color-primary);
      box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.15);
    }

    &::before {
      content: '松开上传文件';
      position: absolute;
      top: -40px;
      left: 50%;
      transform: translateX(-50%);
      background: var(--color-primary);
      color: white;
      padding: 6px 16px;
      border-radius: 20px;
      font-size: 14px;
      z-index: 10;
    }
  }
}

.input-inner {
  background: var(--bg-primary);
  border: 1px solid var(--border-color);
  border-radius: 24px;
  padding: 8px 16px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);

  &:focus-within {
    border-color: var(--color-primary);
    box-shadow: 0 4px 24px rgba(0, 122, 255, 0.15);
  }
}

.main-input {
  :deep(.el-textarea__inner) {
    padding: 8px 0;
    font-size: var(--font-size-base);
    line-height: 1.6;
    border: none;
    background: transparent;
    box-shadow: none;
    resize: none;
    overflow: hidden;

    &::placeholder {
      color: var(--text-tertiary);
    }

    &:focus {
      box-shadow: none;
    }
  }
}

.input-bottom-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 4px;
  border-top: 1px solid var(--border-secondary);
  margin-top: 4px;
}

.feature-buttons {
  display: flex;
  gap: 4px;
}

.feature-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 10px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    background: var(--bg-tertiary);
    color: var(--color-primary);
  }

  &.active {
    background: rgba(0, 122, 255, 0.1);
    color: var(--color-primary);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .el-icon {
    font-size: 18px;
  }
}

.send-area {
  display: flex;
  align-items: center;
}
</style>
