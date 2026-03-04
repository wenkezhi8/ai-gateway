import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

describe('alerts page', () => {
  it('uses backend todayTotal for 今日告警 and supports resolve-similar action', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/alerts/index.vue'), 'utf-8')

    expect(viewFile).toContain("title: '今日告警'")
    expect(viewFile).toContain('todayTotal')
    expect(viewFile).toContain('alertApi.getStats()')
    expect(viewFile).toContain('处理同类')
    expect(viewFile).toContain('@click="resolveSimilar(row)"')
    expect(viewFile).toContain('alertApi.resolveSimilar(')
  })

  it('should render trigger_count and last_triggered_at columns', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/alerts/index.vue'), 'utf-8')

    expect(viewFile).toContain('持续次数')
    expect(viewFile).toContain('最后触发')
    expect(viewFile).toContain('trigger_count')
    expect(viewFile).toContain('last_triggered_at')
  })

  it('should prefer dedup_key when resolving similar alerts', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/alerts/index.vue'), 'utf-8')

    expect(viewFile).toContain('dedup_key')
    expect(viewFile).toContain('dedup_key: alert.dedup_key || undefined')
  })

  it('should render explicit request states for rules/history/stats with retry actions', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/alerts/index.vue'), 'utf-8')

    expect(viewFile).toContain('rulesRequest.loading')
    expect(viewFile).toContain('rulesRequest.error')
    expect(viewFile).toContain('@click="fetchRules"')
    expect(viewFile).toContain('!alertRules.length')

    expect(viewFile).toContain('historyRequest.loading')
    expect(viewFile).toContain('historyRequest.error')
    expect(viewFile).toContain('@click="fetchAlerts"')
    expect(viewFile).toContain('!filteredAlerts.length')

    expect(viewFile).toContain('statsRequest.loading')
    expect(viewFile).toContain('statsRequest.error')
    expect(viewFile).toContain('@click="fetchStats"')
  })

  it('should support clear alert history action with confirm and refresh', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/alerts/index.vue'), 'utf-8')

    expect(viewFile).toContain('清空告警历史')
    expect(viewFile).toContain('@click="clearHistory"')
    expect(viewFile).toContain('alertApi.clearHistory(')
    expect(viewFile).toContain('ElMessageBox.confirm(')
    expect(viewFile).toContain('Promise.all([fetchAlerts(), fetchStats()])')
  })
})
