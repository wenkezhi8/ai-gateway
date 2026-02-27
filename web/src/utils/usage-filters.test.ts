import { describe, expect, it } from 'vitest'
import { filterUsageRows, getTagOptions } from './usage-filters'

describe('usage-filters', () => {
  it('getTagOptions should remove empty and placeholder values', () => {
    const options = getTagOptions(['chat', '', '-', undefined, 'code', 'chat'])
    expect(options).toEqual(['chat', 'code'])
  })

  it('filterUsageRows should filter by model and taskType', () => {
    const rows = [
      { model: 'gpt-4o-mini', taskType: 'chat' },
      { model: 'gpt-4o-mini', taskType: 'code' },
      { model: 'claude-3-5-sonnet', taskType: 'chat' }
    ]

    const filtered = filterUsageRows(rows, {
      model: 'gpt-4o-mini',
      taskType: 'chat'
    })

    expect(filtered).toHaveLength(1)
    expect(filtered[0]).toEqual({ model: 'gpt-4o-mini', taskType: 'chat' })
  })
})
