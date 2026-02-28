<template>
  <TabStateView
    :state="ctx.panelState.policy"
    :error-text="ctx.panelError.policy"
    empty-text="暂无路由策略数据"
    @retry="ctx.reloadPolicyPanel"
  >
    <el-row :gutter="24">
      <el-col :span="16">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>路由策略配置</span>
              <el-button type="primary" size="small" @click="ctx.saveTaskMapping" :loading="ctx.saving">
                <el-icon><Check /></el-icon>
                保存映射
              </el-button>
            </div>
          </template>

          <el-form label-width="120px">
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="当前路由模式">
                  <el-tag size="small" type="info">{{ ctx.modeLabel }}</el-tag>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="默认策略">
                  <el-tag size="small" type="info">{{ ctx.strategyLabel }}</el-tag>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="默认模型">
                  <el-tag size="small" type="info">{{ ctx.config.defaultModel }}</el-tag>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="基础配置">
                  <el-button type="primary" link @click="$router.push('/api-management')">
                    前往 API 管理设置
                  </el-button>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="自动保存">
                  <el-switch v-model="ctx.autoSaveEnabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="最近保存">
                  <span class="last-saved">{{ ctx.lastSavedLabel }}</span>
                </el-form-item>
              </el-col>
            </el-row>

            <el-divider content-position="left">0.5B 分类控制器</el-divider>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="启用分类器">
                  <el-switch v-model="ctx.classifierConfig.enabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="Shadow 模式">
                  <el-switch v-model="ctx.classifierConfig.shadow_mode" active-text="开启" inactive-text="关闭" />
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
                <el-form-item label="运行模型">
                  <el-tag size="small" :type="ctx.classifierHealth.healthy ? 'success' : 'warning'">
                    {{ ctx.classifierConfig.active_model || '-' }}
                  </el-tag>
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="超时(ms)">
                  <el-input-number v-model="ctx.classifierConfig.timeout_ms" :min="50" :max="10000" :step="10" style="width: 180px" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="置信度阈值">
                  <el-slider v-model="ctx.classifierConfidencePercent" :min="30" :max="95" :step="1" show-input style="max-width: 320px" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="14">
                <el-form-item label="手动切换模型">
                  <el-select v-model="ctx.classifierSwitchModel" filterable clearable style="width: 100%" placeholder="选择分类模型">
                    <el-option v-for="model in ctx.classifierConfig.candidate_models" :key="model" :label="model" :value="model" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="10">
                <el-form-item label="操作">
                  <el-button type="primary" :loading="ctx.classifierSaving" @click="ctx.saveClassifierConfig">保存配置</el-button>
                  <el-button :loading="ctx.classifierSwitching" @click="ctx.switchClassifierModel">切换模型</el-button>
                  <el-button :loading="ctx.classifierModelsLoading" @click="ctx.loadClassifierModels">刷新模型列表</el-button>
                  <el-button link @click="ctx.loadClassifierHealth">健康检查</el-button>
                </el-form-item>
              </el-col>
            </el-row>

            <el-alert
              :title="`健康状态: ${ctx.classifierHealth.message || 'unknown'} (延迟 ${ctx.formatDuration(ctx.classifierHealth.latency_ms)})`"
              :type="ctx.classifierHealth.healthy ? 'success' : 'warning'"
              :closable="false"
              style="margin-bottom: 16px"
            />

            <el-descriptions :column="2" border size="small" style="margin-bottom: 16px">
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

            <el-divider content-position="left">任务类型模型映射</el-divider>
            <el-alert type="info" :closable="false" style="margin-bottom: 16px">
              <template #title>
                开启后将根据任务类型自动选择对应模型，关闭则使用默认策略
              </template>
            </el-alert>
            <el-row :gutter="16">
              <el-col :span="12" v-for="task in ctx.taskTypes" :key="task.type">
                <div class="task-model-item">
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
              </el-col>
            </el-row>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="8">
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

        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>任务类型分布</span>
            </div>
          </template>

          <div class="task-types">
            <div v-for="task in ctx.taskTypes" :key="task.type" class="task-type-item">
              <div class="task-row">
                <span class="task-name">{{ task.name }}</span>
                <span class="task-percent">{{ task.percentage }}%</span>
              </div>
              <el-progress
                :percentage="task.percentage"
                :color="task.color"
                :stroke-width="8"
                :show-text="false"
              />
            </div>
          </div>
        </el-card>

        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>效果评估</span>
              <el-button type="primary" link size="small" @click="ctx.reloadPolicyPanel">
                <el-icon><Refresh /></el-icon>
                刷新
              </el-button>
            </div>
          </template>

          <div class="feedback-stats">
            <div class="feedback-item">
              <span class="label">总反馈数</span>
              <span class="value">{{ ctx.feedbackStats.total }}</span>
            </div>
            <div class="feedback-item">
              <span class="label">好评率</span>
              <span class="value positive">{{ ctx.feedbackStats.positiveRate }}%</span>
            </div>
            <div class="feedback-item">
              <span class="label">追踪模型数</span>
              <span class="value">{{ ctx.feedbackStats.modelsTracked }}</span>
            </div>
            <div class="feedback-item">
              <span class="label">平均评分</span>
              <span class="value">{{ ctx.feedbackStats.avgRating.toFixed(1) }}</span>
            </div>
          </div>

          <el-button type="primary" style="width: 100%; margin-top: 16px" @click="ctx.triggerOptimization">
            <el-icon><MagicStick /></el-icon>
            触发自动优化
          </el-button>
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
.task-row {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.task-model-item {
  margin-bottom: 12px;
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

.task-percent {
  font-size: 14px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
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

.task-types {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.feedback-stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.feedback-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.value {
  font-size: 20px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.value.positive {
  color: #67c23a;
}

.last-saved {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
