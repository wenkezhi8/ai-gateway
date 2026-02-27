import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { ACCOUNT_PROVIDER_OPTIONS } from './pages/accounts'
import { OPS_TIME_TABS } from './pages/ops'
import { DASHBOARD_FALLBACK_SERIES } from './pages/dashboard'
import { SETTINGS_MENU_ITEMS, THEME_COLOR_OPTIONS } from './pages/settings'
import { USAGE_CSV_HEADER } from './pages/usage'
import { PROVIDERS_ACCOUNTS_BASE_TYPES } from './pages/providers-accounts'

describe('pages static config extraction', () => {
  it('should expose accounts provider options', () => {
    expect(ACCOUNT_PROVIDER_OPTIONS.length).toBeGreaterThan(5)
  })

  it('should expose ops time tabs', () => {
    expect(OPS_TIME_TABS).toEqual(['1min', '5min', '30min', '1h'])
  })

  it('should expose dashboard fallback time series', () => {
    expect(DASHBOARD_FALLBACK_SERIES.timestamps.length).toBe(6)
    expect(DASHBOARD_FALLBACK_SERIES.requests.length).toBe(6)
    expect(DASHBOARD_FALLBACK_SERIES.successRates.length).toBe(6)
  })

  it('should expose settings static options', () => {
    expect(SETTINGS_MENU_ITEMS.length).toBeGreaterThan(0)
    expect(THEME_COLOR_OPTIONS.length).toBeGreaterThan(0)
  })

  it('should expose usage csv header', () => {
    expect(USAGE_CSV_HEADER).toContain('任务类型')
  })

  it('should expose providers-accounts base types', () => {
    expect(PROVIDERS_ACCOUNTS_BASE_TYPES.length).toBeGreaterThan(0)
  })

  it('should remove hardcoded model defaults from stores', () => {
    const modelsStoreFile = readFileSync(join(process.cwd(), 'src/store/models.ts'), 'utf-8')
    const chatStoreFile = readFileSync(join(process.cwd(), 'src/store/chat.ts'), 'utf-8')

    expect(modelsStoreFile).not.toContain('const defaultModels: Model[] = [')
    expect(chatStoreFile).not.toContain('const DEFAULT_PROVIDERS: ProviderConfig[] = [')
  })
})
