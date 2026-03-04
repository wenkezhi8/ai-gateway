import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('ollama page', () => {
  it('renders service panel above intent/vector tabs', () => {
    const file = resolve(process.cwd(), 'src/views/ollama/index.vue')
    const content = readFileSync(file, 'utf-8')
    const serviceTabIndex = content.indexOf('<OllamaServiceTab :ctx="ctx" />')
    const tabsIndex = content.indexOf('<el-tabs v-model="activeTab" class="console-tabs">')

    expect(content).toContain('Ollama 控制台')
    expect(content).toContain('<OllamaServiceTab :ctx="ctx" />')
    expect(content).toContain('<el-tabs v-model="activeTab" class="console-tabs">')
    expect(serviceTabIndex).toBeGreaterThanOrEqual(0)
    expect(tabsIndex).toBeGreaterThanOrEqual(0)
    expect(serviceTabIndex).toBeLessThan(tabsIndex)
    expect(content).toContain("const activeTab = ref('intent')")
    expect(content).toContain('label="意图路由"')
    expect(content).toContain('label="向量管理"')
    expect(content).not.toContain('label="Ollama"')
    expect(content).toContain('运行中模型总览')
    expect(content).toContain('请先预热模型')
    expect(content).toContain('ctx.ollamaSetup.running_models')
    expect(content).toContain('首版范围说明')
    expect(content).toContain('当前页面首版仅覆盖服务连通、意图路由与向量管理')
    expect(content).toContain('<InfoFilled />')
    expect(content).toContain('class="panel service-panel"')
    expect(content).toContain('class="panel tabs-panel"')
    expect(content.indexOf('class="panel service-panel"')).toBeLessThan(content.indexOf('class="panel tabs-panel"'))
    expect(content.indexOf('运行中模型总览')).toBeLessThan(content.indexOf('<el-tabs'))
  })
})
