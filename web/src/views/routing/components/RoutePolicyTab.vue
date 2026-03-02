<template>
  <TabStateView
    :state="ctx.panelState.policy"
    :error-text="ctx.panelError.policy"
    empty-text="暂无路由策略数据"
    @retry="ctx.reloadPolicyPanel"
  >
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="card-header">
          <span>路由策略配置</span>
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
      </el-form>
    </el-card>

    <el-row :gutter="24" style="margin-top: 20px">
      <el-col :span="12">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>任务类型分布</span>
              <el-button link @click="ctx.reloadPolicyPanel">刷新</el-button>
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
      </el-col>

      <el-col :span="12">
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
