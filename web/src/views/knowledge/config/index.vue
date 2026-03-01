<template>
  <div class="knowledge-config-page">
    <el-card>
      <template #header>
        <div>
          <div class="title">知识库配置</div>
          <div class="subtitle">配置向量后端、分块策略与检索参数。</div>
        </div>
      </template>

      <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" />
      <el-skeleton v-else-if="loading" :rows="8" animated />

      <el-form v-else :model="form" label-width="160px">
        <el-form-item label="向量后端">
          <el-select v-model="form.vector_backend" style="width: 220px">
            <el-option label="SQLite（默认）" value="sqlite" />
            <el-option label="Qdrant" value="qdrant" />
          </el-select>
        </el-form-item>
        <el-form-item label="Embedding 模型">
          <el-input v-model="form.embedding_model" style="width: 320px" />
        </el-form-item>
        <el-form-item label="分块类型">
          <el-select v-model="form.chunking_strategy.type" style="width: 220px">
            <el-option label="fixed_size" value="fixed_size" />
          </el-select>
        </el-form-item>
        <el-form-item label="分块大小">
          <el-input-number v-model="form.chunking_strategy.chunk_size" :min="100" :max="2000" />
        </el-form-item>
        <el-form-item label="分块重叠">
          <el-input-number v-model="form.chunking_strategy.chunk_overlap" :min="0" :max="300" />
        </el-form-item>
        <el-form-item label="检索 TopK">
          <el-input-number v-model="form.retrieval.top_k" :min="1" :max="20" />
        </el-form-item>
        <el-form-item label="相似度阈值">
          <el-slider v-model="form.retrieval.similarity_threshold" :min="0" :max="1" :step="0.1" style="width: 320px" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="saving" @click="save">保存</el-button>
          <el-button @click="load">刷新</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card style="margin-top: 12px">
      <template #header>
        <div class="title">知识库集合</div>
      </template>
      <el-empty v-if="collections.length === 0" description="暂无集合" />
      <el-table v-else :data="collections" stripe>
        <el-table-column prop="name" label="集合名" />
        <el-table-column prop="document_count" label="文档数" width="120" />
        <el-table-column prop="chunk_count" label="分块数" width="120" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { getKnowledgeConfig, updateKnowledgeConfig, type KnowledgeConfig } from '@/api/knowledge-domain'

const loading = ref(false)
const saving = ref(false)
const error = ref('')

const form = ref<KnowledgeConfig>({
  vector_backend: 'sqlite',
  embedding_model: 'nomic-embed-text',
  chunking_strategy: { type: 'fixed_size', chunk_size: 500, chunk_overlap: 50 },
  retrieval: { top_k: 5, similarity_threshold: 0.7 },
  collections: []
})

const collections = ref<Array<{ id: string; name: string; document_count: number; chunk_count: number }>>([])

onMounted(() => {
  load()
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const cfg = await getKnowledgeConfig()
    form.value = {
      vector_backend: cfg.vector_backend,
      embedding_model: cfg.embedding_model,
      chunking_strategy: { ...cfg.chunking_strategy },
      retrieval: { ...cfg.retrieval },
      collections: cfg.collections || []
    }
    collections.value = cfg.collections || []
  } catch (e: any) {
    error.value = e?.message || '加载配置失败'
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await updateKnowledgeConfig({
      vector_backend: form.value.vector_backend,
      embedding_model: form.value.embedding_model,
      chunking_strategy: form.value.chunking_strategy,
      retrieval: form.value.retrieval
    })
    ElMessage.success('配置已保存')
    await load()
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.knowledge-config-page {
  padding: 8px;
}

.title {
  font-size: 18px;
  font-weight: 600;
}

.subtitle {
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
</style>
