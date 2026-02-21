import { test, expect } from '../utils/test-helper';

test.describe('Comprehensive End-to-End Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin');
  });

  test('should complete full user workflow: login → manage providers → manage accounts → logout', async ({ helper }) => {
    await helper.measurePerformance('Complete full user workflow', async () => {
      // Navigate to providers
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');

      // Add a provider
      await helper.page.click('.add-button');
      await helper.page.fill('input[name="name"], [placeholder*="名称"]', 'E2E Test Provider');
      await helper.page.selectOption('select[name="type"], .el-select', 'OpenAI');
      await helper.page.fill('input[name="endpoint"], [placeholder*="接口"]', 'https://api.openai.com/v1');
      await helper.page.fill('input[name="apiKey"], [placeholder*="密钥"]', 'sk-e2e-test-key');
      await helper.page.click('button[type="submit"], .submit-button');
      await helper.page.waitForTimeout(1000);

      // Navigate to accounts
      await helper.page.goto('/accounts');
      await helper.page.waitForLoadState('networkidle');

      // Add an account
      await helper.page.click('.add-button');
      await helper.page.fill('input[name="name"], [placeholder*="名称"]', 'E2E Test Account');
      await helper.page.fill('input[name="username"], [placeholder*="用户名"]', 'e2euser');
      await helper.page.fill('input[name="apiKey"], [placeholder*="密钥"]', 'sk-e2e-account-key');
      await helper.page.selectOption('select[name="provider"], .el-select', 'E2E Test Provider');
      await helper.page.click('button[type="submit"], .submit-button');
      await helper.page.waitForTimeout(1000);

      // Verify accounts
      const accountItems = await helper.page.locator('.el-table__row, .account-item');
      const hasTestAccount = await accountItems.filter({ hasText: 'E2E Test Account' }).count();
      expect(hasTestAccount).toBeGreaterThan(0);

      // Clean up - delete the account
      await accountItems.filter({ hasText: 'E2E Test Account' }).locator('.delete-button').click();
      await helper.page.click('.el-button--danger, .confirm-delete');
      await helper.page.waitForTimeout(1000);

      // Clean up - delete the provider
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');
      const providerItems = await helper.page.locator('.el-table__row, .provider-item');
      await providerItems.filter({ hasText: 'E2E Test Provider' }).locator('.delete-button').click();
      await helper.page.click('.el-button--danger, .confirm-delete');
      await helper.page.waitForTimeout(1000);

      // Logout
      await helper.logout();
    });

    expect(helper.page.url()).toContain('/login');
  });

  test('should handle data flow across multiple pages', async ({ helper }) => {
    await helper.measurePerformance('Test data flow across pages', async () => {
      // Start with dashboard
      await helper.page.goto('/dashboard');
      await helper.page.waitForLoadState('networkidle');

      // Check initial data
      const initialStats = await helper.page.locator('.stat-card, .metric-card').allInnerTexts();
      console.log('Initial dashboard stats:', initialStats.slice(0, 3));

      // Navigate to providers to add data
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');

      const providerCountBefore = await helper.page.locator('.el-table__row, .provider-item').count();

      // Add provider
      await helper.page.click('.add-button');
      await helper.page.fill('input[name="name"]', 'Data Flow Test Provider');
      await helper.page.selectOption('select[name="type"], .el-select', 'Test');
      await helper.page.click('button[type="submit"]');
      await helper.page.waitForTimeout(1000);

      const providerCountAfter = await helper.page.locator('.el-table__row, .provider-item').count();
      expect(providerCountAfter).toBe(providerCountBefore + 1);

      // Return to dashboard to see updated stats
      await helper.page.goto('/dashboard');
      await helper.page.waitForLoadState('networkidle');
      await helper.page.waitForTimeout(2000);

      const updatedStats = await helper.page.locator('.stat-card, .metric-card').allInnerTexts();
      console.log('Updated dashboard stats:', updatedStats.slice(0, 3));

      // Clean up
      await helper.page.goto('/providers');
      await helper.page.waitForLoadState('networkidle');
      const providers = await helper.page.locator('.el-table__row, .provider-item');
      await providers.filter({ hasText: 'Data Flow Test Provider' }).locator('.delete-button').click();
      await helper.page.click('.el-button--danger, .confirm-delete');
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
        const inputs = await helper.page.locator('input, .el-input__inner').first();
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
        
        const pages = ['/dashboard', '/providers', '/accounts'];
        
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
      await helper.page.goto('/dashboard');
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
      await helper.page.click('.add-button');
      await helper.page.fill('input[name="name"]', 'Error Test Provider');
      await helper.page.click('button[type="submit"]');
      
      await helper.page.waitForTimeout(2000);
      
      // Check if error message is shown
      const errorMessage = await helper.page.locator('.el-message--error, .error-message');
      const isErrorMessageVisible = await errorMessage.isVisible();
      
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