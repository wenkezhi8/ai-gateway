<template>
  <TabStateView
    :state="ctx.panelState.models"
    :error-text="ctx.panelError.models"
    empty-text="暂无模型评分数据"
    @retry="ctx.reloadModelsPanel"
  >
    <el-row :gutter="24">
      <el-col :span="12">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>Intent Engine（本地意图+向量）</span>
              <el-button link @click="ctx.loadIntentEngineHealthData">健康检查</el-button>
            </div>
          </template>

          <el-form label-width="120px">
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="启用">
                  <el-switch v-model="ctx.intentEngineConfig.enabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="语言">
                  <el-input v-model="ctx.intentEngineConfig.language" placeholder="zh-CN" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="服务地址">
                  <el-input v-model="ctx.intentEngineConfig.base_url" placeholder="http://127.0.0.1:18566" />
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="超时(ms)">
                  <el-input-number v-model="ctx.intentEngineConfig.timeout_ms" :min="100" :max="10000" :step="100" style="width: 140px" />
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="向量维度">
                  <el-input-number v-model="ctx.intentEngineConfig.expected_dimension" :min="64" :max="4096" :step="64" style="width: 140px" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-button type="primary" :loading="ctx.intentEngineSaving" @click="ctx.saveIntentEngineConfigData">保存 Intent Engine</el-button>
          </el-form>

          <el-alert
            :title="`Intent Engine: ${ctx.intentEngineHealth.message || 'unknown'} (延迟 ${ctx.formatDuration(ctx.intentEngineHealth.latency_ms)})`"
            :type="ctx.intentEngineHealth.healthy ? 'success' : 'warning'"
            :closable="false"
            style="margin-top: 16px"
          />
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>模型评分管理</span>
              <el-input
                v-model="ctx.modelSearch"
                placeholder="搜索模型..."
                style="width: 200px"
                clearable
              >
                <template #prefix>
                  <el-icon><Search /></el-icon>
                </template>
              </el-input>
            </div>
          </template>

          <el-table :data="ctx.filteredModels" stripe max-height="420">
            <el-table-column prop="model" label="模型" width="180" fixed />
            <el-table-column prop="provider" label="服务商" width="100" />
            <el-table-column label="效果" width="120">
              <template #default="{ row }">
                <div class="score-cell">
                  <el-progress
                    :percentage="row.quality_score"
                    :color="ctx.getScoreColor(row.quality_score)"
                    :stroke-width="8"
                    :show-text="false"
                  />
                  <span class="score-text">{{ row.quality_score }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="速度" width="120">
              <template #default="{ row }">
                <div class="score-cell">
                  <el-progress
                    :percentage="row.speed_score"
                    :color="ctx.getScoreColor(row.speed_score)"
                    :stroke-width="8"
                    :show-text="false"
                  />
                  <span class="score-text">{{ row.speed_score }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="成本" width="120">
              <template #default="{ row }">
                <div class="score-cell">
                  <el-progress
                    :percentage="row.cost_score"
                    :color="ctx.getScoreColor(row.cost_score)"
                    :stroke-width="8"
                    :show-text="false"
                  />
                  <span class="score-text">{{ row.cost_score }}</span>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="综合" width="80" align="center">
              <template #default="{ row }">
                <el-tag :type="ctx.getScoreTagType(ctx.calculateCompositeScore(row))" size="small">
                  {{ ctx.calculateCompositeScore(row) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="80" align="center">
              <template #default="{ row }">
                <el-switch v-model="row.enabled" size="small" @change="ctx.toggleModelEnabled(row)" />
              </template>
            </el-table-column>
          </el-table>
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
.score-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.score-cell :deep(.el-progress) {
  flex: 1;
}

.score-text {
  width: 24px;
  text-align: right;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
