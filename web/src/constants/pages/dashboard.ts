export const DASHBOARD_FALLBACK_SERIES = {
  timestamps: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00'],
  requests: [0, 0, 0, 0, 0, 0],
  successRates: [0, 0, 0, 0, 0, 0]
} as const

export const DASHBOARD_PROVIDER_COLORS: Record<string, string> = {
  openai: '#007AFF',
  anthropic: '#AF52DE',
  azure: '#00C7BE',
  google: '#FF9500',
  volcengine: '#FF3B30',
  qwen: '#FF6A00',
  ernie: '#2932E1',
  zhipu: '#3657ED',
  hunyuan: '#00A3FF',
  moonshot: '#1A1A1A',
  minimax: '#615CED',
  baichuan: '#0066FF',
  spark: '#E60012',
  deepseek: '#4D6BFE'
}
