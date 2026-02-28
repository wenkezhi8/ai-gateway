<template>
  <TabStateView
    :state="ctx.panelState.ollama"
    :error-text="ctx.panelError.ollama"
    empty-text="暂无 Ollama 状态数据"
    @retry="ctx.reloadOllamaPanel"
  >
    <el-card shadow="never" class="page-card">
      <template #header>
        <div class="card-header">
          <span>Ollama 一键安装与模型安装</span>
          <el-button link :loading="ctx.ollamaRefreshing" @click="ctx.reloadOllamaPanel">刷新状态</el-button>
        </div>
      </template>

      <el-row :gutter="12" style="margin-bottom: 12px">
        <el-col :span="24">
          <el-tag :type="ctx.ollamaSetup.installed ? 'success' : 'warning'" style="margin-right: 8px">
            Ollama安装: {{ ctx.ollamaSetup.installed ? '已安装' : '未安装' }}
          </el-tag>
          <el-tag :type="ctx.ollamaSetup.running ? 'success' : 'danger'" style="margin-right: 8px">
            服务状态: {{ ctx.ollamaSetup.running ? '运行中' : '未运行' }}
          </el-tag>
          <el-tag :type="ctx.ollamaSetup.model_installed ? 'success' : 'info'">
            模型({{ ctx.ollamaSetup.model }}): {{ ctx.ollamaSetup.model_installed ? '已安装' : '未安装' }}
          </el-tag>
          <el-tag :type="ctx.ollamaSetup.running_model ? 'success' : 'warning'" style="margin-left: 8px">
            当前运行模型: {{ ctx.ollamaSetup.running_model || '无' }}
          </el-tag>
        </el-col>
      </el-row>

      <el-row :gutter="12" style="margin-bottom: 12px">
        <el-col :span="12">
          <el-input v-model="ctx.ollamaModelInput" placeholder="模型名，如 qwen2.5:0.5b-instruct" />
        </el-col>
        <el-col :span="12" class="action-row">
          <el-button :loading="ctx.ollamaInstalling" @click="ctx.installOllama">安装 Ollama</el-button>
          <el-button :loading="ctx.ollamaStarting" type="warning" @click="ctx.startOllama">启动 Ollama</el-button>
          <el-button :loading="ctx.ollamaStopping" type="danger" @click="ctx.stopOllama">停止 Ollama</el-button>
          <el-button :loading="ctx.ollamaPulling" type="primary" @click="ctx.pullOllamaModel">安装模型</el-button>
        </el-col>
      </el-row>

      <el-alert
        v-if="ctx.ollamaSetup.message"
        :title="`Ollama状态: ${ctx.ollamaSetup.message}`"
        :type="ctx.ollamaSetup.running ? 'success' : 'warning'"
        :closable="false"
        style="margin-bottom: 16px"
      />
      <el-alert
        v-if="ctx.ollamaSetup.keep_alive_disabled"
        title="已禁用模型自动休眠（keep_alive=-1）"
        type="success"
        :closable="false"
        style="margin-bottom: 16px"
      />
      <el-alert
        v-if="ctx.ollamaSetup.running_models.length > 0"
        :title="`运行模型列表: ${ctx.ollamaSetup.running_models.join(', ')}`"
        type="info"
        :closable="false"
        style="margin-bottom: 16px"
      />
      <el-alert
        v-if="ctx.ollamaSetup.running_vram_bytes_total > 0"
        :title="`显存占用: ${ctx.formatVramBytes(ctx.ollamaSetup.running_vram_bytes_total)}`"
        type="warning"
        :closable="false"
        style="margin-bottom: 16px"
      />
      <el-descriptions v-if="ctx.ollamaSetup.running_model_details.length > 0" :column="1" border size="small" style="margin-bottom: 16px">
        <el-descriptions-item
          v-for="item in ctx.ollamaSetup.running_model_details"
          :key="item.name"
          :label="`运行模型 ${item.name}`"
        >
          显存占用 {{ ctx.formatVramBytes(item.size_vram) }}
        </el-descriptions-item>
      </el-descriptions>
    </el-card>
  </TabStateView>
</template>

<script setup lang="ts">
import TabStateView from './TabStateView.vue'

defineProps<{
  ctx: any
}>()
</script>

<style scoped lang="scss">
.action-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
