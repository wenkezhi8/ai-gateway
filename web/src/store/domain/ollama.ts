import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import {
  deleteModel,
  getOllamaStatus,
  installOllama,
  pullModel,
  startOllama,
  stopOllama,
  type OllamaStatus
} from '@/api/ollama-domain'
import { ROUTING_OLLAMA_DEFAULT_MODEL } from '../../constants/routing'

function normalizeError(err: unknown): string {
  if (err instanceof Error && err.message) return err.message
  return '请求失败'
}

export const useOllamaStore = defineStore('ollama-domain', () => {
  const model = ref(ROUTING_OLLAMA_DEFAULT_MODEL)
  const status = ref<OllamaStatus | null>(null)
  const loading = ref(false)
  const operating = ref(false)
  const error = ref('')

  const models = computed(() => status.value?.models ?? [])
  const runningModels = computed(() => status.value?.running_models ?? [])
  const runningModelDetails = computed(() => status.value?.running_model_details ?? [])

  async function refreshStatus() {
    loading.value = true
    error.value = ''
    try {
      status.value = await getOllamaStatus(model.value)
    } catch (err) {
      error.value = normalizeError(err)
    } finally {
      loading.value = false
    }
  }

  async function runAndRefresh(action: () => Promise<unknown>) {
    operating.value = true
    error.value = ''
    try {
      await action()
      await refreshStatus()
      return { success: true }
    } catch (err) {
      const message = normalizeError(err)
      error.value = message
      return { success: false, message }
    } finally {
      operating.value = false
    }
  }

  async function install() {
    return runAndRefresh(() => installOllama())
  }

  async function start() {
    return runAndRefresh(() => startOllama())
  }

  async function stop() {
    return runAndRefresh(() => stopOllama())
  }

  async function pull(targetModel: string) {
    return runAndRefresh(() => pullModel(targetModel.trim()))
  }

  async function remove(targetModel: string) {
    return runAndRefresh(() => deleteModel(targetModel.trim()))
  }

  return {
    model,
    status,
    loading,
    operating,
    error,
    models,
    runningModels,
    runningModelDetails,
    refreshStatus,
    install,
    start,
    stop,
    pull,
    remove
  }
})
