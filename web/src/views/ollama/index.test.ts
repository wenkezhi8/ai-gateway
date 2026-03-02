import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('ollama page', () => {
  it('contains required management actions', () => {
    const file = resolve(process.cwd(), 'src/views/ollama/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('Ollama 管理')
    expect(content).toContain('下载模型')
    expect(content).toContain('删除模型')
    expect(content).toContain('启动服务')
    expect(content).toContain('停止服务')
  })
})
