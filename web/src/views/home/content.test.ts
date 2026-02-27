import { describe, expect, it } from 'vitest'
import { HERO_ACTIONS, TDD_STAGES, WORKFLOW_STEPS } from './content'

describe('home content contract', () => {
  it('contains the required workflow steps in order', () => {
    expect(WORKFLOW_STEPS).toHaveLength(6)
    expect(WORKFLOW_STEPS.map((item) => item.title)).toEqual([
      '问题排查测试',
      '修复方案讨论',
      '代码修复',
      '回归验证',
      '合规审计',
      '复盘归档'
    ])
  })

  it('contains TDD stages in order', () => {
    expect(TDD_STAGES.map((item) => item.name)).toEqual([
      'RED',
      'GREEN',
      'REFACTOR',
      'VERIFY'
    ])
  })

  it('has at least three hero actions including workflow and GitHub', () => {
    expect(HERO_ACTIONS.length).toBeGreaterThanOrEqual(3)
    const ids = HERO_ACTIONS.map((item) => item.id)
    expect(ids).toContain('quick-start')
    expect(ids).toContain('workflow')
    expect(ids).toContain('github')
  })
})
