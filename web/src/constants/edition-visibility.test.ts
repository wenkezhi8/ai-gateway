import { describe, expect, it } from 'vitest'

import { canAccessPath } from './edition-visibility'

describe('edition visibility rules', () => {
  it('allows ollama for standard and enterprise', () => {
    expect(canAccessPath('/ollama', 'basic')).toBe(false)
    expect(canAccessPath('/ollama', 'standard')).toBe(true)
    expect(canAccessPath('/ollama', 'enterprise')).toBe(true)
  })

  it('allows vector and knowledge only for enterprise', () => {
    expect(canAccessPath('/vector-db/collections', 'standard')).toBe(false)
    expect(canAccessPath('/knowledge/documents', 'standard')).toBe(false)
    expect(canAccessPath('/vector-db/collections', 'enterprise')).toBe(true)
    expect(canAccessPath('/knowledge/documents', 'enterprise')).toBe(true)
  })
})
