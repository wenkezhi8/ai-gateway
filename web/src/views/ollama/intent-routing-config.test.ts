import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('ollama intent routing config', () => {
  it('keeps novice essentials visible and moves strategy details under advanced settings', () => {
    const tabFile = readFileSync(join(process.cwd(), 'src/views/ollama/components/IntentRoutingTab.vue'), 'utf-8')
    const logicFile = readFileSync(join(process.cwd(), 'src/views/ollama/composables/useOllamaConsoleCore.ts'), 'utf-8')
    const constantsFile = readFileSync(join(process.cwd(), 'src/constants/routing.ts'), 'utf-8')

    expect(tabFile).toContain('意图模型配置')
    expect(tabFile).toContain('基础设置')
    expect(tabFile).toContain('验证/状态')
    expect(tabFile).toContain('高级设置')
    expect(tabFile).toContain('启动模型')
    expect(tabFile).toContain('@click="ctx.startClassifierModel"')
    expect(tabFile).toContain('启用意图分类器')
    expect(tabFile).toContain('主模型')
    expect(tabFile).toContain('候选模型')
    expect(tabFile).toContain('健康检查')
    expect(tabFile).not.toContain('@click="ctx.startOllama"')
    expect(tabFile).not.toContain('下载模型')
    expect(tabFile).not.toContain('删除模型')
    expect(tabFile).toContain('el-collapse')
    expect(tabFile).toContain('任务类型模型映射')
    expect(tabFile).toContain('级联路由策略')
    expect(tabFile).toContain('v-for="task in ctx.taskTypes"')
    expect(constantsFile).toContain("{ type: 'code', name: '代码生成'")
    expect(constantsFile).toContain("{ type: 'chat', name: '日常对话'")
    expect(constantsFile).toContain("{ type: 'reasoning', name: '逻辑推理'")
    expect(constantsFile).toContain("{ type: 'math', name: '数学计算'")
    expect(constantsFile).toContain("{ type: 'fact', name: '事实查询'")
    expect(constantsFile).toContain("{ type: 'creative', name: '创意写作'")
    expect(constantsFile).toContain("{ type: 'translate', name: '翻译'")
    expect(constantsFile).toContain("{ type: 'other', name: '其他'")

    expect(logicFile).toContain('getOllamaDualModelConfig')
    expect(logicFile).toContain('updateOllamaDualModelConfig')
    expect(logicFile).toContain('async function startClassifierModel()')
    expect(logicFile).toContain('createDefaultTaskModelMapping')
  })
})
