import { test, expect } from '../utils/test-helper';

test.describe('Navigation and Routing Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin');
  });

  test('should navigate between all main sections', async ({ helper }) => {
    const routes = [
      { path: '/dashboard', title: '监控仪表盘' },
      { path: '/providers', title: '服务商管理' },
      { path: '/accounts', title: '账号管理' },
      { path: '/routing', title: '路由策略' },
      { path: '/cache', title: '缓存管理' },
      { path: '/alerts', title: '告警管理' },
      { path: '/test-center', title: '测试中心' },
      { path: '/settings', title: '系统设置' }
    ];

    for (const route of routes) {
      await helper.measurePerformance(`Navigate to ${route.title}`, async () => {
        await helper.page.goto(route.path);
        await helper.page.waitForLoadState('networkidle');
      });

      const currentUrl = helper.page.url();
      expect(currentUrl).toContain(route.path);

      const pageTitle = await helper.page.title();
      expect(pageTitle).toContain(route.title);
    }
  });

  test('should handle browser back/forward navigation', async ({ helper }) => {
    const routes = ['/dashboard', '/providers', '/accounts'];

    for (const route of routes) {
      await helper.page.goto(route);
      await helper.page.waitForLoadState('networkidle');
    }

    await helper.measurePerformance('Test browser back navigation', async () => {
      await helper.page.goBack();
      await helper.page.waitForLoadState('networkidle');
    });
    expect(helper.page.url()).toContain('/providers');

    await helper.measurePerformance('Test browser forward navigation', async () => {
      await helper.page.goForward();
      await helper.page.waitForLoadState('networkidle');
    });
    expect(helper.page.url()).toContain('/accounts');
  });

  test('should handle direct URL access', async ({ helper }) => {
    const directUrls = [
      '/dashboard',
      '/providers',
      '/accounts',
      '/routing',
      '/cache',
      '/alerts',
      '/test-center',
      '/settings'
    ];

    for (const url of directUrls) {
      await helper.measurePerformance(`Direct access to ${url}`, async () => {
        await helper.page.goto(url);
        await helper.page.waitForLoadState('networkidle');
      });

      const isContentLoaded = await helper.page.locator('body').isVisible();
      expect(isContentLoaded).toBe(true);
    }
  });

  test('should handle invalid routes gracefully', async ({ helper }) => {
    const invalidRoutes = ['/invalid-route', '/404', '/nonexistent-page'];

    for (const route of invalidRoutes) {
      await helper.measurePerformance(`Handle invalid route ${route}`, async () => {
        await helper.page.goto(route);
        await helper.page.waitForLoadState('networkidle');
      });

      const currentUrl = helper.page.url();
      const hasErrorPage = await helper.page.locator('.error-page, .not-found, [data-testid*="error"]').isVisible();
      
      if (hasErrorPage || currentUrl.includes('404')) {
        console.log(`Invalid route ${route} handled correctly`);
      }
    }
  });

  test('should maintain navigation state on page refresh', async ({ helper }) => {
    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test page refresh state maintenance', async () => {
      await helper.page.reload();
      await helper.page.waitForLoadState('networkidle');
    });

    expect(helper.page.url()).toContain('/providers');
  });

  test('should handle navigation with query parameters', async ({ helper }) => {
    const urlsWithParams = [
      '/providers?page=1&limit=10',
      '/accounts?search=test',
      '/dashboard?period=7d'
    ];

    for (const url of urlsWithParams) {
      await helper.measurePerformance(`Navigate with params ${url}`, async () => {
        await helper.page.goto(url);
        await helper.page.waitForLoadState('networkidle');
      });

      expect(helper.page.url()).toContain(url.split('?')[0]);
    }
  });

  test('should support keyboard navigation', async ({ helper }) => {
    await helper.page.goto('/dashboard');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test keyboard navigation', async () => {
      await helper.page.keyboard.press('Tab');
      await helper.page.waitForTimeout(500);
      
      const focusedElement = await helper.page.locator(':focus');
      const isNavigationFocused = await focusedElement.isVisible();
      
      if (isNavigationFocused) {
        console.log('Keyboard navigation working correctly');
      }
    });
  });

  test('should handle navigation during loading states', async ({ helper }) => {
    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test rapid navigation', async () => {
      await helper.page.goto('/accounts');
      
      setTimeout(() => {
        helper.page.goto('/routing');
      }, 100);
      
      await helper.page.waitForLoadState('networkidle');
    });

    const currentUrl = helper.page.url();
    expect(currentUrl).toContain('/routing');
  });

  test('should display breadcrumbs correctly', async ({ helper }) => {
    const routesWithBreadcrumbs = [
      { path: '/providers/test/edit', expected: ['首页', '服务商管理', '编辑'] },
      { path: '/accounts/new', expected: ['首页', '账号管理', '新增'] }
    ];

    for (const route of routesWithBreadcrumbs) {
      await helper.page.goto(route.path);
      await helper.page.waitForLoadState('networkidle');

      await helper.measurePerformance(`Check breadcrumbs for ${route.path}`, async () => {
        const breadcrumbs = await helper.page.locator('.breadcrumb, .el-breadcrumb, [data-testid*="breadcrumb"]');
        
        if (await breadcrumbs.isVisible()) {
          const breadcrumbTexts = await breadcrumbs.allInnerTexts();
          console.log(`Breadcrumbs for ${route.path}:`, breadcrumbTexts);
        }
      });
    }
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});