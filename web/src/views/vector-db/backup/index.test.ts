import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db backup page', () => {
  it('contains backup and restore actions', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/backup/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('备份恢复管理')
    expect(content).toContain('listBackupTasks')
    expect(content).toContain('createBackupTask')
    expect(content).toContain('triggerBackupRestore')
    expect(content).toContain('retryBackupTask')
  })
})
