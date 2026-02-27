import { Page } from '@playwright/test';
import { LOGIN_ROUTE } from '../../src/constants/navigation';

export class LoginPage {
  private page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async navigate() {
    await this.page.goto(LOGIN_ROUTE);
  }

  async fillCredentials(username: string, password: string) {
    await this.page.fill('input[type="text"], input[name="username"], [placeholder*="用户名"], [placeholder*="用户"], [placeholder*="账号"]', username);
    await this.page.fill('input[type="password"], input[name="password"], [placeholder*="密码"]', password);
  }

  async submitLogin() {
    await this.page.click('button[type="submit"], .login-button, [role="button"]:has-text("登录"), [role="button"]:has-text("Login")');
  }

  async login(username: string, password: string) {
    await this.navigate();
    await this.fillCredentials(username, password);
    await this.submitLogin();
    await this.page.waitForURL(url => {
      const path = new URL(url.toString()).pathname;
      return path !== LOGIN_ROUTE;
    });
  }

  async getLoginButton() {
    return this.page.locator('button[type="submit"], .login-button, [role="button"]:has-text("登录"), [role="button"]:has-text("Login")');
  }

  async getErrorMessage() {
    return this.page.locator('.error-message, .alert-error, [role="alert"]').first();
  }

  async isVisible() {
    return await this.page.isVisible('input[type="text"], input[type="password"]');
  }
}
