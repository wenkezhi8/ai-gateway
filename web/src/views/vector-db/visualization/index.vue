<template>
  <div class="vector-visualization-page">
    <el-card shadow="never" class="toolbar-card">
      <div class="toolbar-row">
        <div>
          <h2>向量可视化</h2>
          <p>采样 Collection 中的向量点并展示二维散点图</p>
        </div>
        <div class="toolbar-actions">
          <el-button @click="loadScatter" :loading="loading">刷新</el-button>
        </div>
      </div>

      <el-form :model="filters" inline>
        <el-form-item label="Collection" required>
          <el-input v-model="filters.collection_name" placeholder="如 docs" style="width: 220px" />
        </el-form-item>
        <el-form-item label="sample_size">
          <el-input-number v-model="filters.sample_size" :min="1" :max="1000" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="loadScatter">加载散点图</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never" class="content-card">
      <template v-if="loading">
        <el-skeleton :rows="5" animated />
      </template>
      <template v-else-if="error">
        <el-empty description="可视化数据加载失败">
          <el-button type="primary" @click="loadScatter">重试</el-button>
        </el-empty>
      </template>
      <template v-else-if="points.length === 0">
        <el-empty description="暂无可视化数据" />
      </template>
      <template v-else>
        <div class="chart-title">散点图（共 {{ points.length }} 点）</div>
        <div class="scatter-stage" aria-label="散点图">
          <div
            v-for="point in points"
            :key="point.id"
            class="scatter-dot"
            :style="dotStyle(point.x, point.y)"
            :title="`${point.label} (${point.x}, ${point.y})`"
          />
        </div>
        <el-table :data="points.slice(0, 20)" stripe>
          <el-table-column prop="id" label="ID" min-width="180" />
          <el-table-column prop="label" label="标签" min-width="180" />
          <el-table-column prop="x" label="X" width="120" />
          <el-table-column prop="y" label="Y" width="120" />
          <el-table-column prop="score" label="Score" width="120" />
        </el-table>
      </template>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getVectorScatterData, type VectorScatterPoint } from '@/api/vector-db-domain'

const loading = ref(false)
const error = ref(false)
const points = ref<VectorScatterPoint[]>([])

const filters = reactive({
  collection_name: 'docs',
  sample_size: 200
})

async function loadScatter() {
  if (!filters.collection_name.trim()) {
    ElMessage.warning('请输入 Collection 名称')
    return
  }
  loading.value = true
  error.value = false
  try {
    const data = await getVectorScatterData(filters.collection_name, filters.sample_size)
    points.value = data.points || []
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

function dotStyle(x: number, y: number) {
  const left = Math.max(0, Math.min(100, (x + 1) * 50))
  const top = Math.max(0, Math.min(100, (1 - (y + 1) / 2) * 100))
  return {
    left: `${left}%`,
    top: `${top}%`
  }
}

void loadScatter()
</script>

<style scoped>
.vector-visualization-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.toolbar-row {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
}

.toolbar-row h2 {
  margin: 0;
}

.toolbar-row p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
}

.chart-title {
  margin-bottom: 8px;
  font-weight: 600;
}

.scatter-stage {
  position: relative;
  height: 320px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  margin-bottom: 12px;
  background: linear-gradient(135deg, #f8fbff 0%, #eef6ff 100%);
}

.scatter-dot {
  position: absolute;
  width: 8px;
  height: 8px;
  border-radius: 999px;
  background: #409eff;
  transform: translate(-50%, -50%);
  box-shadow: 0 0 0 2px rgba(64, 158, 255, 0.2);
}
</style>
