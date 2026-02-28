import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing ollama running model visibility', () => {
  it('should show current running model and anti-sleep hint', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(viewFile).toContain('当前运行模型')
    expect(viewFile).toContain('已禁用模型自动休眠')
    expect(viewFile).toContain('running_model')
    expect(viewFile).toContain('keep_alive_disabled')
    expect(viewFile).toContain('运行模型列表')
    expect(viewFile).toContain('显存占用')
    expect(viewFile).toContain('setInterval(loadOllamaSetupStatus')
    expect(viewFile).toContain('clearInterval(ollamaStatusPollTimer)')
  })
})
