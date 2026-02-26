import fs from 'node:fs/promises'
import path from 'node:path'
import { chromium } from 'playwright'

const baseURL = process.env.E2E_BASE_URL || 'http://127.0.0.1:8566'
const username = process.env.E2E_USERNAME || 'admin'
const password = process.env.E2E_PASSWORD || 'admin123'
const ACTION_DELAY_MS = Number(process.env.E2E_ACTION_DELAY_MS || 600)
const CLICK_TIMEOUT_MS = Number(process.env.E2E_CLICK_TIMEOUT_MS || 5000)
const NETWORKIDLE_TIMEOUT_MS = Number(process.env.E2E_NETWORKIDLE_TIMEOUT_MS || 15000)
const MUTATION_REPEAT = Number(process.env.E2E_MUTATION_REPEAT || 50)

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

const dangerousKeywords = ['删除', '清空', '重置', '移除', '退出', '登出', '注销', 'disable', 'execute', '执行']

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

async function closeBlockingOverlays(page, maxRounds = 3) {
  let closed = 0
  for (let i = 0; i < maxRounds; i++) {
    const hasOverlay = await page.locator('.el-overlay:visible .el-dialog, .el-overlay:visible .el-drawer, .el-message-box:visible').count()
    if (hasOverlay === 0) break

    const closeSelectors = [
      '.el-overlay:visible .el-dialog__headerbtn:visible',
      '.el-overlay:visible .el-drawer__close-btn:visible',
      '.el-message-box__headerbtn:visible',
      '.el-overlay:visible button:has-text("取消"):visible',
      '.el-overlay:visible button:has-text("关闭"):visible',
      '.el-message-box button:has-text("取消"):visible'
    ]

    let clicked = false
    for (const selector of closeSelectors) {
      const btn = page.locator(selector).last()
      if (await btn.isVisible().catch(() => false)) {
        await btn.click({ timeout: CLICK_TIMEOUT_MS }).catch(() => {})
        clicked = true
        closed++
        break
      }
    }

    if (!clicked) {
      await page.keyboard.press('Escape').catch(() => {})
      closed++
    }

    await page.waitForTimeout(ACTION_DELAY_MS)
  }
  return closed
}

async function collectZeroPanelValues(page, route) {
  const items = await page.evaluate(({ route }) => {
    const zeroRegex = /^(0(?:\.0+)?%?|0\s*B|0\s*ms|0\s*条|0)$/i
    const containers = Array.from(document.querySelectorAll('.stat-card, .summary-card, .metric-card, .type-card, .panel, .el-card, .card'))
    const observations = []

    const getText = (el) => (el?.textContent || '').replace(/\s+/g, ' ').trim()

    for (const container of containers) {
      const visible = container instanceof HTMLElement ? container.offsetParent !== null : true
      if (!visible) continue

      const labelFallback = getText(container.querySelector('.stat-label, .card-label, .metric-label, .type-name, .panel-title, .section-title, .title, .label'))
      const valueNodes = container.querySelectorAll('.stat-value, .card-value, .metric-value, .value, .number, .progress-meta span:last-child, .type-meta span')

      for (const node of valueNodes) {
        const value = getText(node)
        if (!value || !zeroRegex.test(value)) continue

        let label = labelFallback
        if (!label) {
          const row = node.closest('.progress-meta, .type-meta, .stat-body, .item, .row')
          label = getText(row?.querySelector('.label, .name, .title, span:first-child'))
        }
        if (!label) {
          label = getText(container.querySelector('h1, h2, h3, h4, .title')) || '未命名面板'
        }

        observations.push({
          route,
          label,
          value
        })
      }
    }

    const uniq = []
    const seen = new Set()
    for (const item of observations) {
      const key = `${item.route}|${item.label}|${item.value}`
      if (seen.has(key)) continue
      seen.add(key)
      uniq.push(item)
    }
    return uniq
  }, { route })

  return items
}

function buildBugsMarkdown(report) {
  const topFailedPages = [...report.pages]
    .map((p) => ({ route: p.route, failCount: p.clicks.filter((c) => !c.ok).length }))
    .filter((p) => p.failCount > 0)
    .sort((a, b) => b.failCount - a.failCount)

  const abortedCount = report.summary.api_canceled || 0

  const lines = []
  lines.push('# AI Gateway Bug 清单（自动巡检）')
  lines.push('')
  lines.push(`生成时间：${new Date(report.generated_at).toLocaleString('zh-CN')}`)
  lines.push('来源：`report.md` / `report.json`')
  lines.push('')
  lines.push('## 结论摘要')
  lines.push('')
  lines.push(`- 页面可达：${report.summary.total_pages - report.summary.page_failures}/${report.summary.total_pages}`)
  lines.push(`- 按钮点击异常：${report.summary.click_failures} 次`)
  lines.push(`- API 异常：${report.summary.api_errors} 条`)
  lines.push(`- API 取消：${abortedCount} 条（ERR_ABORTED，不计为错误）`)
  lines.push(`- 数据联动自检（新增后删除自建缓存）：${report.summary.mutation_ok ? '通过' : '失败'}`)
  lines.push(`- 缓存统计一致性问题：${report.summary.cache_consistency_issues || 0} 条`)
  lines.push(`- 缓存UI绑定问题：${report.summary.cache_ui_binding_issues || 0} 条`)
  lines.push(`- 缓存面板零值位置：${(report.cache_ui_binding?.ui?.zeroPositions || []).length} 处（待人工判定是否合理）`)
  lines.push(`- 全站面板零值记录：${(report.zero_panel_observations || []).length} 处`)
  lines.push('')
  lines.push('## 高优先级问题')
  lines.push('')

  if (topFailedPages.length > 0) {
    const top3 = topFailedPages.slice(0, 3)
    lines.push('1. 弹窗/状态干扰导致点击失败（P1）')
    lines.push(`   - 现象：失败主要集中在 ${top3.map((p) => `\`${p.route}\`(${p.failCount})`).join('、')}。`)
    lines.push('   - 判断：多为弹窗覆盖导致的 pointer-intercept，建议增加弹窗生命周期管理。')
    lines.push('')
  }

  if (report.summary.api_errors > 0) {
    lines.push('2. API 错误噪声需要分级（P2）')
    lines.push('   - 现象：巡检过程中存在 request failed。')
    lines.push('   - 判断：包含真实 HTTP 错误，需排查接口状态码与返回体。')
    lines.push('')
  }

  if (topFailedPages.length === 0 && report.summary.api_errors === 0 && report.summary.mutation_ok) {
    lines.push('1. 暂无高优先级问题（P2）')
    lines.push('   - 说明：本轮自动巡检未发现可稳定复现的业务级错误。')
    lines.push('')
  }

  if (!report.summary.mutation_ok) {
    lines.push('3. 自建数据回滚失败（P1）')
    lines.push('   - 现象：新增后删除自检未通过。')
    lines.push('   - 建议：优先修复该链路，避免巡检污染数据。')
    lines.push('')
  }

  if ((report.summary.cache_consistency_issues || 0) > 0) {
    lines.push('4. 缓存统计不一致（P1）')
    for (const issue of report.cache_consistency?.issues || []) {
      lines.push(`   - ${issue}`)
    }
    lines.push('')
  }

  if ((report.summary.cache_ui_binding_issues || 0) > 0) {
    lines.push('5. 缓存UI与后端数据不一致（P1）')
    for (const issue of report.cache_ui_binding?.issues || []) {
      lines.push(`   - ${issue}`)
    }
    lines.push('')
  }

  if ((report.cache_ui_binding?.ui?.zeroPositions || []).length > 0) {
    lines.push('6. 缓存面板零值观察项（待确认）')
    for (const item of report.cache_ui_binding.ui.zeroPositions) {
      lines.push(`   - [${item.section}] ${item.label}: ${item.value}`)
    }
    lines.push('')
  }

  if ((report.zero_panel_observations || []).length > 0) {
    lines.push('7. 全站面板零值记录（待逐项验证）')
    for (const item of report.zero_panel_observations) {
      lines.push(`   - [${item.route}] ${item.label}: ${item.value}`)
    }
    lines.push('')
  }

  lines.push('## 通过项')
  lines.push('')
  lines.push(`- 主导航页面可达：${report.summary.page_failures === 0 ? '通过' : '部分失败'}`)
  lines.push(`- 缓存新增并删除自建数据：${report.summary.mutation_ok ? '通过' : '失败'}`)
  lines.push('')
  lines.push('## 附件')
  lines.push('')
  lines.push('- 总报告：`report.md`')
  lines.push('- 机器可读：`report.json`')
  lines.push('- 截图目录：`screenshots/`')

  return lines.join('\n')
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
  await closeBlockingOverlays(page)

  const buttons = page.locator('button, .el-button, [role="button"]')
  const count = await buttons.count()
  for (let i = 0; i < count && results.length < limit; i++) {
    await closeBlockingOverlays(page, 2)

    const btn = buttons.nth(i)
    const visible = await btn.isVisible().catch(() => false)
    if (!visible) continue
    const textRaw = (await btn.innerText().catch(() => '')) || ''
    const text = textRaw.replace(/\s+/g, ' ').trim()
    if (!text) continue
    if (dangerousKeywords.some((k) => text.toLowerCase().includes(k.toLowerCase()))) continue
    try {
      await btn.scrollIntoViewIfNeeded().catch(() => {})
      await page.waitForTimeout(ACTION_DELAY_MS)
      await btn.click({ timeout: CLICK_TIMEOUT_MS })
      await page.waitForTimeout(ACTION_DELAY_MS)
      await closeBlockingOverlays(page, 2)
      results.push({ text, ok: true })
    } catch (e) {
      results.push({ text, ok: false, error: String(e?.message || e) })
    }
  }
  return results
}

async function runMutationScenarios(page, repeat = MUTATION_REPEAT) {
  const token = await page.evaluate(() => localStorage.getItem('token') || '')
  if (!token) {
    return {
      ok: false,
      repeat,
      scenarios: [],
      error: 'missing auth token'
    }
  }

  const scenarios = []

  const cacheScenario = await page.evaluate(async ({ token, repeat }) => {
    const result = { feature: 'cache', repeat, create_ok: 0, delete_ok: 0, failed: [] }
    const createdKeys = []
    for (let i = 0; i < repeat; i++) {
      const marker = `auto-audit-cache-${Date.now()}-${i}`
      try {
        const resp = await fetch('/api/admin/cache/test-entry', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
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
        if (!resp.ok) {
          result.failed.push({ stage: 'create', index: i, status: resp.status })
          continue
        }
        result.create_ok++
        createdKeys.push({ response_key: data?.data?.response_key, request_key: data?.data?.request_key, index: i })
      } catch (e) {
        result.failed.push({ stage: 'create', index: i, error: String(e?.message || e) })
      }
    }

    for (const item of createdKeys) {
      try {
        const a = await fetch(`/api/admin/cache/entries/${encodeURIComponent(item.response_key || '')}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${token}` }
        })
        const b = await fetch(`/api/admin/cache/entries/${encodeURIComponent(item.request_key || '')}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${token}` }
        })
        if (a.ok && b.ok) {
          result.delete_ok++
        } else {
          result.failed.push({ stage: 'delete', index: item.index, status: `${a.status}/${b.status}` })
        }
      } catch (e) {
        result.failed.push({ stage: 'delete', index: item.index, error: String(e?.message || e) })
      }
    }

    return result
  }, { token, repeat })
  scenarios.push(cacheScenario)

  const alertsScenario = await page.evaluate(async ({ token, repeat }) => {
    const result = { feature: 'alerts', repeat, create_ok: 0, delete_ok: 0, failed: [] }
    const createdIds = []
    for (let i = 0; i < repeat; i++) {
      const marker = `auto-audit-alert-${Date.now()}-${i}`
      try {
        const resp = await fetch('/api/admin/alerts/rules', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
          body: JSON.stringify({
            name: marker,
            enabled: true,
            condition: {
              type: 'latency',
              operator: '>',
              threshold: 100,
              duration: 60
            },
            notifyChannels: ['email']
          })
        })
        const data = await resp.json().catch(() => ({}))
        if (!resp.ok) {
          result.failed.push({ stage: 'create', index: i, status: resp.status })
          continue
        }
        result.create_ok++
        createdIds.push({ id: data?.data?.id, index: i })
      } catch (e) {
        result.failed.push({ stage: 'create', index: i, error: String(e?.message || e) })
      }
    }

    for (const item of createdIds) {
      try {
        const resp = await fetch(`/api/admin/alerts/rules/${encodeURIComponent(item.id || '')}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${token}` }
        })
        if (resp.ok || resp.status === 404) {
          result.delete_ok++
        } else {
          result.failed.push({ stage: 'delete', index: item.index, status: resp.status })
        }
      } catch (e) {
        result.failed.push({ stage: 'delete', index: item.index, error: String(e?.message || e) })
      }
    }

    return result
  }, { token, repeat })
  scenarios.push(alertsScenario)

  const apiKeysScenario = await page.evaluate(async ({ token, repeat }) => {
    const result = { feature: 'api-keys', repeat, create_ok: 0, delete_ok: 0, failed: [] }
    const createdIds = []
    for (let i = 0; i < repeat; i++) {
      const marker = `auto-audit-key-${Date.now()}-${i}`
      try {
        const resp = await fetch('/api/admin/api-keys', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
          body: JSON.stringify({
            name: marker,
            description: 'auto audit temp key'
          })
        })
        const data = await resp.json().catch(() => ({}))
        if (!resp.ok) {
          result.failed.push({ stage: 'create', index: i, status: resp.status })
          continue
        }
        result.create_ok++
        createdIds.push({ id: data?.data?.id, index: i })
      } catch (e) {
        result.failed.push({ stage: 'create', index: i, error: String(e?.message || e) })
      }
    }

    for (const item of createdIds) {
      try {
        const resp = await fetch(`/api/admin/api-keys/${encodeURIComponent(item.id || '')}`, {
          method: 'DELETE',
          headers: { Authorization: `Bearer ${token}` }
        })
        if (resp.ok || resp.status === 404) {
          result.delete_ok++
        } else {
          result.failed.push({ stage: 'delete', index: item.index, status: resp.status })
        }
      } catch (e) {
        result.failed.push({ stage: 'delete', index: item.index, error: String(e?.message || e) })
      }
    }

    return result
  }, { token, repeat })
  scenarios.push(apiKeysScenario)

  const ok = scenarios.every((s) => s.create_ok === repeat && s.delete_ok === repeat)
  return {
    ok,
    repeat,
    scenarios
  }
}

async function checkCacheStatsConsistency(page) {
  const token = await page.evaluate(() => localStorage.getItem('token') || '')
  if (!token) return { ok: false, issues: ['missing auth token'], stats: null }

  const result = await page.evaluate(async ({ token }) => {
    const issues = []
    const resp = await fetch('/api/admin/cache/stats', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const payload = await resp.json().catch(() => ({}))
    if (!resp.ok) {
      return { ok: false, issues: [`cache stats http ${resp.status}`], stats: null }
    }
    const stats = payload?.data || payload || {}
    const keys = ['request_cache', 'context_cache', 'route_cache', 'usage_cache', 'response_cache']

    let totalEntries = 0
    let totalSize = 0
    for (const key of keys) {
      const s = stats[key]
      if (!s) continue
      const entries = Number(s.entries || 0)
      const size = Number(s.size_bytes || 0)
      const hits = Number(s.hits || 0)
      const misses = Number(s.misses || 0)
      const hitRate = Number(s.hit_rate || 0)

      totalEntries += entries
      totalSize += size

      if (entries > 0 && size === 0) {
        issues.push(`${key}: entries=${entries} but size_bytes=0`)
      }
      if (hitRate > 0 && hits + misses === 0) {
        issues.push(`${key}: hit_rate=${hitRate} but hits/misses are 0`)
      }
    }

    if (totalEntries > 0 && totalSize === 0) {
      issues.push(`overall: total_entries=${totalEntries} but total_size_bytes=0`)
    }

    return { ok: issues.length === 0, issues, stats }
  }, { token })

  return result
}

async function checkCacheUiBinding(page) {
  const token = await page.evaluate(() => localStorage.getItem('token') || '')
  if (!token) return { ok: false, issues: ['missing auth token'], ui: null, api: null }

  await page.goto(`${baseURL}/cache`, { waitUntil: 'domcontentloaded' })
  await page.waitForLoadState('networkidle', { timeout: NETWORKIDLE_TIMEOUT_MS }).catch(() => {})
  await page.waitForTimeout(ACTION_DELAY_MS)

  const uiSnapshot = await page.evaluate(() => {
    const cards = Array.from(document.querySelectorAll('.stats-grid .stat-card'))
    const map = {}
    const zeroPositions = []
    for (const card of cards) {
      const label = card.querySelector('.stat-label')?.textContent?.trim() || ''
      const value = card.querySelector('.stat-value')?.textContent?.trim() || ''
      if (label) map[label] = value
      if (label && /^0([\s\.]|$)|^0\s*B$|^0%$|^0ms$/i.test(value)) {
        zeroPositions.push({ section: 'summary', label, value })
      }
    }

    const typeCards = Array.from(document.querySelectorAll('.type-list .type-card'))
    for (const card of typeCards) {
      const name = card.querySelector('.type-name')?.textContent?.trim() || '未知类型'
      const hitRate = card.querySelector('.progress-meta span:last-child')?.textContent?.trim() || ''
      const entriesText = card.querySelector('.type-meta span:first-child')?.textContent?.trim() || ''
      const sizeText = card.querySelector('.type-meta span:last-child')?.textContent?.trim() || ''
      if (/^0%$/i.test(hitRate)) {
        zeroPositions.push({ section: 'type', label: `${name}-命中率`, value: hitRate })
      }
      if (/\b0\s*条\b/.test(entriesText)) {
        zeroPositions.push({ section: 'type', label: `${name}-条目`, value: entriesText })
      }
      if (/\b0\s*B\b/i.test(sizeText)) {
        zeroPositions.push({ section: 'type', label: `${name}-体积`, value: sizeText })
      }
    }

    return {
      overallHitRate: map['整体命中率'] || '',
      totalEntries: map['缓存条目'] || '',
      totalSize: map['缓存体积'] || '',
      zeroPositions
    }
  })

  const apiSnapshot = await page.evaluate(async ({ token }) => {
    const resp = await fetch('/api/admin/cache/stats', {
      headers: { Authorization: `Bearer ${token}` }
    })
    const payload = await resp.json().catch(() => ({}))
    const stats = payload?.data || payload || {}
    const keys = ['request_cache', 'context_cache', 'route_cache', 'usage_cache', 'response_cache']
    let totalEntries = 0
    let totalSize = 0
    let totalHits = 0
    let totalOps = 0
    for (const key of keys) {
      const s = stats[key] || {}
      totalEntries += Number(s.entries || 0)
      totalSize += Number(s.size_bytes || 0)
      totalHits += Number(s.hits || 0)
      totalOps += Number(s.hits || 0) + Number(s.misses || 0)
    }
    const overallHitRate = totalOps > 0 ? `${Math.round((totalHits / totalOps) * 100)}%` : '0%'
    return { totalEntries, totalSize, overallHitRate }
  }, { token })

  const issues = []
  const uiEntriesNum = Number((uiSnapshot.totalEntries || '0').replace(/[^\d]/g, '') || 0)
  const uiSizeIsZero = /^0\s*B$/i.test(uiSnapshot.totalSize || '')

  if (apiSnapshot.totalEntries > 0 && uiEntriesNum === 0) {
    issues.push(`UI缓存条目显示为0，但API为${apiSnapshot.totalEntries}`)
  }
  if (apiSnapshot.totalSize > 0 && uiSizeIsZero) {
    issues.push(`UI缓存体积显示为0 B，但API size_bytes=${apiSnapshot.totalSize}`)
  }
  if (uiSnapshot.overallHitRate && apiSnapshot.overallHitRate && uiSnapshot.overallHitRate !== apiSnapshot.overallHitRate) {
    issues.push(`UI整体命中率=${uiSnapshot.overallHitRate} 与API计算=${apiSnapshot.overallHitRate} 不一致`)
  }

  return {
    ok: issues.length === 0,
    issues,
    ui: uiSnapshot,
    api: apiSnapshot
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
  page.setDefaultTimeout(CLICK_TIMEOUT_MS)
  page.setDefaultNavigationTimeout(NETWORKIDLE_TIMEOUT_MS)

  const apiIssues = []
  const apiCanceled = []
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
    const errorText = req.failure()?.errorText || 'request_failed'
    if (errorText.includes('ERR_ABORTED')) {
      apiCanceled.push({ type: 'canceled', status: 0, url, error: errorText })
      return
    }
    apiIssues.push({ type: 'failed', status: 0, url, error: errorText })
  })

  const pageResults = []
  const zeroPanelObservations = []
  try {
    await login(page)

    for (const route of pages) {
      const result = { route, ok: true, clicks: [], error: null }
      try {
        await page.goto(`${baseURL}${route}`, { waitUntil: 'domcontentloaded' })
        await page.waitForLoadState('networkidle', { timeout: NETWORKIDLE_TIMEOUT_MS }).catch(() => {})
        await page.waitForTimeout(ACTION_DELAY_MS)
        result.clicks = await clickSafeButtons(page)
      } catch (e) {
        result.ok = false
        result.error = String(e?.message || e)
      }
      const shot = path.join(shotsDir, `${safeName(route || 'root')}.png`)
      await page.screenshot({ path: shot, fullPage: true }).catch(() => {})
      const routeZeroPanels = await collectZeroPanelValues(page, route)
      zeroPanelObservations.push(...routeZeroPanels)
      pageResults.push(result)
    }

    const mutation = await runMutationScenarios(page, MUTATION_REPEAT)
    const cacheConsistency = await checkCacheStatsConsistency(page)
    const cacheUiBinding = await checkCacheUiBinding(page)
    const mutationShot = path.join(shotsDir, 'cache-add-delete.png')
    await page.screenshot({ path: mutationShot, fullPage: true }).catch(() => {})

    const report = {
      generated_at: new Date().toISOString(),
      base_url: baseURL,
      pages: pageResults,
      api_issues: apiIssues,
      api_canceled: apiCanceled,
      mutation_test: mutation,
      cache_consistency: cacheConsistency,
      cache_ui_binding: cacheUiBinding,
      zero_panel_observations: zeroPanelObservations,
      summary: {
        total_pages: pageResults.length,
        page_failures: pageResults.filter((p) => !p.ok).length,
        total_click_actions: pageResults.reduce((s, p) => s + p.clicks.length, 0),
        click_failures: pageResults.reduce((s, p) => s + p.clicks.filter((c) => !c.ok).length, 0),
        api_errors: apiIssues.length,
        api_canceled: apiCanceled.length,
        mutation_ok: mutation.ok,
        cache_consistency_issues: (cacheConsistency.issues || []).length,
        cache_ui_binding_issues: (cacheUiBinding.issues || []).length,
        zero_panel_observations: zeroPanelObservations.length
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
    lines.push(`- API 异常: ${report.summary.api_errors}`)
    lines.push(`- API 取消: ${report.summary.api_canceled}`)
    lines.push(`- 新增后删除自建数据: ${report.summary.mutation_ok ? '通过' : '失败'}`)
    lines.push(`- 缓存统计一致性问题: ${report.summary.cache_consistency_issues}`)
    lines.push(`- 缓存UI绑定问题: ${report.summary.cache_ui_binding_issues}`)
    lines.push(`- 全站面板零值记录: ${report.summary.zero_panel_observations}`)
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
    lines.push('## API 取消（前 30 条）')
    for (const i of report.api_canceled.slice(0, 30)) {
      lines.push(`- [${i.type}] ${i.status} ${i.url}${i.error ? ` (${i.error})` : ''}`)
    }
    if (report.api_canceled.length === 0) lines.push('- 无')
    lines.push('')
    lines.push('## 数据录入与删除自检')
    lines.push(`- 重复次数: ${report.mutation_test.repeat}`)
    lines.push(`- 总结果: ${report.mutation_test.ok ? '通过' : '失败'}`)
    for (const s of report.mutation_test.scenarios || []) {
      lines.push(`- ${s.feature}: 创建 ${s.create_ok}/${s.repeat}，删除 ${s.delete_ok}/${s.repeat}，失败 ${s.failed?.length || 0}`)
    }
    if (report.mutation_test.error) lines.push(`- 错误: ${report.mutation_test.error}`)
    lines.push('')
    lines.push('## 缓存统计一致性检查')
    if ((report.cache_consistency?.issues || []).length === 0) {
      lines.push('- 通过')
    } else {
      for (const issue of report.cache_consistency.issues) {
        lines.push(`- ${issue}`)
      }
    }
    lines.push('')
    lines.push('## 缓存UI绑定检查')
    if ((report.cache_ui_binding?.issues || []).length === 0) {
      lines.push('- 通过')
    } else {
      for (const issue of report.cache_ui_binding.issues) {
        lines.push(`- ${issue}`)
      }
    }
    lines.push('')
    lines.push('## 缓存面板零值位置（观察）')
    const zeroPositions = report.cache_ui_binding?.ui?.zeroPositions || []
    if (zeroPositions.length === 0) {
      lines.push('- 无')
    } else {
      for (const item of zeroPositions) {
        lines.push(`- [${item.section}] ${item.label}: ${item.value}`)
      }
    }
    lines.push('')
    lines.push('## 全站面板零值记录（观察）')
    if ((report.zero_panel_observations || []).length === 0) {
      lines.push('- 无')
    } else {
      for (const item of report.zero_panel_observations) {
        lines.push(`- [${item.route}] ${item.label}: ${item.value}`)
      }
    }

    await fs.writeFile(path.join(outputDir, 'report.md'), lines.join('\n'), 'utf-8')

    const bugsMd = buildBugsMarkdown(report)
    await fs.writeFile(path.join(outputDir, 'BUGS.md'), bugsMd, 'utf-8')

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
