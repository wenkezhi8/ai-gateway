import { describe, expect, it } from 'vitest'

import { normalizeApiKeyRecord } from './api-key-contract'

describe('api key contract', () => {
  it('should keep last_used when present', () => {
    const normalized = normalizeApiKeyRecord({
      id: 'k1',
      name: 'demo',
      key: 'sk-demo',
      created_at: '2026-03-05T00:00:00Z',
      last_used: '2026-03-06T00:00:00Z',
      last_used_at: '2026-03-01T00:00:00Z',
      enabled: true
    })

    expect(normalized.last_used).toBe('2026-03-06T00:00:00Z')
  })

  it('should fallback to legacy last_used_at when last_used is missing', () => {
    const normalized = normalizeApiKeyRecord({
      id: 'k2',
      name: 'legacy',
      key: 'sk-legacy',
      created_at: '2026-03-05T00:00:00Z',
      last_used_at: '2026-03-04T00:00:00Z',
      enabled: true
    })

    expect(normalized.last_used).toBe('2026-03-04T00:00:00Z')
  })
})
