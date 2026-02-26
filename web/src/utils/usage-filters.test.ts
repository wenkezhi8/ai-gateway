import { describe, expect, it } from 'vitest'
import { filterUsageRows, getTagOptions } from './usage-filters'

describe('usage-filters', () => {
  it('getTagOptions should remove empty and placeholder values', () => {
    const options = getTagOptions(['exp-a', '', '-', undefined, 'exp-b', 'exp-a'])
    expect(options).toEqual(['exp-a', 'exp-b'])
  })

  it('filterUsageRows should filter by model/experiment/domain', () => {
    const rows = [
      { model: 'gpt-4o-mini', experimentTag: 'exp-a', domainTag: 'finance' },
      { model: 'gpt-4o-mini', experimentTag: 'exp-b', domainTag: 'general' },
      { model: 'claude-3-5-sonnet', experimentTag: 'exp-a', domainTag: 'finance' }
    ]

    const filtered = filterUsageRows(rows, {
      model: 'gpt-4o-mini',
      experimentTag: 'exp-a',
      domainTag: 'finance'
    })

    expect(filtered).toHaveLength(1)
    expect(filtered[0]).toEqual({ model: 'gpt-4o-mini', experimentTag: 'exp-a', domainTag: 'finance' })
  })
})
