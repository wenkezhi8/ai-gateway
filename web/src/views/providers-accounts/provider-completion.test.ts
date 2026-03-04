import { describe, expect, it } from 'vitest'

import {
  buildProviderCompletionRows,
  getAccountEffectiveLimit,
  type ProviderCompletionInputAccount
} from './provider-completion'

describe('provider completion matrix', () => {
  it('extracts deterministic effective limit from limits and usage', () => {
    const account: ProviderCompletionInputAccount = {
      id: 'a-1',
      name: 'DeepSeek 主账号',
      provider: 'deepseek',
      enabled: true,
      limits: {
        month: { type: 'token', period: 'month', limit: 0, warning: 80 },
        rpm: { type: 'rpm', period: 'minute', limit: 1800, warning: 80 }
      },
      usage: {
        month: {
          key: 'month',
          used: 0,
          limit: 0,
          remaining: 0,
          reset_at: '2026-01-01T00:00:00Z',
          period: 'month',
          percent_used: 0
        },
        rpm: {
          key: 'rpm',
          used: 50,
          limit: 1200,
          remaining: 1150,
          reset_at: '2026-01-01T00:00:00Z',
          period: 'minute',
          percent_used: 4
        }
      }
    }

    expect(getAccountEffectiveLimit(account)).toBe(1800)
  })

  it('builds row state using enabled account, effective limit and provider defaults', () => {
    const rows = buildProviderCompletionRows({
      providerOptions: [{ value: 'deepseek', label: 'DeepSeek' }],
      providerDefaults: { deepseek: 'deepseek-chat' },
      accounts: [
        {
          id: 'a-1',
          name: 'DeepSeek 主账号',
          provider: 'deepseek',
          enabled: true,
          limits: {
            month: { type: 'token', period: 'month', limit: 1000, warning: 80 }
          }
        }
      ]
    })

    expect(rows).toEqual([
      {
        provider: 'deepseek',
        label: 'DeepSeek',
        hasAccount: true,
        hasLimit: true,
        hasDefaultModel: true,
        hasVerify: false
      }
    ])
  })

  it('keeps provider list deterministic and includes defaults-only provider', () => {
    const rows = buildProviderCompletionRows({
      providerOptions: [{ value: 'qwen', label: '通义千问' }],
      providerDefaults: {
        deepseek: 'deepseek-chat'
      },
      accounts: []
    })

    expect(rows.map(row => row.provider)).toEqual(['qwen', 'deepseek'])
    expect(rows[1]).toMatchObject({
      provider: 'deepseek',
      hasAccount: false,
      hasLimit: false,
      hasDefaultModel: true
    })
  })

  it('marks verify done when enabled account has usage signal', () => {
    const rows = buildProviderCompletionRows({
      providerOptions: [{ value: 'deepseek', label: 'DeepSeek' }],
      providerDefaults: { deepseek: 'deepseek-chat' },
      accounts: [
        {
          id: 'a-verify',
          name: 'DeepSeek 验证账号',
          provider: 'deepseek',
          enabled: true,
          limits: {
            month: { type: 'token', period: 'month', limit: 1000, warning: 80 }
          },
          usage: {
            month: {
              key: 'month',
              used: 10,
              limit: 1000,
              remaining: 990,
              reset_at: '2099-01-01T00:00:00Z',
              period: 'month',
              percent_used: 1
            }
          }
        }
      ]
    })

    expect(rows[0]).toMatchObject({
      provider: 'deepseek',
      hasAccount: true,
      hasLimit: true,
      hasDefaultModel: true,
      hasVerify: true
    })
  })
})
