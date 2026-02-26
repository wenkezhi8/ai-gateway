import { test, expect } from '../utils/test-helper'

test.describe('Cache Management Page', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123')
  })

  test('visual: cache type horizontal layout baseline', async ({ helper, browserName }) => {
    test.skip(browserName !== 'chromium', '视觉基线仅在 chromium 维护')

    await helper.page.setViewportSize({ width: 1366, height: 900 })
    await helper.page.goto('/cache')
    await helper.page.waitForLoadState('networkidle')
    await helper.page.waitForTimeout(400)

    const typePanel = helper.page.locator('.types-panel')
    await expect(typePanel).toBeVisible()
  })

  // FIX TEST: 验证筛选变更后分页重置
  test('should reset to first page when task type filter changes', async ({ helper }) => {
    await helper.page.goto('/cache')
    await helper.page.waitForLoadState('networkidle')

    await helper.page.getByRole('tab', { name: '缓存内容' }).click()
    await helper.page.waitForTimeout(300)

    const token = await helper.page.evaluate(() => localStorage.getItem('token'))
    expect(token).toBeTruthy()
    const runId = Date.now()
    const entriesPagination = helper.page
      .locator('.entries-table')
      .locator('xpath=following-sibling::div[contains(@class,"pagination")]')

    await helper.page.evaluate(async ({ authToken, runId }) => {
      const headers = {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`
      }

      const createEntry = async (taskType: string, index: number) => {
        const response = await fetch('/api/admin/cache/test-entry', {
          method: 'POST',
          headers,
          body: JSON.stringify({
            task_type: taskType,
            user_message: `${taskType} user ${runId}-${index}`,
            ai_response: `${taskType} ai ${runId}-${index}`,
            model: 'gpt-4o',
            provider: 'openai',
            ttl: 3600
          })
        })

        if (!response.ok) {
          throw new Error(`warmup failed for ${taskType}-${index}`)
        }
      }

      for (let i = 0; i < 25; i += 1) {
        await createEntry('fact', i)
      }

      await createEntry('math', 0)
    }, { authToken: token, runId })

    const entriesToolbar = helper.page.locator('.entries-toolbar')
    const refreshBtn = helper.page.getByRole('button', { name: '刷新' }).first()
    if (await refreshBtn.isVisible()) {
      await refreshBtn.click()
    }

    const taskTypeSelect = entriesToolbar.locator('.el-select').first()
    await taskTypeSelect.click()
    await helper.page.locator('.el-select-dropdown__item').filter({ hasText: '事实查询' }).click()

    const pageTwo = entriesPagination.locator('.el-pager li.number').filter({ hasText: '2' }).first()
    if (await pageTwo.isVisible()) {
      await pageTwo.click()
      await expect(pageTwo).toHaveClass(/is-active/)
    }

    await taskTypeSelect.click()
    await helper.page.locator('.el-select-dropdown__item').filter({ hasText: '数学计算' }).click()

    const activePage = entriesPagination.locator('.el-pager li.number.is-active')
    if (await activePage.count()) {
      await expect(activePage.first()).toHaveText('1')
    } else {
      await expect(entriesPagination).not.toBeVisible()
    }
    await expect(helper.page.locator('.entries-table')).toBeVisible()
  })
})
