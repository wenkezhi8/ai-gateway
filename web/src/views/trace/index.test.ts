import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

describe('trace detail view', () => {
  it('shows user and ai message preview blocks with full-text actions', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/trace/index.vue'), 'utf-8')

    expect(viewFile).toContain('trace.attributes?.user_message_preview')
    expect(viewFile).toContain('trace.attributes?.ai_response_preview')
    expect(viewFile).toContain("@click=\"showFullMessage(trace, 'user')\"")
    expect(viewFile).toContain("@click=\"showFullMessage(trace, 'ai')\"")
    expect(viewFile).toContain('messageVisible')
    expect(viewFile).toContain('activeMessageContent')
  })
})
