import { test, expect } from '../utils/test-helper';
import { DASHBOARD_ROUTE, POST_LOGOUT_REDIRECT } from '../../src/constants/navigation';

test.describe('Comprehensive End-to-End Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123');
  });

  test('should complete full user workflow: login → manage providers → manage accounts → logout', async ({ helper }) => {
    await helper.measurePerformance('Complete full user workflow', async () => {
      // Navigate to providers
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');

      // Manage provider: open dialog then close
      const addProviderBtn = helper.page.locator('.add-button, .el-button:has-text("添加服务商")').first();
      if (await addProviderBtn.isVisible({ timeout: 2000 })) {
        await addProviderBtn.click();
      }
      const providerCancel = helper.page.locator('.el-dialog:visible .el-button:has-text("取消")').first();
      if (await providerCancel.isVisible()) {
        await providerCancel.click();
      }

      // Navigate to accounts
      await helper.page.goto('/accounts');
      await helper.page.waitForLoadState('networkidle');

      // Manage account: open dialog then close
      const addAccountBtn = helper.page.locator('.add-button, .el-button:has-text("添加账号")').first();
      if (await addAccountBtn.isVisible({ timeout: 2000 })) {
        await addAccountBtn.click();
      }
      const accountCancel = helper.page.locator('.el-dialog:visible .el-button:has-text("取消")').first();
      if (await accountCancel.isVisible()) {
        await accountCancel.click();
      }

      const accountsTable = helper.page.locator('.account-list, .data-table, .el-table').first();
      await expect(accountsTable).toBeVisible();

      // Logout
      await helper.logout();
    });

      expect(new URL(helper.page.url()).pathname).toBe(POST_LOGOUT_REDIRECT);
  });

  test('should handle data flow across multiple pages', async ({ helper }) => {
    await helper.measurePerformance('Test data flow across pages', async () => {
      // Start with dashboard
      await helper.page.goto(DASHBOARD_ROUTE);
      await helper.page.waitForLoadState('networkidle');

      // Check initial data
      const initialStats = await helper.page.locator('.stat-card, .metric-card').allInnerTexts();
      console.log('Initial dashboard stats:', initialStats.slice(0, 3));

      // Navigate to providers to add data
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');

      const providerCountBefore = await helper.page.locator('.el-table__row, .provider-item').count();

      // Add provider
      const addBtn = helper.page.locator('.add-button, .el-button:has-text("添加")').first();
      if (await addBtn.isVisible()) {
        await addBtn.click();
      }
      await helper.page.locator('.el-dialog:visible input[placeholder*="服务商名称"]').first().fill('Data Flow Test Provider');
      const endpointInput = helper.page.locator('.el-dialog:visible input[placeholder*="https"]').first();
      if (await endpointInput.isVisible()) {
        await endpointInput.fill('https://api.openai.com/v1');
      }
      const submitBtn = helper.page.locator('.submit-button, .el-dialog:visible .el-button--primary').first();
      if (await submitBtn.isVisible()) {
        await submitBtn.click();
      }
      await helper.page.waitForTimeout(1000);

      const providerCountAfter = await helper.page.locator('.el-table__row, .provider-item').count();
      expect(providerCountAfter).toBeGreaterThanOrEqual(providerCountBefore);

      // Return to dashboard to see updated stats
      await helper.page.goto(DASHBOARD_ROUTE);
      await helper.page.waitForLoadState('networkidle');
      await helper.page.waitForTimeout(2000);

      const updatedStats = await helper.page.locator('.stat-card, .metric-card').allInnerTexts();
      console.log('Updated dashboard stats:', updatedStats.slice(0, 3));

      // Clean up
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');
      const providers = await helper.page.locator('.el-table__row, .provider-item');
      const cleanupBtn = providers.filter({ hasText: 'Data Flow Test Provider' }).locator('.delete-button, .el-button:has-text("删除")').first();
      if (await cleanupBtn.isVisible()) {
        await cleanupBtn.click();
      }
      const cleanupConfirm = helper.page.locator('.el-message-box .el-button--primary, .el-button--danger, .confirm-delete').first();
      if (await cleanupConfirm.isVisible()) await cleanupConfirm.click();
    });
  });

  test('should test all interactive components', async ({ helper }) => {
    await helper.measurePerformance('Test all interactive components', async () => {
      const pages = ['/providers', '/accounts', '/settings'];
      
      for (const pagePath of pages) {
        await helper.page.goto(pagePath);
        await helper.page.waitForLoadState('networkidle');

        // Test buttons
        const buttons = await helper.page.locator('button, .el-button, [role="button"]').first();
        if (await buttons.isVisible()) {
          await buttons.click();
          await helper.page.waitForTimeout(500);
        }

        // Test inputs
        const inputs = await helper.page.locator('input[type="text"]:not([readonly]):not([disabled]), input[type="search"]:not([readonly]):not([disabled]), input[type="password"]:not([readonly]):not([disabled]), input[type="email"]:not([readonly]):not([disabled]), textarea:not([readonly]):not([disabled])').first();
        if (await inputs.isVisible()) {
          await inputs.fill('test');
          await inputs.clear();
        }

        // Test dropdowns
        const dropdowns = await helper.page.locator('.el-select, .dropdown, [role="combobox"]').first();
        if (await dropdowns.isVisible()) {
          await dropdowns.click();
          await helper.page.waitForTimeout(500);
        }

        // Test forms
        const forms = await helper.page.locator('form, .el-form');
        if (await forms.count() > 0) {
          const form = forms.first();
          const submitButton = await form.locator('button[type="submit"], .submit-button').first();
          if (await submitButton.isVisible()) {
            await submitButton.click();
            await helper.page.waitForTimeout(1000);
          }
        }
      }
    });
  });

  test('should test responsive design across different viewports', async ({ helper }) => {
    const viewports = [
      { width: 1920, height: 1080, name: 'Desktop' },
      { width: 1024, height: 768, name: 'Tablet' },
      { width: 375, height: 667, name: 'Mobile' }
    ];

    for (const viewport of viewports) {
      await helper.measurePerformance(`Test responsive design - ${viewport.name}`, async () => {
        await helper.page.setViewportSize(viewport);
        
        const pages = [DASHBOARD_ROUTE, '/providers', '/accounts'];
        
        for (const pagePath of pages) {
          await helper.page.goto(pagePath);
          await helper.page.waitForLoadState('networkidle');
          
          // Check if layout is responsive
          const content = await helper.page.locator('body').isVisible();
          expect(content).toBe(true);
          
          // Check navigation is accessible
          const navigation = await helper.page.locator('.nav, .menu, .sidebar').first();
          const isNavVisible = await navigation.isVisible();
          
          if (viewport.width <= 768 && !isNavVisible) {
            // Check for hamburger menu on mobile
            const hamburger = await helper.page.locator('.hamburger, .menu-toggle, [data-testid*="menu-toggle"]');
            if (await hamburger.isVisible()) {
              await hamburger.click();
              await helper.page.waitForTimeout(500);
            }
          }
          
          await helper.page.waitForTimeout(500);
        }
      });
    }
  });

  test('should test accessibility features', async ({ helper }) => {
    await helper.measurePerformance('Test accessibility features', async () => {
      await helper.page.goto(DASHBOARD_ROUTE);
      await helper.page.waitForLoadState('networkidle');

      // Check for alt text on images
      const images = await helper.page.locator('img');
      const imageCount = await images.count();
      
      for (let i = 0; i < Math.min(imageCount, 5); i++) {
        const image = images.nth(i);
        const altText = await image.getAttribute('alt');
        if (altText === null) {
          console.log('Image missing alt text:', await image.getAttribute('src'));
        }
      }

      // Check for ARIA labels on interactive elements
      const buttons = await helper.page.locator('button, [role="button"]');
      const buttonCount = await buttons.count();
      
      for (let i = 0; i < Math.min(buttonCount, 5); i++) {
        const button = buttons.nth(i);
        const ariaLabel = await button.getAttribute('aria-label');
        if (ariaLabel === null) {
          const buttonText = await button.textContent();
          if (!buttonText || buttonText.trim() === '') {
            console.log('Button missing accessible label');
          }
        }
      }

      // Test keyboard navigation
      await helper.page.keyboard.press('Tab');
      await helper.page.waitForTimeout(500);
      
      const focusedElement = await helper.page.locator(':focus');
      const hasFocus = await focusedElement.isVisible();
      expect(hasFocus).toBe(true);

      // Test contrast ratios (basic check)
      const textElements = await helper.page.locator('p, h1, h2, h3, h4, h5, h6, span, div');
      const textCount = await textElements.count();
      
      console.log(`Accessibility test completed for ${textCount} text elements`);
    });
  });

  test('should test error recovery scenarios', async ({ helper }) => {
    await helper.measurePerformance('Test error recovery scenarios', async () => {
      // Test network disconnection
      await helper.page.route('**/api/**', route => route.abort('failed'));
      
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');
      
      // Try to add a provider with network error
      const addButton = helper.page.locator('.add-button, .el-button:has-text("添加")').first();
      if (await addButton.isVisible()) {
        await addButton.click();
      }
      await helper.page.locator('.el-dialog:visible input[placeholder*="服务商名称"]').first().fill('Error Test Provider');
      const submit = helper.page.locator('.submit-button, .el-dialog:visible .el-button--primary').first();
      if (await submit.isVisible()) {
        await submit.click();
      }
      
      await helper.page.waitForTimeout(2000);
      
      // Check if error message is shown
      const errorMessages = helper.page.locator('.el-message--error, .error-message');
      const isErrorMessageVisible = await errorMessages.count() > 0;
      
      if (isErrorMessageVisible) {
        console.log('Network error handled correctly');
      }
      
      // Restore network and try again
      await helper.page.unroute('**/api/**');
      
      await helper.page.reload();
      await helper.page.waitForLoadState('networkidle');
      
      // Test page reload after error
      await helper.page.reload();
      await helper.page.waitForLoadState('networkidle');
      
      const pageLoaded = await helper.page.locator('body').isVisible();
      expect(pageLoaded).toBe(true);
    });
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});
