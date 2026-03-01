<template>
  <div class="vector-search-page">
    <el-card>
      <template #header>
        <div class="header">向量检索</div>
      </template>

      <el-form :model="form" label-width="110px" class="form-grid">
        <el-form-item label="Collection">
          <el-input v-model="form.collectionName" placeholder="例如 docs" />
        </el-form-item>
        <el-form-item label="TopK">
          <el-input-number v-model="form.topK" :min="1" :max="100" />
        </el-form-item>
        <el-form-item label="最小分数">
          <el-input-number v-model="form.minScore" :min="0" :max="1" :step="0.05" />
        </el-form-item>
        <el-form-item label="向量(JSON)">
          <el-input v-model="form.vectorJSON" type="textarea" :rows="4" placeholder="[0.1, 0.2, 0.3]" />
        </el-form-item>
      </el-form>

      <div class="actions">
        <el-button type="primary" :loading="loading" @click="search">搜索</el-button>
        <el-button :loading="loading" @click="recommend">推荐</el-button>
      </div>

      <el-alert v-if="error" :title="error" type="error" show-icon class="state" />
      <el-empty v-else-if="!loading && rows.length === 0" description="暂无检索结果" class="state" />
      <el-table v-else v-loading="loading" :data="rows" border>
        <el-table-column prop="id" label="ID" min-width="160" />
        <el-table-column prop="score" label="Score" width="120" />
        <el-table-column label="Payload">
          <template #default="scope">
            <pre class="payload">{{ stringifyPayload(scope.row.payload) }}</pre>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { VectorSearchResult } from '@/api/vector-db-domain'
import { recommendVectorCollection, searchVectorCollection } from '@/api/vector-db-domain'

const loading = ref(false)
const error = ref('')
const rows = ref<VectorSearchResult[]>([])

const form = reactive({
  collectionName: 'docs',
  topK: 5,
  minScore: 0,
  vectorJSON: '[0.1, 0.2, 0.3]'
})

function parseVector(): number[] {
  const parsed = JSON.parse(form.vectorJSON)
  if (!Array.isArray(parsed) || parsed.length === 0) {
    throw new Error('向量必须是非空数组')
  }
  return parsed.map((item) => Number(item))
}

async function runQuery(kind: 'search' | 'recommend') {
  loading.value = true
  error.value = ''
  try {
    const vector = parseVector()
    const payload = { top_k: form.topK, min_score: form.minScore, vector }
    const resp = kind === 'search'
      ? await searchVectorCollection(form.collectionName.trim(), payload)
      : await recommendVectorCollection(form.collectionName.trim(), payload)
    rows.value = resp.results || []
  } catch (err) {
    const message = err instanceof Error ? err.message : '查询失败'
    error.value = message
    rows.value = []
  } finally {
    loading.value = false
  }
}

function search() {
  return runQuery('search')
}

function recommend() {
  return runQuery('recommend')
}

function stringifyPayload(payload: Record<string, unknown>) {
  return JSON.stringify(payload || {}, null, 2)
}
</script>

<style scoped>
.vector-search-page {
  padding: 12px;
}

.header {
  font-weight: 600;
}

.form-grid {
  max-width: 860px;
}

.actions {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.state {
  margin: 10px 0;
}

.payload {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
