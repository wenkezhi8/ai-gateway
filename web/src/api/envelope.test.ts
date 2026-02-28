import { describe, expect, it } from 'vitest'

import { ApiError, unwrapEnvelope } from './envelope'

describe('api envelope', () => {
  it('returns data on success envelope', () => {
    const data = unwrapEnvelope<{ value: number }>({
      success: true,
      data: { value: 1 }
    })
    expect(data).toEqual({ value: 1 })
  })

  it('throws ApiError on failed envelope', () => {
    expect(() => unwrapEnvelope({
      success: false,
      error: {
        code: 'invalid_request',
        message: 'bad request'
      }
    })).toThrow(ApiError)
  })

  it('accepts plain payload when allowPlain is true', () => {
    const data = unwrapEnvelope<{ id: string }>({ id: 'chatcmpl-1' }, { allowPlain: true })
    expect(data).toEqual({ id: 'chatcmpl-1' })
  })
})
