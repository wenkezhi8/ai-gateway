import { test, expect } from '@playwright/test'

test('chat model selector shows provider display_name', async ({ page, request }) => {
  // Login via API to get token
  const loginRes = await request.post('/api/auth/login', {
    data: { username: 'admin', password: 'admin123' }
  })
  expect(loginRes.ok()).toBeTruthy()
  const loginJson = await loginRes.json()
  const token = loginJson.token as string
  expect(token).toBeTruthy()

  // Find the account that uses vpsairobot.com and sync models (ensures display_name persisted)
  const accountsRes = await request.get('/api/admin/accounts', {
    headers: { Authorization: `Bearer ${token}` }
  })
  expect(accountsRes.ok()).toBeTruthy()
  const accountsJson = await accountsRes.json()
  const accounts = (accountsJson.data || []) as Array<{ id: string; base_url?: string; provider?: string; enabled?: boolean }>
  const target = accounts.find(a => (a.base_url || '').includes('vpsairobot.com') && a.enabled)
  expect(target?.id).toBeTruthy()

  const syncRes = await request.get(`/api/admin/accounts/${encodeURIComponent(target!.id)}/fetch-models?sync=true`, {
    headers: { Authorization: `Bearer ${token}` }
  })
  expect(syncRes.ok()).toBeTruthy()

  // Ensure UI uses admin API (requires token in localStorage)
  await page.addInitScript((t) => {
    localStorage.setItem('token', t)
  }, token)

  await page.goto('/chat')

  // Create a new chat so model selector is visible
  await page.getByRole('button', { name: /New Chat/i }).first().click()
  await expect(page.locator('.model-selector')).toBeVisible()

  // Select provider OpenAI
  await page.locator('.provider-select .el-select__wrapper').click()
  await page.locator('.el-select-dropdown__item').filter({ hasText: 'OpenAI' }).first().click()

  // Open model dropdown and assert display name exists
  await page.locator('.model-select .el-select__wrapper').click()
  await expect(page.locator('.el-select-dropdown__item').filter({ hasText: 'GPT-5.3 Codex' }).first()).toBeVisible()
})
