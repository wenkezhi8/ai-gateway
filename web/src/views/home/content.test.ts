import { describe, expect, it } from 'vitest'
import {
  CAPABILITY_COLUMNS,
  FLOW_NODES,
  HERO_ACTIONS,
  QUICK_START_COMMANDS,
  TDD_STAGES,
  WORKFLOW_STEPS
} from './content'

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

  it('defines workflow phase tags for all steps', () => {
    expect(WORKFLOW_STEPS.map((item) => item.phase)).toEqual([
      'DIAGNOSE',
      'DESIGN',
      'FIX',
      'VERIFY',
      'AUDIT',
      'RETRO'
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

  it('contains four flow nodes for request lifecycle', () => {
    expect(FLOW_NODES).toHaveLength(4)
  })

  it('contains four capability columns with at least three points each', () => {
    expect(CAPABILITY_COLUMNS).toHaveLength(4)
    CAPABILITY_COLUMNS.forEach((column) => {
      expect(column.points.length).toBeGreaterThanOrEqual(3)
    })
  })

  it('includes docker/source/api quickstart commands and health check', () => {
    expect(QUICK_START_COMMANDS).toHaveProperty('docker')
    expect(QUICK_START_COMMANDS).toHaveProperty('source')
    expect(QUICK_START_COMMANDS).toHaveProperty('api')
    expect(QUICK_START_COMMANDS.source).toContain('/health')
  })
})
