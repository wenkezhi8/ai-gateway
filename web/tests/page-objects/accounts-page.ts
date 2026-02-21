import { Page } from '@playwright/test';

export class AccountsPage {
  private page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async navigate() {
    await this.page.goto('/accounts');
  }

  async getAddAccountButton() {
    return this.page.locator('.add-button, .el-button:has-text("添加"), [data-testid="add-account"]');
  }

  async clickAddAccount() {
    await (await this.getAddAccountButton()).click();
  }

  async getAccountForm() {
    return this.page.locator('.account-form, .el-form, [data-testid="account-form"]');
  }

  async fillAccountForm(accountData: {
    name: string;
    username?: string;
    apiKey?: string;
    provider?: string;
    description?: string;
  }) {
    if (accountData.name) {
      await this.page.locator('input[name="name"], [placeholder*="名称"], [placeholder*="账号"]').fill(accountData.name);
    }
    
    if (accountData.username) {
      await this.page.locator('input[name="username"], [placeholder*="用户名"], [placeholder*="用户"]').fill(accountData.username);
    }
    
    if (accountData.apiKey) {
      await this.page.locator('input[name="apiKey"], [placeholder*="密钥"], [placeholder*="key"]').fill(accountData.apiKey);
    }
    
    if (accountData.provider) {
      await this.page.selectOption('select[name="provider"], .el-select', accountData.provider);
    }
    
    if (accountData.description) {
      await this.page.locator('textarea[name="description"], [placeholder*="描述"]').fill(accountData.description);
    }
  }

  async submitForm() {
    await this.page.click('.submit-button, .el-button:has-text("提交"), [data-testid="submit-form"]');
  }

  async getAccountList() {
    return this.page.locator('.account-list, .el-table, [data-testid="account-list"]');
  }

  async getAccountItems() {
    return this.page.locator('.account-item, .el-table__row, [data-testid="account-item"]');
  }

  async editAccount(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`);
    await accountRow.locator('.edit-button, .el-button:has-text("编辑"), [data-testid*="edit"]').click();
  }

  async deleteAccount(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`);
    await accountRow.locator('.delete-button, .el-button:has-text("删除"), [data-testid*="delete"]').click();
    
    await this.page.locator('.el-button--danger, .confirm-delete, [data-testid="confirm-delete"]').click();
  }

  async searchAccount(searchTerm: string) {
    await this.page.fill('.search-input, .el-input__inner:has-ancestor(.search-bar), [data-testid="search-input"]', searchTerm);
  }

  async getAccountStatus(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`);
    return accountRow.locator('.status, .el-tag, [data-testid*="status"]');
  }

  async getApiKeyVisibility(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`);
    return accountRow.locator('.api-key, [data-testid*="api-key"]');
  }

  async showApiKey(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`);
    await accountRow.locator('.show-key, .el-icon-view, [data-testid*="show-key"]').click();
  }

  async regenerateApiKey(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`);
    await accountRow.locator('.regenerate-key, [data-testid*="regenerate-key"]').click();
    
    await this.page.locator('.el-button--primary, .confirm-regenerate, [data-testid="confirm-regenerate"]').click();
  }

  async isLoaded() {
    await this.page.waitForLoadState('networkidle');
    const list = this.page.locator('.account-list, .el-table, [data-testid="account-list"]');
    await list.waitFor({ state: 'visible', timeout: 10000 });
    return await list.isVisible();
  }

  async waitForDataLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForSelector('.account-list, .el-table', { state: 'attached' });
  }

  async getSuccessMessage() {
    return this.page.locator('.el-message--success, .success-message, [data-testid="success-message"]');
  }

  async getErrorMessage() {
    return this.page.locator('.el-message--error, .error-message, [data-testid="error-message"]');
  }
}