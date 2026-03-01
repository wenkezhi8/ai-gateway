<template>
  <TabStateView
    :state="ctx.panelState.models"
    :error-text="ctx.panelError.models"
    empty-text="暂无双模型配置数据"
    @retry="ctx.reloadModelsPanel"
  >
    <el-row :gutter="24">
      <el-col :span="24">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>Ollama 双模型管理（意图 + 向量）</span>
              <el-button type="primary" size="small" :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">
                保存双模型配置
              </el-button>
            </div>
          </template>

          <el-form label-width="120px">
            <el-divider content-position="left">意图模型（分类器）</el-divider>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="启用">
                  <el-switch v-model="ctx.dualModelConfig.classifier_enabled" active-text="开启" inactive-text="关闭" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="超时(ms)">
                  <el-input-number v-model="ctx.dualModelConfig.classifier_timeout_ms" :min="100" :max="10000" :step="100" style="width: 100%" />
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
              <el-col :span="24">
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

            <el-divider content-position="left">向量模型（Embedding）</el-divider>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="启用向量 Pipeline">
                  <el-switch v-model="ctx.dualModelConfig.vector_pipeline_enabled" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="启用回写">
                  <el-switch v-model="ctx.dualModelConfig.vector_writeback_enabled" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="24">
                <el-form-item label="向量服务地址">
                  <el-input v-model="ctx.dualModelConfig.vector_ollama_base_url" placeholder="http://127.0.0.1:11434" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="Embedding 模型">
                  <el-input v-model="ctx.dualModelConfig.vector_ollama_embedding_model" placeholder="nomic-embed-text" />
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="维度">
                  <el-input-number v-model="ctx.dualModelConfig.vector_ollama_embedding_dimension" :min="1" :step="1" style="width: 100%" />
                </el-form-item>
              </el-col>
              <el-col :span="6">
                <el-form-item label="超时(ms)">
                  <el-input-number v-model="ctx.dualModelConfig.vector_ollama_embedding_timeout_ms" :min="100" :max="10000" :step="100" style="width: 100%" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-row :gutter="24">
              <el-col :span="12">
                <el-form-item label="端点模式">
                  <el-select v-model="ctx.dualModelConfig.vector_ollama_endpoint_mode" style="width: 100%">
                    <el-option label="auto" value="auto" />
                    <el-option label="embed (/api/embed)" value="embed" />
                    <el-option label="embeddings (/api/embeddings)" value="embeddings" />
                  </el-select>
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>

          <el-row :gutter="12">
            <el-col :span="12">
              <el-alert
                :title="`分类器健康: ${ctx.classifierHealth.message || 'unknown'} (延迟 ${ctx.formatDuration(ctx.classifierHealth.latency_ms)})`"
                :type="ctx.classifierHealth.healthy ? 'success' : 'warning'"
                :closable="false"
              />
            </el-col>
            <el-col :span="12">
              <el-alert
                :title="`向量Pipeline: ${ctx.vectorPipelineHealth.message || 'unknown'} (延迟 ${ctx.vectorPipelineHealth.embedding_latency_ms}ms)`"
                :type="ctx.vectorPipelineHealth.healthy ? 'success' : 'warning'"
                :closable="false"
              />
            </el-col>
          </el-row>
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
