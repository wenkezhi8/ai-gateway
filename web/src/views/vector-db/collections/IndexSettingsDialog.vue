<template>
  <el-dialog :model-value="modelValue" title="索引配置" width="520px" @close="emit('update:modelValue', false)">
    <el-form :model="localForm" label-width="140px">
      <el-form-item label="索引类型">
        <el-select v-model="localForm.index_type" style="width: 220px">
          <el-option label="hnsw" value="hnsw" />
          <el-option label="ivf" value="ivf" />
        </el-select>
      </el-form-item>
      <el-form-item label="HNSW M">
        <el-input-number v-model="localForm.hnsw_m" :min="1" :max="256" />
      </el-form-item>
      <el-form-item label="HNSW EF Construct">
        <el-input-number v-model="localForm.hnsw_ef_construct" :min="1" :max="4096" />
      </el-form-item>
      <el-form-item label="IVF NList">
        <el-input-number v-model="localForm.ivf_nlist" :min="1" :max="65536" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="loading" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import type { VectorCollection } from '@/api/vector-db-domain'

const props = defineProps<{
  modelValue: boolean
  loading: boolean
  collection: VectorCollection | null
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: boolean): void
  (
    event: 'save',
    payload: {
      index_type: string
      hnsw_m: number
      hnsw_ef_construct: number
      ivf_nlist: number
    }
  ): void
}>()

const localForm = reactive({
  index_type: 'hnsw',
  hnsw_m: 16,
  hnsw_ef_construct: 100,
  ivf_nlist: 1024
})

watch(
  () => props.collection,
  (value) => {
    if (!value) return
    localForm.index_type = value.index_type || 'hnsw'
    localForm.hnsw_m = value.hnsw_m || 16
    localForm.hnsw_ef_construct = value.hnsw_ef_construct || 100
    localForm.ivf_nlist = value.ivf_nlist || 1024
  },
  { immediate: true }
)

function handleSave() {
  emit('save', {
    index_type: localForm.index_type,
    hnsw_m: localForm.hnsw_m,
    hnsw_ef_construct: localForm.hnsw_ef_construct,
    ivf_nlist: localForm.ivf_nlist
  })
}
</script>
