import { test, expect } from '../utils/test-helper';

test.describe('Usage Filters', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123');
  });

  test('should filter usage rows by experiment and domain tags', async ({ helper }) => {
    const page = helper.page;

    await page.route('**/api/admin/cache/stats', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            request_cache: { hits: 0, misses: 0 },
            response_cache: { hits: 0, misses: 0 }
          }
        })
      });
    });

    await page.route('**/admin/usage/logs**', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: [
            {
              id: 1,
              timestamp: Date.now(),
              model: 'gpt-4o-mini',
              provider: 'openai',
              api_key: 'account-a',
              tokens: 120,
              input_tokens: 80,
              output_tokens: 40,
              latency_ms: 300,
              ttft_ms: 120,
              cache_hit: true,
              success: true,
              experiment_tag: 'exp-a',
              domain_tag: 'finance'
            },
            {
              id: 2,
              timestamp: Date.now() - 1000,
              model: 'gpt-4o-mini',
              provider: 'openai',
              api_key: 'account-b',
              tokens: 100,
              input_tokens: 60,
              output_tokens: 40,
              latency_ms: 320,
              ttft_ms: 130,
              cache_hit: false,
              success: true,
              experiment_tag: 'exp-b',
              domain_tag: 'general'
            }
          ]
        })
      });
    });

    await page.goto('/usage');
    await page.waitForLoadState('networkidle');

    const tableBody = page.locator('.usage-table .el-table__body-wrapper').first();

    await expect(page.getByText('exp-a').first()).toBeVisible();
    await expect(page.getByText('exp-b').first()).toBeVisible();

    const experimentSelect = page.locator('.filter-item').filter({ hasText: '实验标签' }).locator('.el-select').first();
    await experimentSelect.click();
    await page.locator('.el-select-dropdown__item').filter({ hasText: 'exp-a' }).first().click();

    await expect(tableBody).toContainText('exp-a');
    await expect(tableBody).toContainText('finance');
    await expect(tableBody).not.toContainText('exp-b');
    await expect(tableBody).not.toContainText('general');

    const domainSelect = page.locator('.filter-item').filter({ hasText: '领域标签' }).locator('.el-select').first();
    await domainSelect.click();
    await page.locator('.el-select-dropdown__item').filter({ hasText: 'finance' }).first().click();

    await expect(tableBody).toContainText('finance');
    await expect(tableBody).not.toContainText('general');
  });
});
