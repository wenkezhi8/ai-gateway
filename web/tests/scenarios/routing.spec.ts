import { test, expect } from '../utils/test-helper';

test.describe('Routing Classifier Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123');
  });

  test('should refresh classifier model list and switch model', async ({ helper }) => {
    const page = helper.page;
    let switchedModel = '';

    await page.route('**/api/admin/router/classifier/models', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            active_model: 'qwen2.5:0.5b-instruct',
            models: ['qwen2.5:0.5b-instruct', 'qwen2.5:1.5b-instruct', 'llama3.2:3b']
          }
        })
      });
    });

    await page.route('**/api/admin/router/classifier/switch', async route => {
      const payload = route.request().postDataJSON() as { model?: string };
      switchedModel = payload?.model || '';
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          message: 'Classifier model switched',
          data: { healthy: true, latency_ms: 30, message: 'ok' }
        })
      });
    });

    await page.goto('/routing');
    await page.waitForLoadState('networkidle');

    const refreshButton = page.getByRole('button', { name: '刷新模型列表' }).first();
    await expect(refreshButton).toBeVisible();
    await refreshButton.click();

    const modelSelect = page.locator('.el-form-item').filter({ hasText: '手动切换模型' }).locator('.el-select').first();
    await modelSelect.click();

    const targetOption = page.locator('.el-select-dropdown__item').filter({ hasText: 'llama3.2:3b' }).first();
    await expect(targetOption).toBeVisible();
    await targetOption.click();

    const switchButton = page.getByRole('button', { name: '切换模型' }).first();
    await switchButton.click();

    await expect.poll(() => switchedModel).toBe('llama3.2:3b');
    await expect(page.locator('.el-form-item').filter({ hasText: '运行模型' }).first()).toContainText('llama3.2:3b');
  });
});
