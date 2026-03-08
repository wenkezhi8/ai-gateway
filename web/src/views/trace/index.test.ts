import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

function getIndexOrThrow(content: string, fragment: string) {
  const idx = content.indexOf(fragment)
  expect(idx).toBeGreaterThan(-1)
  return idx
}

describe('trace detail view', () => {
  it('shows raw request, derived prompt and ai preview blocks with full-text actions', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('trace.attributes?.user_message_raw_preview')
    expect(viewFile).toContain('trace.attributes?.user_message_preview')
    expect(viewFile).toContain('trace.attributes?.ai_response_preview')
    expect(viewFile).toContain('原始请求预览')
    expect(viewFile).toContain('清洗后问题')
    expect(viewFile).toContain('@click="showFullMessage(trace, \'user_raw\')"')
    expect(viewFile).toContain('@click="showFullMessage(trace, \'user\')"')
    expect(viewFile).toContain('@click="showFullMessage(trace, \'ai\')"')
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
    expect(viewFile).toContain("exact_raw: '原始缓存'")
    expect(viewFile).toContain("exact_prompt: '精确缓存'")
    expect(viewFile).toContain("semantic: '语义缓存'")
    expect(viewFile).toContain("v2: '向量缓存'")
    expect(viewFile).toContain("provider_chat: '上游回源'")
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
    const providerIdx = getIndexOrThrow(viewFile, 'label="服务商"')
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
    expect(modelIdx).toBeLessThan(providerIdx)
    expect(providerIdx).toBeLessThan(stepIdx)
    expect(stepIdx).toBeLessThan(statusIdx)
    expect(statusIdx).toBeLessThan(durationIdx)
    expect(durationIdx).toBeLessThan(answerSourceIdx)
    expect(answerSourceIdx).toBeLessThan(requestIdIdx)
    expect(requestIdIdx).toBeLessThan(actionIdx)
  })

  it('should display provider column and detail provider field with fallback rendering', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('prop="provider"')
    expect(viewFile).toContain('label="服务商"')
    expect(viewFile).toContain('CHAT_PROVIDER_VISUALS')
    expect(viewFile).toContain('CHAT_PROVIDER_VISUAL_FALLBACK')
    expect(viewFile).toContain('getProviderLabel(row.provider)')
    expect(viewFile).toContain("row.provider || '-'")
    expect(viewFile).toContain('el-descriptions-item label="服务商"')
    expect(viewFile).toContain('detailSummary.provider')
  })

  it('should display model column with correct prop and empty value handling', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('prop="model"')
    expect(viewFile).toContain('label="模型"')
    expect(viewFile).toContain("row.model || '-'")
  })
})
