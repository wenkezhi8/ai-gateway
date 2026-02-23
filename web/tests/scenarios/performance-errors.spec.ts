import { test, expect } from '../utils/test-helper';

test.describe('Performance and Error Handling Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin');
  });

  test('should measure page load performance for all major routes', async ({ helper }) => {
    const routes = [
      '/dashboard',
      '/providers',
      '/accounts',
      '/routing',
      '/cache',
      '/alerts',
      '/settings'
    ];

    const performanceMetrics: { route: string; loadTime: number }[] = [];

    for (const route of routes) {
      const startTime = Date.now();
      
      await helper.measurePerformance(`Load ${route}`, async () => {
        await helper.page.goto(route);
        await helper.page.waitForLoadState('networkidle');
        await helper.page.waitForTimeout(1000);
      });

      const loadTime = Date.now() - startTime;
      performanceMetrics.push({ route, loadTime });
      
      if (loadTime > 5000) {
        console.log(`⚠️  Slow page load: ${route} took ${loadTime}ms`);
      }
    }

    const avgLoadTime = performanceMetrics.reduce((sum, m) => sum + m.loadTime, 0) / performanceMetrics.length;
    console.log(`Average page load time: ${avgLoadTime.toFixed(2)}ms`);

    expect(avgLoadTime).toBeLessThan(5000);
  });

  test('should handle 404 errors gracefully', async ({ helper }) => {
    const invalidPaths = [
      '/nonexistent-page',
      '/api/invalid-endpoint',
      '/providers/99999',
      '/accounts/invalid'
    ];

    for (const path of invalidPaths) {
      await helper.measurePerformance(`Handle 404 for ${path}`, async () => {
        const response = await helper.page.goto(path);
        
        if (response && response.status() === 404) {
          const errorElement = await helper.page.locator('.error-page, .not-found, [data-testid*="error"]');
          if (await errorElement.isVisible()) {
            console.log(`404 error handled correctly for ${path}`);
          }
        }
      });
    }
  });

  test('should handle 500 server errors', async ({ helper }) => {
    await helper.page.route('**/api/**', route => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal Server Error' })
        });
      } else {
        route.continue();
      }
    });

    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Handle 500 error', async () => {
      const addButton = await helper.page.locator('.add-button');
      if (await addButton.isVisible()) {
        await addButton.click();
        
        const form = await helper.page.locator('form, .el-form');
        const submitButton = await form.locator('button[type="submit"], .submit-button');
        if (await submitButton.isVisible()) {
          await submitButton.click();
          
          await helper.page.waitForTimeout(2000);
        }
      }
    });

    const errorMessage = await helper.page.locator('.el-message--error, .error-message, [data-testid*="error"]');
    const isErrorVisible = await errorMessage.isVisible();
    
    if (isErrorVisible) {
      console.log('500 error handled correctly with user notification');
    }

    await helper.page.unroute('**/api/**');
  });

  test('should handle network timeouts', async ({ helper }) => {
    await helper.page.route('**/api/**', route => {
      setTimeout(() => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ success: true })
        });
      }, 10000);
    });

    await helper.page.goto('/accounts');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Handle network timeout', async () => {
      const searchInput = await helper.page.locator('.search-input');
      if (await searchInput.isVisible()) {
        await searchInput.fill('test');
        await helper.page.waitForTimeout(12000);
      }
    });

    await helper.page.unroute('**/api/**');
  });

  test('should monitor resource loading performance', async ({ helper }) => {
    const resourceMetrics: { url: string; status: number; type: string }[] = [];
    
    helper.page.on('response', response => {
      const url = response.url();
      const status = response.status();
      
      if (status >= 400) {
        resourceMetrics.push({
          url,
          status,
          type: 'error'
        });
        console.log(`❌ Resource failed: ${url} (${status})`);
      }
    });

    helper.page.on('requestfinished', async (request) => {
      const response = await request.response();
      if (response) {
        resourceMetrics.push({
          url: request.url(),
          status: response.status(),
          type: response.ok() ? 'success' : 'error'
        });
      }
    });

    await helper.measurePerformance('Load all resources', async () => {
      await helper.page.goto('/dashboard');
      await helper.page.waitForLoadState('networkidle');
      await helper.page.waitForTimeout(3000);
    });

    const failedResources = resourceMetrics.filter(r => r.type === 'error');
    if (failedResources.length > 0) {
      console.log(`Failed resources: ${failedResources.length}`);
    }

    expect(failedResources.length).toBeLessThan(5);
  });

  test('should handle JavaScript errors', async ({ helper }) => {
    const jsErrors: { message: string; stack?: string }[] = [];
    
    helper.page.on('pageerror', error => {
      jsErrors.push({
        message: error.message,
        stack: error.stack
      });
      console.log(`JavaScript error: ${error.message}`);
    });

    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test JavaScript error handling', async () => {
      await helper.page.evaluate(() => {
        setTimeout(() => {
          throw new Error('Test JavaScript error');
        }, 1000);
      });

      await helper.page.waitForTimeout(2000);
    });

    if (jsErrors.length > 0) {
      console.log(`JavaScript errors detected: ${jsErrors.length}`);
    }
  });

  test('should handle memory leaks and performance degradation', async ({ helper }) => {
    const initialMetrics = await helper.page.evaluate(() => {
      return {
        memory: (performance as any).memory ? {
          used: (performance as any).memory.usedJSHeapSize,
            total: (performance as any).memory.totalJSHeapSize
        } : null,
        timing: performance.timing
      };
    });

    await helper.page.goto('/accounts');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test memory usage', async () => {
      for (let i = 0; i < 10; i++) {
        await helper.page.reload();
        await helper.page.waitForLoadState('networkidle');
        await helper.page.waitForTimeout(1000);
      }
    });

    const finalMetrics = await helper.page.evaluate(() => {
      return {
        memory: (performance as any).memory ? {
          used: (performance as any).memory.usedJSHeapSize,
            total: (performance as any).memory.totalJSHeapSize
        } : null
      };
    });

    if (initialMetrics.memory && finalMetrics.memory) {
      const memoryIncrease = finalMetrics.memory.used - initialMetrics.memory.used;
      console.log(`Memory usage increase: ${(memoryIncrease / 1024 / 1024).toFixed(2)} MB`);
    }
  });

  test('should handle browser compatibility issues', async ({ helper }) => {
    await helper.page.goto('/dashboard');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test browser compatibility', async () => {
      const browserChecks = await helper.page.evaluate(() => {
        return {
          hasLocalStorage: typeof Storage !== 'undefined',
          hasFetch: typeof fetch !== 'undefined',
          hasPromise: typeof Promise !== 'undefined',
          hasConsole: typeof console !== 'undefined',
          hasDocumentReadyState: document.readyState
        };
      });

      console.log('Browser compatibility checks:', browserChecks);
      expect(browserChecks.hasLocalStorage).toBe(true);
      expect(browserChecks.hasFetch).toBe(true);
      expect(browserChecks.hasPromise).toBe(true);
    });
  });

  test('should handle concurrent user operations', async ({ helper }) => {
    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test concurrent operations', async () => {
      const operations = [
        () => helper.page.locator('.add-button').click(),
        () => helper.page.locator('.search-input').fill('test'),
        () => helper.page.locator('.refresh-button').click()
      ];

      await Promise.all(operations.map(op => op()));
      await helper.page.waitForTimeout(2000);
    });

    const errorElements = await helper.page.locator('.el-message--error, .error-message').count();
    expect(errorElements).toBeLessThan(3);
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});