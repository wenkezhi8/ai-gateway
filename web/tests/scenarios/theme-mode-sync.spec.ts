import { test, expect } from '@playwright/test'

function editionConfigPayload() {
  return {
    type: 'standard',
    display_name: '标准版',
    description: '网关 + 语义缓存',
    dependencies: ['redis', 'ollama'],
    runtime: 'docker',
    dependency_versions: {
      redis: '7.2.0-v18',
      ollama: 'latest',
      qdrant: 'latest'
    },
    features: {
      vector_cache: true,
      vector_db_management: false,
      knowledge_base: false,
      cold_hot_tiering: false
    }
  }
}

test.describe('Theme Mode Sync', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('token', 'e2e-token')
    })

    await page.route('**/api/admin/edition', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: editionConfigPayload() })
      })
    })

    await page.route('**/api/admin/edition/definitions', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [editionConfigPayload()] })
      })
    })

    await page.route('**/api/admin/edition/dependencies', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            redis: { name: 'redis', address: 'localhost:6379', healthy: true, message: 'ok' },
            ollama: { name: 'ollama', address: 'localhost:11434', healthy: true, message: 'ok' },
            qdrant: { name: 'qdrant', address: 'localhost:6333', healthy: false, message: 'disabled' }
          }
        })
      })
    })

    await page.route('**/api/admin/settings/defaults', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            gateway: {
              host: '0.0.0.0',
              port: 8566,
              timeout: 30,
              max_connections: 1000,
              enable_cors: true,
              cors_origins: '*'
            },
            cache: {
              enabled: true,
              type: 'memory',
              default_ttl: 3600,
              max_size: 1024,
              redis: {
                host: 'localhost:6379',
                password: '',
                db: 0
              }
            },
            logging: {
              level: 'info',
              format: 'json',
              outputs: ['console'],
              file_path: '/var/log/ai-gateway',
              max_file_size: 100,
              max_backups: 7
            },
            security: {
              enabled: true,
              type: 'apikey',
              rate_limit: true,
              rate_limit_rpm: 100,
              ip_whitelist: ''
            }
          }
        })
      })
    })

    await page.route('**/api/admin/settings/ui', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true, data: { settings: {} } })
        })
        return
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: { settings: {} } })
      })
    })
  })

  test('should toggle theme mode from header button and keep settings in sync', async ({ page }) => {
    await page.goto('/settings')
    await page.waitForLoadState('networkidle')

    const lightModeButton = page
      .locator('.settings-card:has-text("外观设置") .el-radio-button')
      .filter({ hasText: '亮色' })
      .first()
    await expect(lightModeButton).toBeVisible({ timeout: 10000 })
    await lightModeButton.click()
    await page.waitForFunction(() => document.documentElement.getAttribute('data-theme') === 'light')

    const themeButton = page.locator('.theme-btn').first()
    await expect(themeButton).toBeVisible()
    await themeButton.click()

    await page.waitForFunction(() => {
      const raw = localStorage.getItem('ai-gateway-theme')
      if (!raw) return false
      try {
        const parsed = JSON.parse(raw) as { mode?: string }
        return parsed.mode === 'dark' && document.documentElement.getAttribute('data-theme') === 'dark'
      } catch {
        return false
      }
    })

    const darkModeButton = page
      .locator('.settings-card:has-text("外观设置") .el-radio-button.is-active')
      .filter({ hasText: '暗色' })
      .first()
    await expect(darkModeButton).toBeVisible()
  })
})
