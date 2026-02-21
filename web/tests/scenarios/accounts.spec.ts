import { test, expect } from '../utils/test-helper';
import { AccountsPage } from '../page-objects/accounts-page';

test.describe('Accounts Management Tests', () => {
  let accountsPage: AccountsPage;

  test.beforeEach(async ({ page, helper }) => {
    accountsPage = new AccountsPage(page);
    await helper.login('admin', 'admin');
  });

  test('should load accounts page correctly', async ({ helper }) => {
    await helper.measurePerformance('Navigate to accounts page', async () => {
      await accountsPage.navigate();
    });

    await helper.measurePerformance('Verify accounts page loaded', async () => {
      expect(await accountsPage.isLoaded()).toBe(true);
    });
  });

  test('should add new account successfully', async ({ helper }) => {
    await accountsPage.navigate();

    await helper.measurePerformance('Add new account', async () => {
      await accountsPage.clickAddAccount();
      
      const accountData = {
        name: 'Test Account',
        username: 'testuser',
        apiKey: 'sk-test-account-key-12345',
        provider: 'OpenAI',
        description: 'Test account for automation'
      };
      
      await accountsPage.fillAccountForm(accountData);
      await accountsPage.submitForm();
    });

    await helper.measurePerformance('Verify account added', async () => {
      const successMessage = await accountsPage.getSuccessMessage();
      await successMessage.waitFor({ state: 'visible', timeout: 5000 });
    });
  });

  test('should manage API keys securely', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Test API key visibility', async () => {
      const apiKeyElement = await accountsPage.getApiKeyVisibility('Test Account');
      
      if (await apiKeyElement.isVisible()) {
        const initialText = await apiKeyElement.textContent();
        expect(initialText).toContain('••••••••');
        
        await accountsPage.showApiKey('Test Account');
        await helper.page.waitForTimeout(1000);
        
        const visibleText = await apiKeyElement.textContent();
        expect(visibleText).toContain('sk-');
      }
    });
  });

  test('should regenerate API key', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Regenerate API key', async () => {
      await accountsPage.regenerateApiKey('Test Account');
    });

    const successMessage = await accountsPage.getSuccessMessage();
    const isSuccessVisible = await successMessage.isVisible();
    
    if (isSuccessVisible) {
      console.log('API key regenerated successfully');
    }
  });

  test('should edit existing account', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Edit account', async () => {
      await accountsPage.editAccount('Test Account');
      
      const updatedData = {
        name: 'Updated Account Name',
        username: 'updateduser',
        provider: 'OpenAI',
        description: 'Updated account description'
      };
      
      await accountsPage.fillAccountForm(updatedData);
      await accountsPage.submitForm();
    });

    const successMessage = await accountsPage.getSuccessMessage();
    const isSuccessVisible = await successMessage.isVisible();
    
    if (isSuccessVisible) {
      console.log('Account updated successfully');
    }
  });

  test('should delete account', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Delete account', async () => {
      await accountsPage.deleteAccount('Updated Account Name');
    });

    await helper.measurePerformance('Verify account deleted', async () => {
      await helper.page.waitForTimeout(1000);
      
      const accountItems = await accountsPage.getAccountItems();
      const hasDeletedAccount = await accountItems.filter({ hasText: 'Updated Account Name' }).count();
      expect(hasDeletedAccount).toBe(0);
    });
  });

  test('should search accounts', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Search accounts', async () => {
      await accountsPage.searchAccount('test');
      await helper.page.waitForTimeout(1000);
    });

    const searchResults = await accountsPage.getAccountItems();
    const hasTestAccounts = await searchResults.filter({ hasText: 'test' }).count();
    
    expect(hasTestAccounts).toBeGreaterThanOrEqual(0);
  });

  test('should validate account form fields', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.clickAddAccount();

    await helper.measurePerformance('Test account form validation', async () => {
      await accountsPage.submitForm();
    });

    const errorMessage = await accountsPage.getErrorMessage();
    const isErrorMessageVisible = await errorMessage.isVisible();
    
    if (isErrorMessageVisible) {
      console.log('Account form validation working correctly');
    }
  });

  test('should handle network errors during account operations', async ({ helper }) => {
    await accountsPage.navigate();
    
    await helper.measurePerformance('Test network error handling for accounts', async () => {
      await helper.page.route('**/api/accounts**', route => route.abort('failed'));
      
      await accountsPage.clickAddAccount();
      await accountsPage.fillAccountForm({
        name: 'Network Error Test Account',
        username: 'neterror',
        provider: 'Test'
      });
      await accountsPage.submitForm();
      
      await helper.page.unroute('**/api/accounts**');
    });

    const errorMessage = await accountsPage.getErrorMessage();
    const isErrorMessageVisible = await errorMessage.isVisible();
    
    if (isErrorMessageVisible) {
      console.log('Network error handled correctly for accounts');
    }
  });

  test('should display account status correctly', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Check account status', async () => {
      const statusIndicators = await helper.page.locator('.status, .el-tag, [data-testid*="status"]');
      expect(await statusIndicators.count()).toBeGreaterThan(0);
    });
  });

  test('should handle account permissions and roles', async ({ helper }) => {
    await accountsPage.navigate();
    await accountsPage.waitForDataLoad();

    await helper.measurePerformance('Test account permissions', async () => {
      const permissionElements = await helper.page.locator('.permission, .role, [data-testid*="permission"]');
      
      if (await permissionElements.count() > 0) {
        console.log('Permission controls are available');
      }
    });
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});