<template>
  <div class="chat-input">
    <div class="feature-bar">
      <div class="feature-toggles">
        <el-tooltip content="上传图片或文件，支持PDF/Word/Excel/PPT/图片" placement="top">
          <button 
            class="feature-btn" 
            :class="{ active: multimodalEnabled }"
            @click="toggleMultimodal"
          >
            <el-icon><Picture /></el-icon>
            <span>多模态</span>
          </button>
        </el-tooltip>
        <el-tooltip content="实时搜索互联网信息" placement="top">
          <button 
            class="feature-btn" 
            :class="{ active: webSearchEnabled }"
            @click="toggleWebSearch"
          >
            <el-icon><Search /></el-icon>
            <span>联网搜索</span>
          </button>
        </el-tooltip>
        <el-tooltip content="深度推理模式，适合复杂问题（开发中）" placement="top">
          <button 
            class="feature-btn" 
            :class="{ active: deepThinkEnabled }"
            @click="toggleDeepThink"
            disabled
          >
            <el-icon><Cpu /></el-icon>
            <span>深度思考</span>
          </button>
        </el-tooltip>
      </div>
      <div class="model-hint" v-if="uploadedFiles.length > 0">
        <el-icon><InfoFilled /></el-icon>
        <span>文件较多时建议使用 GLM-4-Flash 降低费用</span>
      </div>
    </div>

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

    <div class="input-wrapper">
      <div class="input-actions-left">
        <el-tooltip content="上传文件 (支持PDF/Word/Excel/PPT/图片)" placement="top">
          <button 
            class="upload-btn" 
            @click="triggerUpload"
            :disabled="disabled || isLoading || uploadedFiles.length >= 10"
          >
            <el-icon><Upload /></el-icon>
          </button>
        </el-tooltip>
        <input 
          ref="fileInputRef" 
          type="file" 
          multiple 
          accept=".pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.md,.png,.jpg,.jpeg,.gif,.webp"
          style="display: none"
          @change="handleFileSelect"
        />
      </div>
      
      <el-input
        ref="inputRef"
        v-model="inputText"
        type="textarea"
        :rows="1"
        :autosize="{ minRows: 1, maxRows: 6 }"
        :placeholder="placeholder || (uploadedFiles.length > 0 ? '描述文件内容或提出问题...' : t('chat.placeholder'))"
        :disabled="disabled"
        resize="none"
        @keydown="handleKeydown"
        class="main-input"
      />
      
      <div class="input-actions-right">
        <span class="hint">{{ t('chat.sendHint') }}</span>
        <el-button
          v-if="!isLoading"
          type="primary"
          :disabled="(!inputText.trim() && uploadedFiles.length === 0) || disabled"
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

    <div class="upload-hint">
      <span>支持 PDF、Word、Excel、PPT、Text、图片，最多 10 个文件，单个文件最大 50MB，图片最大 5MB</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { 
  Promotion, VideoPause, Picture, Search, Cpu, 
  Upload, Document, Close, InfoFilled 
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

const { t } = useI18n()
const inputText = ref('')
const inputRef = ref()
const fileInputRef = ref()

const multimodalEnabled = ref(false)
const webSearchEnabled = ref(false)
const deepThinkEnabled = ref(false)

const uploadedFiles = ref<UploadedFile[]>([])

const MAX_FILES = 10
const MAX_FILE_SIZE = 50 * 1024 * 1024
const MAX_IMAGE_SIZE = 5 * 1024 * 1024

function toggleMultimodal() {
  multimodalEnabled.value = !multimodalEnabled.value
  emit('update:multimodalEnabled', multimodalEnabled.value)
}

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

async function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (!files) return

  const remainingSlots = MAX_FILES - uploadedFiles.value.length
  const filesToProcess = Array.from(files).slice(0, remainingSlots)

  for (const file of filesToProcess) {
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

  target.value = ''
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
  padding: var(--spacing-sm) var(--spacing-md);
  background: var(--bg-glass);
  backdrop-filter: blur(20px);
  border-top: 1px solid var(--border-color);
}

.feature-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--spacing-sm);
  padding-bottom: var(--spacing-sm);
  border-bottom: 1px solid var(--border-secondary);
}

.feature-toggles {
  display: flex;
  gap: var(--spacing-xs);
}

.feature-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: 1px solid var(--border-secondary);
  border-radius: var(--border-radius-md);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &.active {
    background: rgba(0, 122, 255, 0.1);
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .el-icon {
    font-size: 14px;
  }
}

.model-hint {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--text-tertiary);

  .el-icon {
    font-size: 12px;
  }
}

.uploaded-files {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-xs);
  margin-bottom: var(--spacing-sm);
  max-height: 120px;
  overflow-y: auto;
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

.input-wrapper {
  max-width: 900px;
  margin: 0 auto;
  display: flex;
  align-items: flex-end;
  gap: var(--spacing-sm);
}

.input-actions-left {
  display: flex;
  flex-shrink: 0;
}

.upload-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-secondary);
  border-radius: var(--border-radius-md);
  background: var(--bg-tertiary);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    border-color: var(--color-primary);
    color: var(--color-primary);
  }

  &:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
}

.main-input {
  flex: 1;

  :deep(.el-textarea__inner) {
    padding: var(--spacing-sm) var(--spacing-md);
    padding-bottom: 36px;
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

.input-actions-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-shrink: 0;
}

.hint {
  font-size: var(--font-size-xs);
  color: var(--text-tertiary);
  white-space: nowrap;
}

.upload-hint {
  text-align: center;
  margin-top: var(--spacing-xs);
  
  span {
    font-size: 11px;
    color: var(--text-tertiary);
  }
}

@media (max-width: 768px) {
  .feature-btn span {
    display: none;
  }
  
  .hint {
    display: none;
  }
  
  .model-hint {
    display: none;
  }
  
  .upload-hint {
    display: none;
  }
}
</style>
