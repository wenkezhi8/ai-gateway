import { test, expect } from '../utils/test-helper'

test.describe('Trace Page', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123')
  })

  test('should render answer source column with chinese label', async ({ helper }) => {
    await helper.page.route('**/api/admin/traces**', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          total: 1,
          data: [
            {
              request_id: 'req-e2e-1',
              method: 'POST',
              path: '/api/v1/chat/completions',
              status: 'success',
              duration_ms: 89,
              created_at: '2026-03-09T00:00:00Z',
              step_count: 3,
              answer_source: 'provider_chat',
              task_type: 'analysis',
              model: 'deepseek-chat',
              provider: 'openai'
            }
          ]
        })
      })
    })

    await helper.page.goto('/trace')
    await helper.page.waitForLoadState('networkidle')

    await expect(helper.page.getByText('AI回复来源')).toBeVisible()
    await expect(helper.page.getByText('上游回源')).toBeVisible()
  })
})
