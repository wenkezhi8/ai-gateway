export interface IntentEngineConfig {
  enabled: boolean
  base_url: string
  timeout_ms: number
  language: string
  expected_dimension: number
}

export interface IntentEngineHealth {
  enabled: boolean
  healthy: boolean
  status?: number
  latency_ms?: number
  message?: string
  [key: string]: unknown
}

