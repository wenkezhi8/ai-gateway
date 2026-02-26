import { test, expect } from '../utils/test-helper';

test.describe('Routing Classifier Tests', () => {
  test.beforeEach(async ({ helper }) => {
    await helper.login('admin', 'admin123');
  });

  test('should refresh classifier model list and switch model', async ({ helper }) => {
    const page = helper.page;
    let switchedModel = '';

    await page.route('**/api/admin/router/classifier/models', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            active_model: 'qwen2.5:0.5b-instruct',
            models: ['qwen2.5:0.5b-instruct', 'qwen2.5:1.5b-instruct', 'llama3.2:3b']
          }
        })
      });
    });

    await page.route('**/api/admin/router/classifier/switch', async route => {
      const payload = route.request().postDataJSON() as { model?: string };
      switchedModel = payload?.model || '';
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          message: 'Classifier model switched',
          data: { healthy: true, latency_ms: 30, message: 'ok' }
        })
      });
    });

    await page.goto('/routing');
    await page.waitForLoadState('networkidle');

    const refreshButton = page.getByRole('button', { name: '刷新模型列表' }).first();
    await expect(refreshButton).toBeVisible();
    await refreshButton.click();

    const modelSelect = page.locator('.el-form-item').filter({ hasText: '手动切换模型' }).locator('.el-select').first();
    await modelSelect.click();

    const targetOption = page.locator('.el-select-dropdown__item').filter({ hasText: 'llama3.2:3b' }).first();
    await expect(targetOption).toBeVisible();
    await targetOption.click();

    const switchButton = page.getByRole('button', { name: '切换模型' }).first();
    await switchButton.click();

    await expect.poll(() => switchedModel).toBe('llama3.2:3b');
    await expect(page.locator('.el-form-item').filter({ hasText: '运行模型' }).first()).toContainText('llama3.2:3b');
  });

  test('should save control toggles and display control stats', async ({ helper }) => {
    const page = helper.page;
    let savedControl: Record<string, boolean> = {};

    await page.route('**/api/admin/router/config', async route => {
      if (route.request().method() === 'GET') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            data: {
              use_auto_mode: 'auto',
              default_strategy: 'auto',
              default_model: 'deepseek-chat',
              classifier: {
                enabled: true,
                shadow_mode: false,
                provider: 'ollama',
                base_url: 'http://127.0.0.1:11434',
                active_model: 'qwen2.5:0.5b-instruct',
                candidate_models: ['qwen2.5:0.5b-instruct'],
                timeout_ms: 120,
                confidence_threshold: 0.8,
                fail_open: true,
                max_input_chars: 4000,
                control: {
                  enable: false,
                  shadow_only: true,
                  normalized_query_read_enable: false,
                  cache_write_gate_enable: false,
                  risk_tag_enable: false,
                  tool_gate_enable: false,
                  model_fit_enable: false
                }
              },
              strategies: [
                { value: 'auto', label: '智能平衡', description: '综合效果 + 速度 + 成本' }
              ]
            }
          })
        });
        return;
      }

      const payload = route.request().postDataJSON() as { classifier?: { control?: Record<string, boolean> } };
      savedControl = payload?.classifier?.control || {};
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, message: 'ok' })
      });
    });

    await page.route('**/api/admin/router/classifier/stats', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            total_requests: 18,
            llm_attempts: 16,
            llm_success: 15,
            fallbacks: 3,
            shadow_requests: 2,
            avg_llm_latency_ms: 36.4,
            avg_control_latency_ms: 12.2,
            parse_errors: 1,
            control_fields_missing: 4
          }
        })
      });
    });

    await page.route('**/api/admin/router/classifier/health', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            healthy: true,
            latency_ms: 20,
            message: 'ok'
          }
        })
      });
    });

    await page.route('**/api/admin/router/classifier/models', async route => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          success: true,
          data: {
            active_model: 'qwen2.5:0.5b-instruct',
            models: ['qwen2.5:0.5b-instruct']
          }
        })
      });
    });

    await page.goto('/routing');
    await page.waitForLoadState('networkidle');

    await expect(page.getByText('控制层延迟')).toBeVisible();
    await expect(page.getByText('12.2ms')).toBeVisible();
    await expect(page.getByText('解析错误')).toBeVisible();
    await expect(page.getByText('控制字段缺失')).toBeVisible();

    await page.locator('.el-form-item').filter({ hasText: '控制层总开关' }).locator('.el-switch').first().click();
    await page.locator('.el-form-item').filter({ hasText: '风险打标' }).locator('.el-switch').first().click();
    await page.locator('.el-form-item').filter({ hasText: '缓存写门禁' }).locator('.el-switch').first().click();

    await page.getByRole('button', { name: '保存配置' }).first().click();

    await expect.poll(() => savedControl.enable).toBe(true);
    await expect.poll(() => savedControl.risk_tag_enable).toBe(true);
    await expect.poll(() => savedControl.cache_write_gate_enable).toBe(true);
  });
});
