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
    return this.page.locator('h1, .welcome, .dashboard-title, [data-testid="dashboard-title"]');
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
    const menuItems = {
      'providers': '.el-menu-item:has-text("服务商管理"), [href*="providers"], [data-testid="nav-providers"]',
      'accounts': '.el-menu-item:has-text("账号管理"), [href*="accounts"], [data-testid="nav-accounts"]',
      'routing': '.el-menu-item:has-text("路由策略"), [href*="routing"], [data-testid="nav-routing"]',
      'cache': '.el-menu-item:has-text("缓存管理"), [href*="cache"], [data-testid="nav-cache"]',
      'alerts': '.el-menu-item:has-text("告警管理"), [href*="alerts"], [data-testid="nav-alerts"]',
      'test-center': '.el-menu-item:has-text("测试中心"), [href*="test-center"], [data-testid="nav-test-center"]',
      'settings': '.el-menu-item:has-text("系统设置"), [href*="settings"], [data-testid="nav-settings"]'
    };

    const selector = menuItems[section.toLowerCase()] || `[data-testid="nav-${section}"]`;
    await this.page.click(selector);
    await this.page.waitForLoadState('networkidle');
  }

  async getQuickActions() {
    return this.page.locator('.quick-action, .action-button, [data-testid*="action"]');
  }

  async getRecentActivity() {
    return this.page.locator('.recent-activity, .activity-list, [data-testid*="activity"]');
  }

  async isLoaded() {
    const welcome = this.page.locator('h1, .welcome, .dashboard-title');
    await welcome.waitFor({ state: 'visible', timeout: 10000 });
    return await welcome.isVisible();
  }

  async waitForDataLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForTimeout(1000); // Wait for any async data loading
  }

  async getSystemStatus() {
    return this.page.locator('.system-status, .status-indicator, [data-testid*="status"]');
  }

  async getPerformanceMetrics() {
    return this.page.locator('.performance-metric, .perf-data, [data-testid*="performance"]');
  }
}