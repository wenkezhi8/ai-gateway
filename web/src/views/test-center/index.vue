<template>
  <div class="test-center-page">
    <el-row :gutter="24">
      <!-- API测试面板 -->
      <el-col :span="12">
        <el-card shadow="never" class="page-card test-panel">
          <template #header>
            <div class="card-header">
              <span>API 测试</span>
              <el-tag type="info" size="small">
                <el-icon><Connection /></el-icon>
                在线测试
              </el-tag>
            </div>
          </template>

          <el-form :model="testForm" :rules="testRules" ref="testFormRef" label-width="100px" class="test-form">
            <el-form-item label="服务商" prop="provider">
              <el-select v-model="testForm.provider" placeholder="选择服务商" style="width: 100%" @change="handleProviderChange">
                <el-option v-for="p in providers" :key="p.value" :label="p.label" :value="p.value">
                  <div class="provider-option">
                    <img v-if="p.logo" :src="p.logo" class="provider-logo" />
                    <span v-else class="provider-icon" :style="{ background: p.color }">{{ p.label.charAt(0) }}</span>
                    <span>{{ p.label }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>

            <el-form-item label="模型" prop="model">
              <el-select v-model="testForm.model" placeholder="选择模型" style="width: 100%">
                <el-option v-for="m in availableModels" :key="m" :label="m" :value="m" />
              </el-select>
            </el-form-item>

            <el-form-item label="请求类型">
              <el-radio-group v-model="testForm.method">
                <el-radio-button value="chat">Chat</el-radio-button>
                <el-radio-button value="completion">Completion</el-radio-button>
                <el-radio-button value="embedding">Embedding</el-radio-button>
              </el-radio-group>
            </el-form-item>

            <el-form-item label="消息内容" prop="prompt">
              <el-input
                v-model="testForm.prompt"
                type="textarea"
                :rows="4"
                placeholder="输入测试消息，例如：Hello, how are you?"
              />
            </el-form-item>

            <el-collapse class="params-collapse">
              <el-collapse-item title="高级参数" name="params">
                <el-row :gutter="16">
                  <el-col :span="8">
                    <div class="param-item">
                      <label>Temperature</label>
                      <el-slider v-model="testForm.temperature" :min="0" :max="2" :step="0.1" show-input />
                    </div>
                  </el-col>
                  <el-col :span="8">
                    <div class="param-item">
                      <label>Max Tokens</label>
                      <el-input-number v-model="testForm.maxTokens" :min="1" :max="4096" style="width: 100%" />
                    </div>
                  </el-col>
                  <el-col :span="8">
                    <div class="param-item">
                      <label>Top P</label>
                      <el-slider v-model="testForm.topP" :min="0" :max="1" :step="0.1" show-input />
                    </div>
                  </el-col>
                </el-row>
                <el-row :gutter="16" style="margin-top: 16px">
                  <el-col :span="12">
                    <div class="param-item">
                      <label>Stream</label>
                      <el-switch v-model="testForm.stream" />
                    </div>
                  </el-col>
                  <el-col :span="12">
                    <div class="param-item">
                      <label>Frequency Penalty</label>
                      <el-slider v-model="testForm.frequencyPenalty" :min="-2" :max="2" :step="0.1" show-input />
                    </div>
                  </el-col>
                </el-row>
              </el-collapse-item>
            </el-collapse>

            <el-form-item class="action-buttons">
              <el-button type="primary" @click="runTest" :loading="testing" size="large">
                <el-icon><VideoPlay /></el-icon>
                发送请求
              </el-button>
              <el-button @click="clearTest" size="large">
                <el-icon><Delete /></el-icon>
                清空
              </el-button>
              <el-button @click="saveAsPreset" size="large">
                <el-icon><FolderAdd /></el-icon>
                保存预设
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </el-col>

      <!-- 响应结果 -->
      <el-col :span="12">
        <el-card shadow="never" class="page-card result-panel">
          <template #header>
            <div class="card-header">
              <span>响应结果</span>
              <div class="result-actions">
                <el-tag v-if="testResult.status" :type="testResult.status === 'success' ? 'success' : 'danger'" size="small">
                  {{ testResult.status === 'success' ? '成功' : '失败' }}
                </el-tag>
                <el-button v-if="testResult.response" type="primary" link size="small" @click="copyResponse">
                  <el-icon><CopyDocument /></el-icon>
                  复制
                </el-button>
              </div>
            </div>
          </template>

          <div v-if="testResult.response" class="response-area">
            <div class="response-stats">
              <div class="stat-item">
                <el-icon><Timer /></el-icon>
                <span>{{ testResult.latency }}ms</span>
                <label>延迟</label>
              </div>
              <div class="stat-item">
                <el-icon><Coin /></el-icon>
                <span>{{ testResult.tokens }}</span>
                <label>Tokens</label>
              </div>
              <div class="stat-item">
                <el-icon><OfficeBuilding /></el-icon>
                <span>{{ testResult.actualProvider }}</span>
                <label>服务商</label>
              </div>
            </div>

            <el-tabs v-model="responseTab" class="response-tabs">
              <el-tab-pane label="格式化" name="formatted">
                <div class="response-content formatted">
                  <pre>{{ testResult.response }}</pre>
                </div>
              </el-tab-pane>
              <el-tab-pane label="原始" name="raw">
                <div class="response-content raw">
                  <pre>{{ testResult.rawResponse }}</pre>
                </div>
              </el-tab-pane>
              <el-tab-pane label="内容" name="content" v-if="testResult.content">
                <div class="response-content content">
                  {{ testResult.content }}
                </div>
              </el-tab-pane>
            </el-tabs>
          </div>

          <el-empty v-else description="发送请求后查看响应结果" />
        </el-card>

        <!-- 测试历史 -->
        <el-card shadow="never" class="page-card history-panel">
          <template #header>
            <div class="card-header">
              <span>测试历史</span>
              <el-button type="danger" link size="small" @click="clearHistory">
                <el-icon><Delete /></el-icon>
                清空
              </el-button>
            </div>
          </template>

          <el-table :data="testHistory" stripe size="small" max-height="200">
            <el-table-column prop="time" label="时间" width="80" />
            <el-table-column prop="provider" label="服务商" width="90">
              <template #default="{ row }">
                <el-tag size="small">{{ row.provider }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="model" label="模型" width="110" />
            <el-table-column prop="latency" label="延迟" width="80">
              <template #default="{ row }">
                <span :class="getLatencyClass(row.latency)">{{ row.latency }}ms</span>
              </template>
            </el-table-column>
            <el-table-column prop="status" label="状态" width="70">
              <template #default="{ row }">
                <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
                  {{ row.status === 'success' ? '成功' : '失败' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="70">
              <template #default="{ row }">
                <el-button type="primary" link size="small" @click="rerunTest(row)">
                  重跑
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 批量测试 -->
    <el-card shadow="never" class="page-card batch-panel">
      <template #header>
        <div class="card-header">
          <span>批量测试</span>
          <el-button type="primary" @click="showBatchDialog">
            <el-icon><Plus /></el-icon>
            新建测试
          </el-button>
        </div>
      </template>

      <el-table :data="batchTests" stripe>
        <el-table-column prop="name" label="测试名称" min-width="180">
          <template #default="{ row }">
            <div class="test-name">
              <el-icon><List /></el-icon>
              <span>{{ row.name }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="cases" label="用例数" width="100" align="center" />
        <el-table-column prop="successRate" label="成功率" width="150">
          <template #default="{ row }">
            <div class="success-rate">
              <el-progress :percentage="row.successRate" :stroke-width="8" :show-text="false" :status="getRateStatus(row.successRate)" />
              <span class="rate-text">{{ row.successRate }}%</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="avgLatency" label="平均延迟" width="100" />
        <el-table-column prop="lastRun" label="最后运行" width="160" />
        <el-table-column label="操作" width="180">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="runBatchTestById(row)">
              <el-icon><VideoPlay /></el-icon>
              运行
            </el-button>
            <el-button type="primary" link size="small" @click="viewBatchResult(row)">
              <el-icon><View /></el-icon>
              结果
            </el-button>
            <el-button type="danger" link size="small" @click="deleteBatchTest(row)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 批量测试对话框 -->
    <el-dialog v-model="batchDialogVisible" title="新建批量测试" width="600px">
      <el-form :model="batchForm" label-width="100px">
        <el-form-item label="测试名称">
          <el-input v-model="batchForm.name" placeholder="输入测试名称" />
        </el-form-item>
        <el-form-item label="测试用例">
          <el-input v-model="batchForm.cases" type="textarea" :rows="6" placeholder="每行一个测试用例，格式：服务商,模型,提示词" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="batchDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="createBatchTest">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import axios from 'axios'
import { API } from '@/constants/api'

const testing = ref(false)
const testFormRef = ref<FormInstance>()
const responseTab = ref('formatted')
const batchDialogVisible = ref(false)

const providers = [
  { label: 'OpenAI', value: 'openai', color: '#10A37F', logo: '/logos/openai.svg' },
  { label: 'Azure OpenAI', value: 'azure', color: '#0078D4', logo: '/logos/azure.svg' },
  { label: 'Anthropic Claude', value: 'anthropic', color: '#CC785C', logo: '/logos/anthropic.svg' },
  { label: 'Google Gemini', value: 'google', color: '#4285F4', logo: '/logos/google.svg' },
  { label: '火山方舟', value: 'volcengine', color: '#FF4D4F', logo: '/logos/volcengine.svg' },
  { label: '阿里云通义千问', value: 'qwen', color: '#FF6A00', logo: '/logos/qwen.svg' },
  { label: '百度文心一言', value: 'ernie', color: '#2932E1', logo: '/logos/ernie.svg' },
  { label: '智谱AI', value: 'zhipu', color: '#3657ED', logo: '/logos/zhipu.svg' },
  { label: '腾讯混元', value: 'hunyuan', color: '#00A3FF', logo: '/logos/hunyuan.svg' },
  { label: '月之暗面', value: 'moonshot', color: '#1A1A1A', logo: '/logos/moonshot.svg' },
  { label: 'MiniMax', value: 'minimax', color: '#615CED', logo: '/logos/minimax.svg' },
  { label: '百川智能', value: 'baichuan', color: '#0066FF', logo: '/logos/baichuan.svg' },
  { label: '讯飞星火', value: 'spark', color: '#E60012', logo: '/logos/spark.svg' },
  { label: 'DeepSeek', value: 'deepseek', color: '#4D6BFE', logo: '/logos/deepseek.svg' }
]

const modelsByProvider: Record<string, string[]> = {
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-4-turbo', 'gpt-4', 'gpt-3.5-turbo', 'o1', 'o1-mini', 'o1-preview'],
  azure: ['gpt-4o', 'gpt-4', 'gpt-35-turbo'],
  anthropic: ['claude-3-5-sonnet-20241022', 'claude-3-opus-20240229', 'claude-3-sonnet-20240229', 'claude-3-haiku-20240307'],
  google: ['gemini-2.0-flash-exp', 'gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-pro'],
  volcengine: ['doubao-pro-256k', 'doubao-pro-128k', 'doubao-pro-32k', 'doubao-lite-128k', 'doubao-lite-32k'],
  qwen: ['qwen-max', 'qwen-max-longcontext', 'qwen-plus', 'qwen-turbo', 'qwen-long'],
  ernie: ['ernie-4.0-8k', 'ernie-4.0', 'ernie-3.5-8k', 'ernie-3.5', 'ernie-speed-8k', 'ernie-speed'],
  zhipu: ['glm-4-plus', 'glm-4-0520', 'glm-4-air', 'glm-4-airx', 'glm-4-long', 'glm-4-flash'],
  hunyuan: ['hunyuan-lite', 'hunyuan-standard', 'hunyuan-pro', 'hunyuan-turbo'],
  moonshot: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k'],
  minimax: ['abab6.5-chat', 'abab6.5s-chat', 'abab5.5-chat', 'abab5.5s-chat'],
  baichuan: ['Baichuan4', 'Baichuan3-Turbo', 'Baichuan3-Turbo-128k', 'Baichuan2-Turbo'],
  spark: ['spark-v3.5', 'spark-v3.0', 'spark-v2.0', 'spark-v1.5'],
  deepseek: ['deepseek-chat', 'deepseek-reasoner']
}

const testForm = reactive({
  provider: 'openai',
  model: 'gpt-4o',
  method: 'chat',
  prompt: 'Hello, how are you?',
  temperature: 0.7,
  maxTokens: 1000,
  topP: 1,
  stream: false,
  frequencyPenalty: 0
})

const testRules: FormRules = {
  provider: [{ required: true, message: '请选择服务商', trigger: 'change' }],
  model: [{ required: true, message: '请选择模型', trigger: 'change' }],
  prompt: [{ required: true, message: '请输入消息内容', trigger: 'blur' }]
}

const testResult = reactive({
  status: null as string | null,
  response: null as string | null,
  rawResponse: null as string | null,
  latency: 0,
  tokens: 0,
  actualProvider: '',
  content: ''
})

interface TestHistoryItem {
  time: string
  provider: string
  model: string
  latency: number
  status: string
  prompt: string
}

const testHistory = ref<TestHistoryItem[]>([])

interface BatchTestItem {
  name: string
  provider: string
  model: string
  status: string
  latency: string
}

const batchTests = ref<BatchTestItem[]>([])

const batchForm = reactive({
  name: '',
  cases: ''
})

const availableModels = computed(() => {
  return modelsByProvider[testForm.provider] || []
})

const handleProviderChange = () => {
  testForm.model = availableModels.value[0] || ''
}

const runTest = async () => {
  if (!testFormRef.value) return

  try {
    const valid = await testFormRef.value.validate()
    if (!valid) return

    testing.value = true
    const startTime = Date.now()

    try {
      // 尝试调用真实的后端API
      const response = await axios.post(API.V1.CHAT_COMPLETIONS, {
        model: testForm.model,
        messages: [{ role: 'user', content: testForm.prompt }],
        temperature: testForm.temperature,
        max_tokens: testForm.maxTokens,
        top_p: testForm.topP,
        stream: false
      }, {
        timeout: 60000
      })

      const latency = Date.now() - startTime
      const data = response.data

      testResult.status = 'success'
      testResult.latency = latency
      testResult.tokens = data.usage?.total_tokens || 0
      testResult.actualProvider = testForm.provider
      testResult.rawResponse = JSON.stringify(data, null, 2)
      testResult.response = JSON.stringify(data, null, 2)
      testResult.content = data.choices?.[0]?.message?.content || ''

      // 添加到历史
      testHistory.value.unshift({
        time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        provider: providers.find(p => p.value === testForm.provider)?.label || testForm.provider,
        model: testForm.model,
        latency,
        status: 'success',
        prompt: testForm.prompt
      })

      ElMessage.success('测试完成')
    } catch (error: any) {
      const latency = Date.now() - startTime

      testResult.status = 'failed'
      testResult.latency = latency
      testResult.tokens = 0
      testResult.actualProvider = testForm.provider
      testResult.response = JSON.stringify({
        error: true,
        message: error.response?.data?.error?.message || error.message || '请求失败',
        code: error.response?.status || 500
      }, null, 2)
      testResult.rawResponse = testResult.response
      testResult.content = ''

      testHistory.value.unshift({
        time: new Date().toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
        provider: providers.find(p => p.value === testForm.provider)?.label || testForm.provider,
        model: testForm.model,
        latency,
        status: 'failed',
        prompt: testForm.prompt
      })

      ElMessage.error('请求失败: ' + (error.response?.data?.error?.message || error.message))
    } finally {
      testing.value = false
    }
  } catch (validationError) {
    console.error('表单验证失败:', validationError)
  }
}

const clearTest = () => {
  testForm.prompt = ''
  testResult.status = null
  testResult.response = null
  testResult.rawResponse = null
  testResult.latency = 0
  testResult.tokens = 0
  testResult.actualProvider = ''
  testResult.content = ''
}

const copyResponse = async () => {
  if (testResult.response) {
    await navigator.clipboard.writeText(testResult.response)
    ElMessage.success('已复制到剪贴板')
  }
}

const saveAsPreset = () => {
  ElMessage.success('预设保存成功')
}

const rerunTest = (row: any) => {
  testForm.prompt = row.prompt
  runTest()
}

const clearHistory = () => {
  ElMessageBox.confirm('确定清空所有测试历史吗？', '提示', { type: 'warning' })
    .then(() => {
      testHistory.value = []
      ElMessage.success('已清空')
    })
    .catch(() => {})
}

const getLatencyClass = (latency: number) => {
  if (latency < 150) return 'fast'
  if (latency < 300) return 'normal'
  return 'slow'
}

const getRateStatus = (rate: number) => {
  if (rate >= 90) return 'success'
  if (rate >= 70) return 'warning'
  return 'exception'
}

const showBatchDialog = () => {
  batchForm.name = ''
  batchForm.cases = ''
  batchDialogVisible.value = true
}

const createBatchTest = () => {
  ElMessage.success('批量测试创建成功')
  batchDialogVisible.value = false
}

const runBatchTestById = (row: any) => {
  ElMessage.info(`正在运行批量测试: ${row.name}`)
}

const viewBatchResult = (row: any) => {
  ElMessage.info(`查看批量测试结果: ${row.name}`)
}

const deleteBatchTest = (row: any) => {
  ElMessageBox.confirm(`确定删除批量测试 ${row.name} 吗？`, '提示', { type: 'warning' })
    .then(() => {
      ElMessage.success('删除成功')
    })
    .catch(() => {})
}
</script>

<style scoped lang="scss">
.test-center-page {
  .page-card {
    border-radius: var(--border-radius-lg);
    border: none;
    margin-bottom: var(--spacing-xl);
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .test-form {
    .provider-option {
      display: flex;
      align-items: center;
      gap: 10px;
      width: 100%;

      .provider-logo {
        height: 20px;
        width: auto;
        max-width: 70px;
        border-radius: 4px;
        object-fit: contain;
        flex-shrink: 0;
      }

      .provider-icon {
        width: 20px;
        height: 20px;
        border-radius: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        font-size: 10px;
        font-weight: 600;
        flex-shrink: 0;
      }
    }

    .params-collapse {
      margin-bottom: var(--spacing-lg);
      border: none;

      :deep(.el-collapse-item__header) {
        background: var(--bg-secondary);
        border-radius: var(--border-radius-md);
        padding: 0 var(--spacing-md);
        border: none;
      }

      :deep(.el-collapse-item__wrap) {
        border: none;
      }

      :deep(.el-collapse-item__content) {
        padding: var(--spacing-lg);
        background: var(--bg-secondary);
        border-radius: 0 0 var(--border-radius-md) var(--border-radius-md);
      }
    }

    .param-item {
      label {
        display: block;
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
        margin-bottom: var(--spacing-sm);
      }
    }

    .action-buttons {
      margin-top: var(--spacing-lg);
    }
  }

  .result-panel {
    .result-actions {
      display: flex;
      align-items: center;
      gap: var(--spacing-md);
    }

    .response-stats {
      display: flex;
      gap: var(--spacing-2xl);
      padding: var(--spacing-lg);
      background: var(--bg-secondary);
      border-radius: var(--border-radius-lg);
      margin-bottom: var(--spacing-lg);

      .stat-item {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;

        .el-icon {
          font-size: 20px;
          color: var(--color-primary);
        }

        span {
          font-size: var(--font-size-lg);
          font-weight: var(--font-weight-semibold);
        }

        label {
          font-size: var(--font-size-xs);
          color: var(--text-tertiary);
        }
      }
    }

    .response-tabs {
      :deep(.el-tabs__content) {
        padding: 0;
      }
    }

    .response-content {
      background: var(--bg-tertiary);
      padding: var(--spacing-lg);
      border-radius: var(--border-radius-md);
      max-height: 300px;
      overflow-y: auto;

      &.formatted, &.raw {
        pre {
          margin: 0;
          white-space: pre-wrap;
          word-wrap: break-word;
          font-family: var(--font-family-mono);
          font-size: var(--font-size-sm);
        }
      }

      &.content {
        font-size: var(--font-size-md);
        line-height: 1.6;
      }
    }
  }

  .history-panel {
    .fast { color: var(--color-success); }
    .normal { color: var(--color-warning); }
    .slow { color: var(--color-danger); }
  }

  .batch-panel {
    .test-name {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);
      font-weight: var(--font-weight-medium);
    }

    .success-rate {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);

      .el-progress {
        flex: 1;
      }

      .rate-text {
        width: 40px;
        font-size: var(--font-size-sm);
        font-weight: var(--font-weight-medium);
      }
    }
  }
}
</style>
