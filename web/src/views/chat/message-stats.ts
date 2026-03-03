/**
 * Message Statistics Computation
 * 
 * 统一的统计计算逻辑，支持流式/非流式模式，避免首token延迟导致的吞吐率失真
 */

import type { MessageStats, CompletionMeta } from '@/types/chat'

export interface ComputeMessageStatsInput {
  /** 请求开始时间戳（毫秒） */
  startTime: number
  /** 首token到达时间戳（毫秒），非流式时为 undefined */
  firstTokenTime?: number
  /** 请求结束时间戳（毫秒） */
  endTime: number
  /** API 返回的元数据 */
  meta: CompletionMeta
  /** 估算的 prompt tokens（当 API 未返回时使用） */
  estimatedPromptTokens?: number
  /** 估算的 completion tokens（当 API 未返回时使用） */
  estimatedCompletionTokens?: number
}

/**
 * 计算消息统计信息
 * 
 * 速度计算规则：
 * - non_stream: 使用 totalTime 作为分母
 * - stream + 首token正常: 使用 (end - firstToken) 作为分母
 * - stream + 首token延迟异常（窗口 < 0.2s 且 totalTime > 1s）: 回退到 totalTime
 */
export function computeMessageStats(input: ComputeMessageStatsInput): MessageStats {
  const { startTime, firstTokenTime, endTime, meta, estimatedPromptTokens, estimatedCompletionTokens } = input
  
  const totalTimeMs = endTime - startTime
  const firstTokenMs = firstTokenTime ? firstTokenTime - startTime : undefined
  
  // 确定 tokens
  const promptTokens = meta.promptTokens ?? estimatedPromptTokens
  const completionTokens = meta.completionTokens ?? estimatedCompletionTokens
  const totalTokens = meta.totalTokens ?? (promptTokens && completionTokens ? promptTokens + completionTokens : undefined)
  
  // 计算速度和速度口径
  let outputTokensPerSecond: number | undefined
  let speedBasis: MessageStats['speedBasis'] = undefined
  
  if (meta.requestMode === 'non_stream') {
    // 非流式：使用总时长
    const totalTimeSec = totalTimeMs / 1000
    if (totalTimeSec > 0 && completionTokens) {
      outputTokensPerSecond = Math.round(completionTokens / totalTimeSec)
      speedBasis = 'total_time'
    }
  } else {
    // 流式：优先使用首token后时长
    const totalTimeSec = totalTimeMs / 1000
    const completionTimeSec = firstTokenTime ? (endTime - firstTokenTime) / 1000 : 0
    
    // 判断是否需要回退到总时长（避免首token延迟导致的失真）
    const shouldFallback = completionTimeSec < 0.2 && totalTimeSec > 1
    
    if (shouldFallback && totalTimeSec > 0 && completionTokens) {
      // 回退：使用总时长
      outputTokensPerSecond = Math.round(completionTokens / totalTimeSec)
      speedBasis = 'fallback_total_time'
    } else if (completionTimeSec > 0 && completionTokens) {
      // 正常：使用首token后时长
      outputTokensPerSecond = Math.round(completionTokens / completionTimeSec)
      speedBasis = 'post_first_token'
    }
  }
  
  return {
    firstTokenTime: firstTokenMs ? firstTokenMs / 1000 : undefined,
    totalTime: totalTimeMs / 1000,
    outputTokensPerSecond,
    totalTokens,
    promptTokens,
    completionTokens,
    cacheHit: meta.cacheHit,
    cacheLayer: meta.cacheLayer,
    requestMode: meta.requestMode,
    reasoningEffortDowngraded: meta.reasoningEffortDowngraded,
    speedBasis
  }
}
