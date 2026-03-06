import { describe, expect, it } from 'vitest'

import {
  normalizeUseAutoMode,
  resolveUseAutoModeMigrationNotice
} from './router-mode'

describe('router use_auto_mode contract helpers', () => {
  it('should normalize deprecated latest mode to auto', () => {
    expect(normalizeUseAutoMode('latest')).toBe('auto')
    expect(resolveUseAutoModeMigrationNotice('latest')).toContain('latest')
  })

  it('should keep supported modes unchanged', () => {
    expect(normalizeUseAutoMode('auto')).toBe('auto')
    expect(normalizeUseAutoMode('default')).toBe('default')
    expect(normalizeUseAutoMode('fixed')).toBe('fixed')
    expect(resolveUseAutoModeMigrationNotice('auto')).toBe('')
  })

  it('should fallback unknown modes to auto', () => {
    expect(normalizeUseAutoMode('unknown')).toBe('auto')
    expect(normalizeUseAutoMode(undefined)).toBe('auto')
  })
})
