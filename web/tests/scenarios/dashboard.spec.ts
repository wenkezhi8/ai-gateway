import { test, expect } from '../utils/test-helper';
import { DashboardPage } from '../page-objects/dashboard-page';

test.describe('Dashboard Tests', () => {
  let dashboardPage: DashboardPage;

  test.use({ storageState: { cookies: [], origins: [] } });

  test.beforeEach(async ({ page, helper }) => {
    dashboardPage = new DashboardPage(page);
    await helper.login('admin', 'admin123');
  });

  test('should load dashboard page correctly', async ({ page, helper }) => {
    await helper.measurePerformance('Navigate to dashboard', async () => {
      await dashboardPage.navigate();
    });

    await helper.measurePerformance('Verify dashboard loaded', async () => {
      expect(await dashboardPage.isLoaded()).toBe(true);
    });

    const title = await dashboardPage.getPageTitle();
    expect(title).toContain('监控仪表盘');
  });

  test('should display all dashboard components', async ({ helper }) => {
    await dashboardPage.navigate();
    await dashboardPage.waitForDataLoad();

    await helper.measurePerformance('Verify stat cards', async () => {
      const statCards = await dashboardPage.getStatCards();
      expect(await statCards.count()).toBeGreaterThan(0);
    });

    await helper.measurePerformance('Verify charts', async () => {
      const charts = await dashboardPage.getCharts();
      expect(await charts.count()).toBeGreaterThan(0);
    });

    await helper.measurePerformance('Verify navigation menu', async () => {
      const navMenu = await dashboardPage.getNavigationMenu();
      expect(await navMenu.isVisible()).toBe(true);
    });
  });

  test('should navigate to all sections', async ({ helper }) => {
    await dashboardPage.navigate();
    
    const sections = ['providers', 'accounts', 'routing', 'cache', 'alerts', 'settings'];
    
    for (const section of sections) {
      await helper.measurePerformance(`Navigate to ${section}`, async () => {
        await dashboardPage.navigateToSection(section);
      });

      const currentUrl = helper.page.url();
      expect(currentUrl).toContain(section);
      
      await helper.page.goBack();
      await helper.page.waitForLoadState('networkidle');
    }
  });

  test('should display real-time data updates', async ({ helper }) => {
    await dashboardPage.navigate();
    await dashboardPage.waitForDataLoad();

    const initialMetrics = await dashboardPage.getPerformanceMetrics();
    const initialCount = await initialMetrics.count();

    await helper.page.waitForTimeout(3000);

    await helper.measurePerformance('Check for data updates', async () => {
      const updatedMetrics = await dashboardPage.getPerformanceMetrics();
      const updatedCount = await updatedMetrics.count();
      
      expect(updatedCount).toBeGreaterThanOrEqual(initialCount);
    });
  });

  test('should handle dashboard refresh', async ({ helper }) => {
    await dashboardPage.navigate();
    await dashboardPage.waitForDataLoad();

    await helper.measurePerformance('Refresh dashboard', async () => {
      await helper.page.reload();
      await helper.page.waitForLoadState('networkidle');
    });

    expect(await dashboardPage.isLoaded()).toBe(true);
  });

  test('should display quick actions correctly', async ({ helper }) => {
    await dashboardPage.navigate();
    await dashboardPage.waitForDataLoad();

    await helper.measurePerformance('Verify quick actions', async () => {
      const quickActions = await dashboardPage.getQuickActions();
      expect(await quickActions.count()).toBeGreaterThan(0);
    });
  });

  test('should show system status indicators', async ({ helper }) => {
    await dashboardPage.navigate();
    await dashboardPage.waitForDataLoad();

    await helper.measurePerformance('Verify system status', async () => {
      const systemStatus = await dashboardPage.getSystemStatus();
      expect(await systemStatus.count()).toBeGreaterThan(0);
    });
  });

  test('should handle responsive layout', async ({ helper }) => {
    await dashboardPage.navigate();
    await dashboardPage.waitForDataLoad();

    const viewports = [
      { width: 1920, height: 1080 }, // Desktop
      { width: 768, height: 1024 },  // Tablet
      { width: 375, height: 667 }    // Mobile
    ];

    for (const viewport of viewports) {
      await helper.measurePerformance(`Test viewport ${viewport.width}x${viewport.height}`, async () => {
        await helper.page.setViewportSize(viewport);
        await helper.page.waitForTimeout(500);
      });

      const statCards = await dashboardPage.getStatCards();
      expect(await statCards.count()).toBeGreaterThan(0);
    }
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});
