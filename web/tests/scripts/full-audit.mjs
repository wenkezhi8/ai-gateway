import fs from 'node:fs/promises'
import path from 'node:path'
import { chromium } from 'playwright'

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8566'
const username = process.env.E2E_USERNAME || 'admin'
const password = process.env.E2E_PASSWORD || 'admin123'

const pages = [
  '/docs',
  '/dashboard',
  '/ops',
  '/routing',
  '/cache',
  '/alerts',
  '/api-management',
  '/model-management',
  '/providers-accounts',
  '/usage',
  '/chat',
  '/settings'
]

const dangerousKeywords = ['删除', '清空', '重置', '移除', '退出', '登出', '注销', 'disable']

function nowStamp() {
  const d = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`
}

function safeName(input) {
  return input.replace(/[^a-zA-Z0-9-_]/g, '_')
}

async function ensureDir(dir) {
  await fs.mkdir(dir, { recursive: true })
}

async function login(page) {
  await page.goto(`${baseURL}/login`, { waitUntil: 'domcontentloaded' })
  await page.locator('input[type="text"], input[name="username"], [placeholder*="用户名"], [placeholder*="账号"]').first().fill(username)
  await page.locator('input[type="password"], input[name="password"], [placeholder*="密码"]').first().fill(password)
  const loginBtn = page.locator('button:has-text("登录"), button:has-text("Login"), button[type="submit"]').first()
  await loginBtn.click({ timeout: 5000 })
  await page.waitForURL(/\/dashboard/, { timeout: 15000 })
}

async function clickSafeButtons(page, limit = 12) {
  const results = []
  const buttons = page.locator('button, .el-button, [role="button"]')
  const count = await buttons.count()
  for (let i = 0; i < count && results.length < limit; i++) {
    const btn = buttons.nth(i)
    const visible = await btn.isVisible().catch(() => false)
    if (!visible) continue
    const textRaw = (await btn.innerText().catch(() => '')) || ''
    const text = textRaw.replace(/\s+/g, ' ').trim()
    if (!text) continue
    if (dangerousKeywords.some((k) => text.toLowerCase().includes(k.toLowerCase()))) continue
    try {
      await btn.click({ timeout: 1500 })
      await page.waitForTimeout(250)
      results.push({ text, ok: true })
    } catch (e) {
      results.push({ text, ok: false, error: String(e?.message || e) })
    }
  }
  return results
}

async function addThenDeleteCacheWarmup(page) {
  const marker = `auto-audit-${Date.now()}`
  const result = { ok: false, marker, steps: [] }
  try {
    await page.goto(`${baseURL}/cache`, { waitUntil: 'domcontentloaded' })
    await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {})

    const token = await page.evaluate(() => localStorage.getItem('token') || '')
    if (!token) {
      throw new Error('missing auth token')
    }

    const addRes = await page.evaluate(async ({ marker, token }) => {
      const resp = await fetch('/api/admin/cache/test-entry', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify({
          task_type: 'math',
          user_message: marker,
          ai_response: `reply-${marker}`,
          model: 'audit-model',
          provider: 'audit-provider',
          ttl: 24
        })
      })
      const data = await resp.json().catch(() => ({}))
      return { ok: resp.ok, status: resp.status, data }
    }, { marker, token })
    if (!addRes.ok) {
      throw new Error(`add test entry failed: ${addRes.status}`)
    }
    result.steps.push('api_add_ok')

    const responseKey = addRes?.data?.data?.response_key
    const requestKey = addRes?.data?.data?.request_key
    const delRes = await page.evaluate(async ({ responseKey, requestKey, token }) => {
      const deleteOne = async (key) => {
        if (!key) return { ok: false, status: 0 }
        const resp = await fetch(`/api/admin/cache/entries/${encodeURIComponent(key)}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${token}` }
        })
        return { ok: resp.ok, status: resp.status }
      }
      const a = await deleteOne(responseKey)
      const b = await deleteOne(requestKey)
      return { response: a, request: b }
    }, { responseKey, requestKey, token })

    result.ok = !!(delRes?.response?.ok && delRes?.request?.ok)
    result.steps.push(`api_delete_resp_${delRes?.response?.status || 0}`)
    result.steps.push(`api_delete_req_${delRes?.request?.status || 0}`)
    return result
  } catch (e) {
    result.ok = false
    result.error = String(e?.message || e)
    return result
  }
}

async function run() {
  const stamp = nowStamp()
  const outputDir = path.join('/Users/openclaw/Desktop', `ai-gateway-full-audit-${stamp}`)
  const shotsDir = path.join(outputDir, 'screenshots')
  await ensureDir(shotsDir)

  const browser = await chromium.launch({ headless: true })
  const context = await browser.newContext()
  const page = await context.newPage()

  const apiIssues = []
  page.on('response', (resp) => {
    const url = resp.url()
    if (!url.includes('/api/')) return
    if (resp.status() >= 400) {
      apiIssues.push({ type: 'http', status: resp.status(), url })
    }
  })
  page.on('requestfailed', (req) => {
    const url = req.url()
    if (!url.includes('/api/')) return
    apiIssues.push({ type: 'failed', status: 0, url, error: req.failure()?.errorText || 'request_failed' })
  })

  const pageResults = []
  try {
    await login(page)

    for (const route of pages) {
      const result = { route, ok: true, clicks: [], error: null }
      try {
        await page.goto(`${baseURL}${route}`, { waitUntil: 'domcontentloaded' })
        await page.waitForLoadState('networkidle', { timeout: 10000 }).catch(() => {})
        result.clicks = await clickSafeButtons(page)
      } catch (e) {
        result.ok = false
        result.error = String(e?.message || e)
      }
      const shot = path.join(shotsDir, `${safeName(route || 'root')}.png`)
      await page.screenshot({ path: shot, fullPage: true }).catch(() => {})
      pageResults.push(result)
    }

    const mutation = await addThenDeleteCacheWarmup(page)
    const mutationShot = path.join(shotsDir, 'cache-add-delete.png')
    await page.screenshot({ path: mutationShot, fullPage: true }).catch(() => {})

    const report = {
      generated_at: new Date().toISOString(),
      base_url: baseURL,
      pages: pageResults,
      api_issues: apiIssues,
      mutation_test: mutation,
      summary: {
        total_pages: pageResults.length,
        page_failures: pageResults.filter((p) => !p.ok).length,
        total_click_actions: pageResults.reduce((s, p) => s + p.clicks.length, 0),
        click_failures: pageResults.reduce((s, p) => s + p.clicks.filter((c) => !c.ok).length, 0),
        api_issues: apiIssues.length,
        mutation_ok: mutation.ok
      }
    }

    await fs.writeFile(path.join(outputDir, 'report.json'), JSON.stringify(report, null, 2), 'utf-8')

    const lines = []
    lines.push('# AI Gateway 全站自动巡检报告')
    lines.push(`- 生成时间: ${report.generated_at}`)
    lines.push(`- 基础地址: ${baseURL}`)
    lines.push(`- 页面数: ${report.summary.total_pages}`)
    lines.push(`- 页面失败: ${report.summary.page_failures}`)
    lines.push(`- 按钮点击数: ${report.summary.total_click_actions}`)
    lines.push(`- 点击失败: ${report.summary.click_failures}`)
    lines.push(`- API 异常: ${report.summary.api_issues}`)
    lines.push(`- 新增后删除自建数据: ${report.summary.mutation_ok ? '通过' : '失败'}`)
    lines.push('')
    lines.push('## 页面结果')
    for (const p of report.pages) {
      lines.push(`- ${p.route}: ${p.ok ? '通过' : `失败 (${p.error || 'unknown'})`}，点击 ${p.clicks.length} 次，失败 ${p.clicks.filter((c) => !c.ok).length} 次`)
    }
    lines.push('')
    lines.push('## API 异常（前 30 条）')
    for (const i of report.api_issues.slice(0, 30)) {
      lines.push(`- [${i.type}] ${i.status} ${i.url}${i.error ? ` (${i.error})` : ''}`)
    }
    if (report.api_issues.length === 0) lines.push('- 无')
    lines.push('')
    lines.push('## 缓存增删自检')
    lines.push(`- 标记: ${report.mutation_test.marker}`)
    lines.push(`- 结果: ${report.mutation_test.ok ? '通过' : '失败'}`)
    lines.push(`- 步骤: ${(report.mutation_test.steps || []).join(' -> ')}`)
    if (report.mutation_test.error) lines.push(`- 错误: ${report.mutation_test.error}`)

    await fs.writeFile(path.join(outputDir, 'report.md'), lines.join('\n'), 'utf-8')

    console.log(`REPORT_DIR=${outputDir}`)
  } finally {
    await context.close()
    await browser.close()
  }
}

run().catch((e) => {
  console.error(e)
  process.exit(1)
})
