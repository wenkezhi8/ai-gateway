// @vitest-environment jsdom

import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick } from 'vue'
import ElementPlus from 'element-plus'

import TraceView from './index.vue'

const traceApiMock = vi.hoisted(() => ({
  getTraces: vi.fn(),
  getTraceDetail: vi.fn(),
  clearTraces: vi.fn()
}))

vi.mock('@/api/trace-domain', () => ({
  getTraces: traceApiMock.getTraces,
  getTraceDetail: traceApiMock.getTraceDetail,
  clearTraces: traceApiMock.clearTraces
}))

function getIndexOrThrow(content: string, fragment: string) {
  const idx = content.indexOf(fragment)
  expect(idx).toBeGreaterThan(-1)
  return idx
}

function normalizeHtmlForSnapshot(html: string) {
  return html
    .replace(/el-id-\d+-\d+/g, 'el-id-x')
    .replace(/#el-popper-container-\d+/g, '#el-popper-container-x')
    .replace(/z-index: \d+;/g, 'z-index: X;')
}

class ResizeObserverStub {
  constructor(_callback: ResizeObserverCallback) {}
  observe() {}
  unobserve() {}
  disconnect() {}
}

const globalWithResizeObserver = globalThis as unknown as {
  ResizeObserver?: typeof ResizeObserver
}

if (!globalWithResizeObserver.ResizeObserver) {
  globalWithResizeObserver.ResizeObserver = ResizeObserverStub as unknown as typeof ResizeObserver
}

describe('trace detail view', () => {
  beforeEach(() => {
    traceApiMock.getTraces.mockReset()
    traceApiMock.getTraceDetail.mockReset()
    traceApiMock.clearTraces.mockReset()
  })

  it('renders answer source label in real DOM', async () => {
    traceApiMock.getTraces.mockResolvedValue({
      total: 1,
      data: [
        {
          request_id: 'req-dom-1',
          method: 'POST',
          path: '/api/v1/chat/completions',
          status: 'success',
          duration_ms: 120,
          created_at: '2026-03-09T00:00:00Z',
          step_count: 4,
          answer_source: 'v2',
          task_type: 'analysis',
          model: 'deepseek-chat',
          provider: 'openai'
        }
      ]
    })

    const wrapper = mount(TraceView, {
      global: {
        plugins: [ElementPlus],
        stubs: {
          ElDialog: {
            template: '<div><slot /></div>'
          },
          teleport: true
        }
      }
    })

    await flushPromises()
    await nextTick()

    expect(traceApiMock.getTraces).toHaveBeenCalled()
    expect(wrapper.text()).toContain('AI回复来源')
    expect(wrapper.text()).toContain('向量缓存')
    expect(normalizeHtmlForSnapshot(wrapper.find('.trace-page').html())).toMatchSnapshot()
  })

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

  it('references centralized answer source labels', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('TRACE_ANSWER_SOURCE_LABELS')
    expect(viewFile).toContain('TRACE_ANSWER_SOURCE_FALLBACK')
    expect(viewFile).not.toContain("provider_chat: '上游回源'")
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
