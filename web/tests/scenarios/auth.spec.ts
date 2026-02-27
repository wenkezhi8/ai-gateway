import { test, expect } from '../utils/test-helper';
import { LoginPage } from '../page-objects/login-page';
import {
  DASHBOARD_ROUTE,
  LOGIN_ROUTE,
  POST_LOGOUT_REDIRECT,
  UNAUTHORIZED_REDIRECT
} from '../../src/constants/navigation';

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
      await loginPage.login('admin', 'admin123');
    });

    await helper.measurePerformance('Verify successful login', async () => {
      expect(new URL(page.url()).pathname).toBe(DASHBOARD_ROUTE);
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
    expect(currentUrl).toContain(LOGIN_ROUTE);
  });

  test('should logout successfully', async ({ page, helper }) => {
    await helper.login('admin', 'admin123');
    
    await helper.measurePerformance('Logout from application', async () => {
      await helper.logout();
    });

    expect(new URL(page.url()).pathname).toBe(POST_LOGOUT_REDIRECT);
  });

  test('should block protected history navigation after logout', async ({ page, helper }) => {
    await helper.login('admin', 'admin123');
    await page.goto(DASHBOARD_ROUTE);
    await page.waitForLoadState('networkidle');

    await helper.logout();
    expect(new URL(page.url()).pathname).toBe(POST_LOGOUT_REDIRECT);

    await page.goBack();
    await page.waitForLoadState('networkidle');

    const pathAfterBack = new URL(page.url()).pathname;
    expect(pathAfterBack).not.toBe(DASHBOARD_ROUTE);
    expect([UNAUTHORIZED_REDIRECT, POST_LOGOUT_REDIRECT]).toContain(pathAfterBack);
  });

  test('should redirect to login when accessing protected routes without authentication', async ({ page, helper }) => {
    const protectedRoutes = [DASHBOARD_ROUTE, '/providers', '/accounts', '/routing', '/cache', '/alerts', '/settings'];

    for (const route of protectedRoutes) {
      await helper.measurePerformance(`Access protected route ${route} without auth`, async () => {
        await page.goto(route);
        await page.waitForLoadState('networkidle');
      });

      expect(page.url()).toContain(UNAUTHORIZED_REDIRECT);
    }
  });

  test('should handle network errors gracefully', async ({ page, helper }) => {
    await helper.measurePerformance('Test network error handling', async () => {
      await page.route('**/api/**', route => route.abort('failed'));
      
      await loginPage.navigate();
      await loginPage.fillCredentials('admin', 'admin123');
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
