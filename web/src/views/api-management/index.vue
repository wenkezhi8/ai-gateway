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
            <div class="section-header">
              <div class="section-title">兼容协议端点</div>
            </div>
            <div class="endpoints-list">
              <div class="endpoint-item">
                <div class="endpoint-header">
                  <el-tag type="success">OpenAI</el-tag>
                  <span class="endpoint-desc">兼容 OpenAI 接口协议</span>
                </div>
                <div class="url-box">
                  <code class="api-url">{{ apiBaseUrl }}/api/v1</code>
                  <el-button type="primary" size="small" @click="copyUrl('openai')">
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-button>
                </div>
              </div>
              <div class="endpoint-item">
                <div class="endpoint-header">
                  <el-tag type="warning">Anthropic</el-tag>
                  <span class="endpoint-desc">兼容 Anthropic 接口协议</span>
                </div>
                <div class="url-box">
                  <code class="api-url">{{ apiBaseUrl }}/api/anthropic</code>
                  <el-button type="primary" size="small" @click="copyUrl('anthropic')">
                    <el-icon><CopyDocument /></el-icon>
                    复制
                  </el-button>
                </div>
              </div>
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
              <el-table-column width="60" align="center">
                <template #default="{ row }">
                  <el-radio v-model="selectedConfigKeyId" :value="row.id" :disabled="!row.enabled" />
                </template>
              </el-table-column>
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
              <el-button type="primary" link @click="$router.push('/routing')">
                <el-icon><Guide /></el-icon>
                前往路由策略页面
              </el-button>
            </div>
          </div>

          <div class="api-section">
            <div class="section-header">
              <div class="section-title">一键配置</div>
            </div>

            <el-tabs v-model="selectedConfigTool" type="card">
              <el-tab-pane label="OpenAI SDK" name="openai">
                <div class="config-section">
                  <div class="config-desc">
                    <el-icon><InfoFilled /></el-icon>
                    <span>配置 OpenAI 官方 SDK 使用此网关</span>
                  </div>

                  <div class="config-block">
                    <div class="block-header">
                      <span class="block-title">一键配置脚本（推荐）</span>
                      <el-button type="primary" size="small" @click="copyConfigScript('openai')">
                        <el-icon><CopyDocument /></el-icon>
                        复制命令
                      </el-button>
                    </div>
                    <div class="code-block">
                      <pre><code>{{ configScriptOpenAI }}</code></pre>
                    </div>
                  </div>

                  <div class="config-block">
                    <div class="block-header">
                      <span class="block-title">配置文件</span>
                    </div>
                    <div class="config-files">
                      <div class="config-file">
                        <div class="file-title">环境变量</div>
                        <div class="code-block small">
                          <pre><code>OPENAI_API_KEY={{ selectedConfigKey || '&lt;your-api-key&gt;' }}
OPENAI_BASE_URL={{ apiBaseUrl }}/api/v1
OPENAI_DEFAULT_MODEL={{ selectedModelForConfig }}</code></pre>
                        </div>
                        <el-button type="primary" link size="small" @click="copyConfig('openai-env')">复制</el-button>
                      </div>
                    </div>
                  </div>
                </div>
              </el-tab-pane>

              <el-tab-pane label="Anthropic SDK" name="anthropic">
                <div class="config-section">
                  <div class="config-desc">
                    <el-icon><InfoFilled /></el-icon>
                    <span>配置 Anthropic 官方 SDK 使用此网关</span>
                  </div>

                  <div class="config-block">
                    <div class="block-header">
                      <span class="block-title">一键配置脚本（推荐）</span>
                      <el-button type="primary" size="small" @click="copyConfigScript('anthropic')">
                        <el-icon><CopyDocument /></el-icon>
                        复制命令
                      </el-button>
                    </div>
                    <div class="code-block">
                      <pre><code>{{ configScriptAnthropic }}</code></pre>
                    </div>
                  </div>

                  <div class="config-block">
                    <div class="block-header">
                      <span class="block-title">配置文件</span>
                    </div>
                    <div class="config-files">
                      <div class="config-file">
                        <div class="file-title">环境变量</div>
                        <div class="code-block small">
                          <pre><code>ANTHROPIC_API_KEY={{ selectedConfigKey || '&lt;your-api-key&gt;' }}
ANTHROPIC_BASE_URL={{ apiBaseUrl }}/api/anthropic
ANTHROPIC_DEFAULT_MODEL={{ selectedModelForConfig }}</code></pre>
                        </div>
                        <el-button type="primary" link size="small" @click="copyConfig('anthropic-env')">复制</el-button>
                      </div>
                    </div>
                  </div>
                </div>
              </el-tab-pane>

              <el-tab-pane label="其他工具" name="other">
                <div class="config-section">
                  <div class="config-desc">
                    <el-icon><InfoFilled /></el-icon>
                    <span>配置其他 AI 工具使用此网关</span>
                  </div>

                  <div class="tools-grid">
                    <div class="tool-item">
                      <div class="tool-name">Claude Code</div>
                      <div class="tool-desc">AI 编程助手</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">Base URL</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('claude-code')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">Cursor</div>
                      <div class="tool-desc">AI 代码编辑器</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">OpenAI Base URL</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('cursor')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">Cline</div>
                      <div class="tool-desc">VSCode AI 代理</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">OpenAI URL</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('cline')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">OpenClaw / OpenCode</div>
                      <div class="tool-desc">AI 编程工具</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">API Base</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('openclaw')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">Cherry Studio</div>
                      <div class="tool-desc">AI 聊天客户端</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">接口地址</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('cherry-studio')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">Goose</div>
                      <div class="tool-desc">AI 开发助手</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">Base URL</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('goose')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">TRAE / Kilo Code / Roo Code</div>
                      <div class="tool-desc">AI 编程工具</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">OpenAI Endpoint</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('generic')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">Factory Droid / Crush</div>
                      <div class="tool-desc">AI 开发工具</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">API Base</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyToolConfig('generic')">复制配置</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">ChatGPT Next Web</div>
                      <div class="tool-desc">开源 AI 聊天界面</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">接口地址</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyText(`${apiBaseUrl}/api/v1`)">复制</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">LangChain</div>
                      <div class="tool-desc">AI 应用开发框架</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">base_url</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyText(`${apiBaseUrl}/api/v1`)">复制</el-button>
                        </div>
                      </div>
                    </div>
                    <div class="tool-item">
                      <div class="tool-name">LlamaIndex</div>
                      <div class="tool-desc">数据框架</div>
                      <div class="tool-config">
                        <div class="config-row">
                          <span class="config-label">base_url</span>
                          <code>{{ apiBaseUrl }}/api/v1</code>
                          <el-button type="primary" link size="small" @click="copyText(`${apiBaseUrl}/api/v1`)">复制</el-button>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </el-tab-pane>
            </el-tabs>

            <div class="config-hint">
              <el-tag type="info" size="small">提示</el-tag>
              <span>选择上方 API Key 后，配置会自动填入对应的 Key</span>
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
              <div class="header-controls">
                <el-radio-group v-model="selectedProtocol" size="small">
                  <el-radio-button value="openai">OpenAI</el-radio-button>
                  <el-radio-button value="anthropic">Anthropic</el-radio-button>
                </el-radio-group>
                <el-radio-group v-model="selectedLang" size="small">
                  <el-radio-button value="curl">cURL</el-radio-button>
                  <el-radio-button value="python">Python</el-radio-button>
                  <el-radio-button value="javascript">JavaScript</el-radio-button>
                </el-radio-group>
              </div>
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
const selectedProtocol = ref('openai')
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

const selectedConfigTool = ref('openai')
const selectedConfigKeyId = ref('')

const selectedModelForConfig = computed(() => {
  if (routerConfig.value.use_auto_mode === 'default') {
    return 'default'
  } else if (routerConfig.value.use_auto_mode === 'fixed') {
    return routerConfig.value.default_model || 'auto'
  } else if (routerConfig.value.use_auto_mode === 'latest') {
    return 'latest'
  }
  return 'auto'
})

const selectedConfigKey = computed(() => {
  if (selectedConfigKeyId.value) {
    const key = apiKeys.value.find(k => k.id === selectedConfigKeyId.value)
    if (key) return key.key
  }
  if (testForm.value.apiKey) return testForm.value.apiKey
  const enabledKey = apiKeys.value.find(k => k.enabled)
  if (enabledKey && !selectedConfigKeyId.value) {
    selectedConfigKeyId.value = enabledKey.id
  }
  return enabledKey?.key || ''
})

const configScriptOpenAI = computed(() => {
  const apiKey = selectedConfigKey.value || '<your-api-key>'
  const model = selectedModelForConfig.value
  return `bash << 'SETUP_SCRIPT'
# 创建配置目录
mkdir -p ~/.config/openai

# 备份现有配置
[ -f ~/.config/openai/config.json ] && cp ~/.config/openai/config.json ~/.config/openai/config.json.bak

# 创建新配置
cat > ~/.config/openai/config.json << 'OPENAI_CONFIG'
{
  "api_key": "${apiKey}",
  "base_url": "${apiBaseUrl.value}/api/v1"
}
OPENAI_CONFIG

# 设置环境变量（可选）
echo "export OPENAI_API_KEY=\"${apiKey}\"" >> ~/.bashrc
echo "export OPENAI_BASE_URL=\"${apiBaseUrl.value}/api/v1\"" >> ~/.bashrc
echo "export OPENAI_DEFAULT_MODEL=\"${model}\"" >> ~/.bashrc

echo "✅ Done!"
SETUP_SCRIPT`
})

const configScriptAnthropic = computed(() => {
  const apiKey = selectedConfigKey.value || '<your-api-key>'
  const model = selectedModelForConfig.value
  return `bash << 'SETUP_SCRIPT'
# 创建配置目录
mkdir -p ~/.config/anthropic

# 备份现有配置
[ -f ~/.config/anthropic/config.json ] && cp ~/.config/anthropic/config.json ~/.config/anthropic/config.json.bak

# 创建新配置
cat > ~/.config/anthropic/config.json << 'ANTHROPIC_CONFIG'
{
  "api_key": "${apiKey}",
  "base_url": "${apiBaseUrl.value}/api/anthropic"
}
ANTHROPIC_CONFIG

# 设置环境变量（可选）
echo "export ANTHROPIC_API_KEY=\"${apiKey}\"" >> ~/.bashrc
echo "export ANTHROPIC_BASE_URL=\"${apiBaseUrl.value}/api/anthropic\"" >> ~/.bashrc
echo "export ANTHROPIC_DEFAULT_MODEL=\"${model}\"" >> ~/.bashrc

echo "✅ Done!"
SETUP_SCRIPT`
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
  const apiKey = testForm.value.apiKey || '<your-api-key>'

  let model = 'auto'
  if (routerConfig.value.use_auto_mode === 'default') {
    model = 'default'
  } else if (routerConfig.value.use_auto_mode === 'fixed') {
    model = routerConfig.value.default_model
  } else if (routerConfig.value.use_auto_mode === 'latest') {
    model = 'latest'
  }

  if (selectedProtocol.value === 'openai') {
    const url = `${apiBaseUrl.value}/api/v1/chat/completions`
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
  } else {
    const url = `${apiBaseUrl.value}/api/anthropic/v1/messages`
    const anthropicModel = model === 'auto' ? 'claude-3-5-sonnet-20241022' : model
    if (selectedLang.value === 'curl') {
      return `curl -X POST "${url}" \\
  -H "Content-Type: application/json" \\
  -H "x-api-key: ${apiKey}" \\
  -H "anthropic-version: 2023-06-01" \\
  -d '{
    "model": "${anthropicModel}",
    "max_tokens": 1024,
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'`
    } else if (selectedLang.value === 'python') {
      return `from anthropic import Anthropic

client = Anthropic(
    api_key="${apiKey}",
    base_url="${apiBaseUrl.value}/api/anthropic"
)

message = client.messages.create(
    model="${anthropicModel}",
    max_tokens=1024,
    messages=[
        {"role": "user", "content": "你好"}
    ]
)

print(message.content[0].text)`
    } else {
      return `import Anthropic from '@anthropic-ai/sdk';

const client = new Anthropic({
  apiKey: '${apiKey}',
  baseURL: '${apiBaseUrl.value}/api/anthropic'
});

const message = await client.messages.create({
  model: '${anthropicModel}',
  maxTokens: 1024,
  messages: [{ role: 'user', content: '你好' }]
});

console.log(message.content[0].text);`
    }
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
        use_auto_mode: routerConfig.value.use_auto_mode,
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

function copyUrl(type: string) {
  let url = ''
  if (type === 'anthropic') {
    url = `${apiBaseUrl.value}/api/anthropic`
  } else {
    url = `${apiBaseUrl.value}/api/v1`
  }
  navigator.clipboard.writeText(url)
  ElMessage.success('已复制')
}

function copyCode() {
  navigator.clipboard.writeText(codeExample.value)
  ElMessage.success('已复制')
}

function copyConfigScript(type: string) {
  let script = ''
  if (type === 'openai') {
    script = configScriptOpenAI.value
  } else if (type === 'anthropic') {
    script = configScriptAnthropic.value
  }
  navigator.clipboard.writeText(script)
  ElMessage.success('已复制配置脚本')
}

function copyConfig(type: string) {
  let content = ''
  const apiKey = selectedConfigKey.value || '<your-api-key>'
  const model = selectedModelForConfig.value

  if (type === 'openai-env') {
    content = `OPENAI_API_KEY=${apiKey}
OPENAI_BASE_URL=${apiBaseUrl.value}/api/v1
OPENAI_DEFAULT_MODEL=${model}`
  } else if (type === 'anthropic-env') {
    content = `ANTHROPIC_API_KEY=${apiKey}
ANTHROPIC_BASE_URL=${apiBaseUrl.value}/api/anthropic
ANTHROPIC_DEFAULT_MODEL=${model}`
  }

  navigator.clipboard.writeText(content)
  ElMessage.success('已复制配置')
}

function copyText(text: string) {
  navigator.clipboard.writeText(text)
  ElMessage.success('已复制')
}

function copyToolConfig(tool: string) {
  const apiKey = selectedConfigKey.value || '<your-api-key>'
  const baseUrl = apiBaseUrl.value
  const model = selectedModelForConfig.value
  let content = ''

  switch (tool) {
    case 'claude-code':
      content = `# Claude Code 配置
OPENAI_API_KEY=${apiKey}
OPENAI_BASE_URL=${baseUrl}/api/v1
OPENAI_DEFAULT_MODEL=${model}`
      break
    case 'cursor':
      content = `# Cursor 配置
## 在设置中找到 "OpenAI API" 部分
API Key: ${apiKey}
OpenAI Base URL: ${baseUrl}/api/v1
Default Model: ${model}`
      break
    case 'cline':
      content = `# Cline 配置
## 在设置中配置 OpenAI API
API Key: ${apiKey}
Base URL: ${baseUrl}/api/v1
Model: ${model}`
      break
    case 'openclaw':
      content = `# OpenClaw / OpenCode 配置
API Key: ${apiKey}
API Base: ${baseUrl}/api/v1
Default Model: ${model}`
      break
    case 'cherry-studio':
      content = `# Cherry Studio 配置
## 在服务设置中添加 OpenAI 兼容服务
接口地址: ${baseUrl}/api/v1
API Key: ${apiKey}
默认模型: ${model}`
      break
    case 'goose':
      content = `# Goose 配置
export OPENAI_API_KEY=${apiKey}
export OPENAI_BASE_URL=${baseUrl}/api/v1
export OPENAI_DEFAULT_MODEL=${model}`
      break
    case 'generic':
    default:
      content = `# 通用 OpenAI 兼容配置
API Key: ${apiKey}
Base URL: ${baseUrl}/api/v1
Model: ${model}`
      break
  }

  navigator.clipboard.writeText(content)
  ElMessage.success('已复制配置')
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

  .endpoints-list {
    .endpoint-item {
      margin-bottom: 16px;
      padding: 16px;
      background: var(--el-bg-color-page);
      border-radius: var(--el-border-radius-base);
      border: 1px solid var(--el-border-color-lighter);

      .endpoint-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 12px;

        .endpoint-desc {
          font-size: 13px;
          color: var(--el-text-color-secondary);
        }
      }
    }
  }

  .header-controls {
    display: flex;
    align-items: center;
    gap: 8px;
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

  .header-controls {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .config-section {
    .config-desc {
      display: flex;
      align-items: center;
      gap: 8px;
      color: var(--el-text-color-secondary);
      font-size: 13px;
      margin-bottom: 16px;
    }

    .config-block {
      margin-bottom: 20px;

      .block-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 12px;

        .block-title {
          font-weight: 600;
          color: var(--el-text-color-primary);
          font-size: 14px;
        }
      }

      .code-block {
        &.small {
          pre {
            font-size: 12px;
          }
        }
      }

      .config-files {
        display: flex;
        flex-direction: column;
        gap: 16px;

        .config-file {
          background: var(--el-bg-color-page);
          border: 1px solid var(--el-border-color-lighter);
          border-radius: var(--el-border-radius-base);
          padding: 12px;

          .file-title {
            font-weight: 600;
            color: var(--el-text-color-primary);
            font-size: 13px;
            margin-bottom: 12px;
          }

          .code-block {
            margin-bottom: 12px;
          }
        }
      }
    }
  }

  .tools-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: 16px;

    .tool-item {
      background: var(--el-bg-color-page);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: var(--el-border-radius-base);
      padding: 16px;

      .tool-name {
        font-weight: 600;
        color: var(--el-text-color-primary);
        font-size: 14px;
        margin-bottom: 4px;
      }

      .tool-desc {
        color: var(--el-text-color-secondary);
        font-size: 12px;
        margin-bottom: 12px;
      }

      .tool-config {
        .config-row {
          display: flex;
          align-items: center;
          gap: 12px;
          flex-wrap: wrap;

          .config-label {
            color: var(--el-text-color-secondary);
            font-size: 13px;
            min-width: 60px;
          }

          code {
            flex: 1;
            padding: 8px 12px;
            background: var(--el-fill-color-light);
            border-radius: var(--el-border-radius-base);
            font-family: monospace;
            font-size: 12px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            min-width: 0;
          }
        }
      }
    }
  }

  .config-hint {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 12px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }
}
</style>
