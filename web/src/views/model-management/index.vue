<template>
  <div class="model-management-page">
    <el-alert v-if="sourceContextVisible" type="info" :closable="false" class="source-context-bar">
      <template #title>
        <div class="context-bar-content">
          <div class="context-bar-meta">
            <span>当前服务商</span>
            <el-tag size="small">{{ sourceContextProviderLabel }}</el-tag>
            <span>来源</span>
            <el-tag size="small" type="success">AI服务商</el-tag>
            <el-tag v-if="sourceContextMissingProvider" size="small" type="warning">未匹配到服务商，已回退为普通浏览模式</el-tag>
          </div>
          <el-button link type="primary" @click="goBackToProvidersAccounts">返回AI服务商</el-button>
        </div>
      </template>
    </el-alert>

    <el-row :gutter="24">
      <el-col :span="16">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>服务商默认模型设置</span>
              <div class="header-actions">
                <el-button @click="showAddProviderDialog" :disabled="submitting">
                  <el-icon><Plus /></el-icon>
                  添加服务商
                </el-button>
                <el-button type="primary" @click="saveAllSettings(true)" :loading="saving">
                  <el-icon><Check /></el-icon>
                  保存设置
                </el-button>
              </div>
            </div>
          </template>

          <div class="settings-info">
            <el-alert type="info" :closable="false">
              <template #title>
                设置每个服务商的默认模型。调用 API 时使用 <el-tag size="small">default</el-tag> 会自动使用对应服务商的默认模型。
              </template>
            </el-alert>
          </div>

          <el-table
            ref="providerTableRef"
            :data="providerSettings"
            stripe
            highlight-current-row
            v-loading="loading"
            :row-class-name="getProviderRowClassName"
          >
            <el-table-column label="服务商" width="200">
              <template #default="{ row }">
                <div class="provider-cell">
                  <img
                    v-if="row.logo && !brokenLogoProviders.has(row.id)"
                    :src="row.logo"
                    class="provider-logo"
                    @error="handleLogoError(row.id)"
                  />
                  <div v-else class="provider-icon" :style="{ background: row.color }">
                    <span>{{ row.label.charAt(0) }}</span>
                  </div>
                  <span class="provider-name">{{ row.label }}</span>
                </div>
              </template>
            </el-table-column>
            
            <el-table-column label="默认模型" min-width="250">
              <template #default="{ row }">
                <el-select 
                  v-model="row.defaultModel" 
                  filterable 
                  allow-create
                  default-first-option
                  placeholder="选择或输入模型名称"
                  style="width: 100%"
                  :class="{ 'default-model-select--focus': isDefaultModelFocused(row.id) }"
                  :disabled="submitting"
                  @change="handleModelChange(row)"
                >
                  <el-option
                    v-for="model in row.models"
                    :key="model"
                    :label="getModelLabel(row.id, model)"
                    :value="model"
                  />
                </el-select>
              </template>
            </el-table-column>

            <el-table-column label="可用模型" min-width="200">
              <template #default="{ row }">
                <div class="models-cell">
                  <el-tag 
                    v-for="model in row.models.slice(0, 3)" 
                    :key="model" 
                    size="small"
                    class="model-tag"
                  >
                    {{ getModelLabel(row.id, model) }}
                  </el-tag>
                  <el-tag v-if="row.models.length > 3" size="small" type="info">
                    +{{ row.models.length - 3 }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>

            <el-table-column label="操作" width="200" align="center">
              <template #default="{ row }">
                <el-button 
                  type="primary" 
                  link 
                  size="small" 
                  @click="showEditDialog(row)"
                  :disabled="submitting"
                >
                  编辑
                </el-button>
                <el-button 
                  type="primary" 
                  link 
                  size="small" 
                  @click="showAddModelDialog(row)"
                  :disabled="submitting"
                >
                  添加模型
                </el-button>
                <el-button 
                  type="danger" 
                  link 
                  size="small" 
                  @click="handleDeleteProvider(row)" 
                  :disabled="submitting"
                >
                  删除
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>调用说明</span>
            </div>
          </template>

          <div class="call-modes">
            <div class="mode-item">
              <div class="mode-header">
                <el-tag type="success" size="large">auto</el-tag>
                <span class="mode-title">智能选择</span>
              </div>
              <div class="mode-desc">根据效果+速度+成本综合评分，自动选择最优模型</div>
            </div>
            
            <div class="mode-item">
              <div class="mode-header">
                <el-tag type="warning" size="large">latest</el-tag>
                <span class="mode-title">最新模型</span>
              </div>
              <div class="mode-desc">使用效果评分最高的模型</div>
            </div>
            
            <div class="mode-item">
              <div class="mode-header">
                <el-tag size="large">default</el-tag>
                <span class="mode-title">服务商默认模型</span>
              </div>
              <div class="mode-desc">使用该服务商配置的默认模型</div>
            </div>
          </div>

          <div class="api-example" :class="{ 'api-example--focus': verifyCallFocused }">
            <div class="example-title">API 调用示例</div>
            <pre class="code"><code>POST /api/v1/chat/completions
{
  "model": "default",
  "provider": "deepseek",
  "messages": [...]
}</code></pre>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog 
      v-model="providerDialogVisible" 
      title="添加服务商" 
      width="500px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form 
        :model="providerForm" 
        :rules="providerRules" 
        ref="providerFormRef" 
        label-width="100px"
        @submit.prevent="handleAddProvider"
      >
        <el-form-item label="服务商ID" prop="id">
          <el-input 
            v-model="providerForm.id" 
            placeholder="如: my-provider (小写字母、数字、横线)"
            :disabled="submitting"
          />
        </el-form-item>
        <el-form-item label="服务商名称" prop="label">
          <el-select
            v-model="providerForm.label"
            filterable
            allow-create
            default-first-option
            placeholder="选择或输入服务商名称"
            style="width: 100%"
            :disabled="submitting"
            @change="onProviderLabelChange"
            popper-class="provider-select-dropdown"
          >
            <el-option-group label="国际服务商">
              <el-option v-for="p in internationalProviders" :key="p.value" :label="p.label" :value="p.label">
                <span class="provider-option">
                  <img v-if="getProviderLogo(p.value)" :src="getProviderLogo(p.value)" class="provider-logo" />
                  <span v-else class="dot" :style="{ background: getProviderPaletteColor(p.value) }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
            <el-option-group label="国内服务商">
              <el-option v-for="p in chineseProviders" :key="p.value" :label="p.label" :value="p.label">
                <span class="provider-option">
                  <img v-if="getProviderLogo(p.value)" :src="getProviderLogo(p.value)" class="provider-logo" />
                  <span v-else class="dot" :style="{ background: getProviderPaletteColor(p.value) }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
            <el-option-group label="本地大模型">
              <el-option v-for="p in localProviders" :key="p.value" :label="p.label" :value="p.label">
                <span class="provider-option">
                  <img v-if="getProviderLogo(p.value)" :src="getProviderLogo(p.value)" class="provider-logo" />
                  <span v-else class="dot" :style="{ background: getProviderPaletteColor(p.value) }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
            <el-option-group v-if="customProviders.length > 0" label="自定义服务商">
              <el-option v-for="p in customProviders" :key="p.value" :label="p.label" :value="p.label">
                <span class="provider-option">
                  <span class="dot" :style="{ background: getProviderPaletteColor(p.value) }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="Logo" prop="logoFile">
          <div class="logo-upload">
            <el-upload
              :auto-upload="false"
              :show-file-list="false"
              accept=".svg"
              :on-change="handleLogoChange"
              :disabled="submitting"
            >
              <div class="logo-preview" :style="{ borderColor: providerForm.color }">
                <img v-if="providerForm.logoPreview" :src="providerForm.logoPreview" class="preview-img" />
                <el-icon v-else class="upload-icon"><Plus /></el-icon>
              </div>
            </el-upload>
            <div class="upload-hint">
              <div>上传 SVG 格式 Logo</div>
              <div class="hint-small">建议尺寸: 32x32 或 64x64</div>
            </div>
          </div>
        </el-form-item>
        <el-form-item label="颜色">
          <el-color-picker v-model="providerForm.color" :disabled="submitting" />
        </el-form-item>
        <el-form-item label="默认模型" prop="defaultModel">
          <el-input 
            v-model="providerForm.defaultModel" 
            placeholder="如: my-model-v1"
            :disabled="submitting"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="providerDialogVisible = false" :disabled="submitting">取消</el-button>
        <el-button type="primary" @click="handleAddProvider" :loading="submitting">添加</el-button>
      </template>
    </el-dialog>

    <el-dialog 
      v-model="modelDialogVisible" 
      title="添加模型" 
      width="450px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <el-form 
        :model="modelForm" 
        :rules="modelRules" 
        ref="modelFormRef" 
        label-width="100px"
        @submit.prevent="handleAddModel"
      >
        <el-form-item label="服务商">
          <el-input :value="currentProvider?.label" disabled />
        </el-form-item>
        <el-form-item label="模型名称" prop="model">
          <el-input 
            v-model="modelForm.model" 
            placeholder="如: gpt-4-turbo"
            :disabled="submitting"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="modelDialogVisible = false" :disabled="submitting">取消</el-button>
        <el-button type="primary" @click="handleAddModel" :loading="submitting">添加</el-button>
      </template>
    </el-dialog>

    <!-- Edit Provider Dialog -->
    <el-dialog 
      v-model="editDialogVisible" 
      title="编辑服务商模型" 
      width="600px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <div class="edit-dialog-content">
        <div class="provider-info">
          <img v-if="editProvider?.logo" :src="editProvider.logo" class="edit-provider-logo" />
          <div v-else class="edit-provider-icon" :style="{ background: editProvider?.color }">
            {{ editProvider?.label?.charAt(0) }}
          </div>
          <span class="edit-provider-name">{{ editProvider?.label }}</span>
        </div>

        <el-divider content-position="left">当前模型列表</el-divider>
        
        <div class="models-header">
          <el-checkbox 
            v-model="selectAllModels" 
            :indeterminate="isIndeterminate"
            @change="handleSelectAllModels"
          >全选</el-checkbox>
          <span class="model-count">共 {{ editProvider?.models?.length || 0 }} 个模型</span>
        </div>
        
        <div class="models-list">
          <el-checkbox-group v-model="selectedModels">
            <div v-for="model in editProvider?.models" :key="model" class="model-item">
              <el-checkbox :value="model">{{ getModelLabel(editProvider?.id || '', model) }}</el-checkbox>
            </div>
          </el-checkbox-group>
          <el-empty v-if="!editProvider?.models?.length" description="暂无模型" :image-size="60" />
        </div>

        <div class="batch-actions">
          <el-button 
            type="danger" 
            size="small" 
            :disabled="selectedModels.length === 0"
            @click="handleBatchDeleteModels"
          >
            删除选中 ({{ selectedModels.length }})
          </el-button>
        </div>

        <el-divider content-position="left">批量添加模型</el-divider>
        
        <el-input
          v-model="batchModelsText"
          type="textarea"
          :rows="4"
          placeholder="每行输入一个模型名称，如：&#10;gpt-4-turbo&#10;gpt-4&#10;gpt-3.5-turbo"
        />
      </div>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleBatchAddModels" :loading="submitting">
          批量添加
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, computed, watch, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import { request } from '@/api/request'
import { deleteModelRegistry, getModelRegistry, upsertModelRegistry } from '@/api/routing-domain'
import { updateModelManagementUiSettings } from '@/api/settings-domain'
import { getPublicProviders, providerApi, getProviderTypes } from '@/api/provider'
import { accountApi } from '@/api/account'
import {
  buildProviderOptions,
  type ProviderOption
} from '@/views/providers-accounts/provider-options'
import {
  handleProviderLabelChange,
  createAutoFillContext,
  type ProviderSelectState,
  type AutoFillContext,
  markIdAsManuallyEdited
} from './provider-select-logic'
import { useModelLabels } from '@/composables/useModelLabels'
import {
  MODEL_MANAGEMENT_DEFAULT_COLOR,
  MODEL_MANAGEMENT_FALLBACK_COLOR
} from '@/constants/pages/model-management'
import {
  buildProvidersAccountsBackQuery,
  parseModelManagementContext,
  type ModelManagementContext
} from './provider-context'
import { resolveProviderDisplayMeta } from './provider-display-meta-resolver'
import { buildProviderIdsForSettings } from './provider-settings-sources'

const { getModelLabel, fetchModelLabels } = useModelLabels()
const route = useRoute()
const router = useRouter()

interface ProviderSetting {
  id: string
  label: string
  color: string
  logo?: string
  defaultModel: string
  models: string[]
  custom: boolean
}

const saving = ref(false)
const submitting = ref(false)
const loading = ref(false)

const providerDialogVisible = ref(false)
const modelDialogVisible = ref(false)
const editDialogVisible = ref(false)
const currentProvider = ref<ProviderSetting | null>(null)
const editProvider = ref<ProviderSetting | null>(null)
const selectedModels = ref<string[]>([])
const selectAllModels = ref(false)
const batchModelsText = ref('')

// New data for provider options (from providers-accounts)
const providerTypes = ref<ProviderOption[]>([])
const autoFillContext = ref<AutoFillContext>(createAutoFillContext())
const internationalProviders = computed(() => providerTypes.value.filter(p => p.category === 'international'))
const chineseProviders = computed(() => providerTypes.value.filter(p => p.category === 'chinese'))
const localProviders = computed(() => providerTypes.value.filter(p => p.category === 'local'))
const customProviders = computed(() => providerTypes.value.filter(p => p.category === 'custom'))

const isIndeterminate = computed(() => {
  const total = editProvider.value?.models?.length || 0
  const selected = selectedModels.value.length
  return selected > 0 && selected < total
})

watch(selectedModels, (val: string[]) => {
  const total = editProvider.value?.models?.length || 0
  selectAllModels.value = val.length === total && total > 0
}) // Used in edit dialog

const providerFormRef = ref<FormInstance>()
const modelFormRef = ref()

const providerForm = reactive({
  id: '',
  label: '',
  color: MODEL_MANAGEMENT_DEFAULT_COLOR,
  defaultModel: '',
  logoFile: null as File | null,
  logoPreview: '',
  idAutoFilled: false
})


const modelForm = reactive({
  model: ''
})

const providerRules: FormRules = {
  id: [
    { required: true, message: '请输入服务商ID', trigger: 'blur' },
    { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和横线', trigger: 'blur' }
  ],
  label: [{ required: true, message: '请输入服务商名称', trigger: 'blur' }],
  defaultModel: [{ required: true, message: '请输入默认模型', trigger: 'blur' }]
}

const modelRules: FormRules = {
  model: [{ required: true, message: '请输入模型名称', trigger: 'blur' }]
}

const providerSettings = ref<ProviderSetting[]>([])
const brokenLogoProviders = ref(new Set<string>())
const providerTableRef = ref<any>()
const modelManagementContext = ref<ModelManagementContext>(parseModelManagementContext(route.query))
const focusedProviderId = ref<string | null>(null)
let offModelsChanged: (() => void) | null = null

// Computed helper for provider metadata
const providerMetaMap = computed(() => {
  const entries = providerTypes.value.map(item => [item.value, item] as const)
  return Object.fromEntries(entries) as Record<string, ProviderOption>
})

const getProviderLogo = (provider: string) => providerMetaMap.value[provider]?.logo || ''

const getProviderPaletteColor = (provider: string) => providerMetaMap.value[provider]?.color || '#6B7280'

const sourceContextVisible = computed(() => modelManagementContext.value.from === 'provider' || modelManagementContext.value.from === 'providers-accounts')

const sourceContextProviderLabel = computed(() => {
  const providerId = modelManagementContext.value.providerId
  if (!providerId) return '未指定'
  const target = providerSettings.value.find(item => item.id === providerId)
  return target?.label || providerId
})

const sourceContextMissingProvider = computed(() => {
  const providerId = modelManagementContext.value.providerId
  if (!providerId || loading.value) return false
  return !providerSettings.value.some(item => item.id === providerId)
})

const verifyCallFocused = computed(() => modelManagementContext.value.focus === 'verify-call' && !!focusedProviderId.value)

function isDefaultModelFocused(providerId: string): boolean {
  return modelManagementContext.value.focus === 'default-model' && focusedProviderId.value === providerId
}

function getProviderRowClassName({ row }: { row: ProviderSetting }): string {
  return row.id === focusedProviderId.value ? 'provider-row--highlighted' : ''
}

function applyOnboardingContextFocus() {
  const providerId = modelManagementContext.value.providerId
  const target = providerSettings.value.find(item => item.id === providerId)

  if (!providerId || !target) {
    focusedProviderId.value = null
    nextTick(() => {
      providerTableRef.value?.setCurrentRow?.(null)
    })
    return
  }

  focusedProviderId.value = target.id
  nextTick(() => {
    providerTableRef.value?.setCurrentRow?.(target)
  })
}

function goBackToProvidersAccounts() {
  router.push({
    path: '/providers-accounts',
    query: buildProvidersAccountsBackQuery(modelManagementContext.value)
  })
}

// Watch for manual ID editing to mark as not auto-filled
watch(() => providerForm.id, (newId: string, oldId: string) => {
  if (newId !== oldId && oldId !== '') {
    markIdAsManuallyEdited(autoFillContext.value)
  }
})

watch(
  () => route.query,
  (query) => {
    modelManagementContext.value = parseModelManagementContext(query)
    applyOnboardingContextFocus()
  },
  { immediate: true }
)

function onProviderLabelChange(selectedLabel: string) {
  handleProviderLabelChange(
    providerForm as ProviderSelectState,
    providerTypes.value,
    selectedLabel,
    autoFillContext.value
  )
}

function handleLogoError(providerId: string) {
  if (!providerId) return
  brokenLogoProviders.value.add(providerId)
}


async function loadSettings() {
  loading.value = true
  try {
    // Label loading failure should not block provider visibility
    await fetchModelLabels().catch(() => undefined)

    // Use same data source as providers-accounts for consistency
    // Use same data source as providers-accounts for consistency
    const [typesResult, publicResult, accountsResult, defaultsResult] = await Promise.allSettled([
      getProviderTypes(),
      getPublicProviders(),
      accountApi.getList(),
      request.get('/admin/router/provider-defaults').catch(() => ({ data: {} }))
    ])

    const typeList = typesResult.status === 'fulfilled' ? typesResult.value : []
    const publicProviders = publicResult.status === 'fulfilled' ? publicResult.value : []
    const accountList = accountsResult.status === 'fulfilled' ? ((accountsResult.value as any).data || []) : []
    const providerDefaults = defaultsResult.status === 'fulfilled' ? (defaultsResult.value as any).data || {} : {}


    providerTypes.value = buildProviderOptions({
      types: typeList,
      publicProviders,
      accounts: accountList
    })

    // Load model registry from backend - this is the single source of truth
    const modelsRes = await getModelRegistry().catch(() => [])

    // Group models by provider
    const modelsByProvider: Record<string, string[]> = {}
    if (Array.isArray(modelsRes)) {
      for (const m of modelsRes) {
        if (m.enabled && m.model) {
          if (!modelsByProvider[m.provider]) {
            modelsByProvider[m.provider] = []
          }
          const providerModels = modelsByProvider[m.provider]
          if (providerModels && !providerModels.includes(m.model)) {
            providerModels.push(m.model)
          }
        }
      }
    }

    // Build provider settings from backend models and provider metadata
    const providerIds = buildProviderIdsForSettings({
      publicProviders,
      modelsByProvider,
      providerDefaults
    })
    const newSettings: ProviderSetting[] = []

    for (const providerId of providerIds) {
      const displayMeta = resolveProviderDisplayMeta(providerId, {
        providerTypes: typeList,
        publicProviders,
        fallbackColor: MODEL_MANAGEMENT_FALLBACK_COLOR
      })
      const models = modelsByProvider[providerId] || []
      const defaultModel = providerDefaults[providerId] || displayMeta.defaultModel || models[0] || ''

      newSettings.push({
        id: providerId,
        label: displayMeta.label,
        color: displayMeta.color,
        logo: displayMeta.logo,
        defaultModel,
        models,
        custom: displayMeta.custom
      })
    }

    brokenLogoProviders.value.clear()
    providerSettings.value = newSettings
    applyOnboardingContextFocus()
  } catch (e) {
    console.error('Failed to load settings:', e)
  } finally {
    loading.value = false
  }
}

async function saveAllSettings(showMessage = true) {
  if (saving.value) return
  
  saving.value = true
  try {
    const settings: Record<string, string> = {}
    providerSettings.value.forEach(p => {
      if (p.defaultModel) {
        settings[p.id] = p.defaultModel
      }
    })

    await request.put('/admin/router/provider-defaults', settings)
    await updateModelManagementUiSettings({
      last_saved_at: new Date().toISOString()
    })
    if (showMessage) {
      ElMessage.success('设置已保存')
    }
    eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
  } catch (e: any) {
    ElMessage.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function handleModelChange(_row: ProviderSetting) {
  saveAllSettings(false)
}

function showAddProviderDialog() {
  Object.assign(providerForm, { id: '', label: '', color: MODEL_MANAGEMENT_DEFAULT_COLOR, defaultModel: '', logoFile: null, logoPreview: '' })
  autoFillContext.value = createAutoFillContext()
  providerDialogVisible.value = true
}

function handleLogoChange(file: any) {
  const rawFile = file.raw
  if (!rawFile) return
  
  // Validate file type
  if (!rawFile.name.endsWith('.svg')) {
    ElMessage.warning('请上传 SVG 格式文件')
    return
  }
  
  providerForm.logoFile = rawFile
  
  // Create preview
  const reader = new FileReader()
  reader.onload = (e) => {
    providerForm.logoPreview = e.target?.result as string
  }
  reader.readAsDataURL(rawFile)
  
  // Auto-generate ID from filename
  if (!providerForm.id) {
    const fileName = rawFile.name.replace('.svg', '').toLowerCase().replace(/[^a-z0-9-]/g, '-')
    providerForm.id = fileName
    if (!providerForm.label) {
      providerForm.label = rawFile.name.replace('.svg', '')
    }
  }
}

async function handleAddProvider() {
  if (!providerFormRef.value) return
  
  const valid = await providerFormRef.value.validate().catch(() => false)
  if (!valid) return

  if (providerSettings.value.some(p => p.id === providerForm.id)) {
    ElMessage.error('服务商ID已存在')
    return
  }

  submitting.value = true
  try {
    let logoPath = `/logos/${providerForm.id}.svg`
    if (!providerForm.logoFile && providerForm.logoPreview) {
      logoPath = providerForm.logoPreview
    }
    
    // Upload logo file if exists
    if (providerForm.logoFile) {
      const formData = new FormData()
      formData.append('file', providerForm.logoFile)
      formData.append('filename', `${providerForm.id}.svg`)
      
      try {
        await request.post('/admin/upload/logo', formData, {
          headers: { 'Content-Type': 'multipart/form-data' }
        })
      } catch (e) {
        logoPath = providerForm.logoPreview
      }
    }
    
    const newProvider: ProviderSetting = {
      id: providerForm.id,
      label: providerForm.label || providerForm.id,
      color: providerForm.color,
      logo: logoPath,
      defaultModel: providerForm.defaultModel,
      models: providerForm.defaultModel ? [providerForm.defaultModel] : [],
      custom: true
    }

    // Sync provider default model to backend model registry (single source of truth)
    await upsertModelRegistry(providerForm.defaultModel, {
      provider: providerForm.id,
      display_name: providerForm.defaultModel,
      enabled: true
    })

    providerSettings.value.push(newProvider)
    await saveAllSettings(false)
    providerDialogVisible.value = false
    ElMessage.success('服务商已添加')
    eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
  } catch (e: any) {
    ElMessage.error(e?.message || '添加失败')
  } finally {
    submitting.value = false
  }
}

function showAddModelDialog(provider: ProviderSetting) {
  currentProvider.value = provider
  modelForm.model = ''
  modelDialogVisible.value = true
}

async function handleAddModel() {
  if (!modelFormRef.value) return
  
  const valid = await modelFormRef.value.validate().catch(() => false)
  if (!valid) return

  if (!currentProvider.value) return

  const modelName = modelForm.model.trim()
  if (currentProvider.value.models.includes(modelName)) {
    ElMessage.warning('该模型已存在')
    return
  }

  submitting.value = true
  try {
    const idx = providerSettings.value.findIndex(p => p.id === currentProvider.value!.id)
    if (idx > -1) {
      const provider = providerSettings.value[idx]!
      provider.models = [...provider.models, modelName]
      currentProvider.value = provider
    }
    
    // Sync to backend model registry
    await upsertModelRegistry(modelName, {
      provider: currentProvider.value.id,
      display_name: modelName,
      enabled: true
    })
    
    modelDialogVisible.value = false
    ElMessage.success('模型已添加并同步')
    eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
  } catch (e: any) {
    ElMessage.error(e?.message || '添加失败')
  } finally {
    submitting.value = false
  }
}

async function handleDeleteProvider(row: ProviderSetting) {
  try {
    await ElMessageBox.confirm(
      `确定删除服务商 "${row.label}" 吗？该操作会级联清理服务商账号、模型映射与默认模型。`,
      '确认删除',
      { type: 'warning' }
    )
  } catch {
    return
  }

  submitting.value = true
  try {
    await providerApi.delete(row.id)

    const index = providerSettings.value.findIndex((provider) => provider.id === row.id)
    if (index > -1) {
      providerSettings.value.splice(index, 1)
    }

    await loadSettings()
    ElMessage.success('服务商已删除')
    eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  } finally {
    submitting.value = false
  }
}

function showEditDialog(provider: ProviderSetting) {
  editProvider.value = { ...provider, models: [...provider.models] }
  selectedModels.value = []
  selectAllModels.value = false
  batchModelsText.value = ''
  editDialogVisible.value = true
}

function handleSelectAllModels(val: boolean) {
  if (val && editProvider.value?.models) {
    selectedModels.value = [...editProvider.value.models]
  } else {
    selectedModels.value = []
  }
}

async function handleBatchAddModels() {
  if (!editProvider.value) return
  
  const models = batchModelsText.value
    .split('\n')
    .map(m => m.trim())
    .filter(m => m.length > 0)

  if (models.length === 0) {
    ElMessage.warning('请输入模型名称')
    return
  }

  submitting.value = true
  try {
    const newModels: string[] = []
    for (const model of models) {
      if (!editProvider.value!.models.includes(model)) {
        newModels.push(model)
        // Sync to backend
        await upsertModelRegistry(model, {
          provider: editProvider.value!.id,
          display_name: model,
          enabled: true
        })
      }
    }

    if (newModels.length > 0) {
      const idx = providerSettings.value.findIndex(p => p.id === editProvider.value!.id)
      if (idx > -1) {
        providerSettings.value[idx]!.models = [...providerSettings.value[idx]!.models, ...newModels]
        editProvider.value!.models = [...editProvider.value!.models, ...newModels]
      }
      batchModelsText.value = ''
      ElMessage.success(`成功添加 ${newModels.length} 个模型`)
      eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
    } else {
      ElMessage.info('所有模型已存在')
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '添加失败')
  } finally {
    submitting.value = false
  }
}

async function handleBatchDeleteModels() {
  if (selectedModels.value.length === 0) {
    ElMessage.warning('请选择要删除的模型')
    return
  }

  try {
    await ElMessageBox.confirm(`确定删除选中的 ${selectedModels.value.length} 个模型吗？`, '确认删除', { type: 'warning' })
  } catch {
    return
  }

  submitting.value = true
  try {
    const idx = providerSettings.value.findIndex(p => p.id === editProvider.value!.id)
    if (idx > -1) {
      const provider = providerSettings.value[idx]!
      
      // Sync deletions to backend first
      const deleteErrors: string[] = []
      for (const model of selectedModels.value) {
        try {
          await deleteModelRegistry(model)
        } catch (e: any) {
          deleteErrors.push(`${model}: ${e?.message || 'failed'}`)
        }
      }
      
      if (deleteErrors.length > 0) {
        ElMessage.error(`部分模型删除失败: ${deleteErrors.join(', ')}`)
        // Reload from backend to sync state
        await loadSettings()
        return
      }
      
      // Only update local state after successful backend delete
      provider.models = provider.models.filter(m => !selectedModels.value.includes(m))
      editProvider.value!.models = [...provider.models]

      selectedModels.value = []
      ElMessage.success('模型已删除')
      eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '删除失败')
  } finally {
    submitting.value = false
  }
  }

onMounted(() => {
  loadSettings()
  offModelsChanged = eventBus.on(DATA_EVENTS.MODELS_CHANGED, loadSettings)
})

onUnmounted(() => {
  if (offModelsChanged) {
    offModelsChanged()
    offModelsChanged = null
  }
})
</script>

<style scoped lang="scss">
.model-management-page {
  .source-context-bar {
    margin-bottom: 16px;
  }

  .context-bar-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    flex-wrap: wrap;
  }

  .context-bar-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .header-actions {
      display: flex;
      gap: 12px;
    }
  }

  .settings-info {
    margin-bottom: 20px;
  }

  .provider-cell {
    display: flex;
    align-items: center;
    gap: 12px;

    .provider-logo {
      width: 28px;
      height: 28px;
      border-radius: 6px;
      object-fit: contain;
    }

    .provider-icon {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-weight: 600;
      font-size: 14px;
    }

    .provider-name {
      font-weight: 500;
    }
  }

  .models-cell {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;

    .model-tag {
      margin: 2px;
    }
  }

  .call-modes {
    .mode-item {
      padding: 16px;
      border: 1px solid var(--el-border-color);
      border-radius: var(--el-border-radius-base);
      margin-bottom: 12px;

      .mode-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 8px;

        .mode-title {
          font-weight: 600;
          font-size: 15px;
        }

        .mode-desc {
          color: var(--el-text-color-secondary);
          font-size: 13px;
        }
      }
    }
  }

  .api-example {
    margin-top: 16px;
    border: 1px solid transparent;
    border-radius: var(--el-border-radius-base);
    transition: border-color 0.2s ease, box-shadow 0.2s ease;

    &.api-example--focus {
      border-color: var(--el-color-warning);
      box-shadow: 0 0 0 1px var(--el-color-warning-light-7) inset;
    }

    .example-title {
      font-weight: 600;
      margin-bottom: 12px;
    }
  }

  :deep(.provider-row--highlighted td.el-table__cell) {
    background: var(--el-color-primary-light-9) !important;
  }

  :deep(.default-model-select--focus .el-input__wrapper) {
    box-shadow: 0 0 0 1px var(--el-color-warning) inset;
  }

  .code {
    background: var(--el-fill-color-light);
    padding: 12px;
    border-radius: var(--el-border-radius-base);
    overflow-x: auto;

    code {
      font-family: monospace;
      font-size: 13px;
    }
  }
}

.edit-dialog-content {
  .provider-info {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
    padding: 12px;
    background: var(--el-fill-color-light);
    border-radius: 8px;

    .edit-provider-logo {
      width: 36px;
      height: 36px;
      border-radius: 6px;
      object-fit: contain;
    }

    .edit-provider-icon {
      width: 36px;
      height: 36px;
      border-radius: 6px;
      display: flex;
      align-items: center;
      justify-content: center;
      color: white;
      font-weight: 600;
      font-size: 14px;
    }

    .edit-provider-name {
      font-weight: 600;
      font-size: 16px;
    }
  }
  .models-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 8px;
    padding: 0 4px;

    .model-count {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .models-list {
    max-height: 200px;
    overflow-y: auto;
    padding: 8px;
    border: 1px solid var(--el-border-color);
    border-radius: 6px;
    margin-bottom: 12px;

    .model-item {
      padding: 6px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);
      &:last-child {
        border-bottom: none;
      }
    }
  }

  .batch-actions {
    text-align: right;
    margin-bottom: 16px;
  }
}

.logo-upload {
  display: flex;
  align-items: center;
  gap: 16px;

  .logo-preview {
    width: 64px;
    height: 64px;
    border: 2px dashed var(--el-border-color);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: border-color 0.2s;
    
    &:hover {
      border-color: var(--el-color-primary);
    }

    .preview-img {
      width: 48px;
      height: 48px;
      object-fit: contain;
    }

    .upload-icon {
      font-size: 24px;
      color: var(--el-text-color-placeholder);
    }
  }

  .upload-hint {
    font-size: 13px;
    color: var(--el-text-color-secondary);

    .hint-small {
      font-size: 12px;
      margin-top: 4px;
      color: var(--el-text-color-placeholder);
    }
  }
}
</style>

<style lang="scss">
.provider-select-dropdown {
  min-width: 200px !important;
  
  .el-select-dropdown__item {
    display: flex;
    align-items: center;
    padding: 6px 12px;
    min-height: 34px;
    
    .provider-option {
      display: flex;
      align-items: center;
      gap: 10px;
      width: 100%;
      white-space: nowrap;
      
      .provider-logo {
        height: 22px;
        width: auto;
        max-width: 80px;
        border-radius: 4px;
        object-fit: contain;
        flex-shrink: 0;
      }
      
      .dot {
        width: 10px;
        height: 10px;
        border-radius: 3px;
        flex-shrink: 0;
      }
    }
  }
}
</style>
