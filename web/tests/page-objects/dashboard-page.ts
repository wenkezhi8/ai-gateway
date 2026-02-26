import { Page } from '@playwright/test';

export class DashboardPage {
  private page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async navigate() {
    await this.page.goto('/dashboard');
  }

  async getPageTitle() {
    return this.page.title();
  }

  async getWelcomeMessage() {
    return this.page.locator('h1, .welcome, .dashboard-title, .stats-row, [data-testid="dashboard-title"]');
  }

  async getStatCards() {
    return this.page.locator('.stat-card, .metric-card, .dashboard-card, [data-testid*="stat"], [data-testid*="metric"]');
  }

  async getCharts() {
    return this.page.locator('.chart, .echarts-container, [data-testid*="chart"], canvas');
  }

  async getNavigationMenu() {
    return this.page.locator('.nav-menu, .sidebar-menu, .el-menu, [role="navigation"]');
  }

  async navigateToSection(section: string) {
    const targetPath = section.toLowerCase() === 'providers' ? '/providers' : section.toLowerCase() === 'accounts' ? '/accounts' : `/${section.toLowerCase()}`;
    try {
      await this.page.goto(targetPath);
    } catch {
      await this.page.waitForTimeout(200);
      await this.page.goto(targetPath);
    }
    await this.page.waitForLoadState('networkidle');
  }

  async getQuickActions() {
    return this.page.locator('.quick-action, .action-button, .el-button, [data-testid*="action"]');
  }

  async getRecentActivity() {
    return this.page.locator('.recent-activity, .activity-list, [data-testid*="activity"]');
  }

  async isLoaded() {
    const welcome = this.page.locator('h1, .welcome, .dashboard-title, .stats-row .stat-card').first();
    await welcome.waitFor({ state: 'visible', timeout: 10000 });
    return await welcome.isVisible();
  }

  async waitForDataLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForTimeout(1000); // Wait for any async data loading
  }

  async getSystemStatus() {
    return this.page.locator('.system-status, .status-indicator, .realtime-stats .realtime-item, [data-testid*="status"]');
  }

  async getPerformanceMetrics() {
    return this.page.locator('.performance-metric, .perf-data, .realtime-stats .realtime-item, .stat-card, [data-testid*="performance"]');
  }
}
