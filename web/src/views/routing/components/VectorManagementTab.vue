<template>
  <TabStateView
    :state="ctx.panelState.vector"
    :error-text="ctx.panelError.vector"
    empty-text="向量缓存未启用或暂无状态"
    @retry="ctx.reloadVectorPanel"
  >
    <el-row :gutter="24">
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
              <span>冷热向量分层（只读）</span>
            </div>
          </template>

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
            <el-descriptions-item label="状态信息">{{ ctx.vectorTierStats.message || '-' }}</el-descriptions-item>
          </el-descriptions>

          <el-alert
            title="该页仅提供观测与索引重建操作，迁移/回暖管理请前往缓存管理页"
            type="info"
            :closable="false"
            style="margin-top: 16px"
          />
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
</style>
