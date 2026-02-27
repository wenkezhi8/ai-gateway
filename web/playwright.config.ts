import { defineConfig, devices } from '@playwright/test';

// 改动点: 默认使用已运行服务，避免与 8566 端口冲突
const useExistingServer = process.env.E2E_USE_EXISTING_SERVER !== '0';
// 改动点: 允许通过环境变量覆盖 baseURL
const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8566';
const reporters = process.env.CI
  ? [
      ['html', { outputFolder: 'tests/results/html-report' }],
      ['json', { outputFile: 'tests/results/test-results.json' }],
      ['list'],
      ['./tests/utils/custom-reporter.ts']
    ]
  : [
      ['list'],
      ['./tests/utils/custom-reporter.ts']
    ];

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: reporters,
  // 改动点: 避免 outputDir 与 HTML 报告目录冲突
  outputDir: 'tests/results/artifacts',
  timeout: 30000,
  expect: {
    timeout: 5000
  },
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    navigationTimeout: 15000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  // 改动点: 可选跳过 webServer（使用已启动的服务）
  webServer: useExistingServer ? undefined : {
    command: 'npm run dev -- --host 127.0.0.1 --port 8566',
    url: baseURL,
    reuseExistingServer: !process.env.CI,
    timeout: 120 * 1000,
  },
});
