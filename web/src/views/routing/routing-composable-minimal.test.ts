import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing composable minimal scope', () => {
  it('does not keep classifier or ollama management logic', () => {
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')

    expect(logicFile).not.toContain('switchClassifierModelAsync')
    expect(logicFile).not.toContain('getClassifierSwitchTask')
    expect(logicFile).not.toContain('getOllamaStatus')
    expect(logicFile).not.toContain('getOllamaDualModelConfig')
    expect(logicFile).toContain('getRouterConfig')
    expect(logicFile).toContain('getTaskTypeDistribution')
    expect(logicFile).toContain('getFeedbackStats')
  })
})
