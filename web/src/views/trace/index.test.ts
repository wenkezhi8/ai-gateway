import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

function getIndexOrThrow(content: string, fragment: string) {
  const idx = content.indexOf(fragment)
  expect(idx).toBeGreaterThan(-1)
  return idx
}

describe('trace detail view', () => {
  it('shows user and ai message preview blocks with full-text actions', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('trace.attributes?.user_message_preview')
    expect(viewFile).toContain('trace.attributes?.ai_response_preview')
    expect(viewFile).toContain("@click=\"showFullMessage(trace, 'user')\"")
    expect(viewFile).toContain("@click=\"showFullMessage(trace, 'ai')\"")
    expect(viewFile).toContain('messageVisible')
    expect(viewFile).toContain('activeMessageContent')
  })

  it('shows answer source column and clear traces action', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('AI回复来源')
    expect(viewFile).toContain('任务类型')
    expect(viewFile).toContain('prop="task_type"')
    expect(viewFile).toContain('清理链路记录')
    expect(viewFile).toContain('clearTraces')
    expect(viewFile).toContain('const clearing = ref(false)')
    expect(viewFile).not.toContain("unknown: '未知'")
    expect(viewFile).not.toContain("|| '未知'")
  })

  it('removes frontend span grouping and consumes backend request summaries', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).not.toContain('const grouped = new Map<string, RequestTrace[]>()')
    expect(viewFile).toContain('total.value = result.total')
    expect(viewFile).toContain('traces.value = result.data')
    expect(viewFile).toContain('page.value = 1')
  })

  it('should render trace table columns in requested order', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    const timeIdx = getIndexOrThrow(viewFile, 'label="时间"')
    const taskTypeIdx = getIndexOrThrow(viewFile, 'label="任务类型"')
    const methodIdx = getIndexOrThrow(viewFile, 'label="方法"')
    const pathIdx = getIndexOrThrow(viewFile, 'label="路径"')
    const modelIdx = getIndexOrThrow(viewFile, 'label="模型"')
    const stepIdx = getIndexOrThrow(viewFile, 'label="步骤"')
    const statusIdx = getIndexOrThrow(viewFile, 'label="状态"')
    const durationIdx = getIndexOrThrow(viewFile, 'label="耗时"')
    const answerSourceIdx = getIndexOrThrow(viewFile, 'label="AI回复来源"')
    const requestIdIdx = getIndexOrThrow(viewFile, 'label="Request ID"')
    const actionIdx = getIndexOrThrow(viewFile, 'label="操作"')

    expect(timeIdx).toBeLessThan(taskTypeIdx)
    expect(taskTypeIdx).toBeLessThan(methodIdx)
    expect(methodIdx).toBeLessThan(pathIdx)
    expect(pathIdx).toBeLessThan(modelIdx)
    expect(modelIdx).toBeLessThan(stepIdx)
    expect(stepIdx).toBeLessThan(statusIdx)
    expect(statusIdx).toBeLessThan(durationIdx)
    expect(durationIdx).toBeLessThan(answerSourceIdx)
    expect(answerSourceIdx).toBeLessThan(requestIdIdx)
    expect(requestIdIdx).toBeLessThan(actionIdx)
  })

  it('should display model column with correct prop and empty value handling', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    // 验证模型列的 prop 属性
    expect(viewFile).toContain('prop="model"')
    expect(viewFile).toContain('label="模型"')

    // 验证模型列显示 "-" 当值为空时
    expect(viewFile).toContain('row.model || \'-\'')
  })
})
