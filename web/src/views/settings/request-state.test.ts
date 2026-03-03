import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('settings request state', () => {
  it('should load defaults and ui independently', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/settings/index.vue'), 'utf-8')

    expect(viewFile).toContain('Promise.allSettled')
    expect(viewFile).toContain('loadSettings')
    expect(viewFile).toContain('defaultsLoadError')
    expect(viewFile).toContain('uiLoadError')
  })

  it('should render loading error empty and success states', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/settings/index.vue'), 'utf-8')

    expect(viewFile).toContain('v-if="isInitialLoading"')
    expect(viewFile).toContain('v-else-if="showHardError"')
    expect(viewFile).toContain('v-else-if="showEmptyState"')
    expect(viewFile).toContain('v-else')
    expect(viewFile).toContain('@click="loadSettings"')
  })

  it('should disable reset when defaults unavailable', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/settings/index.vue'), 'utf-8')

    expect(viewFile).toContain(':disabled="!settingsDefaults"')
    expect(viewFile).toContain('defaultsLoadError ?')
  })
})
