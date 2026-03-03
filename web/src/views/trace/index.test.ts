import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

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
})
