import { test as base, expect, Page, BrowserContext } from '@playwright/test';

export interface PerformanceMetrics {
  action: string;
  duration: number;
  timestamp: Date;
  url?: string;
  status: 'pass' | 'fail' | 'slow';
  error?: string;
}

export class TestHelper {
  public page: Page;
  private context: BrowserContext;
  private metrics: PerformanceMetrics[] = [];
  private readonly SLOW_THRESHOLD = 3000; // 3 seconds

  constructor(page: Page, context: BrowserContext) {
    this.page = page;
    this.context = context;
  }

  async measurePerformance<T>(
    actionName: string,
    action: () => Promise<T>
  ): Promise<T> {
    const startTime = Date.now();
    let result: T;
    let error: Error | undefined;

    try {
      result = await action();
    } catch (e) {
      error = e as Error;
      throw e;
    } finally {
      const duration = Date.now() - startTime;
      const status = error ? 'fail' : duration > this.SLOW_THRESHOLD ? 'slow' : 'pass';
      
      this.metrics.push({
        action: actionName,
        duration,
        timestamp: new Date(),
        url: this.page.url(),
        status,
        error: error?.message
      });

      console.log(`[PERF] ${actionName}: ${duration}ms (${status.toUpperCase()})`);
    }

    return result!;
  }

  async waitForPageLoad(url: string): Promise<void> {
    await this.measurePerformance(`Navigate to ${url}`, async () => {
      await this.page.goto(url, { waitUntil: 'networkidle' });
      await this.page.waitForLoadState('domcontentloaded');
    });
  }

  async clickElement(selector: string, options?: { waitFor?: boolean }): Promise<void> {
    const actionName = `Click ${selector}`;
    await this.measurePerformance(actionName, async () => {
      if (options?.waitFor !== false) {
        await this.page.waitForSelector(selector, { state: 'visible' });
      }
      await this.page.click(selector);
    });
  }

  async fillForm(selector: string, value: string): Promise<void> {
    const actionName = `Fill ${selector}`;
    await this.measurePerformance(actionName, async () => {
      await this.page.waitForSelector(selector, { state: 'visible' });
      await this.page.fill(selector, value);
    });
  }

  async waitForElement(selector: string, timeout?: number): Promise<void> {
    const actionName = `Wait for ${selector}`;
    await this.measurePerformance(actionName, async () => {
      await this.page.waitForSelector(selector, { state: 'visible', timeout });
    });
  }

  async captureError(action: string, error: Error): Promise<void> {
    const screenshot = await this.page.screenshot({
      path: `tests/results/screenshots/error-${Date.now()}.png`,
      fullPage: true
    });
    
    console.error(`[ERROR] ${action}: ${error.message}`);
    this.metrics.push({
      action,
      duration: 0,
      timestamp: new Date(),
      url: this.page.url(),
      status: 'fail',
      error: error.message
    });
  }

  async checkNetworkErrors(): Promise<string[]> {
    const errors: string[] = [];
    
    this.page.on('response', response => {
      if (response.status() >= 400) {
        errors.push(`HTTP ${response.status()}: ${response.url()}`);
      }
    });

    this.page.on('requestfailed', request => {
      errors.push(`Request failed: ${request.url()} - ${request.failure()?.errorText}`);
    });

    return errors;
  }

  getMetrics(): PerformanceMetrics[] {
    return this.metrics;
  }

  getSlowOperations(): PerformanceMetrics[] {
    return this.metrics.filter(m => m.status === 'slow');
  }

  getFailedOperations(): PerformanceMetrics[] {
    return this.metrics.filter(m => m.status === 'fail');
  }

  generateReport(): string {
    const passed = this.metrics.filter(m => m.status === 'pass').length;
    const failed = this.metrics.filter(m => m.status === 'fail').length;
    const slow = this.metrics.filter(m => m.status === 'slow').length;
    
    let report = `=== Test Performance Report ===\n`;
    report += `Total Operations: ${this.metrics.length}\n`;
    report += `Passed: ${passed} | Failed: ${failed} | Slow: ${slow}\n\n`;
    
    if (failed > 0) {
      report += `=== Failed Operations ===\n`;
      this.getFailedOperations().forEach(op => {
        report += `- ${op.action}: ${op.error}\n`;
      });
      report += '\n';
    }
    
    if (slow > 0) {
      report += `=== Slow Operations (>3s) ===\n`;
      this.getSlowOperations().forEach(op => {
        report += `- ${op.action}: ${op.duration}ms\n`;
      });
      report += '\n';
    }
    
    report += `=== All Operations ===\n`;
    this.metrics.forEach(op => {
      report += `- ${op.action}: ${op.duration}ms (${op.status.toUpperCase()})\n`;
    });
    
    return report;
  }

  async logout(): Promise<void> {
    await this.measurePerformance('Logout', async () => {
      const userDropdown = this.page.locator('.user-dropdown, [data-testid="user-dropdown"]').first();
      if (await userDropdown.isVisible({ timeout: 3000 })) {
        await userDropdown.click();
      }

      const logoutButton = this.page.locator('.el-dropdown-menu__item:has-text("退出登录"), .el-dropdown-menu__item:has-text("退出"), [role="menuitem"]:has-text("退出登录"), [role="menuitem"]:has-text("Logout"), .logout').first();
      if (await logoutButton.isVisible({ timeout: 3000 })) {
        await logoutButton.click();
      } else {
        await this.page.evaluate(() => localStorage.removeItem('token'));
        await this.page.goto('/login');
      }

      await this.page.waitForURL('**/login', { timeout: 10000 });
    });
  }

  async login(username: string = 'admin', password: string = 'admin123'): Promise<void> {
    await this.measurePerformance('Login', async () => {
      await this.page.goto('/login');
      await this.page.waitForSelector('input[type="text"], input[name="username"], [placeholder*="用户名"], [placeholder*="用户"], [placeholder*="账号"]');

      await this.page.fill('input[type="text"], input[name="username"], [placeholder*="用户名"], [placeholder*="用户"], [placeholder*="账号"]', username);
      await this.page.fill('input[type="password"], input[name="password"], [placeholder*="密码"]', password);

      // Try multiple strategies to find and click the login button
      const loginButtonSelectors = [
        'button:has-text("登录")',
        'button:has-text("Login")',
        '.login-button',
        '.login-btn',
        'button[type="submit"]',
        '.el-button--primary'
      ];

      let clicked = false;
      for (const selector of loginButtonSelectors) {
        try {
          const locator = this.page.locator(selector).first();
          if (await locator.isVisible({ timeout: 1000 })) {
            await locator.click();
            clicked = true;
            break;
          }
        } catch {
          continue;
        }
      }

      if (!clicked) {
        // Fallback: try to find any button and click it
        const allButtons = this.page.locator('button');
        const count = await allButtons.count();
        for (let i = 0; i < count; i++) {
          const btn = allButtons.nth(i);
          const text = await btn.textContent();
          if (text?.includes('登录') || text?.includes('Login')) {
            await btn.click();
            clicked = true;
            break;
          }
        }
      }

      if (!clicked) {
        // Last resort: press Enter
        await this.page.keyboard.press('Enter');
      }

      await this.page.waitForURL(url => {
        const path = new URL(url.toString()).pathname;
        return path !== '/login';
      }, { timeout: 15000 });
    });
  }
}

export const test = base.extend<{ helper: TestHelper }>({
  helper: async ({ page, context }, use) => {
    const helper = new TestHelper(page, context);
    await use(helper);
  },
});

export { expect };
