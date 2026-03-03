import { TOKEN_PRICE_USD, normalizeUsageSource, type UsageSource } from './usage-overview'

export interface UsageRow {
  id: string
  accountName: string
  provider: string
  time: string
  timestamp: number
  firstTokenLatency: string
  totalLatency: string
  firstTokenSeconds: number
  totalDurationSeconds: number
  model: string
  taskType: string
  requestType: string
  inferenceIntensity: string
  userAgent: string
  inputTokens: number
  outputTokens: number
  totalTokens: number
  savedTokens: number
  usageSource: UsageSource
  success: boolean
  cacheHit: string
  cost: number
}

export interface UsageLogPayload {
  id?: number | string
  timestamp?: number | string
  model?: string
  task_type?: string
  provider?: string
  service_provider?: string
  account?: string
  type?: string
  request_type?: string
  inference_intensity?: string
  user_agent?: string
  ttft_ms?: number | string
  time_to_first_token?: number | string
  latency_ms?: number | string
  total_duration?: number | string
  input_tokens?: number | string
  output_tokens?: number | string
  total_tokens?: number | string
  tokens?: number | string
  saved_tokens?: number | string
  cache_hit?: boolean
  success?: boolean
  usage_source?: string
}

function toNumber(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string') {
    const parsed = Number(value)
    if (Number.isFinite(parsed)) {
      return parsed
    }
  }
  return 0
}

function formatDateTime(time: number): string {
  const d = new Date(time)
  if (Number.isNaN(d.getTime())) return '-'
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  const h = String(d.getHours()).padStart(2, '0')
  const min = String(d.getMinutes()).padStart(2, '0')
  const s = String(d.getSeconds()).padStart(2, '0')
  return `${y}/${m}/${day} ${h}:${min}:${s}`
}

export function mapUsageLogToRow(
  log: UsageLogPayload,
  accountNameMap: Map<string, string>
): UsageRow {
  const provider = (log.service_provider || log.provider || '').trim()
  const totalTokens = toNumber(log.total_tokens) || toNumber(log.tokens)
  const inputTokens = toNumber(log.input_tokens) || Math.round(totalTokens * 0.6)
  const outputTokens = toNumber(log.output_tokens) || Math.max(0, totalTokens - inputTokens)
  const success = Boolean(log.success)
  const cacheHit = Boolean(log.cache_hit)
  const defaultSavedTokens = cacheHit && success ? totalTokens : 0
  const savedTokens = toNumber(log.saved_tokens) || defaultSavedTokens
  const ttftMs = toNumber(log.time_to_first_token) || toNumber(log.ttft_ms)
  const latencyMs = toNumber(log.total_duration) || toNumber(log.latency_ms)

  return {
    id: String(log.id || log.timestamp || Date.now()),
    accountName: (log.account || accountNameMap.get(provider) || '-').trim() || '-',
    provider: provider || '-',
    time: log.timestamp ? formatDateTime(toNumber(log.timestamp)) : '-',
    timestamp: toNumber(log.timestamp),
    firstTokenLatency: ttftMs > 0 ? `${(ttftMs / 1000).toFixed(2)}s` : '0 ms',
    totalLatency: latencyMs > 0 ? `${(latencyMs / 1000).toFixed(2)}s` : '0 ms',
    firstTokenSeconds: ttftMs / 1000,
    totalDurationSeconds: latencyMs / 1000,
    model: (log.model || '-').trim() || '-',
    taskType: (log.task_type || '-').trim() || '-',
    requestType: (log.request_type || log.type || '-').trim() || '-',
    inferenceIntensity: (log.inference_intensity || '-').trim() || '-',
    userAgent: (log.user_agent || '-').trim() || '-',
    inputTokens,
    outputTokens,
    totalTokens,
    savedTokens,
    usageSource: normalizeUsageSource(log.usage_source),
    success,
    cacheHit: cacheHit ? '命中' : '未命中',
    cost: totalTokens * TOKEN_PRICE_USD
  }
}
