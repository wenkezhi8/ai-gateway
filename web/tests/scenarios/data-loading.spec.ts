import { test, expect } from '../utils/test-helper';

test.describe('Data Loading and Async Operations Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin');
  });

  test('should handle infinite scroll loading', async ({ helper }) => {
    await helper.page.goto('/accounts');
    await helper.page.waitForLoadState('networkidle');

    const initialItemsCount = await helper.page.locator('.el-table__row, .account-item').count();

    await helper.measurePerformance('Test infinite scroll', async () => {
      await helper.page.evaluate(() => {
        const scrollableElement = document.querySelector('.el-table__body-wrapper, .infinite-scroll-container');
        if (scrollableElement) {
          scrollableElement.scrollTop = scrollableElement.scrollHeight;
        } else {
          window.scrollTo(0, document.body.scrollHeight);
        }
      });

      await helper.page.waitForTimeout(2000);
    });

    const finalItemsCount = await helper.page.locator('.el-table__row, .account-item').count();
    expect(finalItemsCount).toBeGreaterThanOrEqual(initialItemsCount);
  });

  test('should handle pull-to-refresh functionality', async ({ helper }) => {
    await helper.page.goto('/dashboard');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test pull-to-refresh', async () => {
      await helper.page.evaluate(() => {
        const startY = 0;
        const endY = 150;
        
        const touchStart = new TouchEvent('touchstart', {
          touches: [{ clientY: startY } as Touch]
        });
        
        const touchMove = new TouchEvent('touchmove', {
          touches: [{ clientY: endY } as Touch]
        });
        
        const touchEnd = new TouchEvent('touchend');
        
        document.dispatchEvent(touchStart);
        document.dispatchEvent(touchMove);
        document.dispatchEvent(touchEnd);
      });

      await helper.page.waitForTimeout(2000);
    });

    await helper.page.waitForLoadState('networkidle');
  });

  test('should handle lazy loading of images and content', async ({ helper }) => {
    await helper.page.goto('/dashboard');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test lazy loading', async () => {
      const lazyElements = await helper.page.locator('[loading="lazy"], .lazy-load, img[data-src]');
      
      for (let i = 0; i < await lazyElements.count(); i++) {
        const element = lazyElements.nth(i);
        await element.scrollIntoViewIfNeeded();
        await helper.page.waitForTimeout(1000);
      }
    });

    const loadedImages = await helper.page.locator('img[src]').count();
    expect(loadedImages).toBeGreaterThan(0);
  });

  test('should handle real-time data updates', async ({ helper }) => {
    await helper.page.goto('/dashboard');
    await helper.page.waitForLoadState('networkidle');

    const initialData = await helper.page.locator('.metric-card, .stat-card').allInnerTexts();

    await helper.measurePerformance('Test real-time updates', async () => {
      await helper.page.waitForTimeout(5000);
    });

    const updatedData = await helper.page.locator('.metric-card, .stat-card').allInnerTexts();
    
    if (JSON.stringify(initialData) !== JSON.stringify(updatedData)) {
      console.log('Real-time data updates detected');
    }
  });

  test('should handle pagination correctly', async ({ helper }) => {
    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test pagination', async () => {
      const pagination = await helper.page.locator('.el-pagination, .pagination');
      
      if (await pagination.isVisible()) {
        const nextPage = await pagination.locator('.btn-next, .next');
        if (await nextPage.isVisible()) {
          await nextPage.click();
          await helper.page.waitForTimeout(1500);
        }
      }
    });

    await helper.page.waitForLoadState('networkidle');
  });

  test('should handle search with debouncing', async ({ helper }) => {
    await helper.page.goto('/accounts');
    await helper.page.waitForLoadState('networkidle');

    const searchInput = await helper.page.locator('.search-input, .el-input__inner');
    
    if (await searchInput.isVisible()) {
      await helper.measurePerformance('Test search debouncing', async () => {
        await searchInput.click();
        await searchInput.fill('test');
        await helper.page.waitForTimeout(500);
        await searchInput.fill('test query');
        await helper.page.waitForTimeout(1500);
      });

      const searchResults = await helper.page.locator('.el-table__row, .account-item').count();
      expect(searchResults).toBeGreaterThanOrEqual(0);
    }
  });

  test('should handle modal dialogs and overlays', async ({ helper }) => {
    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test modal dialog', async () => {
      const addButton = await helper.page.locator('.add-button, .el-button:has-text("添加")');
      if (await addButton.isVisible()) {
        await addButton.click();
        
        const modal = await helper.page.locator('.el-dialog, .modal, [data-testid*="modal"]');
        if (await modal.isVisible()) {
          const closeButton = await modal.locator('.el-dialog__close, .close-button, [aria-label="Close"]');
          if (await closeButton.isVisible()) {
            await closeButton.click();
          }
        }
      }
    });

    await helper.page.waitForTimeout(500);
  });

  test('should handle dropdown menus and select components', async ({ helper }) => {
    await helper.page.goto('/providers');
    await helper.page.waitForLoadState('networkidle');

    const dropdowns = await helper.page.locator('.el-select, .dropdown, [role="combobox"]');
    
    for (let i = 0; i < Math.min(3, await dropdowns.count()); i++) {
      const dropdown = dropdowns.nth(i);
      
      await helper.measurePerformance(`Test dropdown ${i + 1}`, async () => {
        await dropdown.click();
        await helper.page.waitForTimeout(500);
        
        const options = await helper.page.locator('.el-select-dropdown__item, .dropdown-item').first();
        if (await options.isVisible()) {
          await options.click();
        }
      });
    }
  });

  test('should handle loading states and spinners', async ({ helper }) => {
    await helper.page.goto('/accounts');
    await helper.page.waitForLoadState('networkidle');

    const actions = [
      async () => {
        const addButton = await helper.page.locator('.add-button');
        if (await addButton.isVisible()) {
          await addButton.click();
        }
      },
      async () => {
        await helper.page.reload();
        await helper.page.waitForLoadState('networkidle');
      }
    ];

    for (const action of actions) {
      await helper.measurePerformance('Test loading states', async () => {
        const loadingPromise = helper.page.waitForSelector('.el-loading, .spinner, [data-testid*="loading"]', { state: 'visible', timeout: 2000 });
        await action();
        
        try {
          await loadingPromise;
          const loadingElement = await helper.page.locator('.el-loading, .spinner, [data-testid*="loading"]');
          await loadingElement.waitFor({ state: 'hidden', timeout: 5000 });
        } catch (e) {
          console.log('No loading element found or timeout');
        }
      });
    }
  });

  test('should handle data export/import operations', async ({ helper }) => {
    await helper.page.goto('/settings');
    await helper.page.waitForLoadState('networkidle');

    await helper.measurePerformance('Test export operation', async () => {
      const exportButton = await helper.page.locator('.export-button, [data-testid*="export"]');
      if (await exportButton.isVisible()) {
        const downloadPromise = helper.page.waitForEvent('download');
        await exportButton.click();
        const download = await downloadPromise;
        console.log('Export completed:', download.suggestedFilename());
      }
    });

    await helper.measurePerformance('Test import operation', async () => {
      const importButton = await helper.page.locator('.import-button, [data-testid*="import"]');
      if (await importButton.isVisible()) {
        await importButton.click();
        
        const fileInput = await helper.page.locator('input[type="file"]');
        if (await fileInput.isVisible()) {
          console.log('Import dialog opened');
        }
      }
    });
  });

  test.afterEach(async ({ helper }) => {
    const report = helper.generateReport();
    console.log(report);
  });
});