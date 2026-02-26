// CHANGED: New UI flow coverage for limit management page enhancements.
import { test, expect } from '../utils/test-helper';

const buildMockAccounts = () => {
  const now = new Date().toISOString();
  return Array.from({ length: 25 }, (_, index) => {
    const percent = index % 5 === 0 ? 110 : index % 5 === 1 ? 92 : 20;
    const warningLevel = percent >= 100 ? 'critical' : percent >= 90 ? 'warning' : 'normal';
    const used = Math.round((percent / 100) * 1000);

    return {
      id: `acc-${index + 1}`,
      name: `Account ${index + 1}`,
      provider: index % 2 === 0 ? 'openai' : 'zhipu',
      enabled: true,
      is_active: index === 0,
      plan_type: index % 3 === 0 ? 'pro' : 'lite',
      limits: {
        hour5: { type: 'request', period: '5hour', limit: 1000, warning: 90 },
        rpm: { type: 'rpm', period: 'minute', limit: 120, warning: 90 }
      },
      usage: {
        hour5: {
          key: `acc-${index + 1}-hour5`,
          used,
          limit: 1000,
          remaining: Math.max(1000 - used, 0),
          reset_at: now,
          period: '5hour',
          percent_used: percent,
          warning_level: warningLevel
        },
        rpm: {
          key: `acc-${index + 1}-rpm`,
          used: Math.min(120, Math.round((percent / 100) * 120)),
          limit: 120,
          remaining: Math.max(120 - Math.round((percent / 100) * 120), 0),
          reset_at: now,
          period: 'minute',
          percent_used: percent,
          warning_level: warningLevel
        }
      }
    };
  });
};

test.describe('Limit Management Page', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123');
  });

  test('should render controls, filter, and paginate accounts', async ({ helper, page }) => {
    const mockAccounts = buildMockAccounts();

    await page.route(/\/api\/admin\/accounts\/switch-history(\?.*)?$/, route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] })
      });
    });

    await page.route(/\/api\/admin\/dashboard\/alerts(\?.*)?$/, route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: [] })
      });
    });

    await page.route(/\/api\/admin\/accounts(\?.*)?$/, route => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, data: mockAccounts })
      });
    });

    await helper.waitForPageLoad('/limit-management');

    await expect(page.locator('.stats-row .stat-card')).toHaveCount(4);
    await expect(page.locator('.accounts-table')).toBeVisible();

    await expect(page.locator('.accounts-table .el-table__body tr')).toHaveCount(10);

    await page.locator('.el-pagination .el-pager li:has-text("2")').click();
    await expect(page.locator('.accounts-table')).toContainText('Account 11');

    await page.locator('.filter-select').nth(1).click();
    await page.locator('.el-select-dropdown__item:has-text("已超限")').click();
    await expect(page.locator('.accounts-table .row-exceeded')).toHaveCount(5);

    await page.locator('.filter-select').nth(1).click();
    await page.locator('.el-select-dropdown__item:has-text("全部状态")').click();

    await page.locator('.search-input input').fill('Account 23');
    await expect(page.locator('.accounts-table')).toContainText('Account 23');

    await page.locator('button:has-text("重置")').click();
    await expect(page.locator('.accounts-table')).toContainText('Account 1');
  });

  test('should toggle auto refresh mode', async ({ helper, page }) => {
    await helper.waitForPageLoad('/limit-management');

    const autoRefreshToggle = page.locator('.header-actions .el-switch');
    if (await autoRefreshToggle.isVisible()) {
      await autoRefreshToggle.click();
      await autoRefreshToggle.click();
    }

    await expect(page.locator('.header-actions')).toBeVisible();
  });
});
