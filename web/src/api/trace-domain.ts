import { request } from './request'
import { API } from '@/constants/api'

export interface RequestTrace {
  id: string
  request_id: string
  trace_id: string
  span_id: string
  parent_span_id?: string
  operation: string
  status: string
  start_time: string
  end_time: string
  duration_ms: number
  attributes: TraceAttributes
  events: Record<string, any>
  user_id?: string
  method: string
  path: string
  model?: string
  provider?: string
  error?: string
  created_at: string
}

export interface TraceAttributes extends Record<string, any> {
  // Preview fields are capped at 200 chars.
  user_message_preview?: string
  ai_response_preview?: string
  // Full fields are capped at 4000 chars.
  user_message_full?: string
  ai_response_full?: string
  user_message_truncated?: boolean
  ai_response_truncated?: boolean
}

export type TraceAnswerSource = 'cache_v2' | 'cache_semantic' | 'cache_exact' | 'provider_chat' | 'unknown'

export interface TraceSummary {
  request_id: string
  method: string
  path: string
  status: string
  duration_ms: number
  created_at: string
  step_count: number
  answer_source: TraceAnswerSource
}

export interface TraceListResult {
  data: TraceSummary[]
  total: number
}

export async function getTraces(params?: {
  limit?: number
  offset?: number
  operation?: string
  status?: string
  start_time?: string
  end_time?: string
}): Promise<TraceListResult> {
  const res: any = await request.get(API.TRACES.LIST, { params })

  if (res?.success && Array.isArray(res.data) && typeof res.total === 'number') {
    return {
      data: res.data,
      total: res.total
    }
  }

  if (res?.success && Array.isArray(res.data)) {
    return {
      data: res.data,
      total: res.data.length
    }
  }

  return {
    data: [],
    total: 0
  }
}

export async function getTraceDetail(requestId: string): Promise<RequestTrace[]> {
  const res: any = await request.get(API.TRACES.DETAIL.replace(':request_id', requestId))
  if (res?.success && Array.isArray(res.data)) {
    return res.data
  }
  return []
}

export async function clearTraces(): Promise<{ deleted: number }> {
  const res: any = await request.delete(API.TRACES.CLEAR)
  if (res?.success && typeof res.data?.deleted === 'number') {
    return { deleted: res.data.deleted }
  }
  return { deleted: 0 }
}
