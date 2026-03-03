import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

describe('alerts page', () => {
  it('uses backend todayTotal for 今日告警 and supports resolve-similar action', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/alerts/index.vue'), 'utf-8')

    expect(viewFile).toContain("title: '今日告警'")
    expect(viewFile).toContain('todayTotal')
    expect(viewFile).toContain('alertApi.getStats()')
    expect(viewFile).toContain('处理同类')
    expect(viewFile).toContain('@click="resolveSimilar(row)"')
    expect(viewFile).toContain('alertApi.resolveSimilar(')
  })
})
