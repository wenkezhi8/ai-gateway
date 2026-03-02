import { request } from './request'
import { unwrapEnvelope } from './envelope'

export interface OllamaStatus {
  installed: boolean
  running: boolean
  model: string
  model_installed: boolean
  models: string[]
  running_models: string[]
  running_model_details: Array<{ name: string; size_vram: number }>
  running_vram_bytes_total: number
  running_model: string
  keep_alive_disabled: boolean
  message: string
  os: string
}

export async function getOllamaStatus(model: string) {
  const raw = await request.get(`/admin/router/ollama/status?model=${encodeURIComponent(model)}`)
  return unwrapEnvelope<OllamaStatus>(raw, { allowPlain: true })
}

export async function installOllama() {
  const raw = await request.post('/admin/router/ollama/install')
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function startOllama() {
  const raw = await request.post('/admin/router/ollama/start')
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function stopOllama() {
  const raw = await request.post('/admin/router/ollama/stop')
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function pullModel(model: string) {
  const raw = await request.post('/admin/router/ollama/pull', { model })
  return unwrapEnvelope(raw, { allowPlain: true })
}

export async function deleteModel(model: string) {
  const raw = await request.post('/admin/router/ollama/delete', { model })
  return unwrapEnvelope(raw, { allowPlain: true })
}
