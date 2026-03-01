import request from '@/api/request'
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
  attributes: Record<string, any>
  events: Record<string, any>
  user_id?: string
  method: string
  path: string
  model?: string
  provider?: string
  error?: string
  created_at: string
}

export async function getTraces(params?: {
  limit?: number
  offset?: number
  operation?: string
  status?: string
  start_time?: string
  end_time?: string
}): Promise<RequestTrace[]> {
  const res = await request.get(API.TRACES.LIST, { params })
  return res.data || []
}

export async function getTraceDetail(requestId: string): Promise<RequestTrace[]> {
  const res = await request.get(API.TRACES.DETAIL.replace(':request_id', requestId))
  return res.data || []
}
