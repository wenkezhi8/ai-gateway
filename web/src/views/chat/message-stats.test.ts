/**
 * Message Statistics Computation Tests
 */
import { describe, it, expect } from 'vitest'
import { computeMessageStats } from './message-stats'
import type { CompletionMeta } from '@/types/chat'

describe('computeMessageStats', () => {
  const baseMeta: CompletionMeta = {
    requestMode: 'stream'
  }

  describe('non_stream mode', () => {
    it('should compute throughput by total_time in non_stream mode', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        requestMode: 'non_stream',
        completionTokens: 100
      }
      
      const result = computeMessageStats({
        startTime: 0,
        endTime: 2000, // 2 seconds
        meta,
        estimatedCompletionTokens: 100
      })

      expect(result.outputTokensPerSecond).toBe(50) // 100 tokens / 2s
      expect(result.speedBasis).toBe('total_time')
      expect(result.requestMode).toBe('non_stream')
    })
  })

  describe('stream mode', () => {
    it('should keep post_first_token basis for normal stream case', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        requestMode: 'stream',
        completionTokens: 100
      }
      
      const result = computeMessageStats({
        startTime: 0,
        firstTokenTime: 500, // first token at 0.5s
        endTime: 2500, // end at 2.5s
        meta,
        estimatedCompletionTokens: 100
      })

      // completion time = 2.5s - 0.5s = 2s
      expect(result.outputTokensPerSecond).toBe(50) // 100 tokens / 2s
      expect(result.speedBasis).toBe('post_first_token')
    })

    it('should fallback throughput basis when post-first-token window is too short', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        requestMode: 'stream',
        completionTokens: 100
      }
      
      const result = computeMessageStats({
        startTime: 0,
        firstTokenTime: 2800, // first token at 2.8s (late arrival)
        endTime: 2900, // end at 2.9s (window is only 0.1s)
        meta,
        estimatedCompletionTokens: 100
      })

      // window < 0.2s and totalTime > 1s, should fallback
      expect(result.speedBasis).toBe('fallback_total_time')
      expect(result.outputTokensPerSecond).toBe(34) // 100 tokens / 2.9s ≈ 34
    })

    it('should not fallback when totalTime is short', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        requestMode: 'stream',
        completionTokens: 100
      }
      
      const result = computeMessageStats({
        startTime: 0,
        firstTokenTime: 150, // first token at 0.15s
        endTime: 250, // end at 0.25s (total time < 1s)
        meta,
        estimatedCompletionTokens: 100
      })

      // totalTime < 1s, no fallback
      expect(result.speedBasis).toBe('post_first_token')
    })
  })

  describe('cache metadata', () => {
    it('should pass through cache metadata', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        cacheHit: true,
        cacheLayer: 'exact'
      }
      
      const result = computeMessageStats({
        startTime: 0,
        endTime: 1000,
        meta
      })

      expect(result.cacheHit).toBe(true)
      expect(result.cacheLayer).toBe('exact')
    })

    it('should handle undefined cacheHit', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        cacheHit: undefined
      }
      
      const result = computeMessageStats({
        startTime: 0,
        endTime: 1000,
        meta
      })

      expect(result.cacheHit).toBeUndefined()
    })
  })

  describe('token estimation', () => {
    it('should use API tokens when available', () => {
      const meta: CompletionMeta = {
        ...baseMeta,
        promptTokens: 50,
        completionTokens: 100,
        totalTokens: 150
      }
      
      const result = computeMessageStats({
        startTime: 0,
        endTime: 1000,
        meta,
        estimatedPromptTokens: 999, // should be ignored
        estimatedCompletionTokens: 999 // should be ignored
      })

      expect(result.promptTokens).toBe(50)
      expect(result.completionTokens).toBe(100)
      expect(result.totalTokens).toBe(150)
    })

    it('should use estimated tokens when API tokens not available', () => {
      const meta: CompletionMeta = {
        ...baseMeta
      }
      
      const result = computeMessageStats({
        startTime: 0,
        endTime: 1000,
        meta,
        estimatedPromptTokens: 50,
        estimatedCompletionTokens: 100
      })

      expect(result.promptTokens).toBe(50)
      expect(result.completionTokens).toBe(100)
      expect(result.totalTokens).toBe(150)
    })
  })
})
