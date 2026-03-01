import { describe, expect, it } from 'vitest'
import {
  TOKEN_PRICE_USD,
  buildUsageOverviewFromRows,
  buildUsageOverviewFromStats,
  normalizeUsageSource,
  usageSourceLabel,
  pickUsageOverview
} from './usage-overview'

describe('usage-overview', () => {
  it('buildUsageOverviewFromStats should map saved tokens and cost', () => {
    const summary = buildUsageOverviewFromStats({
      total_requests: 8,
      total_tokens: 2400,
      cache_hits: 5,
      cache_misses: 3,
      cache_hit_rate: 62.5,
      saved_tokens: 1200,
      saved_requests: 4
    })

    expect(summary.totalRequests).toBe(8)
    expect(summary.totalTokens).toBe(2400)
    expect(summary.cacheHits).toBe(5)
    expect(summary.cacheMisses).toBe(3)
    expect(summary.cacheHitRate).toBe(62.5)
    expect(summary.savedTokens).toBe(1200)
    expect(summary.savedRequests).toBe(4)
    expect(summary.savedCost).toBeCloseTo(1200 * TOKEN_PRICE_USD)
  })

  it('buildUsageOverviewFromRows should only count cache_hit rows as saved', () => {
    const summary = buildUsageOverviewFromRows([
      {
        inputTokens: 40,
        outputTokens: 20,
        totalTokens: 60,
        cacheHit: '命中',
        success: true
      },
      {
        inputTokens: 30,
        outputTokens: 20,
        totalTokens: 50,
        cacheHit: '命中',
        success: false
      },
      {
        inputTokens: 10,
        outputTokens: 30,
        totalTokens: 40,
        cacheHit: '未命中',
        success: true
      }
    ])

    expect(summary.totalRequests).toBe(3)
    expect(summary.totalTokens).toBe(150)
    expect(summary.inputTokens).toBe(80)
    expect(summary.outputTokens).toBe(70)
    expect(summary.cacheHits).toBe(2)
    expect(summary.cacheMisses).toBe(1)
    expect(summary.savedTokens).toBe(60)
    expect(summary.savedRequests).toBe(1)
  })

  it('pickUsageOverview should fallback to rows when stats is empty', () => {
    const summary = pickUsageOverview(null, [
      {
        inputTokens: 12,
        outputTokens: 8,
        totalTokens: 20,
        cacheHit: '命中',
        success: true
      }
    ])

    expect(summary.totalRequests).toBe(1)
    expect(summary.savedTokens).toBe(20)
    expect(summary.savedRequests).toBe(1)
  })

  it('usage source helpers should normalize and label values', () => {
    expect(normalizeUsageSource('actual')).toBe('actual')
    expect(normalizeUsageSource('estimated')).toBe('estimated')
    expect(normalizeUsageSource('unknown')).toBe('')

    expect(usageSourceLabel('actual')).toBe('真实')
    expect(usageSourceLabel('estimated')).toBe('估算')
    expect(usageSourceLabel('')).toBe('-')
  })
})
