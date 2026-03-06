import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('layout theme menu', () => {
  it('should only expose apple variant and mode commands', () => {
    const file = readFileSync(resolve(process.cwd(), 'src/components/Layout/index.vue'), 'utf-8')

    expect(file).toContain('@click.stop="handleThemeButtonClick"')
    expect(file).toContain('const handleThemeButtonClick = () => {')
    expect(file).toContain('toggleTheme()')
    expect(file).toContain('command="variant:apple"')
    expect(file).not.toContain('command="variant:dashboard"')
    expect(file).toContain('command="mode:light"')
    expect(file).toContain('command="mode:dark"')
    expect(file).toContain('command="mode:auto"')
  })
})
