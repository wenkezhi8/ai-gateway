import { describe, expect, it } from 'vitest'

import {
  buildFallbackTaskMeta,
  mergeTtlOverrides,
  resolveCacheTaskMetaLoadResult
} from './cache-task-meta'

describe('cache task meta fallback', () => {
  it('should fallback when /admin/cache/task-ttl returns 404', () => {
    const result = resolveCacheTaskMetaLoadResult({
      isTaskTTL404: true,
      taskTypes: [],
      modelOptions: [],
      publicProviders: [
        { id: 'openai', label: 'OpenAI', color: '', logo: '', default_model: 'gpt-4o' }
      ],
      ttlDefaults: { fact: 48 }
    })

    expect(result.mode).toBe('fallback')
    expect(result.taskTypes.length).toBeGreaterThan(0)
    expect(result.modelOptions[0]?.provider_id).toBe('openai')
  })

  it('should keep ttl panel renderable in fallback mode', () => {
    const fallback = buildFallbackTaskMeta()
    const merged = mergeTtlOverrides(fallback.task_types, {
      fact: 36,
      code: 120
    })

    expect(merged.length).toBeGreaterThan(0)
    expect(merged.find(item => item.key === 'fact')?.default_ttl).toBe(36)
    expect(merged.find(item => item.key === 'code')?.default_ttl).toBe(120)
    expect(merged.every(item => item.ttl_unit === 'hours')).toBe(true)
  })

  it('should not show blocking error on 404 fallback', () => {
    const result = resolveCacheTaskMetaLoadResult({
      isTaskTTL404: true,
      taskTypes: [],
      modelOptions: [],
      publicProviders: [],
      ttlDefaults: {}
    })

    expect(result.errorMessage).toBe('')
    expect(result.warningMessage).toContain('兼容模式')
  })
})
