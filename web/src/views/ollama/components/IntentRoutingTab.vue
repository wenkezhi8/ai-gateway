<template>
  <TabStateView
    :state="ctx.panelState.models"
    :error-text="ctx.panelError.models"
    empty-text="暂无意图路由配置数据"
    @retry="ctx.reloadModelsPanel"
  >
    <el-row :gutter="24">
      <el-col :span="16">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>意图模型配置</span>
              <div class="header-actions">
                <el-button type="primary" size="small" :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">保存配置</el-button>
                <el-button :loading="ctx.classifierModelsLoading" @click="ctx.loadClassifierModels">刷新模型</el-button>
              </div>
            </div>
          </template>

          <el-form label-width="120px">
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="启用意图分类器">
                  <el-switch v-model="ctx.classifierConfig.enabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Shadow 模式">
                  <el-switch v-model="ctx.classifierConfig.shadow_mode" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="超时(ms)">
                  <el-input-number v-model="ctx.classifierConfig.timeout_ms" :min="50" :max="10000" :step="10" style="width: 100%" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="置信度阈值">
                  <el-slider v-model="ctx.classifierConfidencePercent" :min="30" :max="95" :step="1" show-input />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="24">
                <el-form-item label="服务地址">
                  <el-input v-model="ctx.dualModelConfig.classifier_base_url" placeholder="http://127.0.0.1:11434" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="主模型">
                  <el-input v-model="ctx.dualModelConfig.classifier_active_model" placeholder="qwen2.5:0.5b-instruct" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="候选模型">
                  <el-select
                    v-model="ctx.dualModelConfig.classifier_candidate_models"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    clearable
                    style="width: 100%"
                    placeholder="输入或选择候选分类模型"
                  >
                    <el-option
                      v-for="model in ctx.classifierConfig.candidate_models"
                      :key="model"
                      :label="model"
                      :value="model"
                    />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>

            <el-divider content-position="left">控制面开关</el-divider>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="控制层总开关">
                  <el-switch v-model="ctx.classifierConfig.control.enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="控制层 Shadow">
                  <el-switch v-model="ctx.classifierConfig.control.shadow_only" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="归一化查询读">
                  <el-switch v-model="ctx.classifierConfig.control.normalized_query_read_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="缓存写门禁">
                  <el-switch v-model="ctx.classifierConfig.control.cache_write_gate_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="风险打标">
                  <el-switch v-model="ctx.classifierConfig.control.risk_tag_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="风险拦截">
                  <el-switch v-model="ctx.classifierConfig.control.risk_block_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="工具门控">
                  <el-switch v-model="ctx.classifierConfig.control.tool_gate_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Model Fit 选模">
                  <el-switch v-model="ctx.classifierConfig.control.model_fit_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="参数建议">
                  <el-switch v-model="ctx.classifierConfig.control.parameter_hint_enable" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="手动切换模型">
                  <el-select v-model="ctx.classifierSwitchModel" filterable clearable style="width: 100%" placeholder="选择分类模型">
                    <el-option v-for="model in ctx.classifierConfig.candidate_models" :key="model" :label="model" :value="model" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>

          <div class="header-actions" style="margin-bottom: 12px">
            <el-button type="primary" :loading="ctx.classifierSaving" @click="ctx.saveClassifierConfig">保存分类器配置</el-button>
            <el-button :loading="ctx.classifierSwitching" @click="ctx.switchClassifierModel">切换模型</el-button>
            <el-button type="warning" :loading="ctx.classifierSwitching" @click="ctx.startClassifierModel">启动模型</el-button>
            <el-button link @click="ctx.loadClassifierHealth">健康检查</el-button>
          </div>

          <el-alert
            :title="`分类器健康: ${ctx.classifierHealth.message || 'unknown'} (延迟 ${ctx.formatDuration(ctx.classifierHealth.latency_ms)})`"
            :type="ctx.classifierHealth.healthy ? 'success' : 'warning'"
            :closable="false"
            style="margin-bottom: 12px"
          />

          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="总请求">{{ ctx.classifierStats.total_requests }}</el-descriptions-item>
            <el-descriptions-item label="LLM 尝试">{{ ctx.classifierStats.llm_attempts }}</el-descriptions-item>
            <el-descriptions-item label="LLM 成功">{{ ctx.classifierStats.llm_success }}</el-descriptions-item>
            <el-descriptions-item label="回退次数">{{ ctx.classifierStats.fallbacks }}</el-descriptions-item>
            <el-descriptions-item label="Shadow 请求">{{ ctx.classifierStats.shadow_requests }}</el-descriptions-item>
            <el-descriptions-item label="平均延迟">{{ ctx.formatDuration(ctx.classifierStats.avg_llm_latency_ms) }}</el-descriptions-item>
            <el-descriptions-item label="控制层延迟">{{ ctx.formatDuration(ctx.classifierStats.avg_control_latency_ms) }}</el-descriptions-item>
            <el-descriptions-item label="解析错误">{{ ctx.classifierStats.parse_errors }}</el-descriptions-item>
            <el-descriptions-item label="控制字段缺失">{{ ctx.classifierStats.control_fields_missing }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :span="8">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>任务类型模型映射</span>
              <el-button type="primary" size="small" :loading="ctx.saving" @click="ctx.saveTaskMapping">保存映射</el-button>
            </div>
          </template>

          <el-alert type="info" :closable="false" style="margin-bottom: 16px">
            <template #title>
              开启后将根据任务类型自动选择对应模型，关闭则使用默认策略
            </template>
          </el-alert>

          <div class="task-model-list">
            <div class="task-model-item" v-for="task in ctx.taskTypes" :key="task.type">
              <div class="task-header">
                <el-switch v-model="ctx.taskModelMapping[task.type].enabled" size="small" />
                <span class="task-name">{{ task.name }}</span>
              </div>
              <el-select
                v-model="ctx.taskModelMapping[task.type].model"
                :disabled="!ctx.taskModelMapping[task.type]?.enabled"
                filterable
                size="small"
                style="width: 100%"
                placeholder="选择模型"
              >
                <el-option
                  v-for="m in ctx.availableModels"
                  :key="m.id"
                  :label="m.display_name || m.id"
                  :value="m.id"
                />
              </el-select>
            </div>
          </div>
        </el-card>

        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>级联路由策略</span>
              <el-tag size="small" type="success">自动升级</el-tag>
            </div>
          </template>

          <div class="cascade-levels">
            <div v-for="level in ctx.cascadeLevels" :key="level.key" class="cascade-level">
              <div class="level-header">
                <el-tag :type="level.type" size="small">{{ level.label }}</el-tag>
                <span class="level-desc">{{ level.desc }}</span>
              </div>
              <div class="level-models">
                <el-tag
                  v-for="model in level.models"
                  :key="model"
                  size="small"
                  class="model-tag"
                >
                  {{ model }}
                </el-tag>
              </div>
            </div>
          </div>

          <el-alert type="info" :closable="false" show-icon style="margin-top: 16px">
            <template #title>
              当小模型无法处理时，自动升级到大模型
            </template>
          </el-alert>
        </el-card>
      </el-col>
    </el-row>
  </TabStateView>
</template>

<script setup lang="ts">
import TabStateView from './TabStateView.vue'

defineProps<{
  ctx: any
}>()
</script>

<style scoped lang="scss">
.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-model-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.task-model-item {
  padding: 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.task-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-primary);
}

.cascade-levels {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.level-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.level-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.level-models {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.model-tag {
  font-size: 11px;
}
</style>
