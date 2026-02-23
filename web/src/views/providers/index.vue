<template>
  <div class="providers-page">
    <el-card shadow="never" class="page-card">
      <!-- 工具栏 -->
      <div class="toolbar">
        <el-input
          v-model="searchText"
          placeholder="搜索服务商..."
          class="search-input"
          clearable
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        <el-button type="primary" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          添加服务商
        </el-button>
      </div>

      <!-- 服务商卡片列表 -->
      <el-row :gutter="20" class="provider-cards" v-if="filteredProviders.length > 0">
        <el-col :span="8" v-for="provider in filteredProviders" :key="provider.id">
          <el-card class="provider-card" shadow="hover" :class="{ disabled: !provider.enabled }">
            <div class="provider-header">
              <div class="provider-icon" :style="{ background: getProviderColor(provider.type) }">
                <el-icon :size="24"><component :is="getProviderIcon(provider.type)" /></el-icon>
              </div>
              <div class="provider-info">
                <h3 class="provider-name">{{ provider.name }}</h3>
                <el-tag size="small" :type="provider.enabled ? 'success' : 'info'">
                  {{ provider.enabled ? '已启用' : '已禁用' }}
                </el-tag>
              </div>
              <el-switch v-model="provider.enabled" @change="handleStatusChange(provider)" />
            </div>

            <div class="provider-stats">
              <div class="stat-item">
                <span class="stat-label">API端点</span>
                <span class="stat-value">{{ provider.endpoint }}</span>
              </div>
              <div class="stat-row">
                <div class="stat-item">
                  <span class="stat-label">关联账号</span>
                  <span class="stat-value">{{ provider.accounts }} 个</span>
                </div>
                <div class="stat-item">
                  <span class="stat-label">平均延迟</span>
                  <span class="stat-value latency" :class="getLatencyClass(provider.latency)">
                    {{ provider.latency }}
                  </span>
                </div>
              </div>
              <div class="stat-item">
                <span class="stat-label">支持模型</span>
                <div class="model-tags">
                  <el-tag v-for="model in provider.models.slice(0, 3)" :key="model" size="small" class="model-tag">
                    {{ model }}
                  </el-tag>
                  <el-tag v-if="provider.models.length > 3" size="small" type="info" class="model-tag">
                    +{{ provider.models.length - 3 }}
                  </el-tag>
                </div>
              </div>
            </div>

            <div class="provider-actions">
              <el-button size="small" @click="showEditDialog(provider)">
                <el-icon><Edit /></el-icon>
                编辑
              </el-button>
              <el-button size="small" type="primary" @click="handleTest(provider)" :loading="provider.testing">
                <el-icon><Connection /></el-icon>
                测试连接
              </el-button>
              <el-dropdown @command="(cmd: string) => handleCommand(cmd, provider)">
                <el-button size="small">
                  <el-icon><More /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item command="models">查看模型</el-dropdown-item>
                    <el-dropdown-item command="accounts">关联账号</el-dropdown-item>
                    <el-dropdown-item command="logs">请求日志</el-dropdown-item>
                    <el-dropdown-item divided command="delete">删除</el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </el-card>
        </el-col>
      </el-row>

      <!-- 空状态 -->
      <el-empty v-else description="未找到匹配的服务商" :image-size="120">
        <el-button type="primary" @click="searchText = ''">清除搜索</el-button>
      </el-empty>
    </el-card>

    <!-- 添加/编辑服务商对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑服务商' : '添加服务商'"
      width="600px"
      destroy-on-close
      class="provider-dialog"
    >
      <el-form :model="providerForm" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="服务商名称" prop="name">
          <el-input v-model="providerForm.name" placeholder="请输入服务商名称" />
        </el-form-item>
        <el-form-item label="服务商类型" prop="type">
          <el-select v-model="providerForm.type" placeholder="选择类型" style="width: 100%" @change="handleTypeChange">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Azure OpenAI" value="azure" />
            <el-option label="Anthropic" value="anthropic" />
            <el-option label="Google Gemini" value="google" />
            <el-option label="火山方舟 (字节跳动)" value="volcengine" />
            <el-option label="阿里云通义千问" value="qwen" />
            <el-option label="百度文心一言" value="ernie" />
            <el-option label="智谱AI" value="zhipu" />
            <el-option label="腾讯混元" value="hunyuan" />
            <el-option label="月之暗面" value="moonshot" />
            <el-option label="MiniMax" value="minimax" />
            <el-option label="百川智能" value="baichuan" />
            <el-option label="讯飞星火" value="spark" />
            <el-option label="DeepSeek" value="deepseek" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </el-form-item>
        <el-form-item label="API端点" prop="endpoint">
          <el-input v-model="providerForm.endpoint" placeholder="https://api.example.com/v1" />
        </el-form-item>
        <el-form-item label="API版本" v-if="providerForm.type === 'azure'">
          <el-input v-model="providerForm.apiVersion" placeholder="2024-02-15-preview" />
        </el-form-item>
        <el-form-item label="支持的模型">
          <el-select v-model="providerForm.models" multiple placeholder="选择支持的模型" style="width: 100%" filterable>
            <el-option v-for="model in availableModels" :key="model" :label="model" :value="model" />
          </el-select>
        </el-form-item>
        <el-form-item label="超时时间">
          <el-input-number v-model="providerForm.timeout" :min="1" :max="300" />
          <span class="form-hint">秒</span>
        </el-form-item>
        <el-form-item label="最大重试">
          <el-input-number v-model="providerForm.maxRetries" :min="0" :max="5" />
        </el-form-item>
        <el-form-item label="权重">
          <el-slider v-model="providerForm.weight" :min="1" :max="100" show-input />
        </el-form-item>
        <el-form-item label="健康检查">
          <el-switch v-model="providerForm.healthCheck" />
        </el-form-item>
        <el-form-item label="检查间隔" v-if="providerForm.healthCheck">
          <el-input-number v-model="providerForm.healthCheckInterval" :min="10" :max="300" />
          <span class="form-hint">秒</span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 测试结果对话框 -->
    <el-dialog v-model="testDialogVisible" title="连接测试" width="400px">
      <div class="test-result" :class="testResult.success ? 'success' : 'failed'">
        <el-icon :size="48">
          <CircleCheck v-if="testResult.success" />
          <CircleClose v-else />
        </el-icon>
        <h3>{{ testResult.success ? '连接成功' : '连接失败' }}</h3>
        <p v-if="testResult.message">{{ testResult.message }}</p>
        <div class="test-details" v-if="testResult.success">
          <div class="detail-item">
            <span>延迟:</span>
            <span>{{ testResult.latency }}ms</span>
          </div>
          <div class="detail-item">
            <span>可用模型:</span>
            <span>{{ testResult.models }} 个</span>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'
import { providerApi } from '@/api/provider'

const loading = ref(false)

interface Provider {
  id: number
  name: string
  type: string
  endpoint: string
  enabled: boolean
  accounts: number
  latency: string
  models: string[]
  testing?: boolean
}

const searchText = ref('')
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const testDialogVisible = ref(false)

const testResult = reactive({
  success: false,
  message: '',
  latency: 0,
  models: 0
})

const providerForm = reactive({
  id: 0,
  name: '',
  type: '',
  endpoint: '',
  apiVersion: '',
  models: [] as string[],
  timeout: 30,
  maxRetries: 3,
  weight: 50,
  healthCheck: true,
  healthCheckInterval: 30
})

const availableModels = [
  // OpenAI
  'gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo', 'o1', 'o1-mini', 'o1-preview',
  // Anthropic
  'claude-3-5-sonnet-20241022', 'claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307',
  // Google
  'gemini-2.0-flash-exp', 'gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-pro',
  // Azure OpenAI
  'gpt-4o', 'gpt-4', 'gpt-35-turbo',
  // 火山方舟 - 字节跳动
  'doubao-pro-256k', 'doubao-pro-128k', 'doubao-pro-32k', 'doubao-lite-128k', 'doubao-lite-32k',
  // 阿里云通义千问
  'qwen-max', 'qwen-max-longcontext', 'qwen-plus', 'qwen-turbo', 'qwen-long',
  // 百度文心一言
  'ernie-4.0-8k', 'ernie-4.0', 'ernie-3.5-8k', 'ernie-3.5', 'ernie-speed-8k', 'ernie-speed',
  // 智谱AI
  'glm-4-plus', 'glm-4-0520', 'glm-4-air', 'glm-4-airx', 'glm-4-long', 'glm-4-flash',
  // 腾讯混元
  'hunyuan-lite', 'hunyuan-standard', 'hunyuan-pro', 'hunyuan-turbo',
  // 月之暗面
  'moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k',
  // MiniMax
  'abab6.5-chat', 'abab6.5s-chat', 'abab5.5-chat', 'abab5.5s-chat',
  // 百川智能
  'Baichuan4', 'Baichuan3-Turbo', 'Baichuan3-Turbo-128k', 'Baichuan2-Turbo',
  // 讯飞星火
  'spark-v3.5', 'spark-v3.0', 'spark-v2.0', 'spark-v1.5',
  // 华为盘古
  'pangu-natural-language-10b', 'pangu-nlg-2b',
  // 商汤日日新
  'nova-ptc-xl-v1', 'nova-ptc-large-v1',
  // 360智脑
  '360gpt2-pro', '360gpt-turbo',
  // DeepSeek
  'deepseek-chat', 'deepseek-reasoner'
]

const formRules: FormRules = {
  name: [{ required: true, message: '请输入服务商名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择服务商类型', trigger: 'change' }],
  endpoint: [{ required: true, message: '请输入API端点', trigger: 'blur' }]
}

const providers = ref<Provider[]>([])

const filteredProviders = computed(() => {
  if (!searchText.value) return providers.value
  return providers.value.filter(p =>
    p.name.toLowerCase().includes(searchText.value.toLowerCase()) ||
    p.type.toLowerCase().includes(searchText.value.toLowerCase())
  )
})

const getProviderColor = (type: string) => {
  const colors: Record<string, string> = {
    openai: '#10A37F',
    azure: '#0078D4',
    anthropic: '#CC785C',
    google: '#4285F4',
    volcengine: '#FF4D4F',
    qwen: '#FF6A00',
    ernie: '#2932E1',
    zhipu: '#3657ED',
    hunyuan: '#00A3FF',
    moonshot: '#1A1A1A',
    minimax: '#615CED',
    baichuan: '#0066FF',
    spark: '#E60012',
    deepseek: '#4D6BFE',
    custom: '#8B5CF6'
  }
  return colors[type] || '#6B7280'
}

const getProviderIcon = (type: string) => {
  const icons: Record<string, string> = {
    openai: 'ChatDotRound',
    azure: 'Platform',
    anthropic: 'ChatLineRound',
    google: 'Star',
    volcengine: 'Lightning',
    qwen: 'Sunny',
    ernie: 'Reading',
    zhipu: 'MagicStick',
    hunyuan: 'Connection',
    moonshot: 'Moon',
    minimax: 'Cpu',
    baichuan: 'TrendCharts',
    spark: 'Promotion',
    deepseek: 'Search',
    custom: 'Setting'
  }
  return icons[type] || 'Connection'
}

const getLatencyClass = (latency: string) => {
  const ms = parseInt(latency)
  if (ms < 150) return 'fast'
  if (ms < 300) return 'normal'
  return 'slow'
}

const showAddDialog = () => {
  isEdit.value = false
  Object.assign(providerForm, {
    id: 0,
    name: '',
    type: '',
    endpoint: '',
    apiVersion: '',
    models: [],
    timeout: 30,
    maxRetries: 3,
    weight: 50,
    healthCheck: true,
    healthCheckInterval: 30
  })
  dialogVisible.value = true
}

const showEditDialog = (row: Provider) => {
  isEdit.value = true
  Object.assign(providerForm, {
    id: row.id,
    name: row.name,
    type: row.type,
    endpoint: row.endpoint,
    models: row.models,
    timeout: 30,
    maxRetries: 3,
    weight: 50,
    healthCheck: true,
    healthCheckInterval: 30
  })
  dialogVisible.value = true
}

const handleTypeChange = (type: string) => {
  const endpoints: Record<string, string> = {
    openai: 'https://api.openai.com/v1',
    azure: 'https://your-resource.openai.azure.com',
    anthropic: 'https://api.anthropic.com/v1',
    google: 'https://generativelanguage.googleapis.com/v1',
    volcengine: 'https://ark.cn-beijing.volces.com/api/v3',
    qwen: 'https://dashscope.aliyuncs.com/api/v1',
    ernie: 'https://aip.baidubce.com/rpc/2.0/ai_custom/v1',
    zhipu: 'https://open.bigmodel.cn/api/paas/v4',
    hunyuan: 'https://hunyuan.tencentcloudapi.com',
    moonshot: 'https://api.moonshot.cn/v1',
    minimax: 'https://api.minimax.chat/v1',
    baichuan: 'https://api.baichuan-ai.com/v1',
    spark: 'https://spark-api-open.xf-yun.com/v1',
    deepseek: 'https://api.deepseek.com/v1'
  }
  providerForm.endpoint = endpoints[type] || ''
}

const submitForm = async () => {
  if (!formRef.value) return
  try {
    const valid = await formRef.value.validate()
    if (valid) {
      if (isEdit.value) {
        await providerApi.update(String(providerForm.id), {
          name: providerForm.name,
          base_url: providerForm.endpoint,
          models: providerForm.models
        })
        const idx = providers.value.findIndex(p => p.id === providerForm.id)
        if (idx !== -1) {
          const existing = providers.value[idx]
          if (existing) {
            providers.value[idx] = {
              id: existing.id,
              name: providerForm.name,
              type: existing.type,
              endpoint: providerForm.endpoint,
              enabled: existing.enabled,
              accounts: existing.accounts,
              latency: existing.latency,
              models: providerForm.models
            }
          }
        }
        handleSuccess('服务商更新成功')
      } else {
        const res = await providerApi.create({
          name: providerForm.name,
          api_key: '',
          base_url: providerForm.endpoint,
          models: providerForm.models
        })
        const newProvider = (res as any)?.data || res
        providers.value.push({
          id: newProvider.id || Date.now(),
          name: providerForm.name,
          type: providerForm.type,
          endpoint: providerForm.endpoint,
          enabled: true,
          accounts: 0,
          latency: '0ms',
          models: providerForm.models
        })
        handleSuccess('服务商添加成功')
      }
      dialogVisible.value = false
    }
  } catch (error) {
    handleApiError(error, '操作失败，请重试')
  }
}

const handleTest = async (provider: Provider) => {
  provider.testing = true
  try {
    const res = await providerApi.testConnection(String(provider.id))
    const data = (res as any)?.data || res
    testResult.success = data.success !== false
    testResult.message = data.message || (testResult.success ? '' : '连接失败')
    testResult.latency = data.response_time_ms || data.latency || 0
    testResult.models = provider.models.length
    testDialogVisible.value = true
  } catch (error: any) {
    testResult.success = false
    testResult.message = error?.response?.data?.error || error?.message || '连接失败，请检查配置'
    testResult.latency = 0
    testResult.models = 0
    testDialogVisible.value = true
  } finally {
    provider.testing = false
  }
}

const handleCommand = (command: string, provider: Provider) => {
  switch (command) {
    case 'models':
      ElMessage.info(`查看 ${provider.name} 的模型`)
      break
    case 'accounts':
      ElMessage.info(`查看 ${provider.name} 的关联账号`)
      break
    case 'logs':
      ElMessage.info(`查看 ${provider.name} 的请求日志`)
      break
    case 'delete':
      handleDelete(provider)
      break
  }
}

const handleDelete = async (provider: Provider) => {
  try {
    await ElMessageBox.confirm(`确定删除服务商 ${provider.name} 吗？`, '提示', {
      type: 'warning'
    })
    await providerApi.delete(String(provider.id))
    providers.value = providers.value.filter(p => p.id !== provider.id)
    handleSuccess('删除成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      handleApiError(error, '删除失败，请重试')
    }
  }
}

const handleStatusChange = async (provider: Provider) => {
  try {
    await providerApi.toggleStatus(String(provider.id), provider.enabled)
    handleSuccess(`${provider.name} 已${provider.enabled ? '启用' : '禁用'}`)
  } catch (error) {
    provider.enabled = !provider.enabled
    handleApiError(error, '状态更新失败，请重试')
  }
}

const fetchProviders = async () => {
  loading.value = true
  try {
    const res = await providerApi.getList()
    const data = (res as any)?.data || res
    if (Array.isArray(data)) {
      providers.value = data.map((p: any) => ({
        id: p.id || p.name,
        name: p.name,
        type: p.type || 'custom',
        endpoint: p.base_url || p.endpoint || '',
        enabled: p.enabled ?? true,
        accounts: p.account_count || p.accounts || 0,
        latency: p.latency || '0ms',
        models: p.models || []
      }))
    }
  } catch (error) {
    handleApiError(error, '加载服务商列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchProviders()
})
</script>

<style scoped lang="scss">
.providers-page {
  .page-card {
    border-radius: var(--border-radius-lg);
    border: none;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--spacing-xl);

    .search-input {
      width: 280px;
    }
  }

  .provider-cards {
    .provider-card {
      border-radius: var(--border-radius-lg);
      border: none;
      margin-bottom: var(--spacing-xl);
      transition: all var(--transition-normal);

      &:hover {
        transform: translateY(-4px);
      }

      &.disabled {
        opacity: 0.6;
      }

      .provider-header {
        display: flex;
        align-items: center;
        gap: var(--spacing-md);
        margin-bottom: var(--spacing-lg);

        .provider-icon {
          width: 48px;
          height: 48px;
          border-radius: var(--border-radius-md);
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
        }

        .provider-info {
          flex: 1;

          .provider-name {
            margin: 0 0 4px 0;
            font-size: var(--font-size-lg);
            font-weight: var(--font-weight-semibold);
          }
        }
      }

      .provider-stats {
        margin-bottom: var(--spacing-lg);

        .stat-item {
          margin-bottom: var(--spacing-sm);

          .stat-label {
            font-size: var(--font-size-sm);
            color: var(--text-tertiary);
            display: block;
            margin-bottom: 4px;
          }

          .stat-value {
            font-weight: var(--font-weight-medium);

            &.latency {
              padding: 2px 8px;
              border-radius: var(--border-radius-sm);
              font-size: var(--font-size-sm);

              &.fast {
                background: rgba(52, 199, 89, 0.1);
                color: var(--color-success);
              }

              &.normal {
                background: rgba(255, 149, 0, 0.1);
                color: var(--color-warning);
              }

              &.slow {
                background: rgba(255, 59, 48, 0.1);
                color: var(--color-danger);
              }
            }
          }
        }

        .stat-row {
          display: flex;
          gap: var(--spacing-xl);

          .stat-item {
            flex: 1;
          }
        }

        .model-tags {
          display: flex;
          flex-wrap: wrap;
          gap: 4px;

          .model-tag {
            font-size: 11px;
          }
        }
      }

      .provider-actions {
        display: flex;
        gap: var(--spacing-sm);
        padding-top: var(--spacing-md);
        border-top: 1px solid var(--border-primary);
      }
    }
  }

  .form-hint {
    margin-left: var(--spacing-sm);
    color: var(--text-tertiary);
  }

  .test-result {
    text-align: center;
    padding: var(--spacing-xl);

    &.success .el-icon {
      color: var(--color-success);
    }

    &.failed .el-icon {
      color: var(--color-danger);
    }

    h3 {
      margin: var(--spacing-md) 0;
    }

    p {
      color: var(--text-secondary);
      margin: var(--spacing-sm) 0;
    }

    .test-details {
      background: var(--bg-secondary);
      padding: var(--spacing-md);
      border-radius: var(--border-radius-md);
      margin-top: var(--spacing-lg);

      .detail-item {
        display: flex;
        justify-content: space-between;
        padding: var(--spacing-sm) 0;

        &:not(:last-child) {
          border-bottom: 1px solid var(--border-primary);
        }
      }
    }
  }
}
</style>
