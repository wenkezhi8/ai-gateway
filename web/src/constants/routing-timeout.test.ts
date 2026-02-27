import { describe, expect, it } from 'vitest'
import { DEFAULT_CLASSIFIER_CONFIG } from './routing'

describe('routing classifier timeout defaults', () => {
  it('should align default timeout with backend baseline', () => {
    expect(DEFAULT_CLASSIFIER_CONFIG.timeout_ms).toBe(5000)
  })
})
