<template>
  <div class="accounts-page">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon total">
              <el-icon><Key /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.total }}</div>
              <div class="stat-label">账号总数</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon enabled">
              <el-icon><CircleCheck /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.enabled }}</div>
              <div class="stat-label">已启用</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon disabled">
              <el-icon><CircleClose /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.disabled }}</div>
              <div class="stat-label">已禁用</div>
            </div>
          </div>
        </el-card>
      </el-col>
      <el-col :xs="24" :sm="12" :md="6">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon providers">
              <el-icon><Grid /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ stats.providers }}</div>
              <div class="stat-label">服务商数</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 主内容卡片 -->
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <h2>账号管理</h2>
            <span class="subtitle">管理所有 AI 服务商账号</span>
          </div>
          <el-button type="primary" @click="showAddAccountDialog">
            <el-icon><Plus /></el-icon>
            添加账号
          </el-button>
        </div>
      </template>

      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <el-select v-model="selectedProviderFilter" placeholder="筛选服务商" clearable class="provider-select">
            <el-option label="全部服务商" value="" />
            <el-option v-for="p in providerTypes" :key="p.value" :label="p.label" :value="p.value" />
          </el-select>
          <el-input
            v-model="accountSearch"
            placeholder="搜索账号名称..."
            class="search-input"
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
        <div class="toolbar-right">
          <el-button @click="loadAccounts" :loading="accountsLoading">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </div>
      </div>

      <!-- 账号表格 -->
      <el-table 
        :data="filteredAccounts" 
        class="data-table" 
        v-loading="accountsLoading"
        :header-cell-style="{ background: 'var(--bg-secondary)', fontWeight: '600' }"
      >
        <el-table-column prop="name" label="账号名称" min-width="180">
          <template #default="{ row }">
            <div class="account-name">
              <img v-if="providerLogos[detectProvider(row)]" :src="providerLogos[detectProvider(row)]" class="account-logo" />
              <div v-else class="account-avatar" :style="{ background: getProviderColor(row) }">
                <span class="avatar-text">{{ getProviderLabel(row).charAt(0) }}</span>
              </div>
              <div class="account-info">
                <span class="name-text">{{ row.name }}</span>
                <span class="account-id">{{ row.id }}</span>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="provider" label="服务商" width="150">
          <template #default="{ row }">
            <div class="provider-badge" :style="{ '--provider-color': getProviderColor(row) }">
              <img v-if="providerLogos[detectProvider(row)]" :src="providerLogos[detectProvider(row)]" class="provider-logo-small" />
              <span v-else class="provider-dot"></span>
              <span class="provider-name">{{ getProviderLabel(row) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="api_key" label="API Key" min-width="180">
          <template #default="{ row }">
            <div class="api-key-cell">
              <code class="api-key">{{ maskApiKey(row.api_key) }}</code>
              <el-button type="primary" link size="small" @click="copyApiKey(row.api_key)" class="copy-btn">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="base_url" label="API 端点" min-width="220">
          <template #default="{ row }">
            <div class="endpoint-cell">
              <el-icon class="endpoint-icon"><Link /></el-icon>
              <span class="endpoint-text">{{ row.base_url || getDefaultEndpoint(row.provider) }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="enabled" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.enabled ? 'success' : 'info'" size="small" effect="light" class="status-tag">
              {{ row.enabled ? '已启用' : '已禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="AI 编程订阅" width="110" align="center">
          <template #default="{ row }">
            <el-switch 
              v-model="row.coding_plan_enabled" 
              size="small"
              :disabled="!row.enabled"
              @change="handleCodingPlanChange(row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑账号" placement="top">
                <el-button type="primary" link size="small" @click="showEditAccountDialog(row)">
                  <el-icon><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="从服务商获取模型列表" placement="top">
                <el-button 
                  type="success" 
                  link 
                  size="small" 
                  @click="handleFetchModels(row)" 
                  :loading="row.fetchingModels"
                >
                  <el-icon><Download /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除账号" placement="top">
                <el-button type="danger" link size="small" @click="handleDeleteAccount(row)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>

      <el-empty v-if="!accountsLoading && filteredAccounts.length === 0" description="暂无账号数据" :image-size="120">
        <template #description>
          <p class="empty-text">还没有添加任何账号</p>
          <p class="empty-hint">点击上方"添加账号"按钮开始配置</p>
        </template>
        <el-button type="primary" @click="showAddAccountDialog">
          <el-icon><Plus /></el-icon>
          添加第一个账号
        </el-button>
      </el-empty>
    </el-card>

    <!-- 添加/编辑账号对话框 -->
    <el-dialog
      v-model="accountDialogVisible"
      :title="isEditAccount ? '编辑账号' : '添加账号'"
      width="520px"
      destroy-on-close
      class="account-dialog"
    >
      <el-form :model="accountForm" :rules="accountRules" ref="accountFormRef" label-width="100px" class="account-form">
        <el-form-item label="账号名称" prop="name">
          <el-input v-model="accountForm.name" placeholder="例如：DeepSeek 主账号">
            <template #prefix>
              <el-icon><User /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="服务商" prop="provider">
          <el-select v-model="accountForm.provider" placeholder="选择服务商" style="width: 100%" @change="handleProviderChange" popper-class="provider-select-dropdown">
            <el-option-group label="国际服务商">
              <el-option v-for="p in internationalProviders" :key="p.value" :label="p.label" :value="p.value">
                <span class="provider-option">
                  <img v-if="providerLogos[p.value]" :src="providerLogos[p.value]" class="provider-logo" />
                  <span v-else class="dot" :style="{ background: providerColors[p.value] }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
            <el-option-group label="国内服务商">
              <el-option v-for="p in chineseProviders" :key="p.value" :label="p.label" :value="p.value">
                <span class="provider-option">
                  <img v-if="providerLogos[p.value]" :src="providerLogos[p.value]" class="provider-logo" />
                  <span v-else class="dot" :style="{ background: providerColors[p.value] }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
            <el-option-group label="本地大模型">
              <el-option v-for="p in localProviders" :key="p.value" :label="p.label" :value="p.value">
                <span class="provider-option">
                  <img v-if="providerLogos[p.value]" :src="providerLogos[p.value]" class="provider-logo" />
                  <span v-else class="dot" :style="{ background: providerColors[p.value] }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
            <el-option-group v-if="customProviders.length > 0" label="自定义服务商">
              <el-option v-for="p in customProviders" :key="p.value" :label="p.label" :value="p.value">
                <span class="provider-option">
                  <span class="dot" :style="{ background: providerColors[p.value] || '#6B7280' }"></span>
                  {{ p.label }}
                </span>
              </el-option>
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="API Key" prop="api_key">
          <el-input 
            v-model="accountForm.api_key" 
            :placeholder="isEditAccount ? '留空则保持原值不变' : '输入 API Key'" 
            show-password
          >
            <template #prefix>
              <el-icon><Key /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="API 端点" prop="base_url">
          <el-input v-model="accountForm.base_url" placeholder="https://api.example.com/v1">
            <template #prefix>
              <el-icon><Link /></el-icon>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="accountForm.enabled" active-text="启用" inactive-text="禁用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitAccountForm" :loading="accountSubmitting">
          {{ isEditAccount ? '保存修改' : '添加账号' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { 
  Plus, Search, Refresh, CopyDocument, Edit, Delete, Download, 
  Key, CircleCheck, CircleClose, Grid, User, Link 
} from '@element-plus/icons-vue'
import { accountApi } from '@/api/account'
import { request } from '@/api/request'
import { eventBus, DATA_EVENTS } from '@/utils/eventBus'
import {
  PROVIDERS_ACCOUNTS_BASE_TYPES,
  INTERNATIONAL_PROVIDER_SET,
  CHINESE_PROVIDER_SET,
  LOCAL_PROVIDER_SET,
  PROVIDERS_ACCOUNTS_DEFAULT_ENDPOINTS,
  PROVIDERS_ACCOUNTS_CODING_PLAN_ENDPOINTS,
  PROVIDERS_ACCOUNTS_PROVIDER_COLORS,
  PROVIDERS_ACCOUNTS_PROVIDER_LOGOS,
} from '@/constants/pages/providers-accounts'

interface Account {
  id: string
  name: string
  provider: string
  api_key?: string
  base_url?: string
  enabled: boolean
  priority?: number
  fetchingModels?: boolean
  coding_plan_enabled?: boolean
}

const accountSearch = ref('')
const selectedProviderFilter = ref('')

const accountsLoading = ref(false)
const accountSubmitting = ref(false)

const baseProviderTypes = [...PROVIDERS_ACCOUNTS_BASE_TYPES]

const providerTypes = ref([...baseProviderTypes])

const internationalProviderSet = INTERNATIONAL_PROVIDER_SET
const chineseProviderSet = CHINESE_PROVIDER_SET
const localProviderSet = LOCAL_PROVIDER_SET

const internationalProviders = computed(() => providerTypes.value.filter(p => internationalProviderSet.has(p.value)))
const chineseProviders = computed(() => providerTypes.value.filter(p => chineseProviderSet.has(p.value)))
const localProviders = computed(() => providerTypes.value.filter(p => localProviderSet.has(p.value)))
const customProviders = computed(() => providerTypes.value.filter(p => !internationalProviderSet.has(p.value) && !chineseProviderSet.has(p.value) && !localProviderSet.has(p.value)))

const defaultEndpoints: Record<string, string> = { ...PROVIDERS_ACCOUNTS_DEFAULT_ENDPOINTS }

// AI 编程订阅 (Coding Plan) 专用端点
const codingPlanEndpoints: Record<string, string> = { ...PROVIDERS_ACCOUNTS_CODING_PLAN_ENDPOINTS }

const providerColors: Record<string, string> = { ...PROVIDERS_ACCOUNTS_PROVIDER_COLORS }

const providerLogos: Record<string, string> = { ...PROVIDERS_ACCOUNTS_PROVIDER_LOGOS }

const accounts = ref<Account[]>([])

const stats = computed(() => {
  const total = accounts.value.length
  const enabled = accounts.value.filter(a => a.enabled).length
  const disabled = total - enabled
  const providers = new Set(accounts.value.map(a => a.provider)).size
  return { total, enabled, disabled, providers }
})

const filteredAccounts = computed(() => {
  let result = accounts.value
  if (selectedProviderFilter.value) {
    result = result.filter(a => a.provider === selectedProviderFilter.value)
  }
  if (accountSearch.value) {
    const query = accountSearch.value.toLowerCase()
    result = result.filter(a => a.name.toLowerCase().includes(query))
  }
  return result.sort((a, b) => {
    if (a.provider !== b.provider) return a.provider.localeCompare(b.provider)
    return a.name.localeCompare(b.name)
  })
})

const accountDialogVisible = ref(false)
const isEditAccount = ref(false)
const accountFormRef = ref<FormInstance>()

const accountForm = reactive({
  id: '', name: '', provider: '', api_key: '', base_url: '', enabled: true
})

const accountRules: FormRules = {
  name: [{ required: true, message: '请输入账号名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择服务商', trigger: 'change' }],
  api_key: [{ 
    validator: (_rule, value, callback) => {
      if (!isEditAccount.value && !value) {
        callback(new Error('请输入API Key'))
      } else {
        callback()
      }
    },
    trigger: 'blur'
  }]
}

const detectProvider = (row: { provider: string; base_url: string }): string => {
  const url = row.base_url || ''
  if (url.includes('deepseek.com')) return 'deepseek'
  if (url.includes('openai.com')) return 'openai'
  if (url.includes('anthropic.com')) return 'anthropic'
  if (url.includes('volces.com') || url.includes('volcengine')) return 'volcengine'
  if (url.includes('dashscope.aliyuncs.com') || url.includes('aliyun')) return 'qwen'
  if (url.includes('zhipuai.cn') || url.includes('bigmodel.cn')) return 'zhipu'
  if (url.includes('moonshot.cn') || url.includes('kimi.ai')) return 'moonshot'
  if (url.includes('minimax')) return 'minimax'
  if (url.includes('baichuan')) return 'baichuan'
  if (url.includes('googleapis.com')) return 'google'
  return row.provider
}

const getProviderColor = (row: { provider: string; base_url: string }) => {
  const actualProvider = detectProvider(row)
  return providerColors[actualProvider] || '#6B7280'
}

const getProviderLabel = (row: { provider: string; base_url: string }) => {
  const actualProvider = detectProvider(row)
  return providerTypes.value.find(p => p.value === actualProvider)?.label || actualProvider
}

const getDefaultEndpoint = (provider: string) => defaultEndpoints[provider] || ''

const maskApiKey = (key?: string) => {
  if (!key) return '未设置'
  if (key.length <= 8) return '****'
  const prefix = key.startsWith('sk-') ? key.substring(0, 7) : key.substring(0, 4)
  const suffix = key.substring(key.length - 4)
  return `${prefix}...${suffix}`
}

const copyApiKey = async (key?: string) => {
  if (!key) {
    ElMessage.warning('API Key 未设置')
    return
  }
  await navigator.clipboard.writeText(key)
  ElMessage.success('已复制到剪贴板')
}

const loadAccounts = async () => {
  accountsLoading.value = true
  try {
    const res = await accountApi.getList()
    accounts.value = (res as any).data || []
  } catch (e: any) {
    console.error('Failed to load accounts:', e)
    if (e?.response?.status !== 401) {
      ElMessage.error('加载账号列表失败')
    }
  } finally {
    accountsLoading.value = false
  }
}

const loadProviderOptions = async () => {
  const providerMap = new Map<string, { label: string; value: string }>()
  for (const p of baseProviderTypes) {
    providerMap.set(p.value, p)
  }

  try {
    const modelsRes: any = await accountApi.getList({ page: 1, pageSize: 200 })
    const accountList = (modelsRes as any)?.data || []
    for (const account of accountList) {
      const provider = String(account?.provider || '').trim().toLowerCase()
      if (!provider) continue
      if (!providerMap.has(provider)) {
        providerMap.set(provider, { value: provider, label: provider })
      }
    }
  } catch {
    // ignore and continue with other sources
  }

  try {
    const routerModelsRes: any = await request.get('/admin/router/models', { silent: true } as any)
    const raw = routerModelsRes?.data || routerModelsRes || []
    if (Array.isArray(raw)) {
      for (const row of raw) {
        const provider = String(row?.provider || '').trim().toLowerCase()
        if (!provider) continue
        if (!providerMap.has(provider)) {
          providerMap.set(provider, { value: provider, label: provider })
        }
      }
    } else if (raw && typeof raw === 'object') {
      for (const provider of Object.keys(raw)) {
        const id = String(provider || '').trim().toLowerCase()
        if (!id) continue
        if (!providerMap.has(id)) {
          providerMap.set(id, { value: id, label: id })
        }
      }
    }
  } catch {
    // ignore if router models unavailable
  }

  providerTypes.value = Array.from(providerMap.values())
}

const handleProviderChange = (provider: string) => {
  // 每次选择服务商时自动更新对应的 API 端点
  accountForm.base_url = defaultEndpoints[provider] || ''
}

const handleCodingPlanChange = async (row: Account) => {
  try {
    const defaultEndpoint = defaultEndpoints[row.provider] || ''
    const codingPlanEndpoint = codingPlanEndpoints[row.provider] || defaultEndpoint
    
    // 开启用 coding plan 端点，关闭恢复默认端点
    const newEndpoint = row.coding_plan_enabled ? codingPlanEndpoint : defaultEndpoint
    
    await accountApi.update(row.id, { 
      coding_plan_enabled: row.coding_plan_enabled,
      base_url: newEndpoint
    })
    
    // 更新本地数据
    row.base_url = newEndpoint
    
    ElMessage.success(row.coding_plan_enabled ? '已开启 AI 编程订阅' : '已关闭 AI 编程订阅')
  } catch (e: any) {
    row.coding_plan_enabled = !row.coding_plan_enabled
    ElMessage.error(e.message || '操作失败')
  }
}

const showAddAccountDialog = () => {
  isEditAccount.value = false
  Object.assign(accountForm, { id: '', name: '', provider: '', api_key: '', base_url: '', enabled: true })
  accountDialogVisible.value = true
}

const showEditAccountDialog = (row: Account) => {
  isEditAccount.value = true
  Object.assign(accountForm, {
    id: row.id,
    name: row.name,
    provider: row.provider,
    api_key: '',
    base_url: row.base_url || '',
    enabled: row.enabled
  })
  accountDialogVisible.value = true
}

const submitAccountForm = async () => {
  if (!accountFormRef.value) return
  const valid = await accountFormRef.value.validate()
  if (!valid) return

  accountSubmitting.value = true
  try {
    if (isEditAccount.value) {
      await accountApi.update(accountForm.id, {
        name: accountForm.name,
        api_key: accountForm.api_key || undefined,
        base_url: accountForm.base_url,
        enabled: accountForm.enabled
      })
      ElMessage.success('账号更新成功')
    } else {
      await accountApi.create({
        name: accountForm.name,
        provider: accountForm.provider,
        api_key: accountForm.api_key,
        base_url: accountForm.base_url,
        enabled: accountForm.enabled
      })
      ElMessage.success('账号添加成功')
    }
    accountDialogVisible.value = false
    loadAccounts()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error?.message || e.message || '操作失败')
  } finally {
    accountSubmitting.value = false
  }
}

const handleDeleteAccount = async (row: Account) => {
  try {
    await ElMessageBox.confirm(`确定删除账号「${row.name}」吗？`, '删除确认', { type: 'warning' })
    await accountApi.delete(row.id)
    ElMessage.success('删除成功')
    loadAccounts()
  } catch (e: any) {
    if (e !== 'cancel') {
      ElMessage.error(e.message || '删除失败')
    }
  }
}

const handleFetchModels = async (row: Account) => {
  row.fetchingModels = true
  try {
    // 默认同步到模型管理，避免“获取了但未保存”的误解
    const res = await accountApi.fetchModels(row.id, true)
    if (res.data?.models) {
      const models = res.data.models
      const syncedCount = (res as any).data?.synced_count ?? models.length
      ElMessage.success(`获取到 ${models.length} 个模型，已同步 ${syncedCount} 个`) 
      eventBus.emit(DATA_EVENTS.MODELS_CHANGED)
    } else {
      ElMessage.warning('未获取到模型列表')
    }
  } catch (e: any) {
    ElMessage.error(e.message || '获取模型失败')
  } finally {
    row.fetchingModels = false
  }
}

onMounted(() => {
  loadProviderOptions()
  loadAccounts()
})
</script>

<style scoped lang="scss">
.accounts-page {
  .stats-row {
    margin-bottom: 20px;
  }

  .stat-card {
    border: none;
    border-radius: 12px;
    margin-bottom: 10px;

    .stat-content {
      display: flex;
      align-items: center;
      gap: 16px;
    }

    .stat-icon {
      width: 48px;
      height: 48px;
      border-radius: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 24px;
      color: white;
    }

    .stat-icon.total { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
    .stat-icon.enabled { background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%); }
    .stat-icon.disabled { background: linear-gradient(135deg, #eb3349 0%, #f45c43 100%); }
    .stat-icon.providers { background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); }

    .stat-info {
      flex: 1;
    }

    .stat-value {
      font-size: 28px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }

    .stat-label {
      font-size: 14px;
      color: var(--el-text-color-secondary);
      margin-top: 2px;
    }
  }

  .page-card {
    border-radius: 12px;
    border: none;

    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;

      .header-left {
        h2 {
          margin: 0;
          font-size: 18px;
          font-weight: 600;
        }
        .subtitle {
          font-size: 13px;
          color: var(--el-text-color-secondary);
          margin-top: 4px;
          display: block;
        }
      }
    }

    .toolbar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;

      .toolbar-left {
        display: flex;
        align-items: center;
        gap: 12px;
      }

      .search-input { width: 240px; }
      .provider-select { width: 160px; }
    }
  }

  .data-table {
    .account-name {
      display: flex;
      align-items: center;
      gap: 12px;

      .account-logo {
        width: 32px;
        height: 32px;
        border-radius: 6px;
        object-fit: contain;
        flex-shrink: 0;
      }

      .account-avatar {
        width: 40px;
        height: 40px;
        border-radius: 10px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;

        .avatar-text {
          color: white;
          font-weight: 600;
          font-size: 16px;
        }
      }

      .account-info {
        display: flex;
        flex-direction: column;
        gap: 2px;

        .name-text {
          font-weight: 500;
          color: var(--el-text-color-primary);
        }

        .account-id {
          font-size: 12px;
          color: var(--el-text-color-placeholder);
        }
      }
    }

    .provider-badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 10px;
      background: color-mix(in srgb, var(--provider-color) 10%, transparent);
      border-radius: 6px;

      .provider-logo-small {
        width: 14px;
        height: 14px;
        border-radius: 2px;
        object-fit: contain;
      }

      .provider-dot {
        width: 8px;
        height: 8px;
        border-radius: 2px;
        background: var(--provider-color);
      }

      .provider-name {
        font-size: 13px;
        font-weight: 500;
        color: var(--provider-color);
      }
    }

    .api-key-cell {
      display: flex;
      align-items: center;
      gap: 8px;

      .api-key {
        font-family: 'SF Mono', Monaco, 'Courier New', monospace;
        font-size: 12px;
        color: var(--el-text-color-secondary);
        background: var(--el-fill-color-light);
        padding: 4px 10px;
        border-radius: 6px;
      }

      .copy-btn {
        opacity: 0;
        transition: opacity 0.2s;
      }
    }

    .endpoint-cell {
      display: flex;
      align-items: center;
      gap: 6px;

      .endpoint-icon {
        color: var(--el-text-color-placeholder);
        flex-shrink: 0;
      }

      .endpoint-text {
        font-size: 13px;
        color: var(--el-text-color-secondary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .status-tag {
      min-width: 60px;
      justify-content: center;
    }

    .action-buttons {
      display: flex;
      justify-content: center;
      gap: 4px;
    }

    tr:hover {
      .copy-btn {
        opacity: 1;
      }
    }
  }

  .empty-text {
    font-size: 15px;
    color: var(--el-text-color-primary);
    margin-bottom: 4px;
  }

  .empty-hint {
    font-size: 13px;
    color: var(--el-text-color-secondary);
    margin-bottom: 16px;
  }

  .provider-option {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;

    .provider-logo {
      width: 14px;
      height: 14px;
      border-radius: 2px;
      object-fit: contain;
      flex-shrink: 0;
    }

    .dot {
      width: 8px;
      height: 8px;
      border-radius: 2px;
      flex-shrink: 0;
    }
  }

  .account-form {
    padding: 10px 20px 0;
  }
}

@media (max-width: 768px) {
  .accounts-page {
    .toolbar {
      flex-direction: column;
      align-items: stretch;
      gap: 12px;

      .toolbar-left {
        flex-wrap: wrap;
      }

      .search-input,
      .provider-select {
        width: 100%;
      }
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
