<template>
  <div class="api-management-page">
    <el-row :gutter="24">
      <el-col :span="16">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>统一 API 入口</span>
              <el-tag type="success">在线</el-tag>
            </div>
          </template>

          <div class="api-section">
            <div class="section-title">API 地址</div>
            <div class="url-box">
              <code class="api-url">{{ apiBaseUrl }}/api/v1/chat/completions</code>
              <el-button type="primary" size="small" @click="copyUrl">
                <el-icon><CopyDocument /></el-icon>
                复制
              </el-button>
            </div>
          </div>

          <div class="api-section">
            <div class="section-header">
              <div class="section-title">API Key 管理</div>
              <el-button type="primary" size="small" @click="showCreateKeyDialog">
                <el-icon><Plus /></el-icon>
                创建 API Key
              </el-button>
            </div>
            
            <el-table :data="apiKeys" stripe size="small" v-loading="loadingKeys">
              <el-table-column label="名称" prop="name" width="150" />
              <el-table-column label="API Key" min-width="280">
                <template #default="{ row }">
                  <div class="key-cell">
                    <code class="key-value">{{ row.visible ? row.key : maskKey(row.key) }}</code>
                    <el-button type="primary" link size="small" @click="toggleKeyVisibility(row)">
                      <el-icon><View v-if="!row.visible" /><Hide v-else /></el-icon>
                    </el-button>
                    <el-button type="primary" link size="small" @click="copyKey(row.key)">
                      <el-icon><CopyDocument /></el-icon>
                    </el-button>
                  </div>
                </template>
              </el-table-column>
              <el-table-column label="创建时间" width="160">
                <template #default="{ row }">
                  {{ formatDate(row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column label="最后使用" width="160">
                <template #default="{ row }">
                  {{ row.last_used ? formatDate(row.last_used) : '从未使用' }}
                </template>
              </el-table-column>
              <el-table-column label="状态" width="80">
                <template #default="{ row }">
                  <el-tag :type="row.enabled ? 'success' : 'info'" size="small">
                    {{ row.enabled ? '启用' : '禁用' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="120" fixed="right">
                <template #default="{ row }">
                  <el-button type="primary" link size="small" @click="toggleKeyStatus(row)">
                    {{ row.enabled ? '禁用' : '启用' }}
                  </el-button>
                  <el-button type="danger" link size="small" @click="deleteKey(row)">
                    删除
                  </el-button>
                </template>
              </el-table-column>
            </el-table>
            
            <el-empty v-if="!loadingKeys && apiKeys.length === 0" description="暂无 API Key，点击上方按钮创建" :image-size="60" />
          </div>

          <div class="api-section">
            <div class="section-inline">
              <div class="section-title">默认模型设置</div>
              <el-form :model="routerConfig" label-width="80px" class="config-form-inline">
                <el-form-item label="调用模式">
                  <el-radio-group v-model="routerConfig.use_auto_mode" @change="updateRouterConfig">
                    <el-radio-button value="auto">Auto 智能选择</el-radio-button>
                    <el-radio-button value="default">Default 默认</el-radio-button>
                    <el-radio-button value="fixed">固定模型</el-radio-button>
                    <el-radio-button value="latest">Latest 最新</el-radio-button>
                  </el-radio-group>
                </el-form-item>
              </el-form>
            </div>
            
            <el-form :model="routerConfig" label-width="100px" class="config-form">
              <el-form-item v-if="routerConfig.use_auto_mode === 'auto'" label="智能策略">
                <el-select v-model="routerConfig.default_strategy" @change="updateRouterConfig" style="width: 300px">
                  <el-option
                    v-for="s in strategies"
                    :key="s.value"
                    :label="s.label"
                    :value="s.value"
                  >
                    <div class="strategy-option">
                      <span>{{ s.label }}</span>
                      <span class="strategy-desc">{{ s.description }}</span>
                    </div>
                  </el-option>
                </el-select>
              </el-form-item>
              
              <el-form-item v-if="routerConfig.use_auto_mode === 'default'">
                <div class="default-hint">
                  <el-icon><InfoFilled /></el-icon>
                  <span>使用服务商配置的默认模型，不同服务商可设置不同的默认模型</span>
                </div>
                <div class="default-providers" v-if="providerDefaults.length > 0">
                  <div class="default-item" v-for="p in providerDefaults" :key="p.id">
                    <span class="provider-name">{{ p.label }}</span>
                    <el-tag size="small">{{ p.defaultModel || '未设置' }}</el-tag>
                  </div>
                </div>
              </el-form-item>
              
              <el-form-item v-if="routerConfig.use_auto_mode === 'fixed'" label="默认模型">
                <el-select v-model="routerConfig.default_model" @change="updateRouterConfig" filterable style="width: 300px">
                  <el-option
                    v-for="model in availableModels"
                    :key="model"
                    :label="model"
                    :value="model"
                  />
                </el-select>
              </el-form-item>

              <el-form-item v-if="routerConfig.use_auto_mode === 'latest'">
                <div class="latest-hint">
                  <el-icon><InfoFilled /></el-icon>
                  <span>自动使用效果评分最高的模型</span>
                </div>
              </el-form-item>
            </el-form>

            <div class="quick-link">
              <el-button type="primary" link @click="$router.push('/model-management')">
                <el-icon><Setting /></el-icon>
                前往模型管理页面
              </el-button>
            </div>
          </div>

          <div class="api-section">
            <div class="section-title">可用模型</div>
            <div class="models-grid">
              <div class="model-tag">
                <el-tag type="success">auto (智能选择)</el-tag>
              </div>
              <div class="model-tag">
                <el-tag type="primary">default (服务商默认)</el-tag>
              </div>
              <div class="model-tag">
                <el-tag type="warning">latest (最新模型)</el-tag>
              </div>
              <div class="model-tag" v-for="model in topModels" :key="model">
                <el-tag>{{ model }}</el-tag>
              </div>
            </div>
            <div class="hint">共 {{ availableModels.length }} 个模型可用</div>
          </div>
        </el-card>

        <el-card shadow="never" class="page-card" style="margin-top: 24px">
          <template #header>
            <div class="card-header">
              <span>调用示例</span>
              <el-radio-group v-model="selectedLang" size="small">
                <el-radio-button value="curl">cURL</el-radio-button>
                <el-radio-button value="python">Python</el-radio-button>
                <el-radio-group value="javascript">JavaScript</el-radio-group>
              </el-radio-group>
            </div>
          </template>

          <div class="code-block">
            <pre><code>{{ codeExample }}</code></pre>
            <el-button class="copy-btn" type="primary" link size="small" @click="copyCode">
              <el-icon><CopyDocument /></el-icon>
              复制代码
            </el-button>
          </div>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never" class="page-card test-card">
          <template #header>
            <div class="card-header">
              <span>在线测试</span>
            </div>
          </template>

          <el-form :model="testForm" label-width="80px" class="test-form">
            <el-form-item label="API Key">
              <el-select v-model="testForm.apiKey" filterable placeholder="选择 API Key" style="width: 100%">
                <el-option
                  v-for="key in enabledApiKeys"
                  :key="key.id"
                  :label="`${key.name} (${maskKey(key.key)})`"
                  :value="key.key"
                />
              </el-select>
            </el-form-item>
            
            <el-form-item label="模型">
              <el-select v-model="testForm.model" filterable style="width: 100%">
                <el-option label="auto (智能选择)" value="auto" />
                <el-option label="default (服务商默认)" value="default" />
                <el-option label="latest (最新模型)" value="latest" />
                <el-option-group label="指定模型">
                  <el-option
                    v-for="model in availableModels"
                    :key="model"
                    :label="model"
                    :value="model"
                  />
                </el-option-group>
              </el-select>
            </el-form-item>

            <el-form-item label="消息">
              <el-input
                v-model="testForm.message"
                type="textarea"
                :rows="4"
                placeholder="输入测试消息..."
              />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="runTest" :loading="testing" style="width: 100%">
                <el-icon><VideoPlay /></el-icon>
                发送请求
              </el-button>
            </el-form-item>
          </el-form>

          <div v-if="testResult" class="test-result">
            <div class="result-header">
              <span>响应结果</span>
              <div class="result-meta">
                <el-tag :type="testResult.success ? 'success' : 'danger'" size="small">
                  {{ testResult.success ? '成功' : '失败' }}
                </el-tag>
                <span v-if="testResult.latency" class="latency">{{ testResult.latency }}ms</span>
              </div>
            </div>
            <div class="result-body">
              <pre>{{ testResult.data }}</pre>
            </div>
          </div>
        </el-card>

        <el-card shadow="never" class="page-card" style="margin-top: 24px">
          <template #header>
            <div class="card-header">
              <span>API 统计</span>
            </div>
          </template>
          
          <div class="stats-list">
            <div class="stat-item">
              <span class="stat-label">今日请求</span>
              <span class="stat-value">{{ apiStats.todayRequests }}</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">成功率</span>
              <span class="stat-value success">{{ apiStats.successRate }}%</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">平均延迟</span>
              <span class="stat-value">{{ apiStats.avgLatency }}ms</span>
            </div>
            <div class="stat-item">
              <span class="stat-label">可用模型</span>
              <span class="stat-value">{{ availableModels.length }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 创建 API Key 对话框 -->
    <el-dialog v-model="createKeyDialogVisible" title="创建 API Key" width="450px" destroy-on-close>
      <el-form :model="newKeyForm" :rules="newKeyRules" ref="newKeyFormRef" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="newKeyForm.name" placeholder="如：生产环境、测试环境" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="newKeyForm.description" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createKeyDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createApiKey" :loading="creating">创建</el-button>
      </template>
    </el-dialog>

    <!-- 显示新创建的 Key -->
    <el-dialog v-model="showKeyDialogVisible" title="API Key 已创建" width="500px" :close-on-click-modal="false">
      <el-alert type="warning" :closable="false" style="margin-bottom: 16px">
        <template #title>请立即复制并保存 API Key</template>
        <template #default>此 Key 只会显示一次，关闭后将无法再次查看完整内容</template>
      </el-alert>
      <div class="new-key-display">
        <code>{{ newlyCreatedKey }}</code>
        <el-button type="primary" size="small" @click="copyKey(newlyCreatedKey)">
          <el-icon><CopyDocument /></el-icon>
          复制
        </el-button>
      </div>
      <template #footer>
        <el-button type="primary" @click="showKeyDialogVisible = false">我已保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { useRouter } from 'vue-router'

interface ApiKey {
  id: string
  name: string
  key: string
  description?: string
  created_at: string
  last_used?: string
  enabled: boolean
  visible?: boolean
}

const router = useRouter()

function getToken() {
  const token = localStorage.getItem('token')
  if (!token) {
    router.push('/login')
    return ''
  }
  return token
}
const apiBaseUrl = ref(window.location.origin)
const loadingKeys = ref(false)
const creating = ref(false)

const apiKeys = ref<ApiKey[]>([])

const createKeyDialogVisible = ref(false)
const showKeyDialogVisible = ref(false)
const newlyCreatedKey = ref('')
const newKeyFormRef = ref<FormInstance>()

const newKeyForm = ref({
  name: '',
  description: ''
})

const newKeyRules: FormRules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }]
}

const enabledApiKeys = computed(() => apiKeys.value.filter(k => k.enabled))

const routerConfig = ref({
  use_auto_mode: 'auto',
  default_strategy: 'auto',
  default_model: 'deepseek-chat'
})

const strategies = ref([
  { value: 'auto', label: '智能平衡', description: '效果+速度+成本综合最优' },
  { value: 'quality', label: '效果优先', description: '优先选择效果最好的模型' },
  { value: 'speed', label: '速度优先', description: '优先选择响应最快的模型' },
  { value: 'cost', label: '成本优先', description: '优先选择成本最低的模型' },
  { value: 'custom', label: '自定义规则', description: '根据任务类型自动选择' }
])

const availableModels = ref<string[]>([])
const topModels = ref<string[]>([])
const providerDefaults = ref<{ id: string; label: string; defaultModel: string }[]>([])

const selectedLang = ref('curl')
const testForm = ref({
  apiKey: '',
  model: 'auto',
  message: '你好，请介绍一下你自己'
})
const testing = ref(false)
const testResult = ref<any>(null)

const apiStats = ref({
  todayRequests: 0,
  successRate: 100,
  avgLatency: 0
})

function maskKey(key: string): string {
  if (!key || key.length <= 12) return '****'
  return key.substring(0, 8) + '...' + key.substring(key.length - 4)
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', { 
    month: '2-digit', 
    day: '2-digit', 
    hour: '2-digit', 
    minute: '2-digit' 
  })
}

function toggleKeyVisibility(row: ApiKey) {
  row.visible = !row.visible
}

function copyKey(key: string) {
  navigator.clipboard.writeText(key)
  ElMessage.success('已复制到剪贴板')
}

const codeExample = computed(() => {
  const url = `${apiBaseUrl.value}/api/v1/chat/completions`
  const apiKey = testForm.value.apiKey || '<your-api-key>'
  
  let model = 'auto'
  if (routerConfig.value.use_auto_mode === 'default') {
    model = 'default'
  } else if (routerConfig.value.use_auto_mode === 'fixed') {
    model = routerConfig.value.default_model
  } else if (routerConfig.value.use_auto_mode === 'latest') {
    model = 'latest'
  }
  
  if (selectedLang.value === 'curl') {
    return `curl -X POST "${url}" \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer ${apiKey}" \\
  -d '{
    "model": "${model}",
    "messages": [
      {"role": "user", "content": "你好"}
    ],
    "stream": true
  }'

# 可用的特殊模型参数:
# - auto: 智能选择 (效果+速度+成本综合最优)
# - default: 服务商默认模型 (根据服务商配置自动选择)
# - latest: 最新模型 (效果评分最高)`
  } else if (selectedLang.value === 'python') {
    return `from openai import OpenAI

client = OpenAI(
    api_key="${apiKey}",
    base_url="${apiBaseUrl.value}/api/v1"
)

response = client.chat.completions.create(
    model="${model}",
    messages=[
        {"role": "user", "content": "你好"}
    ],
    stream=True
)

for chunk in response:
    print(chunk.choices[0].delta.content, end="")`
  } else {
    return `import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: '${apiKey}',
  baseURL: '${apiBaseUrl.value}/api/v1'
});

const stream = await client.chat.completions.create({
  model: '${model}',
  messages: [{ role: 'user', content: '你好' }],
  stream: true
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content || '');
}`
  }
})

async function loadApiKeys() {
  loadingKeys.value = true
  try {
    const token = getToken()
    if (!token) return
    const res = await fetch('/api/admin/api-keys', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.success) {
      apiKeys.value = (data.data || []).map((k: any) => ({ ...k, visible: false }))
      if (apiKeys.value.length > 0 && !testForm.value.apiKey) {
        const enabledKey = apiKeys.value.find(k => k.enabled)
        if (enabledKey) {
          testForm.value.apiKey = enabledKey.key
        }
      }
    }
  } catch (e) {
    console.error('Failed to load API keys:', e)
  } finally {
    loadingKeys.value = false
  }
}

function showCreateKeyDialog() {
  newKeyForm.value = { name: '', description: '' }
  createKeyDialogVisible.value = true
}

async function createApiKey() {
  if (!newKeyFormRef.value) return
  const valid = await newKeyFormRef.value.validate().catch(() => false)
  if (!valid) return

  creating.value = true
  try {
    const token = getToken()
    if (!token) return
    const res = await fetch('/api/admin/api-keys', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`
      },
      body: JSON.stringify(newKeyForm.value)
    })
    const data = await res.json()
    if (data.success) {
      newlyCreatedKey.value = data.data.key
      createKeyDialogVisible.value = false
      showKeyDialogVisible.value = true
      await loadApiKeys()
    } else {
      ElMessage.error(data.error?.message || '创建失败')
    }
  } catch (e: any) {
    ElMessage.error(e?.message || '创建失败')
  } finally {
    creating.value = false
  }
}

async function toggleKeyStatus(row: ApiKey) {
  try {
    const token = getToken()
    if (!token) return
    await fetch(`/api/admin/api-keys/${row.id}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`
      },
      body: JSON.stringify({ enabled: !row.enabled })
    })
    row.enabled = !row.enabled
    ElMessage.success(row.enabled ? '已启用' : '已禁用')
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function deleteKey(row: ApiKey) {
  try {
    await ElMessageBox.confirm(`确定删除 API Key "${row.name}" 吗？删除后无法恢复。`, '确认删除', { type: 'warning' })
  } catch {
    return
  }

  try {
    const token = getToken()
    if (!token) return
    await fetch(`/api/admin/api-keys/${row.id}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${token}` }
    })
    apiKeys.value = apiKeys.value.filter(k => k.id !== row.id)
    ElMessage.success('已删除')
  } catch (e) {
    ElMessage.error('删除失败')
  }
}

async function loadRouterConfig() {
  try {
    const token = getToken()
    if (!token) return
    const res = await fetch('/api/admin/router/config', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.success) {
      routerConfig.value = {
        use_auto_mode: data.data.use_auto_mode || 'auto',
        default_strategy: data.data.default_strategy || 'auto',
        default_model: data.data.default_model || 'deepseek-chat'
      }
    }
  } catch (e) {
    console.error('Failed to load router config:', e)
  }
}

async function updateRouterConfig() {
  try {
    const token = getToken()
    if (!token) return
    await fetch('/api/admin/router/config', {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`
      },
      body: JSON.stringify({
        use_auto_mode: routerConfig.value.use_auto_mode === 'auto',
        default_strategy: routerConfig.value.default_strategy,
        default_model: routerConfig.value.default_model
      })
    })
    ElMessage.success('设置已保存')
  } catch (e) {
    ElMessage.error('保存失败')
  }
}

async function loadAvailableModels() {
  try {
    const token = getToken()
    if (!token) return
    const res = await fetch('/api/admin/router/available-models', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.success) {
      availableModels.value = data.data || []
    }
  } catch (e) {
    console.error('Failed to load models:', e)
  }
}

async function loadTopModels() {
  try {
    const token = getToken()
    if (!token) return
    const res = await fetch('/api/admin/router/top-models', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.success) {
      topModels.value = data.data || []
    }
  } catch (e) {
    console.error('Failed to load top models:', e)
  }
}

async function loadProviderDefaults() {
  try {
    const token = getToken()
    if (!token) return
    const res = await fetch('/api/admin/router/provider-defaults', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const data = await res.json()
    if (data.success && data.data) {
      const providerLabels: Record<string, string> = {
        deepseek: 'DeepSeek',
        openai: 'OpenAI',
        anthropic: 'Anthropic',
        qwen: '通义千问',
        zhipu: '智谱AI',
        moonshot: '月之暗面',
        minimax: 'MiniMax',
        baichuan: '百川智能',
        volcengine: '火山方舟',
        google: 'Google'
      }
      providerDefaults.value = Object.entries(data.data as Record<string, string>).map(([id, defaultModel]) => ({
        id,
        label: providerLabels[id] || id,
        defaultModel
      }))
    }
  } catch (e) {
    console.error('Failed to load provider defaults:', e)
  }
}

async function runTest() {
  if (!testForm.value.apiKey) {
    ElMessage.warning('请选择 API Key')
    return
  }
  if (!testForm.value.message.trim()) {
    ElMessage.warning('请输入测试消息')
    return
  }

  testing.value = true
  testResult.value = null
  const startTime = Date.now()

  try {
    const res = await fetch('/api/v1/chat/completions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${testForm.value.apiKey}`
      },
      body: JSON.stringify({
        model: testForm.value.model,
        messages: [{ role: 'user', content: testForm.value.message }],
        stream: false
      })
    })

    const data = await res.json()
    const latency = Date.now() - startTime

    testResult.value = {
      success: res.ok,
      latency,
      data: JSON.stringify(data, null, 2)
    }

    if (res.ok && data.choices?.[0]?.message?.content) {
      testResult.value.data = data.choices[0].message.content
    }
    
    loadApiKeys()
  } catch (e: any) {
    testResult.value = {
      success: false,
      latency: Date.now() - startTime,
      data: e.message
    }
  } finally {
    testing.value = false
  }
}

function copyUrl() {
  navigator.clipboard.writeText(`${apiBaseUrl.value}/api/v1/chat/completions`)
  ElMessage.success('已复制')
}

function copyCode() {
  navigator.clipboard.writeText(codeExample.value)
  ElMessage.success('已复制')
}

onMounted(() => {
  loadApiKeys()
  loadRouterConfig()
  loadAvailableModels()
  loadTopModels()
  loadProviderDefaults()
})
</script>

<style scoped lang="scss">
.api-management-page {
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .api-section {
    margin-bottom: 24px;

    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
    }

    .section-title {
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .section-inline {
    display: flex;
    align-items: center;
    gap: 20px;
    margin-bottom: 16px;
    flex-wrap: wrap;

    .section-title {
      font-weight: 600;
      color: var(--el-text-color-primary);
      flex-shrink: 0;
    }

    .config-form-inline {
      margin-bottom: 0;
      
      :deep(.el-form-item) {
        margin-bottom: 0;
      }
    }
  }

  .url-box {
    display: flex;
    align-items: center;
    gap: 12px;

    .api-url {
      flex: 1;
      padding: 12px 16px;
      background: var(--el-fill-color-light);
      border-radius: var(--el-border-radius-base);
      font-family: monospace;
      font-size: 13px;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .key-cell {
    display: flex;
    align-items: center;
    gap: 4px;

    .key-value {
      font-family: monospace;
      font-size: 12px;
      background: var(--el-fill-color-light);
      padding: 4px 8px;
      border-radius: 4px;
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }

  .new-key-display {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px;
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);

    code {
      flex: 1;
      font-family: monospace;
      font-size: 14px;
      word-break: break-all;
    }
  }

  .hint {
    margin-top: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .config-form {
    max-width: 500px;
  }

  .models-grid {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 12px;
  }

  .quick-link {
    margin-top: 16px;
    padding-top: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .latest-hint,
  .default-hint {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--el-text-color-secondary);
    font-size: 13px;
    margin-bottom: 12px;
  }

  .default-providers {
    margin-top: 12px;
    padding: 12px;
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);
    max-height: 200px;
    overflow-y: auto;

    .default-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .provider-name {
        font-size: 13px;
        color: var(--el-text-color-regular);
      }
    }
  }

  .code-block {
    position: relative;
    background: var(--el-fill-color-light);
    border-radius: var(--el-border-radius-base);
    padding: 16px;

    pre {
      margin: 0;
      font-family: monospace;
      font-size: 13px;
      overflow-x: auto;
    }

    .copy-btn {
      position: absolute;
      top: 8px;
      right: 8px;
    }
  }

  .strategy-option {
    display: flex;
    flex-direction: column;
    .strategy-desc {
      font-size: 11px;
      color: var(--el-text-color-secondary);
    }
  }

  .test-form {
    margin-bottom: 16px;
  }

  .test-result {
    border-top: 1px solid var(--el-border-color-lighter);
    padding-top: 16px;

    .result-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;

      .result-meta {
        display: flex;
        align-items: center;
        gap: 8px;

        .latency {
          font-size: 11px;
          color: var(--el-text-color-secondary);
        }
      }
    }

    .result-body {
      background: var(--el-fill-color-light);
      border-radius: var(--el-border-radius-base);
      padding: 12px;
      max-height: 300px;
      overflow: auto;

      pre {
        margin: 0;
        font-family: monospace;
        font-size: 12px;
        white-space: pre-wrap;
        word-break: break-word;
      }
    }
  }

  .stats-list {
    .stat-item {
      display: flex;
      justify-content: space-between;
      padding: 12px 0;
      border-bottom: 1px solid var(--el-border-color-lighter);

      &:last-child {
        border-bottom: none;
      }

      .stat-label {
        color: var(--el-text-color-secondary);
      }

      .stat-value {
        font-weight: 600;

        &.success {
          color: var(--el-color-success);
        }
      }
    }
  }
}
</style>
