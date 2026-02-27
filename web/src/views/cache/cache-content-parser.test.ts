import { describe, expect, it } from 'vitest'

import { extractAIResponseFull, extractUserMessageFull } from './cache-content-parser'

describe('cache content parser', () => {
  it('should extract user message from direct payload', () => {
    const row = {
      value: {
        messages: [
          { role: 'system', content: 'you are helpful' },
          { role: 'user', content: 'hello world' }
        ]
      }
    }

    expect(extractUserMessageFull(row)).toBe('hello world')
  })

  it('should extract ai response from nested body payload', () => {
    const row = {
      value: {
        body: JSON.stringify({
          choices: [
            { message: { content: 'nested response' } }
          ]
        })
      }
    }

    expect(extractAIResponseFull(row)).toBe('nested response')
  })

  it('should fallback to dash when content missing', () => {
    expect(extractUserMessageFull({ value: '{}' })).toBe('-')
    expect(extractAIResponseFull({ value: '{}' })).toBe('{}')
    expect(extractAIResponseFull({ value: null })).toBe('-')
  })
})
