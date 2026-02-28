import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing ollama running model visibility', () => {
  it('should show current running model and anti-sleep hint', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/routing/components/OllamaTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/routing/composables/useRoutingConsole.ts'), 'utf-8')

    expect(tabFile).toContain('当前运行模型')
    expect(tabFile).toContain('已禁用模型自动休眠')
    expect(logicFile).toContain('running_model')
    expect(logicFile).toContain('keep_alive_disabled')
    expect(tabFile).toContain('运行模型列表')
    expect(tabFile).toContain('显存占用')
    expect(logicFile).toContain('setInterval(loadOllamaSetupStatus')
    expect(logicFile).toContain('clearInterval(ollamaStatusPollTimer)')
    expect(logicFile).toContain('running_model_details')
    expect(tabFile).toContain('formatVramBytes(item.size_vram)')
  })
})
