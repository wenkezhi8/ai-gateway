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
    return this.page.locator('.add-button, .el-button:has-text("添加账号"), .el-button:has-text("添加"), [data-testid="add-account"]').first();
  }

  async clickAddAccount() {
    const button = await this.getAddAccountButton();
    if (await button.isVisible({ timeout: 3000 })) {
      await button.click();
    }
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
    const dialog = this.page.locator('.el-dialog:visible').first();
    if (!(await dialog.isVisible({ timeout: 1500 }))) {
      return;
    }

    if (accountData.name) {
      await dialog.locator('input[placeholder*="账号名称"], input[placeholder*="请输入账号名称"]').first().fill(accountData.name);
    }
    
    if (accountData.username) {
      const usernameInput = dialog.locator('input[placeholder*="用户名"], input[placeholder*="用户"]').first();
      if (await usernameInput.isVisible()) {
        await usernameInput.fill(accountData.username);
      }
    }
    
    if (accountData.apiKey) {
      await dialog.locator('input[placeholder*="API Key"], input[placeholder*="密钥"], input[placeholder*="key"]').first().fill(accountData.apiKey);
    }
    
    if (accountData.provider) {
      const providerItem = this.page.locator('.el-form-item:has-text("服务商") .el-select').first();
      if (await providerItem.isVisible()) {
        await providerItem.click();
        const option = this.page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: accountData.provider }).first();
        if (await option.isVisible()) {
          await option.click();
        }
      }
    }
    
    if (accountData.description) {
      const desc = dialog.locator('textarea[placeholder*="备注"], textarea[placeholder*="描述"]').first();
      if (await desc.isVisible()) {
        await desc.fill(accountData.description);
      }
    }
  }

  async submitForm() {
    const submit = this.page.locator('.submit-button, .el-button:has-text("确定"), .el-dialog .el-button--primary, [data-testid="submit-form"]').first();
    if (await submit.isVisible()) {
      await submit.click();
      return;
    }
    await this.page.keyboard.press('Enter');
  }

  async getAccountList() {
    return this.page.locator('.account-list, .data-table, .el-table, [data-testid="account-list"]');
  }

  async getAccountItems() {
    return this.page.locator('.account-item, .el-table__row, [data-testid="account-item"]');
  }

  async editAccount(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`).first();
    const btn = accountRow.locator('.edit-button, .el-button:has-text("编辑"), [data-testid*="edit"]').first();
    if (await btn.isVisible()) {
      await btn.click();
    }
  }

  async deleteAccount(accountName: string) {
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`).first();
    const del = accountRow.locator('.delete-button, .el-button:has-text("删除"), [data-testid*="delete"]').first();
    if (await del.isVisible()) {
      await del.click();
      const confirm = this.page.locator('.el-message-box__wrapper:visible .el-message-box__btns .el-button--primary, .el-message-box:visible .el-message-box__btns .el-button--primary, [data-testid="confirm-delete"]').first();
      if (await confirm.isVisible()) {
        await confirm.click();
      }
    }
  }

  async searchAccount(searchTerm: string) {
    await this.page.locator('.search-input input, input[placeholder*="搜索账号名称"], [data-testid="search-input"] input').first().fill(searchTerm);
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
    const accountRow = this.page.locator(`.el-table__row:has-text("${accountName}"), [data-testid*="account"]:has-text("${accountName}")`).first();
    const regen = accountRow.locator('.regenerate-key, [data-testid*="regenerate-key"]').first();
    if (await regen.isVisible()) {
      await regen.click();
      const confirm = this.page.locator('.el-button--primary, .confirm-regenerate, [data-testid="confirm-regenerate"]').first();
      if (await confirm.isVisible()) {
        await confirm.click();
      }
    }
  }

  async isLoaded() {
    await this.page.waitForLoadState('networkidle');
    const list = this.page.locator('.account-list, .data-table, .el-table, [data-testid="account-list"]').first();
    if (await list.isVisible({ timeout: 10000 })) {
      return true;
    }
    return this.page.url().includes('/accounts');
  }

  async waitForDataLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForTimeout(400);
  }

  async getSuccessMessage() {
    return this.page.locator('.el-message--success, .success-message, [data-testid="success-message"]');
  }

  async getErrorMessage() {
    return this.page.locator('.el-message--error, .error-message, [data-testid="error-message"]').first();
  }
}
