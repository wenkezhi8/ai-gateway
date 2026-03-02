import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('layout knowledge entry', () => {
  it('adds a top-right knowledge entry and removes knowledge sidebar menu items', () => {
    const file = resolve(process.cwd(), 'src/components/Layout/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('knowledgeConsoleURL')
    expect(content).toContain(':href="knowledgeConsoleURL"')
    expect(content).toContain('知识库')

    expect(content).not.toContain("{ path: '/knowledge/documents', title: '知识库文档'")
    expect(content).not.toContain("{ path: '/knowledge/chat', title: '知识库问答'")
    expect(content).not.toContain("{ path: '/knowledge/config', title: '知识库配置'")
  })
})
