import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { parse } from '@vue/compiler-sfc'

import { mapRouterConfigForView } from './router-config-contract'

describe('api-management component contract', () => {
  it('should only expose auto/default/fixed routing modes in template', () => {
    const sfc = readFileSync(join(process.cwd(), 'src/views/api-management/index.vue'), 'utf-8')
    const { descriptor } = parse(sfc)
    const template = descriptor.template?.content || ''

    expect(template).toContain('<el-radio-button value="auto">')
    expect(template).toContain('<el-radio-button value="default">')
    expect(template).toContain('<el-radio-button value="fixed">')
    expect(template).not.toContain('<el-radio-button value="latest">')
  })

  it('should map deprecated latest mode from router config api to auto', () => {
    const mapped = mapRouterConfigForView({
      use_auto_mode: 'latest',
      default_strategy: 'auto',
      default_model: ''
    })

    expect(mapped.useAutoMode).toBe('auto')
    expect(mapped.migrationNotice).toContain('latest')
  })
})
