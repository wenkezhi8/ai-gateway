import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('routing basic-only view', () => {
  it('keeps only basic routing, distribution and feedback sections', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/routing/components/RoutePolicyTab.vue'), 'utf-8')

    expect(viewFile).toContain('路由策略配置')
    expect(viewFile).toContain('任务类型分布')
    expect(viewFile).toContain('效果评估')

    expect(viewFile).not.toContain('0.5B 分类控制器')
    expect(viewFile).not.toContain('控制面开关')
    expect(viewFile).not.toContain('任务类型模型映射')
    expect(viewFile).not.toContain('级联路由策略')
  })
})
