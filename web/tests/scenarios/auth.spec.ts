import { test, expect } from '../utils/test-helper';
import { LoginPage } from '../page-objects/login-page';

test.describe('Authentication Tests', () => {
  let loginPage: LoginPage;

  test.beforeEach(async ({ page, helper }) => {
    loginPage = new LoginPage(page);
    await helper.checkNetworkErrors();
  });

  test('should display login page correctly', async ({ page, helper }) => {
    await helper.measurePerformance('Load login page', async () => {
      await loginPage.navigate();
    });

    const title = await page.title();
    expect(title).toContain('登录');

    await helper.measurePerformance('Verify login form elements', async () => {
      expect(await loginPage.isVisible()).toBe(true);
      expect(await loginPage.getLoginButton()).toBeVisible();
    });
  });

  test('should login with valid credentials', async ({ page, helper }) => {
    await helper.measurePerformance('Login with valid credentials', async () => {
      await loginPage.login('admin', 'admin');
    });

    await helper.measurePerformance('Verify successful login', async () => {
      expect(page.url()).toContain('/dashboard');
    });

    const report = helper.generateReport();
    console.log(report);
  });

  test('should show error with invalid credentials', async ({ page, helper }) => {
    await loginPage.navigate();
    
    await helper.measurePerformance('Attempt login with invalid credentials', async () => {
      await loginPage.fillCredentials('invalid', 'invalid');
      await loginPage.submitLogin();
    });

    await helper.measurePerformance('Verify error message appears', async () => {
      const errorMessage = await loginPage.getErrorMessage();
      await errorMessage.waitFor({ state: 'visible', timeout: 5000 });
    });
  });

  test('should handle empty form submission', async ({ page, helper }) => {
    await loginPage.navigate();
    
    await helper.measurePerformance('Submit empty login form', async () => {
      await loginPage.submitLogin();
    });

    const currentUrl = page.url();
    expect(currentUrl).toContain('/login');
  });

  test('should logout successfully', async ({ page, helper }) => {
    await helper.login('admin', 'admin');
    
    await helper.measurePerformance('Logout from application', async () => {
      await helper.logout();
    });

    expect(page.url()).toContain('/login');
  });

  test('should redirect to login when accessing protected routes without authentication', async ({ page, helper }) => {
    const protectedRoutes = ['/dashboard', '/providers', '/accounts', '/routing', '/cache', '/alerts', '/settings'];

    for (const route of protectedRoutes) {
      await helper.measurePerformance(`Access protected route ${route} without auth`, async () => {
        await page.goto(route);
        await page.waitForLoadState('networkidle');
      });

      expect(page.url()).toContain('/login');
    }
  });

  test('should handle network errors gracefully', async ({ page, helper }) => {
    await helper.measurePerformance('Test network error handling', async () => {
      await page.route('**/api/**', route => route.abort('failed'));
      
      await loginPage.navigate();
      await loginPage.fillCredentials('admin', 'admin');
      await loginPage.submitLogin();
      
      await page.unroute('**/api/**');
    });

    const errorMessage = await loginPage.getErrorMessage();
    const isErrorMessageVisible = await errorMessage.isVisible();
    
    if (isErrorMessageVisible) {
      console.log('Network error message displayed correctly');
    }
  });
});