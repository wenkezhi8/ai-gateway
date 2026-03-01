export const TOKEN_PRICE_USD = 0.00000035

export interface UsageOverviewRow {
  inputTokens: number
  outputTokens: number
  totalTokens: number
  cacheHit: string
  success: boolean
}

export interface UsageStatsPayload {
  total_requests?: number
  total_tokens?: number
  cache_hits?: number
  cache_misses?: number
  cache_hit_rate?: number
  saved_tokens?: number
  saved_requests?: number
}

export interface UsageOverviewSummary {
  totalRequests: number
  totalTokens: number
  inputTokens: number
  outputTokens: number
  totalCost: number
  cacheHits: number
  cacheMisses: number
  cacheHitRate: number
  savedTokens: number
  savedRequests: number
  savedCost: number
}

export type UsageSource = 'actual' | 'estimated' | ''

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

export function buildUsageOverviewFromRows(rows: UsageOverviewRow[]): UsageOverviewSummary {
  const totalRequests = rows.length
  let totalTokens = 0
  let inputTokens = 0
  let outputTokens = 0
  let cacheHits = 0
  let cacheMisses = 0
  let savedTokens = 0
  let savedRequests = 0

  for (const row of rows) {
    const rowTotalTokens = toNumber(row.totalTokens)
    totalTokens += rowTotalTokens
    inputTokens += toNumber(row.inputTokens)
    outputTokens += toNumber(row.outputTokens)

    if (row.cacheHit === '命中') {
      cacheHits++
      if (row.success) {
        savedTokens += rowTotalTokens
        savedRequests++
      }
    } else if (row.cacheHit === '未命中') {
      cacheMisses++
    }
  }

  const cacheTotal = cacheHits + cacheMisses
  const cacheHitRate = cacheTotal > 0 ? (cacheHits / cacheTotal) * 100 : 0

  return {
    totalRequests,
    totalTokens,
    inputTokens,
    outputTokens,
    totalCost: totalTokens * TOKEN_PRICE_USD,
    cacheHits,
    cacheMisses,
    cacheHitRate,
    savedTokens,
    savedRequests,
    savedCost: savedTokens * TOKEN_PRICE_USD
  }
}

export function buildUsageOverviewFromStats(stats: UsageStatsPayload): UsageOverviewSummary {
  const totalRequests = toNumber(stats.total_requests)
  const totalTokens = toNumber(stats.total_tokens)
  const cacheHits = toNumber(stats.cache_hits)
  const cacheMisses = toNumber(stats.cache_misses)
  const savedTokens = toNumber(stats.saved_tokens)
  const savedRequests = toNumber(stats.saved_requests)

  const cacheTotal = cacheHits + cacheMisses
  const fallbackHitRate = cacheTotal > 0 ? (cacheHits / cacheTotal) * 100 : 0
  const cacheHitRate = toNumber(stats.cache_hit_rate) || fallbackHitRate

  return {
    totalRequests,
    totalTokens,
    inputTokens: 0,
    outputTokens: 0,
    totalCost: totalTokens * TOKEN_PRICE_USD,
    cacheHits,
    cacheMisses,
    cacheHitRate,
    savedTokens,
    savedRequests,
    savedCost: savedTokens * TOKEN_PRICE_USD
  }
}

export function pickUsageOverview(
  stats: UsageStatsPayload | null | undefined,
  rows: UsageOverviewRow[]
): UsageOverviewSummary {
  if (stats) {
    return buildUsageOverviewFromStats(stats)
  }
  return buildUsageOverviewFromRows(rows)
}

export function normalizeUsageSource(value: unknown): UsageSource {
  if (typeof value !== 'string') {
    return ''
  }
  const normalized = value.trim().toLowerCase()
  if (normalized === 'actual' || normalized === 'estimated') {
    return normalized
  }
  return ''
}

export function usageSourceLabel(source: UsageSource): string {
  if (source === 'actual') {
    return '真实'
  }
  if (source === 'estimated') {
    return '估算'
  }
  return '-'
}
