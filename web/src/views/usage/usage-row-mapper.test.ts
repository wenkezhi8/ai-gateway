import { describe, expect, it } from 'vitest'

import { mapUsageLogToRow } from './usage-row-mapper'

describe('usage-row-mapper', () => {
  it('should map expanded usage log fields with aliases', () => {
    const row = mapUsageLogToRow(
      {
        id: 11,
        account: 'acc-alpha',
        service_provider: 'provider-x',
        provider: 'provider-fallback',
        timestamp: 1700000000000,
        model: 'gpt-x',
        task_type: 'code',
        request_type: 'stream',
        type: 'legacy-type',
        inference_intensity: 'high',
        user_agent: 'Mozilla/5.0',
        input_tokens: 120,
        output_tokens: 80,
        cached_read_tokens: 16,
        total_tokens: 200,
        tokens: 999,
        saved_tokens: 30,
        usage_source: 'actual',
        cache_hit: true,
        success: true,
        time_to_first_token: 420,
        total_duration: 1550
      },
      new Map([['provider-x', 'ignored-by-account-field']])
    )

    expect(row.accountName).toBe('acc-alpha')
    expect(row.provider).toBe('provider-x')
    expect(row.taskType).toBe('编程')
    expect(row.requestType).toBe('流式')
    expect(row.inferenceIntensity).toBe('high')
    expect(row.userAgent).toBe('Mozilla/5.0')
    expect(row.inputTokens).toBe(120)
    expect(row.outputTokens).toBe(80)
    expect(row.totalTokens).toBe(200)
    expect(row.cachedReadTokens).toBe(16)
    expect(row.savedTokens).toBe(30)
    expect(row.firstTokenLatency).toBe('0.42s')
    expect(row.totalLatency).toBe('1.55s')
  })

  it('should fallback to legacy fields and derive totals', () => {
    const row = mapUsageLogToRow(
      {
        id: 12,
        provider: 'provider-y',
        timestamp: 1700000005000,
        model: 'legacy-model',
        task_type: 'chat',
        type: 'nonstream',
        tokens: 50,
        ttft_ms: 0,
        latency_ms: 0,
        cache_hit: true,
        success: true,
        usage_source: 'estimated'
      },
      new Map([['provider-y', '账户Y']])
    )

    expect(row.accountName).toBe('账户Y')
    expect(row.taskType).toBe('对话')
    expect(row.requestType).toBe('非流式')
    expect(row.inferenceIntensity).toBe('-')
    expect(row.userAgent).toBe('-')
    expect(row.inputTokens).toBe(30)
    expect(row.outputTokens).toBe(20)
    expect(row.totalTokens).toBe(50)
    expect(row.cachedReadTokens).toBe(0)
    expect(row.savedTokens).toBe(50)
    expect(row.firstTokenLatency).toBe('0 ms')
    expect(row.totalLatency).toBe('0 ms')
  })

  it('should map task type to chinese label and keep raw value for tooltip tracing', () => {
    const longTextRow = mapUsageLogToRow(
      {
        id: 21,
        provider: 'provider-z',
        task_type: 'long_text',
        type: 'stream',
        cached_read_tokens: 5,
        total_tokens: 10,
        usage_source: 'actual'
      },
      new Map()
    )

    expect(longTextRow.taskType).toBe('长文本')
    expect(longTextRow.taskTypeLabel).toBe('长文本')
    expect(longTextRow.taskTypeRaw).toBe('long_text')
    expect(longTextRow.usageSourceLabel).toBe('真实')
    expect(longTextRow.cachedReadTokens).toBe(5)

    const unknownRow = mapUsageLogToRow(
      {
        id: 22,
        provider: 'provider-z',
        task_type: 'unknown',
        total_tokens: 9,
        usage_source: 'estimated'
      },
      new Map()
    )

    expect(unknownRow.taskType).toBe('未知')
    expect(unknownRow.taskTypeLabel).toBe('未知')
    expect(unknownRow.taskTypeRaw).toBe('unknown')
    expect(unknownRow.usageSourceLabel).toBe('估算')

    const newTypeRow = mapUsageLogToRow(
      {
        id: 23,
        provider: 'provider-z',
        task_type: 'brand_new_type',
        total_tokens: 8
      },
      new Map()
    )

    expect(newTypeRow.taskType).toBe('其他')
    expect(newTypeRow.taskTypeLabel).toBe('其他')
    expect(newTypeRow.taskTypeRaw).toBe('brand_new_type')
    expect(newTypeRow.usageSourceLabel).toBe('-')
  })
})
