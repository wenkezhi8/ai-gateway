import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama running model visibility', () => {
  it('shows running model details and polling controls', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/OllamaServiceTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')

    expect(tabFile).toContain('当前运行模型')
    expect(tabFile).toContain('运行中模型')
    expect(tabFile).toContain('显存占用')
    expect(tabFile).toContain('自动轮询')
    expect(tabFile).toContain('轮询间隔')
    expect(tabFile).toContain('formatVramBytes(item.size_vram)')
    expect(logicFile).toContain('running_model')
    expect(logicFile).toContain('running_model_details')
  })
})
