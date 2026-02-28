import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { ACCOUNT_PROVIDER_OPTIONS } from './pages/accounts'
import { OPS_TIME_TABS } from './pages/ops'
import { DASHBOARD_FALLBACK_SERIES } from './pages/dashboard'
import { SETTINGS_MENU_ITEMS, THEME_COLOR_OPTIONS } from './pages/settings'
import { USAGE_CSV_HEADER } from './pages/usage'
import { PROVIDERS_ACCOUNTS_BASE_TYPES, PROVIDERS_ACCOUNTS_DEFAULT_ENDPOINTS } from './pages/providers-accounts'
import { PROVIDERS_ENDPOINT_MAP } from './pages/providers'

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

  it('should use google native default endpoint', () => {
    expect(PROVIDERS_ACCOUNTS_DEFAULT_ENDPOINTS.google).toBe('https://generativelanguage.googleapis.com/v1beta')
    expect(PROVIDERS_ENDPOINT_MAP.google).toBe('https://generativelanguage.googleapis.com/v1beta')
  })

  it('should remove hardcoded model defaults from stores', () => {
    const modelsStoreFile = readFileSync(join(process.cwd(), 'src/store/models.ts'), 'utf-8')
    const chatStoreFile = readFileSync(join(process.cwd(), 'src/store/chat.ts'), 'utf-8')
    const chatConstantsFile = readFileSync(join(process.cwd(), 'src/constants/store/chat.ts'), 'utf-8')
    const modelConstantsFile = readFileSync(join(process.cwd(), 'src/constants/store/models.ts'), 'utf-8')

    expect(modelsStoreFile).not.toContain('const defaultModels: Model[] = [')
    expect(chatStoreFile).not.toContain('const DEFAULT_PROVIDERS: ProviderConfig[] = [')
    expect(chatConstantsFile).not.toContain('CHAT_DEFAULT_PROVIDERS: ProviderConfig[] = [')
    expect(modelConstantsFile).not.toContain('STORE_DEFAULT_MODELS: DefaultModel[] = [')
    expect(chatStoreFile).not.toContain('PROVIDERS.value = [...DEFAULT_PROVIDERS]')
    expect(chatStoreFile).not.toContain('PROVIDERS.value = DEFAULT_PROVIDERS.filter')
    expect(modelsStoreFile).not.toContain('models.value = [...defaultModels]')
  })

  it('should remove duplicated task type option blocks from cache view', () => {
    const cacheViewFile = readFileSync(join(process.cwd(), 'src/views/cache/index.vue'), 'utf-8')

    expect(cacheViewFile).not.toContain('<el-option label="事实查询" value="fact" />')
    expect(cacheViewFile).not.toContain('<el-option label="代码生成" value="code" />')
    expect(cacheViewFile).not.toContain('<el-option label="逻辑推理" value="reasoning" />')
  })

  it('should move docs and api-management inline strategy/provider lists into constants', () => {
    const docsViewFile = readFileSync(join(process.cwd(), 'src/views/docs/index.vue'), 'utf-8')
    const apiManagementViewFile = readFileSync(join(process.cwd(), 'src/views/api-management/index.vue'), 'utf-8')

    expect(docsViewFile).not.toContain('const providers = ref([')
    expect(apiManagementViewFile).not.toContain('const strategies = ref([')
  })

  it('should show google endpoint mode hint in providers accounts view', () => {
    const providersAccountsViewFile = readFileSync(join(process.cwd(), 'src/views/providers-accounts/index.vue'), 'utf-8')

    expect(providersAccountsViewFile).toContain('Google 端点模式')
    expect(providersAccountsViewFile).toContain('v1beta（原生）')
    expect(providersAccountsViewFile).toContain('v1beta/openai（兼容）')
  })

  it('should redirect to home page after logout', () => {
    const layoutFile = readFileSync(join(process.cwd(), 'src/components/Layout/index.vue'), 'utf-8')
    const navigationConstantsFile = readFileSync(join(process.cwd(), 'src/constants/navigation.ts'), 'utf-8')

    expect(layoutFile).toContain('POST_LOGOUT_REDIRECT')
    expect(layoutFile).not.toContain("router.push('/login')")
    expect(navigationConstantsFile).toContain("export const POST_LOGOUT_REDIRECT = HOME_ROUTE")
  })

  it('should centralize auth routes and redirects in navigation constants', () => {
    const navigationConstantsFile = readFileSync(join(process.cwd(), 'src/constants/navigation.ts'), 'utf-8')
    const routerFile = readFileSync(join(process.cwd(), 'src/router/index.ts'), 'utf-8')
    const requestFile = readFileSync(join(process.cwd(), 'src/api/request.ts'), 'utf-8')
    const errorHandlerFile = readFileSync(join(process.cwd(), 'src/utils/errorHandler.ts'), 'utf-8')
    const apiManagementFile = readFileSync(join(process.cwd(), 'src/views/api-management/index.vue'), 'utf-8')
    const loginViewFile = readFileSync(join(process.cwd(), 'src/views/login/index.vue'), 'utf-8')

    expect(navigationConstantsFile).toContain("export const HOME_ROUTE = '/'")
    expect(navigationConstantsFile).toContain("export const LOGIN_ROUTE = '/login'")
    expect(navigationConstantsFile).toContain("export const DASHBOARD_ROUTE = '/dashboard'")
    expect(navigationConstantsFile).toContain('export const UNAUTHORIZED_REDIRECT = LOGIN_ROUTE')
    expect(navigationConstantsFile).toContain('export const LOGIN_SUCCESS_REDIRECT = DASHBOARD_ROUTE')

    expect(routerFile).toContain('LOGIN_ROUTE')
    expect(routerFile).toContain('UNAUTHORIZED_REDIRECT')
    expect(routerFile).not.toContain("next('/login')")

    expect(requestFile).toContain('UNAUTHORIZED_REDIRECT')
    expect(requestFile).not.toContain("window.location.href = '/login'")

    expect(errorHandlerFile).toContain('UNAUTHORIZED_REDIRECT')
    expect(errorHandlerFile).not.toContain("router.push('/login')")

    expect(apiManagementFile).toContain('LOGIN_ROUTE')
    expect(apiManagementFile).not.toContain("router.push('/login')")

    expect(loginViewFile).toContain('LOGIN_SUCCESS_REDIRECT')
    expect(loginViewFile).not.toContain("router.push('/dashboard')")
  })

  it('should keep e2e report artifacts out of default local git changes', () => {
    const gitignoreFile = readFileSync(join(process.cwd(), '.gitignore'), 'utf-8')
    const playwrightConfigFile = readFileSync(join(process.cwd(), 'playwright.config.ts'), 'utf-8')

    expect(gitignoreFile).toContain('tests/results/html-report/')
    expect(gitignoreFile).toContain('tests/results/artifacts/')
    expect(gitignoreFile).toContain('tests/results/*.json')
    expect(playwrightConfigFile).toContain('process.env.CI')
    expect(playwrightConfigFile).toContain('const reporters =')
  })

  it('should disallow direct request module imports in migrated views', () => {
    const targets = [
      'src/views/routing/index.vue',
      'src/views/cache/index.vue',
      'src/views/ops/index.vue'
    ]

    for (const file of targets) {
      const content = readFileSync(join(process.cwd(), file), 'utf-8')
      expect(content).not.toContain("from '@/api/request'")
    }
  })

  it('should disallow fallback response unwrapping patterns in migrated views', () => {
    const targets = [
      'src/views/routing/index.vue',
      'src/views/cache/index.vue',
      'src/views/ops/index.vue'
    ]

    for (const file of targets) {
      const content = readFileSync(join(process.cwd(), file), 'utf-8')
      expect(content).not.toMatch(/data\??\.data\s*\|\|\s*data/)
      expect(content).not.toMatch(/res\??\.data\s*\|\|\s*\{\}/)
      expect(content).not.toMatch(/res\??\.data\s*\|\|\s*\[\]/)
    }
  })

  it('should migrate business localStorage persistence to settings api', () => {
    const routingViewFile = readFileSync(join(process.cwd(), 'src/views/routing/index.vue'), 'utf-8')
    const settingsViewFile = readFileSync(join(process.cwd(), 'src/views/settings/index.vue'), 'utf-8')
    const modelManagementViewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(routingViewFile).not.toContain('routing_task_mapping_auto_save')
    expect(routingViewFile).not.toContain('routing_task_mapping_last_saved')
    expect(settingsViewFile).not.toContain('ai-gateway-settings')
    expect(modelManagementViewFile).not.toContain('MODEL_MANAGEMENT_STORAGE_KEY')
    expect(modelManagementViewFile).not.toContain('localStorage.setItem(STORAGE_KEY')
  })
})
