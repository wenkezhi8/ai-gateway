import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing classifier async switch flow', () => {
  it('should use async switch endpoints and friendly timeout message', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')

    expect(viewFile).toContain('/admin/router/classifier/switch-async')
    expect(viewFile).toContain('/admin/router/classifier/switch-tasks/')
    expect(viewFile).toContain('正在加载模型，首次可能较慢（最多180秒）')
    expect(viewFile).toContain('模型加载超时，请继续等待Ollama完成加载后重试')
  })
})
