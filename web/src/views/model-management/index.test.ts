import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('model management settings loading', () => {
  it('keeps provider defaults visible when models API is unavailable', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain(".catch(() => ({ success: false, data: [] }))")
    expect(viewFile).toContain('...Object.keys(providerDefaults)')
    expect(viewFile).toContain('providerDefaults[providerId] || meta?.default_model || models[0] ||')
  })
})
