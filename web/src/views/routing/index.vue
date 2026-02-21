<template>
  <div class="routing-page">
    <el-row :gutter="24">
      <!-- 路由规则列表 -->
      <el-col :span="16">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>路由规则</span>
              <el-button type="primary" @click="showAddRuleDialog">
                <el-icon><Plus /></el-icon>
                添加规则
              </el-button>
            </div>
          </template>

          <el-table :data="routingRules" stripe class="rules-table">
            <el-table-column prop="name" label="规则名称" width="150">
              <template #default="{ row }">
                <div class="rule-name">
                  <el-icon :size="16" class="rule-icon"><Guide /></el-icon>
                  <span>{{ row.name }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="priority" label="优先级" width="80" align="center">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.priority }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="conditions" label="匹配条件" min-width="200">
              <template #default="{ row }">
                <div class="condition-tags">
                  <el-tag v-for="c in row.conditions" :key="c" size="small" class="condition-tag">
                    {{ c }}
                  </el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="target" label="目标" width="120">
              <template #default="{ row }">
                <el-tag size="small" type="success">{{ row.target }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="hitRate" label="命中率" width="100">
              <template #default="{ row }">
                <div class="hit-rate">
                  <el-progress :percentage="row.hitRate" :stroke-width="6" :show-text="false" />
                  <span class="rate-text">{{ row.hitRate }}%</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="enabled" label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="handleRuleStatusChange(row)" />
              </template>
            </el-table-column>
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button type="primary" link size="small" @click="showEditRuleDialog(row)">编辑</el-button>
                <el-button type="danger" link size="small" @click="handleDeleteRule(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>

          <div class="rules-footer">
            <el-button @click="moveRuleUp" :disabled="!selectedRule">
              <el-icon><Top /></el-icon>
              上移
            </el-button>
            <el-button @click="moveRuleDown" :disabled="!selectedRule">
              <el-icon><Bottom /></el-icon>
              下移
            </el-button>
            <span class="priority-hint">选中规则后可调整优先级顺序</span>
          </div>
        </el-card>
      </el-col>

      <!-- 路由策略配置 -->
      <el-col :span="8">
        <el-card shadow="never" class="page-card strategy-card">
          <template #header>
            <div class="card-header">
              <span>全局策略</span>
            </div>
          </template>

          <el-form label-position="top" class="strategy-form">
            <el-form-item label="负载均衡策略">
              <el-select v-model="globalStrategy.loadBalance" style="width: 100%">
                <el-option label="轮询 (Round Robin)" value="round-robin">
                  <div class="option-content">
                    <span>轮询</span>
                    <span class="option-desc">按顺序依次分配</span>
                  </div>
                </el-option>
                <el-option label="加权轮询 (Weighted)" value="weighted">
                  <div class="option-content">
                    <span>加权轮询</span>
                    <span class="option-desc">按权重比例分配</span>
                  </div>
                </el-option>
                <el-option label="最少连接 (Least Conn)" value="least-conn">
                  <div class="option-content">
                    <span>最少连接</span>
                    <span class="option-desc">分配给连接数最少的</span>
                  </div>
                </el-option>
                <el-option label="随机 (Random)" value="random">
                  <div class="option-content">
                    <span>随机</span>
                    <span class="option-desc">随机选择服务商</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>

            <el-form-item label="故障转移">
              <div class="form-row">
                <el-switch v-model="globalStrategy.failover" />
                <span class="form-hint">当主服务商不可用时自动切换</span>
              </div>
            </el-form-item>

            <el-form-item label="健康检查间隔">
              <el-input-number v-model="globalStrategy.healthCheckInterval" :min="5" :max="300" style="width: 100%" />
              <span class="form-hint">秒</span>
            </el-form-item>

            <el-form-item label="请求超时">
              <el-input-number v-model="globalStrategy.timeout" :min="1" :max="120" style="width: 100%" />
              <span class="form-hint">秒</span>
            </el-form-item>

            <el-form-item label="重试次数">
              <el-input-number v-model="globalStrategy.retryCount" :min="0" :max="5" style="width: 100%" />
            </el-form-item>

            <el-form-item>
              <el-button type="primary" @click="saveGlobalStrategy" style="width: 100%">
                <el-icon><Check /></el-icon>
                保存配置
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- 服务商权重 -->
        <el-card shadow="never" class="page-card weight-card">
          <template #header>
            <div class="card-header">
              <span>服务商权重</span>
              <el-tag size="small" type="info">加权轮询模式</el-tag>
            </div>
          </template>

          <div class="weight-list">
            <div v-for="provider in providerWeights" :key="provider.name" class="weight-item">
              <div class="weight-header">
                <span class="provider-name">{{ provider.name }}</span>
                <span class="weight-value">{{ provider.weight }}%</span>
              </div>
              <el-slider
                v-model="provider.weight"
                :min="0"
                :max="100"
                :format-tooltip="(val: number) => `${val}%`"
              />
            </div>
          </div>

          <el-button type="primary" link @click="saveWeights" style="margin-top: 16px">
            <el-icon><Check /></el-icon>
            保存权重配置
          </el-button>
        </el-card>
      </el-col>
    </el-row>

    <!-- 添加/编辑规则对话框 -->
    <el-dialog
      v-model="ruleDialogVisible"
      :title="isEditRule ? '编辑路由规则' : '添加路由规则'"
      width="600px"
      destroy-on-close
    >
      <el-form :model="ruleForm" :rules="ruleFormRules" ref="ruleFormRef" label-width="100px">
        <el-form-item label="规则名称" prop="name">
          <el-input v-model="ruleForm.name" placeholder="请输入规则名称" />
        </el-form-item>
        <el-form-item label="优先级" prop="priority">
          <el-input-number v-model="ruleForm.priority" :min="1" :max="100" />
          <span class="form-hint">数字越小优先级越高</span>
        </el-form-item>
        <el-form-item label="匹配条件">
          <div class="condition-builder">
            <div v-for="(condition, index) in ruleForm.conditionList" :key="index" class="condition-row">
              <el-select v-model="condition.type" placeholder="条件类型" style="width: 120px">
                <el-option label="模型" value="model" />
                <el-option label="用户" value="user" />
                <el-option label="成本" value="cost" />
                <el-option label="延迟" value="latency" />
                <el-option label="自定义" value="custom" />
              </el-select>
              <el-select v-model="condition.operator" placeholder="操作符" style="width: 100px">
                <el-option label="=" value="eq" />
                <el-option label="!=" value="neq" />
                <el-option label="匹配" value="match" />
                <el-option label="包含" value="contains" />
              </el-select>
              <el-input v-model="condition.value" placeholder="值" style="flex: 1" />
              <el-button type="danger" link @click="removeCondition(index)" v-if="ruleForm.conditionList.length > 1">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-button type="primary" link @click="addCondition">
              <el-icon><Plus /></el-icon>
              添加条件
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="目标服务商" prop="target">
          <el-select v-model="ruleForm.target" placeholder="选择目标服务商" style="width: 100%">
            <el-option-group label="国际服务商">
              <el-option label="OpenAI" value="OpenAI" />
              <el-option label="Azure" value="Azure" />
              <el-option label="Anthropic" value="Anthropic" />
              <el-option label="Google" value="Google" />
            </el-option-group>
            <el-option-group label="国内服务商">
              <el-option label="火山方舟 (字节跳动)" value="Volcengine" />
              <el-option label="阿里云通义千问" value="Qwen" />
              <el-option label="百度文心一言" value="Ernie" />
              <el-option label="智谱AI" value="Zhipu" />
              <el-option label="腾讯混元" value="Hunyuan" />
              <el-option label="月之暗面" value="Moonshot" />
              <el-option label="MiniMax" value="MiniMax" />
              <el-option label="百川智能" value="Baichuan" />
              <el-option label="讯飞星火" value="Spark" />
              <el-option label="DeepSeek" value="DeepSeek" />
            </el-option-group>
            <el-option-group label="智能策略">
              <el-option label="成本最低" value="Least Cost" />
              <el-option label="延迟最低" value="Fastest" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="备用目标">
          <el-select v-model="ruleForm.fallback" placeholder="可选" clearable style="width: 100%">
            <el-option label="OpenAI" value="OpenAI" />
            <el-option label="Azure" value="Azure" />
            <el-option label="Anthropic" value="Anthropic" />
            <el-option label="阿里云通义千问" value="Qwen" />
            <el-option label="百度文心一言" value="Ernie" />
            <el-option label="智谱AI" value="Zhipu" />
            <el-option label="月之暗面" value="Moonshot" />
            <el-option label="DeepSeek" value="DeepSeek" />
          </el-select>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="ruleForm.enabled" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="ruleForm.remark" type="textarea" :rows="2" placeholder="可选备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="ruleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitRuleForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'

interface RoutingRule {
  id: number
  name: string
  priority: number
  conditions: string[]
  target: string
  hitRate: number
  enabled: boolean
}

const selectedRule = ref<RoutingRule | null>(null)
const ruleDialogVisible = ref(false)
const isEditRule = ref(false)
const ruleFormRef = ref<FormInstance>()

const routingRules = ref<RoutingRule[]>([])

const globalStrategy = reactive({
  loadBalance: 'weighted',
  failover: true,
  healthCheckInterval: 30,
  timeout: 30,
  retryCount: 3
})

const providerWeights = ref([
  { name: 'OpenAI', weight: 0 },
  { name: 'Azure', weight: 0 },
  { name: 'Anthropic', weight: 0 },
  { name: '阿里云通义千问', weight: 0 },
  { name: '百度文心一言', weight: 0 },
  { name: '智谱AI', weight: 0 },
  { name: '腾讯混元', weight: 0 },
  { name: '月之暗面', weight: 0 },
  { name: 'DeepSeek', weight: 0 }
])

const ruleForm = reactive({
  id: 0,
  name: '',
  priority: 1,
  conditionList: [{ type: 'model', operator: 'match', value: '' }],
  target: '',
  fallback: '',
  enabled: true,
  remark: ''
})

const ruleFormRules: FormRules = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  priority: [{ required: true, message: '请设置优先级', trigger: 'blur' }],
  target: [{ required: true, message: '请选择目标服务商', trigger: 'change' }]
}

const showAddRuleDialog = () => {
  isEditRule.value = false
  Object.assign(ruleForm, {
    id: 0,
    name: '',
    priority: routingRules.value.length + 1,
    conditionList: [{ type: 'model', operator: 'match', value: '' }],
    target: '',
    fallback: '',
    enabled: true,
    remark: ''
  })
  ruleDialogVisible.value = true
}

const showEditRuleDialog = (row: RoutingRule) => {
  isEditRule.value = true
  Object.assign(ruleForm, {
    id: row.id,
    name: row.name,
    priority: row.priority,
    conditionList: row.conditions.map(c => {
      const [type, value] = c.split('=')
      return { type: type || 'model', operator: 'eq', value: value || '' }
    }),
    target: row.target,
    fallback: '',
    enabled: row.enabled,
    remark: ''
  })
  ruleDialogVisible.value = true
}

const addCondition = () => {
  ruleForm.conditionList.push({ type: 'model', operator: 'match', value: '' })
}

const removeCondition = (index: number) => {
  ruleForm.conditionList.splice(index, 1)
}

const submitRuleForm = async () => {
  if (!ruleFormRef.value) return
  try {
    const valid = await ruleFormRef.value.validate()
    if (valid) {
      if (isEditRule.value) {
        ElMessage.success('规则更新成功')
      } else {
        ElMessage.success('规则添加成功')
      }
      ruleDialogVisible.value = false
    }
  } catch (error) {
    console.error('表单验证失败:', error)
  }
}

const handleDeleteRule = (row: RoutingRule) => {
  ElMessageBox.confirm(`确定删除规则 ${row.name} 吗？`, '提示', {
    type: 'warning'
  }).then(() => {
    ElMessage.success('删除成功')
  }).catch(() => {})
}

const handleRuleStatusChange = (row: RoutingRule) => {
  ElMessage.success(`${row.name} 已${row.enabled ? '启用' : '禁用'}`)
}

const moveRuleUp = () => {
  ElMessage.info('规则优先级已调整')
}

const moveRuleDown = () => {
  ElMessage.info('规则优先级已调整')
}

const saveGlobalStrategy = () => {
  ElMessage.success('全局策略已保存')
}

const saveWeights = () => {
  ElMessage.success('权重配置已保存')
}
</script>

<style scoped lang="scss">
.routing-page {
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

  .rules-table {
    .rule-name {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);
      font-weight: var(--font-weight-medium);

      .rule-icon {
        color: var(--color-primary);
      }
    }

    .condition-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 4px;

      .condition-tag {
        font-size: 11px;
      }
    }

    .hit-rate {
      display: flex;
      align-items: center;
      gap: var(--spacing-sm);

      .el-progress {
        width: 60px;
      }

      .rate-text {
        font-size: var(--font-size-sm);
        color: var(--text-secondary);
      }
    }
  }

  .rules-footer {
    display: flex;
    align-items: center;
    gap: var(--spacing-md);
    margin-top: var(--spacing-lg);
    padding-top: var(--spacing-lg);
    border-top: 1px solid var(--border-primary);

    .priority-hint {
      margin-left: auto;
      font-size: var(--font-size-sm);
      color: var(--text-tertiary);
    }
  }

  .strategy-card {
    .strategy-form {
      .form-row {
        display: flex;
        align-items: center;
        gap: var(--spacing-md);
      }

      .form-hint {
        font-size: var(--font-size-sm);
        color: var(--text-tertiary);
        margin-left: var(--spacing-sm);
      }
    }

    .option-content {
      display: flex;
      flex-direction: column;

      .option-desc {
        font-size: var(--font-size-xs);
        color: var(--text-tertiary);
      }
    }
  }

  .weight-card {
    .weight-list {
      .weight-item {
        margin-bottom: var(--spacing-lg);

        .weight-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          margin-bottom: var(--spacing-sm);

          .provider-name {
            font-weight: var(--font-weight-medium);
          }

          .weight-value {
            font-size: var(--font-size-sm);
            color: var(--color-primary);
            font-weight: var(--font-weight-semibold);
          }
        }
      }
    }
  }

  .condition-builder {
    .condition-row {
      display: flex;
      gap: var(--spacing-sm);
      margin-bottom: var(--spacing-sm);
    }

    .el-button {
      margin-top: var(--spacing-sm);
    }
  }

  .form-hint {
    font-size: var(--font-size-sm);
    color: var(--text-tertiary);
    margin-left: var(--spacing-sm);
  }
}
</style>
