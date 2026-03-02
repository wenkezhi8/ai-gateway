import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

describe('vector-db import page', () => {
  it('contains importer tabs and cancel action', () => {
    const file = resolve(process.cwd(), 'src/views/vector-db/import/index.vue')
    const content = readFileSync(file, 'utf-8')

    expect(content).toContain('向量导入中心')
    expect(content).toContain('listImportJobs')
    expect(content).toContain('JsonImporter')
    expect(content).toContain('CsvImporter')
    expect(content).toContain('PdfImporter')
    expect(content).toContain('cancelImportJob')
  })
})
