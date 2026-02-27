import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('cache entry detail dialog', () => {
  it('should show full user message in entry detail dialog', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(viewFile).toContain('<h4>用户消息</h4>')
    expect(viewFile).toContain(':model-value="getUserMessageFull(entryDetail)"')
  })
})
