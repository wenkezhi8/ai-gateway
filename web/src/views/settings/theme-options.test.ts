import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('settings theme options', () => {
  it('should remove dashboard variant from appearance settings', () => {
    const file = readFileSync(join(process.cwd(), 'src/views/settings/index.vue'), 'utf-8')

    expect(file).toContain('<el-radio-button value="apple">Apple</el-radio-button>')
    expect(file).not.toContain('<el-radio-button value="dashboard">仪表盘</el-radio-button>')
    expect(file).toContain('<el-radio-button value="light">亮色</el-radio-button>')
    expect(file).toContain('<el-radio-button value="dark">暗色</el-radio-button>')
    expect(file).toContain('<el-radio-button value="auto">跟随系统</el-radio-button>')
  })
})
