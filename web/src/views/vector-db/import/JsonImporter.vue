<template>
  <el-form :model="form" label-width="110px" class="import-form">
    <el-form-item label="Collection" required>
      <el-select v-model="form.collection_name" placeholder="选择 Collection" filterable>
        <el-option v-for="item in collections" :key="item" :label="item" :value="item" />
      </el-select>
    </el-form-item>
    <el-form-item label="JSON 文件名" required>
      <el-input v-model="form.file_name" placeholder="如 docs.json" />
    </el-form-item>
    <el-form-item label="文件路径" required>
      <el-input v-model="form.file_path" placeholder="如 /tmp/docs.json" />
    </el-form-item>
    <el-form-item label="文件大小" required>
      <el-input-number v-model="form.file_size" :min="1" :step="1024" />
    </el-form-item>
    <el-form-item label="记录数" required>
      <el-input-number v-model="form.total_records" :min="1" />
    </el-form-item>
    <el-form-item>
      <el-button type="primary" :loading="submitting" @click="submit">创建 JSON 导入任务</el-button>
    </el-form-item>
  </el-form>
</template>

<script setup lang="ts">
import { reactive } from 'vue'

interface CreatePayload {
  collection_name: string
  file_name: string
  file_path: string
  file_size: number
  total_records: number
}

const props = defineProps<{ collections: string[]; submitting: boolean }>()
const emit = defineEmits<{ create: [payload: CreatePayload] }>()

const form = reactive<CreatePayload>({
  collection_name: '',
  file_name: 'docs.json',
  file_path: '/tmp/docs.json',
  file_size: 1024,
  total_records: 100
})

function submit() {
  emit('create', { ...form })
}
</script>

<style scoped>
.import-form {
  max-width: 640px;
}
</style>
