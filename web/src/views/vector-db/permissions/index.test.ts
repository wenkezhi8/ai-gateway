import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db permissions page', () => {
  it('contains api key permission management actions', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/permissions/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('向量权限管理')
    expect(content).toContain('listVectorPermissions')
    expect(content).toContain('createVectorPermission')
    expect(content).toContain('deleteVectorPermission')
  })
})
