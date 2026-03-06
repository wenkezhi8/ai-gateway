<template>
  <TabStateView
    :state="ctx.panelState.models"
    :error-text="ctx.panelError.models"
    empty-text="暂无意图路由配置数据"
    @retry="ctx.reloadModelsPanel"
  >
    <el-row :gutter="24">
      <el-col :span="24">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>意图模型配置</span>
              <div class="header-actions">
                <el-button type="primary" size="small" :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">保存基础设置</el-button>
                <el-button :loading="ctx.classifierModelsLoading" @click="ctx.loadClassifierModels">刷新模型</el-button>
              </div>
            </div>
          </template>

          <el-alert
            type="info"
            :closable="false"
            show-icon
            class="section-alert"
            title="新手先决定是否启用分类器，以及使用哪个主模型。"
          />

          <section class="panel-section">
            <div class="section-title">基础设置</div>
            <el-form label-width="120px">
              <el-row :gutter="24">
                <el-col :span="12">
                  <el-form-item label="启用意图分类器">
                    <el-switch v-model="ctx.dualModelConfig.classifier_enabled" active-text="开启" inactive-text="关闭" />
                  </el-form-item>
                </el-col>
                <el-col :span="12">
                  <el-form-item label="主模型">
                    <el-input v-model="ctx.dualModelConfig.classifier_active_model" placeholder="qwen2.5:0.5b-instruct" />
                  </el-form-item>
                </el-col>
              </el-row>
              <el-row :gutter="24">
                <el-col :span="24">
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
            </el-form>

            <div class="header-actions section-actions">
              <el-button type="primary" :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">保存基础设置</el-button>
              <el-button type="warning" :loading="ctx.classifierSwitching" @click="ctx.startClassifierModel">启动模型</el-button>
              <el-button link @click="ctx.loadClassifierHealth">健康检查</el-button>
            </div>
          </section>

          <section class="panel-section">
            <div class="section-title">验证/状态</div>
            <el-alert
              :title="`分类器健康: ${ctx.classifierHealth.message || 'unknown'} (延迟 ${ctx.formatDuration(ctx.classifierHealth.latency_ms)})`"
              :type="ctx.classifierHealth.healthy ? 'success' : 'warning'"
              :closable="false"
              class="section-alert"
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
          </section>

          <section class="panel-section advanced-section">
            <div class="section-title">高级设置</div>
            <el-collapse>
              <el-collapse-item name="advanced" title="高级设置">
                <el-form label-width="120px" class="advanced-form">
                  <el-row :gutter="24">
                    <el-col :span="12">
                      <el-form-item label="服务地址">
                        <el-input v-model="ctx.dualModelConfig.classifier_base_url" placeholder="http://127.0.0.1:11434" />
                      </el-form-item>
                    </el-col>
                    <el-col :span="12">
                      <el-form-item label="超时(ms)">
                        <el-input-number v-model="ctx.dualModelConfig.classifier_timeout_ms" :min="50" :max="10000" :step="10" style="width: 100%" />
                      </el-form-item>
                    </el-col>
                  </el-row>
                  <el-row :gutter="24">
                    <el-col :span="12">
                      <el-form-item label="Shadow 模式">
                        <el-switch v-model="ctx.classifierConfig.shadow_mode" active-text="开启" inactive-text="关闭" />
                      </el-form-item>
                    </el-col>
                    <el-col :span="12">
                      <el-form-item label="置信度阈值">
                        <el-slider v-model="ctx.classifierConfidencePercent" :min="30" :max="95" :step="1" show-input />
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

                <div class="header-actions section-actions advanced-actions">
                  <el-button type="primary" :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">保存运行参数</el-button>
                  <el-button :loading="ctx.classifierSaving" @click="ctx.saveClassifierConfig">保存高级策略</el-button>
                  <el-button :loading="ctx.classifierSwitching" @click="ctx.switchClassifierModel">切换模型</el-button>
                </div>

                <el-row :gutter="24">
                  <el-col :span="12">
                    <el-card shadow="never" class="page-card nested-card">
                      <template #header>
                        <div class="card-header">
                          <span>任务类型模型映射</span>
                          <el-button type="primary" size="small" :loading="ctx.saving" @click="ctx.saveTaskMapping">保存映射</el-button>
                        </div>
                      </template>

                      <el-alert type="info" :closable="false" class="section-alert">
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
                  </el-col>

                  <el-col :span="12">
                    <el-card shadow="never" class="page-card nested-card">
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

                      <el-alert type="info" :closable="false" show-icon class="section-alert" style="margin-top: 16px">
                        <template #title>
                          当小模型无法处理时，自动升级到大模型
                        </template>
                      </el-alert>
                    </el-card>
                  </el-col>
                </el-row>
              </el-collapse-item>
            </el-collapse>
          </section>
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
.panel-section {
  margin-bottom: 20px;
}

.section-title {
  margin-bottom: 12px;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.section-alert {
  margin-bottom: 16px;
}

.section-actions {
  margin-top: 12px;
}

.advanced-actions {
  margin-bottom: 20px;
}

.advanced-form {
  margin-bottom: 16px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.task-model-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.task-model-item {
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
}

.task-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.task-name {
  font-weight: 500;
}

.cascade-levels {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.cascade-level {
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--el-border-color-lighter);
}

.level-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.level-desc {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.level-models {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.model-tag {
  margin-right: 0;
}

.nested-card {
  height: 100%;
  margin-bottom: 0;
}

.advanced-section {
  margin-bottom: 0;
}
</style>
