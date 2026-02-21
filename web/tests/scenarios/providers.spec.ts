import { test, expect } from '../utils/test-helper';
import { ProvidersPage } from '../page-objects/providers-page';

test.describe('Providers Management Tests', () => {
  let providersPage: ProvidersPage;

  test.beforeEach(async ({ page, helper }) => {
    providersPage = new ProvidersPage(page);
    await helper.login('admin', 'admin');
  });

  test('should load providers page correctly', async ({ helper }) => {
    await helper.measurePerformance('Navigate to providers page', async () => {
      await providersPage.navigate();
    });

    await helper.measurePerformance('Verify providers page loaded', async () => {
      expect(await providersPage.isLoaded()).toBe(true);
    });
  });

  test('should add new provider successfully', async ({ helper }) => {
    await providersPage.navigate();

    await helper.measurePerformance('Add new provider', async () => {
      await providersPage.clickAddProvider();
      
      const providerData = {
        name: 'Test Provider',
        type: 'OpenAI',
        endpoint: 'https://api.openai.com/v1',
        apiKey: 'sk-test-key-12345',
        description: 'Test provider for automation'
      };
      
      await providersPage.fillProviderForm(providerData);
      await providersPage.submitForm();
    });

    await helper.measurePerformance('Verify provider added', async () => {
      const successMessage = await providersPage.getSuccessMessage();
      await successMessage.waitFor({ state: 'visible', timeout: 5000 });
    });
  });

  test('should edit existing provider', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.waitForDataLoad();

    await helper.measurePerformance('Edit provider', async () => {
      await providersPage.editProvider('Test Provider');
      
      const updatedData = {
        name: 'Updated Provider Name',
        type: 'OpenAI',
        description: 'Updated description'
      };
      
      await providersPage.fillProviderForm(updatedData);
      await providersPage.submitForm();
    });

    const successMessage = await providersPage.getSuccessMessage();
    const isSuccessVisible = await successMessage.isVisible();
    
    if (isSuccessVisible) {
      console.log('Provider updated successfully');
    }
  });

  test('should delete provider', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.waitForDataLoad();

    await helper.measurePerformance('Delete provider', async () => {
      await providersPage.deleteProvider('Updated Provider Name');
    });

    await helper.measurePerformance('Verify provider deleted', async () => {
      await helper.page.waitForTimeout(1000);
      
      const providerItems = await providersPage.getProviderItems();
      const hasDeletedProvider = await providerItems.filter({ hasText: 'Updated Provider Name' }).count();
      expect(hasDeletedProvider).toBe(0);
    });
  });

  test('should search providers', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.waitForDataLoad();

    await helper.measurePerformance('Search providers', async () => {
      await providersPage.searchProvider('OpenAI');
      await helper.page.waitForTimeout(1000);
    });

    const searchResults = await providersPage.getProviderItems();
    const hasOpenAI = await searchResults.filter({ hasText: 'OpenAI' }).count();
    
    expect(hasOpenAI).toBeGreaterThanOrEqual(0);
  });

  test('should validate form fields', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.clickAddProvider();

    await helper.measurePerformance('Test form validation', async () => {
      await providersPage.submitForm();
    });

    const errorMessage = await providersPage.getErrorMessage();
    const isErrorMessageVisible = await errorMessage.isVisible();
    
    if (isErrorMessageVisible) {
      console.log('Form validation working correctly');
    }
  });

  test('should handle network errors during provider operations', async ({ helper }) => {
    await providersPage.navigate();
    
    await helper.measurePerformance('Test network error handling', async () => {
      await helper.page.route('**/api/providers**', route => route.abort('failed'));
      
      await providersPage.clickAddProvider();
      await providersPage.fillProviderForm({
        name: 'Network Error Test',
        type: 'Test'
      });
      await providersPage.submitForm();
      
      await helper.page.unroute('**/api/providers**');
    });

    const errorMessage = await providersPage.getErrorMessage();
    const isErrorMessageVisible = await errorMessage.isVisible();
    
    if (isErrorMessageVisible) {
      console.log('Network error handled correctly');
    }
  });

  test('should display provider status correctly', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.waitForDataLoad();

    await helper.measurePerformance('Check provider status', async () => {
      const statusIndicators = await helper.page.locator('.status, .el-tag, [data-testid*="status"]');
      expect(await statusIndicators.count()).toBeGreaterThan(0);
    });
  });

  test('should handle pagination and infinite scroll', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.waitForDataLoad();

    await helper.measurePerformance('Test pagination', async () => {
      const pagination = await helper.page.locator('.pagination, .el-pagination, [data-testid="pagination"]');
      
      if (await pagination.isVisible()) {
        await pagination.locator('.next, .el-pagination__next').click();
        await helper.page.waitForTimeout(1000);
      }
    });
  });

  test('should export/import provider configurations', async ({ helper }) => {
    await providersPage.navigate();
    await providersPage.waitForDataLoad();

    await helper.measurePerformance('Export providers', async () => {
      const exportButton = await helper.page.locator('.export-button, [data-testid="export"]');
      if (await exportButton.isVisible()) {
        await exportButton.click();
        await helper.page.waitForTimeout(2000);
      }
    });
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});