import { describe, expect, it } from 'vitest'
import { CACHE_TYPE_META, buildCacheTypeCards, listCacheTypeMeta } from './cache-type-meta'

describe('cache-type-meta', () => {
  it('exposes all core cache types with required fields', () => {
    const requiredKeys = ['id', 'name', 'alias', 'description', 'prefix', 'tone', 'icon'] as const
    const metas = listCacheTypeMeta()
    expect(metas.length).toBeGreaterThanOrEqual(6)
    for (const meta of metas) {
      for (const key of requiredKeys) {
        expect(meta[key]).toBeTruthy()
      }
    }
  })

  it('contains explicit content and request cache descriptions', () => {
    expect(CACHE_TYPE_META.response.description).toContain('最终模型响应')
    expect(CACHE_TYPE_META.request.description).toContain('请求参数')
  })

  it('includes cache key prefixes for known types', () => {
    expect(CACHE_TYPE_META.response.prefix).toBe('ai-gateway:ai-response:*')
    expect(CACHE_TYPE_META.request.prefix).toBe('ai-gateway:req:*')
  })

  it('builds cache type cards with stats merged and order preserved', () => {
    const cards = buildCacheTypeCards([
      { id: 'response', hitRate: 75, entries: 12, size: '2 MB', enabled: false }
    ])

    expect(cards[0]).toBeTruthy()
    expect(cards[1]).toBeTruthy()

    const first = cards[0]!
    const second = cards[1]!

    expect(first.id).toBe('response')
    expect(first.hitRate).toBe(75)
    expect(first.entries).toBe(12)
    expect(first.enabled).toBe(false)
    expect(second.id).toBe('request')
    expect(second.hitRate).toBe(0)
  })
})
