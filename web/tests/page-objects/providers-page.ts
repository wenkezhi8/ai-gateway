import { Page } from '@playwright/test';

export class ProvidersPage {
  private page: Page;

  constructor(page: Page) {
    this.page = page;
  }

  async navigate() {
    await this.page.goto('/providers');
  }

  async getAddProviderButton() {
    return this.page.locator('.add-button, .el-button:has-text("添加"), [data-testid="add-provider"]');
  }

  async clickAddProvider() {
    await (await this.getAddProviderButton()).click();
  }

  async getProviderForm() {
    return this.page.locator('.provider-form, .el-form, [data-testid="provider-form"]');
  }

  async fillProviderForm(providerData: {
    name: string;
    type: string;
    endpoint?: string;
    apiKey?: string;
    description?: string;
  }) {
    const form = await this.getProviderForm();
    
    if (providerData.name) {
      await this.page.locator('input[name="name"], [placeholder*="名称"], [placeholder*="服务商"]').fill(providerData.name);
    }
    
    if (providerData.type) {
      await this.page.selectOption('select[name="type"], .el-select', providerData.type);
    }
    
    if (providerData.endpoint) {
      await this.page.locator('input[name="endpoint"], [placeholder*="接口"], [placeholder*="endpoint"]').fill(providerData.endpoint);
    }
    
    if (providerData.apiKey) {
      await this.page.locator('input[name="apiKey"], [placeholder*="密钥"], [placeholder*="key"]').fill(providerData.apiKey);
    }
    
    if (providerData.description) {
      await this.page.locator('textarea[name="description"], [placeholder*="描述"]').fill(providerData.description);
    }
  }

  async submitForm() {
    await this.page.click('.submit-button, .el-button:has-text("提交"), [data-testid="submit-form"]');
  }

  async getProviderList() {
    return this.page.locator('.provider-list, .el-table, [data-testid="provider-list"]');
  }

  async getProviderItems() {
    return this.page.locator('.provider-item, .el-table__row, [data-testid="provider-item"]');
  }

  async editProvider(providerName: string) {
    const providerRow = this.page.locator(`.el-table__row:has-text("${providerName}"), [data-testid*="provider"]:has-text("${providerName}")`);
    await providerRow.locator('.edit-button, .el-button:has-text("编辑"), [data-testid*="edit"]').click();
  }

  async deleteProvider(providerName: string) {
    const providerRow = this.page.locator(`.el-table__row:has-text("${providerName}"), [data-testid*="provider"]:has-text("${providerName}")`);
    await providerRow.locator('.delete-button, .el-button:has-text("删除"), [data-testid*="delete"]').click();
    
    await this.page.locator('.el-button--danger, .confirm-delete, [data-testid="confirm-delete"]').click();
  }

  async searchProvider(searchTerm: string) {
    await this.page.fill('.search-input, .el-input__inner:has-ancestor(.search-bar), [data-testid="search-input"]', searchTerm);
  }

  async getProviderStatus(providerName: string) {
    const providerRow = this.page.locator(`.el-table__row:has-text("${providerName}"), [data-testid*="provider"]:has-text("${providerName}")`);
    return providerRow.locator('.status, .el-tag, [data-testid*="status"]');
  }

  async isLoaded() {
    await this.page.waitForLoadState('networkidle');
    const list = this.page.locator('.provider-list, .el-table, [data-testid="provider-list"]');
    await list.waitFor({ state: 'visible', timeout: 10000 });
    return await list.isVisible();
  }

  async waitForDataLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForSelector('.provider-list, .el-table', { state: 'attached' });
  }

  async getSuccessMessage() {
    return this.page.locator('.el-message--success, .success-message, [data-testid="success-message"]');
  }

  async getErrorMessage() {
    return this.page.locator('.el-message--error, .error-message, [data-testid="error-message"]');
  }
}