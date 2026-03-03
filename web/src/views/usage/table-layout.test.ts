import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('usage table layout and copy', () => {
  it('should use fixed table layout and stable single-line overflow rendering', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/usage/index.vue'), 'utf-8')

    expect(viewFile).toContain('table-layout="fixed"')
    expect(viewFile).toContain('show-overflow-tooltip')
    expect(viewFile).toContain('class-name="cell-single-line"')
  })

  it('should split total token and usage source into two columns', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/usage/index.vue'), 'utf-8')

    expect(viewFile).toContain('label="总 Token"')
    expect(viewFile).toContain('label="Token来源"')
    expect(viewFile).not.toContain('class="token-cell"')
    expect(viewFile).toContain('row.usageSourceLabel')
  })

  it('should display task type in chinese and keep raw english in tooltip', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/usage/index.vue'), 'utf-8')

    expect(viewFile).toContain('row.taskTypeLabel')
    expect(viewFile).toContain('row.taskTypeRaw')
    expect(viewFile).toContain('<el-tooltip')
  })

  it('should align all numeric columns with tabular nums class', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/usage/index.vue'), 'utf-8')

    expect(viewFile).toContain('class-name="cell-num"')
    expect(viewFile).toContain('font-variant-numeric: tabular-nums')
  })
})
