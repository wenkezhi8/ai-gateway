import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('cache page task type distribution', () => {
  it('renders task type distribution panel in cache page', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(viewFile).toContain('任务类型分布')
    expect(viewFile).toContain('loadTaskTypeDistribution')
    expect(viewFile).toContain('taskTypeDistributionState')
  })

  it('uses feedback distribution API and fixed task-type order', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(viewFile).toContain('getTaskTypeDistribution')
    expect(viewFile).toContain("['code', 'chat', 'reasoning', 'math', 'fact', 'creative', 'translate', 'other']")
  })

  it('uses ttl and force refresh for distribution requests', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(viewFile).toContain('const taskTypeDistributionTtlMs = 30 * 1000')
    expect(viewFile).toContain('async function loadTaskTypeDistribution(forceRefresh = false)')
    expect(viewFile).toContain('if (!forceRefresh && taskTypeDistribution.value.length > 0')
    expect(viewFile).toContain('@click="loadTaskTypeDistribution(true)"')
  })
})
