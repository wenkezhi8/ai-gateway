<template>
  <TabStateView
    :state="ctx.panelState.vector"
    :error-text="ctx.panelError.vector"
    empty-text="向量缓存未启用或暂无状态"
    @retry="ctx.reloadVectorPanel"
  >
    <el-row :gutter="24" class="row-gap">
      <el-col :span="24">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>向量模型管理</span>
              <div class="header-actions">
                <el-button size="small" :loading="ctx.dualModelSaving" @click="ctx.switchVectorModel">切换模型</el-button>
                <el-button size="small" :loading="ctx.ollamaStarting" @click="ctx.startVectorModel">启动模型</el-button>
                <el-button type="primary" size="small" :loading="ctx.dualModelSaving" @click="ctx.saveDualModelConfigData">保存配置</el-button>
              </div>
            </div>
          </template>

          <el-form label-width="120px">
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

          <el-alert
            :title="`向量Pipeline: ${ctx.vectorPipelineHealth.message || 'unknown'} (延迟 ${ctx.vectorPipelineHealth.embedding_latency_ms}ms)`"
            :type="ctx.vectorPipelineHealth.healthy ? 'success' : 'warning'"
            :closable="false"
          />
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="24" class="row-gap">
      <el-col :span="12">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>向量索引状态</span>
              <div class="header-actions">
                <el-button link :loading="ctx.vectorRefreshing" @click="ctx.reloadVectorPanel">刷新</el-button>
                <el-button type="warning" size="small" plain :loading="ctx.vectorRebuilding" @click="ctx.rebuildVectorCacheIndex">
                  重建索引
                </el-button>
              </div>
            </div>
          </template>

          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="向量启用">
              <el-tag :type="ctx.vectorStats.enabled ? 'success' : 'info'">{{ ctx.vectorStats.enabled ? '已启用' : '未启用' }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="索引名">{{ ctx.vectorStats.index_name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="键前缀">{{ ctx.vectorStats.key_prefix || '-' }}</el-descriptions-item>
            <el-descriptions-item label="向量维度">{{ ctx.vectorStats.dimension || '-' }}</el-descriptions-item>
            <el-descriptions-item label="查询超时">{{ ctx.vectorStats.query_timeout_ms || 0 }} ms</el-descriptions-item>
            <el-descriptions-item label="状态">
              <el-tag :type="ctx.vectorStats.enabled ? 'success' : 'warning'">
                {{ ctx.vectorStats.message || (ctx.vectorStats.enabled ? 'ready' : 'disabled') }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>冷热向量分层</span>
              <div class="header-actions">
                <el-button link :loading="ctx.vectorRefreshing" @click="ctx.reloadVectorPanel">刷新分层状态</el-button>
                <el-button type="warning" size="small" plain :loading="ctx.tierMigrating" @click="ctx.migrateHotToCold">
                  手动迁移
                </el-button>
              </div>
            </div>
          </template>

          <el-form label-position="top" class="compact-form">
            <el-form-item label="冷向量总开关">
              <el-switch
                :model-value="ctx.vectorTierConfig.cold_vector_enabled"
                :loading="ctx.vectorTierConfigSaving"
                @change="(value: boolean) => ctx.saveVectorTierConfigPatch({ cold_vector_enabled: value })"
              />
            </el-form-item>
            <el-form-item label="冷层查询开关">
              <el-switch
                :model-value="ctx.vectorTierConfig.cold_vector_query_enabled"
                :loading="ctx.vectorTierConfigSaving"
                @change="(value: boolean) => ctx.saveVectorTierConfigPatch({ cold_vector_query_enabled: value })"
              />
            </el-form-item>
            <el-form-item label="冷层后端">
              <el-select
                :model-value="ctx.vectorTierConfig.cold_vector_backend"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: string) => ctx.saveVectorTierConfigPatch({ cold_vector_backend: value })"
              >
                <el-option label="SQLite（默认）" value="sqlite" />
                <el-option label="Qdrant" value="qdrant" />
              </el-select>
            </el-form-item>
            <el-form-item label="冷层双写">
              <el-switch
                :model-value="ctx.vectorTierConfig.cold_vector_dual_write_enabled"
                :loading="ctx.vectorTierConfigSaving"
                @change="(value: boolean) => ctx.saveVectorTierConfigPatch({ cold_vector_dual_write_enabled: value })"
              />
            </el-form-item>
            <el-form-item label="响应冷归档开关">
              <el-switch
                :model-value="ctx.vectorTierConfig.cold_archive_enabled"
                :loading="ctx.vectorTierConfigSaving"
                @change="(value: boolean) => ctx.saveVectorTierConfigPatch({ cold_archive_enabled: value })"
              />
            </el-form-item>
            <el-form-item label="归档模式">
              <el-select
                :model-value="ctx.vectorTierConfig.cold_archive_mode"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: string) => ctx.saveVectorTierConfigPatch({ cold_archive_mode: value })"
              >
                <el-option label="仅可复用回答" value="reusable" />
                <el-option label="全部回答" value="all" />
              </el-select>
            </el-form-item>
            <el-form-item label="临期窗口(s)">
              <el-input-number
                :model-value="ctx.vectorTierConfig.cold_archive_near_expiry_seconds"
                :min="10"
                :max="3600"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ cold_archive_near_expiry_seconds: Number(value || 120) })"
              />
            </el-form-item>
            <el-form-item label="归档扫描周期(s)">
              <el-input-number
                :model-value="ctx.vectorTierConfig.cold_archive_scan_interval_seconds"
                :min="5"
                :max="600"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ cold_archive_scan_interval_seconds: Number(value || 30) })"
              />
            </el-form-item>
            <el-form-item label="冷层相似阈值">
              <el-input-number
                :model-value="ctx.vectorTierConfig.cold_vector_similarity_threshold"
                :min="0.5"
                :max="1"
                :step="0.01"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ cold_vector_similarity_threshold: Number(value || 0.92) })"
              />
            </el-form-item>
            <el-form-item label="冷层 TopK">
              <el-input-number
                :model-value="ctx.vectorTierConfig.cold_vector_top_k"
                :min="1"
                :max="20"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ cold_vector_top_k: Number(value || 1) })"
              />
            </el-form-item>
            <el-form-item label="热层高水位(%)">
              <el-input-number
                :model-value="ctx.vectorTierConfig.hot_memory_high_watermark_percent"
                :min="50"
                :max="99"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ hot_memory_high_watermark_percent: Number(value || 75) })"
              />
            </el-form-item>
            <el-form-item label="热层回落目标(%)">
              <el-input-number
                :model-value="ctx.vectorTierConfig.hot_memory_relief_percent"
                :min="30"
                :max="95"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ hot_memory_relief_percent: Number(value || 65) })"
              />
            </el-form-item>
            <el-form-item label="迁移批大小">
              <el-input-number
                :model-value="ctx.vectorTierConfig.hot_to_cold_batch_size"
                :min="10"
                :max="5000"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ hot_to_cold_batch_size: Number(value || 500) })"
              />
            </el-form-item>
            <el-form-item label="扫描周期(s)">
              <el-input-number
                :model-value="ctx.vectorTierConfig.hot_to_cold_interval_seconds"
                :min="5"
                :max="600"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ hot_to_cold_interval_seconds: Number(value || 30) })"
              />
            </el-form-item>
            <el-form-item v-if="ctx.vectorTierConfig.cold_vector_backend === 'qdrant'" label="Qdrant URL">
              <el-input
                v-model="ctx.vectorTierConfig.cold_vector_qdrant_url"
                :disabled="ctx.vectorTierConfigSaving"
                placeholder="http://127.0.0.1:6333"
                @change="(value: string) => ctx.saveVectorTierConfigPatch({ cold_vector_qdrant_url: value || '' })"
              />
            </el-form-item>
            <el-form-item v-if="ctx.vectorTierConfig.cold_vector_backend === 'qdrant'" label="Qdrant API Key">
              <el-input
                v-model="ctx.vectorTierConfig.cold_vector_qdrant_api_key"
                :disabled="ctx.vectorTierConfigSaving"
                type="password"
                show-password
                @change="(value: string) => ctx.saveVectorTierConfigPatch({ cold_vector_qdrant_api_key: value || '' })"
              />
            </el-form-item>
            <el-form-item v-if="ctx.vectorTierConfig.cold_vector_backend === 'qdrant'" label="Qdrant Collection">
              <el-input
                v-model="ctx.vectorTierConfig.cold_vector_qdrant_collection"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: string) => ctx.saveVectorTierConfigPatch({ cold_vector_qdrant_collection: value || '' })"
              />
            </el-form-item>
            <el-form-item v-if="ctx.vectorTierConfig.cold_vector_backend === 'qdrant'" label="Qdrant 超时(ms)">
              <el-input-number
                :model-value="ctx.vectorTierConfig.cold_vector_qdrant_timeout_ms"
                :min="100"
                :max="10000"
                controls-position="right"
                style="width: 100%"
                :disabled="ctx.vectorTierConfigSaving"
                @change="(value: number) => ctx.saveVectorTierConfigPatch({ cold_vector_qdrant_timeout_ms: Number(value || 1500) })"
              />
            </el-form-item>
          </el-form>

          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="在线状态">
              <el-tag :type="ctx.vectorTierStats.enabled ? 'success' : 'warning'">
                {{ ctx.vectorTierStats.enabled ? '已初始化' : '未初始化' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="活跃后端">{{ ctx.vectorTierStats.cold_vector_backend }}</el-descriptions-item>
            <el-descriptions-item label="热层内存">{{ ctx.vectorTierStats.hot_memory_usage_percent.toFixed(2) }}%</el-descriptions-item>
            <el-descriptions-item label="迁移计数">{{ ctx.vectorTierStats.migration_moved }} / 失败 {{ ctx.vectorTierStats.migration_failed }}</el-descriptions-item>
            <el-descriptions-item label="回暖计数">{{ ctx.vectorTierStats.promote_success }} / 失败 {{ ctx.vectorTierStats.promote_failed }}</el-descriptions-item>
            <el-descriptions-item label="归档入队">{{ ctx.vectorTierStats.archive_enqueued }} / 队列 {{ ctx.vectorTierStats.archive_queue_depth }}</el-descriptions-item>
            <el-descriptions-item label="归档结果">{{ ctx.vectorTierStats.archive_succeeded }} / 失败 {{ ctx.vectorTierStats.archive_failed }}</el-descriptions-item>
            <el-descriptions-item label="归档模式">{{ ctx.vectorTierStats.cold_archive_mode }}</el-descriptions-item>
            <el-descriptions-item label="归档异常">{{ ctx.vectorTierStats.archive_last_error || '-' }}</el-descriptions-item>
            <el-descriptions-item label="状态信息">{{ ctx.vectorTierStats.message || '-' }}</el-descriptions-item>
          </el-descriptions>

          <div class="header-actions" style="margin-top: 16px">
            <el-input
              v-model="ctx.promoteCacheKey"
              placeholder="输入 cache_key 手动回暖"
              clearable
            />
            <el-button type="primary" :loading="ctx.tierPromoting" @click="ctx.promoteToHotTier">
              手动回暖
            </el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="24" class="row-gap">
      <el-col :span="24">
        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>Pipeline 健康检查</span>
              <el-button link :loading="ctx.vectorRefreshing" @click="ctx.reloadVectorPanel">刷新</el-button>
            </div>
          </template>

          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="整体状态">
              <el-tag :type="ctx.vectorPipelineHealth.healthy ? 'success' : 'danger'">
                {{ ctx.vectorPipelineHealth.healthy ? '健康' : '异常' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="说明">{{ ctx.vectorPipelineHealth.message }}</el-descriptions-item>
            <el-descriptions-item label="Embedding耗时">{{ ctx.vectorPipelineHealth.embedding_latency_ms }} ms</el-descriptions-item>
            <el-descriptions-item label="实际维度">{{ ctx.vectorPipelineHealth.embedding_dimension_actual }}</el-descriptions-item>
            <el-descriptions-item label="索引维度">{{ ctx.vectorPipelineHealth.vector_index_dimension }}</el-descriptions-item>
            <el-descriptions-item label="维度匹配">
              <el-tag :type="ctx.vectorPipelineHealth.dimension_match ? 'success' : 'warning'">
                {{ ctx.vectorPipelineHealth.dimension_match ? '匹配' : '不匹配' }}
              </el-tag>
            </el-descriptions-item>
          </el-descriptions>
        </el-card>

        <el-card shadow="never" class="page-card">
          <template #header>
            <div class="card-header">
              <span>Pipeline 在线测试</span>
              <el-button type="primary" size="small" :loading="ctx.vectorPipelineTesting" @click="ctx.runVectorPipelineTest">
                执行测试
              </el-button>
            </div>
          </template>

          <el-form label-position="top" class="compact-form">
            <el-form-item label="测试文本">
              <el-input v-model="ctx.vectorPipelineTestForm.query" type="textarea" :rows="3" placeholder="输入用于向量检索测试的文本" />
            </el-form-item>
            <el-form-item label="任务类型">
              <el-input v-model="ctx.vectorPipelineTestForm.task_type" />
            </el-form-item>
            <el-form-item label="Top K">
              <el-input-number v-model="ctx.vectorPipelineTestForm.top_k" :min="1" :max="20" controls-position="right" style="width: 100%" />
            </el-form-item>
            <el-form-item label="最小相似度">
              <el-input-number v-model="ctx.vectorPipelineTestForm.min_similarity" :min="0.1" :max="1" :step="0.01" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-form>

          <el-empty v-if="!ctx.vectorPipelineTestResult" description="执行测试后显示结果" />
          <template v-else>
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="任务类型">{{ ctx.vectorPipelineTestResult.task_type }}</el-descriptions-item>
              <el-descriptions-item label="归一化查询">{{ ctx.vectorPipelineTestResult.normalized_query }}</el-descriptions-item>
              <el-descriptions-item label="Standard Key">{{ ctx.vectorPipelineTestResult.standard_key }}</el-descriptions-item>
              <el-descriptions-item label="Embedding 维度">{{ ctx.vectorPipelineTestResult.embedding_dimension }}</el-descriptions-item>
              <el-descriptions-item label="Embedding 耗时">{{ ctx.vectorPipelineTestResult.embedding_latency_ms }} ms</el-descriptions-item>
              <el-descriptions-item label="检索耗时">{{ ctx.vectorPipelineTestResult.vector_search_latency }} ms</el-descriptions-item>
            </el-descriptions>

            <el-table :data="ctx.vectorPipelineTestResult.hits || []" size="small" style="margin-top: 12px">
              <el-table-column prop="cache_key" label="Cache Key" min-width="260" show-overflow-tooltip />
              <el-table-column prop="intent" label="Intent" width="120" />
              <el-table-column prop="similarity" label="相似度" width="120">
                <template #default="{ row }">
                  {{ Number(row.similarity || 0).toFixed(4) }}
                </template>
              </el-table-column>
            </el-table>
          </template>
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
.row-gap {
  margin-bottom: 20px;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.compact-form {
  :deep(.el-form-item) {
    margin-bottom: 12px;
  }
}
</style>
