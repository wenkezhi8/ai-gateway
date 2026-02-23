<template>
  <div class="accounts-page">
    <el-card shadow="never" class="page-card">
      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-left">
          <el-select v-model="selectedProvider" placeholder="选择服务商" clearable class="provider-select">
            <el-option label="全部服务商" value="" />
            <el-option v-for="p in providers" :key="p" :label="p" :value="p" />
          </el-select>
          <el-input
            v-model="searchText"
            placeholder="搜索账号名称..."
            class="search-input"
            clearable
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
        </div>
        <el-button type="primary" @click="showAddDialog">
          <el-icon><Plus /></el-icon>
          添加账号
        </el-button>
      </div>

      <!-- 数据表格 -->
      <el-table :data="filteredAccounts" stripe class="data-table">
        <el-table-column prop="name" label="账号名称" min-width="150">
          <template #default="{ row }">
            <div class="account-name">
              <el-avatar :size="32" class="account-avatar">
                {{ row.name.charAt(0) }}
              </el-avatar>
              <span class="name-text">{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="provider" label="服务商" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="getProviderTagType(row.provider)">
              {{ row.provider }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="apiKey" label="API Key" min-width="200">
          <template #default="{ row }">
            <div class="api-key-cell">
              <code class="api-key">{{ maskApiKey(row.apiKey) }}</code>
              <el-button type="primary" link size="small" @click="copyApiKey(row.apiKey)">
                <el-icon><CopyDocument /></el-icon>
              </el-button>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="quota" label="配额使用" width="200">
          <template #default="{ row }">
            <div class="quota-cell">
              <el-progress
                :percentage="row.quotaUsed"
                :status="getQuotaStatus(row.quotaUsed)"
                :stroke-width="8"
              />
              <span class="quota-text">{{ row.quotaUsed }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" @change="handleStatusChange(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="expireAt" label="过期时间" width="120">
          <template #default="{ row }">
            <span :class="{ 'expiring-soon': isExpiringSoon(row.expireAt) }">
              {{ row.expireAt }}
            </span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="showEditDialog(row)">编辑</el-button>
            <el-button type="primary" link @click="showQuotaDialog(row)">配额</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :total="total"
          :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </el-card>

    <!-- 添加/编辑账号对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑账号' : '添加账号'"
      width="500px"
      destroy-on-close
    >
      <el-form :model="accountForm" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="账号名称" prop="name">
          <el-input v-model="accountForm.name" placeholder="请输入账号名称" />
        </el-form-item>
        <el-form-item label="服务商" prop="provider">
          <el-select v-model="accountForm.provider" placeholder="选择服务商" style="width: 100%">
            <el-option v-for="p in providers" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="API Key" prop="apiKey">
          <el-input v-model="accountForm.apiKey" placeholder="请输入API Key" show-password>
            <template #suffix>
              <el-tooltip content="API Key将被加密存储">
                <el-icon><Lock /></el-icon>
              </el-tooltip>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="配额限制">
          <el-input-number v-model="accountForm.quotaLimit" :min="0" :max="1000000" style="width: 100%">
            <template #suffix>美元</template>
          </el-input-number>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker
            v-model="accountForm.expireAt"
            type="date"
            placeholder="选择过期时间"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="accountForm.enabled" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="accountForm.remark" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <!-- 配额管理对话框 -->
    <el-dialog v-model="quotaDialogVisible" title="配额管理" width="400px">
      <div v-if="selectedAccount" class="quota-dialog-content">
        <div class="quota-info">
          <span class="label">账号：</span>
          <span class="value">{{ selectedAccount.name }}</span>
        </div>
        <div class="quota-info">
          <span class="label">已使用：</span>
          <span class="value">${{ selectedAccount.quotaUsedAmount || 0 }}</span>
        </div>
        <div class="quota-info">
          <span class="label">配额上限：</span>
          <el-input-number v-model="quotaForm.limit" :min="0" size="small" />
        </div>
        <el-divider />
        <el-form label-width="100px">
          <el-form-item label="重置配额">
            <el-button type="warning" size="small" @click="resetQuota">重置为0</el-button>
          </el-form-item>
          <el-form-item label="配额预警">
            <el-input-number v-model="quotaForm.warningThreshold" :min="50" :max="99" size="small" />
            <span class="form-hint">%触发预警</span>
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="quotaDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="saveQuota">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { accountApi } from '@/api/account'
import { handleApiError, handleSuccess } from '@/utils/errorHandler'

interface Account {
  id: number
  name: string
  provider: string
  apiKey: string
  quotaUsed: number
  quotaUsedAmount?: number
  enabled: boolean
  expireAt: string
  remark?: string
}

const searchText = ref('')
const selectedProvider = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const total = ref(100)
const loading = ref(false)

const providers = ['OpenAI', 'Azure', 'Anthropic', 'Google', '火山方舟', '阿里云通义千问', '百度文心一言', '智谱AI', '腾讯混元', '月之暗面', 'MiniMax', '百川智能', '讯飞星火', 'DeepSeek']

const accounts = ref<Account[]>([])

const filteredAccounts = computed(() => {
  let result = accounts.value
  if (selectedProvider.value) {
    result = result.filter(a => a.provider === selectedProvider.value)
  }
  if (searchText.value) {
    result = result.filter(a =>
      a.name.toLowerCase().includes(searchText.value.toLowerCase())
    )
  }
  return result
})

// 对话框相关
const dialogVisible = ref(false)
const isEdit = ref(false)
const formRef = ref<FormInstance>()
const selectedAccount = ref<Account | null>(null)
const quotaDialogVisible = ref(false)

const accountForm = reactive({
  id: 0,
  name: '',
  provider: '',
  apiKey: '',
  quotaLimit: 1000,
  expireAt: '',
  enabled: true,
  remark: ''
})

const quotaForm = reactive({
  limit: 1000,
  warningThreshold: 80
})

const formRules: FormRules = {
  name: [{ required: true, message: '请输入账号名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择服务商', trigger: 'change' }],
  apiKey: [{ required: true, message: '请输入API Key', trigger: 'blur' }]
}

const maskApiKey = (key: string) => {
  if (key.length <= 8) return '****'
  return key.substring(0, 7) + '****' + key.substring(key.length - 4)
}

const copyApiKey = async (key: string) => {
  try {
    await navigator.clipboard.writeText(key)
    ElMessage.success('已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const getProviderTagType = (provider: string) => {
  const types: Record<string, string> = {
    OpenAI: 'primary',
    Azure: 'info',
    Anthropic: 'success',
    Google: 'warning',
    '火山方舟': 'danger',
    '阿里云通义千问': 'warning',
    '百度文心一言': 'primary',
    '智谱AI': 'primary',
    '腾讯混元': 'info',
    '月之暗面': 'info',
    MiniMax: 'primary',
    '百川智能': 'primary',
    '讯飞星火': 'danger',
    DeepSeek: 'primary'
  }
  return types[provider] || ''
}

const getQuotaStatus = (percentage: number) => {
  if (percentage >= 90) return 'exception'
  if (percentage >= 70) return 'warning'
  return ''
}

const isExpiringSoon = (date: string) => {
  const expireDate = new Date(date)
  const now = new Date()
  const diffDays = Math.ceil((expireDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  return diffDays <= 30
}

const showAddDialog = () => {
  isEdit.value = false
  Object.assign(accountForm, {
    id: 0,
    name: '',
    provider: '',
    apiKey: '',
    quotaLimit: 1000,
    expireAt: '',
    enabled: true,
    remark: ''
  })
  dialogVisible.value = true
}

const showEditDialog = (row: Account) => {
  isEdit.value = true
  Object.assign(accountForm, {
    id: row.id,
    name: row.name,
    provider: row.provider,
    apiKey: row.apiKey,
    quotaLimit: 1000,
    expireAt: row.expireAt,
    enabled: row.enabled,
    remark: row.remark || ''
  })
  dialogVisible.value = true
}

const showQuotaDialog = (row: Account) => {
  selectedAccount.value = row
  quotaForm.limit = 1000
  quotaForm.warningThreshold = 80
  quotaDialogVisible.value = true
}

const submitForm = async () => {
  if (!formRef.value) return
  try {
    const valid = await formRef.value.validate()
    if (valid) {
      loading.value = true
      if (isEdit.value) {
        await accountApi.update(String(accountForm.id), {
          name: accountForm.name,
          provider: accountForm.provider,
          api_key: accountForm.apiKey,
          enabled: accountForm.enabled,
          remark: accountForm.remark
        })
        handleSuccess('账号更新成功')
      } else {
        await accountApi.create({
          name: accountForm.name,
          provider: accountForm.provider,
          api_key: accountForm.apiKey,
          enabled: accountForm.enabled,
          remark: accountForm.remark
        })
        handleSuccess('账号添加成功')
      }
      dialogVisible.value = false
      fetchAccounts()
    }
  } catch (error) {
    handleApiError(error, '操作失败')
  } finally {
    loading.value = false
  }
}

const saveQuota = async () => {
  if (!selectedAccount.value) return
  try {
    await accountApi.updateLimits(String(selectedAccount.value.id), {
      token: { type: 'token', period: 'month', limit: quotaForm.limit, warning: quotaForm.warningThreshold }
    })
    handleSuccess('配额设置已保存')
    quotaDialogVisible.value = false
    fetchAccounts()
  } catch (error) {
    handleApiError(error, '保存失败')
  }
}

const resetQuota = async () => {
  try {
    await ElMessageBox.confirm('确定要重置配额吗？', '提示', { type: 'warning' })
    if (selectedAccount.value) {
      await accountApi.updateLimits(String(selectedAccount.value.id), {
        token: { type: 'token', period: 'month', limit: quotaForm.limit, warning: quotaForm.warningThreshold }
      })
      handleSuccess('配额已重置')
      fetchAccounts()
    }
  } catch (error) {
    if ((error as any) !== 'cancel') {
      handleApiError(error, '重置失败')
    }
  }
}

const handleDelete = async (row: Account) => {
  try {
    await ElMessageBox.confirm(`确定删除账号 ${row.name} 吗？`, '提示', { type: 'warning' })
    await accountApi.delete(String(row.id))
    handleSuccess('删除成功')
    fetchAccounts()
  } catch (error) {
    if ((error as any) !== 'cancel') {
      handleApiError(error, '删除失败')
    }
  }
}

const handleStatusChange = async (row: Account) => {
  try {
    await accountApi.toggleStatus(String(row.id), row.enabled)
    handleSuccess(`${row.name} 已${row.enabled ? '启用' : '禁用'}`)
  } catch (error) {
    row.enabled = !row.enabled
    handleApiError(error, '状态更新失败')
  }
}

const fetchAccounts = async () => {
  loading.value = true
  try {
    const res = await accountApi.getList({ page: currentPage.value, pageSize: pageSize.value })
    const data = (res as any)?.data || res
    if (Array.isArray(data)) {
      accounts.value = data.map((a: any) => ({
        id: a.id,
        name: a.name,
        provider: a.provider,
        apiKey: a.api_key || '',
        quotaUsed: a.usage?.token?.percent_used || 0,
        quotaUsedAmount: a.usage?.token?.used || 0,
        enabled: a.enabled ?? true,
        expireAt: a.usage?.month?.reset_at || '',
        remark: a.remark || ''
      }))
      total.value = data.length
    }
  } catch (error) {
    handleApiError(error, '加载账号列表失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchAccounts()
})
</script>

<style scoped lang="scss">
.accounts-page {
  .page-card {
    border-radius: var(--border-radius-lg);
    border: none;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--spacing-xl);

    .toolbar-left {
      display: flex;
      align-items: center;
      gap: var(--spacing-md);
    }

    .provider-select {
      width: 150px;
    }

    .search-input {
      width: 240px;
    }
  }

  .data-table {
    .account-name {
      display: flex;
      align-items: center;
      gap: var(--spacing-md);

      .account-avatar {
        background: var(--color-primary);
        color: white;
        font-weight: var(--font-weight-semibold);
      }

      .name-text {
        font-weight: var(--font-weight-medium);
      }
    }

    .api-key-cell {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);

      .api-key {
        font-family: var(--font-family-mono);
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
        background: var(--bg-tertiary);
        padding: 2px 8px;
        border-radius: var(--border-radius-sm);
      }
    }

    .quota-cell {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);

      .el-progress {
        flex: 1;
      }

      .quota-text {
        width: 40px;
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
      }
    }

    .expiring-soon {
      color: var(--color-warning);
      font-weight: var(--font-weight-medium);
    }
  }

  .pagination {
    margin-top: var(--spacing-xl);
    display: flex;
    justify-content: flex-end;
  }

  .quota-dialog-content {
    .quota-info {
      display: flex;
      align-items: center;
      margin-bottom: var(--spacing-md);

      .label {
        width: 80px;
        color: var(--text-secondary);
      }

      .value {
        font-weight: var(--font-weight-medium);
      }
    }

    .form-hint {
      margin-left: var(--spacing-sm);
      color: var(--text-tertiary);
    }
  }
}
</style>
