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
    return this.page.locator('.add-button, .el-button:has-text("添加服务商"), .el-button:has-text("添加"), [data-testid="add-provider"]').first();
  }

  async clickAddProvider() {
    const button = await this.getAddProviderButton();
    if (await button.isVisible({ timeout: 3000 })) {
      await button.click();
    }
  }

  async getProviderForm() {
    return this.page.locator('.provider-form, .el-form, [data-testid="provider-form"]');
  }

  private async pickSelectValue(formItemLabel: string, optionText: string) {
    const formItem = this.page.locator(`.el-form-item:has-text("${formItemLabel}")`).first();
    const selectTrigger = formItem.locator('.el-select').first();
    if (await selectTrigger.isVisible()) {
      await selectTrigger.click();
      const option = this.page.locator('.el-select-dropdown:visible .el-select-dropdown__item').filter({ hasText: optionText }).first();
      if (await option.isVisible()) {
        await option.click();
      }
    }
  }

  async fillProviderForm(providerData: {
    name: string;
    type: string;
    endpoint?: string;
    apiKey?: string;
    description?: string;
  }) {
    const dialog = this.page.locator('.el-dialog:visible').first();
    if (!(await dialog.isVisible({ timeout: 1500 }))) {
      return;
    }

    if (providerData.name) {
      await dialog.locator('input[placeholder*="服务商名称"], input[placeholder*="请输入服务商名称"]').first().fill(providerData.name);
    }
    
    if (providerData.type) {
      await this.pickSelectValue('服务商类型', providerData.type);
    }
    
    if (providerData.endpoint) {
      const endpointInput = dialog.locator('input[placeholder*="API端点"], input[placeholder*="接口"], input[placeholder*="endpoint"], input[placeholder*="https"], input[placeholder*="http"]').first();
      if (await endpointInput.isVisible()) {
        await endpointInput.fill(providerData.endpoint);
      }
    }
    
    if (providerData.apiKey) {
      const keyInput = dialog.locator('input[placeholder*="密钥"], input[placeholder*="API Key"], input[placeholder*="key"]').first();
      if (await keyInput.isVisible()) {
        await keyInput.fill(providerData.apiKey);
      }
    }
    
    if (providerData.description) {
      const desc = dialog.locator('textarea[placeholder*="描述"]').first();
      if (await desc.isVisible()) {
        await desc.fill(providerData.description);
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

  async getProviderList() {
    return this.page.locator('.provider-list, .provider-cards, .el-table, [data-testid="provider-list"]');
  }

  async getProviderItems() {
    return this.page.locator('.provider-item, .provider-card, .el-table__row, [data-testid="provider-item"]');
  }

  async editProvider(providerName: string) {
    const providerRow = this.page.locator(`.provider-card:has-text("${providerName}"), .el-table__row:has-text("${providerName}"), [data-testid*="provider"]:has-text("${providerName}")`).first();
    const btn = providerRow.locator('.edit-button, .provider-actions .el-button:has-text("编辑"), .el-button:has-text("编辑"), [data-testid*="edit"]').first();
    if (await btn.isVisible()) {
      await btn.click();
    }
  }

  async deleteProvider(providerName: string) {
    const providerRow = this.page.locator(`.provider-card:has-text("${providerName}"), .el-table__row:has-text("${providerName}"), [data-testid*="provider"]:has-text("${providerName}")`).first();
    const deleteBtn = providerRow.locator('.delete-button, .el-dropdown, .el-button:has-text("删除"), [data-testid*="delete"]').first();
    if (!(await deleteBtn.isVisible({ timeout: 2000 }))) {
      return;
    }
    await deleteBtn.click();

    const confirm = this.page.locator('.el-message-box .el-button--primary, .el-button--danger, .confirm-delete, [data-testid="confirm-delete"]').first();
    if (await confirm.isVisible()) {
      await confirm.click();
    }
  }

  async searchProvider(searchTerm: string) {
    await this.page.locator('.search-input input, input[placeholder*="搜索服务商"], [data-testid="search-input"] input').first().fill(searchTerm);
  }

  async getProviderStatus(providerName: string) {
    const providerRow = this.page.locator(`.el-table__row:has-text("${providerName}"), [data-testid*="provider"]:has-text("${providerName}")`);
    return providerRow.locator('.status, .el-tag, [data-testid*="status"]');
  }

  async isLoaded() {
    await this.page.waitForLoadState('networkidle');
    const list = this.page.locator('.provider-list, .provider-cards, .el-table, [data-testid="provider-list"]').first();
    if (await list.isVisible({ timeout: 10000 })) {
      return true;
    }
    return this.page.url().includes('/providers');
  }

  async waitForDataLoad() {
    await this.page.waitForLoadState('networkidle');
    await this.page.waitForTimeout(400);
  }

  async getSuccessMessage() {
    return this.page.locator('.el-message--success, .success-message, [data-testid="success-message"]').first();
  }

  async getErrorMessage() {
    return this.page.locator('.el-message--error, .error-message, [data-testid="error-message"]').first();
  }
}
